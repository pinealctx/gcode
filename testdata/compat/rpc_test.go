// Package compat_test verifies the RPC interface generation pipeline:
// person_service.proto → person_service.pb.rpc.go → compilable Go interface.
package compat_test

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"

	"github.com/pinealctx/gcode/httpruntime"
	"github.com/pinealctx/gcode/testdata/compat/dao"
)

// mockPersonService is a compile-time check that the generated PersonService
// interface can be implemented. If the interface signature changes, this will
// fail to compile.
type mockPersonService struct {
	createFn func(ctx context.Context, req *dao.PersonCreate) (*dao.CreatePersonResponse, error)
	getFn    func(ctx context.Context, req *dao.GetPersonRequest) (*dao.GetPersonResponse, error)
	updateFn func(ctx context.Context, req *dao.PersonUpdateByName) (*dao.UpdatePersonResponse, error)
	deleteFn func(ctx context.Context, req *dao.DeletePersonRequest) (*dao.DeletePersonResponse, error)
}

func (m *mockPersonService) CreatePerson(ctx context.Context, req *dao.PersonCreate) (*dao.CreatePersonResponse, error) {
	if m.createFn != nil {
		return m.createFn(ctx, req)
	}
	return nil, errors.New("not implemented")
}

func (m *mockPersonService) GetPerson(ctx context.Context, req *dao.GetPersonRequest) (*dao.GetPersonResponse, error) {
	if m.getFn != nil {
		return m.getFn(ctx, req)
	}
	return nil, errors.New("not implemented")
}

func (m *mockPersonService) UpdatePerson(ctx context.Context, req *dao.PersonUpdateByName) (*dao.UpdatePersonResponse, error) {
	if m.updateFn != nil {
		return m.updateFn(ctx, req)
	}
	return nil, errors.New("not implemented")
}

func (m *mockPersonService) DeletePerson(ctx context.Context, req *dao.DeletePersonRequest) (*dao.DeletePersonResponse, error) {
	if m.deleteFn != nil {
		return m.deleteFn(ctx, req)
	}
	return nil, errors.New("not implemented")
}

// Compile-time assertion: mockPersonService implements dao.PersonService.
var _ dao.PersonService = (*mockPersonService)(nil)

// newTestEngine returns a gin engine in test mode with DefaultErrorHandler registered.
func newTestEngine() *gin.Engine {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.Use(httpruntime.DefaultErrorHandler())
	return r
}

// TestPersonServiceCommentPassthrough verifies that proto comments on the
// PersonService service and its CreatePerson rpc are present in the generated
// .pb.rpc.go snapshot. This is the end-to-end test for the comment passthrough
// pipeline: proto source → parser → transform → render → generated file.
func TestPersonServiceCommentPassthrough(t *testing.T) {
	t.Parallel()

	data, err := os.ReadFile("dao/person_service.pb.rpc.go")
	if err != nil {
		t.Fatalf("read generated file: %v", err)
	}
	src := string(data)

	// Service-level comment from person_service.proto.
	if !strings.Contains(src, "// PersonService provides CRUD operations for person records.") {
		t.Errorf("service comment not found in generated file:\n%s", src)
	}
	// Method-level comment from person_service.proto.
	if !strings.Contains(src, "CreatePerson creates a new person record.") {
		t.Errorf("method comment not found in generated file:\n%s", src)
	}
	// B1 regression guard: no double-space comment prefix.
	if strings.Contains(src, "//  ") {
		t.Errorf("generated file contains double-space comment prefix (B1 regression):\n%s", src)
	}
}

// TestPersonServiceDerivedMessageSignatures verifies that the generated
// PersonService interface correctly references derived message types
// (PersonCreate, PersonUpdateByName) produced by gen-proto.
func TestPersonServiceDerivedMessageSignatures(t *testing.T) {
	t.Parallel()

	data, err := os.ReadFile("dao/person_service.pb.rpc.go")
	if err != nil {
		t.Fatalf("read generated file: %v", err)
	}
	src := string(data)

	if !strings.Contains(src, "req *PersonCreate") {
		t.Errorf("expected *PersonCreate in CreatePerson signature:\n%s", src)
	}
	if !strings.Contains(src, "req *PersonUpdateByName") {
		t.Errorf("expected *PersonUpdateByName in UpdatePerson signature:\n%s", src)
	}
}

