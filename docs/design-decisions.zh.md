# gcode 设计决策

本文档汇总 gcode 的关键架构决策，采用 ADR（Architecture Decision Record）风格：**问题 → 约束 → 决策 → 影响**。

---

## D1：为什么不用 protoc-gen-go

**问题**：proto 文件已有官方 Go 代码生成工具 `protoc-gen-go`，为什么要自己实现？

**约束**：
- `protoc-gen-go` 生成的 struct 包含 `protoimpl.MessageState`、`sizeCache`、`unknownFields` 等运行时字段，不适合直接作为 DAO 层数据结构
- 官方生成代码依赖 `google.golang.org/protobuf` 运行时，引入了不必要的依赖
- 官方生成代码的 JSON tag 不符合项目约定（camelCase vs snake_case）
- 无法通过注解控制 gorm tag、validate 规则等业务层关注点

**决策**：自行实现代码生成器，生成纯 Go struct，不引入 protobuf 运行时。

**影响**：
- 生成的 struct 是普通 Go struct，可直接用于 GORM、JSON 序列化、HTTP binding
- wire format 兼容性需要自行保证（通过 `testdata/compat/pbgo/` 回归验证）
- 需要自行实现 `MarshalBinary`/`UnmarshalBinary`

---

## D2：为什么用 protocompile 而不是 protoc

**问题**：解析 proto 文件有多种方式，为什么选择 `protocompile`？

**约束**：
- `protoc` 是外部二进制，需要用户单独安装，增加使用门槛
- `protoc` 通过插件机制（stdin/stdout）与生成器通信，架构复杂
- 需要在 Go 进程内直接访问 proto 的 AST 和语义信息

**决策**：使用 `github.com/bufbuild/protocompile`，纯 Go 实现，可直接嵌入二进制。

**影响**：
- 用户只需 `go install`，无需安装 protoc
- `gcode/options.proto` 和 `buf/validate/validate.proto` 通过 `//go:embed` 嵌入二进制，用户无需额外文件
- 解析结果直接在 Go 进程内访问，无需跨进程通信

---

## D3：wire format 直接实现 vs 官方反射机制

**问题**：序列化/反序列化是用官方 protobuf 反射机制，还是直接操作 wire format？

**约束**：
- 官方反射机制依赖 `google.golang.org/protobuf`，与 D1 的"不引入运行时"目标冲突
- 需要保证与官方 protobuf 二进制格式完全兼容

**决策**：在 `runtime/` 包中直接实现 protobuf wire format 编码原语（varint、ZigZag、tag、length-delimited），生成代码直接调用这些原语。

**影响**：
- 生成的 `MarshalBinary`/`UnmarshalBinary` 不依赖任何外部运行时
- wire format 兼容性通过 `testdata/compat/compat_test.go` 与 `protoc-gen-go` 输出做字节级对比验证
- `runtime/` 是公开包，用户项目可直接引用

---

## D4：两阶段中间模型（model → transform.GoFile）

**问题**：为什么在 parser 和 render 之间引入两层中间表示？

**约束**：
- proto 语义（snake_case 字段名、嵌套 message、proto 类型系统）与 Go 语义（CamelCase、扁平类型、Go 类型系统）差异显著
- render 层不应关心 proto 语法细节
- 命名冲突解决、类型映射等逻辑需要集中处理

**决策**：
- `model.File`：proto 语义模型，与 proto 语法一一对应，与 Go 无关
- `transform.GoFile`：Go 语义模型，已完成命名转换、类型解析、嵌套展平

**影响**：
- render 层只处理 Go 概念，逻辑简单
- 命名冲突解决（`GoCamelCase` + 冲突后缀）集中在 transform 层
- 嵌套 message 展平为顶层类型，Go 代码无嵌套

---

## D5：optional 字段生成指针类型

**问题**：proto3 `optional` 字段如何在 Go 中表示"未设置"语义？

**约束**：
- proto3 标量字段默认值为零值，无法区分"未设置"和"设置为零值"
- `optional` 关键字引入了 field presence 语义
- `optional bytes` 有特殊语义：nil 表示"未设置"，`[]byte{}` 表示"设置为空"

**决策**：
- `optional` 标量/enum 字段 → 生成 `*T`（指针类型）
- `optional bytes` → 生成 `[]byte`（nil 表示未设置，无需双重指针）
- 非 optional 字段保持原类型

