# gcode Design Decisions

This document records the key architectural decisions for gcode, using ADR (Architecture Decision Record) style: **Problem → Constraints → Decision → Consequences**.

---

## D1: Why not use protoc-gen-go

**Problem**: Proto files already have an official Go code generation tool `protoc-gen-go`. Why implement our own?

**Constraints**:
- `protoc-gen-go` generates structs containing `protoimpl.MessageState`, `sizeCache`, `unknownFields` and other runtime fields, which are not suitable as DAO layer data structures
- Official generated code depends on `google.golang.org/protobuf` runtime, introducing unnecessary dependencies
- Official generated code's JSON tags don't match project conventions (camelCase vs snake_case)
- Cannot control gorm tags, validate rules, and other business-layer concerns via annotations

**Decision**: Implement a custom code generator that produces plain Go structs without introducing a protobuf runtime.

**Consequences**:
- Generated structs are ordinary Go structs, usable directly with GORM, JSON serialization, and HTTP binding
- Wire format compatibility must be self-maintained (regression-tested via `testdata/compat/pbgo/`)
- Must implement `MarshalBinary`/`UnmarshalBinary` ourselves

---

## D2: Why protocompile instead of protoc

**Problem**: There are multiple ways to parse proto files. Why choose `protocompile`?

**Constraints**:
- `protoc` is an external binary requiring separate installation, raising the barrier to use
- `protoc` communicates with generators via a plugin mechanism (stdin/stdout), adding architectural complexity
- Need direct access to proto AST and semantic information within the Go process

**Decision**: Use `github.com/bufbuild/protocompile`, a pure Go implementation that can be embedded directly in the binary.

**Consequences**:
- Users only need `go install` — no protoc installation required
- `gcode/options.proto` and `buf/validate/validate.proto` are embedded via `//go:embed` — no extra files needed
- Parse results are accessed directly in-process, no inter-process communication

---

## D3: Direct wire format implementation vs official reflection

**Problem**: Should serialization/deserialization use the official protobuf reflection mechanism or directly manipulate the wire format?

**Constraints**:
- The official reflection mechanism depends on `google.golang.org/protobuf`, conflicting with D1's "no runtime" goal
- Must guarantee full compatibility with the official protobuf binary format

**Decision**: Implement protobuf wire format encoding primitives directly in the `runtime/` package (varint, ZigZag, tag, length-delimited). Generated code calls these primitives directly.

**Consequences**:
- Generated `MarshalBinary`/`UnmarshalBinary` has no external runtime dependencies
- Wire format compatibility is verified byte-by-byte against `protoc-gen-go` output in `testdata/compat/compat_test.go`
- `runtime/` is a public package, importable by user projects

---

## D4: Two-stage intermediate model (model → transform.GoFile)

**Problem**: Why introduce two layers of intermediate representation between parser and render?

**Constraints**:
- Proto semantics (snake_case field names, nested messages, proto type system) differ significantly from Go semantics (CamelCase, flat types, Go type system)
- The render layer should not need to understand proto syntax details
- Naming conflict resolution and type mapping logic need to be centralized

**Decision**:
- `model.File`: proto semantic model, one-to-one with proto syntax, Go-agnostic
- `transform.GoFile`: Go semantic model, with naming conversion, type resolution, and nesting flattened

**Consequences**:
- The render layer only deals with Go concepts, keeping logic simple
- Naming conflict resolution (`GoCamelCase` + conflict suffix) is centralized in the transform layer
- Nested messages are flattened to top-level types; no nesting in generated Go code

---

## D5: Optional fields generate pointer types

**Problem**: How should proto3 `optional` fields represent "not set" semantics in Go?

**Constraints**:
- Proto3 scalar fields default to zero values, making it impossible to distinguish "not set" from "set to zero"
- The `optional` keyword introduces field presence semantics
- `optional bytes` has special semantics: nil means "not set", `[]byte{}` means "set to empty"

**Decision**:
- `optional` scalar/enum fields → generate `*T` (pointer type)
- `optional bytes` → generate `[]byte` (nil means not set, no double pointer needed)
- Non-optional fields keep their original type

**Consequences**:
- Users check nil to determine whether a field is set
- nil pointer fields are skipped during marshal (not written to wire)
- nil pointer fields skip validation during validate inheritance

---

## D6: Validate inheritance mechanism

**Problem**: How should create/update derived messages reuse validate rules from the source message?

**Constraints**:
- Derived message fields are a subset of source message fields; validate rules should be inherited automatically
- Optional fields (pointer types) in derived messages should not trigger validation when nil
- condition_fields (WHERE clause fields) are required in update scenarios and should not have zero-value guards
- required_fields (forced non-optional in create scenarios) also have no nil guard and are validated directly

**Decision**:
- The render layer tracks source messages via `create_source`/`update_source` annotations
- Locates source message validate rules in the global `MessageIndex`
- Optional fields (pointer types): generate `if p.Field != nil { ... }` guard
- condition_fields: disable zero-value guard (`name != ""`), validate directly

