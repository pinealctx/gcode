package runtime

import (
	"testing"
)

func TestAppendVarint(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		v    uint64
		want []byte
	}{
		{"zero", 0, []byte{0x00}},
		{"one", 1, []byte{0x01}},
		{"127", 127, []byte{0x7f}},
		{"128", 128, []byte{0x80, 0x01}},
		{"300", 300, []byte{0xac, 0x02}},
		{"max_uint64", ^uint64(0), []byte{0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0x01}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := AppendVarint(nil, tt.v)
			if len(got) != len(tt.want) {
				t.Fatalf("AppendVarint(%d) len = %d, want %d", tt.v, len(got), len(tt.want))
			}
			for i := range got {
				if got[i] != tt.want[i] {
					t.Fatalf("AppendVarint(%d)[%d] = 0x%02x, want 0x%02x", tt.v, i, got[i], tt.want[i])
				}
			}
		})
	}
}

func TestAppendTag(t *testing.T) {
	t.Parallel()

	// field 1, wire type 0 (varint) → tag = (1<<3)|0 = 8 → varint 0x08
	got := AppendTag(nil, 1, WireVarint)
	if len(got) != 1 || got[0] != 0x08 {
		t.Errorf("AppendTag(1, VARINT) = %v, want [0x08]", got)
	}

	// field 2, wire type 2 (LEN) → tag = (2<<3)|2 = 18 → varint 0x12
	got = AppendTag(nil, 2, WireBytes)
	if len(got) != 1 || got[0] != 0x12 {
		t.Errorf("AppendTag(2, BYTES) = %v, want [0x12]", got)
	}
}

func TestEncodeZigZag(t *testing.T) {
	t.Parallel()

	tests32 := []struct {
		in   int32
		want uint64
	}{
		{0, 0},
		{-1, 1},
		{1, 2},
		{-2, 3},
		{2147483647, 4294967294},
		{-2147483648, 4294967295},
	}
	for _, tt := range tests32 {
		got := EncodeZigZag32(tt.in)
		if got != tt.want {
			t.Errorf("EncodeZigZag32(%d) = %d, want %d", tt.in, got, tt.want)
		}
	}

	tests64 := []struct {
		in   int64
		want uint64
	}{
		{0, 0},
		{-1, 1},
		{1, 2},
		{-2, 3},
	}
	for _, tt := range tests64 {
		got := EncodeZigZag64(tt.in)
		if got != tt.want {
			t.Errorf("EncodeZigZag64(%d) = %d, want %d", tt.in, got, tt.want)
		}
	}
}

func TestSizeVarint(t *testing.T) {
	t.Parallel()

	tests := []struct {
		v    uint64
		want int
	}{
		{0, 1},
		{127, 1},
		{128, 2},
		{16383, 2},
		{16384, 3},
		{^uint64(0), 10},
	}
	for _, tt := range tests {
		got := SizeVarint(tt.v)
		if got != tt.want {
			t.Errorf("SizeVarint(%d) = %d, want %d", tt.v, got, tt.want)
		}
	}
}

func TestAppendString(t *testing.T) {
	t.Parallel()

	got := AppendString(nil, "testing")
	// length prefix (7) + "testing"
	if len(got) != 8 {
		t.Fatalf("AppendString len = %d, want 8", len(got))
	}
	if got[0] != 7 {
		t.Errorf("length prefix = %d, want 7", got[0])
	}
	if string(got[1:]) != "testing" {
		t.Errorf("payload = %q, want %q", string(got[1:]), "testing")
	}
}

func TestAppendFixed(t *testing.T) {
	t.Parallel()

	got32 := AppendFixed32(nil, 1)
	if len(got32) != 4 || got32[0] != 1 || got32[1] != 0 || got32[2] != 0 || got32[3] != 0 {
		t.Errorf("AppendFixed32(1) = %v, want [1 0 0 0]", got32)
	}

	got64 := AppendFixed64(nil, 1)
	if len(got64) != 8 || got64[0] != 1 {
		t.Errorf("AppendFixed64(1) = %v, want [1 0 0 0 0 0 0 0]", got64)
	}
}

func TestAppendBool(t *testing.T) {
	t.Parallel()

	gotTrue := AppendBool(nil, true)
	if len(gotTrue) != 1 || gotTrue[0] != 1 {
		t.Errorf("AppendBool(true) = %v, want [1]", gotTrue)
	}
	gotFalse := AppendBool(nil, false)
	if len(gotFalse) != 1 || gotFalse[0] != 0 {
		t.Errorf("AppendBool(false) = %v, want [0]", gotFalse)
	}
}

func TestConsumeVarint(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		input   []byte
		wantVal uint64
		wantN   int
	}{
		{"zero", []byte{0x00}, 0, 1},
		{"one", []byte{0x01}, 1, 1},
		{"127", []byte{0x7f}, 127, 1},
		{"128", []byte{0x80, 0x01}, 128, 2},
		{"300", []byte{0xac, 0x02}, 300, 2},
		{"max_uint64", []byte{0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0x01}, ^uint64(0), 10},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, n := ConsumeVarint(tt.input)
			if n != tt.wantN {
				t.Errorf("ConsumeVarint n = %d, want %d", n, tt.wantN)
			}
			if got != tt.wantVal {
				t.Errorf("ConsumeVarint val = %d, want %d", got, tt.wantVal)
			}
		})
	}

	// Truncated.
	_, n := ConsumeVarint([]byte{0x80})
	if n != -1 {
		t.Errorf("truncated varint: n = %d, want -1", n)
	}

	// Overflow (11 bytes).
	overflow := []byte{0x80, 0x80, 0x80, 0x80, 0x80, 0x80, 0x80, 0x80, 0x80, 0x80, 0x01}
	_, n = ConsumeVarint(overflow)
	if n != -2 {
		t.Errorf("overflow varint: n = %d, want -2", n)
	}
}

