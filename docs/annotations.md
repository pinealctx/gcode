# gcode Annotations Reference

This document provides detailed documentation for all annotations supported by gcode, with complete examples for each annotation type.

---

## Table of Contents

- [Prerequisites](#prerequisites)
- [Message-level annotations](#message-level-annotations)
  - [(gcode.message).gorm.table](#gcodemessagegormtable)
  - [Source message design principles](#source-message-design-principles)
  - [(gcode.update_message)](#gcodeupdate_message)
  - [(gcode.create_message)](#gcodecreate_message)
- [Field-level annotations](#field-level-annotations)
  - [(gcode.field).json.omitempty](#gcodefieldjsonomitempty)
  - [(gcode.field).json.ignore](#gcodefieldjsonignore)
  - [(gcode.field).gorm.column](#gcodefieldgormcolumn)
  - [(gcode.field).validate_message](#gcodefieldvalidate_message)
- [Validate annotations (buf/validate)](#validate-annotations-bufvalidate)
  - [string](#string)
  - [Numeric types (int32 / int64 / float)](#numeric-types-int32--int64--float)
  - [bytes](#bytes)
  - [repeated](#repeated)
  - [enum](#enum)
  - [message](#message)
- [DeepClone](#deepclone)

---

## Prerequisites

Import the required proto files before using gcode annotations:

```proto
import "gcode/options.proto";         // gcode.schema / gcode.message / gcode.field / update_message / create_message
import "buf/validate/validate.proto"; // buf.validate.field
```

Both files are embedded in the gcode binary. No extra installation needed.

For schema files (`.meta.proto`), also add the file-level schema marker:

```proto
option (gcode.schema) = {};  // marks this file as a schema source for gen-proto
```

> **Field count limit**: A message may have at most 128 non-repeated fields. Exceeding this limit causes a generation-time error. This is an intentional design constraint: a flat message with more than 128 fields is almost always a design problem. Consider using nested messages to group related fields, or `repeated` fields to represent multiple instances of the same type.

---

## Message-level annotations

### (gcode.message).gorm.table

Overrides the default GORM table name (which defaults to the snake_case plural of the struct name).

**Proto example**:

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

**Generated result**:

```go
type User struct {
    Name string `json:"name" gorm:"column:name"`
    Age  int32  `json:"age"  gorm:"column:age"`
}

func (User) TableName() string { return "sys_users" }
```

> **Note**: gorm tags are only generated when `(gcode.message).gorm` is configured. Without it, struct fields only have `json` tags, and no `gorm` tags or `TableName()` method are generated.

**create derived struct inherits TableName()**: If the source message has `gorm.table` configured, the derived struct generated via `(gcode.create_message)` automatically inherits the same `TableName()` and can be used directly with `db.Create`:

```go
// PersonCreate inherits Person's table name — insert directly
db.Create(&dao.PersonCreate{Nickname: "ali", Email: "ali@example.com"})
// INSERT INTO persons (nickname, email) VALUES ('ali', 'ali@example.com')
```

**update derived struct does not inherit TableName()**: `PersonUpdateByName` has no `TableName()` method. For update scenarios, specify the table explicitly via `db.Model(&Person{})`:

```go
db.Model(&dao.Person{}).Where("name = ?", req.Name).Updates(req.ToMap())
```

---

### Source message design principles

The source message (e.g. `Person`) serves as both a struct definition and a schema definition. When designing source messages for use with `create_message` and `update_message`, follow these principles:

**Do not use `optional` on source message fields.** Optionality is determined by the derived message, not the source. The source message defines what fields exist; the create/update annotations define whether each field is required or optional.

```proto
// ❌ Avoid: optional on source message fields
message Person {
  optional string nickname = 1;
  optional int32  level    = 2;
}

// ✅ Prefer: plain fields on source message
message Person {
  string nickname = 1;
  int32  level    = 2;
}
```

**Do not use `(buf.validate.field).required = true` on scalar/enum source fields.** The source message defines value constraints (e.g. `min_len`, `gte`, `defined_only`), not whether a field must be present. Presence is an orthogonal concern handled by `required_fields` (create) and `condition_fields` (update):

| Concern | Defined by | Example |
|---------|-----------|---------|
| **Constraint** (WHAT) | Source message validate rules | `min_len = 1`, `gte = 0`, `defined_only` |
| **Presence** (WHETHER) | create/update annotation | `required_fields`, `condition_fields` |

```proto
// ❌ Avoid: required on source scalar fields
message Person {
  string email = 1 [(buf.validate.field).string.min_len = 1,
                     (buf.validate.field).required = true];  // unnecessary
}

// ✅ Prefer: constraint on source, presence on derived
message Person {
  string email = 1 [(buf.validate.field).string.min_len = 1];  // constraint only
  option (gcode.create_message) = {
    required_fields: ["email"]  // presence enforced here
  };
}
```

> **Exception**: For message-type fields (e.g. `Address address = 1`), `(buf.validate.field).message.required = true` is valid on the source message — it constrains the nested message to be non-nil when present. This is a value constraint, not a presence check.

**Message-type fields are naturally nullable.** Proto3 message fields always have presence semantics (nil = not set). Both `gcode gen-proto` and the Go render layer handle message-type fields correctly:
- In generated proto: no `optional` keyword is emitted for message-type fields
- In Go validate: required message fields get a nil check (error if nil) followed by recursive validation

---

### (gcode.update_message)

Generates an update derived message from the current message, for partial update scenarios.

**Fields**:

| Field | Type | Description |
|-------|------|-------------|
| `name` | string | Name of the generated derived message (required) |
| `condition_fields` | []string | WHERE clause fields — non-pointer type in the derived message, excluded from `ToMap()` |
| `ignore_fields` | []string | Fields excluded from the derived message |

**Proto example**:

```proto
message Person {
  string name  = 1;
  int32  age   = 2;
  string email = 3;
  string role  = 4;

  option (gcode.update_message) = {
    name: "PersonUpdateByName"
    condition_fields: ["name"]   // name is the WHERE condition, excluded from ToMap()
    ignore_fields: ["role"]      // role is excluded from the derived message
  };
}
```

**Generated result** (after running `gcode gen-proto -in proto/` then `gcode`):

```go
// person.update.pb.dao.go (generated, do not edit)
type PersonUpdateByName struct {
    Name  string  `json:"name"`   // condition_field: non-pointer, required
    Age   *int32  `json:"age"`    // optional update field: pointer type
    Email *string `json:"email"`  // optional update field: pointer type
    // Role is excluded by ignore_fields
}

// ToMap() includes only non-nil fields, excluding condition_fields
func (p *PersonUpdateByName) ToMap() map[string]any {
    um := make(map[string]any)
    if p.Age != nil {
        um["age"] = *p.Age
    }
    if p.Email != nil {
        um["email"] = *p.Email
    }
    return um  // Name is not in the map
}
```

**Usage**:

```go
age := int32(31)
req := &dao.PersonUpdateByName{
    Name: "Alice",  // WHERE name = 'Alice'
    Age:  &age,     // only update age
}

db.Model(&dao.Person{}).Where("name = ?", req.Name).Updates(req.ToMap())
// equivalent to: UPDATE persons SET age = 31 WHERE name = 'Alice'
```

> **ToMap() key semantics**: `ToMap()` uses the gorm column name as the map key. If a field has a `(gcode.field).gorm.column` override, the key uses the overridden column name; otherwise the proto field name is used. This is because GORM's `Updates(map)` uses map keys directly as database column names — it does not walk struct tags.
>
> For example, if `created_at` has `gorm.column = "created_ts"`, the key in `ToMap()` is `"created_ts"`, not `"created_at"`.

#### Condition field convention

`condition_fields` are identified at code-generation time by the rule: **non-optional and non-repeated fields in the derived message are treated as condition fields**. This is an implicit convention, not an explicit proto annotation.

If you write a `*.update.proto` file manually (instead of using `gcode gen-proto`), you must follow this convention: condition fields must be non-optional (non-pointer) and non-repeated. Optional or repeated fields will be treated as update fields and included in `ToMap()`, which is likely not what you want.

> **Recommendation**: Always use `gcode gen-proto` to generate `*.update.proto` files. Do not write them manually.

#### Cross-package references

`gcode gen-proto` propagates imports from the schema file to the generated `*.create.proto` and `*.update.proto`. If the schema file imports `common.proto`, both derived files will also import `common.proto` — no manual import management needed.

```proto
// common.proto — shared enum definitions
enum Status { STATUS_UNSPECIFIED = 0; STATUS_ACTIVE = 1; STATUS_INACTIVE = 2; }

// user.meta.proto — schema file
import "common.proto";
message User {
  Status status = 1;
  option (gcode.update_message) = { name: "UserUpdate" condition_fields: ["status"] };
}
// Generated user.update.proto automatically includes: import "common.proto";
```

The mechanism is straightforward: `gen-proto` only parses `.meta.proto` files. Any file imported by a schema file (e.g. `common.proto`) is resolved automatically by protocompile as a dependency. The generated create/update protos inherit all non-system imports from the schema file directly.

The `Validate()` method of an update derived message uses validate rules copied from the schema by `gen-proto`. The rules are read directly from the derived message's own proto fields — no cross-file lookup. Behavior:

- **Optional fields (pointer types)**: nil values skip validation — no rules are triggered
- **condition_fields**: validated without a zero-value guard — even an empty string triggers `min_len`
- **Fields excluded by ignore_fields**: not included in the derived message — rules are completely skipped

```go
req := &dao.PersonUpdateByName{
    Name: "",    // condition_field, validated directly → triggers min_len error
    Age:  nil,   // optional field, nil → skips validation
}
err := req.Validate()
// err: field=name rule=min_len msg=length must be >= 1
```

#### ApplyTo() method

The update derived message generates an `ApplyTo()` method that merges non-nil fields into an existing source entity. This is useful for in-memory/cache operations where you want to apply a partial update without going through the database:

```go
// Load existing entity from cache or DB
person := cache.Get(key)  // *dao.Person

// Apply partial update — only non-nil fields are overwritten
req.ApplyTo(person)  // condition field "name" is NOT applied

// person now has updated fields; nil fields in req are untouched
cache.Set(key, person)
```

`ApplyTo()` handles pointer type differences between the update and source structs:
- **Optional scalar/enum** (`*T` → `T`): nil-guard + dereference — only set when provided
- **Optional pointer** (`*T` → `*T`): nil-guard + pointer assign — shared reference (modifying one affects the other)
- **Repeated/bytes** (`[]T`): nil-guard — distinguishes "not provided" (nil) from "set to empty"

> **Memory semantics**: Like `ToEntity()`, `ApplyTo()` is not a deep copy. Ptr-to-ptr fields and repeated/bytes fields are assigned by reference, sharing memory between the update struct and the entity.

---

### (gcode.create_message)

Generates a create derived message from the current message, for insert scenarios.

**Fields**:

| Field | Type | Description |
|-------|------|-------------|
| `name` | string | Name of the generated derived message (required) |
| `ignore_fields` | []string | Fields excluded from the derived message |
| `required_fields` | []string | Fields forced to non-pointer type (required), even if the source field is `optional` |

**Proto example**:

```proto
message Person {
  string          name       = 1;
  int32           age        = 2;
  optional string nickname   = 3;  // optional in source message
  int64           created_at = 4;

  option (gcode.create_message) = {
    name: "PersonCreate"
    ignore_fields: ["created_at"]   // system field, not filled by user
    required_fields: ["nickname"]   // forced required, even though source is optional
  };
}
```

**Generated result**:

```go
// person.create.pb.dao.go (generated, do not edit)
type PersonCreate struct {
    Name     *string `json:"name"`     // non-optional source → pointer (create defaults all to optional)
    Age      *int32  `json:"age"`      // same
    Nickname string  `json:"nickname"` // required_fields → non-pointer, required
    // CreatedAt excluded by ignore_fields
}
```

> **Note**: In a create derived message, all fields except `required_fields` default to pointer types (optional), making it easy to fill in only some fields. Fields in `required_fields` are non-pointer — callers must provide a value.

#### Validate inheritance

`Validate()` rules for a create derived message come from the schema (`.meta.proto`), copied by `gen-proto` into the generated `*.create.proto` fields. The render layer reads them directly:

- **Optional fields (pointer types)**: nil skips validation
- **required_fields (non-pointer)**: validated directly, no nil guard
- **Fields excluded by ignore_fields**: not included in the derived message
- **condition_fields**: create_message has no condition_fields — not applicable

```go
req := &dao.PersonCreate{
    Nickname: "",  // required_field, validated directly → triggers min_len (if source has min_len)
    Name:     nil, // optional field, nil → skips validation
}
```

#### ToEntity() method

The create derived message generates a `ToEntity()` method that converts the create struct to the source entity type. This is useful for building an entity in memory before persisting it:

```go
req := &dao.PersonCreate{
    Nickname: "alice",
    Email:    strPtr("alice@example.com"),
}

person := req.ToEntity()  // returns *dao.Person
person.CreatedAt = time.Now().Unix()  // fill server-generated fields

db.Create(person)
cache.Set(key, person)
```

`ToEntity()` handles pointer type differences between the create and source structs:
- **Required field** (`T` → `*T`): copy-then-take-address — memory-isolated from the create struct
- **Optional scalar/enum** (`*T` → `T`): nil-guard + dereference — zero value if not provided
- **Optional pointer** (`*T` → `*T`): nil-guard + pointer assign — shared reference (modifying one affects the other)
- **Repeated/bytes** (`[]T`): direct assign — shared backing array

> **Memory semantics**: `ToEntity()` is a type conversion, not a deep copy. Repeated and bytes fields share the backing array with the create struct, and ptr-to-ptr fields share the pointer target. If you need full isolation, use `entity := req.ToEntity(); clone := entity.DeepClone()`.

Fields excluded by `ignore_fields` remain at their zero values in the returned entity.

---

## Field-level annotations

### (gcode.field).json.omitempty

Generates `json:"field_name,omitempty"` tag. Zero-value fields are omitted during JSON serialization.

> **JSON tag naming**: Proto field names are snake_case (e.g. `created_at`), but the generated json tag defaults to camelCase (`json:"createdAt"`), consistent with protoc-gen-go behavior.

**Proto example**:

```proto
message Response {
  string data  = 1;
  string error = 2 [(gcode.field) = { json: { omitempty: true } }];
}
```

**Generated result**:

```go
type Response struct {
    Data  string `json:"data"`
    Error string `json:"error,omitempty"`  // omitted when empty string
}
```

> **omitempty with optional fields**: Optional fields are generated as pointer types. `omitempty` takes effect when the pointer is nil (field is omitted), but does NOT take effect for a non-nil pointer to a zero value (e.g. `&0`, `&""`). This is consistent with proto3 field presence semantics: nil means "not set", while `&0` means "explicitly set to 0".

---

### (gcode.field).json.ignore

Generates `json:"-"` tag. The field is completely ignored during both JSON serialization and deserialization.

**Proto example**:

```proto
message User {
  string name     = 1;
  string password = 2 [(gcode.field) = { json: { ignore: true } }];
}
```

**Generated result**:

```go
type User struct {
    Name     string `json:"name"`
    Password string `json:"-"`  // excluded from JSON output and input
}
```

> **Bidirectional ignore**: `json:"-"` ignores the field in both Marshal and Unmarshal — not just during serialization. Suitable for passwords, internal state, or any field that should never be exposed externally.

---

### (gcode.field).gorm.column

Overrides the default GORM column name (which defaults to the snake_case of the field name).

**Proto example**:

```proto
message User {
  option (gcode.message) = { gorm: {} };  // enable gorm tag generation

  string name       = 1;
  string created_by = 2 [(gcode.field) = { gorm: { column: "creator" } }];
}
```

**Generated result**:

```go
type User struct {
    Name      string `json:"name"      gorm:"column:name"`
    CreatedBy string `json:"createdBy" gorm:"column:creator"`  // overrides default column name
}
```

> **Effect on ToMap()**: `(gcode.field).gorm.column` also affects the map key in the update derived struct's `ToMap()`. When a column name override is present, `ToMap()` uses the overridden name as the key, ensuring `db.Updates(map)` matches the correct database column.

---

### (gcode.field).validate_message

Overrides the default error message for all validate constraints on this field. When set, all rules on this field use this message instead of their individual defaults.

**Proto example**:

```proto
message LoginRequest {
  string username = 1 [
    (buf.validate.field).string.min_len = 1,
    (buf.validate.field).string.max_len = 50,
    (gcode.field) = { validate_message: "invalid username format" }
  ];
}
```

**Error message comparison**:

```go
// Without validate_message (default messages):
// field=username rule=min_len msg=length must be >= 1

// With validate_message:
// field=username rule=min_len msg=invalid username format
// field=username rule=max_len msg=invalid username format
```

> **Note**: `validate_message` overrides messages for **all** rules on the field. It cannot be set per individual rule.

---

## Validate annotations (buf/validate)

Validate annotations reuse `buf/validate` annotation syntax to generate `Validate() error` methods. The error type is `*validateruntime.ValidationError`, containing `Field`, `Rule`, and `Message` fields.

```go
if err := req.Validate(); err != nil {
    var ve *validateruntime.ValidationError
    if errors.As(err, &ve) {
        fmt.Printf("field=%s rule=%s msg=%s\n", ve.Field, ve.Rule, ve.Message)
    }
}
```

---

### string

#### min_len / max_len — byte length limits

```proto
message CreateUserRequest {
  string username = 1 [
    (buf.validate.field).string.min_len = 3,
    (buf.validate.field).string.max_len = 20
  ];
}
```

Triggered when: `len(username) < 3` or `len(username) > 20` (byte length, not character count).

#### email — email format

```proto
message User {
  string email = 1 [(buf.validate.field).string.email = true];
}
```

Triggered when: value does not match email format (`user@example.com`).

#### uri — URI format

```proto
message Config {
  string webhook_url = 1 [(buf.validate.field).string.uri = true];
}
```

Triggered when: value does not match URI format (must include scheme, e.g. `https://example.com`).

#### pattern — RE2 regex match

```proto
message Product {
  string sku = 1 [(buf.validate.field).string.pattern = "^[A-Z]{2}-[0-9]{4}$"];
}
```

Triggered when: value does not match the regex (RE2 syntax).

#### in / not_in — allowed/disallowed values

`in` and `not_in` can be declared multiple times, one value per declaration:

```proto
message User {
  // only "admin", "user", or "guest" allowed
  string role = 1 [
    (buf.validate.field).string.in = "admin",
    (buf.validate.field).string.in = "user",
    (buf.validate.field).string.in = "guest"
  ];

  // empty string and "root" disallowed
  string username = 2 [
    (buf.validate.field).string.not_in = "",
    (buf.validate.field).string.not_in = "root"
  ];
}
```

---

### Numeric types (int32 / int64 / float)

int32, int64, and float32/64 use the same constraint names — just replace the type prefix.

#### gte / lte — range (inclusive)

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

#### gt / lt — range (exclusive)

```proto
message Order {
  int64 amount = 1 [
    (buf.validate.field).int64.gt = 0   // amount > 0, zero not allowed
  ];
}
```

#### in / not_in — allowed/disallowed values

```proto
message Config {
  int32 type_id = 1 [
    (buf.validate.field).int32.not_in = 0,   // disallow 0 (uninitialized)
    (buf.validate.field).int32.not_in = -1   // disallow -1 (invalid)
  ];
}
```

---

### bytes

#### min_len / max_len — byte count limits

```proto
message File {
  bytes content = 1 [
    (buf.validate.field).bytes.min_len = 1,
    (buf.validate.field).bytes.max_len = 1048576  // max 1MB
  ];
}
```

#### required — disallow nil or empty

```proto
message Avatar {
  bytes data = 1 [(buf.validate.field).required = true];
}
```

Triggered when: `data == nil` (not set).

> **Note**: `optional bytes` fields are generated as `[]byte` (not `*[]byte`). nil means "not set"; `[]byte{}` means "set to empty". The `required` constraint checks for nil, not for empty slice.

---

### repeated

#### min_items / max_items — element count limits

```proto
message BatchRequest {
  repeated string ids = 1 [
    (buf.validate.field).repeated.min_items = 1,
    (buf.validate.field).repeated.max_items = 100
  ];
}
```

#### items — apply constraints to each element

`items` supports the same constraints as the corresponding scalar type, including `in`/`not_in`:

```proto
message TagList {
  repeated string tags = 1 [
    (buf.validate.field).repeated.min_items = 1,
    (buf.validate.field).repeated.items.string.min_len = 1,   // each tag non-empty
    (buf.validate.field).repeated.items.string.max_len = 50,  // each tag max 50 bytes
    (buf.validate.field).repeated.items.string.not_in = "admin"  // each tag not "admin"
  ];
  repeated int32 scores = 2 [
    (buf.validate.field).repeated.items.int32.in = 1,  // each score must be 1, 2, or 3
    (buf.validate.field).repeated.items.int32.in = 2,
    (buf.validate.field).repeated.items.int32.in = 3
  ];
}
```

Triggered when: any element fails its constraint. The error field name is `tags[i]` (e.g. `tags[2]`).

---

### enum

#### defined_only — only allow defined enum values

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

Triggered when: `status` value is not in `{0, 1, 2}` (prevents passing undefined integer values).

---

### message

#### message.required — nested message must not be nil

```proto
message Order {
  Address shipping_address = 1 [(buf.validate.field).message.required = true];
}
```

Triggered when: `shipping_address == nil` (nested message not set).

> **Note**: `(buf.validate.field).required` and `(buf.validate.field).message.required` have the same effect on message-type fields — both check whether the field is nil.

---

## DeepClone

Every generated message struct has a `DeepClone()` method that returns a fully independent copy with no shared memory between the clone and the original.

### Signature

```go
func (x *Msg) DeepClone() *Msg
```

- Returns `nil` when called on a `nil` receiver.
- All pointer, slice, and nested message fields are recursively copied so that mutating the clone never affects the original.

### Field handling

| Field type | Go type example | How it is cloned |
|---|---|---|
| scalar | `string`, `int32`, `bool` | shallow copy is sufficient (value type) |
| enum | `Status` | shallow copy is sufficient (int32 alias) |
| bytes (singular) | `[]byte` | `make` + `copy` |
| bytes (HasPresence) | `[]byte` (nil = absent) | `make` + `copy` when non-nil |
| optional scalar/enum | `*string`, `*int32`, `*Status` | allocate new pointer: `v := *p.F; clone.F = &v` |
| message | `*Address` | recursive `DeepClone()`, nil is preserved |
| repeated scalar/enum | `[]int32`, `[]Status` | `make` + `copy` |
| repeated bytes | `[][]byte` | `make` outer slice; `make` + `copy` each element |
| repeated message | `[]*Address` | `make` outer slice; recursive `DeepClone()` per element |

### Typical usage

```go
// Preserve the original entity before applying an update.
original := entity.DeepClone()
updateMsg.ApplyTo(entity)
// Compare original vs entity for diff, audit log, or optimistic-lock conflict detection.
```
