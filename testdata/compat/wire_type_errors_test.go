// Package compat_test — targeted tests to push unmarshalFrom coverage above 80%.
// Covers wire type errors and truncated payloads for more fields in each type.
package compat_test

import (
	"errors"
	"testing"

	"github.com/pinealctx/gcode/runtime"
	"github.com/pinealctx/gcode/testdata/compat/dao"
)

// --- Address unmarshalFrom coverage ---

func TestAddressWireTypeError(t *testing.T) {
	t.Parallel()

	// Field 1 = street (WireBytes). Encode as WireVarint.
	var wire []byte
	wire = runtime.AppendTag(wire, 1, runtime.WireVarint)
	wire = runtime.AppendVarint(wire, 42)

	var a dao.Address
	err := a.UnmarshalBinary(wire)
	if err == nil {
		t.Fatal("expected error for wrong wire type, got nil")
	}
	if !errors.Is(err, runtime.ErrWireType) {
		t.Errorf("expected ErrWireType, got: %v", err)
	}
}

func TestAddressCityWireTypeError(t *testing.T) {
	t.Parallel()

	// Field 2 = city (WireBytes). Encode as WireVarint.
	var wire []byte
	wire = runtime.AppendTag(wire, 2, runtime.WireVarint)
	wire = runtime.AppendVarint(wire, 42)

	var a dao.Address
	err := a.UnmarshalBinary(wire)
	if err == nil {
		t.Fatal("expected error for wrong wire type, got nil")
	}
	if !errors.Is(err, runtime.ErrWireType) {
		t.Errorf("expected ErrWireType, got: %v", err)
	}
}

func TestAddressCityDuplicateField(t *testing.T) {
	t.Parallel()

	wire := buildDupField(2, runtime.WireBytes, func(b []byte) []byte {
		return runtime.AppendString(b, "Springfield")
	})

	var a dao.Address
	if err := a.UnmarshalBinary(wire); err == nil {
		t.Fatal("expected ErrDuplicateField, got nil")
	}
}

// --- AllScalars more field coverage ---

// TestAllScalarsFixed64WireTypeError covers field 4 (f_sfixed64, WireFixed64).
func TestAllScalarsFixed64WireTypeError(t *testing.T) {
	t.Parallel()

	var wire []byte
	wire = runtime.AppendTag(wire, 4, runtime.WireVarint)
	wire = runtime.AppendVarint(wire, 42)

	var a dao.AllScalars
	err := a.UnmarshalBinary(wire)
	if err == nil {
		t.Fatal("expected error for wrong wire type, got nil")
	}
	if !errors.Is(err, runtime.ErrWireType) {
		t.Errorf("expected ErrWireType, got: %v", err)
	}
}

// TestAllScalarsFixed32WireTypeError covers field 6 (f_fixed32, WireFixed32).
func TestAllScalarsFixed32WireTypeError(t *testing.T) {
	t.Parallel()

	var wire []byte
	wire = runtime.AppendTag(wire, 6, runtime.WireVarint)
	wire = runtime.AppendVarint(wire, 42)

	var a dao.AllScalars
	err := a.UnmarshalBinary(wire)
	if err == nil {
		t.Fatal("expected error for wrong wire type, got nil")
	}
	if !errors.Is(err, runtime.ErrWireType) {
		t.Errorf("expected ErrWireType, got: %v", err)
	}
}

// TestAllScalarsFloatWireTypeError covers field 10 (f_float, WireFixed32).
func TestAllScalarsFloatWireTypeError(t *testing.T) {
	t.Parallel()

	var wire []byte
	wire = runtime.AppendTag(wire, 10, runtime.WireVarint)
	wire = runtime.AppendVarint(wire, 42)

	var a dao.AllScalars
	err := a.UnmarshalBinary(wire)
	if err == nil {
		t.Fatal("expected error for wrong wire type, got nil")
	}
	if !errors.Is(err, runtime.ErrWireType) {
		t.Errorf("expected ErrWireType, got: %v", err)
	}
}

// --- AllRepeated more field coverage ---

// TestAllRepeatedRBytesWireTypeError covers field 4 (r_bytes, WireBytes).
func TestAllRepeatedRBytesWireTypeError(t *testing.T) {
	t.Parallel()

	var wire []byte
	wire = runtime.AppendTag(wire, 4, runtime.WireVarint)
	wire = runtime.AppendVarint(wire, 42)

	var a dao.AllRepeated
	err := a.UnmarshalBinary(wire)
	if err == nil {
		t.Fatal("expected error for wrong wire type, got nil")
	}
	if !errors.Is(err, runtime.ErrWireType) {
		t.Errorf("expected ErrWireType, got: %v", err)
	}
}

