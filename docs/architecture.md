# gcode Architecture Overview

gcode is a pure Go CLI tool that generates Go code from `.proto` files. It requires no `protoc`, introduces no protobuf runtime, and produces ordinary Go structs that work directly with GORM, JSON serialization, and gin HTTP services.

---

## Pipeline

```
.meta.proto files (schema source — define validate rules, update_message / create_message annotations)
    │
    ▼
[gen-proto]     Pre-step: reads .meta.proto schema files,
                generates three types of proto files per schema:
                  *.entity.proto  — struct definition (no validate, with gorm)
                  *.create.proto  — create message (validate annotations copied from schema)
                  *.update.proto  — update message (validate annotations copied from schema)
                All generated files are written to the same directory.
    │
    ▼
.proto files (entity + create + update + service)
    │
    ▼
[source]        Scans directory, discovers all .proto files, stable sort
                (skips .meta.proto files — consumed only by gen-proto)
    │
    ▼
[parser]        Parses proto via protocompile: reads message/field/enum/
                service/comments/custom options, maps to semantic model
    │
    ▼
[model]         Intermediate semantic model (File / Message / Field /
                Enum / Service / RPC) — independent of proto and Go syntax
    │
    ▼
[transform]     model → Go intermediate representation (GoFile / GoMessage /
                GoField / GoEnum / GoService / GoRPCMethod),
                computes Go type names, field names, package names
    │
    ▼
[render]        Go IR → Go source string, go/format formatting, outputs []byte
    │
    ▼
Generated files
  *.entity.pb.dao.go           struct definitions + MarshalBinary/UnmarshalBinary
  *.entity.pb.dao.validate.go  Validate() — returns nil (no validate annotations on entity)
  *.create.pb.dao.go           struct + ToEntity()
  *.create.pb.dao.validate.go  Validate() — rules read directly from create proto fields
  *.update.pb.dao.go           struct + ToMap() + ApplyTo()
  *.update.pb.dao.validate.go  Validate() — rules read directly from update proto fields
  *.pb.rpc.go                  service interface definitions
  *.pb.http.go                 gin HTTP handler factory functions
```

TypeScript generation follows a parallel pipeline using the same parser and transform stages.
`.meta.proto` files are skipped; only `*.entity.proto`, `*.create.proto`, `*.update.proto`, and regular proto files are processed:

```
.proto files (entity / create / update / service)
    │
    ▼
[source → parser → model → transform]
    │
    ▼
[tsrender]       Go IR → TypeScript source string,
                 with TypeRegistry for cross-file import resolution
    │
    ▼
Generated files
  *.entity.pb.ts           interfaces, enums, enum name mapping (no validation metadata)
  *.create.pb.ts           interface + XxxRules validation metadata
  *.update.pb.ts           interface + XxxRules validation metadata
  *.pb.ts                  interfaces, enums, enum name mapping, validation metadata
```

---

## Layer Responsibilities

### source

Scans the specified directory for all `.proto` files. Applies stable sorting (ensures consistent output across runs) and validates path safety (prevents path traversal).

### parser

Parses proto files using the `protocompile` library. Responsibilities:
- Parse message, field, enum, service, and rpc definitions
- Read leading comments (`//` line comments and `/* */` block comments)
- Read custom options (`gcode.message`, `gcode.field`, `buf.validate.field`)
- Map results to `model.File`

Built-in `embeddedResolver`: embeds `gcode/options.proto` and `buf/validate/validate.proto` into the binary. No extra tools needed.

### model

Intermediate semantic model — the contract between parser and transform. Core types:

| Type            | Description                                                          |
| --------------- | -------------------------------------------------------------------- |
| `model.File`    | Complete semantic representation of one proto file                   |
| `model.Message` | Message definition with fields, comments, gcode/validate annotations |
| `model.Field`   | Field definition with type, optional flag, annotations               |
| `model.Enum`    | Enum definition with values and comments                             |
| `model.Service` | Service definition with RPCs and comments                            |
| `model.RPC`     | Single RPC method with request/response types and comments           |
| `model.Comment` | Comment content, `Lines []string`                                    |

### transform

