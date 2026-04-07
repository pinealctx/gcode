package config

import "github.com/pinealctx/x/errorx"

// configTag is a phantom type used as the domain discriminator for ConfigError.
// It is intentionally unexported; only ConfigError (the type alias) is public.
type configTag struct{}

// ConfigError is the domain error type for config validation errors.
//
//nolint:revive // ConfigError intentionally includes the package name for clarity at call sites.
type ConfigError = errorx.Sentinel[configTag]

var (
	// gen-dao flags
	ErrMissingInputDir  = ConfigError("validate cli config: missing -in")
	ErrMissingOutputDir = ConfigError("validate cli config: missing -out")

	// gen-proto flags
	ErrMissingProtoInputDir = ConfigError("validate gen-proto config: missing -in")

	// gen-ts flags
	ErrMissingTSInputDir  = ConfigError("validate gen-ts config: missing -in")
	ErrMissingTSOutputDir = ConfigError("validate gen-ts config: missing -out")
	ErrNoProtoFiles       = ConfigError("no .proto files found")
)
