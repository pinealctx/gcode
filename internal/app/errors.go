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

var (
	// ErrOutputFilenameCollision indicates two proto files produce the same output filename.
	ErrOutputFilenameCollision = AppError("output filename collision")
	// ErrNameEmpty indicates an update_message or create_message annotation has an empty name.
	ErrNameEmpty = AppError("name must not be empty")
	// ErrInvalidName indicates an update_message or create_message annotation name is not a valid proto identifier.
	ErrInvalidName = AppError("not a valid proto identifier")
)
