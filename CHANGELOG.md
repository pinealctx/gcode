# Changelog

All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.1.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [Unreleased]

### Changed

- `update_source` option (field 50005, scalar string) replaced by `update_source_opts` (field 50007, `UpdateSourceOptions` message) carrying both `source` and `condition_fields`. This fixes `ApplyTo` / `ToMap` skipping message-type fields that were incorrectly inferred as condition fields. The old field 50005 is retired and no longer recognized. Since both options are internal (written only by `gcode gen-proto`), user impact is limited to re-running `gen-proto` to regenerate intermediate proto files.

### Fixed

- `ApplyTo` and `ToMap` now correctly handle message-type fields (e.g. `Dimensions`) in update-derived messages.
- `ToMap` now correctly handles optional bytes fields (`[]byte` with `HasPresence`) with nil-guard.

## [0.1.0] — 2026-03-31

### Added

- Proto-to-Go code generation pipeline (proto3 only)
- Pure Go proto parser via `protocompile` — no `protoc` dependency
- Generated structs with `json` tags and optional `gorm` tags
- `MarshalBinary` / `UnmarshalBinary` — protobuf wire format compatible
- `UnmarshalBinaryLenient` — lenient mode (duplicate fields use last value)
- `Validate() error` methods via `buf/validate` annotation syntax
- `ToMap()` for update derived messages (GORM partial update)
- Derived message generation (`update_message` / `create_message` annotations)
- `gen-proto` sub-command for intermediate proto generation
- Service interface generation (`*.pb.rpc.go`)
- gin HTTP handler factory generation (`*.pb.http.go`)
- `c.Error` + `DefaultErrorHandler` middleware pattern
- Comment passthrough (struct, field, enum, service, handler)
- Embedded `gcode/options.proto` and `buf/validate/validate.proto`
- Public runtime packages: `runtime`, `validateruntime`, `httpruntime`
- doc.go for all 4 public packages (`options`, `runtime`, `validateruntime`, `httpruntime`)
- CONTRIBUTING.md and MIT LICENSE
- Chinese/English documentation split (README.md / README.zh-CN.md, docs/*.md / docs/*.zh.md)
- Complete documentation suite: Getting Started, Architecture, Annotations Reference, Design Decisions (Chinese and English)
- Enum and nested message examples in Getting Started guide
- CLI reference (`gcode -h` output) in Getting Started guide
- curl examples for HTTP service in Getting Started guide
- JSON tag naming rule (snake_case → camelCase) explanation in Getting Started guide

[Unreleased]: https://github.com/pinealctx/gcode/compare/v0.1.0...HEAD
[0.1.0]: https://github.com/pinealctx/gcode/releases/tag/v0.1.0
