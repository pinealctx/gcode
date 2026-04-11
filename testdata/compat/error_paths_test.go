// Package compat_test — error path tests to boost unmarshalFrom coverage.
// Covers ErrDuplicateField, ErrWireType, and ErrTruncated for all generated types.
package compat_test

import (
	"errors"
	"testing"

	"github.com/pinealctx/gcode/runtime"
	"github.com/pinealctx/gcode/testdata/compat/dao"
)

// buildDupField builds wire bytes with a field encoded twice (triggers ErrDuplicateField).
func buildDupField(fieldNum int, wireType int, appendValue func([]byte) []byte) []byte {
	var b []byte
	b = runtime.AppendTag(b, fieldNum, wireType)
	b = appendValue(b)
	b = runtime.AppendTag(b, fieldNum, wireType)
	b = appendValue(b)
	return b
}

// buildTruncatedField builds wire bytes with a truncated LEN payload.
func buildTruncatedField(fieldNum int) []byte {
	var b []byte
	b = runtime.AppendTag(b, fieldNum, runtime.WireBytes)
	b = runtime.AppendVarint(b, 100) // claim 100 bytes but provide none
	return b
}

// --- Person duplicate field rejection ---

func TestPersonDuplicateNameRejected(t *testing.T) {
	t.Parallel()

	wire := buildDupField(1, runtime.WireBytes, func(b []byte) []byte {
		return runtime.AppendString(b, "Alice")
	})

	var p dao.Person
	err := p.UnmarshalBinary(wire)
	if err == nil {
		t.Fatal("expected ErrDuplicateField, got nil")
	}
	if !errors.Is(err, runtime.ErrDuplicateField) {
		t.Errorf("expected ErrDuplicateField, got: %v", err)
	}
}

func TestPersonDuplicateAgeRejected(t *testing.T) {
	t.Parallel()

	wire := buildDupField(2, runtime.WireVarint, func(b []byte) []byte {
		return runtime.AppendVarint(b, 30)
	})

	var p dao.Person
	err := p.UnmarshalBinary(wire)
	if err == nil {
		t.Fatal("expected ErrDuplicateField, got nil")
	}
	if !errors.Is(err, runtime.ErrDuplicateField) {
		t.Errorf("expected ErrDuplicateField, got: %v", err)
	}
}

// TestPersonTruncatedName verifies ErrTruncated for truncated string field.
func TestPersonTruncatedName(t *testing.T) {
	t.Parallel()

	wire := buildTruncatedField(1)

	var p dao.Person
	err := p.UnmarshalBinary(wire)
	if err == nil {
		t.Fatal("expected error for truncated field, got nil")
	}
}

// --- AllScalars duplicate field rejection ---

func TestAllScalarsDuplicateField(t *testing.T) {
	t.Parallel()

	// Field 3 = f_sfixed32 (WireFixed32)
	wire := buildDupField(3, runtime.WireFixed32, func(b []byte) []byte {
		return runtime.AppendFixed32(b, 42)
	})

	var a dao.AllScalars
	err := a.UnmarshalBinary(wire)
	if err == nil {
		t.Fatal("expected ErrDuplicateField, got nil")
	}
	if !errors.Is(err, runtime.ErrDuplicateField) {
		t.Errorf("expected ErrDuplicateField, got: %v", err)
	}
}

// TestAllScalarsTruncatedBytes verifies ErrTruncated for truncated bytes field.
func TestAllScalarsTruncatedBytes(t *testing.T) {
	t.Parallel()

	wire := buildTruncatedField(11) // f_bytes

	var a dao.AllScalars
	err := a.UnmarshalBinary(wire)
	if err == nil {
		t.Fatal("expected error for truncated field, got nil")
	}
}

// --- AllRepeated error paths ---

func TestAllRepeatedWireTypeError(t *testing.T) {
	t.Parallel()

	// Field 1 = r_sint32 (WireBytes for packed). Encode as WireVarint.
	var wire []byte
	wire = runtime.AppendTag(wire, 1, runtime.WireVarint)
	wire = runtime.AppendVarint(wire, 42)

	var a dao.AllRepeated
	err := a.UnmarshalBinary(wire)
	if err == nil {
		t.Fatal("expected error for wrong wire type, got nil")
	}
}

func TestAllRepeatedTruncatedPacked(t *testing.T) {
	t.Parallel()

	wire := buildTruncatedField(1) // r_sint32 packed

	var a dao.AllRepeated
	err := a.UnmarshalBinary(wire)
	if err == nil {
		t.Fatal("expected error for truncated packed field, got nil")
	}
}

// --- AllValidate duplicate field rejection ---

func TestAllValidateDuplicateField(t *testing.T) {
	t.Parallel()

	// Field 7 = s_in (WireBytes)
	wire := buildDupField(7, runtime.WireBytes, func(b []byte) []byte {
		return runtime.AppendString(b, "a")
	})

	var a dao.AllValidate
	err := a.UnmarshalBinary(wire)
	if err == nil {
		t.Fatal("expected ErrDuplicateField, got nil")
	}
	if !errors.Is(err, runtime.ErrDuplicateField) {
		t.Errorf("expected ErrDuplicateField, got: %v", err)
	}
}

