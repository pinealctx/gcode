// Package compat_test — wire round-trip tests for AllValidate, AllScalarsUpdate,
// and AllScalarsCreate. These cover the generated marshal/unmarshal paths for
// two-phase generated structs and the AllValidate message.
package compat_test

import (
	"bytes"
	"testing"

	"github.com/pinealctx/gcode/testdata/compat/dao"
)

// --- AllValidate wire round-trip ---

// TestAllValidateRoundTrip verifies MarshalBinary → UnmarshalBinary for AllValidate.
func TestAllValidateRoundTrip(t *testing.T) {
	t.Parallel()

	status := dao.Status_STATUS_ACTIVE
	orig := &dao.AllValidate{
		UGte:    5,
		ULte:    500,
		UIn:     3,
		UNotIn:  7,
		FGt:     1.5,
		DLte:    0.75,
		SIn:     "a",
		SNotIn:  "z",
		IIn:     -1,
		SUri:    "https://example.com",
		OStatus: &status,
		BMinmax: []byte{0xDE, 0xAD},
		RItems:  []int32{0, 1, 100},
	}

	wire, err := orig.MarshalBinary()
	if err != nil {
		t.Fatalf("MarshalBinary: %v", err)
	}

	var got dao.AllValidate
	if err := got.UnmarshalBinary(wire); err != nil {
		t.Fatalf("UnmarshalBinary: %v", err)
	}

	assertAllValidateEqual(t, orig, &got)
}

// TestAllValidateZeroValueRoundTrip verifies zero-value AllValidate encodes to 0 bytes.
func TestAllValidateZeroValueRoundTrip(t *testing.T) {
	t.Parallel()

	var a dao.AllValidate
	wire, err := a.MarshalBinary()
	if err != nil {
		t.Fatalf("MarshalBinary: %v", err)
	}
	if len(wire) != 0 {
		t.Errorf("zero-value AllValidate should encode to 0 bytes, got %d", len(wire))
	}

	var got dao.AllValidate
	if err := got.UnmarshalBinary(wire); err != nil {
		t.Fatalf("UnmarshalBinary empty: %v", err)
	}
}

// TestAllValidateNilOStatusRoundTrip verifies nil OStatus survives round-trip.
func TestAllValidateNilOStatusRoundTrip(t *testing.T) {
	t.Parallel()

	orig := &dao.AllValidate{
		UGte: 1,
		UIn:  1,
		FGt:  0.1,
		IIn:  1,
		// OStatus nil
	}

	wire, err := orig.MarshalBinary()
	if err != nil {
		t.Fatalf("MarshalBinary: %v", err)
	}

	var got dao.AllValidate
	if err := got.UnmarshalBinary(wire); err != nil {
		t.Fatalf("UnmarshalBinary: %v", err)
	}

	if got.OStatus != nil {
		t.Errorf("OStatus: got %v, want nil", got.OStatus)
	}
}

// --- AllScalarsUpdate round-trip ---

// TestAllScalarsUpdateRoundTrip verifies MarshalBinary → UnmarshalBinary for AllScalarsUpdate.
func TestAllScalarsUpdateRoundTrip(t *testing.T) {
	t.Parallel()

	sint64 := int64(-500)
	sfixed32 := int32(-7)
	sfixed64 := int64(-999)
	fdouble := 2.718281828
	fixed32 := uint32(0xDEAD)
	fixed64 := uint64(0xCAFEBABE)
	fuint32 := uint32(12345)
	fuint64 := uint64(9876543210)
	ffloat := float32(1.414)

	orig := &dao.AllScalarsUpdate{
		FSint32:   -42,
		FSint64:   &sint64,
		FSfixed32: &sfixed32,
		FSfixed64: &sfixed64,
		FDouble:   &fdouble,
		FFixed32:  &fixed32,
		FFixed64:  &fixed64,
		FUint32:   &fuint32,
		FUint64:   &fuint64,
		FFloat:    &ffloat,
		FBytes:    []byte{0x01, 0x02, 0x03},
	}

	wire, err := orig.MarshalBinary()
	if err != nil {
		t.Fatalf("MarshalBinary: %v", err)
	}

	var got dao.AllScalarsUpdate
	if err := got.UnmarshalBinary(wire); err != nil {
		t.Fatalf("UnmarshalBinary: %v", err)
	}

	if got.FSint32 != orig.FSint32 {
		t.Errorf("FSint32: got %d, want %d", got.FSint32, orig.FSint32)
	}
	if got.FSint64 == nil || *got.FSint64 != *orig.FSint64 {
		t.Errorf("FSint64: got %v, want %v", got.FSint64, orig.FSint64)
	}
	if got.FSfixed32 == nil || *got.FSfixed32 != *orig.FSfixed32 {
		t.Errorf("FSfixed32: got %v, want %v", got.FSfixed32, orig.FSfixed32)
	}
	if got.FSfixed64 == nil || *got.FSfixed64 != *orig.FSfixed64 {
		t.Errorf("FSfixed64: got %v, want %v", got.FSfixed64, orig.FSfixed64)
	}
	if got.FDouble == nil || *got.FDouble != *orig.FDouble {
		t.Errorf("FDouble: got %v, want %v", got.FDouble, orig.FDouble)
	}
	if got.FFixed32 == nil || *got.FFixed32 != *orig.FFixed32 {
		t.Errorf("FFixed32: got %v, want %v", got.FFixed32, orig.FFixed32)
	}
	if got.FFixed64 == nil || *got.FFixed64 != *orig.FFixed64 {
		t.Errorf("FFixed64: got %v, want %v", got.FFixed64, orig.FFixed64)
	}
	if got.FUint32 == nil || *got.FUint32 != *orig.FUint32 {
		t.Errorf("FUint32: got %v, want %v", got.FUint32, orig.FUint32)
	}
	if got.FUint64 == nil || *got.FUint64 != *orig.FUint64 {
		t.Errorf("FUint64: got %v, want %v", got.FUint64, orig.FUint64)
	}
	if got.FFloat == nil || *got.FFloat != *orig.FFloat {
		t.Errorf("FFloat: got %v, want %v", got.FFloat, orig.FFloat)
	}
	if !bytes.Equal(got.FBytes, orig.FBytes) {
		t.Errorf("FBytes: got %x, want %x", got.FBytes, orig.FBytes)
	}
}

