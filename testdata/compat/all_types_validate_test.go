// Package compat_test — validate constraint tests for AllValidate.
// Covers every constraint type with at least one positive (pass) and one negative (fail) case.
package compat_test

import (
	"testing"

	"github.com/pinealctx/gcode/testdata/compat/dao"
)

// validAllValidate returns an AllValidate that satisfies all constraints.
func validAllValidate() *dao.AllValidate {
	status := dao.Status_STATUS_ACTIVE
	return &dao.AllValidate{
		UGte:     1,                     // gte=1: exactly at boundary
		ULte:     1000,                  // lte=1000: exactly at boundary
		UIn:      2,                     // in=[1,2,3]
		UNotIn:   5,                     // not_in=[0]: any non-zero value
		FGt:      0.1,                   // gt=0
		DLte:     0.5,                   // lte=1.0
		SIn:      "b",                   // in=[a,b,c]
		SNotIn:   "z",                   // not_in=[x,y]
		IIn:      1,                     // in=[1,2,-1]
		SUri:     "https://example.com", // valid URI
		OStatus:  &status,               // defined_only
		BMinmax:  []byte{0x01},          // min_len=1, max_len=100
		RItems:   []int32{0, 1, 2},      // items.gte=0
		IGtLt:    0,                     // gt=-10, lt=10: 0 is in range
		UGtLt:    50,                    // gt=5, lt=100: 50 is in range
		FLt:      50.0,                  // lt=99.5: 50 < 99.5
		DGt:      0.0,                   // gt=-1.0: 0 > -1
		SPattern: "Hello",              // pattern=^[A-Z][a-z]+$
	}
}

// TestAllValidate_Valid verifies that a fully valid AllValidate passes Validate().
func TestAllValidate_Valid(t *testing.T) {
	t.Parallel()
	if err := validAllValidate().Validate(); err != nil {
		t.Errorf("expected nil, got %v", err)
	}
}

// TestAllValidate_UGte verifies uint32 gte constraint.
func TestAllValidate_UGte(t *testing.T) {
	t.Parallel()

	// fail: 0 < 1
	a := validAllValidate()
	a.UGte = 0
	assertVE(t, a.Validate(), "u_gte", "gte")

	// pass: exactly at boundary
	a2 := validAllValidate()
	a2.UGte = 1
	if err := a2.Validate(); err != nil {
		t.Errorf("u_gte=1 should pass, got: %v", err)
	}
}

// TestAllValidate_ULte verifies uint64 lte constraint.
func TestAllValidate_ULte(t *testing.T) {
	t.Parallel()

	// fail: 1001 > 1000
	a := validAllValidate()
	a.ULte = 1001
	assertVE(t, a.Validate(), "u_lte", "lte")

	// pass: exactly at boundary
	a2 := validAllValidate()
	a2.ULte = 1000
	if err := a2.Validate(); err != nil {
		t.Errorf("u_lte=1000 should pass, got: %v", err)
	}
}

// TestAllValidate_UIn verifies uint32 in constraint.
func TestAllValidate_UIn(t *testing.T) {
	t.Parallel()

	// fail: 4 not in [1,2,3]
	a := validAllValidate()
	a.UIn = 4
	assertVE(t, a.Validate(), "u_in", "in")

	// pass: all valid values
	for _, v := range []uint32{1, 2, 3} {
		a2 := validAllValidate()
		a2.UIn = v
		if err := a2.Validate(); err != nil {
			t.Errorf("u_in=%d should pass, got: %v", v, err)
		}
	}
}

// TestAllValidate_UNotIn verifies uint32 not_in constraint.
func TestAllValidate_UNotIn(t *testing.T) {
	t.Parallel()

	// fail: 0 is in not_in list
	a := validAllValidate()
	a.UNotIn = 0
	assertVE(t, a.Validate(), "u_not_in", "not_in")

	// pass: any non-zero value
	a2 := validAllValidate()
	a2.UNotIn = 1
	if err := a2.Validate(); err != nil {
		t.Errorf("u_not_in=1 should pass, got: %v", err)
	}
}

