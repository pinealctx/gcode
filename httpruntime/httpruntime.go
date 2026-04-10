// Package httpruntime provides runtime helpers for generated HTTP handler functions.
// It defines the response envelope, error type, and CodedError interface used by
// generated .pb.http.go files.
package httpruntime

import (
	"context"
	"errors"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/pinealctx/x/errorx"
	"github.com/pinealctx/x/handlerx"

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
	error
	Code() int
}

// BizCode is the business error code type for application-defined errors.
// Use errorx.Error[BizCode] to define typed business errors that integrate
// with ErrResponse without implementing CodedError manually:
//
//	var ErrUnprocessable = errorx.New(httpruntime.BizCode(422), "unprocessable entity")
//
// BizCode values are application-defined and have no inherent relation to HTTP
// status codes. The numeric value is carried as-is in the response Code field.
type BizCode int

const (
	// CodeOK is the response code for successful requests.
	CodeOK = 0
	// CodeDefaultErr is the response code used when no specific business code is available.
	CodeDefaultErr = 500
	// CodeValidationErr is the response code for validation failures (ValidationError).
	CodeValidationErr = 400
	// CodeBadRequest is the response code for malformed request bodies (JSON parse errors).
	// Distinct from CodeValidationErr (field-level constraint failures): CodeBadRequest means
	// the request body could not be decoded at all, while CodeValidationErr means the decoded
	// value failed business validation rules.
	CodeBadRequest = 400
)

// errBadRequest is returned when ShouldBindJSON fails to decode the request body.
// It uses BizCode(CodeBadRequest) so ErrResponse maps it to code 400 with a safe,
// client-visible message that does not expose internal Go type or field details.
var errBadRequest = errorx.New(BizCode(CodeBadRequest), "malformed request body")

// OKResponse constructs a success Response with code CodeOK (0) and the given data.
func OKResponse(data any) Response {
	return Response{Code: CodeOK, Data: data}
}

// ErrResponse constructs an error Response from err.
// Code resolution order:
//  1. If err (or any error in its chain) is *errorx.Error[BizCode], its Code field is used.
//  2. If err implements CodedError, its Code() value is used.
//  3. Otherwise the response code defaults to CodeDefaultErr (500) and the message
//     is "internal error" — the original error is NOT exposed to the client.
//
// Error visibility contract:
//   - Business errors (BizCode / CodedError): message is safe to expose to clients.
//     Wrap internal errors with a business error to control the client-visible message:
//     errorx.Wrap(dbErr, BizCode(503), "service unavailable")
//   - System errors (plain errors.New / fmt.Errorf): message is hidden from clients.
//     Log the full error chain before or after calling ErrResponse to preserve
//     internal context for debugging.
//
// If err is nil, ErrResponse returns a generic CodeDefaultErr response.
func ErrResponse(err error) Response {
	if err == nil {
		return Response{Code: CodeDefaultErr, Error: &Error{Msg: "internal error"}}
	}
	code := CodeDefaultErr
	msg := "internal error"
	if he, ok := errors.AsType[*errorx.Error[BizCode]](err); ok {
		code = int(he.Code)
		msg = err.Error()
	} else if ce, ok := errors.AsType[CodedError](err); ok {
		code = ce.Code()
		msg = err.Error()
	}
	return Response{Code: code, Error: &Error{Msg: msg}}
}

// NewHandler creates a gin.HandlerFunc that binds JSON, validates, and calls
// the service method through a handlerx interceptor chain.
// WithRecovery is always applied as the outermost interceptor.
// Additional interceptors are applied inside recovery, before the service method.
func NewHandler[Req any, Resp any](
	method func(ctx context.Context, req *Req) (*Resp, error),
	interceptors ...handlerx.Interceptor[*Req, *Resp],
) gin.HandlerFunc {
	all := make([]handlerx.Interceptor[*Req, *Resp], 0, 1+len(interceptors))
	all = append(all, handlerx.WithRecovery[*Req, *Resp]())
	all = append(all, interceptors...)
	h := handlerx.Chain(handlerx.Handler[*Req, *Resp](method), all...)
	return func(c *gin.Context) {
		var req Req
		if err := c.ShouldBindJSON(&req); err != nil {
			_ = c.Error(errBadRequest)
			return
		}
		if v, ok := any(&req).(interface{ Validate() error }); ok {
			if err := v.Validate(); err != nil {
				_ = c.Error(err)
				return
			}
		}
		resp, err := h(c.Request.Context(), &req)
		if err != nil {
			_ = c.Error(err)
			return
		}
		c.JSON(http.StatusOK, OKResponse(resp))
	}
}

// DefaultErrorHandler returns a gin middleware that writes a JSON error response
// for any errors accumulated via c.Error() during handler execution.
// ValidationError maps to code 400; all other errors use ErrResponse (see its
// doc for the business vs system error visibility contract).
// Only the last error (c.Errors.Last()) is used when multiple errors are present.
//
// Logging contract: this middleware does NOT log errors. System errors (plain
// errors.New / fmt.Errorf) are hidden from the client response; to preserve the
// full error chain for debugging, log the error in the handler before calling
// c.Error(), or register a separate logging middleware that reads c.Errors:
//
//	// Option A: log in the handler
//	if err := svc.Create(req); err != nil {
//	    logger.Error("create failed", "error", err)
//	    _ = c.Error(err)
//	    return
//	}
//
//	// Option B: separate logging middleware (registered before DefaultErrorHandler)
//	func LogErrors(logger *slog.Logger) gin.HandlerFunc {
//	    return func(c *gin.Context) {
//	        c.Next()
//	        for _, e := range c.Errors {
//	            logger.Error("request error", "error", e.Err)
//	        }
//	    }
//	}
//
// Note: when the error is (or wraps) a *validateruntime.ValidationError, the
// response Msg is taken from the ValidationError itself, not from any outer wrapper.
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
		if ve, ok := errors.AsType[*validateruntime.ValidationError](err); ok {
			c.JSON(http.StatusOK, Response{Code: CodeValidationErr, Error: &Error{Msg: ve.Error()}})
		} else {
			c.JSON(http.StatusOK, ErrResponse(err))
		}
	}
}
