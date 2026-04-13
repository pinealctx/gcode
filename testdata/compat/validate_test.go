// Package compat_test — end-to-end tests for generated Validate() methods.
// Verifies runtime behaviour of the generated validation code in dao/.
package compat_test

import (
	"testing"

	"github.com/pinealctx/gcode/testdata/compat/dao"
	"github.com/pinealctx/gcode/validateruntime"
)

// ptr helpers for create/update message pointer fields.
func int32Ptr(v int32) *int32       { return &v }
func float32Ptr(v float32) *float32 { return &v }
func boolPtr(v bool) *bool          { return &v }

// validPersonCreate returns a dao.PersonCreate that satisfies all validate constraints.
func validPersonCreate() *dao.PersonCreate {
	return &dao.PersonCreate{
		Name:     strPtr("Alice"),
		Age:      int32Ptr(30),
		Active:   boolPtr(true),
		Status:   ptrStatus(dao.Status_STATUS_ACTIVE),
		Rating:   float32Ptr(4.5),
		Nickname: "Alice", // required field, not optional
		Email:    strPtr("alice@example.com"),
		Role:     strPtr("admin"),
		TypeId:   int32Ptr(1),
	}
}

func ptrStatus(s dao.Status) *dao.Status { return &s }

// assertVE checks that err is a *validateruntime.ValidationError with expected field and rule.
func assertVE(t *testing.T, err error, wantField, wantRule string) {
	t.Helper()
	if err == nil {
		t.Fatalf("expected ValidationError{field=%q, rule=%q}, got nil", wantField, wantRule)
	}
	ve, ok := err.(*validateruntime.ValidationError)
	if !ok {
		t.Fatalf("expected *validateruntime.ValidationError, got %T: %v", err, err)
	}
	if ve.Field != wantField {
		t.Errorf("Field: got %q, want %q", ve.Field, wantField)
	}
	if ve.Rule != wantRule {
		t.Errorf("Rule: got %q, want %q", ve.Rule, wantRule)
	}
}

// TestValidate_ValidPersonCreate verifies that a fully valid PersonCreate passes Validate().
func TestValidate_ValidPersonCreate(t *testing.T) {
	t.Parallel()
	if err := validPersonCreate().Validate(); err != nil {
		t.Errorf("expected nil, got %v", err)
	}
}

// TestValidate_Address_NoConstraints verifies Address (no constraints) always passes.
func TestValidate_Address_NoConstraints(t *testing.T) {
	t.Parallel()
	a := &dao.Address{}
	if err := a.Validate(); err != nil {
		t.Errorf("Address.Validate() = %v, want nil", err)
	}
}

// TestValidate_StringMinLen verifies min_len / max_len constraints on name.
// name is optional in PersonCreate; when set to a value violating max_len it must fail.
func TestValidate_StringMinLen(t *testing.T) {
	t.Parallel()
	// name="" with non-nil pointer — zero-value guard skips min_len
	p := validPersonCreate()
	p.Name = strPtr("")
	if err := p.Validate(); err != nil {
		t.Errorf("empty name (no required): got %v, want nil", err)
	}

	// name too long
	p2 := validPersonCreate()
	p2.Name = strPtr(string(make([]byte, 101)))
	assertVE(t, p2.Validate(), "name", "max_len")
}

// TestValidate_IntGteLte verifies gte/lte constraints on age.
func TestValidate_IntGteLte(t *testing.T) {
	t.Parallel()
	p := validPersonCreate()
	p.Age = int32Ptr(-1)
	assertVE(t, p.Validate(), "age", "gte")

	p2 := validPersonCreate()
	p2.Age = int32Ptr(151)
	assertVE(t, p2.Validate(), "age", "lte")
}

// TestValidate_EnumDefinedOnly verifies defined_only constraint on status.
func TestValidate_EnumDefinedOnly(t *testing.T) {
	t.Parallel()
	p := validPersonCreate()
	p.Status = ptrStatus(dao.Status(99))
	assertVE(t, p.Validate(), "status", "defined_only")
}

// TestValidate_EmailConstraint verifies email constraint.
func TestValidate_EmailConstraint(t *testing.T) {
	t.Parallel()
	p := validPersonCreate()
	p.Email = strPtr("not-an-email")
	assertVE(t, p.Validate(), "email", "email")

	// empty email with non-nil pointer — zero-value guard skips check
	p2 := validPersonCreate()
	p2.Email = strPtr("")
	if err := p2.Validate(); err != nil {
		t.Errorf("empty email (no required): got %v, want nil", err)
	}
}

// TestValidate_StringIn verifies in constraint on role.
func TestValidate_StringIn(t *testing.T) {
	t.Parallel()
	p := validPersonCreate()
	p.Role = strPtr("superuser")
	assertVE(t, p.Validate(), "role", "in")

	p2 := validPersonCreate()
	p2.Role = strPtr("admin")
	if err := p2.Validate(); err != nil {
		t.Errorf("role=admin: got %v, want nil", err)
	}

	p3 := validPersonCreate()
	p3.Role = strPtr("")
	if err := p3.Validate(); err != nil {
		t.Errorf("role empty (no required): got %v, want nil", err)
	}
}

// TestValidate_OptionalStringConstraintApplied verifies that an optional string
// field is validated when set to a non-empty value that violates a constraint.
func TestValidate_OptionalStringConstraintApplied(t *testing.T) {
	t.Parallel()
	// nickname has max_len=10; a value exceeding that must fail.
	p := validPersonCreate()
	p.Nickname = "this-is-too-long"
	assertVE(t, p.Validate(), "nickname", "max_len")

	// valid non-empty value passes.
	p2 := validPersonCreate()
	p2.Nickname = "ali"
	if err := p2.Validate(); err != nil {
		t.Errorf("nickname=%q should pass, got: %v", "ali", err)
	}
}

func TestValidate_IntNotIn(t *testing.T) {
	t.Parallel()
	p := validPersonCreate()
	p.TypeId = int32Ptr(0)
	assertVE(t, p.Validate(), "type_id", "not_in")

	p2 := validPersonCreate()
	p2.TypeId = int32Ptr(-1)
	assertVE(t, p2.Validate(), "type_id", "not_in")

	p3 := validPersonCreate()
	p3.TypeId = int32Ptr(1)
	if err := p3.Validate(); err != nil {
		t.Errorf("type_id=1: got %v, want nil", err)
	}
}
