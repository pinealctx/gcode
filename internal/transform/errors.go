package transform

import "github.com/pinealctx/x/errorx"

// transformTag is a phantom type used as the domain discriminator for TransformError.
// It is intentionally unexported; only TransformError (the type alias) is public.
type transformTag struct{}

// TransformError is the domain error type for transform-layer validation errors.
// Use errorx.NewSentinelf[transformTag] to create instances with runtime context.
//
//nolint:revive // TransformError intentionally includes the package name for clarity at call sites.
type TransformError = errorx.Sentinel[transformTag]
