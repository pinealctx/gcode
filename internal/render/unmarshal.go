package render

import (
	"fmt"
	"strings"

	"github.com/pinealctx/gcode/internal/model"
	"github.com/pinealctx/gcode/internal/transform"
)

// writeUnmarshalMethods generates unmarshalFrom, UnmarshalBinary, and
// UnmarshalBinaryLenient for a single message struct.
func writeUnmarshalMethods(b *strings.Builder, msg transform.GoMessage) error {
	if err := writeUnmarshalCore(b, msg); err != nil {
		return err
	}
	writeUnmarshalBinary(b, msg)
	writeUnmarshalBinaryLenient(b, msg)
	return nil
}

// writeUnmarshalBinary generates the public UnmarshalBinary entry point.
// Duplicate non-repeated fields return ErrDuplicateField.
func writeUnmarshalBinary(b *strings.Builder, msg transform.GoMessage) {
	recv := receiverName(msg.GoName)
	fmt.Fprintf(b, "// UnmarshalBinary implements encoding.BinaryUnmarshaler.\n")
	fmt.Fprintf(b, "// Duplicate non-repeated fields return an error.\n")
	fmt.Fprintf(b, "func (%s *%s) UnmarshalBinary(data []byte) error {\n", recv, msg.GoName)
	fmt.Fprintf(b, "\t_, err := %s.unmarshalFrom(data, false)\n", recv)
	fmt.Fprintf(b, "\treturn err\n}\n\n")
}

// writeUnmarshalBinaryLenient generates the lenient variant (last-one-wins for duplicates).
func writeUnmarshalBinaryLenient(b *strings.Builder, msg transform.GoMessage) {
	recv := receiverName(msg.GoName)
	fmt.Fprintf(b, "// UnmarshalBinaryLenient unmarshals like UnmarshalBinary but allows\n")
	fmt.Fprintf(b, "// duplicate non-repeated fields, keeping the last value.\n")
	fmt.Fprintf(b, "func (%s *%s) UnmarshalBinaryLenient(data []byte) error {\n", recv, msg.GoName)
	fmt.Fprintf(b, "\t_, err := %s.unmarshalFrom(data, true)\n", recv)
	fmt.Fprintf(b, "\treturn err\n}\n\n")
}

// writeUnmarshalCore generates the internal unmarshalFrom method that drives
// the decode loop. lenient=true enables last-one-wins for non-repeated fields.
// Returns an error if the message has more than 128 non-repeated fields.
func writeUnmarshalCore(b *strings.Builder, msg transform.GoMessage) error {
	recv := receiverName(msg.GoName)

	fmt.Fprintf(b, "// unmarshalFrom decodes a protobuf wire-format message from b.\n")
	fmt.Fprintf(b, "// Returns the number of bytes consumed.\n")
	fmt.Fprintf(b, "// If lenient is true, duplicate non-repeated fields use last-one-wins.\n")
	fmt.Fprintf(b, "func (%s *%s) unmarshalFrom(b []byte, lenient bool) (int, error) {\n", recv, msg.GoName)

	// Track seen non-repeated fields for duplicate detection.
	// We use a [2]uint64 bitmask (128 bits) and assign each non-repeated field
	// a bit index (0-based) at generation time. Messages with more than 128
	// non-repeated fields are rejected: such flat structures indicate a design
	// problem and should use nested messages or repeated fields instead.
	seenIdx, err := buildSeenIndex(msg)
	if err != nil {
		return err
	}
	if len(seenIdx) > 0 {
		fmt.Fprintf(b, "\tvar seen [2]uint64\n")
	}

	fmt.Fprintf(b, "\toff := 0\n")
	fmt.Fprintf(b, "\tfor off < len(b) {\n")

	// Read tag.
	fmt.Fprintf(b, "\t\tfieldNum, wireType, n := runtime.ConsumeTag(b[off:])\n")
	fmt.Fprintf(b, "\t\tif n < 0 {\n")
	fmt.Fprintf(b, "\t\t\tif n == -2 { return 0, fmt.Errorf(\"field tag: %%w\", runtime.ErrOverflow) }\n")
	fmt.Fprintf(b, "\t\t\treturn 0, fmt.Errorf(\"field tag: %%w\", runtime.ErrTruncated)\n")
	fmt.Fprintf(b, "\t\t}\n")
	fmt.Fprintf(b, "\t\toff += n\n\n")

	// Dispatch on field number.
	fmt.Fprintf(b, "\t\tswitch fieldNum {\n")
	for _, f := range msg.Fields {
		idx, tracked := seenIdx[f.Number]
		writeUnmarshalFieldCase(b, recv, f, idx, tracked)
	}
	// Unknown field: skip.
	fmt.Fprintf(b, "\t\tdefault:\n")
	fmt.Fprintf(b, "\t\t\tn = runtime.SkipField(b[off:], wireType)\n")
	fmt.Fprintf(b, "\t\t\tif n < 0 {\n")
	fmt.Fprintf(b, "\t\t\t\treturn 0, fmt.Errorf(\"unknown field %%d: %%w\", fieldNum, runtime.ErrTruncated)\n")
	fmt.Fprintf(b, "\t\t\t}\n")
	fmt.Fprintf(b, "\t\t\toff += n\n")
	fmt.Fprintf(b, "\t\t}\n") // end switch
	fmt.Fprintf(b, "\t}\n")   // end for
	fmt.Fprintf(b, "\treturn off, nil\n}\n\n")
	return nil
}

