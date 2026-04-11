// Package compat_test — wire-format compatibility tests for AllScalars and AllRepeated.
// Covers pbgo oracle double-direction verification, error paths, and round-trip tests
// for PersonCreate/PersonUpdateByName (no pbgo counterpart).
package compat_test

import (
	"bytes"
	"errors"
	"testing"

	"google.golang.org/protobuf/proto"

	"github.com/pinealctx/gcode/runtime"
	"github.com/pinealctx/gcode/testdata/compat/dao"
	"github.com/pinealctx/gcode/testdata/compat/pbgo"
)

// --- AllScalars pbgo oracle ---

func populatedPbgoAllScalars() *pbgo.AllScalars {
	return &pbgo.AllScalars{
		FSint32:   -42,
		FSint64:   -1000000,
		FSfixed32: -7,
		FSfixed64: -999,
		FDouble:   3.14159265358979,
		FFixed32:  0xDEADBEEF,
		FFixed64:  0xCAFEBABEDEADBEEF,
		FUint32:   4294967295,
		FUint64:   18446744073709551615,
		FFloat:    2.718,
		FBytes:    []byte{0xAA, 0xBB, 0xCC},
	}
}

func populatedDaoAllScalars() *dao.AllScalars {
	return &dao.AllScalars{
		FSint32:   -42,
		FSint64:   -1000000,
		FSfixed32: -7,
		FSfixed64: -999,
		FDouble:   3.14159265358979,
		FFixed32:  0xDEADBEEF,
		FFixed64:  0xCAFEBABEDEADBEEF,
		FUint32:   4294967295,
		FUint64:   18446744073709551615,
		FFloat:    2.718,
		FBytes:    []byte{0xAA, 0xBB, 0xCC},
	}
}

// TestAllScalarsPbgoEncodesDaoDecodes: protoc-gen-go encodes AllScalars → our DAO decodes.
func TestAllScalarsPbgoEncodesDaoDecodes(t *testing.T) {
	t.Parallel()

	wire, err := proto.Marshal(populatedPbgoAllScalars())
	if err != nil {
		t.Fatalf("proto.Marshal: %v", err)
	}

	var got dao.AllScalars
	if err := got.UnmarshalBinary(wire); err != nil {
		t.Fatalf("dao.UnmarshalBinary: %v", err)
	}

	assertAllScalarsEqual(t, populatedPbgoAllScalars(), &got)
}

// TestAllScalarsDaoEncodesPbgoDecodes: our DAO encodes AllScalars → protoc-gen-go decodes.
func TestAllScalarsDaoEncodesPbgoDecodes(t *testing.T) {
	t.Parallel()

	wire, err := populatedDaoAllScalars().MarshalBinary()
	if err != nil {
		t.Fatalf("dao.MarshalBinary: %v", err)
	}

	var got pbgo.AllScalars
	if err := proto.Unmarshal(wire, &got); err != nil {
		t.Fatalf("proto.Unmarshal: %v", err)
	}

	assertAllScalarsEqual(t, &got, populatedDaoAllScalars())
}

// TestAllScalarsWireIdentical: both encoders must produce identical bytes.
func TestAllScalarsWireIdentical(t *testing.T) {
	t.Parallel()

	pbgoWire, err := proto.Marshal(populatedPbgoAllScalars())
	if err != nil {
		t.Fatalf("proto.Marshal: %v", err)
	}

	daoWire, err := populatedDaoAllScalars().MarshalBinary()
	if err != nil {
		t.Fatalf("dao.MarshalBinary: %v", err)
	}

	if !bytes.Equal(pbgoWire, daoWire) {
		t.Errorf("wire bytes differ:\n  pbgo (%d bytes): %x\n  dao  (%d bytes): %x",
			len(pbgoWire), pbgoWire, len(daoWire), daoWire)
	}
}