// TestAllValidate_FGt verifies float32 gt constraint.
func TestAllValidate_FGt(t *testing.T) {
	t.Parallel()

	// fail: 0 is not > 0
	a := validAllValidate()
	a.FGt = 0
	assertVE(t, a.Validate(), "f_gt", "gt")

	// fail: negative value
	a2 := validAllValidate()
	a2.FGt = -1.0
	assertVE(t, a2.Validate(), "f_gt", "gt")

	// pass: positive value
	a3 := validAllValidate()
	a3.FGt = 0.001
	if err := a3.Validate(); err != nil {
		t.Errorf("f_gt=0.001 should pass, got: %v", err)
	}
}

// TestAllValidate_DLte verifies float64 lte constraint.
func TestAllValidate_DLte(t *testing.T) {
	t.Parallel()

	// fail: 1.1 > 1.0
	a := validAllValidate()
	a.DLte = 1.1
	assertVE(t, a.Validate(), "d_lte", "lte")

	// pass: exactly at boundary
	a2 := validAllValidate()
	a2.DLte = 1.0
	if err := a2.Validate(); err != nil {
		t.Errorf("d_lte=1.0 should pass, got: %v", err)
	}

	// pass: zero value skips check (zero-value guard)
	a3 := validAllValidate()
	a3.DLte = 0
	if err := a3.Validate(); err != nil {
		t.Errorf("d_lte=0 (zero value) should pass, got: %v", err)
	}
}

// TestAllValidate_SIn verifies string in constraint.
func TestAllValidate_SIn(t *testing.T) {
	t.Parallel()

	// fail: "d" not in [a,b,c]
	a := validAllValidate()
	a.SIn = "d"
	assertVE(t, a.Validate(), "s_in", "in")

	// pass: all valid values
	for _, v := range []string{"a", "b", "c"} {
		a2 := validAllValidate()
		a2.SIn = v
		if err := a2.Validate(); err != nil {
			t.Errorf("s_in=%q should pass, got: %v", v, err)
		}
	}

	// pass: empty string skips check (zero-value guard)
	a3 := validAllValidate()
	a3.SIn = ""
	if err := a3.Validate(); err != nil {
		t.Errorf("s_in=\"\" (zero value) should skip check, got: %v", err)
	}
}

// TestAllValidate_SNotIn verifies string not_in constraint.
func TestAllValidate_SNotIn(t *testing.T) {
	t.Parallel()

	// fail: "x" is in not_in list
	a := validAllValidate()
	a.SNotIn = "x"
	assertVE(t, a.Validate(), "s_not_in", "not_in")

	// fail: "y" is in not_in list
	a2 := validAllValidate()
	a2.SNotIn = "y"
	assertVE(t, a2.Validate(), "s_not_in", "not_in")

	// pass: value not in forbidden list
	a3 := validAllValidate()
	a3.SNotIn = "z"
	if err := a3.Validate(); err != nil {
		t.Errorf("s_not_in=\"z\" should pass, got: %v", err)
	}

	// pass: empty string skips check (zero-value guard)
	a4 := validAllValidate()
	a4.SNotIn = ""
	if err := a4.Validate(); err != nil {
		t.Errorf("s_not_in=\"\" (zero value) should skip check, got: %v", err)
	}
}

// TestAllValidate_IIn verifies int32 in constraint (signed).
func TestAllValidate_IIn(t *testing.T) {
	t.Parallel()

	// fail: 0 not in [1,2,-1]
	a := validAllValidate()
	a.IIn = 0
	assertVE(t, a.Validate(), "i_in", "in")

	// fail: 3 not in [1,2,-1]
	a2 := validAllValidate()
	a2.IIn = 3
	assertVE(t, a2.Validate(), "i_in", "in")

	// pass: all valid values including negative
	for _, v := range []int32{1, 2, -1} {
		a3 := validAllValidate()
		a3.IIn = v
		if err := a3.Validate(); err != nil {
			t.Errorf("i_in=%d should pass, got: %v", v, err)
		}
	}
}

// TestAllValidate_SUri verifies string uri constraint.
func TestAllValidate_SUri(t *testing.T) {
	t.Parallel()

	// fail: not a valid URI
	a := validAllValidate()
	a.SUri = "not a uri"
	assertVE(t, a.Validate(), "s_uri", "uri")

	// pass: valid URI
	a2 := validAllValidate()
	a2.SUri = "https://example.com/path"
	if err := a2.Validate(); err != nil {
		t.Errorf("valid URI should pass, got: %v", err)
	}

	// pass: empty string skips check (zero-value guard)
	a3 := validAllValidate()
	a3.SUri = ""
	if err := a3.Validate(); err != nil {
		t.Errorf("s_uri=\"\" (zero value) should skip check, got: %v", err)
	}
}