// TestAllScalarsUpdateNilFieldsRoundTrip verifies nil optional fields survive round-trip.
func TestAllScalarsUpdateNilFieldsRoundTrip(t *testing.T) {
	t.Parallel()

	orig := &dao.AllScalarsUpdate{FSint32: -1}
	wire, err := orig.MarshalBinary()
	if err != nil {
		t.Fatalf("MarshalBinary: %v", err)
	}

	var got dao.AllScalarsUpdate
	if err := got.UnmarshalBinary(wire); err != nil {
		t.Fatalf("UnmarshalBinary: %v", err)
	}

	if got.FSint32 != -1 {
		t.Errorf("FSint32: got %d, want -1", got.FSint32)
	}
	if got.FSint64 != nil {
		t.Errorf("FSint64: got %v, want nil", got.FSint64)
	}
	if got.FDouble != nil {
		t.Errorf("FDouble: got %v, want nil", got.FDouble)
	}
}

// TestAllScalarsUpdateToMap verifies ToMap only includes non-nil optional fields.
// FSint32 is the condition field (non-optional) and is never included in ToMap.
func TestAllScalarsUpdateToMap(t *testing.T) {
	t.Parallel()

	fdouble := 3.14
	orig := &dao.AllScalarsUpdate{
		FSint32: -5, // condition field — never in ToMap
		FDouble: &fdouble,
	}

	m := orig.ToMap()

	// FSint32 is the condition field → must NOT be in map
	if _, ok := m["f_sint32"]; ok {
		t.Error("condition field f_sint32 must not appear in ToMap()")
	}
	// FDouble is non-nil → must be in map
	if _, ok := m["f_double"]; !ok {
		t.Error("f_double should be in ToMap()")
	}
	// nil fields must not be in map
	for _, key := range []string{"f_sint64", "f_sfixed32", "f_sfixed64", "f_fixed32",
		"f_fixed64", "f_uint32", "f_uint64", "f_float"} {
		if _, ok := m[key]; ok {
			t.Errorf("nil field %q should not appear in ToMap()", key)
		}
	}
}

// TestAllScalarsUpdateToMapEmpty verifies that ToMap returns empty map when all optional fields are nil.
func TestAllScalarsUpdateToMapEmpty(t *testing.T) {
	t.Parallel()

	orig := &dao.AllScalarsUpdate{FSint32: -5}
	m := orig.ToMap()

	if len(m) != 0 {
		t.Errorf("ToMap() should be empty when all optional fields are nil, got %v", m)
	}
}

// --- AllScalarsCreate round-trip ---

