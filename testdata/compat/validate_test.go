// Package compat_test — end-to-end tests for generated Validate() methods.
// Verifies runtime behaviour of the generated validation code in dao/.
package compat_test

import (
	"strings"
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

// TestValidate_NicknameRequired verifies that nickname (required field) fails when empty.
func TestValidate_NicknameRequired(t *testing.T) {
	t.Parallel()

	// fail: empty nickname violates min_len=1 (required semantics)
	p := validPersonCreate()
	p.Nickname = ""
	assertVE(t, p.Validate(), "nickname", "min_len")

	// fail: nil pointer is not possible (Nickname is non-optional string), but
	// a value exceeding max_len=10 must fail
	p2 := validPersonCreate()
	p2.Nickname = "toolongname"
	assertVE(t, p2.Validate(), "nickname", "max_len")
}

// validPersonUpdate returns a dao.PersonUpdateByName that satisfies all validate constraints.
func validPersonUpdate() *dao.PersonUpdateByName {
	return &dao.PersonUpdateByName{
		Name:     "Alice", // condition field, required
		Age:      int32Ptr(30),
		Status:   ptrStatus(dao.Status_STATUS_ACTIVE),
		Nickname: strPtr("Ali"),
		Email:    strPtr("alice@example.com"),
		Role:     strPtr("admin"),
		TypeId:   int32Ptr(1),
	}
}

// TestValidate_ValidPersonUpdate verifies that a fully valid PersonUpdateByName passes Validate().
func TestValidate_ValidPersonUpdate(t *testing.T) {
	t.Parallel()
	if err := validPersonUpdate().Validate(); err != nil {
		t.Errorf("expected nil, got %v", err)
	}
}

// TestValidate_Update_NameRequired verifies that the condition field name is required.
func TestValidate_Update_NameRequired(t *testing.T) {
	t.Parallel()

	// fail: empty name violates min_len=1
	p := validPersonUpdate()
	p.Name = ""
	assertVE(t, p.Validate(), "name", "min_len")

	// fail: name too long
	p2 := validPersonUpdate()
	p2.Name = string(make([]byte, 101))
	assertVE(t, p2.Validate(), "name", "max_len")
}

// TestValidate_Update_OptionalFieldsSkipped verifies nil optional fields skip validation.
func TestValidate_Update_OptionalFieldsSkipped(t *testing.T) {
	t.Parallel()

	// all optional fields nil — only name is validated
	p := &dao.PersonUpdateByName{Name: "Alice"}
	if err := p.Validate(); err != nil {
		t.Errorf("update with only name set should pass, got: %v", err)
	}
}

// TestValidate_Update_IntNotIn verifies not_in constraint on type_id in update.
func TestValidate_Update_IntNotIn(t *testing.T) {
	t.Parallel()

	p := validPersonUpdate()
	p.TypeId = int32Ptr(0)
	assertVE(t, p.Validate(), "type_id", "not_in")

	p2 := validPersonUpdate()
	p2.TypeId = int32Ptr(-1)
	assertVE(t, p2.Validate(), "type_id", "not_in")

	// pass: nil skips check
	p3 := validPersonUpdate()
	p3.TypeId = nil
	if err := p3.Validate(); err != nil {
		t.Errorf("nil type_id should skip not_in check, got: %v", err)
	}
}

// TestValidate_Update_StringIn verifies in constraint on role in update.
func TestValidate_Update_StringIn(t *testing.T) {
	t.Parallel()

	p := validPersonUpdate()
	p.Role = strPtr("superuser")
	assertVE(t, p.Validate(), "role", "in")

	// pass: nil skips check
	p2 := validPersonUpdate()
	p2.Role = nil
	if err := p2.Validate(); err != nil {
		t.Errorf("nil role should skip in check, got: %v", err)
	}
}

// TestValidate_Update_NicknameOptional verifies optional nickname constraints in update.
func TestValidate_Update_NicknameOptional(t *testing.T) {
	t.Parallel()

	// fail: non-empty value exceeding max_len
	p := validPersonUpdate()
	p.Nickname = strPtr("toolongname")
	assertVE(t, p.Validate(), "nickname", "max_len")

	// pass: nil skips check
	p2 := validPersonUpdate()
	p2.Nickname = nil
	if err := p2.Validate(); err != nil {
		t.Errorf("nil nickname should skip check, got: %v", err)
	}

	// pass: empty string skips check (zero-value guard)
	p3 := validPersonUpdate()
	p3.Nickname = strPtr("")
	if err := p3.Validate(); err != nil {
		t.Errorf("empty nickname should skip check, got: %v", err)
	}
}

// --- T7: enum defined_only cross-file scenarios ---

// TestValidate_EnumDefinedOnly_AllValidValues verifies that every defined enum
// value passes the defined_only check. The Status enum is defined in
// person.meta.proto and used in person.entity.proto — cross-file resolution
// depends on the global EnumIndex built from all input files.
func TestValidate_EnumDefinedOnly_AllValidValues(t *testing.T) {
	t.Parallel()
	for _, s := range []dao.Status{
		dao.Status_STATUS_UNSPECIFIED,
		dao.Status_STATUS_ACTIVE,
		dao.Status_STATUS_INACTIVE,
	} {
		p := validPersonCreate()
		p.Status = ptrStatus(s)
		if err := p.Validate(); err != nil {
			t.Errorf("Status=%v should pass defined_only, got: %v", s, err)
		}
	}
}

// TestValidate_EnumDefinedOnly_NilSkipsCheck verifies that nil *Status
// skips defined_only — PersonCreate.Status is *Status (optional pointer).
func TestValidate_EnumDefinedOnly_NilSkipsCheck(t *testing.T) {
	t.Parallel()
	p := validPersonCreate()
	p.Status = nil
	if err := p.Validate(); err != nil {
		t.Errorf("nil Status should skip defined_only, got: %v", err)
	}
}

// TestValidate_Update_EnumDefinedOnly_AllValidValues tests the same cross-file
// defined_only scenario on PersonUpdateByName.
func TestValidate_Update_EnumDefinedOnly_AllValidValues(t *testing.T) {
	t.Parallel()
	for _, s := range []dao.Status{
		dao.Status_STATUS_UNSPECIFIED,
		dao.Status_STATUS_ACTIVE,
		dao.Status_STATUS_INACTIVE,
	} {
		u := validPersonUpdate()
		u.Status = ptrStatus(s)
		if err := u.Validate(); err != nil {
			t.Errorf("Status=%v should pass defined_only, got: %v", s, err)
		}
	}
}

// TestValidate_Update_EnumDefinedOnly_NilSkipsCheck verifies nil *Status
// skips defined_only on PersonUpdateByName.
func TestValidate_Update_EnumDefinedOnly_NilSkipsCheck(t *testing.T) {
	t.Parallel()
	u := validPersonUpdate()
	u.Status = nil
	if err := u.Validate(); err != nil {
		t.Errorf("nil Status should skip defined_only, got: %v", err)
	}
}

// --- T5 Gap 1: ignore_fields validate-level assertions ---

// TestValidate_Create_IgnoreFields_NoConstraintFired verifies that
// PersonCreate's Validate() works correctly when all present (non-ignored)
// fields are set to valid values. Fields excluded by ignore_fields
// (address, scores, tags, avatar, fingerprint, created_at) do not exist in
// the struct, so their constraints can never fire.
func TestValidate_Create_IgnoreFields_NoConstraintFired(t *testing.T) {
	t.Parallel()
	if err := validPersonCreate().Validate(); err != nil {
		t.Errorf("all-present-fields-valid PersonCreate should pass, got: %v", err)
	}
}

// TestValidate_Update_IgnoreFields_NoConstraintFired verifies the same
// ignore_fields guarantee for PersonUpdateByName (ignores address, scores,
// tags, avatar, fingerprint).
func TestValidate_Update_IgnoreFields_NoConstraintFired(t *testing.T) {
	t.Parallel()
	if err := validPersonUpdate().Validate(); err != nil {
		t.Errorf("all-present-fields-valid PersonUpdateByName should pass, got: %v", err)
	}
}

// TestValidate_Create_IgnoreFields_NonIgnoredStillEnforced verifies that
// constraints on non-ignored fields remain active. Age is present in both
// PersonCreate and Person with gte=0 — violating it must still fail.
func TestValidate_Create_IgnoreFields_NonIgnoredStillEnforced(t *testing.T) {
	t.Parallel()
	p := validPersonCreate()
	p.Age = int32Ptr(-1)
	assertVE(t, p.Validate(), "age", "gte")
}

// TestValidate_Update_IgnoreFields_NonIgnoredStillEnforced verifies that
// non-ignored field constraints still fire on PersonUpdateByName.
func TestValidate_Update_IgnoreFields_NonIgnoredStillEnforced(t *testing.T) {
	t.Parallel()
	u := validPersonUpdate()
	u.Age = int32Ptr(-1)
	assertVE(t, u.Validate(), "age", "gte")

	u2 := validPersonUpdate()
	u2.Age = int32Ptr(151)
	assertVE(t, u2.Validate(), "age", "lte")
}

// TestValidate_Create_OptionalNilSkips verifies that nil optional pointer fields
// (*string, *int32, *Status) on PersonCreate skip all validation, mirroring
// TestValidate_Update_OptionalFieldsSkipped on the update path.
func TestValidate_Create_OptionalNilSkips(t *testing.T) {
	t.Parallel()
	// Name and Age are optional (*string, *int32); nil must skip all constraints.
	c := &dao.PersonCreate{Nickname: "ali", Name: nil, Age: nil}
	if err := c.Validate(); err != nil {
		t.Errorf("nil optional Name/Age/Status should skip constraints, got: %v", err)
	}
}

// --- T5 Gap 2: required/condition field exact boundary values ---

// TestValidate_Create_NicknameBoundaryValues verifies PersonCreate.nickname
// (required_fields, min_len=1, max_len=10) at exact boundary lengths.
func TestValidate_Create_NicknameBoundaryValues(t *testing.T) {
	t.Parallel()
	// Exactly min_len=1 should pass
	p1 := validPersonCreate()
	p1.Nickname = "a"
	if err := p1.Validate(); err != nil {
		t.Errorf("nickname len=1 (exact min_len) should pass, got: %v", err)
	}

	// Exactly max_len=10 should pass
	p2 := validPersonCreate()
	p2.Nickname = "1234567890"
	if err := p2.Validate(); err != nil {
		t.Errorf("nickname len=10 (exact max_len) should pass, got: %v", err)
	}

	// One over max_len should fail
	p3 := validPersonCreate()
	p3.Nickname = "12345678901"
	assertVE(t, p3.Validate(), "nickname", "max_len")
}

// TestValidate_Update_NameBoundaryValues verifies PersonUpdateByName.name
// (condition_fields, min_len=1, max_len=100) at exact boundary lengths.
func TestValidate_Update_NameBoundaryValues(t *testing.T) {
	t.Parallel()
	// Exactly min_len=1 should pass
	u1 := validPersonUpdate()
	u1.Name = "a"
	if err := u1.Validate(); err != nil {
		t.Errorf("name len=1 (exact min_len) should pass, got: %v", err)
	}

	// Exactly max_len=100 should pass
	u2 := validPersonUpdate()
	u2.Name = strings.Repeat("a", 100)
	if err := u2.Validate(); err != nil {
		t.Errorf("name len=100 (exact max_len) should pass, got: %v", err)
	}

	// One over max_len should fail
	u3 := validPersonUpdate()
	u3.Name = strings.Repeat("a", 101)
	assertVE(t, u3.Validate(), "name", "max_len")
}
