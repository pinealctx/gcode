// Package compat_test — wire and validate tests for person_service generated types.
// Covers UnmarshalBinaryLenient, duplicate field handling, and Validate() for all
// RPC request/response types.
package compat_test

import (
	"testing"

	"github.com/pinealctx/gcode/testdata/compat/dao"
)

// --- UnmarshalBinaryLenient for person_service types ---

// TestCreatePersonResponseLenient verifies UnmarshalBinaryLenient accepts duplicate fields.
func TestCreatePersonResponseLenient(t *testing.T) {
	t.Parallel()

	orig := &dao.CreatePersonResponse{Id: "uuid-123"}
	wire, err := orig.MarshalBinary()
	if err != nil {
		t.Fatalf("MarshalBinary: %v", err)
	}

	// Duplicate the wire bytes to simulate duplicate field.
	dup := duplicateWire(wire)

	var got dao.CreatePersonResponse
	if err := got.UnmarshalBinaryLenient(dup); err != nil {
		t.Fatalf("UnmarshalBinaryLenient: %v", err)
	}
	if got.Id != "uuid-123" {
		t.Errorf("Id: got %q, want %q", got.Id, "uuid-123")
	}
}

// TestGetPersonRequestLenient verifies UnmarshalBinaryLenient accepts duplicate fields.
func TestGetPersonRequestLenient(t *testing.T) {
	t.Parallel()

	orig := &dao.GetPersonRequest{Id: "id-42"}
	wire, err := orig.MarshalBinary()
	if err != nil {
		t.Fatalf("MarshalBinary: %v", err)
	}

	dup := duplicateWire(wire)

	var got dao.GetPersonRequest
	if err := got.UnmarshalBinaryLenient(dup); err != nil {
		t.Fatalf("UnmarshalBinaryLenient: %v", err)
	}
	if got.Id != "id-42" {
		t.Errorf("Id: got %q, want %q", got.Id, "id-42")
	}
}

// TestGetPersonResponseLenient verifies UnmarshalBinaryLenient accepts duplicate fields.
func TestGetPersonResponseLenient(t *testing.T) {
	t.Parallel()

	orig := &dao.GetPersonResponse{Name: "Alice", Age: 30}
	wire, err := orig.MarshalBinary()
	if err != nil {
		t.Fatalf("MarshalBinary: %v", err)
	}

	dup := duplicateWire(wire)

	var got dao.GetPersonResponse
	if err := got.UnmarshalBinaryLenient(dup); err != nil {
		t.Fatalf("UnmarshalBinaryLenient: %v", err)
	}
	if got.Name != "Alice" {
		t.Errorf("Name: got %q, want %q", got.Name, "Alice")
	}
}

// TestUpdatePersonResponseLenient verifies UnmarshalBinaryLenient accepts duplicate fields.
func TestUpdatePersonResponseLenient(t *testing.T) {
	t.Parallel()

	orig := &dao.UpdatePersonResponse{Ok: true}
	wire, err := orig.MarshalBinary()
	if err != nil {
		t.Fatalf("MarshalBinary: %v", err)
	}

	dup := duplicateWire(wire)

	var got dao.UpdatePersonResponse
	if err := got.UnmarshalBinaryLenient(dup); err != nil {
		t.Fatalf("UnmarshalBinaryLenient: %v", err)
	}
	if !got.Ok {
		t.Errorf("Ok: got false, want true")
	}
}

// TestDeletePersonRequestLenient verifies UnmarshalBinaryLenient accepts duplicate fields.
func TestDeletePersonRequestLenient(t *testing.T) {
	t.Parallel()

	orig := &dao.DeletePersonRequest{Id: "id-99"}
	wire, err := orig.MarshalBinary()
	if err != nil {
		t.Fatalf("MarshalBinary: %v", err)
	}

	dup := duplicateWire(wire)

	var got dao.DeletePersonRequest
	if err := got.UnmarshalBinaryLenient(dup); err != nil {
		t.Fatalf("UnmarshalBinaryLenient: %v", err)
	}
	if got.Id != "id-99" {
		t.Errorf("Id: got %q, want %q", got.Id, "id-99")
	}
}

