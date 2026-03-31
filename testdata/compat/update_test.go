// Package compat_test — end-to-end tests for generated update/create structs.
// Verifies ToMap(), validate inheritance, and condition_fields exclusion.
package compat_test

import (
	"testing"

	"github.com/pinealctx/gcode/testdata/compat/dao"
)

// --- ToMap tests ---

func TestPersonUpdateByName_ToMap_OnlyNonNilFields(t *testing.T) {
	t.Parallel()

	name := "Alice"
	age := int32(30)
	u := &dao.PersonUpdateByName{
		Name: name,
		Age:  &age,
		// all other fields nil
	}

	m := u.ToMap()

	// condition field (name) must NOT be in map.
	if _, ok := m["name"]; ok {
		t.Error("condition field 'name' must not appear in ToMap()")
	}
	// age is set → must be in map.
	if v, ok := m["age"]; !ok || v != age {
		t.Errorf("ToMap()[\"age\"] = %v, want %v", m["age"], age)
	}
	// nil fields must not be in map.
	for _, key := range []string{"active", "status", "rating", "created_ts", "nickname",
		"level", "verified", "score", "updatedAt", "prevStatus", "email", "role", "typeId"} {
		if _, ok := m[key]; ok {
			t.Errorf("nil field %q should not appear in ToMap()", key)
		}
	}
}

func TestPersonUpdateByName_ToMap_AllOptionalFields(t *testing.T) {
	t.Parallel()

	active := true
	nickname := "ali"
	email := "alice@example.com"
	u := &dao.PersonUpdateByName{
		Name:     "Alice",
		Active:   &active,
		Nickname: &nickname,
		Email:    &email,
	}

	m := u.ToMap()

	if _, ok := m["name"]; ok {
		t.Error("condition field 'name' must not appear in ToMap()")
	}
	if m["active"] != active {
		t.Errorf("ToMap()[\"active\"] = %v, want %v", m["active"], active)
	}
	if m["nickname"] != nickname {
		t.Errorf("ToMap()[\"nickname\"] = %v, want %v", m["nickname"], nickname)
	}
	if m["email"] != email {
		t.Errorf("ToMap()[\"email\"] = %v, want %v", m["email"], email)
	}
}

func TestPersonUpdateByName_ToMap_EmptyWhenAllNil(t *testing.T) {
	t.Parallel()

	u := &dao.PersonUpdateByName{Name: "Alice"}
	m := u.ToMap()

	if len(m) != 0 {
		t.Errorf("ToMap() should be empty when all optional fields are nil, got %v", m)
	}
}

// --- Validate inheritance tests ---

func TestPersonUpdateByName_Validate_NilFieldsSkipped(t *testing.T) {
	t.Parallel()

	// Only name set — all optional fields nil, should pass.
	u := &dao.PersonUpdateByName{Name: "Alice"}
	if err := u.Validate(); err != nil {
		t.Errorf("Validate() with nil optional fields should pass, got: %v", err)
	}
}

func TestPersonUpdateByName_Validate_NameConstraintInherited(t *testing.T) {
	t.Parallel()

	// name is condition field (non-optional) — min_len=1 inherited from Person.
	u := &dao.PersonUpdateByName{Name: ""}
	err := u.Validate()
	assertVE(t, err, "name", "min_len")
}

func TestPersonUpdateByName_Validate_OptionalFieldValidatedWhenSet(t *testing.T) {
	t.Parallel()

	// age is optional — when set, gte=0 lte=150 inherited from Person.
	age := int32(200)
	u := &dao.PersonUpdateByName{Name: "Alice", Age: &age}
	err := u.Validate()
	assertVE(t, err, "age", "lte")
}

func TestPersonUpdateByName_Validate_OptionalEmailValidatedWhenSet(t *testing.T) {
	t.Parallel()

	bad := "not-an-email"
	u := &dao.PersonUpdateByName{Name: "Alice", Email: &bad}
	err := u.Validate()
	assertVE(t, err, "email", "email")
}

func TestPersonUpdateByName_Validate_OptionalEmailNilSkipped(t *testing.T) {
	t.Parallel()

	// email nil → skip validation.
	u := &dao.PersonUpdateByName{Name: "Alice", Email: nil}
	if err := u.Validate(); err != nil {
		t.Errorf("nil email should skip validation, got: %v", err)
	}
}

