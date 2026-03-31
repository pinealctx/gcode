# gcode 架构概览

gcode 是一个纯 Go 的 CLI 工具，从 `.proto` 文件生成 Go 代码。不依赖 `protoc`，不引入 protobuf 官方运行时，生成的代码是普通 Go struct，可直接用于 GORM、JSON 序列化和 gin HTTP 服务。

---

## 整体流水线

```
.proto 文件（含 gcode.update_message / gcode.create_message 注解）
    │
    ▼
[gen-proto]     前置子命令：读取 update_message / create_message 注解，
                生成派生 proto 文件（*.update.proto / *.create.proto），
                输出到同一目录，供主流水线继续处理
    │
    ▼
.proto 文件（原始 + 派生）
    │
    ▼
[source]        扫描目录，发现所有 .proto 文件，稳定排序
    │
    ▼
[parser]        基于 protocompile 解析 proto，读取 message/field/enum/
                service/注释/custom options，映射到语义模型
    │
    ▼
[model]         中间语义模型（File / Message / Field / Enum / Service / RPC）
                与 proto 语法无关，与 Go 语法无关
    │
    ▼
[transform]     model → Go 中间表示（GoFile / GoMessage / GoField /
                GoEnum / GoService / GoRPCMethod），
                计算 Go 类型名、字段名、包名
    │
    ▼
[render]        Go 中间表示 → Go 源码字符串，go/format 格式化，输出 []byte
    │
    ▼
生成文件
  *.pb.dao.go              struct 定义 + MarshalBinary/UnmarshalBinary/ToMap
  *.pb.dao.validate.go     Validate() error 方法
  *.pb.rpc.go              service interface 定义
  *.pb.http.go             gin HTTP handler 工厂函数
```

---

## 各层职责

### source

扫描指定目录，发现所有 `.proto` 文件。对文件列表做稳定排序（保证多次运行输出一致），校验路径安全性（防止路径穿越）。

### parser

基于 `protocompile` 库解析 proto 文件。职责：
- 解析 message、field、enum、service、rpc 定义
- 读取 leading comment（`//` 行注释和 `/* */` 块注释）
- 读取 custom options（`gcode.message`、`gcode.field`、`buf.validate.field`）
- 将解析结果映射到 `model.File`

内置 `embeddedResolver`：将 `gcode/options.proto` 和 `buf/validate/validate.proto` 嵌入二进制，用户无需额外安装任何工具。

### model

中间语义模型，是 parser 和 transform 之间的契约。核心类型：

| 类型            | 说明                                               |
| --------------- | -------------------------------------------------- |
| `model.File`    | 一个 proto 文件的完整语义表示                      |
| `model.Message` | message 定义，含 fields、注释、gcode/validate 注解 |
| `model.Field`   | 字段定义，含类型、optional 标记、注解              |
| `model.Enum`    | enum 定义，含 values 和注释                        |
| `model.Service` | service 定义，含 RPCs 和注释                       |
| `model.RPC`     | 单个 rpc 方法，含请求/响应类型和注释               |
| `model.Comment` | 注释内容，`Lines []string`                         |

### transform

将 `model.File` 转换为 Go 中间表示 `transform.GoFile`。职责：
- 展平嵌套 message（proto 允许嵌套，Go 不支持）
- 计算 Go 类型名（`GoCamelCase`，处理命名冲突）
- 计算 Go 字段名（处理 proto snake_case → Go CamelCase）
- 解析字段类型（scalar → Go 基础类型，message → 指针，optional → 指针）
- 验证 create_message 的 required_fields 语义约束

### render

将 `transform.GoFile` 渲染为 Go 源码。四个生成函数：

| 函数                  | 输出文件               | 说明                                               |
| --------------------- | ---------------------- | -------------------------------------------------- |
| `render.File`         | `*.pb.dao.go`          | struct 定义、MarshalBinary、UnmarshalBinary、ToMap |
| `render.ValidateFile` | `*.pb.dao.validate.go` | `Validate() error` 方法                            |
| `render.RPCFile`      | `*.pb.rpc.go`          | service interface                                  |
| `render.HTTPFile`     | `*.pb.http.go`         | gin handler 工厂函数                               |

所有函数最后调用 `go/format.Source` 格式化输出，保证生成代码风格一致。

proto leading comment 全部透传到生成代码：struct/field/enum（`*.pb.dao.go`）、service interface/method（`*.pb.rpc.go`）、HTTP handler（`*.pb.http.go`）均已支持。

### runtime

protobuf wire format 编码原语（varint、ZigZag、tag、length-delimited、size 计算）。生成的 `MarshalBinary`/`UnmarshalBinary` 直接调用此包，不依赖官方 protobuf 反射机制。公开包，用户项目可直接引用。

