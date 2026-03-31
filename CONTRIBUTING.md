# Contributing to gcode

Thanks for your interest in contributing! This guide covers the basics.

## Development Setup

**Prerequisites**:
- Go 1.25+
- git

**Clone and build**:

```bash
git clone https://github.com/pinealctx/gcode.git
cd gcode
go build ./...
```

**Run tests**:

```bash
go test ./...
```

**Run linters** (requires [golangci-lint](https://golangci-lint.run/)):

```bash
golangci-lint run ./...
```

## Code Style

Follow the [Uber Go Style Guide](https://github.com/uber-go/guide/blob/master/style.md). Key points:

- Use `any` instead of `interface{}`
- Exported functions and types must have doc comments
- Error messages should be lowercase, no trailing punctuation
- Group imports: stdlib / third-party / local

Run `go fmt ./...` before committing.

## Making Changes

1. **Discuss first** — Open an issue describing the problem and proposed solution before writing code.
2. **Small PRs** — One logical change per PR. Keep it reviewable.
3. **Tests required** — Every bug fix or feature must include test cases.
4. **Generated code** — Never manually edit generated files in `testdata/compat/dao/`. If the generator has a bug, fix the generator (under `internal/`), then regenerate via `cd testdata/compat/gen && go run main.go`.

## Testing

```bash
# Unit tests
go test ./...

# End-to-end compatibility tests
go test ./testdata/compat/...

# Regenerate test snapshots (after generator changes)
cd testdata/compat/gen && go run main.go
```

## Project Structure

| Directory | Purpose |
|-----------|---------|
| `cmd/gcode/` | CLI entry point |
| `internal/` | Generator implementation (not for external import) |
| `options/` | Embedded proto definitions (public) |
| `runtime/` | Wire format encoding primitives (public) |
| `validateruntime/` | Validation runtime helpers (public) |
| `httpruntime/` | HTTP runtime helpers (public) |
| `testdata/compat/` | End-to-end compatibility test suite |
| `docs/` | User documentation |

## Reporting Issues

- Include a minimal `.proto` file that reproduces the problem.
- Include the gcode version (`gcode -h` output or commit hash).
- Describe expected vs actual behavior.

## Code of Conduct

This project follows the [Contributor Covenant Code of Conduct](CODE_OF_CONDUCT.md). By participating, you agree to uphold it.
