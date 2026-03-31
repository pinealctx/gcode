# gcode 注解参考

本文档详细说明 gcode 支持的所有注解，每个注解类型提供完整示例。

---

## 目录

- [使用前提](#使用前提)
- [message 级注解](#message-级注解)
  - [(gcode.message).gorm.table](#gcodeMessagegormtable)
  - [(gcode.update_message)](#gcodeupdate_message)
  - [(gcode.create_message)](#gcodecreate_message)
- [field 级注解](#field-级注解)
  - [(gcode.field).json.omitempty](#gcodefield-jsonomiempty)
  - [(gcode.field).json.ignore](#gcodefield-jsonignore)
  - [(gcode.field).gorm.column](#gcodefieldgormcolumn)
  - [(gcode.field).validate_message](#gcodefield-validate_message)
- [validate 注解（buf/validate）](#validate-注解bufvalidate)
  - [string 类型](#string-类型)
  - [数值类型（int32 / int64 / float）](#数值类型int32--int64--float)
  - [bytes 类型](#bytes-类型)
  - [repeated 类型](#repeated-类型)
  - [enum 类型](#enum-类型)
  - [message 类型](#message-类型)

---

## 使用前提

在 proto 文件中使用 gcode 注解前，需要导入对应的 proto 文件：

```proto
import "gcode/options.proto";       // gcode.message / gcode.field / update_message / create_message
import "buf/validate/validate.proto"; // buf.validate.field
```

两个文件均已嵌入 gcode 二进制，无需额外安装。

> **字段数量限制**：每个 message 最多支持 128 个非 repeated 字段，超出限制会在生成阶段报错。这是有意为之的设计约束：超过 128 个非 repeated 字段的扁平 message 几乎都是设计问题。建议用嵌套 message 对相关字段分组，或用 `repeated` 字段表示同类型的多个实例。

---

## message 级注解

### (gcode.message).gorm.table

覆盖 GORM 的默认表名（默认为 struct 名的蛇形复数形式）。

**proto 示例**：

```proto
import "gcode/options.proto";

message User {
  option (gcode.message) = {
    gorm: { table: "sys_users" }
  };

  string name = 1;
  int32  age  = 2;
}
```

**生成结果**：

```go
type User struct {
    Name string `json:"name" gorm:"column:name"`
    Age  int32  `json:"age"  gorm:"column:age"`
}

// GORM 使用此方法获取表名
func (User) TableName() string { return "sys_users" }
```

> **注意**：gorm tag 仅在配置了 `(gcode.message).gorm` 时生成。未配置时，struct 字段只有 `json` tag，不生成 `gorm` tag 和 `TableName()` 方法。

**create 派生 struct 继承 TableName()**：如果源 message 配置了 `gorm.table`，通过 `(gcode.create_message)` 生成的派生 struct 会自动继承相同的 `TableName()`，可直接用于 `db.Create`：

```go
// PersonCreate 继承 Person 的表名，可直接插入
db.Create(&dao.PersonCreate{Nickname: "ali", Email: "ali@example.com"})
// INSERT INTO persons (nickname, email) VALUES ('ali', 'ali@example.com')
```

**update 派生 struct 不继承 TableName()**：`PersonUpdateByName` 没有 `TableName()` 方法，update 场景通过 `db.Model(&Person{})` 显式指定表：

```go
db.Model(&dao.Person{}).Where("name = ?", req.Name).Updates(req.ToMap())
```

---

### (gcode.update_message)

从当前 message 生成一个 update 派生 message，用于部分更新场景。

**字段说明**：

| 字段 | 类型 | 说明 |
|------|------|------|
| `name` | string | 生成的派生 message 名称（必填） |
| `condition_fields` | []string | WHERE 条件字段，在派生 message 中为非指针类型（必填），不写入 `ToMap()` |
| `ignore_fields` | []string | 不包含在派生 message 中的字段 |

**proto 示例**：

```proto
message Person {
  string name  = 1;
  int32  age   = 2;
  string email = 3;
  string role  = 4;

  option (gcode.update_message) = {
    name: "PersonUpdateByName"
    condition_fields: ["name"]       // name 是 WHERE 条件，不写入 ToMap()
    ignore_fields: ["role"]          // role 不包含在派生 message 中
  };
}
```

**生成结果**（运行 `gcode gen-proto -in proto/` 后生成 `person.update.proto`，再运行 `gcode` 生成）：

```go
// person.update.pb.dao.go（生成，勿手动修改）
type PersonUpdateByName struct {
    Name  string  `json:"name"`   // condition_field：非指针，必填
    Age   *int32  `json:"age"`    // 可选更新字段：指针类型
    Email *string `json:"email"`  // 可选更新字段：指针类型
    // Role 被 ignore_fields 排除，不在此 struct 中
}

// ToMap() 只包含非 nil 字段，且排除 condition_fields
func (p *PersonUpdateByName) ToMap() map[string]any {
    um := make(map[string]any)
    if p.Age != nil {
        um["age"] = *p.Age
    }
    if p.Email != nil {
        um["email"] = *p.Email
    }
    return um  // Name 不在 map 中
}
```

**使用示例**：

```go
age := int32(31)
req := &dao.PersonUpdateByName{
    Name: "Alice",  // WHERE name = 'Alice'
    Age:  &age,     // 只更新 age
}

db.Model(&dao.Person{}).Where("name = ?", req.Name).Updates(req.ToMap())
// 等价于：UPDATE persons SET age = 31 WHERE name = 'Alice'
```

> **ToMap() key 说明**：`ToMap()` 的 map key 使用 gorm column 名。如果字段配置了 `(gcode.field).gorm.column` 覆盖，key 使用覆盖后的列名；无覆盖时使用 proto 字段名。这是因为 GORM 的 `Updates(map)` 直接将 map key 作为数据库列名，不走 struct tag 映射。
>
> 例如，若 `created_at` 字段配置了 `gorm.column = "created_ts"`，则 `ToMap()` 中该字段的 key 为 `"created_ts"` 而非 `"created_at"`。

#### validate 继承行为 {#validate-继承行为}

update 派生 message 的 `Validate()` 自动继承源 message 的 validate 规则，但行为有以下差异：

- **可选字段（指针类型）**：值为 nil 时跳过校验，不触发 validate 规则
- **condition_fields**：不做零值守卫，直接校验（即使值为空字符串也会触发 min_len 规则）
- **ignore_fields 排除的字段**：不参与 validate 继承，规则完全跳过

```go
req := &dao.PersonUpdateByName{
    Name: "",    // condition_field，直接校验 → 触发 min_len 错误
    Age:  nil,   // 可选字段，nil → 跳过校验
}
err := req.Validate()
// err: field=name rule=min_len msg=length must be >= 1
```

---

### (gcode.create_message)

从当前 message 生成一个 create 派生 message，用于插入场景。

**字段说明**：

| 字段 | 类型 | 说明 |
|------|------|------|
| `name` | string | 生成的派生 message 名称（必填） |
| `ignore_fields` | []string | 不包含在派生 message 中的字段 |
| `required_fields` | []string | 在派生 message 中强制为非指针类型（必填），即使源字段是 optional |

**proto 示例**：

```proto
message Person {
  string          name     = 1;
  int32           age      = 2;
  optional string nickname = 3;  // 源 message 中是 optional
  int64           created_at = 4;

  option (gcode.create_message) = {
    name: "PersonCreate"
    ignore_fields: ["created_at"]    // 系统字段，不由用户填写
    required_fields: ["nickname"]    // 强制必填，即使源字段是 optional
  };
}
```

**生成结果**：

```go
// person.create.pb.dao.go（生成，勿手动修改）
type PersonCreate struct {
    Name     *string `json:"name"`     // 源字段非 optional → 指针（create 派生默认全部可选）
    Age      *int32  `json:"age"`      // 同上
    Nickname string  `json:"nickname"` // required_fields → 非指针，必填
    // CreatedAt 被 ignore_fields 排除
}
```

> **说明**：create 派生 message 中，除 `required_fields` 外的字段默认全部为指针类型（可选），方便只填写部分字段。`required_fields` 中的字段强制为非指针类型，调用方必须提供值。

#### validate 继承行为

create 派生 message 的 `Validate()` 继承规则：

- **可选字段（指针类型）**：nil 时跳过校验
- **required_fields 字段（非指针）**：直接校验，不做 nil 守卫
- **ignore_fields 排除的字段**：不参与 validate 继承
- **condition_fields**：create_message 无此概念，不适用

```go
req := &dao.PersonCreate{
    Nickname: "",  // required_field，直接校验 → 触发 min_len 错误（如果源字段有 min_len）
    Name:     nil, // 可选字段，nil → 跳过校验
}
```

---

## field 级注解

### (gcode.field).json.omitempty

为字段生成 `json:"field_name,omitempty"` tag，JSON 序列化时零值字段不输出。

> **json tag 命名规则**：proto 字段名是 snake_case（如 `created_at`），生成的 json tag 默认是 camelCase（`json:"createdAt"`），与 protoc-gen-go 行为一致。gcode 目前不支持自定义 json key 格式。

**proto 示例**：

```proto
message Response {
  string data  = 1;
  string error = 2 [(gcode.field) = { json: { omitempty: true } }];
}
```

**生成结果**：

```go
type Response struct {
    Data  string `json:"data"`
    Error string `json:"error,omitempty"`  // 空字符串时不输出
}
```

> **optional 字段与 omitempty**：optional 字段生成指针类型，`omitempty` 对 nil 指针生效（字段不输出），但对非 nil 的零值指针（如 `&0`、`&""`）不生效——因为指针本身非零值，字段仍会输出。这与 proto3 的 field presence 语义一致：nil 表示"未设置"，`&0` 表示"明确设置为 0"。

---

### (gcode.field).json.ignore

为字段生成 `json:"-"` tag，JSON 序列化和反序列化时完全忽略此字段。

**proto 示例**：

```proto
message User {
  string name     = 1;
  string password = 2 [(gcode.field) = { json: { ignore: true } }];
}
```

**生成结果**：

```go
type User struct {
    Name     string `json:"name"`
    Password string `json:"-"`  // JSON 序列化时不输出，也不从 JSON 读取
}
```

> **双向忽略**：`json:"-"` 在序列化（Marshal）和反序列化（Unmarshal）时都忽略该字段，不只是序列化时忽略。适合密码、内部状态等不应暴露给外部的字段。

---

### (gcode.field).gorm.column

覆盖 GORM 的默认列名（默认为字段名的蛇形形式）。

**proto 示例**：

```proto
message User {
  option (gcode.message) = { gorm: {} };  // 启用 gorm tag 生成

  string name       = 1;
  string created_by = 2 [(gcode.field) = { gorm: { column: "creator" } }];
}
```

**生成结果**：

```go
type User struct {
    Name      string `json:"name"      gorm:"column:name"`
    CreatedBy string `json:"createdBy" gorm:"column:creator"`  // 覆盖默认列名
}
```

> **对 ToMap() 的影响**：`(gcode.field).gorm.column` 同时影响 update 派生 struct 的 `ToMap()` key。若字段有列名覆盖，`ToMap()` 使用覆盖后的列名作为 key，确保 `db.Updates(map)` 能正确匹配数据库列。

---

### (gcode.field).validate_message

覆盖该字段所有 validate 约束的默认错误消息。设置后，该字段的所有规则触发时都使用此消息，而非各规则的默认消息。

**proto 示例**：

```proto
message LoginRequest {
  string username = 1 [
    (buf.validate.field).string.min_len = 1,
    (buf.validate.field).string.max_len = 50,
    (gcode.field) = { validate_message: "用户名格式不正确" }
  ];
}
```

**生成结果对比**：

```go
// 不设置 validate_message 时（默认消息）：
// field=username rule=min_len msg=length must be >= 1

// 设置 validate_message 后：
// field=username rule=min_len msg=用户名格式不正确
// field=username rule=max_len msg=用户名格式不正确
```

> **注意**：`validate_message` 覆盖该字段的**所有**规则消息，不能针对单个规则单独覆盖。

---

## validate 注解（buf/validate）

validate 注解复用 `buf/validate` 的注解语法，生成 `Validate() error` 方法。错误类型为 `*validateruntime.ValidationError`，包含 `Field`、`Rule`、`Message` 三个字段。

```go
if err := req.Validate(); err != nil {
    var ve *validateruntime.ValidationError
    if errors.As(err, &ve) {
        fmt.Printf("field=%s rule=%s msg=%s\n", ve.Field, ve.Rule, ve.Message)
    }
}
```

---

### string 类型

#### min_len / max_len — 字节长度限制

```proto
message CreateUserRequest {
  string username = 1 [
    (buf.validate.field).string.min_len = 3,
    (buf.validate.field).string.max_len = 20
  ];
}
```

触发条件：`len(username) < 3` 或 `len(username) > 20`（字节长度，非字符数）。

#### email — 邮箱格式

```proto
message User {
  string email = 1 [(buf.validate.field).string.email = true];
}
```

触发条件：不符合邮箱格式（`user@example.com`）。

#### uri — URI 格式

```proto
message Config {
  string webhook_url = 1 [(buf.validate.field).string.uri = true];
}
```

触发条件：不符合 URI 格式（需包含 scheme，如 `https://example.com`）。

#### pattern — RE2 正则匹配

```proto
message Product {
  string sku = 1 [(buf.validate.field).string.pattern = "^[A-Z]{2}-[0-9]{4}$"];
}
```

触发条件：不匹配正则表达式（使用 RE2 语法）。

#### in / not_in — 枚举值限制

`in` 和 `not_in` 可多次声明，每次声明一个允许/禁止的值：

```proto
message User {
  // 只允许 "admin" / "user" / "guest"
  string role = 1 [
    (buf.validate.field).string.in = "admin",
    (buf.validate.field).string.in = "user",
    (buf.validate.field).string.in = "guest"
  ];

  // 禁止空字符串和 "root"
  string username = 2 [
    (buf.validate.field).string.not_in = "",
    (buf.validate.field).string.not_in = "root"
  ];
}
```

---

### 数值类型（int32 / int64 / float）

int32、int64、float32/64 使用相同的约束名，只需替换类型前缀。

#### gte / lte — 范围限制（含边界）

```proto
message Person {
  int32 age = 1 [
    (buf.validate.field).int32.gte = 0,    // age >= 0
    (buf.validate.field).int32.lte = 150   // age <= 150
  ];

  float rating = 2 [
    (buf.validate.field).float.gte = 0.0,
    (buf.validate.field).float.lte = 5.0
  ];
}
```

#### gt / lt — 范围限制（不含边界）

```proto
message Order {
  int64 amount = 1 [
    (buf.validate.field).int64.gt = 0   // amount > 0，不允许 0
  ];
}
```

#### in / not_in — 枚举值限制

```proto
message Config {
  int32 type_id = 1 [
    (buf.validate.field).int32.not_in = 0,   // 禁止 0（未初始化值）
    (buf.validate.field).int32.not_in = -1   // 禁止 -1（无效值）
  ];
}
```

---

### bytes 类型

#### min_len / max_len — 字节数限制

```proto
message File {
  bytes content = 1 [
    (buf.validate.field).bytes.min_len = 1,
    (buf.validate.field).bytes.max_len = 1048576  // 最大 1MB
  ];
}
```

#### required — 不允许 nil 或空

```proto
message Avatar {
  bytes data = 1 [(buf.validate.field).required = true];
}
```

触发条件：`data == nil`（未设置）。

> **注意**：`optional bytes` 字段生成为 `[]byte`（非 `*[]byte`），nil 表示未设置，`[]byte{}` 表示设置为空。`required` 约束检查 nil，不检查空切片。

---

### repeated 类型

#### min_items / max_items — 元素数量限制

```proto
message BatchRequest {
  repeated string ids = 1 [
    (buf.validate.field).repeated.min_items = 1,
    (buf.validate.field).repeated.max_items = 100
  ];
}
```

#### items — 对每个元素应用约束

`items` 下可使用与对应类型相同的约束：

```proto
message TagList {
  repeated string tags = 1 [
    (buf.validate.field).repeated.min_items = 1,
    (buf.validate.field).repeated.items.string.min_len = 1,   // 每个 tag 非空
    (buf.validate.field).repeated.items.string.max_len = 50   // 每个 tag 最长 50 字节
  ];
}
```

触发条件：任意元素不满足约束时，错误字段名为 `tags[i]`（如 `tags[2]`）。

---

### enum 类型

#### defined_only — 只允许已定义的枚举值

```proto
enum Status {
  STATUS_UNSPECIFIED = 0;
  STATUS_ACTIVE      = 1;
  STATUS_INACTIVE    = 2;
}

message User {
  Status status = 1 [(buf.validate.field).enum.defined_only = true];
}
```

触发条件：`status` 的值不在 `{0, 1, 2}` 中（防止传入未定义的整数值）。

---

### message 类型

#### message.required — 嵌套 message 不允许 nil

```proto
message Order {
  Address shipping_address = 1 [(buf.validate.field).message.required = true];
}
```

触发条件：`shipping_address == nil`（未设置嵌套 message）。

> **注意**：`(buf.validate.field).required` 和 `(buf.validate.field).message.required` 对 message 类型字段效果相同，两者均检查字段是否为 nil。
