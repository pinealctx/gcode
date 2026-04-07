package parser

import "github.com/pinealctx/x/errorx"

// parserTag is a phantom type used as the domain discriminator for ParseError.
// It is intentionally unexported; only ParseError (the type alias) is public.
type parserTag struct{}

// ParseError is the domain error type for all parser constraint violations.
// Use errorx.NewSentinelf[parserTag] to create instances with runtime context.
//
//nolint:revive // ParseError intentionally includes the package name for clarity at call sites.
type ParseError = errorx.Sentinel[parserTag]