// TestAllRepeatedRMessageWireTypeError covers field 5 (r_message, WireBytes).
func TestAllRepeatedRMessageWireTypeError(t *testing.T) {
	t.Parallel()

	var wire []byte
	wire = runtime.AppendTag(wire, 5, runtime.WireVarint)
	wire = runtime.AppendVarint(wire, 42)

	var a dao.AllRepeated
	err := a.UnmarshalBinary(wire)
	if err == nil {
		t.Fatal("expected error for wrong wire type, got nil")
	}
	if !errors.Is(err, runtime.ErrWireType) {
		t.Errorf("expected ErrWireType, got: %v", err)
	}
}

// TestAllRepeatedREnumWireTypeError covers field 6 (r_enum, WireBytes for packed).
func TestAllRepeatedREnumWireTypeError(t *testing.T) {
	t.Parallel()

	var wire []byte
	wire = runtime.AppendTag(wire, 6, runtime.WireVarint)
	wire = runtime.AppendVarint(wire, 42)

	var a dao.AllRepeated
	err := a.UnmarshalBinary(wire)
	if err == nil {
		t.Fatal("expected error for wrong wire type, got nil")
	}
	if !errors.Is(err, runtime.ErrWireType) {
		t.Errorf("expected ErrWireType, got: %v", err)
	}
}

// TestAllRepeatedRDoubleWireTypeError covers field 3 (r_double, WireBytes for packed).
func TestAllRepeatedRDoubleWireTypeError(t *testing.T) {
	t.Parallel()

	var wire []byte
	wire = runtime.AppendTag(wire, 3, runtime.WireVarint)
	wire = runtime.AppendVarint(wire, 42)

	var a dao.AllRepeated
	err := a.UnmarshalBinary(wire)
	if err == nil {
		t.Fatal("expected error for wrong wire type, got nil")
	}
	if !errors.Is(err, runtime.ErrWireType) {
		t.Errorf("expected ErrWireType, got: %v", err)
	}
}

// --- AllValidate more field coverage ---

// TestAllValidateFGtWireTypeError covers field 5 (f_gt, WireFixed32).
func TestAllValidateFGtWireTypeError(t *testing.T) {
	t.Parallel()

	var wire []byte
	wire = runtime.AppendTag(wire, 5, runtime.WireVarint)
	wire = runtime.AppendVarint(wire, 42)

	var a dao.AllValidate
	err := a.UnmarshalBinary(wire)
	if err == nil {
		t.Fatal("expected error for wrong wire type, got nil")
	}
	if !errors.Is(err, runtime.ErrWireType) {
		t.Errorf("expected ErrWireType, got: %v", err)
	}
}

// TestAllValidateDLteWireTypeError covers field 6 (d_lte, WireFixed64).
func TestAllValidateDLteWireTypeError(t *testing.T) {
	t.Parallel()

	var wire []byte
	wire = runtime.AppendTag(wire, 6, runtime.WireVarint)
	wire = runtime.AppendVarint(wire, 42)

	var a dao.AllValidate
	err := a.UnmarshalBinary(wire)
	if err == nil {
		t.Fatal("expected error for wrong wire type, got nil")
	}
	if !errors.Is(err, runtime.ErrWireType) {
		t.Errorf("expected ErrWireType, got: %v", err)
	}
}

