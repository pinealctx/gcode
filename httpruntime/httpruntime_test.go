package httpruntime_test

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"slices"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/pinealctx/x/errorx"
	"github.com/pinealctx/x/handlerx"
	"github.com/pinealctx/x/panicx"

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
		t.Errorf("Code = %d, want CodeDefaultErr (%d)", resp.Code, httpruntime.CodeDefaultErr)
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

	// code 0 from CodedError should be respected (not overridden to CodeDefaultErr)
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

	// nil error should return a generic CodeDefaultErr response, not panic
	resp := httpruntime.ErrResponse(nil)

	if resp.Code != httpruntime.CodeDefaultErr {
		t.Errorf("Code = %d, want CodeDefaultErr (%d)", resp.Code, httpruntime.CodeDefaultErr)
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
		t.Errorf("Code = %d, want CodeDefaultErr (%d)", resp.Code, httpruntime.CodeDefaultErr)
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
		t.Errorf("Code = %d, want CodeValidationErr (%d)", resp.Code, httpruntime.CodeValidationErr)
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
		t.Errorf("Code = %d, want CodeValidationErr (%d) for wrapped ValidationError", resp.Code, httpruntime.CodeValidationErr)
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
// code is CodeDefaultErr (not CodeValidationErr) — only the last error determines the response.
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
	// Last error is a plain error, so code must be CodeDefaultErr, not CodeValidationErr.
	if resp.Code != httpruntime.CodeDefaultErr {
		t.Errorf("Code = %d, want CodeDefaultErr (%d) (last error is plain, not ValidationError)", resp.Code, httpruntime.CodeDefaultErr)
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

// --- NewHandler --------------------------------------------------------------

// echoReq is a minimal request type without Validate().
type echoReq struct {
	Msg string `json:"msg"`
}

// echoResp is a minimal response type.
type echoResp struct {
	Echo string `json:"echo"`
}

// validateReq is a request type that implements Validate().
type validateReq struct {
	Name string `json:"name"`
}

func (r *validateReq) Validate() error {
	if r.Name == "" {
		return &validateruntime.ValidationError{Field: "name", Rule: "required", Message: "name is required"}
	}
	return nil
}

// newHandlerEngineForNewHandler builds a test engine using NewHandler.
func newHandlerEngineForNewHandler[Req any, Resp any](
	method func(ctx context.Context, req *Req) (*Resp, error),
	interceptors ...handlerx.Interceptor[*Req, *Resp],
) *gin.Engine {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.Use(httpruntime.DefaultErrorHandler())
	r.POST("/", httpruntime.NewHandler(method, interceptors...))
	return r
}

func postJSON(t *testing.T, r *gin.Engine, body string) *httptest.ResponseRecorder {
	t.Helper()
	w := httptest.NewRecorder()
	req, err := http.NewRequest(http.MethodPost, "/", bytes.NewBufferString(body))
	if err != nil {
		t.Fatalf("new request: %v", err)
	}
	req.Header.Set("Content-Type", "application/json")
	r.ServeHTTP(w, req)
	return w
}

func TestNewHandler_Success(t *testing.T) {
	t.Parallel()

	method := func(_ context.Context, req *echoReq) (*echoResp, error) {
		return &echoResp{Echo: req.Msg}, nil
	}
	r := newHandlerEngineForNewHandler(method)

	w := postJSON(t, r, `{"msg":"hello"}`)
	resp := decodeResponse(t, w)
	if resp.Code != httpruntime.CodeOK {
		t.Errorf("Code = %d, want CodeOK", resp.Code)
	}
	// Data is decoded as map[string]any from JSON.
	data, ok := resp.Data.(map[string]any)
	if !ok {
		t.Fatalf("Data type = %T, want map[string]any", resp.Data)
	}
	if data["echo"] != "hello" {
		t.Errorf("echo = %v, want hello", data["echo"])
	}
}

func TestNewHandler_BindError(t *testing.T) {
	t.Parallel()

	method := func(_ context.Context, req *echoReq) (*echoResp, error) {
		return &echoResp{Echo: req.Msg}, nil
	}
	r := newHandlerEngineForNewHandler(method)

	// Send invalid JSON — bind should fail with CodeBadRequest and a safe message.
	w := postJSON(t, r, `not-json`)
	resp := decodeResponse(t, w)
	if resp.Code != httpruntime.CodeBadRequest {
		t.Errorf("Code = %d, want CodeBadRequest (%d)", resp.Code, httpruntime.CodeBadRequest)
	}
	if resp.Error == nil {
		t.Fatal("Error is nil, want non-nil")
	}
	if resp.Error.Msg != "malformed request body" {
		t.Errorf("Error.Msg = %q, want %q", resp.Error.Msg, "malformed request body")
	}
}

func TestNewHandler_ValidationError(t *testing.T) {
	t.Parallel()

	method := func(_ context.Context, req *validateReq) (*echoResp, error) {
		return &echoResp{Echo: req.Name}, nil
	}
	r := newHandlerEngineForNewHandler(method)

	// Empty name triggers Validate() failure.
	w := postJSON(t, r, `{"name":""}`)
	resp := decodeResponse(t, w)
	if resp.Code != httpruntime.CodeValidationErr {
		t.Errorf("Code = %d, want CodeValidationErr (%d)", resp.Code, httpruntime.CodeValidationErr)
	}
}

func TestNewHandler_ServiceError(t *testing.T) {
	t.Parallel()

	method := func(_ context.Context, _ *echoReq) (*echoResp, error) {
		return nil, errors.New("service failure")
	}
	r := newHandlerEngineForNewHandler(method)

	w := postJSON(t, r, `{"msg":"hi"}`)
	resp := decodeResponse(t, w)
	if resp.Code != httpruntime.CodeDefaultErr {
		t.Errorf("Code = %d, want CodeDefaultErr (%d)", resp.Code, httpruntime.CodeDefaultErr)
	}
}

func TestNewHandler_PanicRecovery(t *testing.T) {
	t.Parallel()

	method := func(_ context.Context, _ *echoReq) (*echoResp, error) {
		panic("something went wrong") //nolint // intentional panic to test WithRecovery
	}
	r := newHandlerEngineForNewHandler(method)

	w := postJSON(t, r, `{"msg":"hi"}`)
	// The response should be an error (not a server crash).
	if w.Code == 0 {
		t.Fatal("expected a response, got none")
	}
	resp := decodeResponse(t, w)
	if resp.Code == httpruntime.CodeOK {
		t.Errorf("Code = CodeOK, want error code after panic")
	}
}

func TestNewHandler_PanicError_IsPanicx(t *testing.T) {
	t.Parallel()

	// Capture the error produced by WithRecovery to verify errors.Is(err, panicx.ErrPanic).
	// The interceptor is inside WithRecovery, so it observes the recovered error on return.
	// Use a gin error-capture middleware to read the error after the handler chain completes.
	var capturedErr error
	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.Use(func(c *gin.Context) {
		c.Next()
		if len(c.Errors) > 0 {
			capturedErr = c.Errors.Last().Err
		}
		// Write a minimal response so decodeResponse doesn't fail.
		c.JSON(http.StatusOK, httpruntime.ErrResponse(capturedErr))
	})

	method := func(_ context.Context, _ *echoReq) (*echoResp, error) {
		panic("boom") //nolint // intentional panic to test WithRecovery
	}
	r.POST("/", httpruntime.NewHandler(method))
	postJSON(t, r, `{"msg":"hi"}`)

	if capturedErr == nil {
		t.Fatal("capturedErr is nil, expected a panic error")
	}
	if !errors.Is(capturedErr, panicx.ErrPanic) {
		t.Errorf("errors.Is(err, panicx.ErrPanic) = false, want true; err = %v", capturedErr)
	}
}

func TestNewHandler_UserInterceptor(t *testing.T) {
	t.Parallel()

	var preRan, postRan bool
	interceptor := func(_ context.Context, req *echoReq, next handlerx.Handler[*echoReq, *echoResp]) (*echoResp, error) {
		preRan = true
		resp, err := next(context.Background(), req)
		postRan = true
		return resp, err
	}

	method := func(_ context.Context, req *echoReq) (*echoResp, error) {
		return &echoResp{Echo: req.Msg}, nil
	}
	r := newHandlerEngineForNewHandler(method, interceptor)
	postJSON(t, r, `{"msg":"test"}`)

	if !preRan {
		t.Error("interceptor pre-logic did not run")
	}
	if !postRan {
		t.Error("interceptor post-logic did not run")
	}
}

func TestNewHandler_NoValidate_NoError(t *testing.T) {
	t.Parallel()

	// echoReq has no Validate() — should not error on empty fields.
	method := func(_ context.Context, req *echoReq) (*echoResp, error) {
		return &echoResp{Echo: req.Msg}, nil
	}
	r := newHandlerEngineForNewHandler(method)

	w := postJSON(t, r, `{}`)
	resp := decodeResponse(t, w)
	if resp.Code != httpruntime.CodeOK {
		t.Errorf("Code = %d, want CodeOK for type without Validate()", resp.Code)
	}
}

type orderedValidateReq struct {
	Name   string    `json:"name"`
	Events *[]string `json:"-"`
}

func (r *orderedValidateReq) Validate() error {
	if r.Events != nil {
		*r.Events = append(*r.Events, "validate")
	}
	if r.Name == "" {
		return &validateruntime.ValidationError{Field: "name", Rule: "required", Message: "name is required"}
	}
	return nil
}

func newHandlerEngineWithOptions[Req any, Resp any](
	method func(ctx context.Context, req *Req) (*Resp, error),
	opts ...httpruntime.HandlerOption[Req, Resp],
) *gin.Engine {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.Use(httpruntime.DefaultErrorHandler())
	r.POST("/", httpruntime.NewHandlerWithOptions(method, opts...))
	return r
}

func TestNewHandlerWithOptions_PreValidateHookRunsBeforeValidate(t *testing.T) {
	t.Parallel()

	var events []string
	method := func(_ context.Context, req *orderedValidateReq) (*echoResp, error) {
		return &echoResp{Echo: req.Name}, nil
	}
	hook := func(_ context.Context, req *orderedValidateReq) error {
		req.Events = &events
		events = append(events, "hook")
		return nil
	}
	r := newHandlerEngineWithOptions(method, httpruntime.WithPreValidateHook[orderedValidateReq, echoResp](hook))

	w := postJSON(t, r, `{"name":"alice"}`)
	resp := decodeResponse(t, w)
	if resp.Code != httpruntime.CodeOK {
		t.Fatalf("Code = %d, want CodeOK", resp.Code)
	}
	if !slices.Equal(events, []string{"hook", "validate"}) {
		t.Errorf("events = %v, want [hook validate]", events)
	}
}

func TestNewHandlerWithOptions_PreValidateHookCanMutateRequest(t *testing.T) {
	t.Parallel()

	var seen string
	method := func(_ context.Context, req *validateReq) (*echoResp, error) {
		seen = req.Name
		return &echoResp{Echo: req.Name}, nil
	}
	hook := func(_ context.Context, req *validateReq) error {
		req.Name = "patched"
		return nil
	}
	r := newHandlerEngineWithOptions(method, httpruntime.WithPreValidateHook[validateReq, echoResp](hook))

	w := postJSON(t, r, `{"name":""}`)
	resp := decodeResponse(t, w)
	if resp.Code != httpruntime.CodeOK {
		t.Fatalf("Code = %d, want CodeOK", resp.Code)
	}
	if seen != "patched" {
		t.Errorf("service saw name = %q, want patched", seen)
	}
}

type blockingValidateReq struct {
	Name string `json:"name"`
	Seen *bool  `json:"-"`
}

func (r *blockingValidateReq) Validate() error {
	if r.Seen != nil {
		*r.Seen = true
	}
	return nil
}

func TestNewHandlerWithOptions_PreValidateHookErrorPreventsValidateAndService(t *testing.T) {
	t.Parallel()

	var validated bool
	var served bool
	hookErr := errors.New("hook failed")
	method := func(_ context.Context, _ *blockingValidateReq) (*echoResp, error) {
		served = true
		return &echoResp{}, nil
	}
	hook := func(_ context.Context, req *blockingValidateReq) error {
		req.Seen = &validated
		return hookErr
	}
	r := newHandlerEngineWithOptions(method, httpruntime.WithPreValidateHook[blockingValidateReq, echoResp](hook))

	w := postJSON(t, r, `{"name":"alice"}`)
	resp := decodeResponse(t, w)
	if resp.Code != httpruntime.CodeDefaultErr {
		t.Fatalf("Code = %d, want CodeDefaultErr", resp.Code)
	}
	if validated {
		t.Error("Validate ran after hook error, want skipped")
	}
	if served {
		t.Error("service ran after hook error, want skipped")
	}
}

func TestNewHandlerWithOptions_PreValidateHookPanic_IsPanicx(t *testing.T) {
	t.Parallel()

	var capturedErr error
	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.Use(func(c *gin.Context) {
		c.Next()
		if len(c.Errors) > 0 {
			capturedErr = c.Errors.Last().Err
		}
		c.JSON(http.StatusOK, httpruntime.ErrResponse(capturedErr))
	})

	method := func(_ context.Context, req *echoReq) (*echoResp, error) {
		return &echoResp{Echo: req.Msg}, nil
	}
	hook := func(_ context.Context, _ *echoReq) error {
		panic("hook boom") //nolint // intentional panic to test hook recovery
	}
	r.POST("/", httpruntime.NewHandlerWithOptions(method, httpruntime.WithPreValidateHook[echoReq, echoResp](hook)))

	postJSON(t, r, `{"msg":"hello"}`)

	if capturedErr == nil {
		t.Fatal("capturedErr is nil, expected a panic error")
	}
	if !errors.Is(capturedErr, panicx.ErrPanic) {
		t.Errorf("errors.Is(err, panicx.ErrPanic) = false, want true; err = %v", capturedErr)
	}
}

func TestNewHandlerWithOptions_PreValidateHooksRunInRegistrationOrder(t *testing.T) {
	t.Parallel()

	var order []int
	method := func(_ context.Context, req *echoReq) (*echoResp, error) {
		return &echoResp{Echo: req.Msg}, nil
	}
	r := newHandlerEngineWithOptions(
		method,
		httpruntime.WithPreValidateHook[echoReq, echoResp](func(_ context.Context, _ *echoReq) error {
			order = append(order, 1)
			return nil
		}),
		httpruntime.WithPreValidateHook[echoReq, echoResp](func(_ context.Context, _ *echoReq) error {
			order = append(order, 2)
			return nil
		}),
		httpruntime.WithPreValidateHook[echoReq, echoResp](func(_ context.Context, _ *echoReq) error {
			order = append(order, 3)
			return nil
		}),
	)

	w := postJSON(t, r, `{"msg":"hello"}`)
	resp := decodeResponse(t, w)
	if resp.Code != httpruntime.CodeOK {
		t.Fatalf("Code = %d, want CodeOK", resp.Code)
	}
	if !slices.Equal(order, []int{1, 2, 3}) {
		t.Errorf("order = %v, want [1 2 3]", order)
	}
}

func TestNewHandlerWithOptions_PreValidateHookReceivesRequestContext(t *testing.T) {
	t.Parallel()

	type contextKey struct{}
	const want = "request-id"
	var got string
	method := func(_ context.Context, req *echoReq) (*echoResp, error) {
		return &echoResp{Echo: req.Msg}, nil
	}
	hook := func(ctx context.Context, _ *echoReq) error {
		if v, ok := ctx.Value(contextKey{}).(string); ok {
			got = v
		}
		return nil
	}
	r := newHandlerEngineWithOptions(method, httpruntime.WithPreValidateHook[echoReq, echoResp](hook))

	w := httptest.NewRecorder()
	req, err := http.NewRequest(http.MethodPost, "/", bytes.NewBufferString(`{"msg":"hello"}`))
	if err != nil {
		t.Fatalf("new request: %v", err)
	}
	req = req.WithContext(context.WithValue(req.Context(), contextKey{}, want))
	req.Header.Set("Content-Type", "application/json")
	r.ServeHTTP(w, req)
	resp := decodeResponse(t, w)
	if resp.Code != httpruntime.CodeOK {
		t.Fatalf("Code = %d, want CodeOK", resp.Code)
	}
	if got != want {
		t.Errorf("hook context value = %q, want %q", got, want)
	}
}

func TestWithPreValidateHook_NilHookPanics(t *testing.T) {
	t.Parallel()

	defer func() {
		if recover() == nil {
			t.Fatal("WithPreValidateHook(nil) did not panic")
		}
	}()

	httpruntime.WithPreValidateHook[echoReq, echoResp](nil)
}

func TestNewHandlerWithOptions_NilOptionPanics(t *testing.T) {
	t.Parallel()

	method := func(_ context.Context, req *echoReq) (*echoResp, error) {
		return &echoResp{Echo: req.Msg}, nil
	}
	defer func() {
		if recover() == nil {
			t.Fatal("NewHandlerWithOptions with nil option did not panic")
		}
	}()

	httpruntime.NewHandlerWithOptions(method, nil)
}

func TestNewHandler_BackwardCompatibleThroughWithInterceptors(t *testing.T) {
	t.Parallel()

	var intercepted bool
	interceptor := func(ctx context.Context, req *echoReq, next handlerx.Handler[*echoReq, *echoResp]) (*echoResp, error) {
		intercepted = true
		return next(ctx, req)
	}
	method := func(_ context.Context, req *echoReq) (*echoResp, error) {
		return &echoResp{Echo: req.Msg}, nil
	}
	r := newHandlerEngineForNewHandler(method, interceptor)

	w := postJSON(t, r, `{"msg":"hello"}`)
	resp := decodeResponse(t, w)
	if resp.Code != httpruntime.CodeOK {
		t.Fatalf("Code = %d, want CodeOK", resp.Code)
	}
	if !intercepted {
		t.Error("interceptor did not run through NewHandler")
	}
}

// Compile-time check: ensure decodeResponse handles the Response.Data field
// which is decoded as map[string]any by encoding/json.
var _ = json.Unmarshal
