// Package dao_test verifies deviation behaviors of the generated Person codec.
//
// Deviations from the official protobuf spec (intentional, stage-1 constraints):
//  1. Packed-only: repeated numeric fields reject unpacked wire encoding.
//  2. Duplicate singular: UnmarshalBinary rejects duplicate non-repeated fields;
//     UnmarshalBinaryLenient accepts them (last-one-wins).
package dao_test

import (
	"errors"
	"testing"

	"github.com/pinealctx/gcode/runtime"
	"github.com/pinealctx/gcode/testdata/compat/dao"
)

// buildUnpackedScores encodes field 6 (Scores, []int32) using the unpacked
// wire format: one tag+varint per element instead of a single packed LEN record.
func buildUnpackedScores(values []int32) []byte {
	var b []byte
	for _, v := range values {
		b = runtime.AppendTag(b, 6, runtime.WireVarint) // WireVarint, not WireBytes
		b = runtime.AppendVarint(b, uint64(v))
	}
	return b
}

// buildDuplicateName encodes field 1 (Name, string) twice.
func buildDuplicateName(first, second string) []byte {
	var b []byte
	b = runtime.AppendTag(b, 1, runtime.WireBytes)
	b = runtime.AppendString(b, first)
	b = runtime.AppendTag(b, 1, runtime.WireBytes)
	b = runtime.AppendString(b, second)
	return b
}

// TestUnpackedScoresRejected verifies that unpacked encoding for a packed-only
// repeated field returns ErrWireType.
func TestUnpackedScoresRejected(t *testing.T) {
	t.Parallel()

	wire := buildUnpackedScores([]int32{1, 2, 3})
	var p dao.Person
	err := p.UnmarshalBinary(wire)
	if err == nil {
		t.Fatal("expected error for unpacked repeated int32, got nil")
	}
	if !errors.Is(err, runtime.ErrWireType) {
		t.Errorf("expected ErrWireType, got: %v", err)
	}
}

// TestUnpackedScoresRejectedLenient verifies that even lenient mode rejects
// unpacked encoding (wire type mismatch is not a leniency concern).
func TestUnpackedScoresRejectedLenient(t *testing.T) {
	t.Parallel()

	wire := buildUnpackedScores([]int32{10, 20})
	var p dao.Person
	err := p.UnmarshalBinaryLenient(wire)
	if err == nil {
		t.Fatal("expected error for unpacked repeated int32 in lenient mode, got nil")
	}
	if !errors.Is(err, runtime.ErrWireType) {
		t.Errorf("expected ErrWireType, got: %v", err)
	}
}

// TestDuplicateSingularRejected verifies that UnmarshalBinary returns
// ErrDuplicateField when a non-repeated field appears more than once.
func TestDuplicateSingularRejected(t *testing.T) {
	t.Parallel()

	wire := buildDuplicateName("Alice", "Bob")
	var p dao.Person
	err := p.UnmarshalBinary(wire)
	if err == nil {
		t.Fatal("expected error for duplicate singular field, got nil")
	}
	if !errors.Is(err, runtime.ErrDuplicateField) {
		t.Errorf("expected ErrDuplicateField, got: %v", err)
	}
}

// TestDuplicateSingularLenientLastWins verifies that UnmarshalBinaryLenient
// accepts duplicate singular fields and keeps the last value.
func TestDuplicateSingularLenientLastWins(t *testing.T) {
	t.Parallel()

	wire := buildDuplicateName("Alice", "Bob")
	var p dao.Person
	if err := p.UnmarshalBinaryLenient(wire); err != nil {
		t.Fatalf("UnmarshalBinaryLenient: unexpected error: %v", err)
	}
	if p.Name != "Bob" {
		t.Errorf("last-one-wins: got %q, want %q", p.Name, "Bob")
	}
}

// TestDuplicateAddressRejected verifies duplicate nested message field is rejected.
func TestDuplicateAddressRejected(t *testing.T) {
	t.Parallel()

	var b []byte
	// Encode field 5 (Address) twice.
	addr1, _ := (&dao.Address{Street: "First St", City: "A"}).MarshalBinary()
	addr2, _ := (&dao.Address{Street: "Second St", City: "B"}).MarshalBinary()
	b = runtime.AppendTag(b, 5, runtime.WireBytes)
	b = runtime.AppendVarint(b, uint64(len(addr1)))
	b = append(b, addr1...)
	b = runtime.AppendTag(b, 5, runtime.WireBytes)
	b = runtime.AppendVarint(b, uint64(len(addr2)))
	b = append(b, addr2...)

	var p dao.Person
	err := p.UnmarshalBinary(b)
	if err == nil {
		t.Fatal("expected error for duplicate address field, got nil")
	}
	if !errors.Is(err, runtime.ErrDuplicateField) {
		t.Errorf("expected ErrDuplicateField, got: %v", err)
	}
}

