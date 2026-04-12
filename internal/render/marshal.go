package render

import (
	"fmt"
	"strings"

	"github.com/pinealctx/gcode/internal/model"
	"github.com/pinealctx/gcode/internal/transform"
)

// writeMarshalMethods generates Size, MarshalBinary, and MarshalAppend methods
// for a single message struct.
func writeMarshalMethods(b *strings.Builder, msg transform.GoMessage) {
	writeSize(b, msg)
	writeMarshalBinary(b, msg)
	writeMarshalAppend(b, msg)
}

// writeSize generates the Size() int method that calculates the total wire size.
func writeSize(b *strings.Builder, msg transform.GoMessage) {
	recv := receiverName(msg.GoName)
	fmt.Fprintf(b, "// Size returns the protobuf wire size of %s.\n", msg.GoName)
	fmt.Fprintf(b, "func (%s *%s) Size() int {\n", recv, msg.GoName)
	fmt.Fprintf(b, "if %s == nil { return 0 }\n", recv)
	b.WriteString("var n int\n")

	for _, f := range msg.Fields {
		writeSizeField(b, recv, f)
	}

	b.WriteString("return n\n}\n\n")
}

// writeMarshalBinary generates the MarshalBinary method (encoding.BinaryMarshaler).
func writeMarshalBinary(b *strings.Builder, msg transform.GoMessage) {
	recv := receiverName(msg.GoName)
	fmt.Fprintf(b, "// MarshalBinary implements encoding.BinaryMarshaler.\n")
	fmt.Fprintf(b, "func (%s *%s) MarshalBinary() ([]byte, error) {\n", recv, msg.GoName)
	fmt.Fprintf(b, "return %s.MarshalAppend(make([]byte, 0, %s.Size()))\n", recv, recv)
	b.WriteString("}\n\n")
}

// writeMarshalAppend generates the MarshalAppend method that appends wire bytes to b.
func writeMarshalAppend(b *strings.Builder, msg transform.GoMessage) {
	recv := receiverName(msg.GoName)
	fmt.Fprintf(b, "// MarshalAppend appends the protobuf wire encoding of %s to b.\n", msg.GoName)
	fmt.Fprintf(b, "func (%s *%s) MarshalAppend(b []byte) ([]byte, error) {\n", recv, msg.GoName)

	for _, f := range msg.Fields {
		writeMarshalField(b, recv, f)
	}

	b.WriteString("return b, nil\n}\n\n")
}

// receiverName returns "x" as the universal receiver name for generated methods,
// matching the protobuf-go convention and avoiding conflicts with parameter names.
func receiverName(_ string) string {
	return "x"
}

// writeSizeField generates the size calculation for a single field with zero-value skip.
func writeSizeField(b *strings.Builder, recv string, f transform.GoField) {
	accessor := fmt.Sprintf("%s.%s", recv, f.GoName)
	tagSize := tagSizeConst(f.Number)

	if f.Cardinality == model.CardinalityRepeated {
		writeSizeRepeated(b, accessor, tagSize, f)
		return
	}

	// Optional (pointer) field: nil check; deref for value access.
	if f.Optional {
		fmt.Fprintf(b, "if %s != nil {\n", accessor)
		deref := "*" + accessor
		switch f.Type.Kind {
		case model.FieldKindScalar:
			writeSizeScalar(b, deref, tagSize, f.Type.Scalar)
		case model.FieldKindEnum:
			fmt.Fprintf(b, "n += %d + runtime.SizeEnum(int32(%s))\n", tagSize, deref)
		}
		b.WriteString("}\n")
		return
	}

	// HasPresence bytes field (optional bytes): nil means absent, []byte{} means present-empty.
	if f.HasPresence && f.Type.Kind == model.FieldKindScalar && f.Type.Scalar == model.ScalarBytes {
		fmt.Fprintf(b, "if %s != nil {\n", accessor)
		fmt.Fprintf(b, "n += %d + runtime.SizeBytes(%s)\n", tagSize, accessor)
		b.WriteString("}\n")
		return
	}

	// Singular field: skip if zero value.
	zeroCheck := zeroValueCheck(accessor, f)
	fmt.Fprintf(b, "if %s {\n", zeroCheck)

	switch f.Type.Kind {
	case model.FieldKindScalar:
		writeSizeScalar(b, accessor, tagSize, f.Type.Scalar)
	case model.FieldKindEnum:
		fmt.Fprintf(b, "n += %d + runtime.SizeEnum(int32(%s))\n", tagSize, accessor)
	case model.FieldKindMessage:
		fmt.Fprintf(b, "s := %s.Size()\n", accessor)
		fmt.Fprintf(b, "n += %d + runtime.SizeVarint(uint64(s)) + s\n", tagSize)
	}

	b.WriteString("}\n")
}