Converts `model.File` to Go intermediate representation `transform.GoFile`. Responsibilities:
- Flatten nested messages (proto allows nesting; Go does not)
- Compute Go type names (`GoCamelCase`, resolve naming conflicts)
- Compute Go field names (proto snake_case → Go CamelCase)
- Resolve field types (scalar → Go primitive, message → pointer, optional → pointer)
- Validate `create_message` required_fields semantic constraints

### render

Renders `transform.GoFile` to Go source code. Four generation functions:

| Function              | Output file            | Description                                               |
| --------------------- | ---------------------- | --------------------------------------------------------- |
| `render.File`         | `*.pb.dao.go`          | struct definitions, MarshalBinary, UnmarshalBinary, ToMap |
| `render.ValidateFile` | `*.pb.dao.validate.go` | `Validate() error` methods                                |
| `render.RPCFile`      | `*.pb.rpc.go`          | service interface                                         |
| `render.HTTPFile`     | `*.pb.http.go`         | gin handler factory functions                             |

All functions call `go/format.Source` at the end to ensure consistent code style.

Proto leading comments are passed through to all generated code: structs/fields/enums (`*.pb.dao.go`), service interfaces/methods (`*.pb.rpc.go`), and HTTP handlers (`*.pb.http.go`).

### tsrender

Renders `transform.GoFile` to TypeScript source code. Uses a `TypeRegistry` to resolve cross-file type references and generate ES module import statements.

| Function          | Output file  | Content                                                    |
| ----------------- | ------------ | ---------------------------------------------------------- |
| `tsrender.TSFile` | `*.pb.ts`    | interfaces, enums, enum name mapping, validation metadata  |

Generated code is pure type definitions (no runtime serialization). Cross-file types are imported via relative paths with `.js` extension (e.g. `import { Status } from "./person.pb.js"`) for maximum module resolution compatibility.

### runtime

Protobuf wire format encoding primitives (varint, ZigZag, tag, length-delimited, size calculation). Generated `MarshalBinary`/`UnmarshalBinary` call this package directly, with no dependency on the official protobuf reflection mechanism. Public package, importable by user projects.

### validateruntime

Validation runtime helpers. Provides:
- `ValidationError` (with Field/Rule/Message fields)
- `IsEmail` / `IsURI` (replaceable package-level variables for test injection)
- `MatchPattern` (RE2 regex with `sync.Map` compilation cache)

Public package, importable by user projects.

### version

Build-time metadata resolution. Version, commit, and build time are resolved in priority order:
1. `-ldflags` overrides (custom builds)
2. Go module version + VCS metadata from `runtime/debug.ReadBuildInfo` (automatically embedded by Go 1.18+)

This ensures `go install github.com/pinealctx/gcode/cmd/gcode@latest` produces meaningful version output without any build script.

### httpruntime

HTTP adapter runtime helpers. Provides:
- `Response` (`Code int`, `Data any`, `Error *Error`) — unified response envelope
- `Error` (`Msg string`, `Fields map[string]any`)
- `CodedError` interface — application errors implement this to carry a custom code
- `OKResponse(data any) Response` — constructs a success response (code 0)
- `ErrResponse(err error) Response` — constructs an error response (extracts CodedError.Code(), defaults to CodeDefaultErr (5000))
- `DefaultErrorHandler() gin.HandlerFunc` — gin middleware that converts `c.Error()` errors to JSON responses (ValidationError → CodeValidationErr (1001), others → CodeDefaultErr (5000) or CodedError.Code())

Public package, importable by user projects.

---

## Generated File Types

