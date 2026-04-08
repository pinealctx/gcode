// Package compat_test — ToMap full coverage and MarshalAppend coverage tests.
package compat_test

import (
	"testing"

	"github.com/pinealctx/gcode/testdata/compat/dao"
)

// TestPersonUpdateByNameToMapAllFields verifies ToMap includes all non-nil optional fields.
func TestPersonUpdateByNameToMapAllFields(t *testing.T) {
	t.Parallel()

	age := int32(30)
	active := true
	status := dao.Status_STATUS_ACTIVE
	rating := float32(4.5)
	createdAt := int64(1700000000)
	nickname := "ali"
	level := int32(5)
	verified := true
	score := float32(9.9)
	updatedAt := int64(1700000001)
	prevStatus := dao.Status_STATUS_INACTIVE
	email := "alice@example.com"
	role := "admin"
	typeId := int32(1)

	u := &dao.PersonUpdateByName{
		Name:       "Alice",
		Age:        &age,
		Active:     &active,
		Status:     &status,
		Rating:     &rating,
		CreatedAt:  &createdAt,
		Nickname:   &nickname,
		Level:      &level,
		Verified:   &verified,
		Score:      &score,
		UpdatedAt:  &updatedAt,
		PrevStatus: &prevStatus,
		Email:      &email,
		Role:       &role,
		TypeId:     &typeId,
	}

	m := u.ToMap()

	// condition field must NOT be in map
	if _, ok := m["name"]; ok {
		t.Error("condition field 'name' must not appear in ToMap()")
	}

	// all optional fields must be in map
	for _, key := range []string{"age", "active", "status", "rating", "created_ts",
		"nickname", "level", "verified", "score", "updated_at", "prev_status",
		"email", "role", "type_id"} {
		if _, ok := m[key]; !ok {
			t.Errorf("field %q should be in ToMap()", key)
		}
	}
}

// TestPersonMarshalAppendNilOptionals verifies MarshalAppend with nil optional fields.
// This covers the nil-check branches in MarshalAppend.
func TestPersonMarshalAppendNilOptionals(t *testing.T) {
	t.Parallel()

	// Person with all optional fields nil — only required fields set.
	p := &dao.Person{
		Name:      "Alice",
		Age:       30,
		Active:    true,
		Status:    dao.Status_STATUS_ACTIVE,
		Address:   &dao.Address{Street: "Main St", City: "Springfield"},
		Scores:    []int32{1, 2, 3},
		Tags:      []string{"go"},
		Rating:    4.5,
		CreatedAt: 1700000000,
		Avatar:    []byte{0x01},
		// All optional fields nil: Nickname, Level, Verified, Score, UpdatedAt, PrevStatus, Fingerprint
	}

	wire, err := p.MarshalBinary()
	if err != nil {
		t.Fatalf("MarshalBinary: %v", err)
	}

	var got dao.Person
	if err := got.UnmarshalBinary(wire); err != nil {
		t.Fatalf("UnmarshalBinary: %v", err)
	}

	if got.Nickname != nil {
		t.Errorf("Nickname: got %v, want nil", got.Nickname)
	}
	if got.Level != nil {
		t.Errorf("Level: got %v, want nil", got.Level)
	}
}

// TestPersonMarshalAppendAllOptionals verifies MarshalAppend with all optional fields set.
func TestPersonMarshalAppendAllOptionals(t *testing.T) {
	t.Parallel()

	p := populatedDaoWithOptionals()
	wire, err := p.MarshalBinary()
	if err != nil {
		t.Fatalf("MarshalBinary: %v", err)
	}

	var got dao.Person
	if err := got.UnmarshalBinary(wire); err != nil {
		t.Fatalf("UnmarshalBinary: %v", err)
	}

	if got.Nickname == nil || *got.Nickname != *p.Nickname {
		t.Errorf("Nickname: got %v, want %v", got.Nickname, p.Nickname)
	}
	if got.Level == nil || *got.Level != *p.Level {
		t.Errorf("Level: got %v, want %v", got.Level, p.Level)
	}
}

// TestPersonSizeNilOptionals verifies Size() equals len(MarshalBinary()) with nil optional fields.
func TestPersonSizeNilOptionals(t *testing.T) {
	t.Parallel()

	p := &dao.Person{Name: "Alice", Age: 30}
	wire, err := p.MarshalBinary()
	if err != nil {
		t.Fatalf("MarshalBinary: %v", err)
	}
	if p.Size() != len(wire) {
		t.Errorf("Size() = %d, want %d (len of wire)", p.Size(), len(wire))
	}
}

// TestPersonSizeAllOptionals verifies Size() equals len(MarshalBinary()) with all optional fields set.
func TestPersonSizeAllOptionals(t *testing.T) {
	t.Parallel()

	p := populatedDaoWithOptionals()
	wire, err := p.MarshalBinary()
	if err != nil {
		t.Fatalf("MarshalBinary: %v", err)
	}
	if p.Size() != len(wire) {
		t.Errorf("Size() = %d, want %d (len of wire)", p.Size(), len(wire))
	}
}

// TestAddressSizeNonNil verifies Address.Size() equals len(MarshalBinary()).
func TestAddressSizeNonNil(t *testing.T) {
	t.Parallel()

	a := &dao.Address{Street: "Main St", City: "Springfield"}
	wire, err := a.MarshalBinary()
	if err != nil {
		t.Fatalf("MarshalBinary: %v", err)
	}
	if a.Size() != len(wire) {
		t.Errorf("Size() = %d, want %d (len of wire)", a.Size(), len(wire))
	}
}
