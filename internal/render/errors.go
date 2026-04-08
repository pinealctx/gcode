package render

import "github.com/pinealctx/x/errorx"

// renderTag is the phantom type used to scope render-package sentinel errors.
type renderTag struct{}

// Error is the sentinel error type for render-package errors.
// Use errors.As(err, new(render.Error)) to match any render-package error.
type Error = errorx.Sentinel[renderTag]

// ErrTooManyFields is returned when a message has more than 128 non-repeated
// fields, which exceeds the bitmask capacity used for duplicate detection.
var ErrTooManyFields = errorx.NewSentinel[renderTag]("render: message has more than 128 non-repeated fields")
