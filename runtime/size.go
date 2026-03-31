package runtime

import "math"

// SizeBool returns the wire size of a bool field value (always 1).
func SizeBool(_ bool) int { return 1 }

// SizeInt32 returns the varint-encoded size of an int32.
// Negative values are sign-extended to 10 bytes.
func SizeInt32(v int32) int { return SizeVarint(uint64(v)) }

// SizeInt64 returns the varint-encoded size of an int64.
func SizeInt64(v int64) int { return SizeVarint(uint64(v)) }

// SizeUint32 returns the varint-encoded size of a uint32.
func SizeUint32(v uint32) int { return SizeVarint(uint64(v)) }

// SizeUint64 returns the varint-encoded size of a uint64.
func SizeUint64(v uint64) int { return SizeVarint(v) }

// SizeSint32 returns the varint-encoded size of a ZigZag-encoded sint32.
func SizeSint32(v int32) int { return SizeVarint(EncodeZigZag32(v)) }

// SizeSint64 returns the varint-encoded size of a ZigZag-encoded sint64.
func SizeSint64(v int64) int { return SizeVarint(EncodeZigZag64(v)) }

// SizeFloat returns the wire size of a float (always 4).
func SizeFloat(_ float32) int { return 4 }

// SizeDouble returns the wire size of a double (always 8).
func SizeDouble(_ float64) int { return 8 }

// SizeFixed32 returns the wire size of a fixed32 (always 4).
func SizeFixed32(_ uint32) int { return 4 }

// SizeFixed64 returns the wire size of a fixed64 (always 8).
func SizeFixed64(_ uint64) int { return 8 }

// SizeSfixed32 returns the wire size of an sfixed32 (always 4).
func SizeSfixed32(_ int32) int { return 4 }

// SizeSfixed64 returns the wire size of an sfixed64 (always 8).
func SizeSfixed64(_ int64) int { return 8 }

// SizeString returns the wire size of a length-delimited string (length prefix + content).
func SizeString(v string) int { return SizeVarint(uint64(len(v))) + len(v) }

// SizeBytes returns the wire size of a length-delimited byte slice.
func SizeBytes(v []byte) int { return SizeVarint(uint64(len(v))) + len(v) }

// SizeEnum returns the varint-encoded size of an enum value (int32).
func SizeEnum(v int32) int { return SizeVarint(uint64(v)) }

// IsZeroFloat returns true if v is positive zero.
func IsZeroFloat(v float32) bool { return math.Float32bits(v) == 0 }

// IsZeroDouble returns true if v is positive zero.
func IsZeroDouble(v float64) bool { return math.Float64bits(v) == 0 }
