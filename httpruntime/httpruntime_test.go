package httpruntime_test

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/pinealctx/x/errorx"

	"github.com/pinealctx/gcode/httpruntime"
	"github.com/pinealctx/gcode/validateruntime"
)

// codedErr is a test error that implements CodedError.
type codedErr struct {
	code int
	msg  string
}

func (e *codedErr) Error() string { return e.msg }
func (e *codedErr) Code() int     { return e.code }

// Compile-time assertion: codedErr implements httpruntime.CodedError.
var _ httpruntime.CodedError = (*codedErr)(nil)

func TestOKResponse(t *testing.T) {
	t.Parallel()

	type payload struct{ ID string }
	resp := httpruntime.OKResponse(payload{ID: "abc"})

	if resp.Code != httpruntime.CodeOK {
		t.Errorf("Code = %d, want CodeOK (0)", resp.Code)
	}
	if resp.Error != nil {
		t.Errorf("Error = %v, want nil", resp.Error)
	}
	p, ok := resp.Data.(payload)
	if !ok || p.ID != "abc" {
		t.Errorf("Data = %v, want {ID:abc}", resp.Data)
	}
}

func TestOKResponse_NilData(t *testing.T) {
	t.Parallel()

	resp := httpruntime.OKResponse(nil)
	if resp.Code != httpruntime.CodeOK {
		t.Errorf("Code = %d, want CodeOK (0)", resp.Code)
	}
	if resp.Data != nil {
		t.Errorf("Data = %v, want nil", resp.Data)
	}
}

func TestErrResponse_DefaultCode(t *testing.T) {
	t.Parallel()

	resp := httpruntime.ErrResponse(errors.New("something went wrong"))

	if resp.Code != httpruntime.CodeDefaultErr {
		t.Errorf("Code = %d, want CodeDefaultErr (500)", resp.Code)
	}
	if resp.Data != nil {
		t.Errorf("Data = %v, want nil", resp.Data)
	}
	if resp.Error == nil {
		t.Fatal("Error is nil, want non-nil")
	}
	if resp.Error.Msg != "internal error" {
		t.Errorf("Error.Msg = %q, want %q", resp.Error.Msg, "internal error")
	}
}

func TestErrResponse_CodedError(t *testing.T) {
	t.Parallel()

	err := &codedErr{code: 404, msg: "not found"}
	resp := httpruntime.ErrResponse(err)

	if resp.Code != 404 {
		t.Errorf("Code = %d, want 404", resp.Code)
	}
	if resp.Error == nil {
		t.Fatal("Error is nil, want non-nil")
	}
	if resp.Error.Msg != "not found" {
		t.Errorf("Error.Msg = %q, want %q", resp.Error.Msg, "not found")
	}
}

func TestErrResponse_CodedError_Zero(t *testing.T) {
	t.Parallel()

	// code 0 from CodedError should be respected (not overridden to 500)
	err := &codedErr{code: 0, msg: "unusual"}
	resp := httpruntime.ErrResponse(err)

	if resp.Code != httpruntime.CodeOK {
		t.Errorf("Code = %d, want CodeOK (0)", resp.Code)
	}
}

func TestErrResponse_CodedError_Negative(t *testing.T) {
	t.Parallel()

	// negative code from CodedError should be passed through as-is
	err := &codedErr{code: -1, msg: "negative code"}
	resp := httpruntime.ErrResponse(err)

	if resp.Code != -1 {
		t.Errorf("Code = %d, want -1", resp.Code)
	}
}

func TestErrResponse_NilError(t *testing.T) {
	t.Parallel()

	// nil error should return a generic 500 response, not panic
	resp := httpruntime.ErrResponse(nil)

	if resp.Code != httpruntime.CodeDefaultErr {
		t.Errorf("Code = %d, want CodeDefaultErr (500)", resp.Code)
	}
	if resp.Error == nil {
		t.Fatal("Error is nil, want non-nil")
	}
}

func TestErrResponse_Fields_NilByDefault(t *testing.T) {
	t.Parallel()

	resp := httpruntime.ErrResponse(errors.New("err"))
	if resp.Error.Fields != nil {
		t.Errorf("Fields = %v, want nil", resp.Error.Fields)
	}
}

// --- DefaultErrorHandler -----------------------------------------------------

