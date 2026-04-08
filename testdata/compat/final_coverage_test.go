// Package compat_test — final coverage push tests.
// Targets Address.unmarshalFrom, PersonCreate.unmarshalFrom, and person_service unmarshalFrom.
package compat_test

import (
	"errors"
	"testing"

	"github.com/pinealctx/gcode/runtime"
	"github.com/pinealctx/gcode/testdata/compat/dao"
)

// --- Address unmarshalFrom coverage ---

// TestAddressFullRoundTrip exercises all Address fields through unmarshalFrom.
func TestAddressFullRoundTrip(t *testing.T) {
	t.Parallel()

	orig := &dao.Address{Street: "42 Elm St", City: "Shelbyville"}
	wire, err := orig.MarshalBinary()
	if err != nil {
		t.Fatalf("MarshalBinary: %v", err)
	}

	var got dao.Address
	if err := got.UnmarshalBinary(wire); err != nil {
		t.Fatalf("UnmarshalBinary: %v", err)
	}
	if got.Street != orig.Street || got.City != orig.City {
		t.Errorf("got %+v, want %+v", got, orig)
	}
}

// TestAddressLenientDuplicate exercises the lenient branch in Address.unmarshalFrom.
func TestAddressLenientDuplicate(t *testing.T) {
	t.Parallel()

	orig := &dao.Address{Street: "Main St", City: "Springfield"}
	wire, err := orig.MarshalBinary()
	if err != nil {
		t.Fatalf("MarshalBinary: %v", err)
	}

	// Duplicate all fields
	dup := duplicateWire(wire)

	var got dao.Address
	if err := got.UnmarshalBinaryLenient(dup); err != nil {
		t.Fatalf("UnmarshalBinaryLenient: %v", err)
	}
	if got.Street != "Main St" || got.City != "Springfield" {
		t.Errorf("got %+v, want {Main St, Springfield}", got)
	}
}

// --- PersonCreate more unmarshal coverage ---

// TestPersonCreateAllFieldsLenient exercises PersonCreate.unmarshalFrom in lenient mode.
func TestPersonCreateAllFieldsLenient(t *testing.T) {
	t.Parallel()

	name := "Alice"
	age := int32(30)
	orig := &dao.PersonCreate{
		Name:     &name,
		Age:      &age,
		Nickname: "ali",
	}
	wire, err := orig.MarshalBinary()
	if err != nil {
		t.Fatalf("MarshalBinary: %v", err)
	}

	dup := duplicateWire(wire)

	var got dao.PersonCreate
	if err := got.UnmarshalBinaryLenient(dup); err != nil {
		t.Fatalf("UnmarshalBinaryLenient: %v", err)
	}
	if got.Nickname != "ali" {
		t.Errorf("Nickname: got %q, want %q", got.Nickname, "ali")
	}
}

// TestPersonCreateNicknameTruncated covers the truncated path for Nickname field.
func TestPersonCreateNicknameTruncated(t *testing.T) {
	t.Parallel()

	wire := buildTruncatedField(6) // nickname (field 6 in PersonCreate)
	var p dao.PersonCreate
	if err := p.UnmarshalBinary(wire); err == nil {
		t.Fatal("expected error for truncated nickname, got nil")
	}
}

// TestPersonCreateNicknameWireTypeError covers the wire type error for Nickname.
func TestPersonCreateNicknameWireTypeError(t *testing.T) {
	t.Parallel()

	var wire []byte
	wire = runtime.AppendTag(wire, 6, runtime.WireVarint)
	wire = runtime.AppendVarint(wire, 42)
	var p dao.PersonCreate
	err := p.UnmarshalBinary(wire)
	if err == nil {
		t.Fatal("expected error for wrong wire type, got nil")
	}
	if !errors.Is(err, runtime.ErrWireType) {
		t.Errorf("expected ErrWireType, got: %v", err)
	}
}

// --- PersonUpdateByName more unmarshal coverage ---

// TestPersonUpdateByNameAllFieldsLenient exercises PersonUpdateByName.unmarshalFrom in lenient mode.
func TestPersonUpdateByNameAllFieldsLenient(t *testing.T) {
	t.Parallel()

	age := int32(25)
	orig := &dao.PersonUpdateByName{
		Name: "Bob",
		Age:  &age,
	}
	wire, err := orig.MarshalBinary()
	if err != nil {
		t.Fatalf("MarshalBinary: %v", err)
	}

	dup := duplicateWire(wire)

	var got dao.PersonUpdateByName
	if err := got.UnmarshalBinaryLenient(dup); err != nil {
		t.Fatalf("UnmarshalBinaryLenient: %v", err)
	}
	if got.Name != "Bob" {
		t.Errorf("Name: got %q, want %q", got.Name, "Bob")
	}
}

