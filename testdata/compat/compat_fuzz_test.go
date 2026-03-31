package compat_test

import (
	"testing"

	"google.golang.org/protobuf/proto"

	"github.com/pinealctx/gcode/testdata/compat/dao"
	"github.com/pinealctx/gcode/testdata/compat/pbgo"
)

// FuzzUnmarshalBinary verifies that our UnmarshalBinary never panics on
// arbitrary input, and that any input accepted by proto.Unmarshal is also
// accepted by our decoder (lenient mode).
func FuzzUnmarshalBinary(f *testing.F) {
	// Seed corpus: valid wire bytes from both encoders.
	wire, _ := populatedDao().MarshalBinary()
	f.Add(wire)

	pbgoWire, _ := proto.Marshal(populatedPbgo())
	f.Add(pbgoWire)

	// Empty message.
	f.Add([]byte{})

	// Single-field messages.
	f.Add([]byte{0x0a, 0x05, 'A', 'l', 'i', 'c', 'e'}) // field 1 (name), len=5
	f.Add([]byte{0x10, 0x1e})                          // field 2 (age), varint 30

	f.Fuzz(func(t *testing.T, data []byte) {
		// Must not panic.
		var d dao.Person
		_ = d.UnmarshalBinaryLenient(data)

		// If proto.Unmarshal accepts it, our lenient decoder must too.
		var pb pbgo.Person
		if err := proto.Unmarshal(data, &pb); err == nil {
			var d2 dao.Person
			if err2 := d2.UnmarshalBinaryLenient(data); err2 != nil {
				t.Errorf("proto.Unmarshal accepted input but dao.UnmarshalBinaryLenient rejected it: %v\ninput: %x", err2, data)
			}
		}
	})
}

// FuzzPersonCreateUnmarshalBinary verifies that PersonCreate.UnmarshalBinary
// never panics on arbitrary input. No cross-implementation oracle is available
// (PersonCreate has no protoc-gen-go counterpart), so only panic-safety is checked.
func FuzzPersonCreateUnmarshalBinary(f *testing.F) {
	c := &dao.PersonCreate{Nickname: "ali"}
	wire, _ := c.MarshalBinary()
	f.Add(wire)
	f.Add([]byte{})
	f.Add([]byte{0x0a, 0x03, 'a', 'l', 'i'}) // field 1 (name), len=3

	f.Fuzz(func(t *testing.T, data []byte) {
		var d dao.PersonCreate
		_ = d.UnmarshalBinaryLenient(data)
	})
}

// FuzzPersonUpdateByNameUnmarshalBinary verifies that PersonUpdateByName.UnmarshalBinary
// never panics on arbitrary input. No cross-implementation oracle is available
// (PersonUpdateByName has no protoc-gen-go counterpart), so only panic-safety is checked.
func FuzzPersonUpdateByNameUnmarshalBinary(f *testing.F) {
	u := &dao.PersonUpdateByName{Name: "Alice"}
	wire, _ := u.MarshalBinary()
	f.Add(wire)
	f.Add([]byte{})
	f.Add([]byte{0x0a, 0x05, 'A', 'l', 'i', 'c', 'e'}) // field 1 (name), len=5

	f.Fuzz(func(t *testing.T, data []byte) {
		var d dao.PersonUpdateByName
		_ = d.UnmarshalBinaryLenient(data)
	})
}

// FuzzCreatePersonResponseUnmarshalBinary verifies that CreatePersonResponse.UnmarshalBinary
// never panics on arbitrary input. No cross-implementation oracle is available
// (CreatePersonResponse has no protoc-gen-go counterpart), so only panic-safety is checked.
func FuzzCreatePersonResponseUnmarshalBinary(f *testing.F) {
	r := &dao.CreatePersonResponse{Id: "uuid-123"}
	wire, _ := r.MarshalBinary()
	f.Add(wire)
	f.Add([]byte{})
	f.Add([]byte{0x0a, 0x08, 'u', 'u', 'i', 'd', '-', '1', '2', '3'}) // field 1 (id)

	f.Fuzz(func(t *testing.T, data []byte) {
		var d dao.CreatePersonResponse
		_ = d.UnmarshalBinaryLenient(data)
	})
}