// TestCreatePersonHandlerSuccess verifies the generated CreatePersonHandler
// end-to-end: bind succeeds → svc returns response → OKResponse written.
func TestCreatePersonHandlerSuccess(t *testing.T) {
	t.Parallel()

	svc := &mockPersonService{
		createFn: func(_ context.Context, req *dao.PersonCreate) (*dao.CreatePersonResponse, error) {
			return &dao.CreatePersonResponse{Id: "42"}, nil
		},
	}

	r := newTestEngine()
	r.POST("/", dao.CreatePersonHandler(svc))

	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodPost, "/", strings.NewReader(`{"name":"Alice","age":30,"nickname":"Ali"}`))
	req.Header.Set("Content-Type", "application/json")
	r.ServeHTTP(w, req)

	var resp httpruntime.Response
	if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if resp.Code != httpruntime.CodeOK {
		t.Errorf("expected code 0, got %d", resp.Code)
	}
	if resp.Error != nil {
		t.Errorf("expected no error, got %+v", resp.Error)
	}
}

// TestCreatePersonHandlerSvcError verifies the generated CreatePersonHandler
// end-to-end: bind succeeds → svc returns error → ErrResponse written.
func TestCreatePersonHandlerSvcError(t *testing.T) {
	t.Parallel()

	svc := &mockPersonService{
		createFn: func(_ context.Context, _ *dao.PersonCreate) (*dao.CreatePersonResponse, error) {
			return nil, errors.New("db unavailable")
		},
	}

	r := newTestEngine()
	r.POST("/", dao.CreatePersonHandler(svc))

	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodPost, "/", strings.NewReader(`{"name":"Alice","age":30,"nickname":"Ali"}`))
	req.Header.Set("Content-Type", "application/json")
	r.ServeHTTP(w, req)

	var resp httpruntime.Response
	if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if resp.Code != httpruntime.CodeDefaultErr {
		t.Errorf("expected code 500, got %d", resp.Code)
	}
	if resp.Error == nil || resp.Error.Msg != "internal error" {
		t.Errorf("expected error msg 'internal error', got %+v", resp.Error)
	}
}

// TestCreatePersonHandlerBindError verifies the generated CreatePersonHandler
// end-to-end: bind fails → ErrResponse written, svc not called.
func TestCreatePersonHandlerBindError(t *testing.T) {
	t.Parallel()

	called := false
	svc := &mockPersonService{
		createFn: func(_ context.Context, _ *dao.PersonCreate) (*dao.CreatePersonResponse, error) {
			called = true
			return nil, nil
		},
	}

	r := newTestEngine()
	r.POST("/", dao.CreatePersonHandler(svc))

	w := httptest.NewRecorder()
	// Send invalid JSON to trigger bind error.
	req, _ := http.NewRequest(http.MethodPost, "/", strings.NewReader(`not-json`))
	req.Header.Set("Content-Type", "application/json")
	r.ServeHTTP(w, req)

	if called {
		t.Error("svc should not be called when binding fails")
	}
	var resp httpruntime.Response
	if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if resp.Code != httpruntime.CodeDefaultErr {
		t.Errorf("expected code 500 for bind failure, got %d", resp.Code)
	}
}

// TestUpdatePersonHandlerDerivedMessage verifies that UpdatePersonHandler
// correctly uses PersonUpdateByName (a derived message from gen-proto) as
// the request type — end-to-end derived message chain validation.
func TestUpdatePersonHandlerDerivedMessage(t *testing.T) {
	t.Parallel()

	svc := &mockPersonService{
		updateFn: func(_ context.Context, req *dao.PersonUpdateByName) (*dao.UpdatePersonResponse, error) {
			return &dao.UpdatePersonResponse{Ok: true}, nil
		},
	}

	r := newTestEngine()
	r.POST("/", dao.UpdatePersonHandler(svc))

	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodPost, "/", strings.NewReader(`{"name":"Alice"}`))
	req.Header.Set("Content-Type", "application/json")
	r.ServeHTTP(w, req)

	var resp httpruntime.Response
	if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if resp.Code != httpruntime.CodeOK {
		t.Errorf("expected code 0, got %d (error: %+v)", resp.Code, resp.Error)
	}
}

