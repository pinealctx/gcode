// Package compat_test — end-to-end tests for generated Validate() methods.
// Verifies runtime behaviour of the generated validation code in dao/.
package compat_test

import (
	"testing"

	"github.com/pinealctx/gcode/testdata/compat/dao"
	"github.com/pinealctx/gcode/validateruntime"
)

// validPerson returns a dao.Person that satisfies all validate constraints.
func validPerson() *dao.Person {
	return &dao.Person{
		Name:      "Alice",
		Age:       30,
		Active:    true,
		Status:    dao.Status_STATUS_ACTIVE,
		Address:   &dao.Address{Street: "123 Main St", City: "Springfield"},
		Scores:    []int32{10, 20, 30},
		Tags:      []string{"go", "proto"},
		Rating:    4.5,
		CreatedAt: 1700000000,
		Avatar:    []byte{0x01, 0x02, 0x03},
		Email:     "alice@example.com",
		Role:      "admin",
		TypeId:    1,
	}
}

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

// TestValidate_ValidPerson verifies that a fully valid Person passes Validate().
func TestValidate_ValidPerson(t *testing.T) {
	t.Parallel()
	if err := validPerson().Validate(); err != nil {
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
func TestValidate_StringMinLen(t *testing.T) {
	t.Parallel()
	// name="" is zero value — skips min_len check (no required constraint)
	p := validPerson()
	p.Name = ""
	if err := p.Validate(); err != nil {
		t.Errorf("empty name (no required): got %v, want nil", err)
	}

	// name too long
	p2 := validPerson()
	p2.Name = string(make([]byte, 101))
	assertVE(t, p2.Validate(), "name", "max_len")
}

// TestValidate_IntGteLte verifies gte/lte constraints on age.
func TestValidate_IntGteLte(t *testing.T) {
	t.Parallel()
	p := validPerson()
	p.Age = -1
	assertVE(t, p.Validate(), "age", "gte")

	p2 := validPerson()
	p2.Age = 151
	assertVE(t, p2.Validate(), "age", "lte")
}

// TestValidate_EnumDefinedOnly verifies defined_only constraint on status.
func TestValidate_EnumDefinedOnly(t *testing.T) {
	t.Parallel()
	p := validPerson()
	p.Status = dao.Status(99)
	assertVE(t, p.Validate(), "status", "defined_only")
}

// TestValidate_MessageRequired verifies required constraint on address.
func TestValidate_MessageRequired(t *testing.T) {
	t.Parallel()
	p := validPerson()
	p.Address = nil
	assertVE(t, p.Validate(), "address", "required")
}

// TestValidate_NestedMessage verifies recursive Validate() call on address.
func TestValidate_NestedMessage(t *testing.T) {
	t.Parallel()
	// Address has no constraints, so nested call always returns nil.
	p := validPerson()
	p.Address = &dao.Address{Street: "", City: ""}
	if err := p.Validate(); err != nil {
		t.Errorf("nested address with empty fields: got %v, want nil", err)
	}
}

// TestValidate_RepeatedMinItems verifies min_items on scores.
func TestValidate_RepeatedMinItems(t *testing.T) {
	t.Parallel()
	p := validPerson()
	p.Scores = nil
	assertVE(t, p.Validate(), "scores", "min_items")
}

// TestValidate_RepeatedMaxItems verifies max_items on scores.
func TestValidate_RepeatedMaxItems(t *testing.T) {
	t.Parallel()
	p := validPerson()
	p.Scores = make([]int32, 101)
	assertVE(t, p.Validate(), "scores", "max_items")
}

// TestValidate_RepeatedItemsConstraint verifies items.string.min_len on tags.
func TestValidate_RepeatedItemsConstraint(t *testing.T) {
	t.Parallel()
	p := validPerson()
	p.Tags = []string{"go", ""}
	assertVE(t, p.Validate(), "tags[1]", "min_len")
}

// TestValidate_BytesRequired verifies required constraint on avatar.
func TestValidate_BytesRequired(t *testing.T) {
	t.Parallel()
	p := validPerson()
	p.Avatar = nil
	assertVE(t, p.Validate(), "avatar", "required")
}

// TestValidate_BytesMinLen verifies min_len on avatar ([]byte{} is non-nil but empty).
func TestValidate_BytesMinLen(t *testing.T) {
	t.Parallel()
	p := validPerson()
	p.Avatar = []byte{}
	assertVE(t, p.Validate(), "avatar", "min_len")
}

// TestValidate_EmailConstraint verifies email constraint.
func TestValidate_EmailConstraint(t *testing.T) {
	t.Parallel()
	p := validPerson()
	p.Email = "not-an-email"
	assertVE(t, p.Validate(), "email", "email")

	// empty email is zero value — skips check (no required constraint)
	p2 := validPerson()
	p2.Email = ""
	if err := p2.Validate(); err != nil {
		t.Errorf("empty email (no required): got %v, want nil", err)
	}
}

// TestValidate_StringIn verifies in constraint on role.
func TestValidate_StringIn(t *testing.T) {
	t.Parallel()
	p := validPerson()
	p.Role = "superuser"
	assertVE(t, p.Validate(), "role", "in")

	p2 := validPerson()
	p2.Role = "admin"
	if err := p2.Validate(); err != nil {
		t.Errorf("role=admin: got %v, want nil", err)
	}

	p3 := validPerson()
	p3.Role = ""
	if err := p3.Validate(); err != nil {
		t.Errorf("role empty (no required): got %v, want nil", err)
	}
}

// TestValidate_OptionalStringNilSkipped verifies that an optional string field
// with constraints is skipped entirely when nil.
func TestValidate_OptionalStringNilSkipped(t *testing.T) {
	t.Parallel()
	p := validPerson()
	p.Nickname = nil
	if err := p.Validate(); err != nil {
		t.Errorf("nil optional nickname should skip validation, got: %v", err)
	}
}

// TestValidate_OptionalStringEmptySkipped verifies that an optional string field
// with min_len skips the constraint when set to empty string (zero-value guard).
func TestValidate_OptionalStringEmptySkipped(t *testing.T) {
	t.Parallel()
	empty := ""
	p := validPerson()
	p.Nickname = &empty
	if err := p.Validate(); err != nil {
		t.Errorf("optional nickname=\"\" should skip min_len, got: %v", err)
	}
}

// TestValidate_OptionalStringConstraintApplied verifies that an optional string
// field is validated when set to a non-empty value that violates a constraint.
func TestValidate_OptionalStringConstraintApplied(t *testing.T) {
	t.Parallel()
	// nickname has max_len=10; a value exceeding that must fail.
	long := "this-is-too-long"
	p := validPerson()
	p.Nickname = &long
	assertVE(t, p.Validate(), "nickname", "max_len")

	// valid non-empty value passes.
	nick := "ali"
	p2 := validPerson()
	p2.Nickname = &nick
	if err := p2.Validate(); err != nil {
		t.Errorf("optional nickname=%q should pass, got: %v", nick, err)
	}
}
func TestValidate_IntNotIn(t *testing.T) {
	t.Parallel()
	p := validPerson()
	p.TypeId = 0
	assertVE(t, p.Validate(), "type_id", "not_in")

	p2 := validPerson()
	p2.TypeId = -1
	assertVE(t, p2.Validate(), "type_id", "not_in")

	p3 := validPerson()
	p3.TypeId = 1
	if err := p3.Validate(); err != nil {
		t.Errorf("type_id=1: got %v, want nil", err)
	}
}