// TestAllValidateOStatusWireTypeError covers field 11 (o_status, WireVarint).
func TestAllValidateOStatusWireTypeError(t *testing.T) {
	t.Parallel()

	var wire []byte
	wire = runtime.AppendTag(wire, 11, runtime.WireFixed32)
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

// --- AllScalarsUpdate more field coverage ---

// TestAllScalarsUpdateFixed32WireTypeError covers field 3 (f_sfixed32, WireFixed32).
func TestAllScalarsUpdateFixed32WireTypeError(t *testing.T) {
	t.Parallel()

	var wire []byte
	wire = runtime.AppendTag(wire, 3, runtime.WireVarint)
	wire = runtime.AppendVarint(wire, 42)

	var a dao.AllScalarsUpdate
	err := a.UnmarshalBinary(wire)
	if err == nil {
		t.Fatal("expected error for wrong wire type, got nil")
	}
	if !errors.Is(err, runtime.ErrWireType) {
		t.Errorf("expected ErrWireType, got: %v", err)
	}
}

// TestAllScalarsUpdateDoubleWireTypeError covers field 5 (f_double, WireFixed64).
func TestAllScalarsUpdateDoubleWireTypeError(t *testing.T) {
	t.Parallel()

	var wire []byte
	wire = runtime.AppendTag(wire, 5, runtime.WireVarint)
	wire = runtime.AppendVarint(wire, 42)

	var a dao.AllScalarsUpdate
	err := a.UnmarshalBinary(wire)
	if err == nil {
		t.Fatal("expected error for wrong wire type, got nil")
	}
	if !errors.Is(err, runtime.ErrWireType) {
		t.Errorf("expected ErrWireType, got: %v", err)
	}
}

// --- AllScalarsCreate more field coverage ---

// TestAllScalarsCreateFixed32WireTypeError covers field 3 (f_sfixed32, WireFixed32).
func TestAllScalarsCreateFixed32WireTypeError(t *testing.T) {
	t.Parallel()

	var wire []byte
	wire = runtime.AppendTag(wire, 3, runtime.WireVarint)
	wire = runtime.AppendVarint(wire, 42)

	var a dao.AllScalarsCreate
	err := a.UnmarshalBinary(wire)
	if err == nil {
		t.Fatal("expected error for wrong wire type, got nil")
	}
	if !errors.Is(err, runtime.ErrWireType) {
		t.Errorf("expected ErrWireType, got: %v", err)
	}
}

// TestAllScalarsCreateFixed64WireTypeError covers field 4 (f_sfixed64, WireFixed64).
func TestAllScalarsCreateFixed64WireTypeError(t *testing.T) {
	t.Parallel()

	var wire []byte
	wire = runtime.AppendTag(wire, 4, runtime.WireVarint)
	wire = runtime.AppendVarint(wire, 42)

	var a dao.AllScalarsCreate
	err := a.UnmarshalBinary(wire)
	if err == nil {
		t.Fatal("expected error for wrong wire type, got nil")
	}
	if !errors.Is(err, runtime.ErrWireType) {
		t.Errorf("expected ErrWireType, got: %v", err)
	}
}

// --- PersonCreate more field coverage ---

// TestPersonCreateActiveWireTypeError covers field 3 (active, WireVarint).
func TestPersonCreateActiveWireTypeError(t *testing.T) {
	t.Parallel()

	// active is bool (WireVarint). Encode as WireFixed32.
	var wire []byte
	wire = runtime.AppendTag(wire, 3, runtime.WireFixed32)
	wire = runtime.AppendFixed32(wire, 1)

	var p dao.PersonCreate
	err := p.UnmarshalBinary(wire)
	if err == nil {
		t.Fatal("expected error for wrong wire type, got nil")
	}
	if !errors.Is(err, runtime.ErrWireType) {
		t.Errorf("expected ErrWireType, got: %v", err)
	}
}

// TestPersonCreateRatingWireTypeError covers field 5 (rating, WireFixed32).
func TestPersonCreateRatingWireTypeError(t *testing.T) {
	t.Parallel()

	var wire []byte
	wire = runtime.AppendTag(wire, 5, runtime.WireVarint)
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

// --- PersonUpdateByName more field coverage ---

// TestPersonUpdateByNameRatingWireTypeError covers field 5 (rating, WireFixed32).
func TestPersonUpdateByNameRatingWireTypeError(t *testing.T) {
	t.Parallel()

	var wire []byte
	wire = runtime.AppendTag(wire, 5, runtime.WireVarint)
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

// TestPersonUpdateByNameActiveWireTypeError covers field 3 (active, WireVarint).
func TestPersonUpdateByNameActiveWireTypeError(t *testing.T) {
	t.Parallel()

	var wire []byte
	wire = runtime.AppendTag(wire, 3, runtime.WireFixed32)
	wire = runtime.AppendFixed32(wire, 1)

	var p dao.PersonUpdateByName
	err := p.UnmarshalBinary(wire)
	if err == nil {
		t.Fatal("expected error for wrong wire type, got nil")
	}
	if !errors.Is(err, runtime.ErrWireType) {
		t.Errorf("expected ErrWireType, got: %v", err)
	}
}