// writeSizeRepeated generates size calculation for repeated fields.
func writeSizeRepeated(b *strings.Builder, accessor string, tagSize int, f transform.GoField) {
	fmt.Fprintf(b, "if len(%s) > 0 {\n", accessor)

	switch f.Type.Kind {
	case model.FieldKindScalar:
		if isPackable(f.Type.Scalar) {
			// Packed: tag + length prefix + sum of element sizes.
			writeSizePackedScalar(b, accessor, tagSize, f.Type.Scalar)
		} else {
			// string/bytes: each element is tag + length-delimited.
			writeSizeRepeatedLEN(b, accessor, tagSize, f.Type.Scalar)
		}
	case model.FieldKindEnum:
		// Packed enum.
		fmt.Fprintf(b, "var es int\n")
		fmt.Fprintf(b, "for _, v := range %s {\n", accessor)
		b.WriteString("es += runtime.SizeEnum(int32(v))\n}\n")
		fmt.Fprintf(b, "n += %d + runtime.SizeVarint(uint64(es)) + es\n", tagSize)
	case model.FieldKindMessage:
		// Each message is tag + length-delimited.
		fmt.Fprintf(b, "for _, v := range %s {\n", accessor)
		b.WriteString("s := v.Size()\n")
		fmt.Fprintf(b, "n += %d + runtime.SizeVarint(uint64(s)) + s\n", tagSize)
		b.WriteString("}\n")
	}

	b.WriteString("}\n")
}

// writeSizeScalar generates the size expression for a singular scalar.
func writeSizeScalar(b *strings.Builder, accessor string, tagSize int, scalar model.ScalarKind) {
	switch scalar {
	case model.ScalarBool:
		fmt.Fprintf(b, "n += %d + 1\n", tagSize)
	case model.ScalarInt32:
		fmt.Fprintf(b, "n += %d + runtime.SizeInt32(%s)\n", tagSize, accessor)
	case model.ScalarInt64:
		fmt.Fprintf(b, "n += %d + runtime.SizeInt64(%s)\n", tagSize, accessor)
	case model.ScalarUint32:
		fmt.Fprintf(b, "n += %d + runtime.SizeUint32(%s)\n", tagSize, accessor)
	case model.ScalarUint64:
		fmt.Fprintf(b, "n += %d + runtime.SizeUint64(%s)\n", tagSize, accessor)
	case model.ScalarSint32:
		fmt.Fprintf(b, "n += %d + runtime.SizeSint32(%s)\n", tagSize, accessor)
	case model.ScalarSint64:
		fmt.Fprintf(b, "n += %d + runtime.SizeSint64(%s)\n", tagSize, accessor)
	case model.ScalarFixed32, model.ScalarSfixed32, model.ScalarFloat:
		fmt.Fprintf(b, "n += %d + 4\n", tagSize)
	case model.ScalarFixed64, model.ScalarSfixed64, model.ScalarDouble:
		fmt.Fprintf(b, "n += %d + 8\n", tagSize)
	case model.ScalarString:
		fmt.Fprintf(b, "n += %d + runtime.SizeString(%s)\n", tagSize, accessor)
	case model.ScalarBytes:
		fmt.Fprintf(b, "n += %d + runtime.SizeBytes(%s)\n", tagSize, accessor)
	}
}