// buildSeenIndex assigns a bitmask index (0-based) to each non-repeated field
// for duplicate detection. Returns map[fieldNumber]bitIndex.
// Returns an error if the message has more than 128 non-repeated fields.
func buildSeenIndex(msg transform.GoMessage) (map[int]int, error) {
	idx := make(map[int]int)
	bit := 0
	for _, f := range msg.Fields {
		if f.Cardinality != model.CardinalityRepeated {
			if bit >= 128 {
				return nil, fmt.Errorf("message %q has more than 128 non-repeated fields; use nested messages or repeated fields to reduce field count", msg.GoName)
			}
			idx[f.Number] = bit
			bit++
		}
	}
	return idx, nil
}

// writeUnmarshalFieldCase generates the case branch for a single field.
func writeUnmarshalFieldCase(b *strings.Builder, recv string, f transform.GoField, seenBit int, tracked bool) {
	accessor := fmt.Sprintf("%s.%s", recv, f.GoName)
	fmt.Fprintf(b, "\t\tcase %d:\n", f.Number)

	if f.Cardinality == model.CardinalityRepeated {
		writeUnmarshalRepeatedField(b, accessor, f)
		return
	}

	// Singular field: check for duplicate, then decode.
	// seen is [2]uint64: slot = seenBit/64, bit = 1 << (seenBit%64).
	if tracked {
		slot := seenBit / 64
		bit := uint64(1) << (seenBit % 64)
		fmt.Fprintf(b, "\t\t\tif seen[%d]&%d != 0 {\n", slot, bit)
		fmt.Fprintf(b, "\t\t\t\tif !lenient { return 0, fmt.Errorf(\"field %d: %%w\", runtime.ErrDuplicateField) }\n", f.Number)
		fmt.Fprintf(b, "\t\t\t}\n")
		fmt.Fprintf(b, "\t\t\tseen[%d] |= %d\n", slot, bit)
	}

	writeUnmarshalSingularField(b, recv, accessor, f)
}

