// Package compat_test — additional error path tests for AllValidate and AllRepeated
// to push unmarshalFrom coverage above 80%.
package compat_test

import (
	"errors"
	"testing"

	"github.com/pinealctx/gcode/runtime"
	"github.com/pinealctx/gcode/testdata/compat/dao"
)

// --- AllValidate more error paths ---

func TestAllValidateUGteDuplicateField(t *testing.T) {
	t.Parallel()

	wire := buildDupField(1, runtime.WireVarint, func(b []byte) []byte {
		return runtime.AppendVarint(b, 5)
	})
	var a dao.AllValidate
	if err := a.UnmarshalBinary(wire); err == nil {
		t.Fatal("expected ErrDuplicateField, got nil")
	}
}

func TestAllValidateULteDuplicateField(t *testing.T) {
	t.Parallel()

	wire := buildDupField(2, runtime.WireVarint, func(b []byte) []byte {
		return runtime.AppendVarint(b, 500)
	})
	var a dao.AllValidate
	if err := a.UnmarshalBinary(wire); err == nil {
		t.Fatal("expected ErrDuplicateField, got nil")
	}
}

func TestAllValidateIInDuplicateField(t *testing.T) {
	t.Parallel()

	wire := buildDupField(9, runtime.WireVarint, func(b []byte) []byte {
		return runtime.AppendVarint(b, 1)
	})
	var a dao.AllValidate
	if err := a.UnmarshalBinary(wire); err == nil {
		t.Fatal("expected ErrDuplicateField, got nil")
	}
}

func TestAllValidateSUriDuplicateField(t *testing.T) {
	t.Parallel()

	wire := buildDupField(10, runtime.WireBytes, func(b []byte) []byte {
		return runtime.AppendString(b, "https://example.com")
	})
	var a dao.AllValidate
	if err := a.UnmarshalBinary(wire); err == nil {
		t.Fatal("expected ErrDuplicateField, got nil")
	}
}

func TestAllValidateOStatusDuplicateField(t *testing.T) {
	t.Parallel()

	wire := buildDupField(11, runtime.WireVarint, func(b []byte) []byte {
		return runtime.AppendVarint(b, 1)
	})
	var a dao.AllValidate
	if err := a.UnmarshalBinary(wire); err == nil {
		t.Fatal("expected ErrDuplicateField, got nil")
	}
}

func TestAllValidateBMinmaxDuplicateField(t *testing.T) {
	t.Parallel()

	wire := buildDupField(12, runtime.WireBytes, func(b []byte) []byte {
		return runtime.AppendBytes(b, []byte{0x01})
	})
	var a dao.AllValidate
	if err := a.UnmarshalBinary(wire); err == nil {
		t.Fatal("expected ErrDuplicateField, got nil")
	}
}

func TestAllValidateUInWireTypeError(t *testing.T) {
	t.Parallel()

	var wire []byte
	wire = runtime.AppendTag(wire, 3, runtime.WireFixed32)
	wire = runtime.AppendFixed32(wire, 2)
	var a dao.AllValidate
	err := a.UnmarshalBinary(wire)
	if err == nil {
		t.Fatal("expected error for wrong wire type, got nil")
	}
	if !errors.Is(err, runtime.ErrWireType) {
		t.Errorf("expected ErrWireType, got: %v", err)
	}
}

func TestAllValidateUNotInWireTypeError(t *testing.T) {
	t.Parallel()

	var wire []byte
	wire = runtime.AppendTag(wire, 4, runtime.WireFixed32)
	wire = runtime.AppendFixed32(wire, 5)
	var a dao.AllValidate
	err := a.UnmarshalBinary(wire)
	if err == nil {
		t.Fatal("expected error for wrong wire type, got nil")
	}
	if !errors.Is(err, runtime.ErrWireType) {
		t.Errorf("expected ErrWireType, got: %v", err)
	}
}