// TestUpdatePersonHandlerBindError verifies that UpdatePersonHandler returns
// an error response when binding fails, and does not call the service.
func TestUpdatePersonHandlerBindError(t *testing.T) {
	t.Parallel()

	called := false
	svc := &mockPersonService{
		updateFn: func(_ context.Context, _ *dao.PersonUpdateByName) (*dao.UpdatePersonResponse, error) {
			called = true
			return nil, nil
		},
	}

	r := newTestEngine()
	r.POST("/", dao.UpdatePersonHandler(svc))

	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodPost, "/", strings.NewReader(`not-json`))
	req.Header.Set("Content-Type", "application/json")
	r.ServeHTTP(w, req)

	if called {
		t.Error("svc should not be called when binding fails")
	}
	var resp httpruntime.Response
	if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if resp.Code != httpruntime.CodeDefaultErr {
		t.Errorf("expected code 500 for bind failure, got %d", resp.Code)
	}
}

// TestCreatePersonHandlerValidationError verifies that CreatePersonHandler
// forwards a ValidationError via c.Error, and DefaultErrorHandler maps it to
// code 400 — end-to-end validation of the bind → validate → svc pipeline.
func TestCreatePersonHandlerValidationError(t *testing.T) {
	t.Parallel()

	called := false
	svc := &mockPersonService{
		createFn: func(_ context.Context, _ *dao.PersonCreate) (*dao.CreatePersonResponse, error) {
			called = true
			return nil, nil
		},
	}

	r := newTestEngine()
	r.POST("/", dao.CreatePersonHandler(svc))

	w := httptest.NewRecorder()
	// nickname is a non-optional string field with min_len=1 constraint
	// (person.create.proto: string nickname = 7 [(buf.validate.field).string.min_len = 1]).
	// Omitting it triggers a ValidationError from req.Validate().
	req, _ := http.NewRequest(http.MethodPost, "/", strings.NewReader(`{"name":"Alice","age":30}`))
	req.Header.Set("Content-Type", "application/json")
	r.ServeHTTP(w, req)

	if called {
		t.Error("svc should not be called when validation fails")
	}
	var resp httpruntime.Response
	if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if resp.Code != httpruntime.CodeValidationErr {
		t.Errorf("expected code 400 for validation failure, got %d (body: %s)", resp.Code, w.Body.String())
	}
}

// TestUpdatePersonHandlerSvcError verifies that UpdatePersonHandler returns
// an error response when the service returns an error.
func TestUpdatePersonHandlerSvcError(t *testing.T) {
	t.Parallel()

	svc := &mockPersonService{
		updateFn: func(_ context.Context, _ *dao.PersonUpdateByName) (*dao.UpdatePersonResponse, error) {
			return nil, errors.New("update failed")
		},
	}

	r := newTestEngine()
	r.POST("/", dao.UpdatePersonHandler(svc))

	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodPost, "/", strings.NewReader(`{"name":"Alice"}`))
	req.Header.Set("Content-Type", "application/json")
	r.ServeHTTP(w, req)

	var resp httpruntime.Response
	if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if resp.Code != httpruntime.CodeDefaultErr {
		t.Errorf("expected code 500, got %d", resp.Code)
	}
	if resp.Error == nil || resp.Error.Msg != "internal error" {
		t.Errorf("expected error msg 'internal error', got %+v", resp.Error)
	}
}

// --- GetPerson handler tests -------------------------------------------------