// writeUnmarshalSingularField generates decode logic for a singular field.
func writeUnmarshalSingularField(b *strings.Builder, _ string, accessor string, f transform.GoField) {
	switch f.Type.Kind {
	case model.FieldKindScalar:
		writeUnmarshalScalar(b, accessor, f.Number, f.Type.Scalar, f.Optional)
	case model.FieldKindEnum:
		// Enum is varint-encoded.
		fmt.Fprintf(b, "\t\t\tif wireType != runtime.WireVarint { return 0, fmt.Errorf(\"field %d: %%w\", runtime.ErrWireType) }\n", f.Number)
		fmt.Fprintf(b, "\t\t\tv, n := runtime.ConsumeVarint(b[off:])\n")
		fmt.Fprintf(b, "\t\t\tif n < 0 {\n")
		fmt.Fprintf(b, "\t\t\t\tif n == -2 { return 0, fmt.Errorf(\"field %d: %%w\", runtime.ErrOverflow) }\n", f.Number)
		fmt.Fprintf(b, "\t\t\t\treturn 0, fmt.Errorf(\"field %d: %%w\", runtime.ErrTruncated)\n", f.Number)
		fmt.Fprintf(b, "\t\t\t}\n")
		if f.Optional {
			// optional enum: allocate pointer
			baseType := strings.TrimPrefix(f.GoType, "*")
			fmt.Fprintf(b, "\t\t\ttmp := %s(v)\n", baseType)
			fmt.Fprintf(b, "\t\t\t%s = &tmp\n", accessor)
		} else {
			fmt.Fprintf(b, "\t\t\t%s = %s(v)\n", accessor, f.GoType)
		}
		fmt.Fprintf(b, "\t\t\toff += n\n")
	case model.FieldKindMessage:
		// Message is length-delimited.
		fmt.Fprintf(b, "\t\t\tif wireType != runtime.WireBytes { return 0, fmt.Errorf(\"field %d: %%w\", runtime.ErrWireType) }\n", f.Number)
		fmt.Fprintf(b, "\t\t\tpayload, n := runtime.ConsumeBytes(b[off:])\n")
		fmt.Fprintf(b, "\t\t\tif n < 0 {\n")
		fmt.Fprintf(b, "\t\t\t\tif n == -2 { return 0, fmt.Errorf(\"field %d: %%w\", runtime.ErrOverflow) }\n", f.Number)
		fmt.Fprintf(b, "\t\t\t\treturn 0, fmt.Errorf(\"field %d: %%w\", runtime.ErrTruncated)\n", f.Number)
		fmt.Fprintf(b, "\t\t\t}\n")
		// GoType is "*TypeName" — strip the leading "*" for new().
		goTypeName := strings.TrimPrefix(f.GoType, "*")
		fmt.Fprintf(b, "\t\t\tif %s == nil { %s = new(%s) }\n", accessor, accessor, goTypeName)
		fmt.Fprintf(b, "\t\t\tif _, err := %s.unmarshalFrom(payload, lenient); err != nil {\n", accessor)
		fmt.Fprintf(b, "\t\t\t\treturn 0, fmt.Errorf(\"field %d: %%w\", err)\n", f.Number)
		fmt.Fprintf(b, "\t\t\t}\n")
		fmt.Fprintf(b, "\t\t\toff += n\n")
	}
}