func TestAllValidateSNotInWireTypeError(t *testing.T) {
	t.Parallel()

	var wire []byte
	wire = runtime.AppendTag(wire, 8, runtime.WireVarint)
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

func TestAllValidateIInWireTypeError(t *testing.T) {
	t.Parallel()

	var wire []byte
	wire = runtime.AppendTag(wire, 9, runtime.WireFixed32)
	wire = runtime.AppendFixed32(wire, 1)
	var a dao.AllValidate
	err := a.UnmarshalBinary(wire)
	if err == nil {
		t.Fatal("expected error for wrong wire type, got nil")
	}
	if !errors.Is(err, runtime.ErrWireType) {
		t.Errorf("expected ErrWireType, got: %v", err)
	}
}

func TestAllValidateSUriWireTypeError(t *testing.T) {
	t.Parallel()

	var wire []byte
	wire = runtime.AppendTag(wire, 10, runtime.WireVarint)
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

func TestAllValidateBMinmaxWireTypeError(t *testing.T) {
	t.Parallel()

	var wire []byte
	wire = runtime.AppendTag(wire, 12, runtime.WireVarint)
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

func TestAllValidateRItemsWireTypeError(t *testing.T) {
	t.Parallel()

	var wire []byte
	wire = runtime.AppendTag(wire, 13, runtime.WireVarint)
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

func TestAllValidateSUriTruncated(t *testing.T) {
	t.Parallel()

	wire := buildTruncatedField(10) // s_uri
	var a dao.AllValidate
	if err := a.UnmarshalBinary(wire); err == nil {
		t.Fatal("expected error for truncated s_uri, got nil")
	}
}

func TestAllValidateRItemsTruncated(t *testing.T) {
	t.Parallel()

	wire := buildTruncatedField(13) // r_items
	var a dao.AllValidate
	if err := a.UnmarshalBinary(wire); err == nil {
		t.Fatal("expected error for truncated r_items, got nil")
	}
}

// --- AllRepeated more error paths ---

func TestAllRepeatedRSfixed32Truncated(t *testing.T) {
	t.Parallel()

	wire := buildTruncatedField(2) // r_sfixed32
	var a dao.AllRepeated
	if err := a.UnmarshalBinary(wire); err == nil {
		t.Fatal("expected error for truncated r_sfixed32, got nil")
	}
}

func TestAllRepeatedRDoubleTruncated(t *testing.T) {
	t.Parallel()

	wire := buildTruncatedField(3) // r_double
	var a dao.AllRepeated
	if err := a.UnmarshalBinary(wire); err == nil {
		t.Fatal("expected error for truncated r_double, got nil")
	}
}

func TestAllRepeatedRBytesTruncated(t *testing.T) {
	t.Parallel()

	wire := buildTruncatedField(4) // r_bytes
	var a dao.AllRepeated
	if err := a.UnmarshalBinary(wire); err == nil {
		t.Fatal("expected error for truncated r_bytes, got nil")
	}
}

func TestAllRepeatedRMessageTruncated(t *testing.T) {
	t.Parallel()

	wire := buildTruncatedField(5) // r_message
	var a dao.AllRepeated
	if err := a.UnmarshalBinary(wire); err == nil {
		t.Fatal("expected error for truncated r_message, got nil")
	}
}

func TestAllRepeatedREnumTruncated(t *testing.T) {
	t.Parallel()

	wire := buildTruncatedField(6) // r_enum
	var a dao.AllRepeated
	if err := a.UnmarshalBinary(wire); err == nil {
		t.Fatal("expected error for truncated r_enum, got nil")
	}
}

// --- AllScalarsCreate more error paths ---

func TestAllScalarsCreateFixed32WireTypeError2(t *testing.T) {
	t.Parallel()

	// Field 5 = f_fixed32 (WireFixed32). Encode as WireVarint.
	var wire []byte
	wire = runtime.AppendTag(wire, 5, runtime.WireVarint)
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

func TestAllScalarsCreateFixed64WireTypeError2(t *testing.T) {
	t.Parallel()

	// Field 6 = f_fixed64 (WireFixed64). Encode as WireVarint.
	var wire []byte
	wire = runtime.AppendTag(wire, 6, runtime.WireVarint)
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

func TestAllScalarsCreateFloatWireTypeError(t *testing.T) {
	t.Parallel()

	// Field 9 = f_float (WireFixed32). Encode as WireVarint.
	var wire []byte
	wire = runtime.AppendTag(wire, 9, runtime.WireVarint)
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

func TestAllScalarsCreateBytesTruncated(t *testing.T) {
	t.Parallel()

	wire := buildTruncatedField(10) // f_bytes
	var a dao.AllScalarsCreate
	if err := a.UnmarshalBinary(wire); err == nil {
		t.Fatal("expected error for truncated f_bytes, got nil")
	}
}

// --- AllScalarsUpdate more error paths ---

func TestAllScalarsUpdateFixed32WireTypeError2(t *testing.T) {
	t.Parallel()

	// Field 6 = f_fixed32 (WireFixed32). Encode as WireVarint.
	var wire []byte
	wire = runtime.AppendTag(wire, 6, runtime.WireVarint)
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

func TestAllScalarsUpdateFixed64WireTypeError(t *testing.T) {
	t.Parallel()

	// Field 7 = f_fixed64 (WireFixed64). Encode as WireVarint.
	var wire []byte
	wire = runtime.AppendTag(wire, 7, runtime.WireVarint)
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

func TestAllScalarsUpdateFloatWireTypeError(t *testing.T) {
	t.Parallel()

	// Field 10 = f_float (WireFixed32). Encode as WireVarint.
	var wire []byte
	wire = runtime.AppendTag(wire, 10, runtime.WireVarint)
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
