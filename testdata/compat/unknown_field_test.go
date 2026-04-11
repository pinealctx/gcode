// Package compat_test — additional error path tests for person_service types
// and unknown field skip paths to boost unmarshalFrom coverage.
package compat_test

import (
	"errors"
	"testing"

	"github.com/pinealctx/gcode/runtime"
	"github.com/pinealctx/gcode/testdata/compat/dao"
)

// buildUnknownField builds wire bytes with an unknown field number (triggers default/SkipField).
func buildUnknownField() []byte {
	var b []byte
	// Field 999, WireVarint — unknown to all our types
	b = runtime.AppendTag(b, 999, runtime.WireVarint)
	b = runtime.AppendVarint(b, 42)
	return b
}

// --- Unknown field skip (default branch) ---

func TestPersonUnknownFieldSkipped(t *testing.T) {
	t.Parallel()

	wire := buildUnknownField()
	var p dao.Person
	if err := p.UnmarshalBinary(wire); err != nil {
		t.Errorf("unknown field should be skipped, got: %v", err)
	}
}

func TestAllScalarsUnknownFieldSkipped(t *testing.T) {
	t.Parallel()

	wire := buildUnknownField()
	var a dao.AllScalars
	if err := a.UnmarshalBinary(wire); err != nil {
		t.Errorf("unknown field should be skipped, got: %v", err)
	}
}

func TestAllRepeatedUnknownFieldSkipped(t *testing.T) {
	t.Parallel()

	wire := buildUnknownField()
	var a dao.AllRepeated
	if err := a.UnmarshalBinary(wire); err != nil {
		t.Errorf("unknown field should be skipped, got: %v", err)
	}
}

func TestAllValidateUnknownFieldSkipped(t *testing.T) {
	t.Parallel()

	wire := buildUnknownField()
	var a dao.AllValidate
	if err := a.UnmarshalBinary(wire); err != nil {
		t.Errorf("unknown field should be skipped, got: %v", err)
	}
}

func TestAllScalarsUpdateUnknownFieldSkipped(t *testing.T) {
	t.Parallel()

	wire := buildUnknownField()
	var a dao.AllScalarsUpdate
	if err := a.UnmarshalBinary(wire); err != nil {
		t.Errorf("unknown field should be skipped, got: %v", err)
	}
}

func TestAllScalarsCreateUnknownFieldSkipped(t *testing.T) {
	t.Parallel()

	wire := buildUnknownField()
	var a dao.AllScalarsCreate
	if err := a.UnmarshalBinary(wire); err != nil {
		t.Errorf("unknown field should be skipped, got: %v", err)
	}
}

func TestPersonCreateUnknownFieldSkipped(t *testing.T) {
	t.Parallel()

	wire := buildUnknownField()
	var p dao.PersonCreate
	if err := p.UnmarshalBinary(wire); err != nil {
		t.Errorf("unknown field should be skipped, got: %v", err)
	}
}

func TestPersonUpdateByNameUnknownFieldSkipped(t *testing.T) {
	t.Parallel()

	wire := buildUnknownField()
	var p dao.PersonUpdateByName
	if err := p.UnmarshalBinary(wire); err != nil {
		t.Errorf("unknown field should be skipped, got: %v", err)
	}
}

// --- person_service types duplicate field rejection ---

func TestCreatePersonResponseDuplicateField(t *testing.T) {
	t.Parallel()

	wire := buildDupField(1, runtime.WireBytes, func(b []byte) []byte {
		return runtime.AppendString(b, "uuid-1")
	})

	var c dao.CreatePersonResponse
	err := c.UnmarshalBinary(wire)
	if err == nil {
		t.Fatal("expected ErrDuplicateField, got nil")
	}
	if !errors.Is(err, runtime.ErrDuplicateField) {
		t.Errorf("expected ErrDuplicateField, got: %v", err)
	}
}

func TestGetPersonRequestDuplicateField(t *testing.T) {
	t.Parallel()

	wire := buildDupField(1, runtime.WireBytes, func(b []byte) []byte {
		return runtime.AppendString(b, "id-1")
	})

	var g dao.GetPersonRequest
	err := g.UnmarshalBinary(wire)
	if err == nil {
		t.Fatal("expected ErrDuplicateField, got nil")
	}
	if !errors.Is(err, runtime.ErrDuplicateField) {
		t.Errorf("expected ErrDuplicateField, got: %v", err)
	}
}

func TestGetPersonResponseDuplicateField(t *testing.T) {
	t.Parallel()

	wire := buildDupField(1, runtime.WireBytes, func(b []byte) []byte {
		return runtime.AppendString(b, "Alice")
	})

	var g dao.GetPersonResponse
	err := g.UnmarshalBinary(wire)
	if err == nil {
		t.Fatal("expected ErrDuplicateField, got nil")
	}
	if !errors.Is(err, runtime.ErrDuplicateField) {
		t.Errorf("expected ErrDuplicateField, got: %v", err)
	}
}