// --- inherited optional enum, int not_in, string in ---

func TestPersonUpdateByName_Validate_OptionalEnumNilSkipped(t *testing.T) {
	t.Parallel()

	// status is optional in PersonUpdateByName; nil must skip defined_only check.
	u := &dao.PersonUpdateByName{Name: "Alice", Status: nil}
	if err := u.Validate(); err != nil {
		t.Errorf("nil optional status should skip validation, got: %v", err)
	}
}

func TestPersonUpdateByName_Validate_OptionalEnumValidatedWhenSet(t *testing.T) {
	t.Parallel()

	// status set to undefined value must trigger defined_only.
	bad := dao.Status(99)
	u := &dao.PersonUpdateByName{Name: "Alice", Status: &bad}
	assertVE(t, u.Validate(), "status", "defined_only")
}

func TestPersonUpdateByName_Validate_OptionalIntNotInNilSkipped(t *testing.T) {
	t.Parallel()

	// type_id is optional; nil must skip not_in check.
	u := &dao.PersonUpdateByName{Name: "Alice", TypeId: nil}
	if err := u.Validate(); err != nil {
		t.Errorf("nil optional type_id should skip validation, got: %v", err)
	}
}

func TestPersonUpdateByName_Validate_OptionalIntNotInValidatedWhenSet(t *testing.T) {
	t.Parallel()

	// type_id set to forbidden value must trigger not_in.
	bad := int32(0)
	u := &dao.PersonUpdateByName{Name: "Alice", TypeId: &bad}
	assertVE(t, u.Validate(), "type_id", "not_in")
}

func TestPersonUpdateByName_Validate_OptionalStringInNilSkipped(t *testing.T) {
	t.Parallel()

	// role is optional; nil must skip in check.
	u := &dao.PersonUpdateByName{Name: "Alice", Role: nil}
	if err := u.Validate(); err != nil {
		t.Errorf("nil optional role should skip validation, got: %v", err)
	}
}

func TestPersonUpdateByName_Validate_OptionalStringInValidatedWhenSet(t *testing.T) {
	t.Parallel()

	// role set to value not in allowed list must trigger in.
	bad := "superuser"
	u := &dao.PersonUpdateByName{Name: "Alice", Role: &bad}
	assertVE(t, u.Validate(), "role", "in")

	// all valid values in the in-list must pass (exhaustive for small set).
	for _, good := range []string{"admin", "user", "guest"} {
		v := good
		u2 := &dao.PersonUpdateByName{Name: "Alice", Role: &v}
		if err := u2.Validate(); err != nil {
			t.Errorf("role=%q should pass, got: %v", good, err)
		}
	}
}

func TestPersonUpdateByName_Validate_OptionalStringInEmptySkipped(t *testing.T) {
	t.Parallel()

	// role set to empty string: zero-value guard skips in check.
	empty := ""
	u := &dao.PersonUpdateByName{Name: "Alice", Role: &empty}
	if err := u.Validate(); err != nil {
		t.Errorf("role=\"\" should skip in check, got: %v", err)
	}
}

func TestPersonUpdateByName_Validate_OptionalEmailEmptySkipped(t *testing.T) {
	t.Parallel()

	// email set to empty string: zero-value guard skips email check.
	empty := ""
	u := &dao.PersonUpdateByName{Name: "Alice", Email: &empty}
	if err := u.Validate(); err != nil {
		t.Errorf("email=\"\" should skip email check, got: %v", err)
	}
}

func TestPersonUpdateByName_Validate_OptionalIntNotInSecondValue(t *testing.T) {
	t.Parallel()

	// type_id=-1 is also forbidden (second not_in value).
	bad := int32(-1)
	u := &dao.PersonUpdateByName{Name: "Alice", TypeId: &bad}
	assertVE(t, u.Validate(), "type_id", "not_in")
}

func TestPersonUpdateByName_Validate_NameMaxLen(t *testing.T) {
	t.Parallel()

	// name has max_len=100 inherited from Person; exceeding it must trigger max_len.
	u := &dao.PersonUpdateByName{Name: string(make([]byte, 101))}
	assertVE(t, u.Validate(), "name", "max_len")
}