// TestGetPersonHandlerSuccess verifies GetPersonHandler end-to-end:
// bind succeeds → validate passes → svc returns response → OKResponse written.
func TestGetPersonHandlerSuccess(t *testing.T) {
	t.Parallel()

	svc := &mockPersonService{
		getFn: func(_ context.Context, req *dao.GetPersonRequest) (*dao.GetPersonResponse, error) {
			return &dao.GetPersonResponse{Name: "Alice", Age: 30}, nil
		},
	}

	r := newTestEngine()
	r.POST("/", dao.GetPersonHandler(svc))

	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodPost, "/", strings.NewReader(`{"id":"42"}`))
	req.Header.Set("Content-Type", "application/json")
	r.ServeHTTP(w, req)

	var resp httpruntime.Response
	if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if resp.Code != httpruntime.CodeOK {
		t.Errorf("expected code 0, got %d (error: %+v)", resp.Code, resp.Error)
	}
}

// TestGetPersonHandlerSvcError verifies GetPersonHandler end-to-end:
// bind succeeds → validate passes → svc returns error → ErrResponse written.
func TestGetPersonHandlerSvcError(t *testing.T) {
	t.Parallel()

	svc := &mockPersonService{
		getFn: func(_ context.Context, _ *dao.GetPersonRequest) (*dao.GetPersonResponse, error) {
			return nil, errors.New("not found")
		},
	}

	r := newTestEngine()
	r.POST("/", dao.GetPersonHandler(svc))

	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodPost, "/", strings.NewReader(`{"id":"42"}`))
	req.Header.Set("Content-Type", "application/json")
	r.ServeHTTP(w, req)

	var resp httpruntime.Response
	if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if resp.Code != httpruntime.CodeDefaultErr {
		t.Errorf("expected code 500, got %d", resp.Code)
	}
	if resp.Error == nil || resp.Error.Msg != "internal error" {
		t.Errorf("expected error msg 'internal error', got %+v", resp.Error)
	}
}

// TestGetPersonHandlerBindError verifies GetPersonHandler end-to-end:
// bind fails → ErrResponse written, svc not called.
func TestGetPersonHandlerBindError(t *testing.T) {
	t.Parallel()

	called := false
	svc := &mockPersonService{
		getFn: func(_ context.Context, _ *dao.GetPersonRequest) (*dao.GetPersonResponse, error) {
			called = true
			return nil, nil
		},
	}

	r := newTestEngine()
	r.POST("/", dao.GetPersonHandler(svc))

	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodPost, "/", strings.NewReader(`not-json`))
	req.Header.Set("Content-Type", "application/json")
	r.ServeHTTP(w, req)

	if called {
		t.Error("svc should not be called when binding fails")
	}
	var resp httpruntime.Response
	if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if resp.Code != httpruntime.CodeDefaultErr {
		t.Errorf("expected code 500 for bind failure, got %d", resp.Code)
	}
}

// --- DeletePerson handler tests ----------------------------------------------

// TestDeletePersonHandlerSuccess verifies DeletePersonHandler end-to-end:
// bind succeeds → validate passes → svc returns response → OKResponse written.
func TestDeletePersonHandlerSuccess(t *testing.T) {
	t.Parallel()

	svc := &mockPersonService{
		deleteFn: func(_ context.Context, _ *dao.DeletePersonRequest) (*dao.DeletePersonResponse, error) {
			return &dao.DeletePersonResponse{Ok: true}, nil
		},
	}

	r := newTestEngine()
	r.POST("/", dao.DeletePersonHandler(svc))

	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodPost, "/", strings.NewReader(`{"id":"42"}`))
	req.Header.Set("Content-Type", "application/json")
	r.ServeHTTP(w, req)

	var resp httpruntime.Response
	if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if resp.Code != httpruntime.CodeOK {
		t.Errorf("expected code 0, got %d (error: %+v)", resp.Code, resp.Error)
	}
}

// TestDeletePersonHandlerSvcError verifies DeletePersonHandler end-to-end:
// bind succeeds → validate passes → svc returns error → ErrResponse written.
func TestDeletePersonHandlerSvcError(t *testing.T) {
	t.Parallel()

	svc := &mockPersonService{
		deleteFn: func(_ context.Context, _ *dao.DeletePersonRequest) (*dao.DeletePersonResponse, error) {
			return nil, errors.New("delete failed")
		},
	}

	r := newTestEngine()
	r.POST("/", dao.DeletePersonHandler(svc))

	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodPost, "/", strings.NewReader(`{"id":"42"}`))
	req.Header.Set("Content-Type", "application/json")
	r.ServeHTTP(w, req)

	var resp httpruntime.Response
	if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if resp.Code != httpruntime.CodeDefaultErr {
		t.Errorf("expected code 500, got %d", resp.Code)
	}
	if resp.Error == nil || resp.Error.Msg != "internal error" {
		t.Errorf("expected error msg 'internal error', got %+v", resp.Error)
	}
}