| File                          | Trigger                                | Content                                                                                                                                                                          |
| ----------------------------- | -------------------------------------- | -------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- |
| `*.entity.pb.dao.go`          | `*.entity.proto` files                 | struct definitions, json/gorm tags, MarshalBinary, UnmarshalBinary, UnmarshalBinaryLenient, TableName() (when gorm.table annotation is present) |
| `*.entity.pb.dao.validate.go` | `*.entity.proto` files                 | `Validate() error` — returns nil (entity proto carries no validate annotations) |
| `*.create.pb.dao.go`          | `*.create.proto` files                 | struct definitions, ToEntity() |
| `*.create.pb.dao.validate.go` | `*.create.proto` files                 | `Validate() error` — rules read directly from create proto fields |
| `*.update.pb.dao.go`          | `*.update.proto` files                 | struct definitions, ToMap(), ApplyTo() |
| `*.update.pb.dao.validate.go` | `*.update.proto` files                 | `Validate() error` — rules read directly from update proto fields |
| `*.pb.dao.go`                 | Regular proto files                    | struct definitions, json/gorm tags, MarshalBinary, UnmarshalBinary, UnmarshalBinaryLenient, ToMap (update derived messages), TableName() (when gorm.table annotation is present) |
| `*.pb.dao.validate.go`        | Regular proto files                    | `Validate() error` methods covering all buf/validate constraint types                                                                                                            |
| `*.pb.rpc.go`                 | Proto files with `service` definitions | Go interface, method signature: `Method(ctx context.Context, req *XxxRequest) (*XxxResponse, error)`                                                                             |
| `*.pb.http.go`                | Proto files with `service` definitions | gin handler factory functions `XxxHandler(svc XxxService, interceptors ...handlerx.Interceptor[*Req, *Resp]) gin.HandlerFunc`; delegates to `httpruntime.NewHandler` (bind → validate → interceptor chain → svc call, with built-in panic recovery) |
| `*.pb.ts`                     | `gcode gen-ts` subcommand              | TypeScript interfaces, enums, enum name mapping, validation metadata (entity files: no validation metadata), cross-file ES module imports                                        |

---

## Directory Structure

```
github.com/pinealctx/gcode/
├── cmd/gcode/              CLI entry point
├── internal/
│   ├── app/                Pipeline orchestration (Run / RunGenProto / RunGenTS)
│   ├── config/             CLI argument parsing and validation
│   ├── model/              Intermediate semantic model
│   ├── version/            Build-time metadata (version, commit, build time)
│   ├── parser/             proto → model
│   ├── naming/             protobuf-to-Go naming rules
│   ├── transform/          model → Go intermediate representation
│   ├── render/             Go IR → Go source code
│   ├── tsrender/           Go IR → TypeScript source code
│   └── source/             Directory scanning and file discovery
├── options/                gcode_options.proto (embed source)
├── runtime/                Wire format encoding primitives (public package)
├── validateruntime/        Validation runtime helpers (public package)
├── httpruntime/            HTTP adapter runtime helpers (public package)
└── testdata/compat/        End-to-end compatibility test suite
    ├── proto/              Proto source files
    ├── dao/                Generated Go files (snapshots)
    ├── pbgo/               protoc-gen-go output (wire compatibility baseline)
    ├── ts/                 Generated TS files (snapshots, ESM)
    ├── ts-test/            TS runtime verification (tsc + tsx, invoked by Go tests)
    └── gen/main.go         Entry point to regenerate all snapshots
```

---

## Design Goals

1. **No protoc dependency** — Uses `protocompile` to parse proto schema, generates plain Go structs, no `google.golang.org/protobuf` runtime
2. **Wire format compatible** — Generated `MarshalBinary`/`UnmarshalBinary` is fully compatible with the official protobuf binary format, regression-tested against `testdata/compat/pbgo/`
3. **JSON tags built-in** — Generates `json:"field_name"` by default; supports `omitempty`/`ignore` via `(gcode.field).json` annotations
4. **gorm tags optional** — Controlled by `(gcode.message).gorm` annotation; `(gcode.field).gorm.column` overrides column name
5. **Validation via annotations** — Reuses buf/validate annotation semantics to generate `Validate() error` methods
6. **Validate rules explicit in proto** — validate annotations are defined in `.meta.proto` and copied by `gen-proto` into `*.create.proto` / `*.update.proto`. The render layer reads rules directly from those proto fields — no cross-file lookup required
7. **RPC interface transport-agnostic** — Generates Go interfaces only; no routing, serialization, or client stubs; user controls the transport layer entirely
8. **HTTP adapter decoupled from business logic** — Handlers propagate errors via `c.Error(err)+return`; `DefaultErrorHandler` middleware handles response writing; users can replace it with a custom implementation