func TestPersonUpdateByName_Validate_OptionalNicknameNilSkipped(t *testing.T) {
	t.Parallel()

	u := &dao.PersonUpdateByName{Name: "Alice", Nickname: nil}
	if err := u.Validate(); err != nil {
		t.Errorf("nil optional nickname should skip validation, got: %v", err)
	}
}

func TestPersonUpdateByName_Validate_OptionalNicknameEmptySkipped(t *testing.T) {
	t.Parallel()

	empty := ""
	u := &dao.PersonUpdateByName{Name: "Alice", Nickname: &empty}
	if err := u.Validate(); err != nil {
		t.Errorf("nickname=\"\" should skip min_len, got: %v", err)
	}
}

func TestPersonUpdateByName_Validate_OptionalNicknameMaxLen(t *testing.T) {
	t.Parallel()

	long := "this-is-long"
	u := &dao.PersonUpdateByName{Name: "Alice", Nickname: &long}
	assertVE(t, u.Validate(), "nickname", "max_len")
}

func TestPersonCreate_RequiredFieldNonOptional(t *testing.T) {
	t.Parallel()

	// nickname is required_fields → non-optional in PersonCreate.
	// min_len=1 inherited from Person; non-empty value passes.
	c := &dao.PersonCreate{Name: strPtr("Alice"), Nickname: "ali"}
	if err := c.Validate(); err != nil {
		t.Errorf("valid PersonCreate should pass Validate(), got: %v", err)
	}
}

func TestPersonCreate_RequiredNicknameConstraintInherited(t *testing.T) {
	t.Parallel()

	// nickname is required (non-optional) with min_len=1 inherited from Person.
	// Empty string must trigger min_len — noZeroGuard path.
	c := &dao.PersonCreate{Name: strPtr("Alice"), Nickname: ""}
	assertVE(t, c.Validate(), "nickname", "min_len")
}

func TestPersonCreate_RequiredNicknameMaxLen(t *testing.T) {
	t.Parallel()

	// nickname has max_len=10; exceeding it must trigger max_len.
	c := &dao.PersonCreate{Name: strPtr("Alice"), Nickname: "this-is-long"}
	assertVE(t, c.Validate(), "nickname", "max_len")
}

func TestPersonCreate_OptionalFieldsNilSkipped(t *testing.T) {
	t.Parallel()

	// All optional fields nil — only nickname (required) set.
	c := &dao.PersonCreate{Nickname: "ali"}
	if err := c.Validate(); err != nil {
		t.Errorf("Validate() with nil optional fields should pass, got: %v", err)
	}
}

func TestPersonCreate_OptionalEmailValidatedWhenSet(t *testing.T) {
	t.Parallel()

	bad := "not-an-email"
	c := &dao.PersonCreate{Nickname: "ali", Email: &bad}
	err := c.Validate()
	assertVE(t, err, "email", "email")
}

func TestPersonCreate_OptionalAgeValidatedWhenSet(t *testing.T) {
	t.Parallel()

	age := int32(-1)
	c := &dao.PersonCreate{Nickname: "ali", Age: &age}
	err := c.Validate()
	assertVE(t, err, "age", "gte")
}

func TestPersonCreate_OptionalAgeLte(t *testing.T) {
	t.Parallel()

	age := int32(200)
	c := &dao.PersonCreate{Nickname: "ali", Age: &age}
	assertVE(t, c.Validate(), "age", "lte")
}

func TestPersonCreate_OptionalNameNilSkipped(t *testing.T) {
	t.Parallel()

	// name is optional in PersonCreate; nil must skip all name constraints.
	c := &dao.PersonCreate{Nickname: "ali", Name: nil}
	if err := c.Validate(); err != nil {
		t.Errorf("nil optional name should skip validation, got: %v", err)
	}
}

func TestPersonCreate_OptionalNameEmptySkipped(t *testing.T) {
	t.Parallel()

	// name set to empty string: zero-value guard skips min_len.
	c := &dao.PersonCreate{Nickname: "ali", Name: strPtr("")}
	if err := c.Validate(); err != nil {
		t.Errorf("name=\"\" should skip min_len, got: %v", err)
	}
}

func TestPersonCreate_OptionalNameMaxLen(t *testing.T) {
	t.Parallel()

	// name has max_len=100; exceeding it must trigger max_len.
	c := &dao.PersonCreate{Nickname: "ali", Name: strPtr(string(make([]byte, 101)))}
	assertVE(t, c.Validate(), "name", "max_len")
}