// writeSizePackedScalar generates size for a packed repeated scalar field.
func writeSizePackedScalar(b *strings.Builder, accessor string, tagSize int, scalar model.ScalarKind) {
	switch scalar {
	case model.ScalarBool:
		// Each bool is 1 byte.
		fmt.Fprintf(b, "es := len(%s)\n", accessor)
	case model.ScalarFixed32, model.ScalarSfixed32, model.ScalarFloat:
		fmt.Fprintf(b, "es := len(%s) * 4\n", accessor)
	case model.ScalarFixed64, model.ScalarSfixed64, model.ScalarDouble:
		fmt.Fprintf(b, "es := len(%s) * 8\n", accessor)
	default:
		// Variable-length varint elements.
		b.WriteString("var es int\n")
		fmt.Fprintf(b, "for _, v := range %s {\n", accessor)
		switch scalar {
		case model.ScalarInt32:
			b.WriteString("es += runtime.SizeInt32(v)\n")
		case model.ScalarInt64:
			b.WriteString("es += runtime.SizeInt64(v)\n")
		case model.ScalarUint32:
			b.WriteString("es += runtime.SizeUint32(v)\n")
		case model.ScalarUint64:
			b.WriteString("es += runtime.SizeUint64(v)\n")
		case model.ScalarSint32:
			b.WriteString("es += runtime.SizeSint32(v)\n")
		case model.ScalarSint64:
			b.WriteString("es += runtime.SizeSint64(v)\n")
		}
		b.WriteString("}\n")
	}
	fmt.Fprintf(b, "n += %d + runtime.SizeVarint(uint64(es)) + es\n", tagSize)
}

// writeSizeRepeatedLEN generates size for repeated string/bytes (not packed).
func writeSizeRepeatedLEN(b *strings.Builder, accessor string, tagSize int, scalar model.ScalarKind) {
	fmt.Fprintf(b, "for _, v := range %s {\n", accessor)
	switch scalar {
	case model.ScalarString:
		fmt.Fprintf(b, "n += %d + runtime.SizeString(v)\n", tagSize)
	case model.ScalarBytes:
		fmt.Fprintf(b, "n += %d + runtime.SizeBytes(v)\n", tagSize)
	}
	b.WriteString("}\n")
}

// writeMarshalField generates the marshal code for a single field.
func writeMarshalField(b *strings.Builder, recv string, f transform.GoField) {
	accessor := fmt.Sprintf("%s.%s", recv, f.GoName)

	if f.Cardinality == model.CardinalityRepeated {
		writeMarshalRepeated(b, accessor, f)
		return
	}

	// Optional (pointer) field: nil check; deref for value access.
	if f.Optional {
		fmt.Fprintf(b, "if %s != nil {\n", accessor)
		deref := "*" + accessor
		switch f.Type.Kind {
		case model.FieldKindScalar:
			writeMarshalScalar(b, deref, f.Number, f.Type.Scalar)
		case model.FieldKindEnum:
			fmt.Fprintf(b, "b = runtime.AppendTag(b, %d, runtime.WireVarint)\n", f.Number)
			fmt.Fprintf(b, "b = runtime.AppendVarint(b, uint64(%s))\n", deref)
		}
		b.WriteString("}\n")
		return
	}

	// HasPresence bytes field (optional bytes): nil means absent, []byte{} means present-empty.
	if f.HasPresence && f.Type.Kind == model.FieldKindScalar && f.Type.Scalar == model.ScalarBytes {
		fmt.Fprintf(b, "if %s != nil {\n", accessor)
		fmt.Fprintf(b, "b = runtime.AppendTag(b, %d, runtime.WireBytes)\n", f.Number)
		fmt.Fprintf(b, "b = runtime.AppendBytes(b, %s)\n", accessor)
		b.WriteString("}\n")
		return
	}

	// Singular field: skip if zero value.
	zeroCheck := zeroValueCheck(accessor, f)
	fmt.Fprintf(b, "if %s {\n", zeroCheck)

	switch f.Type.Kind {
	case model.FieldKindScalar:
		writeMarshalScalar(b, accessor, f.Number, f.Type.Scalar)
	case model.FieldKindEnum:
		fmt.Fprintf(b, "b = runtime.AppendTag(b, %d, runtime.WireVarint)\n", f.Number)
		fmt.Fprintf(b, "b = runtime.AppendVarint(b, uint64(%s))\n", accessor)
	case model.FieldKindMessage:
		fmt.Fprintf(b, "b = runtime.AppendTag(b, %d, runtime.WireBytes)\n", f.Number)
		fmt.Fprintf(b, "b = runtime.AppendVarint(b, uint64(%s.Size()))\n", accessor)
		fmt.Fprintf(b, "var err error\n")
		fmt.Fprintf(b, "b, err = %s.MarshalAppend(b)\n", accessor)
		b.WriteString("if err != nil { return nil, err }\n")
	}

	b.WriteString("}\n")
}

