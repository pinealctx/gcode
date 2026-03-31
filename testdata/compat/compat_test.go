// Package compat_test verifies binary wire-format compatibility between
// protoc-gen-go (pbgo) and our generated DAO codec (dao).
//
// Two directions are tested for each case:
//  1. pbgo.Marshal → dao.UnmarshalBinary  (protoc-gen-go encodes, we decode)
//  2. dao.MarshalBinary → pbgo.Unmarshal  (we encode, protoc-gen-go decodes)
package compat_test

import (
	"bytes"
	"testing"

	"google.golang.org/protobuf/proto"

	"github.com/pinealctx/gcode/testdata/compat/dao"
	"github.com/pinealctx/gcode/testdata/compat/pbgo"
)

// populatedPbgo returns a fully-populated pbgo.Person for encoding.
func populatedPbgo() *pbgo.Person {
	return &pbgo.Person{
		Name:      "Alice",
		Age:       30,
		Active:    true,
		Status:    pbgo.Status_STATUS_ACTIVE,
		Address:   &pbgo.Address{Street: "123 Main St", City: "Springfield"},
		Scores:    []int32{10, 20, 30},
		Tags:      []string{"go", "proto"},
		Rating:    4.5,
		CreatedAt: 1700000000,
		Avatar:    []byte{0x01, 0x02, 0x03},
	}
}

// populatedDao returns a fully-populated dao.Person for encoding.
func populatedDao() *dao.Person {
	return &dao.Person{
		Name:      "Alice",
		Age:       30,
		Active:    true,
		Status:    dao.Status_STATUS_ACTIVE,
		Address:   &dao.Address{Street: "123 Main St", City: "Springfield"},
		Scores:    []int32{10, 20, 30},
		Tags:      []string{"go", "proto"},
		Rating:    4.5,
		CreatedAt: 1700000000,
		Avatar:    []byte{0x01, 0x02, 0x03},
	}
}

// TestPbgoEncodesDaoDecodes: protoc-gen-go encodes → our DAO decodes.
func TestPbgoEncodesDaoDecodes(t *testing.T) {
	t.Parallel()

	wire, err := proto.Marshal(populatedPbgo())
	if err != nil {
		t.Fatalf("proto.Marshal: %v", err)
	}

	var got dao.Person
	if err := got.UnmarshalBinary(wire); err != nil {
		t.Fatalf("dao.UnmarshalBinary: %v", err)
	}

	assertPersonEqual(t, populatedPbgo(), &got)
}

// TestDaoEncodesPbgoDecodes: our DAO encodes → protoc-gen-go decodes.
func TestDaoEncodesPbgoDecodes(t *testing.T) {
	t.Parallel()

	wire, err := populatedDao().MarshalBinary()
	if err != nil {
		t.Fatalf("dao.MarshalBinary: %v", err)
	}

	var got pbgo.Person
	if err := proto.Unmarshal(wire, &got); err != nil {
		t.Fatalf("proto.Unmarshal: %v", err)
	}

	assertPersonEqual(t, &got, populatedDao())
}

// TestRoundTripDaoOnly: dao.MarshalBinary → dao.UnmarshalBinary.
func TestRoundTripDaoOnly(t *testing.T) {
	t.Parallel()

	orig := populatedDao()
	wire, err := orig.MarshalBinary()
	if err != nil {
		t.Fatalf("MarshalBinary: %v", err)
	}

	var got dao.Person
	if err := got.UnmarshalBinary(wire); err != nil {
		t.Fatalf("UnmarshalBinary: %v", err)
	}

	assertPersonEqual(t, populatedPbgo(), &got)
}

// TestZeroValueRoundTrip: empty message encodes to zero bytes and decodes cleanly.
func TestZeroValueRoundTrip(t *testing.T) {
	t.Parallel()

	var p dao.Person
	wire, err := p.MarshalBinary()
	if err != nil {
		t.Fatalf("MarshalBinary: %v", err)
	}
	if len(wire) != 0 {
		t.Errorf("zero-value Person should encode to 0 bytes, got %d", len(wire))
	}

	var got dao.Person
	if err := got.UnmarshalBinary(wire); err != nil {
		t.Fatalf("UnmarshalBinary empty: %v", err)
	}
}