// TestAllScalarsZeroValueRoundTrip: zero-value AllScalars encodes to 0 bytes.
func TestAllScalarsZeroValueRoundTrip(t *testing.T) {
	t.Parallel()

	var a dao.AllScalars
	wire, err := a.MarshalBinary()
	if err != nil {
		t.Fatalf("MarshalBinary: %v", err)
	}
	if len(wire) != 0 {
		t.Errorf("zero-value AllScalars should encode to 0 bytes, got %d", len(wire))
	}

	var got dao.AllScalars
	if err := got.UnmarshalBinary(wire); err != nil {
		t.Fatalf("UnmarshalBinary empty: %v", err)
	}
}

func assertAllScalarsEqual(t *testing.T, pb *pbgo.AllScalars, d *dao.AllScalars) {
	t.Helper()
	if pb.FSint32 != d.FSint32 {
		t.Errorf("FSint32: pbgo=%d dao=%d", pb.FSint32, d.FSint32)
	}
	if pb.FSint64 != d.FSint64 {
		t.Errorf("FSint64: pbgo=%d dao=%d", pb.FSint64, d.FSint64)
	}
	if pb.FSfixed32 != d.FSfixed32 {
		t.Errorf("FSfixed32: pbgo=%d dao=%d", pb.FSfixed32, d.FSfixed32)
	}
	if pb.FSfixed64 != d.FSfixed64 {
		t.Errorf("FSfixed64: pbgo=%d dao=%d", pb.FSfixed64, d.FSfixed64)
	}
	if pb.FDouble != d.FDouble {
		t.Errorf("FDouble: pbgo=%v dao=%v", pb.FDouble, d.FDouble)
	}
	if pb.FFixed32 != d.FFixed32 {
		t.Errorf("FFixed32: pbgo=%d dao=%d", pb.FFixed32, d.FFixed32)
	}
	if pb.FFixed64 != d.FFixed64 {
		t.Errorf("FFixed64: pbgo=%d dao=%d", pb.FFixed64, d.FFixed64)
	}
	if pb.FUint32 != d.FUint32 {
		t.Errorf("FUint32: pbgo=%d dao=%d", pb.FUint32, d.FUint32)
	}
	if pb.FUint64 != d.FUint64 {
		t.Errorf("FUint64: pbgo=%d dao=%d", pb.FUint64, d.FUint64)
	}
	if pb.FFloat != d.FFloat {
		t.Errorf("FFloat: pbgo=%v dao=%v", pb.FFloat, d.FFloat)
	}
	if !bytes.Equal(pb.FBytes, d.FBytes) {
		t.Errorf("FBytes: pbgo=%x dao=%x", pb.FBytes, d.FBytes)
	}
}

// --- AllRepeated pbgo oracle ---

func populatedPbgoAllRepeated() *pbgo.AllRepeated {
	return &pbgo.AllRepeated{
		RSint32:   []int32{-1, 0, 1, -128, 127},
		RSfixed32: []int32{-2147483648, 0, 2147483647},
		RDouble:   []float64{-1.5, 0.0, 3.14},
		RBytes:    [][]byte{{0x01, 0x02}, {0x03}},
		RMessage: []*pbgo.Address{
			{Street: "Main St", City: "Springfield"},
			{Street: "", City: ""},
		},
		REnum: []pbgo.Status{pbgo.Status_STATUS_ACTIVE, pbgo.Status_STATUS_INACTIVE},
	}
}

func populatedDaoAllRepeated() *dao.AllRepeated {
	return &dao.AllRepeated{
		RSint32:   []int32{-1, 0, 1, -128, 127},
		RSfixed32: []int32{-2147483648, 0, 2147483647},
		RDouble:   []float64{-1.5, 0.0, 3.14},
		RBytes:    [][]byte{{0x01, 0x02}, {0x03}},
		RMessage: []*dao.Address{
			{Street: "Main St", City: "Springfield"},
			{Street: "", City: ""},
		},
		REnum: []dao.Status{dao.Status_STATUS_ACTIVE, dao.Status_STATUS_INACTIVE},
	}
}