// TestDeletePersonResponseLenient verifies UnmarshalBinaryLenient accepts duplicate fields.
func TestDeletePersonResponseLenient(t *testing.T) {
	t.Parallel()

	orig := &dao.DeletePersonResponse{Ok: true}
	wire, err := orig.MarshalBinary()
	if err != nil {
		t.Fatalf("MarshalBinary: %v", err)
	}

	dup := duplicateWire(wire)

	var got dao.DeletePersonResponse
	if err := got.UnmarshalBinaryLenient(dup); err != nil {
		t.Fatalf("UnmarshalBinaryLenient: %v", err)
	}
	if !got.Ok {
		t.Errorf("Ok: got false, want true")
	}
}

// --- Validate() for person_service types ---

// TestCreatePersonResponseValidate verifies Validate() always returns nil.
func TestCreatePersonResponseValidate(t *testing.T) {
	t.Parallel()
	c := &dao.CreatePersonResponse{Id: "uuid-123"}
	if err := c.Validate(); err != nil {
		t.Errorf("CreatePersonResponse.Validate() = %v, want nil", err)
	}
}

// TestGetPersonResponseValidate verifies Validate() always returns nil.
func TestGetPersonResponseValidate(t *testing.T) {
	t.Parallel()
	g := &dao.GetPersonResponse{Name: "Alice", Age: 30}
	if err := g.Validate(); err != nil {
		t.Errorf("GetPersonResponse.Validate() = %v, want nil", err)
	}
}

// TestUpdatePersonResponseValidate verifies Validate() always returns nil.
func TestUpdatePersonResponseValidate(t *testing.T) {
	t.Parallel()
	u := &dao.UpdatePersonResponse{Ok: true}
	if err := u.Validate(); err != nil {
		t.Errorf("UpdatePersonResponse.Validate() = %v, want nil", err)
	}
}

// TestDeletePersonResponseValidate verifies Validate() always returns nil.
func TestDeletePersonResponseValidate(t *testing.T) {
	t.Parallel()
	d := &dao.DeletePersonResponse{Ok: false}
	if err := d.Validate(); err != nil {
		t.Errorf("DeletePersonResponse.Validate() = %v, want nil", err)
	}
}

// TestGetPersonRequestValidate_MaxLen verifies max_len constraint on id.
func TestGetPersonRequestValidate_MaxLen(t *testing.T) {
	t.Parallel()

	// fail: id exceeds max_len=64
	g := &dao.GetPersonRequest{Id: string(make([]byte, 65))}
	assertVE(t, g.Validate(), "id", "max_len")

	// pass: id at boundary
	g2 := &dao.GetPersonRequest{Id: string(make([]byte, 64))}
	if err := g2.Validate(); err != nil {
		t.Errorf("id len=64 should pass max_len=64, got: %v", err)
	}

	// pass: empty id skips check (zero-value guard)
	g3 := &dao.GetPersonRequest{Id: ""}
	if err := g3.Validate(); err != nil {
		t.Errorf("empty id should skip check, got: %v", err)
	}
}

// TestDeletePersonRequestValidate_MaxLen verifies max_len constraint on id.
func TestDeletePersonRequestValidate_MaxLen(t *testing.T) {
	t.Parallel()

	// fail: id exceeds max_len=64
	d := &dao.DeletePersonRequest{Id: string(make([]byte, 65))}
	assertVE(t, d.Validate(), "id", "max_len")

	// pass: id at boundary
	d2 := &dao.DeletePersonRequest{Id: string(make([]byte, 64))}
	if err := d2.Validate(); err != nil {
		t.Errorf("id len=64 should pass max_len=64, got: %v", err)
	}

	// pass: empty id skips check (zero-value guard)
	d3 := &dao.DeletePersonRequest{Id: ""}
	if err := d3.Validate(); err != nil {
		t.Errorf("empty id should skip check, got: %v", err)
	}
}

// --- Person.UnmarshalBinaryLenient ---