// writeMarshalRepeated generates marshal code for repeated fields.
func writeMarshalRepeated(b *strings.Builder, accessor string, f transform.GoField) {
	fmt.Fprintf(b, "if len(%s) > 0 {\n", accessor)

	switch f.Type.Kind {
	case model.FieldKindScalar:
		if isPackable(f.Type.Scalar) {
			writeMarshalPackedScalar(b, accessor, f.Number, f.Type.Scalar)
		} else {
			writeMarshalRepeatedLEN(b, accessor, f.Number, f.Type.Scalar)
		}
	case model.FieldKindEnum:
		writeMarshalPackedEnum(b, accessor, f.Number)
	case model.FieldKindMessage:
		fmt.Fprintf(b, "for _, v := range %s {\n", accessor)
		fmt.Fprintf(b, "b = runtime.AppendTag(b, %d, runtime.WireBytes)\n", f.Number)
		b.WriteString("b = runtime.AppendVarint(b, uint64(v.Size()))\n")
		b.WriteString("var err error\n")
		b.WriteString("b, err = v.MarshalAppend(b)\n")
		b.WriteString("if err != nil { return nil, err }\n")
		b.WriteString("}\n")
	}

	b.WriteString("}\n")
}

// writeMarshalScalar generates the append calls for a singular scalar field.
func writeMarshalScalar(b *strings.Builder, accessor string, fieldNumber int, scalar model.ScalarKind) {
	wt := wireType(scalar)
	fmt.Fprintf(b, "b = runtime.AppendTag(b, %d, %s)\n", fieldNumber, wireTypeConst(wt))

	switch scalar {
	case model.ScalarBool:
		fmt.Fprintf(b, "b = runtime.AppendBool(b, %s)\n", accessor)
	case model.ScalarInt32:
		fmt.Fprintf(b, "b = runtime.AppendVarint(b, uint64(%s))\n", accessor)
	case model.ScalarInt64:
		fmt.Fprintf(b, "b = runtime.AppendVarint(b, uint64(%s))\n", accessor)
	case model.ScalarUint32:
		fmt.Fprintf(b, "b = runtime.AppendVarint(b, uint64(%s))\n", accessor)
	case model.ScalarUint64:
		fmt.Fprintf(b, "b = runtime.AppendVarint(b, %s)\n", accessor)
	case model.ScalarSint32:
		fmt.Fprintf(b, "b = runtime.AppendSint32(b, %s)\n", accessor)
	case model.ScalarSint64:
		fmt.Fprintf(b, "b = runtime.AppendSint64(b, %s)\n", accessor)
	case model.ScalarFixed32:
		fmt.Fprintf(b, "b = runtime.AppendFixed32(b, %s)\n", accessor)
	case model.ScalarSfixed32:
		fmt.Fprintf(b, "b = runtime.AppendFixed32(b, uint32(%s))\n", accessor)
	case model.ScalarFloat:
		fmt.Fprintf(b, "b = runtime.AppendFloat(b, %s)\n", accessor)
	case model.ScalarFixed64:
		fmt.Fprintf(b, "b = runtime.AppendFixed64(b, %s)\n", accessor)
	case model.ScalarSfixed64:
		fmt.Fprintf(b, "b = runtime.AppendFixed64(b, uint64(%s))\n", accessor)
	case model.ScalarDouble:
		fmt.Fprintf(b, "b = runtime.AppendDouble(b, %s)\n", accessor)
	case model.ScalarString:
		fmt.Fprintf(b, "b = runtime.AppendString(b, %s)\n", accessor)
	case model.ScalarBytes:
		fmt.Fprintf(b, "b = runtime.AppendBytes(b, %s)\n", accessor)
	}
}