**Consequences**:
- Validate rules are written once in the source message; derived messages inherit automatically
- Derived message `Validate()` methods are semantically consistent with the source message

---

## D7: Two-stage pipeline for create/update derived messages

**Problem**: How should derived messages be generated from a proto file?

**Constraints**:
- Derived messages must exist as independent proto messages to be referenced by service RPCs
- Generated intermediate proto files can be reused by other tools (e.g. protoc-gen-go, buf), maintaining proto ecosystem compatibility
- CLI maintains single responsibility: gen-proto handles proto generation, gcode handles Go generation

**Decision**: Two-stage pipeline:
1. `gcode gen-proto`: reads `gcode.update_message`/`gcode.create_message` annotations, generates intermediate proto files (`*.update.proto`/`*.create.proto`)
2. `gcode`: processes all proto files (including generated intermediate protos) uniformly to generate Go code

**Consequences**:
- Derived messages are real proto messages, directly referenceable by service RPCs
- Two-stage decoupling: gen-proto only cares about proto generation, gcode only cares about Go generation
- Intermediate proto files record their source via `update_source`/`create_source` annotations for validate inheritance

---

## D8: RPC interface not bound to transport protocol

**Problem**: What should the generated service code contain?

**Constraints**:
- Different projects use different transport protocols (HTTP, gRPC, message queues)
- Route registration, middleware, and authentication are application-layer concerns, outside the scope of code generation
- Streaming RPCs require special handling and are not supported in the current version

**Decision**: Generate only Go interfaces with fixed method signatures: `Method(ctx context.Context, req *XxxRequest) (*XxxResponse, error)`. No routing, serialization, or client stubs.

**Consequences**:
- Users have full control over the transport layer (route path, HTTP method, middleware)
- Interfaces can be implemented by any transport protocol (HTTP, gRPC, test mocks)
- Streaming RPCs cause an error exit rather than being silently ignored

---

## D9: HTTP adapter design

**Problem**: How should HTTP handlers be generated while keeping the transport layer decoupled from the business layer?

**Constraints**:
- Handlers should not depend on concrete service implementation types
- Generated code cannot know which field corresponds to a path param (proto fields have no positional semantics), so path params are not supported; internal services use JSON body uniformly
- `Validate()` is a public method usable in any context; built-in handler invocation ensures uniform interception at the transport layer

**Decision**:
- Generate handler factory functions `XxxHandler(svc XxxService) gin.HandlerFunc` accepting an interface, not a concrete type
- Use `c.ShouldBindJSON` uniformly (enforces JSON body; no path param support)
- Handlers call `req.Validate()` built-in (after bind, before svc call); `Validate()` remains a public method for reuse in other contexts
- Use `c.Request.Context()` to propagate request context, preserving deadline/cancel/trace information

**Consequences**:
- Handlers are decoupled from service implementations; mocks can be injected in tests
- Route paths and HTTP methods are fully controlled by the user
- Validation runs automatically at the transport layer without preventing standalone calls in service or other contexts

---

## D10: Response format and HTTP status

**Problem**: How should HTTP status codes and business codes be separated?

**Constraints**:
- HTTP status is transport-layer semantics; business code is application-layer semantics
- Mixing them causes middleware, load balancers, and monitoring systems to misinterpret business errors

**Decision**: HTTP status always returns 200; business results are conveyed via the `code` field in the response body:
- Success: `{"code": 0, "data": {...}}`
- Error: `{"code": 500, "error": {"msg": "..."}}`

Two-layer error code mechanism:
1. `CodedError` interface: business errors implement `Code() int`; `ErrResponse` extracts it automatically; defaults to 500 otherwise
2. gin middleware: can completely replace the error response format for cross-cutting concerns

**Consequences**:
- Clients determine success/failure via the `code` field, not HTTP status
- Business layer customizes error codes by implementing `CodedError`, without modifying generated code

**Scope limitation**: This design targets internal business RPC handlers only. Infrastructure endpoints that must use HTTP status code semantics — health checks (`/healthz`, `/readyz`), load balancer probes, and monitoring scrapers — are outside the scope of gcode and should be implemented manually. Those are infrastructure concerns, not business logic.

---

## D11: Pluggable TagProvider interface

**Problem**: How should multiple struct tag types (json, gorm, potentially validate tags, etc.) be supported without hardcoding?

**Constraints**:
- json tags are built-in and required for all structs; omitempty/ignore logic is tightly coupled to json tag string construction and is not suitable for abstraction as a provider
- gorm tags are optional, generated only for messages configured with `(gcode.message).gorm`
- Future tag types may be needed (e.g. `mapstructure`, `yaml`)

**Decision**: Define a `TagProvider` interface. The render layer accepts `[]TagProvider` and calls each provider in order to generate tag fragments. json tags are built-in logic; gorm tags are implemented via `GormTagProvider`.

**Consequences**:
- Adding a new tag type only requires implementing the `TagProvider` interface — no changes to render core logic
- `_defaultProviders` includes `GormTagProvider`; users control gorm tag generation via annotations