// TestAllValidate_OStatus verifies optional enum defined_only constraint.
func TestAllValidate_OStatus(t *testing.T) {
	t.Parallel()

	// pass: nil skips check
	a := validAllValidate()
	a.OStatus = nil
	if err := a.Validate(); err != nil {
		t.Errorf("nil o_status should skip check, got: %v", err)
	}

	// pass: all defined values
	for _, v := range []dao.Status{
		dao.Status_STATUS_UNSPECIFIED,
		dao.Status_STATUS_ACTIVE,
		dao.Status_STATUS_INACTIVE,
	} {
		s := v
		a2 := validAllValidate()
		a2.OStatus = &s
		if err := a2.Validate(); err != nil {
			t.Errorf("o_status=%v should pass defined_only, got: %v", v, err)
		}
	}

	// fail: undefined enum value triggers defined_only
	invalid := dao.Status(999)
	a3 := validAllValidate()
	a3.OStatus = &invalid
	assertVE(t, a3.Validate(), "o_status", "defined_only")
}

// TestAllValidate_BMinmax verifies bytes min_len/max_len constraints.
func TestAllValidate_BMinmax(t *testing.T) {
	t.Parallel()

	// pass: nil skips check
	a := validAllValidate()
	a.BMinmax = nil
	if err := a.Validate(); err != nil {
		t.Errorf("nil b_minmax should skip check, got: %v", err)
	}

	// fail: empty slice violates min_len=1
	a2 := validAllValidate()
	a2.BMinmax = []byte{}
	assertVE(t, a2.Validate(), "b_minmax", "min_len")

	// fail: 101 bytes violates max_len=100
	a3 := validAllValidate()
	a3.BMinmax = make([]byte, 101)
	assertVE(t, a3.Validate(), "b_minmax", "max_len")

	// pass: exactly at boundaries
	a4 := validAllValidate()
	a4.BMinmax = []byte{0x01}
	if err := a4.Validate(); err != nil {
		t.Errorf("b_minmax len=1 should pass min_len=1, got: %v", err)
	}

	a5 := validAllValidate()
	a5.BMinmax = make([]byte, 100)
	if err := a5.Validate(); err != nil {
		t.Errorf("b_minmax len=100 should pass max_len=100, got: %v", err)
	}
}

// TestAllValidate_RItems verifies repeated min_items, max_items, and items.gte constraints.
func TestAllValidate_RItems(t *testing.T) {
	t.Parallel()

	// fail: empty list violates min_items=1
	a := validAllValidate()
	a.RItems = nil
	assertVE(t, a.Validate(), "r_items", "min_items")

	// fail: empty slice also violates min_items=1
	a2 := validAllValidate()
	a2.RItems = []int32{}
	assertVE(t, a2.Validate(), "r_items", "min_items")

	// fail: 6 items violates max_items=5
	a3 := validAllValidate()
	a3.RItems = []int32{0, 1, 2, 3, 4, 5}
	assertVE(t, a3.Validate(), "r_items", "max_items")

	// fail: negative value in list violates items.gte=0
	a4 := validAllValidate()
	a4.RItems = []int32{0, 1, -1}
	assertVE(t, a4.Validate(), "r_items[2]", "gte")

	// fail: first element negative
	a5 := validAllValidate()
	a5.RItems = []int32{-5, 1, 2}
	assertVE(t, a5.Validate(), "r_items[0]", "gte")

	// pass: exactly at min boundary
	a6 := validAllValidate()
	a6.RItems = []int32{0}
	if err := a6.Validate(); err != nil {
		t.Errorf("r_items len=1 should pass min_items=1, got: %v", err)
	}

	// pass: exactly at max boundary
	a7 := validAllValidate()
	a7.RItems = []int32{0, 1, 2, 3, 4}
	if err := a7.Validate(); err != nil {
		t.Errorf("r_items len=5 should pass max_items=5, got: %v", err)
	}
}

// TestAllValidate_IGtLt verifies int32 gt+lt (exclusive bounds for signed int).
func TestAllValidate_IGtLt(t *testing.T) {
	t.Parallel()

	// fail: -10 is not > -10
	a := validAllValidate()
	a.IGtLt = -10
	assertVE(t, a.Validate(), "i_gt_lt", "gt")

	// fail: 10 is not < 10
	a2 := validAllValidate()
	a2.IGtLt = 10
	assertVE(t, a2.Validate(), "i_gt_lt", "lt")

	// pass: within range
	a3 := validAllValidate()
	a3.IGtLt = 0
	if err := a3.Validate(); err != nil {
		t.Errorf("i_gt_lt=0 should pass, got: %v", err)
	}
}