**影响**：
- 用户通过 nil 检查判断字段是否设置
- marshal 时 nil 指针字段跳过（不写入 wire）
- validate 继承时 nil 指针字段跳过校验

---

## D6：validate 继承机制

**问题**：create/update 派生 message 如何复用源 message 的 validate 规则？

**约束**：
- 派生 message 的字段是源 message 字段的子集，validate 规则应自动继承
- 派生 message 中 optional 字段（指针类型）nil 时不应触发校验
- condition_fields（WHERE 条件字段）在 update 场景中是必填的，不应做零值守卫
- required_fields（create 场景强制非 optional 字段）同样不做 nil 守卫，直接校验

**决策**：
- render 层通过 `create_source`/`update_source` 注解追踪源 message
- 在全局 `MessageIndex` 中定位源 message 的 validate 规则
- optional 字段（指针类型）：生成 `if p.Field != nil { ... }` 守卫
- condition_fields：禁用零值守卫（`name != ""`），直接校验

**影响**：
- validate 规则只在源 message 中写一次，派生 message 自动继承
- 派生 message 的 `Validate()` 方法与源 message 语义一致

---

## D7：create/update 派生 message 两阶段 pipeline

**问题**：如何从一个 proto 文件生成派生 message？

**约束**：
- 派生 message 需要作为独立的 proto message 存在，才能被 service rpc 引用
- 生成的中间 proto 文件可被其他工具复用（如 protoc-gen-go、buf），保持与 proto 生态的兼容性
- CLI 保持单一职责：gen-proto 只负责 proto 生成，gcode 只负责 Go 生成

**决策**：两阶段 pipeline：
1. `gcode gen-proto`：读取 `gcode.update_message`/`gcode.create_message` 注解，生成中间 proto 文件（`*.update.proto`/`*.create.proto`）
2. `gcode`：将所有 proto 文件（含生成的中间 proto）统一处理，生成 Go 代码

**影响**：
- 派生 message 是真正的 proto message，可被 service rpc 直接引用
- 两阶段解耦：gen-proto 只关心 proto 生成，gcode 只关心 Go 生成
- 中间 proto 文件通过 `update_source`/`create_source` 注解记录来源，供 validate 继承使用

---

## D8：RPC interface 不绑定传输协议

**问题**：生成的 service 代码应该包含什么？

**约束**：
- 不同项目使用不同的传输协议（HTTP、gRPC、消息队列）
- 路由注册、middleware、认证等是应用层关注点，不属于代码生成范畴
- streaming rpc 需要特殊处理，当前阶段不支持

**决策**：只生成 Go interface，方法签名固定为 `Method(ctx context.Context, req *XxxRequest) (*XxxResponse, error)`，不生成路由、序列化、client stub。

**影响**：
- 用户完全控制传输层（路由 path、HTTP method、middleware）
- interface 可被任何传输协议实现（HTTP、gRPC、测试 mock）
- streaming rpc 遇到时报错退出，不静默忽略

---

## D9：HTTP adapter 设计

**问题**：如何生成 HTTP handler，同时保持传输层与业务层解耦？

**约束**：
- handler 不应依赖具体的 service 实现类型
- 生成代码无法知道哪个字段对应 path param（proto 字段无位置语义），因此不支持 path param；内部服务统一使用 JSON body
- validate 是公共方法，可在任何场景调用；handler 内置调用保证传输层统一拦截

**决策**：
- 生成 handler 工厂函数 `XxxHandler(svc XxxService, interceptors ...handlerx.Interceptor[*Req, *Resp]) gin.HandlerFunc`，接收 interface 而非具体类型；委托给 `httpruntime.NewHandler`，由其应用 interceptor chain 并内置 panic 恢复
- 统一使用 `c.ShouldBindJSON`（强制 JSON body），不支持 path param
- handler 内置 `req.Validate()` 调用（bind 后、svc 调用前），同时 `Validate()` 作为公共方法可在其他场景复用
- 使用 `c.Request.Context()` 传递请求 context，保留 deadline/cancel/trace 信息

**影响**：
- handler 与 service 实现解耦，测试时可注入 mock
- 路由 path 和 HTTP method 由用户完全控制
- validate 在传输层自动执行，同时不妨碍在 service 层或其他场景单独调用

---

## D10：响应格式与 HTTP status