func newHandlerEngine(handler gin.HandlerFunc) *gin.Engine {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.Use(httpruntime.DefaultErrorHandler())
	r.POST("/", handler)
	return r
}

func decodeResponse(t *testing.T, body *httptest.ResponseRecorder) httpruntime.Response {
	t.Helper()
	var resp httpruntime.Response
	if err := json.NewDecoder(body.Body).Decode(&resp); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	return resp
}

func TestDefaultErrorHandler_NoErrors(t *testing.T) {
	t.Parallel()

	r := newHandlerEngine(func(c *gin.Context) {
		c.JSON(http.StatusOK, httpruntime.OKResponse("ok"))
	})

	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodPost, "/", nil)
	r.ServeHTTP(w, req)

	resp := decodeResponse(t, w)
	if resp.Code != httpruntime.CodeOK {
		t.Errorf("Code = %d, want CodeOK (0)", resp.Code)
	}
}

func TestDefaultErrorHandler_PlainError(t *testing.T) {
	t.Parallel()

	r := newHandlerEngine(func(c *gin.Context) {
		_ = c.Error(errors.New("something failed"))
	})

	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodPost, "/", nil)
	r.ServeHTTP(w, req)

	resp := decodeResponse(t, w)
	if resp.Code != httpruntime.CodeDefaultErr {
		t.Errorf("Code = %d, want CodeDefaultErr (500)", resp.Code)
	}
	if resp.Error == nil || resp.Error.Msg != "internal error" {
		t.Errorf("Error = %+v, want msg 'internal error'", resp.Error)
	}
}

func TestDefaultErrorHandler_ValidationError(t *testing.T) {
	t.Parallel()

	ve := &validateruntime.ValidationError{Field: "name", Rule: "required", Message: "name is required"}
	r := newHandlerEngine(func(c *gin.Context) {
		_ = c.Error(ve)
	})

	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodPost, "/", nil)
	r.ServeHTTP(w, req)

	resp := decodeResponse(t, w)
	if resp.Code != httpruntime.CodeValidationErr {
		t.Errorf("Code = %d, want CodeValidationErr (400)", resp.Code)
	}
	if resp.Error == nil {
		t.Fatal("Error is nil, want non-nil")
	}
	if resp.Error.Msg != ve.Error() {
		t.Errorf("Error.Msg = %q, want %q", resp.Error.Msg, ve.Error())
	}
}

func TestDefaultErrorHandler_WrappedValidationError(t *testing.T) {
	t.Parallel()

	ve := &validateruntime.ValidationError{Field: "age", Rule: "gte", Message: "age must be >= 0"}
	wrapped := fmt.Errorf("validation: %w", ve)
	r := newHandlerEngine(func(c *gin.Context) {
		_ = c.Error(wrapped)
	})

	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodPost, "/", nil)
	r.ServeHTTP(w, req)

	resp := decodeResponse(t, w)
	if resp.Code != httpruntime.CodeValidationErr {
		t.Errorf("Code = %d, want CodeValidationErr (400) for wrapped ValidationError", resp.Code)
	}
	// Msg must be ve.Error(), not the outer wrapper message.
	if resp.Error == nil || resp.Error.Msg != ve.Error() {
		t.Errorf("Error.Msg = %q, want %q (ve.Error(), not outer wrapper)", resp.Error, ve.Error())
	}
}

func TestDefaultErrorHandler_MultipleErrors_UsesLast(t *testing.T) {
	t.Parallel()

	r := newHandlerEngine(func(c *gin.Context) {
		_ = c.Error(errors.New("first error"))
		_ = c.Error(errors.New("last error"))
	})

	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodPost, "/", nil)
	r.ServeHTTP(w, req)

	resp := decodeResponse(t, w)
	if resp.Error == nil || resp.Error.Msg != "internal error" {
		t.Errorf("Error.Msg = %q, want 'internal error'", resp.Error)
	}
}