// TestAddressRoundTrip: nested message encodes/decodes correctly.
func TestAddressRoundTrip(t *testing.T) {
	t.Parallel()

	orig := &dao.Address{Street: "42 Elm St", City: "Shelbyville"}
	wire, err := orig.MarshalBinary()
	if err != nil {
		t.Fatalf("MarshalBinary: %v", err)
	}

	var got dao.Address
	if err := got.UnmarshalBinary(wire); err != nil {
		t.Fatalf("UnmarshalBinary: %v", err)
	}

	if got.Street != orig.Street {
		t.Errorf("Street: got %q, want %q", got.Street, orig.Street)
	}
	if got.City != orig.City {
		t.Errorf("City: got %q, want %q", got.City, orig.City)
	}
}

// TestWireIdentical: both encoders must produce identical bytes for the same data.
func TestWireIdentical(t *testing.T) {
	t.Parallel()

	pbgoWire, err := proto.Marshal(populatedPbgo())
	if err != nil {
		t.Fatalf("proto.Marshal: %v", err)
	}

	daoWire, err := populatedDao().MarshalBinary()
	if err != nil {
		t.Fatalf("dao.MarshalBinary: %v", err)
	}

	if !bytes.Equal(pbgoWire, daoWire) {
		t.Errorf("wire bytes differ:\n  pbgo (%d bytes): %x\n  dao  (%d bytes): %x",
			len(pbgoWire), pbgoWire, len(daoWire), daoWire)
	}
}

// assertPersonEqual compares a pbgo.Person against a dao.Person field by field.
func assertPersonEqual(t *testing.T, pb *pbgo.Person, d *dao.Person) {
	t.Helper()

	if pb.Name != d.Name {
		t.Errorf("Name: pbgo=%q dao=%q", pb.Name, d.Name)
	}
	if pb.Age != d.Age {
		t.Errorf("Age: pbgo=%d dao=%d", pb.Age, d.Age)
	}
	if pb.Active != d.Active {
		t.Errorf("Active: pbgo=%v dao=%v", pb.Active, d.Active)
	}
	if int32(pb.Status) != int32(d.Status) {
		t.Errorf("Status: pbgo=%d dao=%d", pb.Status, d.Status)
	}
	if pb.Address == nil && d.Address != nil || pb.Address != nil && d.Address == nil {
		t.Errorf("Address nil mismatch: pbgo=%v dao=%v", pb.Address, d.Address)
	} else if pb.Address != nil {
		if pb.Address.Street != d.Address.Street {
			t.Errorf("Address.Street: pbgo=%q dao=%q", pb.Address.Street, d.Address.Street)
		}
		if pb.Address.City != d.Address.City {
			t.Errorf("Address.City: pbgo=%q dao=%q", pb.Address.City, d.Address.City)
		}
	}
	if len(pb.Scores) != len(d.Scores) {
		t.Errorf("Scores len: pbgo=%d dao=%d", len(pb.Scores), len(d.Scores))
	} else {
		for i := range pb.Scores {
			if pb.Scores[i] != d.Scores[i] {
				t.Errorf("Scores[%d]: pbgo=%d dao=%d", i, pb.Scores[i], d.Scores[i])
			}
		}
	}
	if len(pb.Tags) != len(d.Tags) {
		t.Errorf("Tags len: pbgo=%d dao=%d", len(pb.Tags), len(d.Tags))
	} else {
		for i := range pb.Tags {
			if pb.Tags[i] != d.Tags[i] {
				t.Errorf("Tags[%d]: pbgo=%q dao=%q", i, pb.Tags[i], d.Tags[i])
			}
		}
	}
	if pb.Rating != d.Rating {
		t.Errorf("Rating: pbgo=%v dao=%v", pb.Rating, d.Rating)
	}
	if pb.CreatedAt != d.CreatedAt {
		t.Errorf("CreatedAt: pbgo=%d dao=%d", pb.CreatedAt, d.CreatedAt)
	}
	if !bytes.Equal(pb.Avatar, d.Avatar) {
		t.Errorf("Avatar: pbgo=%x dao=%x", pb.Avatar, d.Avatar)
	}
}

// --- Deviation tests ---

// TestDeviationUnpackedRepeatedRejected verifies that our decoder rejects
// unpacked wire encoding for packed-only repeated numeric fields.
func TestDeviationUnpackedRepeatedRejected(t *testing.T) {
	t.Parallel()

	// Encode Scores (field 6) using unpacked varint wire type instead of packed LEN.
	var wire []byte
	wire = append(wire, 0x30) // tag: field 6, WireVarint (6<<3|0)
	wire = append(wire, 0x0a) // value: 10
	wire = append(wire, 0x30)
	wire = append(wire, 0x14) // value: 20

	var p dao.Person
	err := p.UnmarshalBinary(wire)
	if err == nil {
		t.Fatal("expected error for unpacked repeated int32, got nil")
	}
}