func TestPersonCreate_OptionalStatusNilSkipped(t *testing.T) {
	t.Parallel()

	c := &dao.PersonCreate{Nickname: "ali", Status: nil}
	if err := c.Validate(); err != nil {
		t.Errorf("nil optional status should skip validation, got: %v", err)
	}
}

func TestPersonCreate_OptionalStatusValidatedWhenSet(t *testing.T) {
	t.Parallel()

	bad := dao.Status(99)
	c := &dao.PersonCreate{Nickname: "ali", Status: &bad}
	assertVE(t, c.Validate(), "status", "defined_only")
}

func TestPersonCreate_OptionalEmailEmptySkipped(t *testing.T) {
	t.Parallel()

	c := &dao.PersonCreate{Nickname: "ali", Email: strPtr("")}
	if err := c.Validate(); err != nil {
		t.Errorf("email=\"\" should skip email check, got: %v", err)
	}
}

func TestPersonCreate_OptionalRoleNilSkipped(t *testing.T) {
	t.Parallel()

	c := &dao.PersonCreate{Nickname: "ali", Role: nil}
	if err := c.Validate(); err != nil {
		t.Errorf("nil optional role should skip validation, got: %v", err)
	}
}

func TestPersonCreate_OptionalRoleValidatedWhenSet(t *testing.T) {
	t.Parallel()

	bad := "superuser"
	c := &dao.PersonCreate{Nickname: "ali", Role: &bad}
	assertVE(t, c.Validate(), "role", "in")

	for _, good := range []string{"admin", "user", "guest"} {
		v := good
		c2 := &dao.PersonCreate{Nickname: "ali", Role: &v}
		if err := c2.Validate(); err != nil {
			t.Errorf("role=%q should pass, got: %v", good, err)
		}
	}
}

func TestPersonCreate_OptionalRoleEmptySkipped(t *testing.T) {
	t.Parallel()

	c := &dao.PersonCreate{Nickname: "ali", Role: strPtr("")}
	if err := c.Validate(); err != nil {
		t.Errorf("role=\"\" should skip in check, got: %v", err)
	}
}

func TestPersonCreate_OptionalTypeIdNilSkipped(t *testing.T) {
	t.Parallel()

	c := &dao.PersonCreate{Nickname: "ali", TypeId: nil}
	if err := c.Validate(); err != nil {
		t.Errorf("nil optional type_id should skip validation, got: %v", err)
	}
}

func TestPersonCreate_OptionalTypeIdNotIn(t *testing.T) {
	t.Parallel()

	for _, bad := range []int32{0, -1} {
		v := bad
		c := &dao.PersonCreate{Nickname: "ali", TypeId: &v}
		assertVE(t, c.Validate(), "type_id", "not_in")
	}
}

func TestPersonCreate_OptionalTypeIdValidValue(t *testing.T) {
	t.Parallel()

	good := int32(1)
	c := &dao.PersonCreate{Nickname: "ali", TypeId: &good}
	if err := c.Validate(); err != nil {
		t.Errorf("type_id=1 should pass, got: %v", err)
	}
}

func TestPersonCreate_OptionalEmailNilSkipped(t *testing.T) {
	t.Parallel()

	c := &dao.PersonCreate{Nickname: "ali", Email: nil}
	if err := c.Validate(); err != nil {
		t.Errorf("nil optional email should skip validation, got: %v", err)
	}
}

func TestPersonCreate_OptionalStatusAllValidValues(t *testing.T) {
	t.Parallel()

	// Exhaustive check: all defined Status values must pass defined_only.
	for _, good := range []dao.Status{
		dao.Status_STATUS_UNSPECIFIED,
		dao.Status_STATUS_ACTIVE,
		dao.Status_STATUS_INACTIVE,
	} {
		v := good
		c := &dao.PersonCreate{Nickname: "ali", Status: &v}
		if err := c.Validate(); err != nil {
			t.Errorf("status=%v should pass defined_only, got: %v", good, err)
		}
	}
}