// TestAllScalarsCreateRoundTrip verifies MarshalBinary → UnmarshalBinary for AllScalarsCreate.
func TestAllScalarsCreateRoundTrip(t *testing.T) {
	t.Parallel()

	sint32 := int32(-42)
	sint64 := int64(-1000)
	sfixed32 := int32(-7)
	sfixed64 := int64(-999)
	fixed32 := uint32(0xDEAD)
	fixed64 := uint64(0xCAFEBABE)
	fuint32 := uint32(12345)
	fuint64 := uint64(9876543210)
	ffloat := float32(1.414)

	orig := &dao.AllScalarsCreate{
		FSint32:   &sint32,
		FSint64:   &sint64,
		FSfixed32: &sfixed32,
		FSfixed64: &sfixed64,
		FFixed32:  &fixed32,
		FFixed64:  &fixed64,
		FUint32:   &fuint32,
		FUint64:   &fuint64,
		FFloat:    &ffloat,
		FBytes:    []byte{0xAA, 0xBB},
	}

	wire, err := orig.MarshalBinary()
	if err != nil {
		t.Fatalf("MarshalBinary: %v", err)
	}

	var got dao.AllScalarsCreate
	if err := got.UnmarshalBinary(wire); err != nil {
		t.Fatalf("UnmarshalBinary: %v", err)
	}

	if got.FSint32 == nil || *got.FSint32 != *orig.FSint32 {
		t.Errorf("FSint32: got %v, want %v", got.FSint32, orig.FSint32)
	}
	if got.FSint64 == nil || *got.FSint64 != *orig.FSint64 {
		t.Errorf("FSint64: got %v, want %v", got.FSint64, orig.FSint64)
	}
	if got.FSfixed32 == nil || *got.FSfixed32 != *orig.FSfixed32 {
		t.Errorf("FSfixed32: got %v, want %v", got.FSfixed32, orig.FSfixed32)
	}
	if got.FSfixed64 == nil || *got.FSfixed64 != *orig.FSfixed64 {
		t.Errorf("FSfixed64: got %v, want %v", got.FSfixed64, orig.FSfixed64)
	}
	if got.FFixed32 == nil || *got.FFixed32 != *orig.FFixed32 {
		t.Errorf("FFixed32: got %v, want %v", got.FFixed32, orig.FFixed32)
	}
	if got.FFixed64 == nil || *got.FFixed64 != *orig.FFixed64 {
		t.Errorf("FFixed64: got %v, want %v", got.FFixed64, orig.FFixed64)
	}
	if got.FUint32 == nil || *got.FUint32 != *orig.FUint32 {
		t.Errorf("FUint32: got %v, want %v", got.FUint32, orig.FUint32)
	}
	if got.FUint64 == nil || *got.FUint64 != *orig.FUint64 {
		t.Errorf("FUint64: got %v, want %v", got.FUint64, orig.FUint64)
	}
	if got.FFloat == nil || *got.FFloat != *orig.FFloat {
		t.Errorf("FFloat: got %v, want %v", got.FFloat, orig.FFloat)
	}
	if !bytes.Equal(got.FBytes, orig.FBytes) {
		t.Errorf("FBytes: got %x, want %x", got.FBytes, orig.FBytes)
	}
}

// TestAllScalarsCreateNilFieldsRoundTrip verifies nil optional fields survive round-trip.
func TestAllScalarsCreateNilFieldsRoundTrip(t *testing.T) {
	t.Parallel()

	orig := &dao.AllScalarsCreate{FBytes: []byte{0x01}}
	wire, err := orig.MarshalBinary()
	if err != nil {
		t.Fatalf("MarshalBinary: %v", err)
	}

	var got dao.AllScalarsCreate
	if err := got.UnmarshalBinary(wire); err != nil {
		t.Fatalf("UnmarshalBinary: %v", err)
	}

	if got.FSint32 != nil {
		t.Errorf("FSint32: got %v, want nil", got.FSint32)
	}
	if !bytes.Equal(got.FBytes, []byte{0x01}) {
		t.Errorf("FBytes: got %x, want 01", got.FBytes)
	}
}

// TestAllScalarsCreateValidate verifies AllScalarsCreate.Validate() always returns nil.
func TestAllScalarsCreateValidate(t *testing.T) {
	t.Parallel()

	c := &dao.AllScalarsCreate{}
	if err := c.Validate(); err != nil {
		t.Errorf("AllScalarsCreate.Validate() = %v, want nil", err)
	}
}

// TestAllScalarsUpdateValidate verifies AllScalarsUpdate.Validate() always returns nil.
func TestAllScalarsUpdateValidate(t *testing.T) {
	t.Parallel()

	u := &dao.AllScalarsUpdate{}
	if err := u.Validate(); err != nil {
		t.Errorf("AllScalarsUpdate.Validate() = %v, want nil", err)
	}
}

// TestAllScalarsValidate verifies AllScalars.Validate() always returns nil.
func TestAllScalarsValidate(t *testing.T) {
	t.Parallel()

	a := &dao.AllScalars{}
	if err := a.Validate(); err != nil {
		t.Errorf("AllScalars.Validate() = %v, want nil", err)
	}
}

// TestAllRepeatedValidate verifies AllRepeated.Validate() always returns nil.
func TestAllRepeatedValidate(t *testing.T) {
	t.Parallel()

	a := &dao.AllRepeated{}
	if err := a.Validate(); err != nil {
		t.Errorf("AllRepeated.Validate() = %v, want nil", err)
	}
}
