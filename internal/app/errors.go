package app

import "github.com/pinealctx/x/errorx"

// appTag is a phantom type used as the domain discriminator for AppError.
// It is intentionally unexported; only AppError (the type alias) is public.
type appTag struct{}

// AppError is the domain error type for app-level coordination errors.
// Use errorx.NewSentinelf[appTag] to create instances with runtime context.
//
//nolint:revive // AppError intentionally includes the package name for clarity at call sites.
type AppError = errorx.Sentinel[appTag]