// TestDeletePersonHandlerBindError verifies DeletePersonHandler end-to-end:
// bind fails → ErrResponse written, svc not called.
func TestDeletePersonHandlerBindError(t *testing.T) {
	t.Parallel()

	called := false
	svc := &mockPersonService{
		deleteFn: func(_ context.Context, _ *dao.DeletePersonRequest) (*dao.DeletePersonResponse, error) {
			called = true
			return nil, nil
		},
	}

	r := newTestEngine()
	r.POST("/", dao.DeletePersonHandler(svc))

	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodPost, "/", strings.NewReader(`not-json`))
	req.Header.Set("Content-Type", "application/json")
	r.ServeHTTP(w, req)

	if called {
		t.Error("svc should not be called when binding fails")
	}
	var resp httpruntime.Response
	if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if resp.Code != httpruntime.CodeDefaultErr {
		t.Errorf("expected code 500 for bind failure, got %d", resp.Code)
	}
}

// TestGetPersonHandlerValidationError verifies GetPersonHandler end-to-end:
// bind succeeds → req.Validate() returns ValidationError → svc not called → code 400.
// GetPersonRequest.id has max_len=64; a 65-char id triggers the constraint.
func TestGetPersonHandlerValidationError(t *testing.T) {
	t.Parallel()

	called := false
	svc := &mockPersonService{
		getFn: func(_ context.Context, _ *dao.GetPersonRequest) (*dao.GetPersonResponse, error) {
			called = true
			return nil, nil
		},
	}

	r := newTestEngine()
	r.POST("/", dao.GetPersonHandler(svc))

	w := httptest.NewRecorder()
	// id exceeds max_len=64 (65 chars) — triggers ValidationError from req.Validate().
	req, _ := http.NewRequest(http.MethodPost, "/", strings.NewReader(`{"id":"`+strings.Repeat("x", 65)+`"}`))
	req.Header.Set("Content-Type", "application/json")
	r.ServeHTTP(w, req)

	if called {
		t.Error("svc should not be called when validation fails")
	}
	var resp httpruntime.Response
	if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if resp.Code != httpruntime.CodeValidationErr {
		t.Errorf("expected code 400 for validation failure, got %d (body: %s)", resp.Code, w.Body.String())
	}
}

// TestDeletePersonHandlerValidationError verifies DeletePersonHandler end-to-end:
// bind succeeds → req.Validate() returns ValidationError → svc not called → code 400.
// DeletePersonRequest.id has max_len=64; a 65-char id triggers the constraint.
func TestDeletePersonHandlerValidationError(t *testing.T) {
	t.Parallel()

	called := false
	svc := &mockPersonService{
		deleteFn: func(_ context.Context, _ *dao.DeletePersonRequest) (*dao.DeletePersonResponse, error) {
			called = true
			return nil, nil
		},
	}

	r := newTestEngine()
	r.POST("/", dao.DeletePersonHandler(svc))

	w := httptest.NewRecorder()
	// id exceeds max_len=64 (65 chars) — triggers ValidationError from req.Validate().
	req, _ := http.NewRequest(http.MethodPost, "/", strings.NewReader(`{"id":"`+strings.Repeat("x", 65)+`"}`))
	req.Header.Set("Content-Type", "application/json")
	r.ServeHTTP(w, req)

	if called {
		t.Error("svc should not be called when validation fails")
	}
	var resp httpruntime.Response
	if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if resp.Code != httpruntime.CodeValidationErr {
		t.Errorf("expected code 400 for validation failure, got %d (body: %s)", resp.Code, w.Body.String())
	}
}
