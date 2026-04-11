// Package compat_test — additional wire tests to boost dao/ coverage.
// Covers PersonCreate/PersonUpdateByName unmarshal paths, Address.UnmarshalBinaryLenient,
// TableName methods, and error paths in unmarshalFrom.
package compat_test

import (
	"errors"
	"testing"

	"github.com/pinealctx/gcode/runtime"
	"github.com/pinealctx/gcode/testdata/compat/dao"
)

// --- TableName coverage ---

func TestTableNames(t *testing.T) {
	t.Parallel()

	p := dao.Person{}
	if p.TableName() != "persons" {
		t.Errorf("Person.TableName() = %q, want %q", p.TableName(), "persons")
	}
	as := dao.AllScalars{}
	if as.TableName() != "all_scalars" {
		t.Errorf("AllScalars.TableName() = %q, want %q", as.TableName(), "all_scalars")
	}
	ac := dao.AllScalarsCreate{}
	if ac.TableName() != "all_scalars" {
		t.Errorf("AllScalarsCreate.TableName() = %q, want %q", ac.TableName(), "all_scalars")
	}
	pc := dao.PersonCreate{}
	if pc.TableName() != "persons" {
		t.Errorf("PersonCreate.TableName() = %q, want %q", pc.TableName(), "persons")
	}
}

// --- Address.UnmarshalBinaryLenient ---

func TestAddressUnmarshalBinaryLenient(t *testing.T) {
	t.Parallel()

	orig := &dao.Address{Street: "Main St", City: "Springfield"}
	wire, err := orig.MarshalBinary()
	if err != nil {
		t.Fatalf("MarshalBinary: %v", err)
	}

	dup := duplicateWire(wire)

	var got dao.Address
	if err := got.UnmarshalBinaryLenient(dup); err != nil {
		t.Fatalf("UnmarshalBinaryLenient: %v", err)
	}
	if got.Street != "Main St" {
		t.Errorf("Street: got %q, want %q", got.Street, "Main St")
	}
}

// --- PersonCreate unmarshal coverage ---

// TestPersonCreateUnmarshalAllFields exercises all fields in PersonCreate.unmarshalFrom.
func TestPersonCreateUnmarshalAllFields(t *testing.T) {
	t.Parallel()

	name := "Alice"
	age := int32(30)
	active := true
	status := dao.Status_STATUS_ACTIVE
	rating := float32(4.5)
	level := int32(5)
	verified := true
	score := float32(9.9)
	updatedAt := int64(1700000001)
	prevStatus := dao.Status_STATUS_INACTIVE
	email := "alice@example.com"
	role := "admin"
	typeId := int32(1)

	orig := &dao.PersonCreate{
		Name:       &name,
		Age:        &age,
		Active:     &active,
		Status:     &status,
		Rating:     &rating,
		Nickname:   "ali",
		Level:      &level,
		Verified:   &verified,
		Score:      &score,
		UpdatedAt:  &updatedAt,
		PrevStatus: &prevStatus,
		Email:      &email,
		Role:       &role,
		TypeId:     &typeId,
	}

	wire, err := orig.MarshalBinary()
	if err != nil {
		t.Fatalf("MarshalBinary: %v", err)
	}

	var got dao.PersonCreate
	if err := got.UnmarshalBinary(wire); err != nil {
		t.Fatalf("UnmarshalBinary: %v", err)
	}

	if got.Nickname != "ali" {
		t.Errorf("Nickname: got %q, want %q", got.Nickname, "ali")
	}
	if got.Name == nil || *got.Name != name {
		t.Errorf("Name: got %v, want %q", got.Name, name)
	}
	if got.Age == nil || *got.Age != age {
		t.Errorf("Age: got %v, want %d", got.Age, age)
	}
	if got.Active == nil || *got.Active != active {
		t.Errorf("Active: got %v, want %v", got.Active, active)
	}
	if got.Status == nil || *got.Status != status {
		t.Errorf("Status: got %v, want %v", got.Status, status)
	}
	if got.Rating == nil || *got.Rating != rating {
		t.Errorf("Rating: got %v, want %v", got.Rating, rating)
	}
	if got.Level == nil || *got.Level != level {
		t.Errorf("Level: got %v, want %d", got.Level, level)
	}
	if got.Verified == nil || *got.Verified != verified {
		t.Errorf("Verified: got %v, want %v", got.Verified, verified)
	}
	if got.Score == nil || *got.Score != score {
		t.Errorf("Score: got %v, want %v", got.Score, score)
	}
	if got.UpdatedAt == nil || *got.UpdatedAt != updatedAt {
		t.Errorf("UpdatedAt: got %v, want %d", got.UpdatedAt, updatedAt)
	}
	if got.PrevStatus == nil || *got.PrevStatus != prevStatus {
		t.Errorf("PrevStatus: got %v, want %v", got.PrevStatus, prevStatus)
	}
	if got.Email == nil || *got.Email != email {
		t.Errorf("Email: got %v, want %q", got.Email, email)
	}
	if got.Role == nil || *got.Role != role {
		t.Errorf("Role: got %v, want %q", got.Role, role)
	}
	if got.TypeId == nil || *got.TypeId != typeId {
		t.Errorf("TypeId: got %v, want %d", got.TypeId, typeId)
	}
}