// TestDuplicateAddressLenientLastWins verifies lenient mode keeps last address.
func TestDuplicateAddressLenientLastWins(t *testing.T) {
	t.Parallel()

	var b []byte
	addr1, _ := (&dao.Address{Street: "First St", City: "A"}).MarshalBinary()
	addr2, _ := (&dao.Address{Street: "Second St", City: "B"}).MarshalBinary()
	b = runtime.AppendTag(b, 5, runtime.WireBytes)
	b = runtime.AppendVarint(b, uint64(len(addr1)))
	b = append(b, addr1...)
	b = runtime.AppendTag(b, 5, runtime.WireBytes)
	b = runtime.AppendVarint(b, uint64(len(addr2)))
	b = append(b, addr2...)

	var p dao.Person
	if err := p.UnmarshalBinaryLenient(b); err != nil {
		t.Fatalf("UnmarshalBinaryLenient: unexpected error: %v", err)
	}
	if p.Address == nil {
		t.Fatal("Address is nil after lenient unmarshal")
	}
	if p.Address.Street != "Second St" {
		t.Errorf("last-one-wins: got %q, want %q", p.Address.Street, "Second St")
	}
}

// buildNestedTreeNodeWire constructs wire bytes for a TreeNode whose child field
// (field 2) is itself a TreeNode, nested n levels deep.
// Each layer wraps the previous payload as field 2 (WireBytes).
func buildNestedTreeNodeWire(n int) []byte {
	payload := []byte{} // empty innermost TreeNode
	for i := 0; i < n; i++ {
		var layer []byte
		layer = runtime.AppendTag(layer, 2, runtime.WireBytes)
		layer = runtime.AppendVarint(layer, uint64(len(payload)))
		layer = append(layer, payload...)
		payload = layer
	}
	return payload
}

// TestNestingDepthExceeded verifies that UnmarshalBinary returns ErrNestingDepth
// when message nesting exceeds runtime.DefaultRecursionLimit (100).
// Depth semantics: UnmarshalBinary starts at depth=DefaultRecursionLimit; each
// nested call receives depth-1. The check is depth<=0, so:
//   - n = DefaultRecursionLimit     → innermost call receives depth=0 → rejected (exact boundary)
//   - n = DefaultRecursionLimit + 1 → innermost call receives depth=-1 → rejected (one past boundary)
func TestNestingDepthExceeded(t *testing.T) {
	t.Parallel()

	t.Run("exact_boundary", func(t *testing.T) {
		t.Parallel()
		wire := buildNestedTreeNodeWire(runtime.DefaultRecursionLimit)
		var node dao.TreeNode
		err := node.UnmarshalBinary(wire)
		if err == nil {
			t.Fatal("expected ErrNestingDepth at exact limit, got nil")
		}
		if !errors.Is(err, runtime.ErrNestingDepth) {
			t.Errorf("expected ErrNestingDepth at exact limit, got: %v", err)
		}
	})

	t.Run("one_past_boundary", func(t *testing.T) {
		t.Parallel()
		wire := buildNestedTreeNodeWire(runtime.DefaultRecursionLimit + 1)
		var node dao.TreeNode
		err := node.UnmarshalBinary(wire)
		if err == nil {
			t.Fatal("expected ErrNestingDepth one past limit, got nil")
		}
		if !errors.Is(err, runtime.ErrNestingDepth) {
			t.Errorf("expected ErrNestingDepth one past limit, got: %v", err)
		}
	})
}

// TestNestingDepthAtLimit verifies that nesting one level below the limit is accepted.
// With DefaultRecursionLimit=100, the outermost UnmarshalBinary starts at depth=100.
// Each nested call decrements depth by 1, so depth=1 at the 99th nested level (still > 0).
// The 100th nested call would receive depth=0 and be rejected, so 99 is the maximum
// accepted nesting depth.
func TestNestingDepthAtLimit(t *testing.T) {
	t.Parallel()

	wire := buildNestedTreeNodeWire(runtime.DefaultRecursionLimit - 1)
	var node dao.TreeNode
	if err := node.UnmarshalBinary(wire); err != nil {
		t.Errorf("expected success at DefaultRecursionLimit-1 nesting, got: %v", err)
	}
}

// TestNestingDepthExceededLenient verifies that lenient mode also enforces the
// nesting depth limit (depth checking is independent of duplicate-field leniency).
func TestNestingDepthExceededLenient(t *testing.T) {
	t.Parallel()

	wire := buildNestedTreeNodeWire(runtime.DefaultRecursionLimit)
	var node dao.TreeNode
	err := node.UnmarshalBinaryLenient(wire)
	if err == nil {
		t.Fatal("expected ErrNestingDepth in lenient mode, got nil")
	}
	if !errors.Is(err, runtime.ErrNestingDepth) {
		t.Errorf("expected ErrNestingDepth in lenient mode, got: %v", err)
	}
}