// TestDefaultErrorHandler_MultipleErrors_ValidationErrorFirst verifies that when
// the first error is a ValidationError but the last is a plain error, the response
// code is 500 (not 400) — only the last error determines the response.
func TestDefaultErrorHandler_MultipleErrors_ValidationErrorFirst(t *testing.T) {
	t.Parallel()

	ve := &validateruntime.ValidationError{Field: "name", Rule: "required", Message: "name is required"}
	r := newHandlerEngine(func(c *gin.Context) {
		_ = c.Error(ve)
		_ = c.Error(errors.New("plain error after validation"))
	})

	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodPost, "/", nil)
	r.ServeHTTP(w, req)

	resp := decodeResponse(t, w)
	// Last error is a plain error, so code must be 500, not 400.
	if resp.Code != httpruntime.CodeDefaultErr {
		t.Errorf("Code = %d, want 500 (last error is plain, not ValidationError)", resp.Code)
	}
}

func TestDefaultErrorHandler_CodedError(t *testing.T) {
	t.Parallel()

	r := newHandlerEngine(func(c *gin.Context) {
		_ = c.Error(&codedErr{code: 403, msg: "forbidden"})
	})

	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodPost, "/", nil)
	r.ServeHTTP(w, req)

	resp := decodeResponse(t, w)
	if resp.Code != 403 {
		t.Errorf("Code = %d, want 403", resp.Code)
	}
	if resp.Error == nil || resp.Error.Msg != "forbidden" {
		t.Errorf("Error = %+v, want msg 'forbidden'", resp.Error)
	}
}

func TestErrResponse_BizCode(t *testing.T) {
	t.Parallel()

	err := errorx.New(httpruntime.BizCode(422), "unprocessable entity")
	resp := httpruntime.ErrResponse(err)
	if resp.Code != 422 {
		t.Errorf("Code = %d, want 422", resp.Code)
	}
	if resp.Error == nil || resp.Error.Msg != "unprocessable entity" {
		t.Errorf("Error = %+v, want msg 'unprocessable entity'", resp.Error)
	}
}

func TestErrResponse_BizCode_PriorityOverCodedError(t *testing.T) {
	t.Parallel()

	// errorx.Error[BizCode] should take priority over CodedError.
	// Wrap a CodedError (code=403) inside an errorx.Error[BizCode] (code=422).
	inner := &codedErr{code: 403, msg: "forbidden"}
	err := errorx.Wrap(inner, httpruntime.BizCode(422), "unprocessable entity")
	resp := httpruntime.ErrResponse(err)
	if resp.Code != 422 {
		t.Errorf("Code = %d, want 422 (errorx.Error[BizCode] should take priority)", resp.Code)
	}
}

func TestErrResponse_BizCode_WrappedInFmtErrorf(t *testing.T) {
	t.Parallel()

	// errors.AsType should penetrate fmt.Errorf %w wrapping.
	inner := errorx.New(httpruntime.BizCode(404), "not found")
	err := fmt.Errorf("lookup failed: %w", inner)
	resp := httpruntime.ErrResponse(err)
	if resp.Code != 404 {
		t.Errorf("Code = %d, want 404 (should penetrate %%w wrapping)", resp.Code)
	}
}

func TestDefaultErrorHandler_BizCodeError(t *testing.T) {
	t.Parallel()

	r := newHandlerEngine(func(c *gin.Context) {
		_ = c.Error(errorx.New(httpruntime.BizCode(422), "unprocessable entity"))
	})

	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodPost, "/", nil)
	r.ServeHTTP(w, req)

	resp := decodeResponse(t, w)
	if resp.Code != 422 {
		t.Errorf("Code = %d, want 422", resp.Code)
	}
	if resp.Error == nil || resp.Error.Msg != "unprocessable entity" {
		t.Errorf("Error = %+v, want msg 'unprocessable entity'", resp.Error)
	}
}

func TestDefaultErrorHandler_BizCodeWrappedInFmtErrorf(t *testing.T) {
	t.Parallel()

	// BizCode error wrapped in fmt.Errorf should still be resolved via errors.AsType.
	inner := errorx.New(httpruntime.BizCode(404), "not found")
	r := newHandlerEngine(func(c *gin.Context) {
		_ = c.Error(fmt.Errorf("lookup failed: %w", inner))
	})

	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodPost, "/", nil)
	r.ServeHTTP(w, req)

	resp := decodeResponse(t, w)
	if resp.Code != 404 {
		t.Errorf("Code = %d, want 404 (BizCode should penetrate %%w wrapping)", resp.Code)
	}
}
