# gcode 开箱即用指南

## gcode 是什么

gcode 是一个以 proto 文件为输入的代码生成工具，输出 Go struct、序列化方法、校验逻辑、HTTP handler 以及 TypeScript 类型定义。它面向使用 protobuf 作为 schema 语言、但不需要完整 gRPC 栈的后端服务。

**gcode 生成什么：**

| 输入 | 输出 |
|---|---|
| 任意 `.proto` 文件 | Go struct + `MarshalBinary` / `UnmarshalBinary` + `Validate()` |
| `.meta.proto` schema 文件 | 三个派生 proto 文件（entity / create / update），由 `gen-proto` 生成 |
| `.entity.proto` | 带 GORM tag 的 Go struct + `TableName()` + `DeepClone()` |
| `.create.proto` | Go struct + `Validate()` + `ToEntity()` + `DeepClone()` |
| `.update.proto` | Go struct + `Validate()` + `ToMap()` + `ApplyTo()` + `DeepClone()` |
| service 定义 | Go interface + gin HTTP handler 工厂函数 |
| 任意 `.proto` 文件 | TypeScript interface + enum + 验证元数据 |

**适用范围与约束：**

- 仅支持 proto3，不支持 proto2。
- 生成的 Go 代码面向 GORM 持久化和 gin HTTP 服务，不支持其他 ORM 或 HTTP 框架。
- 不生成 gRPC stub，生成的 Go interface 是普通 Go interface，不是 gRPC service。
- 不支持的 proto 特性（streaming RPC、`map<K,V>`、`oneof`、well-known types）会在生成阶段报错退出，不会静默生成错误代码。详见[已知限制](#已知限制)。

---

## 安装

```bash
go install github.com/pinealctx/gcode/cmd/gcode@latest
```

验证安装：

```bash
gcode -h
```

CLI 参数：

```
gcode [flags]                 从 proto 文件生成 Go 代码
  -in string                  输入 proto 目录
  -out string                 输出目录

gcode version                 打印版本信息

gcode gen-proto [flags]       从 schema（.meta.proto）文件生成 entity/create/update proto 文件
  -in string                  输入 proto 目录（生成文件写入同一目录）

gcode gen-ts [flags]          从 proto 文件生成 TypeScript 类型定义
  -in string                  输入 proto 目录
  -out string                 输出目录
```

---

## 用户项目依赖

生成的代码依赖以下公开包，需要在用户项目中单独安装：

```bash
# 序列化/反序列化运行时（*.pb.dao.go 依赖）
go get github.com/pinealctx/gcode/runtime

# validate 运行时（*.pb.dao.validate.go 依赖）
go get github.com/pinealctx/gcode/validateruntime

# HTTP adapter 运行时（*.pb.http.go 依赖）
go get github.com/pinealctx/gcode/httpruntime
```

如果只生成 struct 和序列化代码（不使用 validate 和 HTTP），只需安装 `runtime`。

> **运行时 import 路径固定**：生成的代码始终 import `github.com/pinealctx/gcode/runtime`、`github.com/pinealctx/gcode/validateruntime` 和 `github.com/pinealctx/gcode/httpruntime`，这些路径在生成器中硬编码，无法自定义。如果你 fork 或重命名了模块，需要同步修改生成代码中的 import 路径——这属于大版本升级级别的变更。

---

## 快速开始

最小示例：一个 proto 文件，生成 Go struct。

**1. 编写 proto 文件**

```proto
// proto/user.proto
syntax = "proto3";
package myapp;
option go_package = "myapp/dao;dao";

message User {
  string name = 1;
  int32  age  = 2;
}
```

**2. 生成代码**

```bash
gcode -in proto/ -out dao/
```

**3. 使用生成的 struct**

```go
import "myapp/dao"

u := &dao.User{Name: "Alice", Age: 30}

// 序列化为 protobuf wire format
wire, err := u.MarshalBinary()

// 反序列化
var u2 dao.User
err = u2.UnmarshalBinary(wire)
```

> **注意**：生成的代码会导入 `github.com/pinealctx/gcode/runtime`。在执行 `go get` 之前运行 `go build` 会报缺少模块的错误，这是正常的——先执行 `go get`，再构建。

---

## 完整示例

以下示例展示从 proto 定义到完整 HTTP 服务的全流程，基于 `testdata/compat/` 中的真实代码。

> **说明**：下方 proto 示例为简化版，仅保留关键字段以突出各注解的用法。完整的 proto 文件见 `testdata/compat/proto/`。

### 第一步：编写 proto 文件

> **说明**：`gcode/options.proto` 和 `buf/validate/validate.proto` 已内嵌在 gcode 二进制中，无需额外安装或下载，直接在 proto 文件中 import 即可。

#### message 定义（含 validate 和派生 message 注解）

```proto
// proto/person.meta.proto
syntax = "proto3";
package myapp;
option go_package = "myapp/dao;dao";

import "buf/validate/validate.proto";
import "gcode/options.proto";

// 标记此文件为 schema 源。gen-proto 读取此文件，生成
// person.entity.proto、person.create.proto 和 person.update.proto。
option (gcode.schema) = {};

// 原始 message：字段不使用 optional。
// 字段的可选语义由派生 message 的注解决定，而非原始定义。
// 所有字段均为普通字段（非 optional）——gen-proto 控制派生 proto 中的指针语义。
message Person {
  string name = 1 [
    (buf.validate.field).string.min_len = 1,
    (buf.validate.field).string.max_len = 100
  ];
  int32 age = 2 [
    (buf.validate.field).int32.gte = 0,
    (buf.validate.field).int32.lte = 150
  ];
  string email = 3 [(buf.validate.field).string.email = true];
  string nickname = 4 [
    (buf.validate.field).string.min_len = 1,
    (buf.validate.field).string.max_len = 10
  ];

  // 生成 update 派生 message：PersonUpdateByName
  // condition_fields 是 WHERE 条件字段，在派生 struct 中为非指针类型，不写入 ToMap()
  option (gcode.update_message) = {
    name: "PersonUpdateByName"
    condition_fields: ["name"]
    ignore_fields: []
  };

  // 生成 create 派生 message：PersonCreate
  // 派生 struct 中所有字段默认为指针类型（可选）。
  // required_fields 强制指定字段为非指针类型（必填）。
  option (gcode.create_message) = {
    name: "PersonCreate"
    ignore_fields: []
    required_fields: ["nickname"]
  };
}
```

#### service 定义

```proto
// proto/person_service.proto
syntax = "proto3";
package myapp;
option go_package = "myapp/dao;dao";

import "person.entity.proto";
import "person.create.proto";
import "person.update.proto";

message CreatePersonResponse { string id = 1; }
message GetPersonRequest     { string id = 1; }
message GetPersonResponse    { string name = 1; int32 age = 2; }
message UpdatePersonResponse { bool ok = 1; }
message DeletePersonRequest  { string id = 1; }
message DeletePersonResponse { bool ok = 1; }

// PersonService provides CRUD operations for person records.
service PersonService {
  rpc CreatePerson(PersonCreate)         returns (CreatePersonResponse);
  rpc GetPerson(GetPersonRequest)        returns (GetPersonResponse);
  rpc UpdatePerson(PersonUpdateByName)   returns (UpdatePersonResponse);
  rpc DeletePerson(DeletePersonRequest)  returns (DeletePersonResponse);
}
```

> **注意**：`person.entity.proto`、`person.create.proto` 和 `person.update.proto` 由第二步的 `gen-proto` 命令生成，不需要手动编写。

---

### 第二步：生成派生 proto

`gen-proto` 读取 `.meta.proto` schema 文件，为每个 schema 生成三类 proto 文件：

```bash
gcode gen-proto -in proto/
```

> **schema 文件命名约束**：`gen-proto` 以 `.meta.proto` 后缀作为识别 schema 文件的唯一依据。不带此后缀的文件——包括 `common.proto`、service proto 及其他共享定义文件——不会被 `gen-proto` 直接处理，而是在 protocompile 解析 `.meta.proto` 时作为依赖自动解析。
>
> 如果 `.meta.proto` 文件 import 了 `common.proto`，生成的 `*.create.proto` 和 `*.update.proto` 会自动包含 `import "common.proto"`，无需手动管理 import。

生成结果：

```
proto/
  person.meta.proto         ← schema 源文件（不变）
  common.proto              ← 共享定义文件（不变，gen-proto 不处理）
  person_service.proto      ← service 定义文件（不变，gen-proto 不处理）
  person.entity.proto       ← 生成：Person message（无 validate，有 gorm）
  person.update.proto       ← 生成：PersonUpdateByName message（含 validate）
  person.create.proto       ← 生成：PersonCreate message（含 validate）
```

- `person.entity.proto` — 包含 `Person` struct 定义和 gorm 注解，无 `buf.validate` 注解；`Person.Validate()` 返回 nil。
- `person.create.proto` — 包含 `PersonCreate`，validate 注解从 schema 拷贝；`PersonCreate.Validate()` 执行所有规则。
- `person.update.proto` — 包含 `PersonUpdateByName`，validate 注解从 schema 拷贝；`PersonUpdateByName.Validate()` 执行所有规则。

> **注意**：`gcode gen-proto` 每次运行会覆盖已有的生成文件。不要手动修改 `*.entity.proto`、`*.create.proto` 或 `*.update.proto`，修改会在下次运行时丢失。

---

### 第三步：生成所有 Go 文件

```bash
gcode -in proto/ -out dao/
```

生成结果：

```
dao/
  person.entity.pb.dao.go           ← Person struct + 序列化方法
  person.entity.pb.dao.validate.go  ← Person.Validate()（返回 nil，无 validate 注解）
  person.update.pb.dao.go           ← PersonUpdateByName struct + ToMap() + ApplyTo()
  person.update.pb.dao.validate.go  ← PersonUpdateByName.Validate()
  person.create.pb.dao.go           ← PersonCreate struct + ToEntity()
  person.create.pb.dao.validate.go  ← PersonCreate.Validate()
  person_service.pb.dao.go          ← 请求/响应 message struct
  person_service.pb.dao.validate.go
  person_service.pb.rpc.go          ← PersonService interface
  person_service.pb.http.go         ← gin handler 工厂函数
```

**派生 struct 的字段指针规则：**

| Struct | 字段类型 | Go 类型 | 说明 |
|---|---|---|---|
| `PersonCreate` | `required_fields` 中的字段 | `T`（非指针） | 调用方必须提供值 |
| `PersonCreate` | 其余字段 | `*T`（指针） | nil = 未提供，跳过校验 |
| `PersonUpdateByName` | `condition_fields` 中的字段 | `T`（非指针） | WHERE 条件，不写入 `ToMap()` |
| `PersonUpdateByName` | 其余字段 | `*T`（指针） | nil = 不更新此字段 |

---

### 第四步：使用生成的 struct

#### 序列化与反序列化

```go
p := &dao.Person{Name: "Alice", Age: 30, Email: "alice@example.com"}

// 序列化为 protobuf wire format
wire, err := p.MarshalBinary()
if err != nil {
    log.Fatal(err)
}

// 反序列化（严格模式：重复字段报错）
var p2 dao.Person
if err := p2.UnmarshalBinary(wire); err != nil {
    log.Fatal(err)
}

// 反序列化（宽松模式：重复字段取最后一个值）
var p3 dao.Person
if err := p3.UnmarshalBinaryLenient(wire); err != nil {
    log.Fatal(err)
}
```

> **JSON tag 命名规则**：proto 字段名使用 `snake_case`（如 `created_at`），但生成的 json tag 默认使用 `camelCase`（`json:"createdAt"`），与 protoc-gen-go 行为一致。

#### optional 字段

`optional` 字段生成为指针类型，nil 表示"未设置"：

```go
nickname := "ali"
p := &dao.Person{
    Name:     "Alice",
    Nickname: &nickname,  // optional string → *string
}

if p.Nickname != nil {
    fmt.Println(*p.Nickname)
}
```

#### enum 类型

Proto `enum` 定义生成 Go `int32` 类型别名和常量：

```proto
enum Status {
  STATUS_UNSPECIFIED = 0;
  STATUS_ACTIVE      = 1;
  STATUS_INACTIVE    = 2;
}

message User {
  Status status = 1;
}
```

生成的 Go 代码：

```go
type Status int32

const (
    Status_STATUS_UNSPECIFIED Status = 0
    Status_STATUS_ACTIVE      Status = 1
    Status_STATUS_INACTIVE    Status = 2
)

type User struct {
    Status Status `json:"status"`
}
```

使用 `(buf.validate.field).enum.defined_only = true` 拒绝未定义的值，详见 [注解参考](annotations.zh.md)。

#### 嵌套 message

Proto 允许在一个 message 内部定义另一个 message。gcode 会将它们展平为顶层 Go 类型：

```proto
message Order {
  message Item {
    string product = 1;
    int32  quantity = 2;
  }
  Item item = 1;
}
```

生成的 Go 代码（嵌套类型名变为 `OrderItem`）：

```go
type OrderItem struct {
    Product  string `json:"product"`
    Quantity int32  `json:"quantity"`
}

type Order struct {
    Item *OrderItem `json:"item"`
}
```

> **命名规则**：proto 中的 `Parent_Child` → Go 中的 `ParentChild`（GoCamelCase 转换）。生成的 Go 代码无嵌套，所有类型均为顶层定义。

---

### 第五步：使用 Validate()

每个 message 都会生成 `Validate()` 方法。entity message（来自 `*.entity.proto`）的 `Validate()` 返回 nil——它们不含 validate 注解。validate 校验在 create/update message 上有意义：

```go
import (
    "errors"
    "fmt"

    "github.com/pinealctx/gcode/validateruntime"
    "myapp/dao"
)

// PersonCreate.Validate() 执行 schema 中定义的所有规则
req := &dao.PersonCreate{Name: "", Age: 200}

if err := req.Validate(); err != nil {
    var ve *validateruntime.ValidationError
    if errors.As(err, &ve) {
        fmt.Printf("field=%s rule=%s msg=%s\n", ve.Field, ve.Rule, ve.Message)
        // field=name rule=min_len msg=length must be >= 1
    }
}

// PersonUpdateByName.Validate() 同样执行所有规则
upd := &dao.PersonUpdateByName{Name: "Alice"}
if err := upd.Validate(); err != nil { ... }
```

`Validate()` 是生成的公共方法，可以在任何场景调用——service 实现层、消息队列消费、批量导入等。生成的 HTTP handler 内部也会自动调用 `req.Validate()`（在 bind 之后、调用 service 之前），两者不冲突：handler 内置调用保证传输层统一拦截，公共方法保证其他场景也能复用同一套校验逻辑。

---

### 第六步：使用 ToMap()（update 场景）

`PersonUpdateByName` 的 `ToMap()` 只包含非 nil 字段，适合 GORM 的 `Updates` 方法做部分更新：

```go
age := int32(31)
req := &dao.PersonUpdateByName{
    Name: "Alice",  // condition_fields，不写入 ToMap()
    Age:  &age,     // 只更新 age
}

// ToMap() 返回 map[string]any{"age": 31}
// Name 作为 condition_fields 不包含在 map 中
db.Model(&dao.Person{}).Where("name = ?", req.Name).Updates(req.ToMap())
```

> **ToMap() key 规则**：`ToMap()` 的 map key 使用 gorm column 名。如果字段配置了 `(gcode.field).gorm.column` 覆盖，key 使用覆盖后的列名；无覆盖时使用 proto 字段名。这确保 `db.Updates(map)` 能正确匹配数据库列（GORM 对 map 直接用 key 作为列名，不走 struct tag 映射）。
>
> **validate 规则**：validate 规则在 schema（`.meta.proto`）中定义，由 `gen-proto` 拷贝到生成的 `*.create.proto` / `*.update.proto` 文件中。render 层直接从这些 proto 字段读取规则，无需跨文件反查。optional 字段（指针类型）为 nil 时跳过校验；condition_fields 不做零值守卫，直接校验。详见 [注解参考 — validate 继承行为](annotations.zh.md#validate-继承行为)。

---

### 第七步：使用 DeepClone()

每个生成的 struct 都有 `DeepClone()` 方法，返回一个完全独立的副本，克隆体与原对象之间不共享任何内存。适用于需要在应用变更前保留原始状态的场景：

```go
// 在应用更新前保留原始状态
original := entity.DeepClone()
updateMsg.ApplyTo(entity)

// 对比 original 与 entity 的差异，用于 diff、审计日志或乐观锁冲突检测
if original.Age != entity.Age {
    log.Printf("age changed: %d → %d", original.Age, entity.Age)
}
```

`DeepClone()` 正确处理所有字段类型：
- scalar 和 enum 字段：按值复制
- optional 字段（`*T`）：分配新指针，修改克隆体的字段不影响原对象
- bytes 和 repeated 字段：分配新 slice 并复制内容
- 嵌套 message 字段：递归克隆
- nil 接收者：返回 nil

---

### 第八步：实现 RPC interface

生成的 `PersonService` interface：

```go
// dao/person_service.pb.rpc.go（生成，勿手动修改）
type PersonService interface {
    CreatePerson(ctx context.Context, req *PersonCreate) (*CreatePersonResponse, error)
    GetPerson(ctx context.Context, req *GetPersonRequest) (*GetPersonResponse, error)
    UpdatePerson(ctx context.Context, req *PersonUpdateByName) (*UpdatePersonResponse, error)
    DeletePerson(ctx context.Context, req *DeletePersonRequest) (*DeletePersonResponse, error)
}
```

实现 interface：

```go
type personServiceImpl struct {
    db *gorm.DB
}

func (s *personServiceImpl) CreatePerson(ctx context.Context, req *dao.PersonCreate) (*dao.CreatePersonResponse, error) {
    // Validate() 在 HTTP handler 中已自动调用
    // 在非 HTTP 场景（如直接调用、消息队列）中，可在此处手动调用
    // if err := req.Validate(); err != nil { return nil, err }
    return &dao.CreatePersonResponse{Id: "new-id"}, nil
}

// 实现其余方法...
```

---

### 第九步：注册 gin 路由（完整 HTTP 服务）

> **gin 依赖**：生成的 `*.pb.http.go` 文件会导入 [gin](https://github.com/gin-gonic/gin)，需要在项目中安装：
> ```bash
> go get github.com/gin-gonic/gin
> ```

生成的 handler 工厂函数接收 service interface 和可选的 interceptor 列表，返回 `gin.HandlerFunc`：

```go
// dao/person_service.pb.http.go（生成，勿手动修改）
func CreatePersonHandler(svc PersonService, interceptors ...handlerx.Interceptor[*PersonCreate, *CreatePersonResponse]) gin.HandlerFunc
func GetPersonHandler(svc PersonService, interceptors ...handlerx.Interceptor[*GetPersonRequest, *GetPersonResponse]) gin.HandlerFunc
func UpdatePersonHandler(svc PersonService, interceptors ...handlerx.Interceptor[*PersonUpdateByName, *UpdatePersonResponse]) gin.HandlerFunc
func DeletePersonHandler(svc PersonService, interceptors ...handlerx.Interceptor[*DeletePersonRequest, *DeletePersonResponse]) gin.HandlerFunc
```

`interceptors` 是可变参数——现有调用 `dao.CreatePersonHandler(svc)` 无需任何修改，继续有效。

注册路由：

```go
func main() {
    svc := &personServiceImpl{db: setupDB()}

    r := gin.New()

    // ⚠️  必须注册 DefaultErrorHandler（或自定义等价 middleware）
    // 生成的 handler 使用 c.Error(err)+return 传递错误，不直接写响应。
    // 如果不注册此 middleware，所有错误路径将静默返回 HTTP 200 空 body。
    r.Use(httpruntime.DefaultErrorHandler())

    // 路由 path 和 HTTP method 由用户完全控制
    r.POST("/persons",        dao.CreatePersonHandler(svc))
    r.GET("/persons/:id",     dao.GetPersonHandler(svc))
    r.PUT("/persons/:name",   dao.UpdatePersonHandler(svc))
    r.DELETE("/persons/:id",  dao.DeletePersonHandler(svc))

    r.Run(":8080")
}
```

**响应格式**（由 `httpruntime` 统一）：

```json
// 成功
{"code": 0, "data": {"id": "new-id"}}

// validate 错误（CodeValidationErr）
{"code": 1001, "error": {"msg": "length must be >= 1"}}

// 业务错误（CodeDefaultErr，或 CodedError.Code()）
{"code": 5000, "error": {"msg": "internal error"}}
```

业务层可通过实现 `httpruntime.CodedError` 接口自定义错误 code：

```go
type AppError struct {
    code int
    msg  string
}

func (e *AppError) Error() string { return e.msg }
func (e *AppError) Code() int     { return e.code }

// 返回此 error 时，响应 code 为 404 而非默认 CodeDefaultErr (5000)
return nil, &AppError{code: 404, msg: "person not found"}
```

#### 请求示例（curl）

```bash
# 创建 person
curl -X POST http://localhost:8080/persons \
  -H "Content-Type: application/json" \
  -d '{"nickname": "alice", "email": "alice@example.com"}'
# → {"code": 0, "data": {"id": "new-id"}}

# validate 错误（nickname 过长）
curl -X POST http://localhost:8080/persons \
  -H "Content-Type: application/json" \
  -d '{"nickname": "this-name-is-way-too-long"}'
# → {"code": 1001, "error": {"msg": "length must be <= 10"}}

# 查询 person
curl http://localhost:8080/persons/some-id
# → {"code": 0, "data": {"name": "Alice", "age": 30}}
```

#### 请求体大小限制

每个 handler 委托给 `httpruntime.NewHandler`，由其内部调用 `c.ShouldBindJSON`。gin 默认不限制请求体大小。生产环境中应通过 middleware 设置上限，防止超大请求体：

```go
import "net/http"

func MaxBodyBytes(n int64) gin.HandlerFunc {
    return func(c *gin.Context) {
        c.Request.Body = http.MaxBytesReader(c.Writer, c.Request.Body, n)
        c.Next()
    }
}

r.Use(MaxBodyBytes(1 << 20)) // 1 MiB
```

#### 给请求配置超时

生成的 handler 将 `c.Request.Context()` 传递给 service 层。gin 默认不为请求注入 deadline，如需超时控制，通过 middleware 注入：

```go
func TimeoutMiddleware(d time.Duration) gin.HandlerFunc {
    return func(c *gin.Context) {
        ctx, cancel := context.WithTimeout(c.Request.Context(), d)
        defer cancel()
        c.Request = c.Request.WithContext(ctx)
        c.Next()
    }
}

r.Use(TimeoutMiddleware(5 * time.Second))
```

#### 添加 interceptor（可选）

每个生成的 handler 内置 panic 恢复——service 方法发生 panic 时会被自动捕获并转为错误，服务不会崩溃。在此基础上，可以为每个路由注入自定义 interceptor，用于日志、metrics、tracing 等横切关注点。

interceptor 的签名为：

```go
func(ctx context.Context, req *Req, next handlerx.Handler[*Req, *Resp]) (*Resp, error)
```

**示例：请求日志 interceptor**

```go
import (
    "context"
    "log/slog"

    "github.com/pinealctx/x/handlerx"
)

func LoggingInterceptor[Req, Resp any](logger *slog.Logger) handlerx.Interceptor[*Req, *Resp] {
    return func(ctx context.Context, req *Req, next handlerx.Handler[*Req, *Resp]) (*Resp, error) {
        logger.Info("request", "type", fmt.Sprintf("%T", req))
        resp, err := next(ctx, req)
        if err != nil {
            logger.Error("request failed", "error", err)
        }
        return resp, err
    }
}
```

**注册路由时传入 interceptor**

```go
logger := slog.Default()

// 不传 interceptor——与之前完全一致
r.POST("/persons", dao.CreatePersonHandler(svc))

// 为特定路由注入日志 interceptor
r.DELETE("/persons/:id", dao.DeletePersonHandler(svc,
    LoggingInterceptor[DeletePersonRequest, DeletePersonResponse](logger),
))
```

interceptor 按传入顺序执行，位于内置 panic 恢复层的内侧。service 方法始终是最内层调用。

---

## 注解速查表

详细说明和示例见 [注解参考](annotations.zh.md)。

### message 级注解

| 注解                                      | 类型     | 说明                                  |
| ----------------------------------------- | -------- | ------------------------------------- |
| `(gcode.message).gorm.table`              | string   | 覆盖 gorm 表名                        |
| `(gcode.update_message).name`             | string   | 生成的 update 派生 message 名称       |
| `(gcode.update_message).condition_fields` | []string | WHERE 条件字段，不写入 `ToMap()`      |
| `(gcode.update_message).ignore_fields`    | []string | 不包含在派生 message 中的字段         |
| `(gcode.create_message).name`             | string   | 生成的 create 派生 message 名称       |
| `(gcode.create_message).ignore_fields`    | []string | 不包含在派生 message 中的字段         |
| `(gcode.create_message).required_fields`  | []string | 在派生 message 中为非指针类型（必填） |

### field 级注解

| 注解                             | 类型   | 说明                             |
| -------------------------------- | ------ | -------------------------------- |
| `(gcode.field).json.omitempty`   | bool   | 生成 `json:"field,omitempty"`    |
| `(gcode.field).json.ignore`      | bool   | 生成 `json:"-"`                  |
| `(gcode.field).gorm.column`      | string | 覆盖 gorm 列名                   |
| `(gcode.field).validate_message` | string | 覆盖该字段所有约束的默认错误消息 |

### validate 注解（buf/validate）

| 注解                                      | 适用类型      | 说明                           |
| ----------------------------------------- | ------------- | ------------------------------ |
| `(buf.validate.field).string.min_len`     | string        | 最小字节长度                   |
| `(buf.validate.field).string.max_len`     | string        | 最大字节长度                   |
| `(buf.validate.field).string.email`       | string        | 邮箱格式校验                   |
| `(buf.validate.field).string.uri`         | string        | URI 格式校验                   |
| `(buf.validate.field).string.pattern`     | string        | RE2 正则匹配                   |
| `(buf.validate.field).string.in`          | string        | 枚举值校验（可多次声明）       |
| `(buf.validate.field).string.not_in`      | string        | 排除值校验（可多次声明）       |
| `(buf.validate.field).int32.gte`          | int32         | 大于等于                       |
| `(buf.validate.field).int32.lte`          | int32         | 小于等于                       |
| `(buf.validate.field).int32.gt`           | int32         | 大于                           |
| `(buf.validate.field).int32.lt`           | int32         | 小于                           |
| `(buf.validate.field).int32.in`           | int32         | 枚举值校验                     |
| `(buf.validate.field).int32.not_in`       | int32         | 排除值校验                     |
| `(buf.validate.field).int64.*`            | int64         | 同 int32 系列                  |
| `(buf.validate.field).float.*`            | float32/64    | 同 int32 系列（gte/lte/gt/lt） |
| `(buf.validate.field).bytes.min_len`      | bytes         | 最小字节数                     |
| `(buf.validate.field).bytes.max_len`      | bytes         | 最大字节数                     |
| `(buf.validate.field).repeated.min_items` | repeated      | 最少元素数                     |
| `(buf.validate.field).repeated.max_items` | repeated      | 最多元素数                     |
| `(buf.validate.field).repeated.items.*`   | repeated      | 对每个元素应用约束             |
| `(buf.validate.field).enum.defined_only`  | enum          | 只允许已定义的枚举值           |
| `(buf.validate.field).required`           | message/bytes | 不允许 nil / 空                |
| `(buf.validate.field).message.required`   | message       | 嵌套 message 不允许 nil        |

---

## TypeScript 代码生成

gcode 从 proto 文件生成 TypeScript 类型定义，为前端提供类型安全和一致的验证元数据。

### 前置条件

无额外依赖。`gen-ts` 命令使用与 Go 生成相同的 proto 解析流水线。

### 生成 TS 文件

如果 proto 文件使用了 `gcode.update_message` / `gcode.create_message` 注解，需要先运行 `gen-proto`（见上方 Go 部分第二步）生成派生 proto 文件。然后：

```bash
gcode gen-ts -in proto/ -out ts/
```

生成结果：

```
ts/
  person.entity.pb.ts       ← Person interface + Status enum（无验证元数据）
  person.create.pb.ts       ← PersonCreate interface + PersonCreateRules 验证元数据
  person.update.pb.ts       ← PersonUpdateByName interface + PersonUpdateByNameRules 验证元数据
  person_service.pb.ts      ← 请求/响应 interface + 验证元数据
```

### 生成内容

**Interface** — proto message 生成为 TypeScript interface，属性名使用 camelCase：

```typescript
export interface Person {
  name: string
  age: number
  status: Status
  scores: number[]
  nickname?: string  // optional 字段 → T | undefined
}
```

**Enum** — proto enum 生成为 TypeScript enum + 名称映射 Record：

```typescript
export enum Status {
  STATUS_UNSPECIFIED = 0,
  STATUS_ACTIVE = 1,
  STATUS_INACTIVE = 2,
}

export const StatusName: Record<Status, string> = {
  [Status.STATUS_UNSPECIFIED]: "STATUS_UNSPECIFIED",
  [Status.STATUS_ACTIVE]: "STATUS_ACTIVE",
  [Status.STATUS_INACTIVE]: "STATUS_INACTIVE",
} as const
```

**验证元数据** — `buf/validate` 注解生成为带类型的常量对象：

```typescript
export const PersonRules = {
  name: { required: false, type: "string", minLength: 1, maxLength: 100 },
  age: { required: false, type: "integer", minimum: 0, maximum: 150 },
  email: { required: false, type: "string", format: "email" },
} as const
```

**跨文件 import** — 在其他 `.pb.ts` 文件中定义的类型会自动生成 import 语句：

```typescript
import { Status } from "./person.pb.js"
```

### 类型映射

| Proto 类型                    | TypeScript 类型     | 说明                        |
| ----------------------------- | ------------------- | --------------------------- |
| int32, uint32, float, double  | `number`            |                             |
| int64, uint64                 | `string`            | 避免 JS 精度丢失             |
| bool                          | `boolean`           |                             |
| string                        | `string`            |                             |
| bytes                         | `string`            | base64 编码                 |
| enum                          | `enum` + `Record`   | 数字枚举 + 名称映射          |
| repeated T                    | `T[]`               |                             |
| optional T                    | `T \| undefined`    | 简写：`field?: T`           |
| message                       | `interface`         |                             |

### 验证生成产物

`testdata/compat/ts-test/` 中提供了自动化验证：

```bash
cd testdata/compat/ts-test

# 安装依赖（仅首次）
npm install

# 类型检查 — 对所有生成文件执行 tsc --noEmit
npm run typecheck

# 运行时测试 — 验证枚举值、名称映射、验证规则、跨文件 import
npm test
```

这些测试也通过 `go test ./testdata/compat/...`（TestTSTypeCheck、TestTSRuntime）集成到 Go 测试中，当本地有 Node.js 时自动调用 npm。

---

## 已知限制

以下 proto 特性尚未支持。遇到不支持的特性时，gcode 会报错退出，不会静默生成错误代码。

| 限制 | 严重程度 | 说明 |
| --- | --- | --- |
| 不支持 streaming RPC | 中 | service 定义中使用 `stream` 关键字会报错退出 |
| 不支持 `map<K,V>` | 中 | map 字段在解析阶段报错 |
| 不支持 `oneof` | 中 | 非合成的 oneof 字段在解析阶段报错 |
| 不支持 well-known types | 中 | `google.protobuf.*` 类型（如 `Timestamp`）会报错 |
| 不支持 proto2 | 低 | 仅接受 `syntax = "proto3"` |
| HTTP path param 不支持 | 低 | 生成的 handler 仅从请求体绑定数据，路径参数（如 `/users/:id`）需在 service 层手动提取 |
| Go 输出目录平铺 | 低 | Go 生成会把所有 Go 文件写入同一个输出 package 目录。一次生成中不支持同 basename 的 proto 文件，即使它们位于不同源子目录。 |