**问题**：HTTP status code 与业务 code 如何分离？

**约束**：
- HTTP status 是传输层语义，业务 code 是应用层语义
- 混用会导致 middleware、负载均衡、监控系统对业务错误产生误判

**决策**：HTTP status 永远返回 200，业务结果通过响应体的 `code` 字段传递：
- 成功：`{"code": 0, "data": {...}}`
- 错误：`{"code": 5000, "error": {"msg": "..."}}`

错误 code 两层机制：
1. `CodedError` interface：业务 error 实现 `Code() int`，`ErrResponse` 自动提取；否则缺省 CodeDefaultErr (5000)
2. gin middleware：可完全替换错误响应格式，处理跨切面逻辑

**影响**：
- 客户端通过 `code` 字段判断业务成功/失败，不依赖 HTTP status
- 业务层通过实现 `CodedError` 自定义错误 code，无需修改生成代码

**适用范围**：本设计仅针对内部业务 RPC handler。需要依赖 HTTP status code 语义的基础设施端点——健康检查（`/healthz`、`/readyz`）、负载均衡探针、监控采集——不在 gcode 的职责范围内，应自行实现。这类端点属于系统辅助代码，与业务逻辑无关。

---

## D11：TagProvider 可插拔接口

**问题**：如何支持多种 struct tag（json、gorm、未来可能的 validate tag 等）而不硬编码？

**约束**：
- json tag 是内置的，所有 struct 都需要；且 omitempty/ignore 逻辑与 json tag 字符串构造强耦合，不适合抽象为 provider
- gorm tag 是可选的，只有配置了 `(gcode.message).gorm` 的 message 才生成
- 未来可能需要支持其他 tag（如 `mapstructure`、`yaml`）

**决策**：定义 `TagProvider` interface，render 层接受 `[]TagProvider`，按顺序调用每个 provider 生成 tag 片段。json tag 作为内置逻辑，gorm tag 通过 `GormTagProvider` 实现。

**影响**：
- 新增 tag 类型只需实现 `TagProvider` interface，不修改 render 核心逻辑
- `_defaultProviders` 包含 `GormTagProvider`，用户可通过注解控制是否生成 gorm tag

---

## D12：buf/validate 注解复用

**问题**：validate 规则如何定义？自己设计注解语法还是复用现有标准？

**约束**：
- 自定义注解语法需要用户学习新的 DSL
- buf/validate 是业界广泛使用的 proto validate 标准，语义清晰

**决策**：直接复用 `buf/validate/validate.proto` 的注解语法，将 `buf/validate/validate.proto` 嵌入二进制，用户可直接在 proto 文件中使用 `(buf.validate.field).*` 注解。

**影响**：
- 用户无需学习新的注解语法
- validate 规则与 buf/validate 生态兼容（但运行时是自行实现的，不依赖 buf/validate 运行时）
- `buf/validate/validate.proto` 通过 `embeddedResolver` 注入，用户无需安装 buf 工具链

---

## D13：c.Error + DefaultErrorHandler 模式

**问题**：生成的 HTTP handler 应该直接写错误响应，还是通过 gin context 传递错误？

**约束**：
- handler 直接写响应（`c.JSON`）时，middleware 无法拦截错误，无法统一处理 ValidationError（CodeValidationErr）和业务错误（CodeDefaultErr）
- 用户可能需要自定义错误响应格式（如添加 request_id、trace_id、国际化消息）
- ValidationError 需要映射到 CodeValidationErr (1001)，其他错误映射到 CodeDefaultErr (5000) 或 CodedError.Code()
- 未注册错误处理 middleware 时，应有明确的行为说明，不能静默丢失错误

**决策**：
- handler 内所有错误路径改为 `_ = c.Error(err); return`，不直接写响应
- `httpruntime.DefaultErrorHandler()` 作为 gin middleware 兜底：ValidationError → CodeValidationErr (1001)，其他 → CodeDefaultErr (5000) 或 CodedError.Code()
- 函数注释中明确警告：未注册 middleware 时错误路径返回 HTTP 200 空 body

**影响**：
- 用户可替换 DefaultErrorHandler 实现自定义错误格式，无需修改生成代码
- ValidationError 自动映射 CodeValidationErr (1001)，无需在每个 handler 中重复处理
- 错误处理逻辑集中在 middleware，handler 保持简洁
- 未注册 DefaultErrorHandler 的风险已通过文档和注释明确告知

