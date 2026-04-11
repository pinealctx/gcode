// Package compat_test — truncated payload tests to cover ErrTruncated branches.
package compat_test

import (
	"testing"

	"github.com/pinealctx/gcode/runtime"
	"github.com/pinealctx/gcode/testdata/compat/dao"
)

// buildTruncatedFixed32 builds wire bytes with a truncated fixed32 payload.
func buildTruncatedFixed32(fieldNum int) []byte {
	var b []byte
	b = runtime.AppendTag(b, fieldNum, runtime.WireFixed32)
	// Only 3 bytes instead of 4
	b = append(b, 0x01, 0x02, 0x03)
	return b
}

// buildTruncatedFixed64 builds wire bytes with a truncated fixed64 payload.
func buildTruncatedFixed64(fieldNum int) []byte {
	var b []byte
	b = runtime.AppendTag(b, fieldNum, runtime.WireFixed64)
	// Only 7 bytes instead of 8
	b = append(b, 0x01, 0x02, 0x03, 0x04, 0x05, 0x06, 0x07)
	return b
}

// --- Address truncated field tests ---

func TestAddressStreetTruncated(t *testing.T) {
	t.Parallel()

	wire := buildTruncatedField(1) // street
	var a dao.Address
	if err := a.UnmarshalBinary(wire); err == nil {
		t.Fatal("expected error for truncated street, got nil")
	}
}

func TestAddressCityTruncated(t *testing.T) {
	t.Parallel()

	wire := buildTruncatedField(2) // city
	var a dao.Address
	if err := a.UnmarshalBinary(wire); err == nil {
		t.Fatal("expected error for truncated city, got nil")
	}
}

// --- AllScalars truncated fixed fields ---

func TestAllScalarsSfixed32Truncated(t *testing.T) {
	t.Parallel()

	wire := buildTruncatedFixed32(3) // f_sfixed32
	var a dao.AllScalars
	if err := a.UnmarshalBinary(wire); err == nil {
		t.Fatal("expected error for truncated sfixed32, got nil")
	}
}

func TestAllScalarsSfixed64Truncated(t *testing.T) {
	t.Parallel()

	wire := buildTruncatedFixed64(4) // f_sfixed64
	var a dao.AllScalars
	if err := a.UnmarshalBinary(wire); err == nil {
		t.Fatal("expected error for truncated sfixed64, got nil")
	}
}

func TestAllScalarsDoubleTruncated(t *testing.T) {
	t.Parallel()

	wire := buildTruncatedFixed64(5) // f_double
	var a dao.AllScalars
	if err := a.UnmarshalBinary(wire); err == nil {
		t.Fatal("expected error for truncated double, got nil")
	}
}

func TestAllScalarsFixed32Truncated(t *testing.T) {
	t.Parallel()

	wire := buildTruncatedFixed32(6) // f_fixed32
	var a dao.AllScalars
	if err := a.UnmarshalBinary(wire); err == nil {
		t.Fatal("expected error for truncated fixed32, got nil")
	}
}

func TestAllScalarsFixed64Truncated(t *testing.T) {
	t.Parallel()

	wire := buildTruncatedFixed64(7) // f_fixed64
	var a dao.AllScalars
	if err := a.UnmarshalBinary(wire); err == nil {
		t.Fatal("expected error for truncated fixed64, got nil")
	}
}

func TestAllScalarsFloatTruncated(t *testing.T) {
	t.Parallel()

	wire := buildTruncatedFixed32(10) // f_float
	var a dao.AllScalars
	if err := a.UnmarshalBinary(wire); err == nil {
		t.Fatal("expected error for truncated float, got nil")
	}
}

// --- AllValidate truncated fixed fields ---

func TestAllValidateFGtTruncated(t *testing.T) {
	t.Parallel()

	wire := buildTruncatedFixed32(5) // f_gt
	var a dao.AllValidate
	if err := a.UnmarshalBinary(wire); err == nil {
		t.Fatal("expected error for truncated f_gt, got nil")
	}
}

func TestAllValidateDLteTruncated(t *testing.T) {
	t.Parallel()

	wire := buildTruncatedFixed64(6) // d_lte
	var a dao.AllValidate
	if err := a.UnmarshalBinary(wire); err == nil {
		t.Fatal("expected error for truncated d_lte, got nil")
	}
}