// TestAllRepeatedPbgoEncodesDaoDecodes: protoc-gen-go encodes AllRepeated → our DAO decodes.
func TestAllRepeatedPbgoEncodesDaoDecodes(t *testing.T) {
	t.Parallel()

	wire, err := proto.Marshal(populatedPbgoAllRepeated())
	if err != nil {
		t.Fatalf("proto.Marshal: %v", err)
	}

	var got dao.AllRepeated
	if err := got.UnmarshalBinary(wire); err != nil {
		t.Fatalf("dao.UnmarshalBinary: %v", err)
	}

	assertAllRepeatedEqual(t, populatedPbgoAllRepeated(), &got)
}

// TestAllRepeatedDaoEncodesPbgoDecodes: our DAO encodes AllRepeated → protoc-gen-go decodes.
func TestAllRepeatedDaoEncodesPbgoDecodes(t *testing.T) {
	t.Parallel()

	wire, err := populatedDaoAllRepeated().MarshalBinary()
	if err != nil {
		t.Fatalf("dao.MarshalBinary: %v", err)
	}

	var got pbgo.AllRepeated
	if err := proto.Unmarshal(wire, &got); err != nil {
		t.Fatalf("proto.Unmarshal: %v", err)
	}

	assertAllRepeatedEqual(t, &got, populatedDaoAllRepeated())
}

// TestAllRepeatedWireIdentical: both encoders must produce identical bytes.
func TestAllRepeatedWireIdentical(t *testing.T) {
	t.Parallel()

	pbgoWire, err := proto.Marshal(populatedPbgoAllRepeated())
	if err != nil {
		t.Fatalf("proto.Marshal: %v", err)
	}

	daoWire, err := populatedDaoAllRepeated().MarshalBinary()
	if err != nil {
		t.Fatalf("dao.MarshalBinary: %v", err)
	}

	if !bytes.Equal(pbgoWire, daoWire) {
		t.Errorf("wire bytes differ:\n  pbgo (%d bytes): %x\n  dao  (%d bytes): %x",
			len(pbgoWire), pbgoWire, len(daoWire), daoWire)
	}
}

// TestAllRepeatedZeroValueRoundTrip: zero-value AllRepeated encodes to 0 bytes.
func TestAllRepeatedZeroValueRoundTrip(t *testing.T) {
	t.Parallel()

	var a dao.AllRepeated
	wire, err := a.MarshalBinary()
	if err != nil {
		t.Fatalf("MarshalBinary: %v", err)
	}
	if len(wire) != 0 {
		t.Errorf("zero-value AllRepeated should encode to 0 bytes, got %d", len(wire))
	}
}

