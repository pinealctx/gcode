// Package compat_test — shared test helpers.
package compat_test

import (
	"bytes"
	"testing"

	"github.com/pinealctx/gcode/testdata/compat/dao"
)

// duplicateWire concatenates wire with itself, simulating a message encoded
// twice in sequence. Used to exercise UnmarshalBinaryLenient duplicate-field
// handling: lenient mode must accept the result and apply last-one-wins.
func duplicateWire(wire []byte) []byte {
	return append(wire, wire...)
}

// assertAllValidateEqual compares two dao.AllValidate values field by field.
func assertAllValidateEqual(t *testing.T, want, got *dao.AllValidate) {
	t.Helper()
	if got.UGte != want.UGte {
		t.Errorf("UGte: got %d, want %d", got.UGte, want.UGte)
	}
	if got.ULte != want.ULte {
		t.Errorf("ULte: got %d, want %d", got.ULte, want.ULte)
	}
	if got.UIn != want.UIn {
		t.Errorf("UIn: got %d, want %d", got.UIn, want.UIn)
	}
	if got.UNotIn != want.UNotIn {
		t.Errorf("UNotIn: got %d, want %d", got.UNotIn, want.UNotIn)
	}
	if got.FGt != want.FGt {
		t.Errorf("FGt: got %v, want %v", got.FGt, want.FGt)
	}
	if got.DLte != want.DLte {
		t.Errorf("DLte: got %v, want %v", got.DLte, want.DLte)
	}
	if got.SIn != want.SIn {
		t.Errorf("SIn: got %q, want %q", got.SIn, want.SIn)
	}
	if got.SNotIn != want.SNotIn {
		t.Errorf("SNotIn: got %q, want %q", got.SNotIn, want.SNotIn)
	}
	if got.IIn != want.IIn {
		t.Errorf("IIn: got %d, want %d", got.IIn, want.IIn)
	}
	if got.SUri != want.SUri {
		t.Errorf("SUri: got %q, want %q", got.SUri, want.SUri)
	}
	if (want.OStatus == nil) != (got.OStatus == nil) {
		t.Errorf("OStatus nil mismatch: want %v, got %v", want.OStatus, got.OStatus)
	} else if want.OStatus != nil && *got.OStatus != *want.OStatus {
		t.Errorf("OStatus: got %v, want %v", *got.OStatus, *want.OStatus)
	}
	if !bytes.Equal(got.BMinmax, want.BMinmax) {
		t.Errorf("BMinmax: got %x, want %x", got.BMinmax, want.BMinmax)
	}
	if len(got.RItems) != len(want.RItems) {
		t.Errorf("RItems len: got %d, want %d", len(got.RItems), len(want.RItems))
	} else {
		for i := range want.RItems {
			if got.RItems[i] != want.RItems[i] {
				t.Errorf("RItems[%d]: got %d, want %d", i, got.RItems[i], want.RItems[i])
			}
		}
	}
	if got.EStatus != want.EStatus {
		t.Errorf("EStatus: got %v, want %v", got.EStatus, want.EStatus)
	}
}