// TestPersonUnmarshalBinaryLenient verifies that Person.UnmarshalBinaryLenient
// accepts duplicate singular fields (last-one-wins).
func TestPersonUnmarshalBinaryLenient(t *testing.T) {
	t.Parallel()

	orig := populatedDao()
	wire, err := orig.MarshalBinary()
	if err != nil {
		t.Fatalf("MarshalBinary: %v", err)
	}

	var got dao.Person
	if err := got.UnmarshalBinaryLenient(wire); err != nil {
		t.Fatalf("UnmarshalBinaryLenient: %v", err)
	}

	if got.Name != orig.Name {
		t.Errorf("Name: got %q, want %q", got.Name, orig.Name)
	}
}

// TestAllScalarsUnmarshalBinaryLenient verifies AllScalars.UnmarshalBinaryLenient
// accepts duplicate singular fields.
func TestAllScalarsUnmarshalBinaryLenient(t *testing.T) {
	t.Parallel()

	orig := populatedDaoAllScalars()
	wire, err := orig.MarshalBinary()
	if err != nil {
		t.Fatalf("MarshalBinary: %v", err)
	}

	dup := duplicateWire(wire)

	var got dao.AllScalars
	if err := got.UnmarshalBinaryLenient(dup); err != nil {
		t.Fatalf("UnmarshalBinaryLenient: %v", err)
	}
	if got.FSint32 != orig.FSint32 {
		t.Errorf("FSint32: got %d, want %d", got.FSint32, orig.FSint32)
	}
}

// TestAllValidateUnmarshalBinaryLenient verifies AllValidate.UnmarshalBinaryLenient
// accepts duplicate singular fields.
func TestAllValidateUnmarshalBinaryLenient(t *testing.T) {
	t.Parallel()

	status := dao.Status_STATUS_ACTIVE
	orig := &dao.AllValidate{
		UGte:    5,
		OStatus: &status,
	}
	wire, err := orig.MarshalBinary()
	if err != nil {
		t.Fatalf("MarshalBinary: %v", err)
	}

	dup := duplicateWire(wire)

	var got dao.AllValidate
	if err := got.UnmarshalBinaryLenient(dup); err != nil {
		t.Fatalf("UnmarshalBinaryLenient: %v", err)
	}
	if got.UGte != orig.UGte {
		t.Errorf("UGte: got %d, want %d", got.UGte, orig.UGte)
	}
}

// TestAllScalarsUpdateUnmarshalBinaryLenient verifies AllScalarsUpdate.UnmarshalBinaryLenient
// accepts duplicate singular fields.
func TestAllScalarsUpdateUnmarshalBinaryLenient(t *testing.T) {
	t.Parallel()

	orig := &dao.AllScalarsUpdate{FSint32: -5}
	wire, err := orig.MarshalBinary()
	if err != nil {
		t.Fatalf("MarshalBinary: %v", err)
	}

	dup := duplicateWire(wire)

	var got dao.AllScalarsUpdate
	if err := got.UnmarshalBinaryLenient(dup); err != nil {
		t.Fatalf("UnmarshalBinaryLenient: %v", err)
	}
	if got.FSint32 != -5 {
		t.Errorf("FSint32: got %d, want -5", got.FSint32)
	}
}

// TestAllScalarsCreateUnmarshalBinaryLenient verifies AllScalarsCreate.UnmarshalBinaryLenient
// accepts duplicate singular fields.
func TestAllScalarsCreateUnmarshalBinaryLenient(t *testing.T) {
	t.Parallel()

	sint32 := int32(-42)
	orig := &dao.AllScalarsCreate{FSint32: &sint32}
	wire, err := orig.MarshalBinary()
	if err != nil {
		t.Fatalf("MarshalBinary: %v", err)
	}

	dup := duplicateWire(wire)

	var got dao.AllScalarsCreate
	if err := got.UnmarshalBinaryLenient(dup); err != nil {
		t.Fatalf("UnmarshalBinaryLenient: %v", err)
	}
	if got.FSint32 == nil || *got.FSint32 != -42 {
		t.Errorf("FSint32: got %v, want -42", got.FSint32)
	}
}