func assertAllRepeatedEqual(t *testing.T, pb *pbgo.AllRepeated, d *dao.AllRepeated) {
	t.Helper()

	if len(pb.RSint32) != len(d.RSint32) {
		t.Errorf("RSint32 len: pbgo=%d dao=%d", len(pb.RSint32), len(d.RSint32))
	} else {
		for i := range pb.RSint32 {
			if pb.RSint32[i] != d.RSint32[i] {
				t.Errorf("RSint32[%d]: pbgo=%d dao=%d", i, pb.RSint32[i], d.RSint32[i])
			}
		}
	}

	if len(pb.RSfixed32) != len(d.RSfixed32) {
		t.Errorf("RSfixed32 len: pbgo=%d dao=%d", len(pb.RSfixed32), len(d.RSfixed32))
	} else {
		for i := range pb.RSfixed32 {
			if pb.RSfixed32[i] != d.RSfixed32[i] {
				t.Errorf("RSfixed32[%d]: pbgo=%d dao=%d", i, pb.RSfixed32[i], d.RSfixed32[i])
			}
		}
	}

	if len(pb.RDouble) != len(d.RDouble) {
		t.Errorf("RDouble len: pbgo=%d dao=%d", len(pb.RDouble), len(d.RDouble))
	} else {
		for i := range pb.RDouble {
			if pb.RDouble[i] != d.RDouble[i] {
				t.Errorf("RDouble[%d]: pbgo=%v dao=%v", i, pb.RDouble[i], d.RDouble[i])
			}
		}
	}

	if len(pb.RBytes) != len(d.RBytes) {
		t.Errorf("RBytes len: pbgo=%d dao=%d", len(pb.RBytes), len(d.RBytes))
	} else {
		for i := range pb.RBytes {
			if !bytes.Equal(pb.RBytes[i], d.RBytes[i]) {
				t.Errorf("RBytes[%d]: pbgo=%x dao=%x", i, pb.RBytes[i], d.RBytes[i])
			}
		}
	}

	if len(pb.RMessage) != len(d.RMessage) {
		t.Errorf("RMessage len: pbgo=%d dao=%d", len(pb.RMessage), len(d.RMessage))
	} else {
		for i := range pb.RMessage {
			if pb.RMessage[i].Street != d.RMessage[i].Street {
				t.Errorf("RMessage[%d].Street: pbgo=%q dao=%q", i, pb.RMessage[i].Street, d.RMessage[i].Street)
			}
			if pb.RMessage[i].City != d.RMessage[i].City {
				t.Errorf("RMessage[%d].City: pbgo=%q dao=%q", i, pb.RMessage[i].City, d.RMessage[i].City)
			}
		}
	}

	if len(pb.REnum) != len(d.REnum) {
		t.Errorf("REnum len: pbgo=%d dao=%d", len(pb.REnum), len(d.REnum))
	} else {
		for i := range pb.REnum {
			if int32(pb.REnum[i]) != int32(d.REnum[i]) {
				t.Errorf("REnum[%d]: pbgo=%d dao=%d", i, pb.REnum[i], d.REnum[i])
			}
		}
	}
}

// --- Error path tests ---

// TestErrOverflow verifies that a varint exceeding 10 bytes returns ErrOverflow.
func TestErrOverflow(t *testing.T) {
	t.Parallel()

	// Construct a varint with 11 continuation bytes (all 0x80) — exceeds 10-byte limit.
	// Field 2 in dao.Person is "age" (int32, WireVarint). Tag = (2<<3)|0 = 0x10.
	wire := []byte{
		0x10,                                                             // tag: field 2 (age), WireVarint
		0x80, 0x80, 0x80, 0x80, 0x80, 0x80, 0x80, 0x80, 0x80, 0x80, 0x01, // 11-byte varint
	}

	var p dao.Person
	err := p.UnmarshalBinary(wire)
	if err == nil {
		t.Fatal("expected ErrOverflow, got nil")
	}
	if !errors.Is(err, runtime.ErrOverflow) {
		t.Errorf("expected ErrOverflow, got: %v", err)
	}
}

// TestErrUnknownWireType verifies that an unknown wire type triggers ErrUnknownWireType.
func TestErrUnknownWireType(t *testing.T) {
	t.Parallel()

	// Wire type 6 is reserved/unknown in proto3.
	// Tag = (field_num << 3) | wire_type. Use field 100, wire type 6: (100<<3)|6 = 806.
	// Encode 806 as varint: 806 = 0x326, varint = 0xa6 0x06
	wire := []byte{0xa6, 0x06}

	var p dao.Person
	err := p.UnmarshalBinary(wire)
	if err == nil {
		t.Fatal("expected ErrUnknownWireType, got nil")
	}
	if !errors.Is(err, runtime.ErrUnknownWireType) {
		t.Errorf("expected ErrUnknownWireType, got: %v", err)
	}
}

// TestErrPackedLenFixed32 verifies that a packed fixed32 field with payload length
// not divisible by 4 returns ErrPackedLen.
func TestErrPackedLenFixed32(t *testing.T) {
	t.Parallel()

	// AllRepeated field 2 = r_sfixed32 (packed fixed32).
	// Tag = (2<<3)|2 = 18 = 0x12. Payload length = 3 (not divisible by 4).
	wire := []byte{
		0x12,             // tag: field 2, WireBytes
		0x03,             // length: 3
		0x01, 0x02, 0x03, // 3 bytes — not a multiple of 4
	}

	var a dao.AllRepeated
	err := a.UnmarshalBinary(wire)
	if err == nil {
		t.Fatal("expected ErrPackedLen for fixed32, got nil")
	}
	if !errors.Is(err, runtime.ErrPackedLen) {
		t.Errorf("expected ErrPackedLen, got: %v", err)
	}
}

