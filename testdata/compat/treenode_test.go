// Package compat_test — TreeNode marshal, round-trip, and validate tests.
package compat_test

import (
	"testing"

	"github.com/pinealctx/gcode/testdata/compat/dao"
)

// TestTreeNodeSizeNil verifies Size returns 0 for nil TreeNode.
func TestTreeNodeSizeNil(t *testing.T) {
	t.Parallel()

	var node *dao.TreeNode
	if got := node.Size(); got != 0 {
		t.Errorf("nil TreeNode.Size() = %d, want 0", got)
	}
}

// TestTreeNodeSizeEmpty verifies Size returns 0 for zero-value TreeNode.
func TestTreeNodeSizeEmpty(t *testing.T) {
	t.Parallel()

	var node dao.TreeNode
	if got := node.Size(); got != 0 {
		t.Errorf("empty TreeNode.Size() = %d, want 0", got)
	}
}

// TestTreeNodeSizeWithValue verifies Size returns the exact wire size.
// "hello" → tag(1,bytes)=1 + varint(5)=1 + "hello"=5 → 7 bytes.
func TestTreeNodeSizeWithValue(t *testing.T) {
	t.Parallel()

	node := dao.TreeNode{Value: "hello"}
	if got := node.Size(); got != 7 {
		t.Errorf("TreeNode{Value:\"hello\"}.Size() = %d, want 7", got)
	}
}

// TestTreeNodeMarshalRoundTrip verifies marshal → unmarshal round-trip
// for a single-level TreeNode (value only, no child).
func TestTreeNodeMarshalRoundTrip(t *testing.T) {
	t.Parallel()

	original := dao.TreeNode{Value: "root"}
	data, err := original.MarshalBinary()
	if err != nil {
		t.Fatalf("MarshalBinary: %v", err)
	}

	var decoded dao.TreeNode
	if err := decoded.UnmarshalBinary(data); err != nil {
		t.Fatalf("UnmarshalBinary: %v", err)
	}
	if decoded.Value != original.Value {
		t.Errorf("Value: got %q, want %q", decoded.Value, original.Value)
	}
	if decoded.Child != nil {
		t.Error("Child: got non-nil, want nil")
	}
}

// TestTreeNodeMarshalNestedRoundTrip verifies marshal → unmarshal round-trip
// for a nested TreeNode (root → child with value).
func TestTreeNodeMarshalNestedRoundTrip(t *testing.T) {
	t.Parallel()

	original := dao.TreeNode{
		Value: "root",
		Child: &dao.TreeNode{
			Value: "child",
		},
	}
	data, err := original.MarshalBinary()
	if err != nil {
		t.Fatalf("MarshalBinary: %v", err)
	}

	var decoded dao.TreeNode
	if err := decoded.UnmarshalBinary(data); err != nil {
		t.Fatalf("UnmarshalBinary: %v", err)
	}
	if decoded.Value != "root" {
		t.Errorf("Value: got %q, want %q", decoded.Value, "root")
	}
	if decoded.Child == nil {
		t.Fatal("Child is nil")
	}
	if decoded.Child.Value != "child" {
		t.Errorf("Child.Value: got %q, want %q", decoded.Child.Value, "child")
	}
}

// TestTreeNodeValidateEmpty verifies Validate returns nil for empty TreeNode.
func TestTreeNodeValidateEmpty(t *testing.T) {
	t.Parallel()

	var node dao.TreeNode
	if err := node.Validate(); err != nil {
		t.Errorf("empty TreeNode.Validate() = %v, want nil", err)
	}
}

// TestTreeNodeValidateWithChild verifies Validate recurses into child.
func TestTreeNodeValidateWithChild(t *testing.T) {
	t.Parallel()

	node := dao.TreeNode{
		Value: "root",
		Child: &dao.TreeNode{Value: "leaf"},
	}
	if err := node.Validate(); err != nil {
		t.Errorf("TreeNode with Child.Validate() = %v, want nil", err)
	}
}
