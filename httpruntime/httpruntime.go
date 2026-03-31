// Package httpruntime provides runtime helpers for generated HTTP handler functions.
// It defines the response envelope, error type, and CodedError interface used by
// generated .pb.http.go files.
package httpruntime

import (
	"errors"
	"net/http"

	"github.com/gin-gonic/gin"

	"github.com/pinealctx/gcode/validateruntime"
)

// Response is the JSON envelope returned by all generated HTTP handlers.
// Code 0 indicates success; non-zero indicates an error.
type Response struct {
	Code  int    `json:"code"`
	Data  any    `json:"data,omitempty"`
	Error *Error `json:"error,omitempty"`
}

// Error carries error details inside a Response.
// Fields is an optional map for structured error metadata (e.g. per-field
// validation failures).
type Error struct {
	Msg    string         `json:"msg"`
	Fields map[string]any `json:"fields,omitempty"`
}

// CodedError may be implemented by application errors to supply a custom
// business error code (not an HTTP status code). If an error passed to
// ErrResponse implements this interface, its Code() value is used; otherwise
// the default code 500 applies.
type CodedError interface {
	Code() int
}

// OKResponse constructs a success Response with code 0 and the given data.
func OKResponse(data any) Response {
	return Response{Code: 0, Data: data}
}

// ErrResponse constructs an error Response from err.
// If err implements CodedError, its Code() is used as the response code.
// Otherwise the response code defaults to 500.
// If err is nil, ErrResponse returns a generic code-500 error response.
func ErrResponse(err error) Response {
	if err == nil {
		return Response{Code: 500, Error: &Error{Msg: "internal error"}}
	}
	code := 500
	if ce, ok := err.(CodedError); ok {
		code = ce.Code()
	}
	return Response{Code: code, Error: &Error{Msg: err.Error()}}
}

// DefaultErrorHandler returns a gin middleware that writes a JSON error response
// for any errors accumulated via c.Error() during handler execution.
// ValidationError maps to code 400; all other errors map to code 500 (or the
// code returned by CodedError.Code() if the error implements that interface).
// Only the last error (c.Errors.Last()) is used when multiple errors are present.
//
// WARNING: Generated handlers use c.Error(err)+return and do not write their own
// error responses. If this middleware is not registered, error paths will return
// HTTP 200 with an empty body. Always register DefaultErrorHandler (or a custom
// equivalent) on any router that serves generated handlers.
func DefaultErrorHandler() gin.HandlerFunc {
	return func(c *gin.Context) {
		c.Next()
		if len(c.Errors) == 0 {
			return
		}
		err := c.Errors.Last().Err
		var ve *validateruntime.ValidationError
		if errors.As(err, &ve) {
			c.JSON(http.StatusOK, Response{Code: 400, Error: &Error{Msg: ve.Error()}})
		} else {
			c.JSON(http.StatusOK, ErrResponse(err))
		}
	}
}
