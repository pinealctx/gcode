# gcode

A code generator that produces plain Go structs from `.proto` files. No `protoc` dependency, no protobuf runtime — generated code is ordinary Go structs that work directly with GORM, JSON serialization, and gin HTTP services.

[中文](README.zh-CN.md)

**Why gcode?** Official `protoc-gen-go` generates structs containing runtime fields (`protoimpl.MessageState`, `sizeCache`, `unknownFields`) that are not suitable as DAO layer data structures and forces a dependency on `google.golang.org/protobuf` runtime. gcode produces plain Go structs with zero runtime dependencies — usable directly with GORM, JSON serialization, and gin HTTP binding, while maintaining full wire format compatibility with the official protobuf binary format.

---

## Features

- **No protoc required** — Parses proto files in pure Go via `protocompile`. `go install` and you're done.
- **Wire format compatible** — Generated `MarshalBinary`/`UnmarshalBinary` is fully compatible with the official protobuf binary format.
- **JSON tags built-in** — Generates `json:"camelCase"` tags by default; supports `omitempty`/`ignore` via annotations.
- **GORM support** — Generates gorm struct tags and `TableName()` via `(gcode.message).gorm` annotation.
- **Built-in validation** — Reuses `buf/validate` annotation syntax to generate `Validate() error` methods.
- **Derived message generation** — Declare update/create derived messages via annotations in `.meta.proto` schema files; `gen-proto` generates entity/create/update proto files with validate annotations explicitly copied.
- **gin HTTP adapter** — Generates handler factory functions decoupled from service interfaces.
- **TypeScript generation** — Generates interfaces, enums, and validation metadata from proto files via `gcode gen-ts`.
- **Comment passthrough** — Proto leading comments pass through to all generated code.

---

## Installation

```bash
go install github.com/pinealctx/gcode/cmd/gcode@latest
```

Verify:

```bash
gcode -h
```

---

## Project Setup

Create a new Go module and proto directory:

```bash
mkdir myapp && cd myapp
go mod init myapp
mkdir proto dao
```

---

## Quick Start

**1. Write a proto file**

```proto
// proto/user.proto
syntax = "proto3";
package myapp;

// go_package determines the Go package name for generated code:
//   "import/path;pkg" — the part after ';' is the Go package name.
//   The import path is used by gcode for package resolution.
//   The -out flag controls where files are written to disk.
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

**3. Install runtime dependencies**

Generated code imports packages from the gcode module:

```bash
go get github.com/pinealctx/gcode/runtime
```

> Only `runtime` is needed for basic struct generation. If you use `buf/validate` annotations, add `go get github.com/pinealctx/gcode/validateruntime`. If you use `service` definitions with gin handlers, add `go get github.com/gin-gonic/gin` and `go get github.com/pinealctx/gcode/httpruntime`.
>
> **Note**: Running `go build` before `go get` will produce a missing module error — that is expected. Run `go get` first, then build.

**4. Use the generated struct**

```go
package main

import (
    "fmt"
    "log"
    "myapp/dao"
)

func main() {
    u := &dao.User{Name: "Alice", Age: 30}

    // Serialize to protobuf wire format
    wire, err := u.MarshalBinary()
    if err != nil {
        log.Fatal(err)
    }

    // Deserialize
    var u2 dao.User
    if err := u2.UnmarshalBinary(wire); err != nil {
        log.Fatal(err)
    }
    fmt.Println(u2.Name, u2.Age)
}
```

> **Derived messages**: If you use `gcode.update_message` or `gcode.create_message` annotations, write them in a `.meta.proto` schema file with `option (gcode.schema) = {}`. Then run `gcode gen-proto -in proto/` to generate `*.entity.proto`, `*.create.proto`, and `*.update.proto`, then `gcode -in proto/ -out dao/`. See [Getting Started](docs/getting-started.md#step-2-generate-derived-proto-files) for the full workflow.

**5. Generate TypeScript types (optional)**

If you need frontend type definitions:

```bash
gcode gen-ts -in proto/ -out ts/
```

This generates `.pb.ts` files with TypeScript interfaces, enums, and validation metadata. See [Getting Started — TypeScript Generation](docs/getting-started.md#typescript-generation) for details.

---

## Documentation

| Document                                     | Description                                                             |
| -------------------------------------------- | ----------------------------------------------------------------------- |
| [Getting Started](docs/getting-started.md)   | Full 8-step example (proto to HTTP service), annotation quick reference |
| [Architecture](docs/architecture.md)         | Pipeline overview, layer responsibilities, directory structure          |
| [Annotations Reference](docs/annotations.md) | Detailed documentation and examples for all annotations                 |
| [Design Decisions](docs/design-decisions.md) | Key architectural decisions (ADR style, D1–D15)                         |

---

## Known Limitations

| Limitation                          | Details                                                                       |
| ----------------------------------- | ----------------------------------------------------------------------------- |
| No streaming RPC                    | Exits with error when `stream` keyword is encountered                         |
| No path params                      | HTTP handlers use `c.ShouldBindJSON` uniformly; URL path parameters not supported |
| No `map`, `oneof`, well-known types | Proto files using these features may produce incorrect code                   |
| Flat Go output directory            | Go generation writes all generated Go files into one output package directory. Proto files with the same basename are not supported in one generation run, even when they are in different source subdirectories. |
| Cross-package references untested   | Same-package cross-file works; cross-package not fully tested                 |
| proto3 only                         | proto2 syntax is not supported                                                |

---

## Contributing

See [CONTRIBUTING.md](CONTRIBUTING.md) for development setup, code style, and PR guidelines.

## Changelog

See [CHANGELOG.md](CHANGELOG.md) for release history.

---

## License

[MIT](LICENSE)