func TestDeletePersonRequestDuplicateField(t *testing.T) {
	t.Parallel()

	wire := buildDupField(1, runtime.WireBytes, func(b []byte) []byte {
		return runtime.AppendString(b, "id-1")
	})

	var d dao.DeletePersonRequest
	err := d.UnmarshalBinary(wire)
	if err == nil {
		t.Fatal("expected ErrDuplicateField, got nil")
	}
	if !errors.Is(err, runtime.ErrDuplicateField) {
		t.Errorf("expected ErrDuplicateField, got: %v", err)
	}
}

// --- person_service types unknown field skip ---

func TestCreatePersonResponseUnknownField(t *testing.T) {
	t.Parallel()

	wire := buildUnknownField()
	var c dao.CreatePersonResponse
	if err := c.UnmarshalBinary(wire); err != nil {
		t.Errorf("unknown field should be skipped, got: %v", err)
	}
}

func TestGetPersonRequestUnknownField(t *testing.T) {
	t.Parallel()

	wire := buildUnknownField()
	var g dao.GetPersonRequest
	if err := g.UnmarshalBinary(wire); err != nil {
		t.Errorf("unknown field should be skipped, got: %v", err)
	}
}

func TestGetPersonResponseUnknownField(t *testing.T) {
	t.Parallel()

	wire := buildUnknownField()
	var g dao.GetPersonResponse
	if err := g.UnmarshalBinary(wire); err != nil {
		t.Errorf("unknown field should be skipped, got: %v", err)
	}
}

func TestUpdatePersonResponseUnknownField(t *testing.T) {
	t.Parallel()

	wire := buildUnknownField()
	var u dao.UpdatePersonResponse
	if err := u.UnmarshalBinary(wire); err != nil {
		t.Errorf("unknown field should be skipped, got: %v", err)
	}
}

func TestDeletePersonRequestUnknownField(t *testing.T) {
	t.Parallel()

	wire := buildUnknownField()
	var d dao.DeletePersonRequest
	if err := d.UnmarshalBinary(wire); err != nil {
		t.Errorf("unknown field should be skipped, got: %v", err)
	}
}

func TestDeletePersonResponseUnknownField(t *testing.T) {
	t.Parallel()

	wire := buildUnknownField()
	var d dao.DeletePersonResponse
	if err := d.UnmarshalBinary(wire); err != nil {
		t.Errorf("unknown field should be skipped, got: %v", err)
	}
}

// --- person_service types wire type errors ---

func TestCreatePersonResponseWireTypeError(t *testing.T) {
	t.Parallel()

	// Field 1 = id (WireBytes). Encode as WireVarint.
	var wire []byte
	wire = runtime.AppendTag(wire, 1, runtime.WireVarint)
	wire = runtime.AppendVarint(wire, 42)

	var c dao.CreatePersonResponse
	err := c.UnmarshalBinary(wire)
	if err == nil {
		t.Fatal("expected error for wrong wire type, got nil")
	}
}

func TestGetPersonRequestWireTypeError(t *testing.T) {
	t.Parallel()

	var wire []byte
	wire = runtime.AppendTag(wire, 1, runtime.WireVarint)
	wire = runtime.AppendVarint(wire, 42)

	var g dao.GetPersonRequest
	err := g.UnmarshalBinary(wire)
	if err == nil {
		t.Fatal("expected error for wrong wire type, got nil")
	}
}

func TestGetPersonResponseWireTypeError(t *testing.T) {
	t.Parallel()

	var wire []byte
	wire = runtime.AppendTag(wire, 1, runtime.WireVarint)
	wire = runtime.AppendVarint(wire, 42)

	var g dao.GetPersonResponse
	err := g.UnmarshalBinary(wire)
	if err == nil {
		t.Fatal("expected error for wrong wire type, got nil")
	}
}

func TestDeletePersonRequestWireTypeError(t *testing.T) {
	t.Parallel()

	var wire []byte
	wire = runtime.AppendTag(wire, 1, runtime.WireVarint)
	wire = runtime.AppendVarint(wire, 42)

	var d dao.DeletePersonRequest
	err := d.UnmarshalBinary(wire)
	if err == nil {
		t.Fatal("expected error for wrong wire type, got nil")
	}
}

// --- Address duplicate field rejection ---

func TestAddressDuplicateField(t *testing.T) {
	t.Parallel()

	wire := buildDupField(1, runtime.WireBytes, func(b []byte) []byte {
		return runtime.AppendString(b, "Main St")
	})

	var a dao.Address
	err := a.UnmarshalBinary(wire)
	if err == nil {
		t.Fatal("expected ErrDuplicateField, got nil")
	}
	if !errors.Is(err, runtime.ErrDuplicateField) {
		t.Errorf("expected ErrDuplicateField, got: %v", err)
	}
}

func TestAddressUnknownField(t *testing.T) {
	t.Parallel()

	wire := buildUnknownField()
	var a dao.Address
	if err := a.UnmarshalBinary(wire); err != nil {
		t.Errorf("unknown field should be skipped, got: %v", err)
	}
}