// TestPersonUpdateByNameNicknameTruncated covers the truncated path for Nickname field.
func TestPersonUpdateByNameNicknameTruncated(t *testing.T) {
	t.Parallel()

	wire := buildTruncatedField(7) // nickname (field 7 in PersonUpdateByName)
	var p dao.PersonUpdateByName
	if err := p.UnmarshalBinary(wire); err == nil {
		t.Fatal("expected error for truncated nickname, got nil")
	}
}

// --- person_service types more coverage ---

// TestGetPersonResponseAgeDuplicateField covers duplicate age field in GetPersonResponse.
func TestGetPersonResponseAgeDuplicateField(t *testing.T) {
	t.Parallel()

	wire := buildDupField(2, runtime.WireVarint, func(b []byte) []byte {
		return runtime.AppendVarint(b, 30)
	})
	var g dao.GetPersonResponse
	if err := g.UnmarshalBinary(wire); err == nil {
		t.Fatal("expected ErrDuplicateField, got nil")
	}
}

// TestGetPersonResponseAgeWireTypeError covers wire type error for age in GetPersonResponse.
func TestGetPersonResponseAgeWireTypeError(t *testing.T) {
	t.Parallel()

	var wire []byte
	wire = runtime.AppendTag(wire, 2, runtime.WireFixed32)
	wire = runtime.AppendFixed32(wire, 30)
	var g dao.GetPersonResponse
	err := g.UnmarshalBinary(wire)
	if err == nil {
		t.Fatal("expected error for wrong wire type, got nil")
	}
	if !errors.Is(err, runtime.ErrWireType) {
		t.Errorf("expected ErrWireType, got: %v", err)
	}
}

// TestUpdatePersonResponseOkDuplicateField covers duplicate ok field in UpdatePersonResponse.
func TestUpdatePersonResponseOkDuplicateField(t *testing.T) {
	t.Parallel()

	wire := buildDupField(1, runtime.WireVarint, func(b []byte) []byte {
		return runtime.AppendVarint(b, 1)
	})
	var u dao.UpdatePersonResponse
	if err := u.UnmarshalBinary(wire); err == nil {
		t.Fatal("expected ErrDuplicateField, got nil")
	}
}

// TestUpdatePersonResponseOkWireTypeError covers wire type error for ok in UpdatePersonResponse.
func TestUpdatePersonResponseOkWireTypeError(t *testing.T) {
	t.Parallel()

	var wire []byte
	wire = runtime.AppendTag(wire, 1, runtime.WireFixed32)
	wire = runtime.AppendFixed32(wire, 1)
	var u dao.UpdatePersonResponse
	err := u.UnmarshalBinary(wire)
	if err == nil {
		t.Fatal("expected error for wrong wire type, got nil")
	}
	if !errors.Is(err, runtime.ErrWireType) {
		t.Errorf("expected ErrWireType, got: %v", err)
	}
}

// TestDeletePersonResponseOkDuplicateField covers duplicate ok field in DeletePersonResponse.
func TestDeletePersonResponseOkDuplicateField(t *testing.T) {
	t.Parallel()

	wire := buildDupField(1, runtime.WireVarint, func(b []byte) []byte {
		return runtime.AppendVarint(b, 1)
	})
	var d dao.DeletePersonResponse
	if err := d.UnmarshalBinary(wire); err == nil {
		t.Fatal("expected ErrDuplicateField, got nil")
	}
}

// TestDeletePersonResponseOkWireTypeError covers wire type error for ok in DeletePersonResponse.
func TestDeletePersonResponseOkWireTypeError(t *testing.T) {
	t.Parallel()

	var wire []byte
	wire = runtime.AppendTag(wire, 1, runtime.WireFixed32)
	wire = runtime.AppendFixed32(wire, 1)
	var d dao.DeletePersonResponse
	err := d.UnmarshalBinary(wire)
	if err == nil {
		t.Fatal("expected error for wrong wire type, got nil")
	}
	if !errors.Is(err, runtime.ErrWireType) {
		t.Errorf("expected ErrWireType, got: %v", err)
	}
}