// TestErrPackedLenFixed64 verifies that a packed fixed64 field with payload length
// not divisible by 8 returns ErrPackedLen.
func TestErrPackedLenFixed64(t *testing.T) {
	t.Parallel()

	// AllRepeated field 3 = r_double (packed fixed64).
	// Tag = (3<<3)|2 = 26 = 0x1a. Payload length = 5 (not divisible by 8).
	wire := []byte{
		0x1a,                         // tag: field 3, WireBytes
		0x05,                         // length: 5
		0x01, 0x02, 0x03, 0x04, 0x05, // 5 bytes — not a multiple of 8
	}

	var a dao.AllRepeated
	err := a.UnmarshalBinary(wire)
	if err == nil {
		t.Fatal("expected ErrPackedLen for fixed64, got nil")
	}
	if !errors.Is(err, runtime.ErrPackedLen) {
		t.Errorf("expected ErrPackedLen, got: %v", err)
	}
}

// --- PersonCreate / PersonUpdateByName round-trip ---

// TestPersonCreateRoundTrip verifies MarshalBinary → UnmarshalBinary for PersonCreate.
func TestPersonCreateRoundTrip(t *testing.T) {
	t.Parallel()

	name := "Alice"
	age := int32(30)
	active := true
	status := dao.Status_STATUS_ACTIVE
	rating := float32(4.5)
	orig := &dao.PersonCreate{
		Name:     &name,
		Age:      &age,
		Active:   &active,
		Status:   &status,
		Rating:   &rating,
		Nickname: "ali",
	}

	wire, err := orig.MarshalBinary()
	if err != nil {
		t.Fatalf("MarshalBinary: %v", err)
	}

	var got dao.PersonCreate
	if err := got.UnmarshalBinary(wire); err != nil {
		t.Fatalf("UnmarshalBinary: %v", err)
	}

	if got.Nickname != orig.Nickname {
		t.Errorf("Nickname: got %q, want %q", got.Nickname, orig.Nickname)
	}
	if got.Name == nil || *got.Name != *orig.Name {
		t.Errorf("Name: got %v, want %v", got.Name, orig.Name)
	}
	if got.Age == nil || *got.Age != *orig.Age {
		t.Errorf("Age: got %v, want %v", got.Age, orig.Age)
	}
	if got.Active == nil || *got.Active != *orig.Active {
		t.Errorf("Active: got %v, want %v", got.Active, orig.Active)
	}
	if got.Status == nil || *got.Status != *orig.Status {
		t.Errorf("Status: got %v, want %v", got.Status, orig.Status)
	}
	if got.Rating == nil || *got.Rating != *orig.Rating {
		t.Errorf("Rating: got %v, want %v", got.Rating, orig.Rating)
	}
}

// TestPersonCreateNilFieldsRoundTrip verifies that nil optional fields survive round-trip.
func TestPersonCreateNilFieldsRoundTrip(t *testing.T) {
	t.Parallel()

	orig := &dao.PersonCreate{Nickname: "ali"}
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
	if got.Name != nil {
		t.Errorf("Name: got %v, want nil", got.Name)
	}
	if got.Age != nil {
		t.Errorf("Age: got %v, want nil", got.Age)
	}
}