// TestPersonCreateLenient verifies PersonCreate.UnmarshalBinaryLenient accepts duplicates.
func TestPersonCreateLenient(t *testing.T) {
	t.Parallel()

	orig := &dao.PersonCreate{Nickname: "ali"}
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

// --- PersonUpdateByName unmarshal coverage ---

// TestPersonUpdateByNameUnmarshalAllFields exercises all fields in PersonUpdateByName.unmarshalFrom.
func TestPersonUpdateByNameUnmarshalAllFields(t *testing.T) {
	t.Parallel()

	age := int32(25)
	active := false
	status := dao.Status_STATUS_INACTIVE
	rating := float32(3.0)
	createdAt := int64(1700000000)
	nickname := "bob"
	level := int32(2)
	verified := false
	score := float32(5.5)
	updatedAt := int64(1700000002)
	prevStatus := dao.Status_STATUS_ACTIVE
	email := "bob@example.com"
	role := "user"
	typeId := int32(2)

	orig := &dao.PersonUpdateByName{
		Name:       "Bob",
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

	wire, err := orig.MarshalBinary()
	if err != nil {
		t.Fatalf("MarshalBinary: %v", err)
	}

	var got dao.PersonUpdateByName
	if err := got.UnmarshalBinary(wire); err != nil {
		t.Fatalf("UnmarshalBinary: %v", err)
	}

	if got.Name != "Bob" {
		t.Errorf("Name: got %q, want %q", got.Name, "Bob")
	}
	if got.Age == nil || *got.Age != age {
		t.Errorf("Age: got %v, want %d", got.Age, age)
	}
	if got.Active == nil || *got.Active != active {
		t.Errorf("Active: got %v, want %v", got.Active, active)
	}
	if got.Status == nil || *got.Status != status {
		t.Errorf("Status: got %v, want %v", got.Status, status)
	}
	if got.Rating == nil || *got.Rating != rating {
		t.Errorf("Rating: got %v, want %v", got.Rating, rating)
	}
	if got.CreatedAt == nil || *got.CreatedAt != createdAt {
		t.Errorf("CreatedAt: got %v, want %d", got.CreatedAt, createdAt)
	}
	if got.Nickname == nil || *got.Nickname != nickname {
		t.Errorf("Nickname: got %v, want %q", got.Nickname, nickname)
	}
	if got.Level == nil || *got.Level != level {
		t.Errorf("Level: got %v, want %d", got.Level, level)
	}
	if got.Verified == nil || *got.Verified != verified {
		t.Errorf("Verified: got %v, want %v", got.Verified, verified)
	}
	if got.Score == nil || *got.Score != score {
		t.Errorf("Score: got %v, want %v", got.Score, score)
	}
	if got.UpdatedAt == nil || *got.UpdatedAt != updatedAt {
		t.Errorf("UpdatedAt: got %v, want %d", got.UpdatedAt, updatedAt)
	}
	if got.PrevStatus == nil || *got.PrevStatus != prevStatus {
		t.Errorf("PrevStatus: got %v, want %v", got.PrevStatus, prevStatus)
	}
	if got.Email == nil || *got.Email != email {
		t.Errorf("Email: got %v, want %q", got.Email, email)
	}
	if got.Role == nil || *got.Role != role {
		t.Errorf("Role: got %v, want %q", got.Role, role)
	}
	if got.TypeId == nil || *got.TypeId != typeId {
		t.Errorf("TypeId: got %v, want %d", got.TypeId, typeId)
	}
}

// TestPersonUpdateByNameLenient verifies PersonUpdateByName.UnmarshalBinaryLenient accepts duplicates.
func TestPersonUpdateByNameLenient(t *testing.T) {
	t.Parallel()

	orig := &dao.PersonUpdateByName{Name: "Alice"}
	wire, err := orig.MarshalBinary()
	if err != nil {
		t.Fatalf("MarshalBinary: %v", err)
	}

	dup := duplicateWire(wire)

	var got dao.PersonUpdateByName
	if err := got.UnmarshalBinaryLenient(dup); err != nil {
		t.Fatalf("UnmarshalBinaryLenient: %v", err)
	}
	if got.Name != "Alice" {
		t.Errorf("Name: got %q, want %q", got.Name, "Alice")
	}
}

// --- Error paths in unmarshalFrom ---

// TestPersonCreateWireTypeError verifies that wrong wire type returns ErrWireType.
func TestPersonCreateWireTypeError(t *testing.T) {
	t.Parallel()

	// Field 1 in PersonCreate is "name" (string, WireBytes).
	// Encode it as WireVarint instead.
	var wire []byte
	wire = runtime.AppendTag(wire, 1, runtime.WireVarint)
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

// TestPersonUpdateByNameWireTypeError verifies that wrong wire type returns ErrWireType.
func TestPersonUpdateByNameWireTypeError(t *testing.T) {
	t.Parallel()

	// Field 1 in PersonUpdateByName is "name" (string, WireBytes).
	// Encode it as WireVarint instead.
	var wire []byte
	wire = runtime.AppendTag(wire, 1, runtime.WireVarint)
	wire = runtime.AppendVarint(wire, 42)

	var p dao.PersonUpdateByName
	err := p.UnmarshalBinary(wire)
	if err == nil {
		t.Fatal("expected error for wrong wire type, got nil")
	}
	if !errors.Is(err, runtime.ErrWireType) {
		t.Errorf("expected ErrWireType, got: %v", err)
	}
}

// TestAllScalarsWireTypeError verifies that wrong wire type returns ErrWireType.
func TestAllScalarsWireTypeError(t *testing.T) {
	t.Parallel()

	// Field 1 in AllScalars is "f_sint32" (sint32, WireVarint).
	// Encode it as WireFixed32 instead.
	var wire []byte
	wire = runtime.AppendTag(wire, 1, runtime.WireFixed32)
	wire = runtime.AppendFixed32(wire, 42)

	var a dao.AllScalars
	err := a.UnmarshalBinary(wire)
	if err == nil {
		t.Fatal("expected error for wrong wire type, got nil")
	}
	if !errors.Is(err, runtime.ErrWireType) {
		t.Errorf("expected ErrWireType, got: %v", err)
	}
}

// TestAllValidateWireTypeError verifies that wrong wire type returns ErrWireType.
func TestAllValidateWireTypeError(t *testing.T) {
	t.Parallel()

	// Field 1 in AllValidate is "u_gte" (uint32, WireVarint).
	// Encode it as WireFixed32 instead.
	var wire []byte
	wire = runtime.AppendTag(wire, 1, runtime.WireFixed32)
	wire = runtime.AppendFixed32(wire, 42)

	var a dao.AllValidate
	err := a.UnmarshalBinary(wire)
	if err == nil {
		t.Fatal("expected error for wrong wire type, got nil")
	}
	if !errors.Is(err, runtime.ErrWireType) {
		t.Errorf("expected ErrWireType, got: %v", err)
	}
}

// TestAllScalarsUpdateWireTypeError verifies that wrong wire type returns ErrWireType.
func TestAllScalarsUpdateWireTypeError(t *testing.T) {
	t.Parallel()

	// Field 1 in AllScalarsUpdate is "f_sint32" (sint32, WireVarint).
	// Encode it as WireFixed32 instead.
	var wire []byte
	wire = runtime.AppendTag(wire, 1, runtime.WireFixed32)
	wire = runtime.AppendFixed32(wire, 42)

	var a dao.AllScalarsUpdate
	err := a.UnmarshalBinary(wire)
	if err == nil {
		t.Fatal("expected error for wrong wire type, got nil")
	}
	if !errors.Is(err, runtime.ErrWireType) {
		t.Errorf("expected ErrWireType, got: %v", err)
	}
}

// TestAllScalarsCreateWireTypeError verifies that wrong wire type returns ErrWireType.
func TestAllScalarsCreateWireTypeError(t *testing.T) {
	t.Parallel()

	// Field 1 in AllScalarsCreate is "f_sint32" (sint32, WireVarint).
	// Encode it as WireFixed32 instead.
	var wire []byte
	wire = runtime.AppendTag(wire, 1, runtime.WireFixed32)
	wire = runtime.AppendFixed32(wire, 42)

	var a dao.AllScalarsCreate
	err := a.UnmarshalBinary(wire)
	if err == nil {
		t.Fatal("expected error for wrong wire type, got nil")
	}
	if !errors.Is(err, runtime.ErrWireType) {
		t.Errorf("expected ErrWireType, got: %v", err)
	}
}