// TestDeviationDuplicateSingularRejected verifies that UnmarshalBinary returns
// an error when a non-repeated field appears more than once.
func TestDeviationDuplicateSingularRejected(t *testing.T) {
	t.Parallel()

	// Encode Name (field 1) twice.
	var wire []byte
	wire = append(wire, 0x0a, 0x05, 'A', 'l', 'i', 'c', 'e') // field 1, "Alice"
	wire = append(wire, 0x0a, 0x03, 'B', 'o', 'b')           // field 1, "Bob"

	var p dao.Person
	err := p.UnmarshalBinary(wire)
	if err == nil {
		t.Fatal("expected error for duplicate singular field, got nil")
	}
}

// TestDeviationDuplicateSingularLenientAccepted verifies that
// UnmarshalBinaryLenient accepts duplicate singular fields (last-one-wins).
func TestDeviationDuplicateSingularLenientAccepted(t *testing.T) {
	t.Parallel()

	var wire []byte
	wire = append(wire, 0x0a, 0x05, 'A', 'l', 'i', 'c', 'e')
	wire = append(wire, 0x0a, 0x03, 'B', 'o', 'b')

	var p dao.Person
	if err := p.UnmarshalBinaryLenient(wire); err != nil {
		t.Fatalf("UnmarshalBinaryLenient: unexpected error: %v", err)
	}
	if p.Name != "Bob" {
		t.Errorf("last-one-wins: got %q, want %q", p.Name, "Bob")
	}
}

// --- Optional field tests ---

// ptr returns a pointer to v, used for constructing optional field values.
func ptr[T any](v T) *T { return &v }

// populatedPbgoWithOptionals returns a pbgo.Person with all optional fields set to non-zero values.
func populatedPbgoWithOptionals() *pbgo.Person {
	p := populatedPbgo()
	p.Nickname = ptr("gopher")
	p.Level = ptr(int32(42))
	p.Verified = ptr(true)
	p.Score = ptr(float32(9.9))
	p.UpdatedAt = ptr(int64(1700000001))
	p.PrevStatus = ptr(pbgo.Status_STATUS_INACTIVE)
	p.Fingerprint = []byte{0xde, 0xad, 0xbe, 0xef}
	return p
}

// populatedDaoWithOptionals returns a dao.Person with all optional fields set to non-zero values.
func populatedDaoWithOptionals() *dao.Person {
	p := populatedDao()
	p.Nickname = ptr("gopher")
	p.Level = ptr(int32(42))
	p.Verified = ptr(true)
	p.Score = ptr(float32(9.9))
	p.UpdatedAt = ptr(int64(1700000001))
	p.PrevStatus = ptr(dao.Status_STATUS_INACTIVE)
	p.Fingerprint = []byte{0xde, 0xad, 0xbe, 0xef}
	return p
}

// TestOptionalFieldsWireIdentical verifies that optional fields produce identical
// wire bytes between pbgo and dao for non-zero values.
func TestOptionalFieldsWireIdentical(t *testing.T) {
	t.Parallel()

	pbgoWire, err := proto.Marshal(populatedPbgoWithOptionals())
	if err != nil {
		t.Fatalf("proto.Marshal: %v", err)
	}

	daoWire, err := populatedDaoWithOptionals().MarshalBinary()
	if err != nil {
		t.Fatalf("dao.MarshalBinary: %v", err)
	}

	if !bytes.Equal(pbgoWire, daoWire) {
		t.Errorf("wire bytes differ with optional fields:\n  pbgo (%d bytes): %x\n  dao  (%d bytes): %x",
			len(pbgoWire), pbgoWire, len(daoWire), daoWire)
	}
}

// TestOptionalFieldsNilWireIdentical verifies that nil optional fields produce
// identical wire bytes (field omitted) between pbgo and dao.
func TestOptionalFieldsNilWireIdentical(t *testing.T) {
	t.Parallel()

	// All optional fields nil — should produce same bytes as base populated message.
	pbgoWire, err := proto.Marshal(populatedPbgo())
	if err != nil {
		t.Fatalf("proto.Marshal: %v", err)
	}

	daoWire, err := populatedDao().MarshalBinary()
	if err != nil {
		t.Fatalf("dao.MarshalBinary: %v", err)
	}

	if !bytes.Equal(pbgoWire, daoWire) {
		t.Errorf("wire bytes differ (nil optional fields):\n  pbgo (%d bytes): %x\n  dao  (%d bytes): %x",
			len(pbgoWire), pbgoWire, len(daoWire), daoWire)
	}
}