### validateruntime

validate 运行时辅助包。提供：
- `ValidationError`（含 Field/Rule/Message 字段）
- `IsEmail` / `IsURI`（可替换的包级变量，方便测试注入）
- `MatchPattern`（RE2 正则，`sync.Map` 缓存编译结果）

公开包，用户项目可直接引用。

### httpruntime

HTTP adapter 运行时辅助包。提供：
- `Response`（`Code int`、`Data any`、`Error *Error`）— 统一响应信封
- `Error`（`Msg string`、`Fields map[string]any`）
- `CodedError` interface — 业务 error 实现此接口可携带自定义 code
- `OKResponse(data any) Response` — 构造成功响应（code 0）
- `ErrResponse(err error) Response` — 构造错误响应（自动提取 CodedError.Code()，缺省 500）
- `DefaultErrorHandler() gin.HandlerFunc` — gin middleware，将 `c.Error()` 中的错误转换为 JSON 响应（ValidationError → code 400，其他 → code 500 或 CodedError.Code()）

公开包，用户项目可直接引用。

---

## 生成文件类型

| 文件                   | 触发条件                    | 内容                                                                                                                                                  |
| ---------------------- | --------------------------- | ----------------------------------------------------------------------------------------------------------------------------------------------------- |
| `*.pb.dao.go`          | 所有 proto 文件             | struct 定义、json/gorm tag、MarshalBinary、UnmarshalBinary、UnmarshalBinaryLenient、ToMap（update 派生 message）、TableName()（有 gorm.table 注解时） |
| `*.pb.dao.validate.go` | 所有 proto 文件             | `Validate() error` 方法，覆盖全部 buf/validate 约束类型                                                                                               |
| `*.pb.rpc.go`          | proto 文件含 `service` 定义 | Go interface，方法签名 `Method(ctx context.Context, req *XxxRequest) (*XxxResponse, error)`                                                           |
| `*.pb.http.go`         | proto 文件含 `service` 定义 | gin handler 工厂函数 `XxxHandler(svc XxxService) gin.HandlerFunc`，内置 bind → validate → svc 调用流程                                                |

---

## 目录结构

```
github.com/pinealctx/gcode/
├── cmd/gcode/              CLI 入口
├── internal/
│   ├── app/                流水线编排（Run / RunGenProto）
│   ├── config/             CLI 参数解析与校验
│   ├── model/              中间语义模型
│   ├── parser/             proto → model
│   ├── naming/             protobuf-to-Go 命名规则
│   ├── transform/          model → Go 中间表示
│   ├── render/             Go 中间表示 → Go 源码
│   └── source/             目录扫描与文件发现
├── options/                gcode_options.proto（embed 源）
├── runtime/                wire format 编码原语（公开包）
├── validateruntime/        validate 运行时辅助（公开包）
├── httpruntime/            HTTP adapter 运行时辅助（公开包）
├── reference/              protobuf/wire format 技术参考文档
└── testdata/compat/        端到端兼容性测试套件
    ├── proto/              proto 源文件
    ├── dao/                生成的 Go 文件（快照）
    ├── pbgo/               protoc-gen-go 官方输出（wire 兼容性基准）
    └── gen/main.go         重新生成所有快照的入口
```

---

## 设计目标

1. **不依赖 protoc 工具链** — 用 `protocompile` 解析 proto schema，生成纯 Go struct，不引入 `google.golang.org/protobuf` 运行时
2. **wire format 兼容** — 生成的 `MarshalBinary`/`UnmarshalBinary` 与官方 protobuf 二进制格式完全兼容，由 `testdata/compat/pbgo/` 回归验证
3. **JSON tag 内置** — 默认生成 `json:"field_name"`，通过 `(gcode.field).json` 注解支持 `omitempty`/`ignore`
4. **gorm tag 可选** — 通过 `(gcode.message).gorm` 注解控制是否生成 gorm tag；`(gcode.field).gorm.column` 覆盖列名
5. **validate 通过注解支持** — 复用 buf/validate 注解语义，生成 `Validate() error` 方法
6. **派生 message 自动继承 validate** — create/update 派生 message 通过 `create_source`/`update_source` 追踪源 message，render 层自动继承 validate 规则
7. **RPC interface 不绑定传输协议** — 生成 Go interface，不生成路由/序列化/client stub，用户完全控制传输层
8. **HTTP adapter 与业务层解耦** — handler 通过 `c.Error(err)+return` 传递错误，由 `DefaultErrorHandler` middleware 统一处理响应，用户可替换为自定义实现