// writeMarshalPackedScalar generates packed encoding for repeated scalar fields.
func writeMarshalPackedScalar(b *strings.Builder, accessor string, fieldNumber int, scalar model.ScalarKind) {
	// Write tag + packed payload length + elements.
	fmt.Fprintf(b, "b = runtime.AppendTag(b, %d, runtime.WireBytes)\n", fieldNumber)

	// Calculate packed payload size inline.
	switch scalar {
	case model.ScalarBool:
		fmt.Fprintf(b, "b = runtime.AppendVarint(b, uint64(len(%s)))\n", accessor)
		fmt.Fprintf(b, "for _, v := range %s {\n", accessor)
		b.WriteString("b = runtime.AppendBool(b, v)\n}\n")
	case model.ScalarFixed32, model.ScalarSfixed32, model.ScalarFloat:
		fmt.Fprintf(b, "b = runtime.AppendVarint(b, uint64(len(%s)*4))\n", accessor)
		fmt.Fprintf(b, "for _, v := range %s {\n", accessor)
		writeMarshalPackedElement(b, scalar)
		b.WriteString("}\n")
	case model.ScalarFixed64, model.ScalarSfixed64, model.ScalarDouble:
		fmt.Fprintf(b, "b = runtime.AppendVarint(b, uint64(len(%s)*8))\n", accessor)
		fmt.Fprintf(b, "for _, v := range %s {\n", accessor)
		writeMarshalPackedElement(b, scalar)
		b.WriteString("}\n")
	default:
		// Variable-length: need to pre-calculate size.
		b.WriteString("var es int\n")
		fmt.Fprintf(b, "for _, v := range %s {\n", accessor)
		writeSizePackedElement(b, scalar)
		b.WriteString("}\n")
		b.WriteString("b = runtime.AppendVarint(b, uint64(es))\n")
		fmt.Fprintf(b, "for _, v := range %s {\n", accessor)
		writeMarshalPackedVarintElement(b, scalar)
		b.WriteString("}\n")
	}
}

func writeMarshalPackedElement(b *strings.Builder, scalar model.ScalarKind) {
	switch scalar {
	case model.ScalarFixed32:
		b.WriteString("b = runtime.AppendFixed32(b, v)\n")
	case model.ScalarSfixed32:
		b.WriteString("b = runtime.AppendFixed32(b, uint32(v))\n")
	case model.ScalarFloat:
		b.WriteString("b = runtime.AppendFloat(b, v)\n")
	case model.ScalarFixed64:
		b.WriteString("b = runtime.AppendFixed64(b, v)\n")
	case model.ScalarSfixed64:
		b.WriteString("b = runtime.AppendFixed64(b, uint64(v))\n")
	case model.ScalarDouble:
		b.WriteString("b = runtime.AppendDouble(b, v)\n")
	}
}

func writeSizePackedElement(b *strings.Builder, scalar model.ScalarKind) {
	switch scalar {
	case model.ScalarInt32:
		b.WriteString("es += runtime.SizeInt32(v)\n")
	case model.ScalarInt64:
		b.WriteString("es += runtime.SizeInt64(v)\n")
	case model.ScalarUint32:
		b.WriteString("es += runtime.SizeUint32(v)\n")
	case model.ScalarUint64:
		b.WriteString("es += runtime.SizeUint64(v)\n")
	case model.ScalarSint32:
		b.WriteString("es += runtime.SizeSint32(v)\n")
	case model.ScalarSint64:
		b.WriteString("es += runtime.SizeSint64(v)\n")
	}
}

func writeMarshalPackedVarintElement(b *strings.Builder, scalar model.ScalarKind) {
	switch scalar {
	case model.ScalarInt32:
		b.WriteString("b = runtime.AppendVarint(b, uint64(v))\n")
	case model.ScalarInt64:
		b.WriteString("b = runtime.AppendVarint(b, uint64(v))\n")
	case model.ScalarUint32:
		b.WriteString("b = runtime.AppendVarint(b, uint64(v))\n")
	case model.ScalarUint64:
		b.WriteString("b = runtime.AppendVarint(b, v)\n")
	case model.ScalarSint32:
		b.WriteString("b = runtime.AppendSint32(b, v)\n")
	case model.ScalarSint64:
		b.WriteString("b = runtime.AppendSint64(b, v)\n")
	}
}