// writeUnmarshalScalar generates decode logic for a singular scalar field.
// If optional is true, the decoded value is stored via a pointer (tmp := val; m.F = &tmp).
func writeUnmarshalScalar(b *strings.Builder, accessor string, fieldNum int, scalar model.ScalarKind, optional bool) {
	expectedWT := scalarWireType(scalar)
	fmt.Fprintf(b, "\t\t\tif wireType != %s { return 0, fmt.Errorf(\"field %d: %%w\", runtime.ErrWireType) }\n",
		wireTypeConst(expectedWT), fieldNum)

	// assign writes the final assignment, wrapping in a pointer if optional.
	assign := func(expr string) {
		if optional {
			fmt.Fprintf(b, "\t\t\ttmp := %s\n", expr)
			fmt.Fprintf(b, "\t\t\t%s = &tmp\n", accessor)
		} else {
			fmt.Fprintf(b, "\t\t\t%s = %s\n", accessor, expr)
		}
	}

	switch scalar {
	case model.ScalarBool:
		fmt.Fprintf(b, "\t\t\tv, n := runtime.ConsumeVarint(b[off:])\n")
		fmt.Fprintf(b, "\t\t\tif n < 0 { return 0, fmt.Errorf(\"field %d: %%w\", runtime.ErrTruncated) }\n", fieldNum)
		assign("v != 0")
		fmt.Fprintf(b, "\t\t\toff += n\n")
	case model.ScalarInt32:
		fmt.Fprintf(b, "\t\t\tv, n := runtime.ConsumeVarint(b[off:])\n")
		fmt.Fprintf(b, "\t\t\tif n < 0 { return 0, fmt.Errorf(\"field %d: %%w\", runtime.ErrTruncated) }\n", fieldNum)
		assign("int32(v)")
		fmt.Fprintf(b, "\t\t\toff += n\n")
	case model.ScalarInt64:
		fmt.Fprintf(b, "\t\t\tv, n := runtime.ConsumeVarint(b[off:])\n")
		fmt.Fprintf(b, "\t\t\tif n < 0 { return 0, fmt.Errorf(\"field %d: %%w\", runtime.ErrTruncated) }\n", fieldNum)
		assign("int64(v)")
		fmt.Fprintf(b, "\t\t\toff += n\n")
	case model.ScalarUint32:
		fmt.Fprintf(b, "\t\t\tv, n := runtime.ConsumeVarint(b[off:])\n")
		fmt.Fprintf(b, "\t\t\tif n < 0 { return 0, fmt.Errorf(\"field %d: %%w\", runtime.ErrTruncated) }\n", fieldNum)
		assign("uint32(v)")
		fmt.Fprintf(b, "\t\t\toff += n\n")
	case model.ScalarUint64:
		fmt.Fprintf(b, "\t\t\tv, n := runtime.ConsumeVarint(b[off:])\n")
		fmt.Fprintf(b, "\t\t\tif n < 0 { return 0, fmt.Errorf(\"field %d: %%w\", runtime.ErrTruncated) }\n", fieldNum)
		assign("v")
		fmt.Fprintf(b, "\t\t\toff += n\n")
	case model.ScalarSint32:
		fmt.Fprintf(b, "\t\t\tv, n := runtime.ConsumeVarint(b[off:])\n")
		fmt.Fprintf(b, "\t\t\tif n < 0 { return 0, fmt.Errorf(\"field %d: %%w\", runtime.ErrTruncated) }\n", fieldNum)
		assign("runtime.DecodeZigZag32(v)")
		fmt.Fprintf(b, "\t\t\toff += n\n")
	case model.ScalarSint64:
		fmt.Fprintf(b, "\t\t\tv, n := runtime.ConsumeVarint(b[off:])\n")
		fmt.Fprintf(b, "\t\t\tif n < 0 { return 0, fmt.Errorf(\"field %d: %%w\", runtime.ErrTruncated) }\n", fieldNum)
		assign("runtime.DecodeZigZag64(v)")
		fmt.Fprintf(b, "\t\t\toff += n\n")
	case model.ScalarFixed32:
		fmt.Fprintf(b, "\t\t\tv, n := runtime.ConsumeFixed32(b[off:])\n")
		fmt.Fprintf(b, "\t\t\tif n < 0 { return 0, fmt.Errorf(\"field %d: %%w\", runtime.ErrTruncated) }\n", fieldNum)
		assign("v")
		fmt.Fprintf(b, "\t\t\toff += n\n")
	case model.ScalarSfixed32:
		fmt.Fprintf(b, "\t\t\tv, n := runtime.ConsumeFixed32(b[off:])\n")
		fmt.Fprintf(b, "\t\t\tif n < 0 { return 0, fmt.Errorf(\"field %d: %%w\", runtime.ErrTruncated) }\n", fieldNum)
		assign("int32(v)")
		fmt.Fprintf(b, "\t\t\toff += n\n")
	case model.ScalarFloat:
		fmt.Fprintf(b, "\t\t\tv, n := runtime.ConsumeFixed32(b[off:])\n")
		fmt.Fprintf(b, "\t\t\tif n < 0 { return 0, fmt.Errorf(\"field %d: %%w\", runtime.ErrTruncated) }\n", fieldNum)
		assign("math.Float32frombits(v)")
		fmt.Fprintf(b, "\t\t\toff += n\n")
	case model.ScalarFixed64:
		fmt.Fprintf(b, "\t\t\tv, n := runtime.ConsumeFixed64(b[off:])\n")
		fmt.Fprintf(b, "\t\t\tif n < 0 { return 0, fmt.Errorf(\"field %d: %%w\", runtime.ErrTruncated) }\n", fieldNum)
		assign("v")
		fmt.Fprintf(b, "\t\t\toff += n\n")
	case model.ScalarSfixed64:
		fmt.Fprintf(b, "\t\t\tv, n := runtime.ConsumeFixed64(b[off:])\n")
		fmt.Fprintf(b, "\t\t\tif n < 0 { return 0, fmt.Errorf(\"field %d: %%w\", runtime.ErrTruncated) }\n", fieldNum)
		assign("int64(v)")
		fmt.Fprintf(b, "\t\t\toff += n\n")
	case model.ScalarDouble:
		fmt.Fprintf(b, "\t\t\tv, n := runtime.ConsumeFixed64(b[off:])\n")
		fmt.Fprintf(b, "\t\t\tif n < 0 { return 0, fmt.Errorf(\"field %d: %%w\", runtime.ErrTruncated) }\n", fieldNum)
		assign("math.Float64frombits(v)")
		fmt.Fprintf(b, "\t\t\toff += n\n")
	case model.ScalarString:
		fmt.Fprintf(b, "\t\t\tpayload, n := runtime.ConsumeBytes(b[off:])\n")
		fmt.Fprintf(b, "\t\t\tif n < 0 { return 0, fmt.Errorf(\"field %d: %%w\", runtime.ErrTruncated) }\n", fieldNum)
		assign("string(payload)")
		fmt.Fprintf(b, "\t\t\toff += n\n")
	case model.ScalarBytes:
		fmt.Fprintf(b, "\t\t\tpayload, n := runtime.ConsumeBytes(b[off:])\n")
		fmt.Fprintf(b, "\t\t\tif n < 0 { return 0, fmt.Errorf(\"field %d: %%w\", runtime.ErrTruncated) }\n", fieldNum)
		fmt.Fprintf(b, "\t\t\ttmp := make([]byte, len(payload))\n")
		fmt.Fprintf(b, "\t\t\tcopy(tmp, payload)\n")
		if optional {
			fmt.Fprintf(b, "\t\t\t%s = &tmp\n", accessor)
		} else {
			fmt.Fprintf(b, "\t\t\t%s = tmp\n", accessor)
		}
		fmt.Fprintf(b, "\t\t\toff += n\n")
	}
}

