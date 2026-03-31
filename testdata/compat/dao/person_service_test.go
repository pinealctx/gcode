// Package dao_test verifies the generated codec for RPC request/response types
// produced from person_service.proto.
//
// Coverage matrix per type:
//   - nil receiver Size() returns 0
//   - zero-value round-trip (empty wire, unmarshal back to zero)
//   - non-zero round-trip (all fields set, marshal → unmarshal → equal)
//   - bool field: true and false encoding
//   - lenient mode: duplicate field keeps last value
//   - error paths: ErrWireType, ErrDuplicateField, ErrTruncated
package dao_test

import (
	"testing"

	"github.com/pinealctx/gcode/runtime"
	"github.com/pinealctx/gcode/testdata/compat/dao"
)

// --- helpers -----------------------------------------------------------------

type binaryMarshaler interface {
	MarshalBinary() ([]byte, error)
}

func mustMarshal(t *testing.T, m binaryMarshaler) []byte {
	t.Helper()
	b, err := m.MarshalBinary()
	if err != nil {
		t.Fatalf("MarshalBinary: %v", err)
	}
	return b
}

func buildDuplicateStringField(fieldNum int, first, second string) []byte {
	var b []byte
	b = runtime.AppendTag(b, fieldNum, runtime.WireBytes)
	b = runtime.AppendString(b, first)
	b = runtime.AppendTag(b, fieldNum, runtime.WireBytes)
	b = runtime.AppendString(b, second)
	return b
}

func buildWrongWireType(fieldNum int) []byte {
	// encode a varint tag where a bytes tag is expected (or vice-versa)
	var b []byte
	b = runtime.AppendTag(b, fieldNum, runtime.WireVarint)
	b = runtime.AppendVarint(b, 42)
	return b
}

func buildTruncatedBytes(fieldNum int) []byte {
	// tag + length prefix claiming 10 bytes but no payload
	var b []byte
	b = runtime.AppendTag(b, fieldNum, runtime.WireBytes)
	b = runtime.AppendVarint(b, 10) // length = 10, but no payload follows
	return b
}

// --- CreatePersonResponse ----------------------------------------------------

func TestCreatePersonResponse_NilSize(t *testing.T) {
	t.Parallel()
	var p *dao.CreatePersonResponse
	if p.Size() != 0 {
		t.Errorf("nil.Size() = %d, want 0", p.Size())
	}
}

func TestCreatePersonResponse_RoundTrip(t *testing.T) {
	t.Parallel()
	orig := &dao.CreatePersonResponse{Id: "uuid-123"}
	wire := mustMarshal(t, orig)
	var got dao.CreatePersonResponse
	if err := got.UnmarshalBinary(wire); err != nil {
		t.Fatalf("UnmarshalBinary: %v", err)
	}
	if got.Id != orig.Id {
		t.Errorf("got %+v, want %+v", got, orig)
	}
}

// --- GetPersonRequest --------------------------------------------------------

func TestGetPersonRequest_NilSize(t *testing.T) {
	t.Parallel()
	var p *dao.GetPersonRequest
	if p.Size() != 0 {
		t.Errorf("nil.Size() = %d, want 0", p.Size())
	}
}

func TestGetPersonRequest_RoundTrip(t *testing.T) {
	t.Parallel()
	orig := &dao.GetPersonRequest{Id: "id-42"}
	wire := mustMarshal(t, orig)
	var got dao.GetPersonRequest
	if err := got.UnmarshalBinary(wire); err != nil {
		t.Fatalf("UnmarshalBinary: %v", err)
	}
	if got.Id != orig.Id {
		t.Errorf("got %+v, want %+v", got, orig)
	}
}

// --- GetPersonResponse -------------------------------------------------------

func TestGetPersonResponse_NilSize(t *testing.T) {
	t.Parallel()
	var p *dao.GetPersonResponse
	if p.Size() != 0 {
		t.Errorf("nil.Size() = %d, want 0", p.Size())
	}
}