// TestPersonUpdateByNameRoundTrip verifies MarshalBinary → UnmarshalBinary for PersonUpdateByName.
func TestPersonUpdateByNameRoundTrip(t *testing.T) {
	t.Parallel()

	age := int32(25)
	active := false
	email := "bob@example.com"
	orig := &dao.PersonUpdateByName{
		Name:   "Bob",
		Age:    &age,
		Active: &active,
		Email:  &email,
	}

	wire, err := orig.MarshalBinary()
	if err != nil {
		t.Fatalf("MarshalBinary: %v", err)
	}

	var got dao.PersonUpdateByName
	if err := got.UnmarshalBinary(wire); err != nil {
		t.Fatalf("UnmarshalBinary: %v", err)
	}

	if got.Name != orig.Name {
		t.Errorf("Name: got %q, want %q", got.Name, orig.Name)
	}
	if got.Age == nil || *got.Age != *orig.Age {
		t.Errorf("Age: got %v, want %v", got.Age, orig.Age)
	}
	if got.Active == nil || *got.Active != *orig.Active {
		t.Errorf("Active: got %v, want %v", got.Active, orig.Active)
	}
	if got.Email == nil || *got.Email != *orig.Email {
		t.Errorf("Email: got %v, want %v", got.Email, orig.Email)
	}
}

// TestPersonUpdateByNameNilFieldsRoundTrip verifies nil optional fields survive round-trip.
func TestPersonUpdateByNameNilFieldsRoundTrip(t *testing.T) {
	t.Parallel()

	orig := &dao.PersonUpdateByName{Name: "Alice"}
	wire, err := orig.MarshalBinary()
	if err != nil {
		t.Fatalf("MarshalBinary: %v", err)
	}

	var got dao.PersonUpdateByName
	if err := got.UnmarshalBinary(wire); err != nil {
		t.Fatalf("UnmarshalBinary: %v", err)
	}

	if got.Name != "Alice" {
		t.Errorf("Name: got %q, want %q", got.Name, "Alice")
	}
	if got.Age != nil {
		t.Errorf("Age: got %v, want nil", got.Age)
	}
	if got.Email != nil {
		t.Errorf("Email: got %v, want nil", got.Email)
	}
}

// --- Fuzz seeds for AllScalars and AllRepeated ---

// FuzzAllScalarsUnmarshalBinary verifies AllScalars.UnmarshalBinary never panics,
// and that any input accepted by proto.Unmarshal is also accepted in lenient mode.
func FuzzAllScalarsUnmarshalBinary(f *testing.F) {
	wire, _ := populatedDaoAllScalars().MarshalBinary()
	f.Add(wire)

	pbgoWire, _ := proto.Marshal(populatedPbgoAllScalars())
	f.Add(pbgoWire)

	f.Add([]byte{})

	f.Fuzz(func(t *testing.T, data []byte) {
		var d dao.AllScalars
		_ = d.UnmarshalBinaryLenient(data)

		var pb pbgo.AllScalars
		if err := proto.Unmarshal(data, &pb); err == nil {
			var d2 dao.AllScalars
			if err2 := d2.UnmarshalBinaryLenient(data); err2 != nil {
				t.Errorf("proto.Unmarshal accepted but dao rejected: %v\ninput: %x", err2, data)
			}
		}
	})
}

// FuzzAllRepeatedUnmarshalBinary verifies AllRepeated.UnmarshalBinary never panics,
// and that any input accepted by proto.Unmarshal is also accepted in lenient mode.
func FuzzAllRepeatedUnmarshalBinary(f *testing.F) {
	wire, _ := populatedDaoAllRepeated().MarshalBinary()
	f.Add(wire)

	pbgoWire, _ := proto.Marshal(populatedPbgoAllRepeated())
	f.Add(pbgoWire)

	f.Add([]byte{})

	f.Fuzz(func(t *testing.T, data []byte) {
		var d dao.AllRepeated
		_ = d.UnmarshalBinaryLenient(data)

		var pb pbgo.AllRepeated
		if err := proto.Unmarshal(data, &pb); err == nil {
			var d2 dao.AllRepeated
			if err2 := d2.UnmarshalBinaryLenient(data); err2 != nil {
				t.Errorf("proto.Unmarshal accepted but dao rejected: %v\ninput: %x", err2, data)
			}
		}
	})
}