// writeUnmarshalRepeatedField generates decode logic for a repeated field.
func writeUnmarshalRepeatedField(b *strings.Builder, accessor string, f transform.GoField) {
	switch f.Type.Kind {
	case model.FieldKindScalar:
		if isPackable(f.Type.Scalar) {
			writeUnmarshalPackedScalar(b, accessor, f.Number, f.Type.Scalar)
		} else {
			writeUnmarshalRepeatedLEN(b, accessor, f.Number, f.Type.Scalar)
		}
	case model.FieldKindEnum:
		writeUnmarshalPackedEnum(b, accessor, f.Number, f.GoType)
	case model.FieldKindMessage:
		writeUnmarshalRepeatedMessage(b, accessor, f)
	}
}

// writeUnmarshalPackedScalar generates decode for a packed repeated scalar field.
// Per §6.3: only packed encoding is accepted; unpacked wire type returns ErrWireType.
func writeUnmarshalPackedScalar(b *strings.Builder, accessor string, fieldNum int, scalar model.ScalarKind) {
	fmt.Fprintf(b, "\t\t\tif wireType != runtime.WireBytes { return 0, fmt.Errorf(\"field %d: %%w\", runtime.ErrWireType) }\n", fieldNum)
	fmt.Fprintf(b, "\t\t\tpacked, n := runtime.ConsumeBytes(b[off:])\n")
	fmt.Fprintf(b, "\t\t\tif n < 0 {\n")
	fmt.Fprintf(b, "\t\t\t\tif n == -2 { return 0, fmt.Errorf(\"field %d: %%w\", runtime.ErrOverflow) }\n", fieldNum)
	fmt.Fprintf(b, "\t\t\t\treturn 0, fmt.Errorf(\"field %d: %%w\", runtime.ErrTruncated)\n", fieldNum)
	fmt.Fprintf(b, "\t\t\t}\n")
	fmt.Fprintf(b, "\t\t\toff += n\n")

	// Decode elements from packed payload.
	switch scalar {
	case model.ScalarFixed32, model.ScalarSfixed32, model.ScalarFloat:
		fmt.Fprintf(b, "\t\t\tif len(packed)%%4 != 0 { return 0, fmt.Errorf(\"field %d: %%w\", runtime.ErrPackedLen) }\n", fieldNum)
		fmt.Fprintf(b, "\t\t\tfor pi := 0; pi < len(packed); pi += 4 {\n")
		fmt.Fprintf(b, "\t\t\t\tv, _ := runtime.ConsumeFixed32(packed[pi:])\n")
		writePackedFixed32Assign(b, accessor, scalar)
		fmt.Fprintf(b, "\t\t\t}\n")
	case model.ScalarFixed64, model.ScalarSfixed64, model.ScalarDouble:
		fmt.Fprintf(b, "\t\t\tif len(packed)%%8 != 0 { return 0, fmt.Errorf(\"field %d: %%w\", runtime.ErrPackedLen) }\n", fieldNum)
		fmt.Fprintf(b, "\t\t\tfor pi := 0; pi < len(packed); pi += 8 {\n")
		fmt.Fprintf(b, "\t\t\t\tv, _ := runtime.ConsumeFixed64(packed[pi:])\n")
		writePackedFixed64Assign(b, accessor, scalar)
		fmt.Fprintf(b, "\t\t\t}\n")
	default:
		// Varint elements.
		fmt.Fprintf(b, "\t\t\tfor pi := 0; pi < len(packed); {\n")
		fmt.Fprintf(b, "\t\t\t\tv, pn := runtime.ConsumeVarint(packed[pi:])\n")
		fmt.Fprintf(b, "\t\t\t\tif pn < 0 {\n")
		fmt.Fprintf(b, "\t\t\t\t\tif pn == -2 { return 0, fmt.Errorf(\"field %d: %%w\", runtime.ErrOverflow) }\n", fieldNum)
		fmt.Fprintf(b, "\t\t\t\t\treturn 0, fmt.Errorf(\"field %d: %%w\", runtime.ErrTruncated)\n", fieldNum)
		fmt.Fprintf(b, "\t\t\t\t}\n")
		writePackedVarintAssign(b, accessor, scalar)
		fmt.Fprintf(b, "\t\t\t\tpi += pn\n")
		fmt.Fprintf(b, "\t\t\t}\n")
	}
}

