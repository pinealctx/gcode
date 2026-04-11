// Package runtime provides low-level protobuf wire format encoding primitives.
// Generated marshal/unmarshal code calls these functions directly.
package runtime

import (
	"encoding/binary"
	"math"
	"math/bits"
)

// Wire type constants.
const (
	WireVarint  = 0
	WireFixed64 = 1
	WireBytes   = 2
	WireFixed32 = 5
)

// AppendTag appends a protobuf field tag (field_number << 3 | wire_type).
func AppendTag(b []byte, fieldNumber int, wireType int) []byte {
	return AppendVarint(b, uint64(fieldNumber)<<3|uint64(wireType))
}

// AppendVarint appends a base-128 varint-encoded uint64.
func AppendVarint(b []byte, v uint64) []byte {
	for v >= 0x80 {
		b = append(b, byte(v)|0x80)
		v >>= 7
	}
	return append(b, byte(v))
}

// AppendSint32 appends a ZigZag-encoded sint32 as a varint.
func AppendSint32(b []byte, v int32) []byte {
	return AppendVarint(b, EncodeZigZag32(v))
}

// AppendSint64 appends a ZigZag-encoded sint64 as a varint.
func AppendSint64(b []byte, v int64) []byte {
	return AppendVarint(b, EncodeZigZag64(v))
}

// AppendFixed32 appends a 4-byte little-endian value.
func AppendFixed32(b []byte, v uint32) []byte {
	return binary.LittleEndian.AppendUint32(b, v)
}

// AppendFixed64 appends an 8-byte little-endian value.
func AppendFixed64(b []byte, v uint64) []byte {
	return binary.LittleEndian.AppendUint64(b, v)
}

// AppendFloat appends a float32 as fixed32.
func AppendFloat(b []byte, v float32) []byte {
	return AppendFixed32(b, math.Float32bits(v))
}

// AppendDouble appends a float64 as fixed64.
func AppendDouble(b []byte, v float64) []byte {
	return AppendFixed64(b, math.Float64bits(v))
}

// AppendBool appends a bool as a single-byte varint (0 or 1).
func AppendBool(b []byte, v bool) []byte {
	if v {
		return append(b, 1)
	}
	return append(b, 0)
}

// AppendString appends a length-delimited string.
func AppendString(b []byte, v string) []byte {
	b = AppendVarint(b, uint64(len(v)))
	return append(b, v...)
}

// AppendBytes appends a length-delimited byte slice.
func AppendBytes(b []byte, v []byte) []byte {
	b = AppendVarint(b, uint64(len(v)))
	return append(b, v...)
}

// EncodeZigZag32 encodes a signed int32 using ZigZag encoding.
func EncodeZigZag32(v int32) uint64 {
	return uint64(uint32(v<<1) ^ uint32(v>>31))
}

// EncodeZigZag64 encodes a signed int64 using ZigZag encoding.
func EncodeZigZag64(v int64) uint64 {
	return uint64(v<<1) ^ uint64(v>>63)
}

// SizeVarint returns the number of bytes needed to varint-encode v.
func SizeVarint(v uint64) int {
	// bits.Len64 returns 0 for v==0; we need 1 byte for zero.
	return int(9*uint32(bits.Len64(v))+64) / 64
}

// SizeTag returns the number of bytes needed to encode a field tag.
func SizeTag(fieldNumber int) int {
	return SizeVarint(uint64(fieldNumber) << 3)
}

// DecodeZigZag32 decodes a ZigZag-encoded uint64 back to int32.
func DecodeZigZag32(v uint64) int32 {
	return int32((uint32(v) >> 1) ^ -(uint32(v) & 1))
}

// DecodeZigZag64 decodes a ZigZag-encoded uint64 back to int64.
func DecodeZigZag64(v uint64) int64 {
	return int64((v >> 1) ^ -(v & 1))
}

