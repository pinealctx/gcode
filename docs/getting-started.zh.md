# gcode 开箱即用指南

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
  -allow-json-unknown-fields  JSON 反序列化时允许未知字段
  -duplicate-singular string  重复标量字段策略：error|last-wins（默认 "error"）

gcode gen-proto [flags]       生成派生 proto 文件（*.update.proto / *.create.proto）
  -in string                  输入 proto 目录（生成文件写入同一目录）
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
// proto/person.proto
syntax = "proto3";
package myapp;
option go_package = "myapp/dao;dao";

import "buf/validate/validate.proto";
import "gcode/options.proto";

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
  optional string nickname = 4 [
    (buf.validate.field).string.min_len = 1,
    (buf.validate.field).string.max_len = 10
  ];

  // 生成 update 派生 message：PersonUpdateByName
  // condition_fields 是 WHERE 条件字段，不写入 ToMap()
  option (gcode.update_message) = {
    name: "PersonUpdateByName"
    condition_fields: ["name"]
    ignore_fields: []
  };

  // 生成 create 派生 message：PersonCreate
  // required_fields 在派生 message 中为非指针类型（必填）
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

> **注意**：`person.create.proto` 和 `person.update.proto` 由第二步的 `gen-proto` 命令生成，不需要手动编写。

---

### 第二步：生成派生 proto

`gen-proto` 读取 `gcode.update_message` / `gcode.create_message` 注解，生成派生 proto 文件：

```bash
gcode gen-proto -in proto/
```

生成结果：

```
proto/
  person.proto              ← 原始文件（不变）
  person_service.proto      ← 原始文件（不变）
  person.update.proto       ← 生成：PersonUpdateByName message
  person.create.proto       ← 生成：PersonCreate message
```

> **注意**：`gcode gen-proto` 每次运行会覆盖已有的 `*.update.proto` / `*.create.proto` 文件。不要手动修改这些生成文件，修改会在下次运行时丢失。

---

### 第三步：生成所有 Go 文件

```bash
gcode -in proto/ -out dao/
```

生成结果：

```
dao/
  person.pb.dao.go              ← Person struct + 序列化方法
  person.pb.dao.validate.go     ← Person.Validate() 方法
  person.update.pb.dao.go       ← PersonUpdateByName struct + ToMap()
  person.update.pb.dao.validate.go
  person.create.pb.dao.go       ← PersonCreate struct
  person.create.pb.dao.validate.go
  person_service.pb.dao.go      ← 请求/响应 message struct
  person_service.pb.dao.validate.go
  person_service.pb.rpc.go      ← PersonService interface
  person_service.pb.http.go     ← gin handler 工厂函数
```

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

```go
import (
    "errors"
    "fmt"

    "github.com/pinealctx/gcode/validateruntime"
    "myapp/dao"
)

p := &dao.Person{Name: "", Age: 200}

if err := p.Validate(); err != nil {
    var ve *validateruntime.ValidationError
    if errors.As(err, &ve) {
        fmt.Printf("field=%s rule=%s msg=%s\n", ve.Field, ve.Rule, ve.Message)
        // field=name rule=min_len msg=length must be >= 1
    }
}
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
> **validate 继承**：派生 message 的 `Validate()` 自动继承源 message 的规则。optional 字段（指针类型）为 nil 时跳过校验；condition_fields 不做零值守卫，直接校验。详见 [注解参考 — validate 继承行为](annotations.zh.md#validate-继承行为)。

---

### 第七步：实现 RPC interface

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

### 第八步：注册 gin 路由（完整 HTTP 服务）

> **gin 依赖**：生成的 `*.pb.http.go` 文件会导入 [gin](https://github.com/gin-gonic/gin)，需要在项目中安装：
> ```bash
> go get github.com/gin-gonic/gin
> ```

生成的 handler 工厂函数接收 service interface，返回 `gin.HandlerFunc`：

```go
// dao/person_service.pb.http.go（生成，勿手动修改）
func CreatePersonHandler(svc PersonService) gin.HandlerFunc
func GetPersonHandler(svc PersonService) gin.HandlerFunc
func UpdatePersonHandler(svc PersonService) gin.HandlerFunc
func DeletePersonHandler(svc PersonService) gin.HandlerFunc
```

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

// validate 错误（code 400）
{"code": 400, "error": {"msg": "length must be >= 1"}}

// 业务错误（code 500，或 CodedError.Code()）
{"code": 500, "error": {"msg": "internal error"}}
```

业务层可通过实现 `httpruntime.CodedError` 接口自定义错误 code：

```go
type AppError struct {
    code int
    msg  string
}

func (e *AppError) Error() string { return e.msg }
func (e *AppError) Code() int     { return e.code }

// 返回此 error 时，响应 code 为 404 而非默认 500
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
# → {"code": 400, "error": {"msg": "length must be <= 10"}}

# 查询 person
curl http://localhost:8080/persons/some-id
# → {"code": 0, "data": {"name": "Alice", "age": 30}}
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