func writePackedFixed32Assign(b *strings.Builder, accessor string, scalar model.ScalarKind) {
	switch scalar {
	case model.ScalarFixed32:
		fmt.Fprintf(b, "\t\t\t\t%s = append(%s, v)\n", accessor, accessor)
	case model.ScalarSfixed32:
		fmt.Fprintf(b, "\t\t\t\t%s = append(%s, int32(v))\n", accessor, accessor)
	case model.ScalarFloat:
		fmt.Fprintf(b, "\t\t\t\t%s = append(%s, math.Float32frombits(v))\n", accessor, accessor)
	}
}

func writePackedFixed64Assign(b *strings.Builder, accessor string, scalar model.ScalarKind) {
	switch scalar {
	case model.ScalarFixed64:
		fmt.Fprintf(b, "\t\t\t\t%s = append(%s, v)\n", accessor, accessor)
	case model.ScalarSfixed64:
		fmt.Fprintf(b, "\t\t\t\t%s = append(%s, int64(v))\n", accessor, accessor)
	case model.ScalarDouble:
		fmt.Fprintf(b, "\t\t\t\t%s = append(%s, math.Float64frombits(v))\n", accessor, accessor)
	}
}

func writePackedVarintAssign(b *strings.Builder, accessor string, scalar model.ScalarKind) {
	switch scalar {
	case model.ScalarBool:
		fmt.Fprintf(b, "\t\t\t\t%s = append(%s, v != 0)\n", accessor, accessor)
	case model.ScalarInt32:
		fmt.Fprintf(b, "\t\t\t\t%s = append(%s, int32(v))\n", accessor, accessor)
	case model.ScalarInt64:
		fmt.Fprintf(b, "\t\t\t\t%s = append(%s, int64(v))\n", accessor, accessor)
	case model.ScalarUint32:
		fmt.Fprintf(b, "\t\t\t\t%s = append(%s, uint32(v))\n", accessor, accessor)
	case model.ScalarUint64:
		fmt.Fprintf(b, "\t\t\t\t%s = append(%s, v)\n", accessor, accessor)
	case model.ScalarSint32:
		fmt.Fprintf(b, "\t\t\t\t%s = append(%s, runtime.DecodeZigZag32(v))\n", accessor, accessor)
	case model.ScalarSint64:
		fmt.Fprintf(b, "\t\t\t\t%s = append(%s, runtime.DecodeZigZag64(v))\n", accessor, accessor)
	}
}

