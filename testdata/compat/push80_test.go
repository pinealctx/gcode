// Package compat_test — final push to reach 80% coverage.
// Targets AllScalarsCreate.unmarshalFrom and AllScalars.unmarshalFrom.
package compat_test

import (
	"testing"

	"github.com/pinealctx/gcode/runtime"
	"github.com/pinealctx/gcode/testdata/compat/dao"
)

// --- AllScalarsCreate more unmarshal coverage ---

// TestAllScalarsCreateLenient exercises AllScalarsCreate.unmarshalFrom in lenient mode.
func TestAllScalarsCreateLenient(t *testing.T) {
	t.Parallel()

	sint32 := int32(-42)
	orig := &dao.AllScalarsCreate{FSint32: &sint32, FBytes: []byte{0x01}}
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

// TestAllScalarsCreateFixed32Truncated2 covers truncated f_fixed32 (field 5).
func TestAllScalarsCreateFixed32Truncated2(t *testing.T) {
	t.Parallel()

	wire := buildTruncatedFixed32(5) // f_fixed32
	var a dao.AllScalarsCreate
	if err := a.UnmarshalBinary(wire); err == nil {
		t.Fatal("expected error for truncated f_fixed32, got nil")
	}
}

// TestAllScalarsCreateFixed64Truncated2 covers truncated f_fixed64 (field 6).
func TestAllScalarsCreateFixed64Truncated2(t *testing.T) {
	t.Parallel()

	wire := buildTruncatedFixed64(6) // f_fixed64
	var a dao.AllScalarsCreate
	if err := a.UnmarshalBinary(wire); err == nil {
		t.Fatal("expected error for truncated f_fixed64, got nil")
	}
}

// TestAllScalarsCreateFloatTruncated covers truncated f_float (field 9).
func TestAllScalarsCreateFloatTruncated(t *testing.T) {
	t.Parallel()

	wire := buildTruncatedFixed32(9) // f_float
	var a dao.AllScalarsCreate
	if err := a.UnmarshalBinary(wire); err == nil {
		t.Fatal("expected error for truncated f_float, got nil")
	}
}

// TestAllScalarsCreateSint64DuplicateField covers duplicate f_sint64 (field 2).
func TestAllScalarsCreateSint64DuplicateField(t *testing.T) {
	t.Parallel()

	wire := buildDupField(2, runtime.WireVarint, func(b []byte) []byte {
		return runtime.AppendVarint(b, 42)
	})
	var a dao.AllScalarsCreate
	if err := a.UnmarshalBinary(wire); err == nil {
		t.Fatal("expected ErrDuplicateField, got nil")
	}
}

// TestAllScalarsCreateFixed32DuplicateField covers duplicate f_fixed32 (field 5).
func TestAllScalarsCreateFixed32DuplicateField(t *testing.T) {
	t.Parallel()

	wire := buildDupField(5, runtime.WireFixed32, func(b []byte) []byte {
		return runtime.AppendFixed32(b, 42)
	})
	var a dao.AllScalarsCreate
	if err := a.UnmarshalBinary(wire); err == nil {
		t.Fatal("expected ErrDuplicateField, got nil")
	}
}

// TestAllScalarsCreateFixed64DuplicateField covers duplicate f_fixed64 (field 6).
func TestAllScalarsCreateFixed64DuplicateField(t *testing.T) {
	t.Parallel()

	wire := buildDupField(6, runtime.WireFixed64, func(b []byte) []byte {
		return runtime.AppendFixed64(b, 42)
	})
	var a dao.AllScalarsCreate
	if err := a.UnmarshalBinary(wire); err == nil {
		t.Fatal("expected ErrDuplicateField, got nil")
	}
}

// TestAllScalarsCreateFloatDuplicateField covers duplicate f_float (field 9).
func TestAllScalarsCreateFloatDuplicateField(t *testing.T) {
	t.Parallel()

	wire := buildDupField(9, runtime.WireFixed32, func(b []byte) []byte {
		return runtime.AppendFixed32(b, 42)
	})
	var a dao.AllScalarsCreate
	if err := a.UnmarshalBinary(wire); err == nil {
		t.Fatal("expected ErrDuplicateField, got nil")
	}
}

// --- AllScalars more unmarshal coverage ---

// TestAllScalarsDuplicateFixed64 covers duplicate f_sfixed64 (field 4).
func TestAllScalarsDuplicateFixed64(t *testing.T) {
	t.Parallel()

	wire := buildDupField(4, runtime.WireFixed64, func(b []byte) []byte {
		return runtime.AppendFixed64(b, 42)
	})
	var a dao.AllScalars
	if err := a.UnmarshalBinary(wire); err == nil {
		t.Fatal("expected ErrDuplicateField, got nil")
	}
}

// TestAllScalarsDuplicateDouble covers duplicate f_double (field 5).
func TestAllScalarsDuplicateDouble(t *testing.T) {
	t.Parallel()

	wire := buildDupField(5, runtime.WireFixed64, func(b []byte) []byte {
		return runtime.AppendFixed64(b, 42)
	})
	var a dao.AllScalars
	if err := a.UnmarshalBinary(wire); err == nil {
		t.Fatal("expected ErrDuplicateField, got nil")
	}
}

// TestAllScalarsDuplicateFixed32 covers duplicate f_fixed32 (field 6).
func TestAllScalarsDuplicateFixed32(t *testing.T) {
	t.Parallel()

	wire := buildDupField(6, runtime.WireFixed32, func(b []byte) []byte {
		return runtime.AppendFixed32(b, 42)
	})
	var a dao.AllScalars
	if err := a.UnmarshalBinary(wire); err == nil {
		t.Fatal("expected ErrDuplicateField, got nil")
	}
}

// TestAllScalarsDuplicateFixed64Field7 covers duplicate f_fixed64 (field 7).
func TestAllScalarsDuplicateFixed64Field7(t *testing.T) {
	t.Parallel()

	wire := buildDupField(7, runtime.WireFixed64, func(b []byte) []byte {
		return runtime.AppendFixed64(b, 42)
	})
	var a dao.AllScalars
	if err := a.UnmarshalBinary(wire); err == nil {
		t.Fatal("expected ErrDuplicateField, got nil")
	}
}

// TestAllScalarsDuplicateFloat covers duplicate f_float (field 10).
func TestAllScalarsDuplicateFloat(t *testing.T) {
	t.Parallel()

	wire := buildDupField(10, runtime.WireFixed32, func(b []byte) []byte {
		return runtime.AppendFixed32(b, 42)
	})
	var a dao.AllScalars
	if err := a.UnmarshalBinary(wire); err == nil {
		t.Fatal("expected ErrDuplicateField, got nil")
	}
}

// TestAllScalarsDuplicateBytes covers duplicate f_bytes (field 11).
func TestAllScalarsDuplicateBytes(t *testing.T) {
	t.Parallel()

	wire := buildDupField(11, runtime.WireBytes, func(b []byte) []byte {
		return runtime.AppendBytes(b, []byte{0x01})
	})
	var a dao.AllScalars
	if err := a.UnmarshalBinary(wire); err == nil {
		t.Fatal("expected ErrDuplicateField, got nil")
	}
}