// TestPersonUpdateByName_Validate_NicknamePreciseBoundary verifies the exact
// boundary of max_len=10: len==10 passes, len==11 fails.
// Note: the min_len=1 branch inside the zero-value guard is unreachable for
// optional fields — a non-empty string always has len>=1, so min_len<1 can
// never trigger after the "" guard. This is a known code-generation artefact.
func TestPersonUpdateByName_Validate_NicknamePreciseBoundary(t *testing.T) {
	t.Parallel()

	exact := "1234567890" // len==10, at boundary — must pass
	u := &dao.PersonUpdateByName{Name: "Alice", Nickname: &exact}
	if err := u.Validate(); err != nil {
		t.Errorf("nickname len=10 should pass max_len=10, got: %v", err)
	}

	over := "12345678901" // len==11, one over boundary — must fail
	u2 := &dao.PersonUpdateByName{Name: "Alice", Nickname: &over}
	assertVE(t, u2.Validate(), "nickname", "max_len")
}

// TestPersonCreate_NicknamePreciseBoundary verifies the exact boundary of
// max_len=10 for the required (non-optional) Nickname field.
// Unlike optional fields, the required Nickname has no zero-value guard, so
// min_len=1 is reachable (empty string triggers it via the noZeroGuard path).
func TestPersonCreate_NicknamePreciseBoundary(t *testing.T) {
	t.Parallel()

	c := &dao.PersonCreate{Nickname: "1234567890"} // len==10 — must pass
	if err := c.Validate(); err != nil {
		t.Errorf("nickname len=10 should pass max_len=10, got: %v", err)
	}

	c2 := &dao.PersonCreate{Nickname: "12345678901"} // len==11 — must fail
	assertVE(t, c2.Validate(), "nickname", "max_len")
}

// strPtr returns a pointer to s, used to construct optional string fields.
func strPtr(s string) *string { return &s }

func TestPersonUpdateByName_Validate_OptionalAgeGte(t *testing.T) {
	t.Parallel()

	// age<0 must trigger gte (symmetric with lte already tested).
	bad := int32(-1)
	u := &dao.PersonUpdateByName{Name: "Alice", Age: &bad}
	assertVE(t, u.Validate(), "age", "gte")
}

func TestPersonUpdateByName_Validate_OptionalStatusAllValidValues(t *testing.T) {
	t.Parallel()

	// Exhaustive check: all defined Status values must pass defined_only.
	for _, good := range []dao.Status{
		dao.Status_STATUS_UNSPECIFIED,
		dao.Status_STATUS_ACTIVE,
		dao.Status_STATUS_INACTIVE,
	} {
		v := good
		u := &dao.PersonUpdateByName{Name: "Alice", Status: &v}
		if err := u.Validate(); err != nil {
			t.Errorf("status=%v should pass defined_only, got: %v", good, err)
		}
	}
}

// TestNameMaxLenInnerBoundary verifies that name at exactly max_len=100 passes
// for both PersonUpdateByName (non-optional) and PersonCreate (optional).
func TestNameMaxLenInnerBoundary(t *testing.T) {
	t.Parallel()

	name100 := string(make([]byte, 100))

	u := &dao.PersonUpdateByName{Name: name100}
	if err := u.Validate(); err != nil {
		t.Errorf("PersonUpdateByName name len=100 should pass max_len=100, got: %v", err)
	}

	c := &dao.PersonCreate{Nickname: "ali", Name: strPtr(name100)}
	if err := c.Validate(); err != nil {
		t.Errorf("PersonCreate name len=100 should pass max_len=100, got: %v", err)
	}
}

// --- T3: ignore_fields and required_fields boundary tests -------------------

// TestPersonUpdateByName_IgnoreFields_NotPresent verifies that fields listed in
// ignore_fields (address, scores, tags, avatar, fingerprint) are absent from
// PersonUpdateByName — the struct does not have these fields at all, so their
// validate constraints are never evaluated.
func TestPersonUpdateByName_IgnoreFields_NotPresent(t *testing.T) {
	t.Parallel()

	// A fully valid PersonUpdateByName with all present fields set correctly
	// must pass Validate() — no ignored-field constraints can fire.
	age := int32(30)
	u := &dao.PersonUpdateByName{Name: "Alice", Age: &age}
	if err := u.Validate(); err != nil {
		t.Errorf("PersonUpdateByName with valid fields should pass Validate(), got: %v", err)
	}

	// Compile-time proof: the following fields do not exist on PersonUpdateByName.
	// If ignore_fields were not applied, these would be present and their
	// constraints (required, min_items, etc.) could fire on zero values.
	// Uncommenting any line below would cause a compile error:
	//   _ = u.Address
	//   _ = u.Scores
	//   _ = u.Tags
	//   _ = u.Avatar
}

