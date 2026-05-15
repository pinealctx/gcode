# gcode Getting Started

## What is gcode

gcode is a code generator that takes proto files as input and produces Go structs, serialization methods, validation logic, HTTP handlers, and TypeScript type definitions. It is designed for backend services that use protobuf as a schema language but do not need the full gRPC stack.

**What gcode generates:**

| Input | Output |
|---|---|
| Any `.proto` file | Go struct + `MarshalBinary` / `UnmarshalBinary` + `Validate()` |
| `.meta.proto` schema file | Three derived proto files (entity / create / update) via `gen-proto` |
| `.entity.proto` | Go struct with GORM tags + `TableName()` + `DeepClone()` |
| `.create.proto` | Go struct with `Validate()` + `ToEntity()` + `DeepClone()` |
| `.update.proto` | Go struct with `Validate()` + `ToMap()` + `ApplyTo()` + `DeepClone()` |
| Service definition | Go interface + gin HTTP handler factory functions |
| Any `.proto` file | TypeScript interfaces + enums + validation metadata |

**Scope and constraints:**

- Targets proto3 only. proto2 is not supported.
- Generates Go code for GORM-based persistence and gin-based HTTP services. Other ORMs or HTTP frameworks are not supported.
- Does not generate gRPC stubs. The generated Go interface is a plain Go interface, not a gRPC service.
- Unsupported proto features (streaming RPC, `map<K,V>`, `oneof`, well-known types) cause a generation-time error rather than silently producing incorrect code. See [Known Limitations](#known-limitations).

---

## Installation

```bash
go install github.com/pinealctx/gcode/cmd/gcode@latest
```

Verify:

```bash
gcode -h
```

CLI flags:

```
gcode [flags]                 Generate Go code from proto files
  -in string                  Input proto directory
  -out string                 Output directory

gcode version                 Print version information

gcode gen-proto [flags]       Generate entity/create/update proto files from schema (.meta.proto) files
  -in string                  Input proto directory (generated files are written to the same directory)

gcode gen-ts [flags]          Generate TypeScript type definitions from proto files
  -in string                  Input proto directory
  -out string                 Output directory
```

---

## Project Dependencies

Generated code depends on the following public packages. Install them in your project:

```bash
# Serialization runtime (required by *.pb.dao.go)
go get github.com/pinealctx/gcode/runtime

# Validation runtime (required by *.pb.dao.validate.go)
go get github.com/pinealctx/gcode/validateruntime

# HTTP adapter runtime (required by *.pb.http.go)
go get github.com/pinealctx/gcode/httpruntime
```

If you only generate structs and serialization code (no validate or HTTP), only `runtime` is needed.

> **Runtime import path is fixed**: Generated code always imports `github.com/pinealctx/gcode/runtime`, `github.com/pinealctx/gcode/validateruntime`, and `github.com/pinealctx/gcode/httpruntime`. These paths are hardcoded in the generator and cannot be customized. If you fork or rename the module, you will need to update the generated import paths accordingly — this is a major-version-level change.

---

## Minimal Example

**1. Write a proto file**

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

**2. Generate code**

```bash
gcode -in proto/ -out dao/
```

**3. Use the generated struct**

```go
import "myapp/dao"

u := &dao.User{Name: "Alice", Age: 30}

// Serialize to protobuf wire format
wire, err := u.MarshalBinary()

// Deserialize
var u2 dao.User
err = u2.UnmarshalBinary(wire)
```

> **Note**: The generated code imports `github.com/pinealctx/gcode/runtime`. Running `go build` before `go get` will produce a missing module error — that is expected. Run `go get` first, then build.

---

## Full Example

The following example walks through the complete flow from proto definition to a working HTTP service, based on real code in `testdata/compat/`.

> **Note**: The proto examples below are simplified to highlight annotation usage. Full proto files are in `testdata/compat/proto/`.

### Step 1: Write proto files

> **Note**: `gcode/options.proto` and `buf/validate/validate.proto` are embedded in the gcode binary. No extra installation or download needed — just import them directly in your proto files.

#### Message definition (with validate and derived message annotations)

```proto
// proto/person.meta.proto
syntax = "proto3";
package myapp;
option go_package = "myapp/dao;dao";

import "buf/validate/validate.proto";
import "gcode/options.proto";

// Mark this file as a schema source. gen-proto reads this file and generates
// person.entity.proto, person.create.proto, and person.update.proto.
option (gcode.schema) = {};

// Source message: do NOT use optional on fields here.
// Optionality is determined by the derived message annotations, not the source.
// All fields are plain (non-optional) — gen-proto controls pointer semantics in derived protos.
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

  // Generate update derived message: PersonUpdateByName
  // condition_fields are WHERE clause fields, non-pointer in derived struct, excluded from ToMap()
  option (gcode.update_message) = {
    name: "PersonUpdateByName"
    condition_fields: ["name"]
    ignore_fields: []
  };

  // Generate create derived message: PersonCreate
  // All fields default to pointer (optional) in the derived struct.
  // required_fields forces specific fields to non-pointer (required).
  option (gcode.create_message) = {
    name: "PersonCreate"
    ignore_fields: []
    required_fields: ["nickname"]
  };
}
```

#### Service definition

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

> **Note**: `person.entity.proto`, `person.create.proto`, and `person.update.proto` are generated by the `gen-proto` command in Step 2. Do not write them manually.

---

### Step 2: Generate derived proto files

`gen-proto` reads `.meta.proto` schema files and generates three types of proto files for each schema:

```bash
gcode gen-proto -in proto/
```

> **Schema file naming rule**: `gen-proto` identifies schema files exclusively by the `.meta.proto` suffix. This is the only naming convention it enforces. Files without this suffix — including `common.proto`, service protos, and any other shared definition files — are not processed by `gen-proto` directly. They are resolved automatically as dependencies when protocompile parses the `.meta.proto` files.
>
> If a `.meta.proto` file imports `common.proto`, the generated `*.create.proto` and `*.update.proto` will automatically include `import "common.proto"` — no manual import management needed.

Result:

```
proto/
  person.meta.proto         ← schema source (unchanged)
  common.proto              ← shared definitions (unchanged, not processed by gen-proto)
  person_service.proto      ← service definition (unchanged, not processed by gen-proto)
  person.entity.proto       ← generated: Person message (no validate, with gorm)
  person.update.proto       ← generated: PersonUpdateByName message (with validate)
  person.create.proto       ← generated: PersonCreate message (with validate)
```

- `person.entity.proto` — contains the `Person` struct definition with gorm annotations. No `buf.validate` annotations; `Person.Validate()` returns nil.
- `person.create.proto` — contains `PersonCreate` with validate annotations copied from the schema. `PersonCreate.Validate()` enforces all rules.
- `person.update.proto` — contains `PersonUpdateByName` with validate annotations copied from the schema. `PersonUpdateByName.Validate()` enforces all rules.

> **Note**: `gcode gen-proto` overwrites existing generated files on every run. Do not manually edit `*.entity.proto`, `*.create.proto`, or `*.update.proto` — changes will be lost on the next run.

---

### Step 3: Generate all Go files

```bash
gcode -in proto/ -out dao/
```

Result:

```
dao/
  person.entity.pb.dao.go           ← Person struct + serialization methods
  person.entity.pb.dao.validate.go  ← Person.Validate() — returns nil (no validate annotations)
  person.update.pb.dao.go           ← PersonUpdateByName struct + ToMap() + ApplyTo()
  person.update.pb.dao.validate.go  ← PersonUpdateByName.Validate()
  person.create.pb.dao.go           ← PersonCreate struct + ToEntity()
  person.create.pb.dao.validate.go  ← PersonCreate.Validate()
  person_service.pb.dao.go          ← request/response message structs
  person_service.pb.dao.validate.go
  person_service.pb.rpc.go          ← PersonService interface
  person_service.pb.http.go         ← gin handler factory functions
```

**Field pointer rules in derived structs:**

| Struct | Field kind | Go type | Notes |
|---|---|---|---|
| `PersonCreate` | in `required_fields` | `T` (non-pointer) | caller must provide a value |
| `PersonCreate` | all other fields | `*T` (pointer) | nil = not provided, skips validation |
| `PersonUpdateByName` | in `condition_fields` | `T` (non-pointer) | WHERE clause, excluded from `ToMap()` |
| `PersonUpdateByName` | all other fields | `*T` (pointer) | nil = not updating this field |

---

### Step 4: Use the generated struct

#### Serialization and deserialization

```go
p := &dao.Person{Name: "Alice", Age: 30, Email: "alice@example.com"}

// Serialize to protobuf wire format
wire, err := p.MarshalBinary()
if err != nil {
    log.Fatal(err)
}

// Deserialize (strict mode: duplicate fields return error)
var p2 dao.Person
if err := p2.UnmarshalBinary(wire); err != nil {
    log.Fatal(err)
}

// Deserialize (lenient mode: duplicate fields use last value)
var p3 dao.Person
if err := p3.UnmarshalBinaryLenient(wire); err != nil {
    log.Fatal(err)
}
```

> **JSON tag naming**: Proto field names use `snake_case` (e.g. `created_at`), but generated JSON tags use `camelCase` (e.g. `json:"createdAt"`), consistent with protoc-gen-go behavior.

#### Optional fields

`optional` fields are generated as pointer types; nil means "not set":

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

#### Enum types

Proto `enum` definitions generate Go `int32` type aliases and constants:

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

Generated Go code:

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

Use `(buf.validate.field).enum.defined_only = true` to reject undefined values — see [Annotations Reference](annotations.md).

#### Nested messages

Proto allows messages nested inside other messages. gcode flattens them to top-level Go types:

```proto
message Order {
  message Item {
    string product = 1;
    int32  quantity = 2;
  }
  Item item = 1;
}
```

Generated Go code (nested type name becomes `OrderItem`):

```go
type OrderItem struct {
    Product  string `json:"product"`
    Quantity int32  `json:"quantity"`
}

type Order struct {
    Item *OrderItem `json:"item"`
}
```

> **Naming rule**: `Parent_Child` in proto → `ParentChild` in Go (via GoCamelCase). The generated Go code has no nesting — all types are top-level.

---

### Step 5: Use Validate()

`Validate()` is generated for every message. For entity messages (from `*.entity.proto`), `Validate()` returns nil — they carry no validate annotations. Validation is meaningful on create/update messages:

```go
import (
    "errors"
    "fmt"

    "github.com/pinealctx/gcode/validateruntime"
    "myapp/dao"
)

// PersonCreate.Validate() enforces all rules from the schema
req := &dao.PersonCreate{Name: "", Age: 200}

if err := req.Validate(); err != nil {
    var ve *validateruntime.ValidationError
    if errors.As(err, &ve) {
        fmt.Printf("field=%s rule=%s msg=%s\n", ve.Field, ve.Rule, ve.Message)
        // field=name rule=min_len msg=length must be >= 1
    }
}

// PersonUpdateByName.Validate() also enforces all rules
upd := &dao.PersonUpdateByName{Name: "Alice"}
if err := upd.Validate(); err != nil { ... }
```

`Validate()` is a public method on every generated struct. You can call it anywhere — in service implementations, message queue consumers, batch imports, etc. The generated HTTP handler also calls `req.Validate()` automatically (after binding, before calling the service). The two don't conflict: the handler's built-in call ensures validation at the transport layer; the public method lets you reuse the same validation logic in any other context.

---

### Step 6: Use ToMap() (update scenario)

`PersonUpdateByName.ToMap()` returns only non-nil fields, suitable for GORM's `Updates` method for partial updates:

```go
age := int32(31)
req := &dao.PersonUpdateByName{
    Name: "Alice",  // condition_fields — excluded from ToMap()
    Age:  &age,     // only update age
}

// ToMap() returns map[string]any{"age": 31}
// Name is a condition_field and is not included in the map
db.Model(&dao.Person{}).Where("name = ?", req.Name).Updates(req.ToMap())
```

> **ToMap() key semantics**: `ToMap()` uses the gorm column name as the map key. If a field has a `(gcode.field).gorm.column` override, the key uses the overridden column name; otherwise the proto field name is used. This ensures `db.Updates(map)` matches the correct database column (GORM uses map keys directly as column names — it does not walk struct tags).
>
> **Validate rules**: Validate rules are defined in the schema (`.meta.proto`) and copied by `gen-proto` into the generated `*.create.proto` / `*.update.proto` files. The render layer reads them directly from those proto fields — no cross-file lookup. Optional fields (pointer types) skip validation when nil; condition_fields are validated without a zero-value guard. See [Annotations Reference — Validate Inheritance](annotations.md#validate-inheritance).

---

### Step 7: Use DeepClone()

Every generated struct has a `DeepClone()` method that returns a fully independent copy with no shared memory. This is useful when you need to preserve the original state before applying a mutation:

```go
// Preserve the original before applying an update
original := entity.DeepClone()
updateMsg.ApplyTo(entity)

// Compare original vs entity for diff, audit log, or optimistic-lock conflict detection
if original.Age != entity.Age {
    log.Printf("age changed: %d → %d", original.Age, entity.Age)
}
```

`DeepClone()` handles all field kinds correctly:
- Scalar and enum fields: copied by value
- Optional fields (`*T`): a new pointer is allocated — mutating the clone's field does not affect the original
- Bytes and repeated fields: new slices are allocated and contents are copied
- Nested message fields: recursively cloned
- Nil receiver: returns nil

---

### Step 8: Implement the RPC interface

Generated `PersonService` interface:

```go
// dao/person_service.pb.rpc.go (generated, do not edit)
type PersonService interface {
    CreatePerson(ctx context.Context, req *PersonCreate) (*CreatePersonResponse, error)
    GetPerson(ctx context.Context, req *GetPersonRequest) (*GetPersonResponse, error)
    UpdatePerson(ctx context.Context, req *PersonUpdateByName) (*UpdatePersonResponse, error)
    DeletePerson(ctx context.Context, req *DeletePersonRequest) (*DeletePersonResponse, error)
}
```

Implement the interface:

```go
type personServiceImpl struct {
    db *gorm.DB
}

func (s *personServiceImpl) CreatePerson(ctx context.Context, req *dao.PersonCreate) (*dao.CreatePersonResponse, error) {
    // Validate() is called automatically by the HTTP handler.
    // In non-HTTP contexts (direct calls, message queues), call it manually:
    // if err := req.Validate(); err != nil { return nil, err }
    return &dao.CreatePersonResponse{Id: "new-id"}, nil
}

// Implement remaining methods...
```

---

### Step 9: Register gin routes (full HTTP service)

> **gin dependency**: The generated `*.pb.http.go` files import [gin](https://github.com/gin-gonic/gin). Add it to your project:
> ```bash
> go get github.com/gin-gonic/gin
> ```

Generated handler factory functions accept a service interface and an optional list of interceptors, returning `gin.HandlerFunc`:

```go
// dao/person_service.pb.http.go (generated, do not edit)
func CreatePersonHandler(svc PersonService, interceptors ...handlerx.Interceptor[*PersonCreate, *CreatePersonResponse]) gin.HandlerFunc
func GetPersonHandler(svc PersonService, interceptors ...handlerx.Interceptor[*GetPersonRequest, *GetPersonResponse]) gin.HandlerFunc
func UpdatePersonHandler(svc PersonService, interceptors ...handlerx.Interceptor[*PersonUpdateByName, *UpdatePersonResponse]) gin.HandlerFunc
func DeletePersonHandler(svc PersonService, interceptors ...handlerx.Interceptor[*DeletePersonRequest, *DeletePersonResponse]) gin.HandlerFunc
```

The `interceptors` parameter is variadic — existing calls like `dao.CreatePersonHandler(svc)` continue to work without any changes.

Register routes:

```go
func main() {
    svc := &personServiceImpl{db: setupDB()}

    r := gin.New()

    // ⚠️  DefaultErrorHandler (or a custom equivalent) MUST be registered.
    // Generated handlers use c.Error(err)+return to propagate errors and do
    // not write their own error responses. Without this middleware, all error
    // paths silently return HTTP 200 with an empty body.
    r.Use(httpruntime.DefaultErrorHandler())

    // Route paths and HTTP methods are fully controlled by you
    r.POST("/persons",        dao.CreatePersonHandler(svc))
    r.GET("/persons/:id",     dao.GetPersonHandler(svc))
    r.PUT("/persons/:name",   dao.UpdatePersonHandler(svc))
    r.DELETE("/persons/:id",  dao.DeletePersonHandler(svc))

    r.Run(":8080")
}
```

**Response format** (provided by `httpruntime`):

```json
// Success
{"code": 0, "data": {"id": "new-id"}}

// Validation error (CodeValidationErr)
{"code": 1001, "error": {"msg": "length must be >= 1"}}

// Business error (CodeDefaultErr, or CodedError.Code())
{"code": 5000, "error": {"msg": "internal error"}}
```

Implement `httpruntime.CodedError` to return a custom error code:

```go
type AppError struct {
    code int
    msg  string
}

func (e *AppError) Error() string { return e.msg }
func (e *AppError) Code() int     { return e.code }

// This error produces code 404 instead of the default CodeDefaultErr (5000)
return nil, &AppError{code: 404, msg: "person not found"}
```

#### Example requests (curl)

```bash
# Create a person
curl -X POST http://localhost:8080/persons \
  -H "Content-Type: application/json" \
  -d '{"nickname": "alice", "email": "alice@example.com"}'
# → {"code": 0, "data": {"id": "new-id"}}

# Validation error (nickname too long)
curl -X POST http://localhost:8080/persons \
  -H "Content-Type: application/json" \
  -d '{"nickname": "this-name-is-way-too-long"}'
# → {"code": 1001, "error": {"msg": "length must be <= 10"}}

# Get a person
curl http://localhost:8080/persons/some-id
# → {"code": 0, "data": {"name": "Alice", "age": 30}}
```

#### Request body size limit

Each handler delegates to `httpruntime.NewHandler`, which calls `c.ShouldBindJSON` internally. gin does not impose a default body size limit. For production deployments, set a limit via middleware to prevent oversized payloads:

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

#### Configuring request timeouts

Generated handlers pass `c.Request.Context()` to the service layer. gin does not inject a deadline by default. To add a timeout, inject it via middleware:

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

#### Adding interceptors (optional)

Every generated handler has panic recovery built in — a panic in the service method is caught and converted to an error, so the server never crashes. On top of that, you can inject custom interceptors per route for logging, metrics, tracing, or any cross-cutting concern.

An interceptor has the signature:

```go
func(ctx context.Context, req *Req, next handlerx.Handler[*Req, *Resp]) (*Resp, error)
```

**Example: request logging interceptor**

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

**Registering routes with interceptors**

```go
logger := slog.Default()

// No interceptors — works exactly as before
r.POST("/persons", dao.CreatePersonHandler(svc))

// With a logging interceptor on a specific route
r.DELETE("/persons/:id", dao.DeletePersonHandler(svc,
    LoggingInterceptor[DeletePersonRequest, DeletePersonResponse](logger),
))
```

Interceptors are applied in the order they are passed, inside the built-in panic recovery layer. The service method is always the innermost call.

---

## Annotation Quick Reference

For detailed documentation and examples, see [Annotations Reference](annotations.md).

### Message-level annotations

| Annotation                                | Type     | Description                                                            |
| ----------------------------------------- | -------- | ---------------------------------------------------------------------- |
| `(gcode.message).gorm.table`              | string   | Override gorm table name                                               |
| `(gcode.update_message).name`             | string   | Name of the generated update derived message                           |
| `(gcode.update_message).condition_fields` | []string | WHERE clause fields, excluded from `ToMap()`                           |
| `(gcode.update_message).ignore_fields`    | []string | Fields excluded from the derived message                               |
| `(gcode.create_message).name`             | string   | Name of the generated create derived message                           |
| `(gcode.create_message).ignore_fields`    | []string | Fields excluded from the derived message                               |
| `(gcode.create_message).required_fields`  | []string | Fields that become non-pointer types (required) in the derived message |

### Field-level annotations

| Annotation                       | Type   | Description                                                          |
| -------------------------------- | ------ | -------------------------------------------------------------------- |
| `(gcode.field).json.omitempty`   | bool   | Generate `json:"field,omitempty"`                                    |
| `(gcode.field).json.ignore`      | bool   | Generate `json:"-"`                                                  |
| `(gcode.field).gorm.column`      | string | Override gorm column name                                            |
| `(gcode.field).validate_message` | string | Override the default error message for all constraints on this field |

### Validate annotations (buf/validate)

| Annotation                                | Applies to    | Description                                        |
| ----------------------------------------- | ------------- | -------------------------------------------------- |
| `(buf.validate.field).string.min_len`     | string        | Minimum byte length                                |
| `(buf.validate.field).string.max_len`     | string        | Maximum byte length                                |
| `(buf.validate.field).string.email`       | string        | Email format validation                            |
| `(buf.validate.field).string.uri`         | string        | URI format validation                              |
| `(buf.validate.field).string.pattern`     | string        | RE2 regex match                                    |
| `(buf.validate.field).string.in`          | string        | Allowed values (can be declared multiple times)    |
| `(buf.validate.field).string.not_in`      | string        | Disallowed values (can be declared multiple times) |
| `(buf.validate.field).int32.gte`          | int32         | Greater than or equal                              |
| `(buf.validate.field).int32.lte`          | int32         | Less than or equal                                 |
| `(buf.validate.field).int32.gt`           | int32         | Greater than                                       |
| `(buf.validate.field).int32.lt`           | int32         | Less than                                          |
| `(buf.validate.field).int32.in`           | int32         | Allowed values                                     |
| `(buf.validate.field).int32.not_in`       | int32         | Disallowed values                                  |
| `(buf.validate.field).int64.*`            | int64         | Same as int32 series                               |
| `(buf.validate.field).float.*`            | float32/64    | Same as int32 series (gte/lte/gt/lt)               |
| `(buf.validate.field).bytes.min_len`      | bytes         | Minimum byte count                                 |
| `(buf.validate.field).bytes.max_len`      | bytes         | Maximum byte count                                 |
| `(buf.validate.field).repeated.min_items` | repeated      | Minimum element count                              |
| `(buf.validate.field).repeated.max_items` | repeated      | Maximum element count                              |
| `(buf.validate.field).repeated.items.*`   | repeated      | Apply constraints to each element                  |
| `(buf.validate.field).enum.defined_only`  | enum          | Only allow defined enum values                     |
| `(buf.validate.field).enum.not_in`        | enum          | Disallowed enum numbers                            |
| `(buf.validate.field).required`           | message/bytes | Disallow nil / empty                               |
| `(buf.validate.field).message.required`   | message       | Nested message must not be nil                     |

---

## TypeScript Generation

gcode generates TypeScript type definitions from proto files, enabling type-safe frontend code with consistent validation metadata.

### Prerequisites

No additional dependencies. The `gen-ts` command uses the same proto parsing pipeline as Go generation.

### Generate TS files

If your proto files use `gcode.update_message` / `gcode.create_message` annotations, run `gen-proto` first (see Step 2 in the Go section above) to generate the derived proto files. Then:

```bash
gcode gen-ts -in proto/ -out ts/
```

Result:

```
ts/
  person.entity.pb.ts       ← Person interface + Status enum (no validation metadata)
  person.create.pb.ts       ← PersonCreate interface + PersonCreateRules validation metadata
  person.update.pb.ts       ← PersonUpdateByName interface + PersonUpdateByNameRules validation metadata
  person_service.pb.ts      ← request/response interfaces + validation metadata
```

### What is generated

**Interfaces** — proto messages become TypeScript interfaces with camelCase property names:

```typescript
export interface Person {
  name: string
  age: number
  status: Status
  scores: number[]
  nickname?: string  // optional field → T | undefined
}
```

**Enums** — proto enums become TypeScript enums with a name mapping record:

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

**Validation metadata** — `buf/validate` annotations become typed constant objects:

```typescript
export const PersonRules = {
  name: { required: false, type: "string", minLength: 1, maxLength: 100 },
  age: { required: false, type: "integer", minimum: 0, maximum: 150 },
  email: { required: false, type: "string", format: "email" },
} as const
```

**Cross-file imports** — types defined in another `.pb.ts` file are automatically imported:

```typescript
import { Status } from "./person.pb.js"
```

### Type mapping

| Proto type                    | TypeScript type     | Notes                        |
| ----------------------------- | ------------------- | ---------------------------- |
| int32, uint32, float, double  | `number`            |                              |
| int64, uint64                 | `string`            | Avoids JS precision loss     |
| bool                          | `boolean`           |                              |
| string                        | `string`            |                              |
| bytes                         | `string`            | base64 encoded               |
| enum                          | `enum` + `Record`   | Numeric enum + name mapping  |
| repeated T                    | `T[]`               |                              |
| optional T                    | `T \| undefined`    | Shorthand: `field?: T`       |
| message                       | `interface`         |                              |

### Verify generated output

The compatibility test suite in `testdata/compat/ts-test/` provides automated verification:

```bash
cd testdata/compat/ts-test

# Install dependencies (first time only)
npm install

# Type check — tsc --noEmit on all generated files
npm run typecheck

# Runtime tests — verify enum values, name mapping, validation rules, cross-file imports
npm test
```

These tests are also integrated into Go via `go test ./testdata/compat/...` (TestTSTypeCheck, TestTSRuntime), which automatically invokes npm when Node.js is available.

---

## Known Limitations

The following proto features are not supported. When unsupported features are encountered, gcode reports an error and exits rather than silently producing incorrect code.

| Limitation | Severity | Details |
| --- | --- | --- |
| Streaming RPC not supported | Medium | Service definitions with `stream` keyword cause an error exit |
| `map<K,V>` not supported | Medium | Map fields cause an error at parse time |
| `oneof` not supported | Medium | Non-synthetic oneof fields cause an error at parse time |
| Well-known types not supported | Medium | `google.protobuf.*` types (e.g. `Timestamp`) cause an error |
| proto2 not supported | Low | Only `syntax = "proto3"` is accepted |
| HTTP path params not supported | Low | Generated handlers bind from request body only. Extract path params (e.g. `/users/:id`) manually in the service layer. |
| Flat Go output directory | Low | Go generation writes all generated Go files into one output package directory. Proto files with the same basename are rejected in one generation run, even when they are in different source subdirectories. |