// TestOptionalFieldsZeroValueWireIdentical verifies that optional fields set to
// zero values (e.g. &0, &false, &"") produce identical wire bytes between pbgo and dao.
func TestOptionalFieldsZeroValueWireIdentical(t *testing.T) {
	t.Parallel()

	pbgoMsg := populatedPbgo()
	pbgoMsg.Nickname = ptr("")
	pbgoMsg.Level = ptr(int32(0))
	pbgoMsg.Verified = ptr(false)
	pbgoMsg.Score = ptr(float32(0))
	pbgoMsg.UpdatedAt = ptr(int64(0))
	pbgoMsg.PrevStatus = ptr(pbgo.Status_STATUS_UNSPECIFIED)
	pbgoMsg.Fingerprint = []byte{}

	daoMsg := populatedDao()
	daoMsg.Nickname = ptr("")
	daoMsg.Level = ptr(int32(0))
	daoMsg.Verified = ptr(false)
	daoMsg.Score = ptr(float32(0))
	daoMsg.UpdatedAt = ptr(int64(0))
	daoMsg.PrevStatus = ptr(dao.Status_STATUS_UNSPECIFIED)
	daoMsg.Fingerprint = []byte{}

	pbgoWire, err := proto.Marshal(pbgoMsg)
	if err != nil {
		t.Fatalf("proto.Marshal: %v", err)
	}

	daoWire, err := daoMsg.MarshalBinary()
	if err != nil {
		t.Fatalf("dao.MarshalBinary: %v", err)
	}

	if !bytes.Equal(pbgoWire, daoWire) {
		t.Errorf("wire bytes differ (zero-value optional fields):\n  pbgo (%d bytes): %x\n  dao  (%d bytes): %x",
			len(pbgoWire), pbgoWire, len(daoWire), daoWire)
	}
}

// TestOptionalFieldsRoundTrip verifies nil→nil and &v→&v round-trip for optional fields.
func TestOptionalFieldsRoundTrip(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name string
		orig *dao.Person
	}{
		{"all_nil", populatedDao()},
		{"non_zero", populatedDaoWithOptionals()},
		{
			"zero_values",
			func() *dao.Person {
				p := populatedDao()
				p.Nickname = ptr("")
				p.Level = ptr(int32(0))
				p.Verified = ptr(false)
				p.Score = ptr(float32(0))
				p.UpdatedAt = ptr(int64(0))
				p.PrevStatus = ptr(dao.Status_STATUS_UNSPECIFIED)
				p.Fingerprint = []byte{}
				return p
			}(),
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			wire, err := tc.orig.MarshalBinary()
			if err != nil {
				t.Fatalf("MarshalBinary: %v", err)
			}

			var got dao.Person
			if err := got.UnmarshalBinary(wire); err != nil {
				t.Fatalf("UnmarshalBinary: %v", err)
			}

			assertOptionalPersonEqual(t, tc.orig, &got)
		})
	}
}

// TestOptionalFieldsPbgoEncodesDaoDecodes verifies pbgo→dao cross-decode for optional fields.
func TestOptionalFieldsPbgoEncodesDaoDecodes(t *testing.T) {
	t.Parallel()

	pbgoMsg := populatedPbgoWithOptionals()
	wire, err := proto.Marshal(pbgoMsg)
	if err != nil {
		t.Fatalf("proto.Marshal: %v", err)
	}

	var got dao.Person
	if err := got.UnmarshalBinary(wire); err != nil {
		t.Fatalf("dao.UnmarshalBinary: %v", err)
	}

	assertOptionalPersonEqual(t, populatedDaoWithOptionals(), &got)
}

