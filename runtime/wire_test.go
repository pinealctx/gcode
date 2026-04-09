package runtime

import (
	"math"
	"strings"
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
	if n != -3 {
		t.Errorf("SkipField unknown = %d, want -3", n)
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

// TestAppendSint verifies that AppendSint32 and AppendSint64 produce the correct
// ZigZag-encoded varint bytes, including boundary values.
func TestAppendSint(t *testing.T) {
	t.Parallel()

	// AppendSint32: ZigZag encode then varint append.
	// 0 → ZigZag 0 → [0x00]
	got := AppendSint32(nil, 0)
	if len(got) != 1 || got[0] != 0x00 {
		t.Errorf("AppendSint32(0) = %v, want [0x00]", got)
	}
	// -1 → ZigZag 1 → [0x01]
	got = AppendSint32(nil, -1)
	if len(got) != 1 || got[0] != 0x01 {
		t.Errorf("AppendSint32(-1) = %v, want [0x01]", got)
	}
	// 1 → ZigZag 2 → [0x02]
	got = AppendSint32(nil, 1)
	if len(got) != 1 || got[0] != 0x02 {
		t.Errorf("AppendSint32(1) = %v, want [0x02]", got)
	}

	// AppendSint64: same ZigZag logic for int64.
	got64 := AppendSint64(nil, 0)
	if len(got64) != 1 || got64[0] != 0x00 {
		t.Errorf("AppendSint64(0) = %v, want [0x00]", got64)
	}
	got64 = AppendSint64(nil, -1)
	if len(got64) != 1 || got64[0] != 0x01 {
		t.Errorf("AppendSint64(-1) = %v, want [0x01]", got64)
	}

	// Boundary values: MaxInt32 → ZigZag 4294967294 → 5-byte varint.
	got = AppendSint32(nil, math.MaxInt32)
	if len(got) != 5 {
		t.Errorf("AppendSint32(MaxInt32) len = %d, want 5", len(got))
	}
	// MinInt32 → ZigZag 4294967295 → 5-byte varint.
	got = AppendSint32(nil, math.MinInt32)
	if len(got) != 5 {
		t.Errorf("AppendSint32(MinInt32) len = %d, want 5", len(got))
	}
	// MaxInt64 → ZigZag 18446744073709551614 → 10-byte varint.
	got64 = AppendSint64(nil, math.MaxInt64)
	if len(got64) != 10 {
		t.Errorf("AppendSint64(MaxInt64) len = %d, want 10", len(got64))
	}
	// MinInt64 → ZigZag 18446744073709551615 → 10-byte varint.
	got64 = AppendSint64(nil, math.MinInt64)
	if len(got64) != 10 {
		t.Errorf("AppendSint64(MinInt64) len = %d, want 10", len(got64))
	}
}

func TestAppendFloatDouble(t *testing.T) {
	t.Parallel()

	// AppendFloat: float32 as fixed32 (4 bytes little-endian).
	got := AppendFloat(nil, 0.0)
	if len(got) != 4 {
		t.Fatalf("AppendFloat(0.0) len = %d, want 4", len(got))
	}
	// 0.0 float32 bits = 0x00000000
	for i, b := range got {
		if b != 0 {
			t.Errorf("AppendFloat(0.0)[%d] = 0x%02x, want 0x00", i, b)
		}
	}

	// 1.0 float32 bits = 0x3F800000 → little-endian [0x00, 0x00, 0x80, 0x3F]
	got = AppendFloat(nil, 1.0)
	if len(got) != 4 || got[3] != 0x3F || got[2] != 0x80 || got[1] != 0x00 || got[0] != 0x00 {
		t.Errorf("AppendFloat(1.0) = %v, want [0x00 0x00 0x80 0x3F]", got)
	}

	// AppendDouble: float64 as fixed64 (8 bytes little-endian).
	got64 := AppendDouble(nil, 0.0)
	if len(got64) != 8 {
		t.Fatalf("AppendDouble(0.0) len = %d, want 8", len(got64))
	}
	for i, b := range got64 {
		if b != 0 {
			t.Errorf("AppendDouble(0.0)[%d] = 0x%02x, want 0x00", i, b)
		}
	}
}

func TestAppendBytes(t *testing.T) {
	t.Parallel()

	// nil slice → length prefix 0 + no payload
	got := AppendBytes(nil, nil)
	if len(got) != 1 || got[0] != 0 {
		t.Errorf("AppendBytes(nil) = %v, want [0x00]", got)
	}

	// []byte{1,2,3} → length prefix 3 + payload
	got = AppendBytes(nil, []byte{1, 2, 3})
	if len(got) != 4 || got[0] != 3 || got[1] != 1 || got[2] != 2 || got[3] != 3 {
		t.Errorf("AppendBytes([1,2,3]) = %v, want [3 1 2 3]", got)
	}
}

func TestSizeFunctions(t *testing.T) {
	t.Parallel()

	// SizeBool: always 1.
	if SizeBool(true) != 1 || SizeBool(false) != 1 {
		t.Error("SizeBool must always return 1")
	}

	// SizeInt32: negative values sign-extend to 10 bytes.
	if SizeInt32(0) != 1 {
		t.Errorf("SizeInt32(0) = %d, want 1", SizeInt32(0))
	}
	if SizeInt32(-1) != 10 {
		t.Errorf("SizeInt32(-1) = %d, want 10", SizeInt32(-1))
	}
	if SizeInt32(1) != 1 {
		t.Errorf("SizeInt32(1) = %d, want 1", SizeInt32(1))
	}

	// SizeInt64.
	if SizeInt64(0) != 1 {
		t.Errorf("SizeInt64(0) = %d, want 1", SizeInt64(0))
	}
	if SizeInt64(-1) != 10 {
		t.Errorf("SizeInt64(-1) = %d, want 10", SizeInt64(-1))
	}

	// SizeUint32 / SizeUint64.
	if SizeUint32(0) != 1 || SizeUint32(128) != 2 {
		t.Errorf("SizeUint32 unexpected: 0→%d, 128→%d", SizeUint32(0), SizeUint32(128))
	}
	if SizeUint64(0) != 1 || SizeUint64(128) != 2 {
		t.Errorf("SizeUint64 unexpected: 0→%d, 128→%d", SizeUint64(0), SizeUint64(128))
	}

	// SizeSint32 / SizeSint64: ZigZag, so -1 → 1 (1 byte).
	if SizeSint32(0) != 1 || SizeSint32(-1) != 1 || SizeSint32(1) != 1 {
		t.Errorf("SizeSint32 unexpected: 0→%d, -1→%d, 1→%d", SizeSint32(0), SizeSint32(-1), SizeSint32(1))
	}
	if SizeSint64(0) != 1 || SizeSint64(-1) != 1 {
		t.Errorf("SizeSint64 unexpected: 0→%d, -1→%d", SizeSint64(0), SizeSint64(-1))
	}

	// Fixed-size types.
	if SizeFloat(0) != 4 || SizeFloat(1.5) != 4 {
		t.Error("SizeFloat must always return 4")
	}
	if SizeDouble(0) != 8 || SizeDouble(1.5) != 8 {
		t.Error("SizeDouble must always return 8")
	}
	if SizeFixed32(0) != 4 || SizeFixed32(^uint32(0)) != 4 {
		t.Error("SizeFixed32 must always return 4")
	}
	if SizeFixed64(0) != 8 || SizeFixed64(^uint64(0)) != 8 {
		t.Error("SizeFixed64 must always return 8")
	}
	if SizeSfixed32(0) != 4 || SizeSfixed32(-1) != 4 {
		t.Error("SizeSfixed32 must always return 4")
	}
	if SizeSfixed64(0) != 8 || SizeSfixed64(-1) != 8 {
		t.Error("SizeSfixed64 must always return 8")
	}

	// SizeString.
	if SizeString("") != 1 {
		t.Errorf("SizeString(\"\") = %d, want 1", SizeString(""))
	}
	// SizeString: length >= 128 crosses varint byte boundary (2-byte prefix).
	if SizeString("hi") != 3 {
		t.Errorf("SizeString(\"hi\") = %d, want 3", SizeString("hi"))
	}
	s128 := strings.Repeat("x", 128)
	if SizeString(s128) != 130 {
		t.Errorf("SizeString(128-byte) = %d, want 130", SizeString(s128))
	}

	// SizeBytes.
	if SizeBytes(nil) != 1 {
		t.Errorf("SizeBytes(nil) = %d, want 1", SizeBytes(nil))
	}
	if SizeBytes([]byte{1, 2}) != 3 {
		t.Errorf("SizeBytes([1,2]) = %d, want 3", SizeBytes([]byte{1, 2}))
	}
	b128 := make([]byte, 128)
	if SizeBytes(b128) != 130 {
		t.Errorf("SizeBytes(128-byte) = %d, want 130", SizeBytes(b128))
	}

	// SizeEnum: same as SizeInt32 for non-negative values.
	if SizeEnum(0) != 1 || SizeEnum(1) != 1 || SizeEnum(128) != 2 {
		t.Errorf("SizeEnum unexpected: 0→%d, 1→%d, 128→%d", SizeEnum(0), SizeEnum(1), SizeEnum(128))
	}
}

func TestIsZeroFloatDouble(t *testing.T) {
	t.Parallel()

	// Positive zero.
	if !IsZeroFloat(0.0) {
		t.Error("IsZeroFloat(+0.0) = false, want true")
	}
	if !IsZeroDouble(0.0) {
		t.Error("IsZeroDouble(+0.0) = false, want true")
	}

	// Non-zero values.
	if IsZeroFloat(1.0) {
		t.Error("IsZeroFloat(1.0) = true, want false")
	}
	if IsZeroDouble(1.0) {
		t.Error("IsZeroDouble(1.0) = true, want false")
	}

	// Negative zero: bits differ from positive zero, so IsZero returns false.
	negZeroF := float32(math.Float32frombits(0x80000000))
	if IsZeroFloat(negZeroF) {
		t.Error("IsZeroFloat(-0.0) = true, want false (negative zero has different bits)")
	}
	negZeroD := math.Float64frombits(0x8000000000000000)
	if IsZeroDouble(negZeroD) {
		t.Error("IsZeroDouble(-0.0) = true, want false (negative zero has different bits)")
	}
}

func TestErrorString(t *testing.T) {
	t.Parallel()

	if ErrTruncated.Error() != "protobuf: truncated message" {
		t.Errorf("ErrTruncated.Error() = %q", ErrTruncated.Error())
	}
	if ErrOverflow.Error() != "protobuf: varint overflow" {
		t.Errorf("ErrOverflow.Error() = %q", ErrOverflow.Error())
	}
	if ErrWireType.Error() != "protobuf: wrong wire type" {
		t.Errorf("ErrWireType.Error() = %q", ErrWireType.Error())
	}
	if ErrPackedLen.Error() != "protobuf: packed field length mismatch" {
		t.Errorf("ErrPackedLen.Error() = %q", ErrPackedLen.Error())
	}
	if ErrDuplicateField.Error() != "protobuf: duplicate non-repeated field" {
		t.Errorf("ErrDuplicateField.Error() = %q", ErrDuplicateField.Error())
	}
	if ErrUnknownWireType.Error() != "protobuf: unknown wire type" {
		t.Errorf("ErrUnknownWireType.Error() = %q", ErrUnknownWireType.Error())
	}
	if ErrNestingDepth.Error() != "protobuf: message nesting depth exceeded" {
		t.Errorf("ErrNestingDepth.Error() = %q", ErrNestingDepth.Error())
	}
}

func TestSizeTag(t *testing.T) {
	t.Parallel()

	tests := []struct {
		fieldNumber int
		want        int
	}{
		{1, 1},         // tag = 8, fits in 1 byte
		{15, 1},        // tag = 120, fits in 1 byte (max 1-byte tag)
		{16, 2},        // tag = 128, needs 2 bytes
		{2047, 2},      // tag = 16376, fits in 2 bytes
		{2048, 3},      // tag = 16384, needs 3 bytes
		{536870911, 5}, // proto max field number (2^29-1); tag = 4294967288, needs 5 bytes
	}

	for _, tt := range tests {
		got := SizeTag(tt.fieldNumber)
		if got != tt.want {
			t.Errorf("SizeTag(%d) = %d, want %d", tt.fieldNumber, got, tt.want)
		}
		// Cross-check: SizeTag must equal len(AppendTag(...)) for any wire type.
		encoded := AppendTag(nil, tt.fieldNumber, WireVarint)
		if got != len(encoded) {
			t.Errorf("SizeTag(%d) = %d, but AppendTag produced %d bytes", tt.fieldNumber, got, len(encoded))
		}
	}
}
