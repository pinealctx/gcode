// Package compat_test — UpdatePersonHandler validate error path.
// This is the only semantically meaningful missing test: the validate error
// branch in UpdatePersonHandler was not covered.
package compat_test

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/pinealctx/gcode/httpruntime"
	"github.com/pinealctx/gcode/testdata/compat/dao"
)

// TestUpdatePersonHandlerValidationError verifies that UpdatePersonHandler
// returns code 400 when req.Validate() fails.
// PersonUpdateByName.name has min_len=1; an empty name triggers the constraint.
func TestUpdatePersonHandlerValidationError(t *testing.T) {
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
	// name="" triggers min_len=1 from PersonUpdateByName.Validate()
	req, _ := http.NewRequest(http.MethodPost, "/", strings.NewReader(`{"name":""}`))
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