---

## D14：gorm TableName 继承与 ToMap key 语义

**问题**：create 派生 struct 是否应该继承源 struct 的 `TableName()`？`ToMap()` 的 key 应该用什么？

**约束**：
- create 派生 struct 的设计意图是"插入一条源 struct 对应的记录"，语义上属于同一张表
- GORM 的 `db.Create(&PersonCreate{...})` 需要 `TableName()` 才能找到正确的表，否则 GORM 会将 struct 名推断为表名（`person_creates`），导致错误
- GORM 的 `db.Updates(map)` 直接用 map key 作为列名，不走 struct tag 映射，所以 `ToMap()` 的 key 必须是数据库列名
- update 派生 struct 的 update 场景通过 `db.Model(&Person{})` 显式指定表，不需要 `TableName()`
- create 派生 struct 的 `GormMessageOptions` 在 transform 层为 nil（派生 proto 无 gorm 注解），需要在 render 层通过 `Context.MessageIndex` 继承

**决策**：
- 原始 struct 和 create 派生 struct 都生成 `TableName()`，create 派生 struct 在 render 层通过 `Context.MessageIndex` 查找源 message 的 `GormMessageOptions` 并继承（浅拷贝，不修改调用方的 `GoFile`）
- update 派生 struct 不生成 `TableName()`
- `ToMap()` 的 key 优先使用 `(gcode.field).gorm.column` 覆盖的列名，无覆盖时使用 proto 字段名
- `genproto` 生成派生 proto 时复制字段级 `gcode.field.gorm.column` 注解，确保派生 struct 的字段携带正确的列名信息

**影响**：
- 用户可以直接 `db.Create(&PersonCreate{...})`，无需额外指定 `db.Model(&Person{})`
- `db.Model(&Person{}).Updates(req.ToMap())` 能正确匹配数据库列名，即使字段有列名覆盖
- `(gcode.field).gorm.column` 注解同时影响 struct tag 和 `ToMap()` key，保持一致性
- render 层的继承逻辑使用浅拷贝，不产生副作用，调用方的 `GoFile` 不被修改

---

## D15：TypeScript 生成 — 纯类型定义，不做运行时序列化

**问题**：gcode 应该生成完整的 TypeScript SDK（HTTP 客户端、序列化）还是只生成类型定义？

**约束**：
- gcode 的核心目标是 Go 代码生成，TypeScript 支持是补充功能
- 前端项目使用不同的 HTTP 客户端（fetch、axios、tRPC）和校验库（zod、yup、ajv）
- proto 注解定义了验证规则，应可在不同前端库中复用
- 前端做 protobuf 二进制序列化增加了复杂度，但对 JSON API 收益有限

**决策**：只生成纯类型定义：
- `interface` 用于 proto message（属性名使用 camelCase，与 Go JSON tag 一致）
- `enum` + 名称映射 `Record` 用于 proto enum
- 验证元数据为带类型的 `const` 对象（不绑定特定库）
- ES Module 格式，import 使用 `.js` 扩展名（最大兼容性）

**影响**：
- 前端获得类型安全和验证元数据，不绑定特定库
- 无运行时序列化——前端通过 JSON fetch 消费数据（这是主流模式）
- 验证元数据可驱动表单校验、UI 约束，或转换为 zod/yup schema
- `gen-ts` 作为独立子命令，与 Go 生成完全解耦

---

## D16：运行时 import 路径硬编码

**问题**：`runtime`、`validateruntime`、`httpruntime` 的 import 路径是否应该可配置？

**约束**：
- 生成的代码必须 import 运行时包才能编译
- 让 import 路径可配置会增加 CLI 参数、配置复杂度和文档负担
- 模块路径 `github.com/pinealctx/gcode` 是稳定的；修改它属于大版本级别的变更

**决策**：在生成器中硬编码运行时 import 路径。生成的代码始终 import：
- `github.com/pinealctx/gcode/runtime`
- `github.com/pinealctx/gcode/validateruntime`
- `github.com/pinealctx/gcode/httpruntime`

**影响**：
- 常见场景无需任何配置
- 如果模块被 fork 或重命名，必须手动更新生成代码中的 import 路径——这是有意为之：模块重命名是破坏性变更，应该有明确的操作
- 可配置 import 路径的需求推迟到未来大版本中处理