// TestPersonCreate_IgnoreFields_NotPresent verifies that fields listed in
// ignore_fields for PersonCreate (created_at, address, scores, tags, avatar,
// fingerprint) are absent from the struct — their constraints never fire.
func TestPersonCreate_IgnoreFields_NotPresent(t *testing.T) {
	t.Parallel()

	// A minimal valid PersonCreate (only required nickname set) must pass
	// Validate() — no ignored-field constraints (e.g. avatar required,
	// scores min_items) can fire.
	c := &dao.PersonCreate{Nickname: "ali"}
	if err := c.Validate(); err != nil {
		t.Errorf("PersonCreate with only required fields should pass Validate(), got: %v", err)
	}

	// Compile-time proof: ignored fields do not exist on PersonCreate.
	// Uncommenting any line below would cause a compile error:
	//   _ = c.Address
	//   _ = c.Scores
	//   _ = c.Tags
	//   _ = c.Avatar
	//   _ = c.CreatedAt
}

// TestPersonCreate_RequiredFields_NonOptionalType verifies that required_fields
// entries are generated as non-pointer (non-optional) types in PersonCreate,
// meaning the zero value is a valid Go value but still subject to constraints.
func TestPersonCreate_RequiredFields_NonOptionalType(t *testing.T) {
	t.Parallel()

	// nickname is in required_fields → generated as string (not *string).
	// Zero value "" triggers min_len=1 (no zero-value guard for non-optional fields).
	c := &dao.PersonCreate{Nickname: ""}
	assertVE(t, c.Validate(), "nickname", "min_len")

	// Non-zero value satisfying constraints must pass.
	c2 := &dao.PersonCreate{Nickname: "ali"}
	if err := c2.Validate(); err != nil {
		t.Errorf("required nickname='ali' should pass Validate(), got: %v", err)
	}
}

// TestPersonCreate_RequiredFields_OptionalFieldsStillOptional verifies that
// fields NOT in required_fields remain optional (*T) in PersonCreate, and their
// constraints are skipped when nil.
func TestPersonCreate_RequiredFields_OptionalFieldsStillOptional(t *testing.T) {
	t.Parallel()

	// name is NOT in required_fields → generated as *string in PersonCreate.
	// nil must skip all name constraints (min_len, max_len).
	c := &dao.PersonCreate{Nickname: "ali", Name: nil}
	if err := c.Validate(); err != nil {
		t.Errorf("nil optional name should skip constraints, got: %v", err)
	}

	// When set to a violating value, the constraint must fire.
	c2 := &dao.PersonCreate{Nickname: "ali", Name: strPtr(string(make([]byte, 101)))}
	assertVE(t, c2.Validate(), "name", "max_len")
}

// TestPersonUpdateByName_Validate_AgePreciseBoundary verifies the exact boundary
// values of age constraints (gte=0, lte=150): at-boundary values must pass.
func TestPersonUpdateByName_Validate_AgePreciseBoundary(t *testing.T) {
	t.Parallel()

	zero := int32(0)
	u := &dao.PersonUpdateByName{Name: "Alice", Age: &zero}
	if err := u.Validate(); err != nil {
		t.Errorf("age=0 should pass gte=0, got: %v", err)
	}

	max := int32(150)
	u2 := &dao.PersonUpdateByName{Name: "Alice", Age: &max}
	if err := u2.Validate(); err != nil {
		t.Errorf("age=150 should pass lte=150, got: %v", err)
	}
}

// TestPersonCreate_Validate_AgePreciseBoundary verifies the exact boundary
// values of age constraints (gte=0, lte=150) for PersonCreate: at-boundary values must pass.
func TestPersonCreate_Validate_AgePreciseBoundary(t *testing.T) {
	t.Parallel()

	zero := int32(0)
	c := &dao.PersonCreate{Nickname: "ali", Age: &zero}
	if err := c.Validate(); err != nil {
		t.Errorf("age=0 should pass gte=0, got: %v", err)
	}

	max := int32(150)
	c2 := &dao.PersonCreate{Nickname: "ali", Age: &max}
	if err := c2.Validate(); err != nil {
		t.Errorf("age=150 should pass lte=150, got: %v", err)
	}
}
