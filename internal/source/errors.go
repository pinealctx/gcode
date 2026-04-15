package source

import "github.com/pinealctx/x/errorx"

// sourceTag is the domain discriminator for source-level errors.
type sourceTag struct{}

// SourceError is the sentinel error type for source-package errors.
// Use errors.As(err, new(source.SourceError)) to match any source-package error.
//
//nolint:revive // SourceError intentionally includes the package name for clarity at call sites.
type SourceError = errorx.Sentinel[sourceTag]