// writeMarshalPackedEnum generates packed encoding for repeated enum fields.
func writeMarshalPackedEnum(b *strings.Builder, accessor string, fieldNumber int) {
	fmt.Fprintf(b, "b = runtime.AppendTag(b, %d, runtime.WireBytes)\n", fieldNumber)
	b.WriteString("var es int\n")
	fmt.Fprintf(b, "for _, v := range %s {\n", accessor)
	b.WriteString("es += runtime.SizeEnum(int32(v))\n}\n")
	b.WriteString("b = runtime.AppendVarint(b, uint64(es))\n")
	fmt.Fprintf(b, "for _, v := range %s {\n", accessor)
	b.WriteString("b = runtime.AppendVarint(b, uint64(v))\n}\n")
}

// writeMarshalRepeatedLEN generates marshal code for repeated string/bytes.
func writeMarshalRepeatedLEN(b *strings.Builder, accessor string, fieldNumber int, scalar model.ScalarKind) {
	fmt.Fprintf(b, "for _, v := range %s {\n", accessor)
	fmt.Fprintf(b, "b = runtime.AppendTag(b, %d, runtime.WireBytes)\n", fieldNumber)
	switch scalar {
	case model.ScalarString:
		b.WriteString("b = runtime.AppendString(b, v)\n")
	case model.ScalarBytes:
		b.WriteString("b = runtime.AppendBytes(b, v)\n")
	}
	b.WriteString("}\n")
}

// zeroValueCheck returns a Go expression that is true when the field is NOT zero.
func zeroValueCheck(accessor string, f transform.GoField) string {
	switch f.Type.Kind {
	case model.FieldKindMessage:
		return accessor + " != nil"
	case model.FieldKindEnum:
		return accessor + " != 0"
	case model.FieldKindScalar:
		switch f.Type.Scalar {
		case model.ScalarBool:
			return accessor
		case model.ScalarString:
			return "len(" + accessor + `) > 0`
		case model.ScalarBytes:
			return "len(" + accessor + `) > 0`
		case model.ScalarFloat:
			return "!runtime.IsZeroFloat(" + accessor + ")"
		case model.ScalarDouble:
			return "!runtime.IsZeroDouble(" + accessor + ")"
		default:
			// Numeric types: != 0
			return accessor + " != 0"
		}
	}
	return "true"
}

// isPackable returns true if the scalar type uses packed encoding for repeated fields.
// string and bytes are not packable (each element is length-delimited individually).
func isPackable(scalar model.ScalarKind) bool {
	return scalar != model.ScalarString && scalar != model.ScalarBytes
}

// wireType returns the protobuf wire type for a scalar kind.
func wireType(scalar model.ScalarKind) int {
	switch scalar {
	case model.ScalarFixed32, model.ScalarSfixed32, model.ScalarFloat:
		return 5 // I32
	case model.ScalarFixed64, model.ScalarSfixed64, model.ScalarDouble:
		return 1 // I64
	case model.ScalarString, model.ScalarBytes:
		return 2 // LEN
	default:
		return 0 // VARINT
	}
}

// wireTypeConst returns the runtime constant name for a wire type.
func wireTypeConst(wt int) string {
	switch wt {
	case 0:
		return "runtime.WireVarint"
	case 1:
		return "runtime.WireFixed64"
	case 2:
		return "runtime.WireBytes"
	case 5:
		return "runtime.WireFixed32"
	default:
		return "runtime.WireVarint"
	}
}

// tagSizeConst returns the byte size of a field tag (field_number << 3 | wire_type).
// Since wire_type is at most 5 (3 bits), the tag value is field_number*8 + wt.
// For field numbers 1-15, tag fits in 1 byte; 16-2047 in 2 bytes, etc.
func tagSizeConst(fieldNumber int) int {
	v := uint64(fieldNumber) << 3
	n := 1
	for v >= 0x80 {
		v >>= 7
		n++
	}
	return n
}