// writeUnmarshalPackedEnum generates decode for a packed repeated enum field.
func writeUnmarshalPackedEnum(b *strings.Builder, accessor string, fieldNum int, goType string) {
	fmt.Fprintf(b, "\t\t\tif wireType != runtime.WireBytes { return 0, fmt.Errorf(\"field %d: %%w\", runtime.ErrWireType) }\n", fieldNum)
	fmt.Fprintf(b, "\t\t\tpacked, n := runtime.ConsumeBytes(b[off:])\n")
	fmt.Fprintf(b, "\t\t\tif n < 0 {\n")
	fmt.Fprintf(b, "\t\t\t\tif n == -2 { return 0, fmt.Errorf(\"field %d: %%w\", runtime.ErrOverflow) }\n", fieldNum)
	fmt.Fprintf(b, "\t\t\t\treturn 0, fmt.Errorf(\"field %d: %%w\", runtime.ErrTruncated)\n", fieldNum)
	fmt.Fprintf(b, "\t\t\t}\n")
	fmt.Fprintf(b, "\t\t\toff += n\n")
	fmt.Fprintf(b, "\t\t\tfor pi := 0; pi < len(packed); {\n")
	fmt.Fprintf(b, "\t\t\t\tv, pn := runtime.ConsumeVarint(packed[pi:])\n")
	fmt.Fprintf(b, "\t\t\t\tif pn < 0 {\n")
	fmt.Fprintf(b, "\t\t\t\t\tif pn == -2 { return 0, fmt.Errorf(\"field %d: %%w\", runtime.ErrOverflow) }\n", fieldNum)
	fmt.Fprintf(b, "\t\t\t\t\treturn 0, fmt.Errorf(\"field %d: %%w\", runtime.ErrTruncated)\n", fieldNum)
	fmt.Fprintf(b, "\t\t\t\t}\n")
	// The element type is the enum slice element type — strip the "[]" prefix.
	elemType := strings.TrimPrefix(goType, "[]")
	fmt.Fprintf(b, "\t\t\t\t%s = append(%s, %s(int32(v)))\n", accessor, accessor, elemType)
	fmt.Fprintf(b, "\t\t\t\tpi += pn\n")
	fmt.Fprintf(b, "\t\t\t}\n")
}