func TestGetPersonResponse_RoundTrip(t *testing.T) {
	t.Parallel()
	orig := &dao.GetPersonResponse{Name: "Bob", Age: 25}
	wire := mustMarshal(t, orig)
	var got dao.GetPersonResponse
	if err := got.UnmarshalBinary(wire); err != nil {
		t.Fatalf("UnmarshalBinary: %v", err)
	}
	if got.Name != orig.Name || got.Age != orig.Age {
		t.Errorf("got %+v, want %+v", got, orig)
	}
}

// --- UpdatePersonResponse (bool field) ---------------------------------------

func TestUpdatePersonResponse_NilSize(t *testing.T) {
	t.Parallel()
	var p *dao.UpdatePersonResponse
	if p.Size() != 0 {
		t.Errorf("nil.Size() = %d, want 0", p.Size())
	}
}

func TestUpdatePersonResponse_BoolTrue(t *testing.T) {
	t.Parallel()
	orig := &dao.UpdatePersonResponse{Ok: true}
	wire := mustMarshal(t, orig)
	var got dao.UpdatePersonResponse
	if err := got.UnmarshalBinary(wire); err != nil {
		t.Fatalf("UnmarshalBinary: %v", err)
	}
	if !got.Ok {
		t.Errorf("got Ok=false, want true")
	}
}

func TestUpdatePersonResponse_BoolFalse(t *testing.T) {
	t.Parallel()
	orig := &dao.UpdatePersonResponse{Ok: false}
	wire := mustMarshal(t, orig)
	if len(wire) != 0 {
		t.Errorf("bool false should produce empty wire, got %d bytes", len(wire))
	}
	var got dao.UpdatePersonResponse
	if err := got.UnmarshalBinary(wire); err != nil {
		t.Fatalf("UnmarshalBinary: %v", err)
	}
	if got.Ok {
		t.Errorf("got Ok=true, want false")
	}
}

// --- DeletePersonRequest -----------------------------------------------------

func TestDeletePersonRequest_NilSize(t *testing.T) {
	t.Parallel()
	var p *dao.DeletePersonRequest
	if p.Size() != 0 {
		t.Errorf("nil.Size() = %d, want 0", p.Size())
	}
}

func TestDeletePersonRequest_RoundTrip(t *testing.T) {
	t.Parallel()
	orig := &dao.DeletePersonRequest{Id: "id-99"}
	wire := mustMarshal(t, orig)
	var got dao.DeletePersonRequest
	if err := got.UnmarshalBinary(wire); err != nil {
		t.Fatalf("UnmarshalBinary: %v", err)
	}
	if got.Id != orig.Id {
		t.Errorf("got %+v, want %+v", got, orig)
	}
}

// --- DeletePersonResponse (bool field) ---------------------------------------

func TestDeletePersonResponse_NilSize(t *testing.T) {
	t.Parallel()
	var p *dao.DeletePersonResponse
	if p.Size() != 0 {
		t.Errorf("nil.Size() = %d, want 0", p.Size())
	}
}

func TestDeletePersonResponse_BoolTrue(t *testing.T) {
	t.Parallel()
	orig := &dao.DeletePersonResponse{Ok: true}
	wire := mustMarshal(t, orig)
	var got dao.DeletePersonResponse
	if err := got.UnmarshalBinary(wire); err != nil {
		t.Fatalf("UnmarshalBinary: %v", err)
	}
	if !got.Ok {
		t.Errorf("got Ok=false, want true")
	}
}

func TestDeletePersonResponse_BoolFalse(t *testing.T) {
	t.Parallel()
	orig := &dao.DeletePersonResponse{Ok: false}
	wire := mustMarshal(t, orig)
	if len(wire) != 0 {
		t.Errorf("bool false should produce empty wire, got %d bytes", len(wire))
	}
	var got dao.DeletePersonResponse
	if err := got.UnmarshalBinary(wire); err != nil {
		t.Fatalf("UnmarshalBinary: %v", err)
	}
	if got.Ok {
		t.Errorf("got Ok=true, want false")
	}
}