---

## D12: Reusing buf/validate annotations

**Problem**: How should validate rules be defined? Design a custom annotation syntax or reuse an existing standard?

**Constraints**:
- A custom annotation syntax requires users to learn a new DSL
- buf/validate is a widely-used proto validation standard with clear semantics

**Decision**: Directly reuse `buf/validate/validate.proto` annotation syntax. Embed `buf/validate/validate.proto` in the binary so users can use `(buf.validate.field).*` annotations directly in their proto files.

**Consequences**:
- Users don't need to learn a new annotation syntax
- Validate rules are compatible with the buf/validate ecosystem (though the runtime is self-implemented, not dependent on the buf/validate runtime)
- `buf/validate/validate.proto` is injected via `embeddedResolver`; no buf toolchain installation required

---

## D13: c.Error + DefaultErrorHandler pattern

**Problem**: Should generated HTTP handlers write error responses directly, or propagate errors through the gin context?

**Constraints**:
- When handlers write responses directly (`c.JSON`), middleware cannot intercept errors, making it impossible to handle ValidationError (400) and business errors (500) uniformly
- Users may need to customize error response format (e.g. adding request_id, trace_id, internationalized messages)
- ValidationError should map to code 400; other errors should map to code 500 or CodedError.Code()
- When no error-handling middleware is registered, behavior must be clearly documented — errors must not be silently lost

**Decision**:
- All error paths in handlers use `_ = c.Error(err); return` — no direct response writing
- `httpruntime.DefaultErrorHandler()` serves as a gin middleware fallback: ValidationError → code 400, others → code 500 or CodedError.Code()
- The function's doc comment explicitly warns: without this middleware, error paths return HTTP 200 with an empty body

**Consequences**:
- Users can replace DefaultErrorHandler with a custom implementation without modifying generated code
- ValidationError automatically maps to code 400 — no per-handler repetition needed
- Error handling logic is centralized in middleware; handlers remain clean
- The risk of not registering DefaultErrorHandler is clearly communicated via documentation and code comments

---

## D14: gorm TableName inheritance and ToMap key semantics

**Problem**: Should create derived structs inherit `TableName()` from the source struct? What should `ToMap()` use as map keys?

**Constraints**:
- A create derived struct is intended to "insert a record into the source struct's table" — semantically it belongs to the same table
- GORM's `db.Create(&PersonCreate{...})` requires `TableName()` to find the correct table; without it, GORM infers the table name from the struct name (`person_creates`), causing errors
- GORM's `db.Updates(map)` uses map keys directly as database column names — it does not walk struct tags — so `ToMap()` keys must be column names
- Update derived structs use `db.Model(&Person{})` to specify the table explicitly; they do not need `TableName()`
- Create derived structs have `GormMessageOptions == nil` at the transform layer (derived protos have no gorm annotation); inheritance must happen at the render layer via `Context.MessageIndex`

**Decision**:
- Both the original struct and create derived structs generate `TableName()`; the create derived struct inherits `GormMessageOptions` at render time by looking up the source message in `Context.MessageIndex` (shallow copy — the caller's `GoFile` is not mutated)
- Update derived structs do not generate `TableName()`
- `ToMap()` keys use the `(gcode.field).gorm.column` override when present; otherwise the proto field name is used
- `genproto` copies field-level `gcode.field.gorm.column` annotations to derived protos, ensuring derived struct fields carry the correct column name information

**Consequences**:
- Users can call `db.Create(&PersonCreate{...})` directly without specifying `db.Model(&Person{})`
- `db.Model(&Person{}).Updates(req.ToMap())` correctly matches database column names even when column name overrides are present
- `(gcode.field).gorm.column` consistently affects both struct tags and `ToMap()` keys
- The render-layer inheritance uses shallow copy, producing no side effects — the caller's `GoFile` is not modified

---

## D15: TypeScript generation — pure types, no runtime serialization

**Problem**: Should gcode generate a full TypeScript SDK (HTTP client, serialization) or just type definitions?

**Constraints**:
- gcode's primary goal is Go code generation; TypeScript support is supplementary
- Frontend projects use diverse HTTP clients (fetch, axios, tRPC) and validation libraries (zod, yup, ajv)
- Proto annotations define validation rules that should be reusable across frontend libraries
- protobuf binary serialization on the frontend adds complexity with limited benefit for JSON-based APIs

**Decision**: Generate pure type definitions only:
- `interface` for proto messages (camelCase property names matching Go JSON tags)
- `enum` + name mapping `Record` for proto enums
- Validation metadata as typed `const` objects (library-agnostic format)
- ES Module format with `.js` extension imports for maximum compatibility

**Consequences**:
- Frontend gets type safety and validation metadata without being locked into a specific library
- No runtime serialization — frontend consumes data via JSON fetch, which is the dominant pattern
- Validation metadata can drive form validation, UI constraints, or be converted to zod/yup schemas
- `gen-ts` is a separate subcommand, cleanly decoupled from Go generation