// writeUnmarshalRepeatedLEN generates decode for repeated string/bytes (not packed).
func writeUnmarshalRepeatedLEN(b *strings.Builder, accessor string, fieldNum int, scalar model.ScalarKind) {
	fmt.Fprintf(b, "\t\t\tif wireType != runtime.WireBytes { return 0, fmt.Errorf(\"field %d: %%w\", runtime.ErrWireType) }\n", fieldNum)
	fmt.Fprintf(b, "\t\t\tpayload, n := runtime.ConsumeBytes(b[off:])\n")
	fmt.Fprintf(b, "\t\t\tif n < 0 { return 0, fmt.Errorf(\"field %d: %%w\", runtime.ErrTruncated) }\n", fieldNum)
	switch scalar {
	case model.ScalarString:
		fmt.Fprintf(b, "\t\t\t%s = append(%s, string(payload))\n", accessor, accessor)
	case model.ScalarBytes:
		fmt.Fprintf(b, "\t\t\ttmp := make([]byte, len(payload))\n")
		fmt.Fprintf(b, "\t\t\tcopy(tmp, payload)\n")
		fmt.Fprintf(b, "\t\t\t%s = append(%s, tmp)\n", accessor, accessor)
	}
	fmt.Fprintf(b, "\t\t\toff += n\n")
}

// writeUnmarshalRepeatedMessage generates decode for repeated message fields.
func writeUnmarshalRepeatedMessage(b *strings.Builder, accessor string, f transform.GoField) {
	fmt.Fprintf(b, "\t\t\tif wireType != runtime.WireBytes { return 0, fmt.Errorf(\"field %d: %%w\", runtime.ErrWireType) }\n", f.Number)
	fmt.Fprintf(b, "\t\t\tpayload, n := runtime.ConsumeBytes(b[off:])\n")
	fmt.Fprintf(b, "\t\t\tif n < 0 {\n")
	fmt.Fprintf(b, "\t\t\t\tif n == -2 { return 0, fmt.Errorf(\"field %d: %%w\", runtime.ErrOverflow) }\n", f.Number)
	fmt.Fprintf(b, "\t\t\t\treturn 0, fmt.Errorf(\"field %d: %%w\", runtime.ErrTruncated)\n", f.Number)
	fmt.Fprintf(b, "\t\t\t}\n")
	// GoType is "[]*TypeName" — strip "[]" and "*" to get the base type name.
	goTypeName := strings.TrimPrefix(strings.TrimPrefix(f.GoType, "[]"), "*")
	fmt.Fprintf(b, "\t\t\telem := new(%s)\n", goTypeName)
	fmt.Fprintf(b, "\t\t\tif _, err := elem.unmarshalFrom(payload, lenient); err != nil {\n")
	fmt.Fprintf(b, "\t\t\t\treturn 0, fmt.Errorf(\"field %d: %%w\", err)\n", f.Number)
	fmt.Fprintf(b, "\t\t\t}\n")
	fmt.Fprintf(b, "\t\t\t%s = append(%s, elem)\n", accessor, accessor)
	fmt.Fprintf(b, "\t\t\toff += n\n")
}

// scalarWireType returns the expected wire type for a scalar kind.
func scalarWireType(scalar model.ScalarKind) int {
	switch scalar {
	case model.ScalarFixed32, model.ScalarSfixed32, model.ScalarFloat:
		return 5 // WireFixed32
	case model.ScalarFixed64, model.ScalarSfixed64, model.ScalarDouble:
		return 1 // WireFixed64
	case model.ScalarString, model.ScalarBytes:
		return 2 // WireBytes
	default:
		return 0 // WireVarint
	}
}