// TestAllValidate_UGtLt verifies uint32 gt+lt (exclusive bounds for unsigned int).
func TestAllValidate_UGtLt(t *testing.T) {
	t.Parallel()

	// fail: 5 is not > 5
	a := validAllValidate()
	a.UGtLt = 5
	assertVE(t, a.Validate(), "u_gt_lt", "gt")

	// fail: 100 is not < 100
	a2 := validAllValidate()
	a2.UGtLt = 100
	assertVE(t, a2.Validate(), "u_gt_lt", "lt")

	// pass: within range
	a3 := validAllValidate()
	a3.UGtLt = 50
	if err := a3.Validate(); err != nil {
		t.Errorf("u_gt_lt=50 should pass, got: %v", err)
	}
}

// TestAllValidate_FLt verifies float32 lt (exclusiveMaximum).
func TestAllValidate_FLt(t *testing.T) {
	t.Parallel()

	// fail: 99.5 is not < 99.5
	a := validAllValidate()
	a.FLt = 99.5
	assertVE(t, a.Validate(), "f_lt", "lt")

	// pass: below limit
	a2 := validAllValidate()
	a2.FLt = 50.0
	if err := a2.Validate(); err != nil {
		t.Errorf("f_lt=50.0 should pass, got: %v", err)
	}
}

// TestAllValidate_DGt verifies float64 gt (exclusiveMinimum).
func TestAllValidate_DGt(t *testing.T) {
	t.Parallel()

	// fail: -1 is not > -1
	a := validAllValidate()
	a.DGt = -1
	assertVE(t, a.Validate(), "d_gt", "gt")

	// pass: above limit
	a2 := validAllValidate()
	a2.DGt = 0
	if err := a2.Validate(); err != nil {
		t.Errorf("d_gt=0 should pass, got: %v", err)
	}
}

// TestAllRepeatedUpdate_Valid verifies that a fully valid AllRepeatedUpdate passes Validate().
func TestAllRepeatedUpdate_Valid(t *testing.T) {
	t.Parallel()
	u := &dao.AllRepeatedUpdate{
		RSint32:   []int32{-100, 0},
		RSfixed32: []int32{-1, -5},
		RDouble:   []float64{0.6, 1.0},
		RBytes:    [][]byte{{0x01}, {0x02}},
		REnum:     []dao.Status{dao.Status_STATUS_UNSPECIFIED, dao.Status_STATUS_ACTIVE},
	}
	if err := u.Validate(); err != nil {
		t.Errorf("expected nil, got %v", err)
	}
}

// TestAllRepeatedUpdate_ItemsConstraints verifies items-level constraints on repeated fields.
func TestAllRepeatedUpdate_ItemsConstraints(t *testing.T) {
	t.Parallel()

	// fail: sint32 item below gte=-100
	u1 := &dao.AllRepeatedUpdate{RSint32: []int32{-101}}
	assertVE(t, u1.Validate(), "r_sint32[0]", "gte")

	// fail: sfixed32 item >= 0 (lt=0)
	u2 := &dao.AllRepeatedUpdate{RSfixed32: []int32{0}}
	assertVE(t, u2.Validate(), "r_sfixed32[0]", "lt")

	// fail: double item <= 0.5 (gt=0.5)
	u3 := &dao.AllRepeatedUpdate{RDouble: []float64{0.5}}
	assertVE(t, u3.Validate(), "r_double[0]", "gt")

	// fail: bytes item with len < 1 (min_len=1)
	u4 := &dao.AllRepeatedUpdate{RBytes: [][]byte{{}}}
	assertVE(t, u4.Validate(), "r_bytes[0]", "min_len")

	// fail: undefined enum value (defined_only)
	u5 := &dao.AllRepeatedUpdate{REnum: []dao.Status{dao.Status(999)}}
	assertVE(t, u5.Validate(), "r_enum[0]", "defined_only")

	// pass: all defined enum values accepted
	for _, v := range []dao.Status{
		dao.Status_STATUS_UNSPECIFIED,
		dao.Status_STATUS_ACTIVE,
		dao.Status_STATUS_INACTIVE,
	} {
		u := &dao.AllRepeatedUpdate{REnum: []dao.Status{v}}
		if err := u.Validate(); err != nil {
			t.Errorf("r_enum=%v should pass defined_only, got: %v", v, err)
		}
	}

	// pass: empty slices skip items validation
	u6 := &dao.AllRepeatedUpdate{}
	if err := u6.Validate(); err != nil {
		t.Errorf("empty AllRepeatedUpdate should pass, got: %v", err)
	}
}