// ConsumeVarint reads a varint from b, returning the value and bytes consumed.
// Returns (0, -1) if the buffer is too short, (0, -2) if the varint overflows.
func ConsumeVarint(b []byte) (uint64, int) {
	var x uint64
	var s uint
	for i, c := range b {
		if i == 10 {
			return 0, -2 // overflow
		}
		if c < 0x80 {
			if i == 9 && c > 1 {
				return 0, -2 // overflow
			}
			return x | uint64(c)<<s, i + 1
		}
		x |= uint64(c&0x7f) << s
		s += 7
	}
	return 0, -1 // truncated
}

// ConsumeFixed32 reads 4 bytes little-endian from b.
// Returns (0, -1) if the buffer is too short.
func ConsumeFixed32(b []byte) (uint32, int) {
	if len(b) < 4 {
		return 0, -1
	}
	return binary.LittleEndian.Uint32(b), 4
}

// ConsumeFixed64 reads 8 bytes little-endian from b.
// Returns (0, -1) if the buffer is too short.
func ConsumeFixed64(b []byte) (uint64, int) {
	if len(b) < 8 {
		return 0, -1
	}
	return binary.LittleEndian.Uint64(b), 8
}

// ConsumeBytes reads a length-delimited byte slice from b.
// Returns (nil, -1) if truncated, (nil, -2) if length overflows.
func ConsumeBytes(b []byte) ([]byte, int) {
	l, n := ConsumeVarint(b)
	if n < 0 {
		return nil, n
	}
	if uint64(len(b)-n) < l {
		return nil, -1
	}
	// Safe to convert l to int: the check above guarantees l <= len(b)-n,
	// and len(b)-n fits in int by definition, so int(l) cannot overflow
	// on any platform where this code can run.
	return b[n : n+int(l)], n + int(l)
}

// ErrTruncated is returned when the input buffer ends unexpectedly.
var ErrTruncated = errorString("protobuf: truncated message")

// ErrOverflow is returned when a varint overflows 64 bits.
var ErrOverflow = errorString("protobuf: varint overflow")

// ErrWireType is returned when a field's wire type does not match the expected type.
var ErrWireType = errorString("protobuf: wrong wire type")

// ErrPackedLen is returned when a packed field payload length is not a multiple of element size.
var ErrPackedLen = errorString("protobuf: packed field length mismatch")

// ErrDuplicateField is returned when a non-repeated field appears more than once.
var ErrDuplicateField = errorString("protobuf: duplicate non-repeated field")

// ErrUnknownWireType is returned when SkipField encounters an unrecognized wire type.
// Wire types 0-5 are defined by the protobuf spec; any other value is invalid.
var ErrUnknownWireType = errorString("protobuf: unknown wire type")

// ErrNestingDepth is returned when message nesting exceeds DefaultRecursionLimit.
var ErrNestingDepth = errorString("protobuf: message nesting depth exceeded")

// DefaultRecursionLimit is the maximum message nesting depth allowed during
// unmarshal. Generated UnmarshalBinary / UnmarshalBinaryLenient start with
// this budget and decrement it on each nested message call.
// 100 is sufficient for any realistic business schema; deeply nested messages
// indicate a design problem and are rejected early to prevent stack exhaustion.
const DefaultRecursionLimit = 100

// errorString is a simple error type to avoid importing errors package.
type errorString string

func (e errorString) Error() string { return string(e) }

// ConsumeTag reads a field tag from b, returning (fieldNumber, wireType, bytesConsumed).
// Returns (-1, -1, n<0) on error.
func ConsumeTag(b []byte) (int, int, int) {
	v, n := ConsumeVarint(b)
	if n < 0 {
		return -1, -1, n
	}
	return int(v >> 3), int(v & 0x7), n
}

// SkipField skips a field value given its wire type.
// Returns the number of bytes consumed, or a negative error code:
// -1 for truncated input, -3 for an unrecognized wire type.
func SkipField(b []byte, wireType int) int {
	switch wireType {
	case WireVarint:
		_, n := ConsumeVarint(b)
		return n
	case WireFixed64:
		if len(b) < 8 {
			return -1
		}
		return 8
	case WireBytes:
		_, n := ConsumeBytes(b)
		return n
	case WireFixed32:
		if len(b) < 4 {
			return -1
		}
		return 4
	default:
		return -3 // unrecognized wire type
	}
}