func TestAllValidateTruncatedField(t *testing.T) {
	t.Parallel()

	wire := buildTruncatedField(7) // s_in

	var a dao.AllValidate
	err := a.UnmarshalBinary(wire)
	if err == nil {
		t.Fatal("expected error for truncated field, got nil")
	}
}

// --- AllScalarsUpdate duplicate field rejection ---

func TestAllScalarsUpdateDuplicateField(t *testing.T) {
	t.Parallel()

	// Field 2 = f_sint64 (WireVarint, optional)
	wire := buildDupField(2, runtime.WireVarint, func(b []byte) []byte {
		return runtime.AppendVarint(b, 42)
	})

	var a dao.AllScalarsUpdate
	err := a.UnmarshalBinary(wire)
	if err == nil {
		t.Fatal("expected ErrDuplicateField, got nil")
	}
	if !errors.Is(err, runtime.ErrDuplicateField) {
		t.Errorf("expected ErrDuplicateField, got: %v", err)
	}
}

// --- AllScalarsCreate duplicate field rejection ---

func TestAllScalarsCreateDuplicateField(t *testing.T) {
	t.Parallel()

	// Field 1 = f_sint32 (WireVarint, optional)
	wire := buildDupField(1, runtime.WireVarint, func(b []byte) []byte {
		return runtime.AppendVarint(b, 42)
	})

	var a dao.AllScalarsCreate
	err := a.UnmarshalBinary(wire)
	if err == nil {
		t.Fatal("expected ErrDuplicateField, got nil")
	}
	if !errors.Is(err, runtime.ErrDuplicateField) {
		t.Errorf("expected ErrDuplicateField, got: %v", err)
	}
}

// --- PersonCreate duplicate field rejection ---

func TestPersonCreateDuplicateField(t *testing.T) {
	t.Parallel()

	// Field 1 = name (WireBytes, optional)
	wire := buildDupField(1, runtime.WireBytes, func(b []byte) []byte {
		return runtime.AppendString(b, "Alice")
	})

	var p dao.PersonCreate
	err := p.UnmarshalBinary(wire)
	if err == nil {
		t.Fatal("expected ErrDuplicateField, got nil")
	}
	if !errors.Is(err, runtime.ErrDuplicateField) {
		t.Errorf("expected ErrDuplicateField, got: %v", err)
	}
}

func TestPersonCreateTruncatedField(t *testing.T) {
	t.Parallel()

	wire := buildTruncatedField(1) // name

	var p dao.PersonCreate
	err := p.UnmarshalBinary(wire)
	if err == nil {
		t.Fatal("expected error for truncated field, got nil")
	}
}

// --- PersonUpdateByName duplicate field rejection ---

func TestPersonUpdateByNameDuplicateField(t *testing.T) {
	t.Parallel()

	// Field 1 = name (WireBytes, non-optional)
	wire := buildDupField(1, runtime.WireBytes, func(b []byte) []byte {
		return runtime.AppendString(b, "Alice")
	})

	var p dao.PersonUpdateByName
	err := p.UnmarshalBinary(wire)
	if err == nil {
		t.Fatal("expected ErrDuplicateField, got nil")
	}
	if !errors.Is(err, runtime.ErrDuplicateField) {
		t.Errorf("expected ErrDuplicateField, got: %v", err)
	}
}

func TestPersonUpdateByNameTruncatedField(t *testing.T) {
	t.Parallel()

	wire := buildTruncatedField(1) // name

	var p dao.PersonUpdateByName
	err := p.UnmarshalBinary(wire)
	if err == nil {
		t.Fatal("expected error for truncated field, got nil")
	}
}

// --- AllScalarsUpdate ToMap full coverage ---

// TestAllScalarsUpdateToMapAllFields verifies ToMap includes all non-nil optional fields.
func TestAllScalarsUpdateToMapAllFields(t *testing.T) {
	t.Parallel()

	sint64 := int64(-500)
	sfixed32 := int32(-7)
	sfixed64 := int64(-999)
	fdouble := 2.718
	fixed32 := uint32(0xDEAD)
	fixed64 := uint64(0xCAFEBABE)
	fuint32 := uint32(12345)
	fuint64 := uint64(9876543210)
	ffloat := float32(1.414)

	orig := &dao.AllScalarsUpdate{
		FSint64:   &sint64,
		FSfixed32: &sfixed32,
		FSfixed64: &sfixed64,
		FDouble:   &fdouble,
		FFixed32:  &fixed32,
		FFixed64:  &fixed64,
		FUint32:   &fuint32,
		FUint64:   &fuint64,
		FFloat:    &ffloat,
	}

	m := orig.ToMap()

	for _, key := range []string{"f_sint64", "f_sfixed32", "f_sfixed64", "f_double",
		"f_fixed32", "f_fixed64", "f_uint32", "f_uint64", "f_float"} {
		if _, ok := m[key]; !ok {
			t.Errorf("field %q should be in ToMap()", key)
		}
	}
}