// TestAllValidate_SPattern verifies string pattern constraint.
func TestAllValidate_SPattern(t *testing.T) {
	t.Parallel()

	// fail: lowercase first char
	a := validAllValidate()
	a.SPattern = "hello"
	assertVE(t, a.Validate(), "s_pattern", "pattern")

	// fail: all uppercase doesn't match [A-Z][a-z]+
	a2 := validAllValidate()
	a2.SPattern = "HELLO"
	assertVE(t, a2.Validate(), "s_pattern", "pattern")

	// pass: matches pattern
	a3 := validAllValidate()
	a3.SPattern = "Hello"
	if err := a3.Validate(); err != nil {
		t.Errorf("s_pattern=\"Hello\" should pass, got: %v", err)
	}

	// pass: empty string skips check
	a4 := validAllValidate()
	a4.SPattern = ""
	if err := a4.Validate(); err != nil {
		t.Errorf("s_pattern=\"\" (zero value) should skip check, got: %v", err)
	}
}

// TestAllValidate_ItemsInNotIn verifies items-level in/not_in constraints on repeated fields.
func TestAllValidate_ItemsInNotIn(t *testing.T) {
	t.Parallel()

	// --- r_str_in: items must be "foo" or "bar" ---

	// fail: value not in allowed set
	a1 := validAllValidate()
	a1.RStrIn = []string{"baz"}
	assertVE(t, a1.Validate(), "r_str_in[0]", "in")

	// fail: second element not in set
	a2 := validAllValidate()
	a2.RStrIn = []string{"foo", "other"}
	assertVE(t, a2.Validate(), "r_str_in[1]", "in")

	// pass: all elements in allowed set
	a3 := validAllValidate()
	a3.RStrIn = []string{"foo", "bar", "foo"}
	if err := a3.Validate(); err != nil {
		t.Errorf("r_str_in all valid should pass, got: %v", err)
	}

	// pass: empty slice skips items validation
	a4 := validAllValidate()
	a4.RStrIn = nil
	if err := a4.Validate(); err != nil {
		t.Errorf("r_str_in nil should pass, got: %v", err)
	}

	// --- r_str_not_in: items must not be "bad" ---

	// fail: forbidden value present
	b1 := validAllValidate()
	b1.RStrNotIn = []string{"ok", "bad"}
	assertVE(t, b1.Validate(), "r_str_not_in[1]", "not_in")

	// pass: no forbidden values
	b2 := validAllValidate()
	b2.RStrNotIn = []string{"ok", "fine"}
	if err := b2.Validate(); err != nil {
		t.Errorf("r_str_not_in no forbidden values should pass, got: %v", err)
	}

	// pass: nil slice skips items validation
	b3 := validAllValidate()
	b3.RStrNotIn = nil
	if err := b3.Validate(); err != nil {
		t.Errorf("r_str_not_in nil should pass, got: %v", err)
	}

	// --- r_int_in: items must be 1, 2, or 3 ---

	// fail: value not in allowed set
	c1 := validAllValidate()
	c1.RIntIn = []int32{4}
	assertVE(t, c1.Validate(), "r_int_in[0]", "in")

	// pass: all elements in allowed set
	c2 := validAllValidate()
	c2.RIntIn = []int32{1, 2, 3, 1}
	if err := c2.Validate(); err != nil {
		t.Errorf("r_int_in all valid should pass, got: %v", err)
	}

	// --- r_uint_not_in: items must not be 0 ---

	// fail: zero value present
	d1 := validAllValidate()
	d1.RUintNotIn = []uint32{1, 0}
	assertVE(t, d1.Validate(), "r_uint_not_in[1]", "not_in")

	// pass: no zero values
	d2 := validAllValidate()
	d2.RUintNotIn = []uint32{1, 2, 3}
	if err := d2.Validate(); err != nil {
		t.Errorf("r_uint_not_in no zero values should pass, got: %v", err)
	}
}