func TestConsumeFixed32(t *testing.T) {
	t.Parallel()

	got, n := ConsumeFixed32([]byte{0x01, 0x00, 0x00, 0x00})
	if n != 4 || got != 1 {
		t.Errorf("ConsumeFixed32 = (%d, %d), want (1, 4)", got, n)
	}

	// Truncated.
	_, n = ConsumeFixed32([]byte{0x01, 0x00})
	if n != -1 {
		t.Errorf("truncated fixed32: n = %d, want -1", n)
	}
}

func TestConsumeFixed64(t *testing.T) {
	t.Parallel()

	got, n := ConsumeFixed64([]byte{0x01, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00})
	if n != 8 || got != 1 {
		t.Errorf("ConsumeFixed64 = (%d, %d), want (1, 8)", got, n)
	}

	// Truncated.
	_, n = ConsumeFixed64([]byte{0x01, 0x00})
	if n != -1 {
		t.Errorf("truncated fixed64: n = %d, want -1", n)
	}
}

func TestConsumeBytes(t *testing.T) {
	t.Parallel()

	// "hello" = length 5 + bytes.
	input := []byte{0x05, 'h', 'e', 'l', 'l', 'o'}
	got, n := ConsumeBytes(input)
	if n != 6 || string(got) != "hello" {
		t.Errorf("ConsumeBytes = (%q, %d), want (hello, 6)", got, n)
	}

	// Truncated length prefix.
	_, n = ConsumeBytes([]byte{0x80})
	if n != -1 {
		t.Errorf("truncated length: n = %d, want -1", n)
	}

	// Truncated payload.
	_, n = ConsumeBytes([]byte{0x05, 'h', 'i'})
	if n != -1 {
		t.Errorf("truncated payload: n = %d, want -1", n)
	}
}

func TestConsumeTag(t *testing.T) {
	t.Parallel()

	// field 1, wire type 0 → tag = 0x08
	fn, wt, n := ConsumeTag([]byte{0x08})
	if fn != 1 || wt != 0 || n != 1 {
		t.Errorf("ConsumeTag = (%d, %d, %d), want (1, 0, 1)", fn, wt, n)
	}

	// field 2, wire type 2 → tag = 0x12
	fn, wt, n = ConsumeTag([]byte{0x12})
	if fn != 2 || wt != 2 || n != 1 {
		t.Errorf("ConsumeTag = (%d, %d, %d), want (2, 2, 1)", fn, wt, n)
	}
}

func TestSkipField(t *testing.T) {
	t.Parallel()

	// Skip varint.
	n := SkipField([]byte{0x01}, WireVarint)
	if n != 1 {
		t.Errorf("SkipField varint = %d, want 1", n)
	}

	// Skip fixed32.
	n = SkipField([]byte{0x01, 0x02, 0x03, 0x04}, WireFixed32)
	if n != 4 {
		t.Errorf("SkipField fixed32 = %d, want 4", n)
	}

	// Skip fixed64.
	n = SkipField([]byte{1, 2, 3, 4, 5, 6, 7, 8}, WireFixed64)
	if n != 8 {
		t.Errorf("SkipField fixed64 = %d, want 8", n)
	}

	// Skip LEN.
	n = SkipField([]byte{0x03, 'a', 'b', 'c'}, WireBytes)
	if n != 4 {
		t.Errorf("SkipField bytes = %d, want 4", n)
	}

	// Unknown wire type.
	n = SkipField([]byte{0x01}, 7)
	if n != -1 {
		t.Errorf("SkipField unknown = %d, want -1", n)
	}
}

func TestDecodeZigZag(t *testing.T) {
	t.Parallel()

	tests32 := []struct {
		encoded uint64
		want    int32
	}{
		{0, 0},
		{1, -1},
		{2, 1},
		{3, -2},
		{4294967294, 2147483647},
		{4294967295, -2147483648},
	}
	for _, tt := range tests32 {
		got := DecodeZigZag32(tt.encoded)
		if got != tt.want {
			t.Errorf("DecodeZigZag32(%d) = %d, want %d", tt.encoded, got, tt.want)
		}
	}

	tests64 := []struct {
		encoded uint64
		want    int64
	}{
		{0, 0},
		{1, -1},
		{2, 1},
		{3, -2},
	}
	for _, tt := range tests64 {
		got := DecodeZigZag64(tt.encoded)
		if got != tt.want {
			t.Errorf("DecodeZigZag64(%d) = %d, want %d", tt.encoded, got, tt.want)
		}
	}
}

// TestRoundtripVarint verifies encode→decode roundtrip for varint.
func TestRoundtripVarint(t *testing.T) {
	t.Parallel()

	vals := []uint64{0, 1, 127, 128, 300, 16383, 16384, ^uint64(0)}
	for _, v := range vals {
		encoded := AppendVarint(nil, v)
		decoded, n := ConsumeVarint(encoded)
		if n != len(encoded) || decoded != v {
			t.Errorf("roundtrip %d: got (%d, %d), want (%d, %d)", v, decoded, n, v, len(encoded))
		}
	}
}