// TestOptionalFieldsDaoEncodesPbgoDecodes verifies dao→pbgo cross-decode for optional fields.
func TestOptionalFieldsDaoEncodesPbgoDecodes(t *testing.T) {
	t.Parallel()

	daoMsg := populatedDaoWithOptionals()
	wire, err := daoMsg.MarshalBinary()
	if err != nil {
		t.Fatalf("dao.MarshalBinary: %v", err)
	}

	var got pbgo.Person
	if err := proto.Unmarshal(wire, &got); err != nil {
		t.Fatalf("proto.Unmarshal: %v", err)
	}

	// Verify optional fields decoded correctly by pbgo.
	if got.Nickname == nil || *got.Nickname != "gopher" {
		t.Errorf("Nickname: got %v, want &%q", got.Nickname, "gopher")
	}
	if got.Level == nil || *got.Level != 42 {
		t.Errorf("Level: got %v, want &42", got.Level)
	}
	if got.Verified == nil || *got.Verified != true {
		t.Errorf("Verified: got %v, want &true", got.Verified)
	}
	if got.Score == nil || *got.Score != float32(9.9) {
		t.Errorf("Score: got %v, want &9.9", got.Score)
	}
	if got.UpdatedAt == nil || *got.UpdatedAt != 1700000001 {
		t.Errorf("UpdatedAt: got %v, want &1700000001", got.UpdatedAt)
	}
	if got.PrevStatus == nil || *got.PrevStatus != pbgo.Status_STATUS_INACTIVE {
		t.Errorf("PrevStatus: got %v, want &STATUS_INACTIVE", got.PrevStatus)
	}
	if !bytes.Equal(got.Fingerprint, []byte{0xde, 0xad, 0xbe, 0xef}) {
		t.Errorf("Fingerprint: got %x, want deadbeef", got.Fingerprint)
	}
}

// assertOptionalPersonEqual compares two dao.Person values including optional fields.
func assertOptionalPersonEqual(t *testing.T, want, got *dao.Person) {
	t.Helper()
	assertPersonEqual(t, &pbgo.Person{
		Name:      want.Name,
		Age:       want.Age,
		Active:    want.Active,
		Status:    pbgo.Status(want.Status),
		Rating:    want.Rating,
		CreatedAt: want.CreatedAt,
		Avatar:    want.Avatar,
	}, &dao.Person{
		Name:      got.Name,
		Age:       got.Age,
		Active:    got.Active,
		Status:    got.Status,
		Rating:    got.Rating,
		CreatedAt: got.CreatedAt,
		Avatar:    got.Avatar,
	})

	// Optional string
	if (want.Nickname == nil) != (got.Nickname == nil) {
		t.Errorf("Nickname nil mismatch: want %v got %v", want.Nickname, got.Nickname)
	} else if want.Nickname != nil && *want.Nickname != *got.Nickname {
		t.Errorf("Nickname: want %q got %q", *want.Nickname, *got.Nickname)
	}
	// Optional int32
	if (want.Level == nil) != (got.Level == nil) {
		t.Errorf("Level nil mismatch: want %v got %v", want.Level, got.Level)
	} else if want.Level != nil && *want.Level != *got.Level {
		t.Errorf("Level: want %d got %d", *want.Level, *got.Level)
	}
	// Optional bool
	if (want.Verified == nil) != (got.Verified == nil) {
		t.Errorf("Verified nil mismatch: want %v got %v", want.Verified, got.Verified)
	} else if want.Verified != nil && *want.Verified != *got.Verified {
		t.Errorf("Verified: want %v got %v", *want.Verified, *got.Verified)
	}
	// Optional float32
	if (want.Score == nil) != (got.Score == nil) {
		t.Errorf("Score nil mismatch: want %v got %v", want.Score, got.Score)
	} else if want.Score != nil && *want.Score != *got.Score {
		t.Errorf("Score: want %v got %v", *want.Score, *got.Score)
	}
	// Optional int64
	if (want.UpdatedAt == nil) != (got.UpdatedAt == nil) {
		t.Errorf("UpdatedAt nil mismatch: want %v got %v", want.UpdatedAt, got.UpdatedAt)
	} else if want.UpdatedAt != nil && *want.UpdatedAt != *got.UpdatedAt {
		t.Errorf("UpdatedAt: want %d got %d", *want.UpdatedAt, *got.UpdatedAt)
	}
	// Optional enum
	if (want.PrevStatus == nil) != (got.PrevStatus == nil) {
		t.Errorf("PrevStatus nil mismatch: want %v got %v", want.PrevStatus, got.PrevStatus)
	} else if want.PrevStatus != nil && *want.PrevStatus != *got.PrevStatus {
		t.Errorf("PrevStatus: want %d got %d", *want.PrevStatus, *got.PrevStatus)
	}
	// Optional bytes ([]byte, nil means not set)
	if !bytes.Equal(want.Fingerprint, got.Fingerprint) {
		t.Errorf("Fingerprint: want %x got %x", want.Fingerprint, got.Fingerprint)
	}
}