func TestAllValidateSInTruncated(t *testing.T) {
	t.Parallel()

	wire := buildTruncatedField(7) // s_in
	var a dao.AllValidate
	if err := a.UnmarshalBinary(wire); err == nil {
		t.Fatal("expected error for truncated s_in, got nil")
	}
}

func TestAllValidateBMinmaxTruncated(t *testing.T) {
	t.Parallel()

	wire := buildTruncatedField(12) // b_minmax
	var a dao.AllValidate
	if err := a.UnmarshalBinary(wire); err == nil {
		t.Fatal("expected error for truncated b_minmax, got nil")
	}
}

// --- AllScalarsUpdate truncated fixed fields ---

func TestAllScalarsUpdateSfixed32Truncated(t *testing.T) {
	t.Parallel()

	wire := buildTruncatedFixed32(3) // f_sfixed32
	var a dao.AllScalarsUpdate
	if err := a.UnmarshalBinary(wire); err == nil {
		t.Fatal("expected error for truncated sfixed32, got nil")
	}
}

func TestAllScalarsUpdateDoubleTruncated(t *testing.T) {
	t.Parallel()

	wire := buildTruncatedFixed64(5) // f_double
	var a dao.AllScalarsUpdate
	if err := a.UnmarshalBinary(wire); err == nil {
		t.Fatal("expected error for truncated double, got nil")
	}
}

func TestAllScalarsUpdateBytesTruncated(t *testing.T) {
	t.Parallel()

	wire := buildTruncatedField(11) // f_bytes
	var a dao.AllScalarsUpdate
	if err := a.UnmarshalBinary(wire); err == nil {
		t.Fatal("expected error for truncated bytes, got nil")
	}
}

// --- AllScalarsCreate truncated fixed fields ---

func TestAllScalarsCreateSfixed32Truncated(t *testing.T) {
	t.Parallel()

	wire := buildTruncatedFixed32(3) // f_sfixed32
	var a dao.AllScalarsCreate
	if err := a.UnmarshalBinary(wire); err == nil {
		t.Fatal("expected error for truncated sfixed32, got nil")
	}
}

func TestAllScalarsCreateFixed64Truncated(t *testing.T) {
	t.Parallel()

	wire := buildTruncatedFixed64(4) // f_sfixed64
	var a dao.AllScalarsCreate
	if err := a.UnmarshalBinary(wire); err == nil {
		t.Fatal("expected error for truncated sfixed64, got nil")
	}
}

// --- PersonCreate truncated fields ---

func TestPersonCreateRatingTruncated(t *testing.T) {
	t.Parallel()

	wire := buildTruncatedFixed32(5) // rating
	var p dao.PersonCreate
	if err := p.UnmarshalBinary(wire); err == nil {
		t.Fatal("expected error for truncated rating, got nil")
	}
}

func TestPersonCreateEmailTruncated(t *testing.T) {
	t.Parallel()

	wire := buildTruncatedField(13) // email
	var p dao.PersonCreate
	if err := p.UnmarshalBinary(wire); err == nil {
		t.Fatal("expected error for truncated email, got nil")
	}
}

// --- PersonUpdateByName truncated fields ---

func TestPersonUpdateByNameRatingTruncated(t *testing.T) {
	t.Parallel()

	wire := buildTruncatedFixed32(5) // rating
	var p dao.PersonUpdateByName
	if err := p.UnmarshalBinary(wire); err == nil {
		t.Fatal("expected error for truncated rating, got nil")
	}
}

func TestPersonUpdateByNameEmailTruncated(t *testing.T) {
	t.Parallel()

	wire := buildTruncatedField(13) // email
	var p dao.PersonUpdateByName
	if err := p.UnmarshalBinary(wire); err == nil {
		t.Fatal("expected error for truncated email, got nil")
	}
}

// --- Person truncated fields ---

func TestPersonRatingTruncated(t *testing.T) {
	t.Parallel()

	wire := buildTruncatedFixed32(8) // rating (field 8 in Person)
	var p dao.Person
	if err := p.UnmarshalBinary(wire); err == nil {
		t.Fatal("expected error for truncated rating, got nil")
	}
}

func TestPersonEmailTruncated(t *testing.T) {
	t.Parallel()

	wire := buildTruncatedField(18) // email (field 18 in Person)
	var p dao.Person
	if err := p.UnmarshalBinary(wire); err == nil {
		t.Fatal("expected error for truncated email, got nil")
	}
}
