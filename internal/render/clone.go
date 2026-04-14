package render

import (
	"fmt"
	"strings"

	"github.com/pinealctx/gcode/internal/model"
	"github.com/pinealctx/gcode/internal/transform"
)

// writeDeepCloneMethod generates a DeepClone() method for a single message struct.
// The method returns a deep copy with no shared memory between the clone and the original.
func writeDeepCloneMethod(b *strings.Builder, msg transform.GoMessage) {
	recv := receiverName(msg.GoName)
	fmt.Fprintf(b, "// DeepClone returns a deep copy of %s with no shared memory.\n", msg.GoName)
	fmt.Fprintf(b, "func (%s *%s) DeepClone() *%s {\n", recv, msg.GoName, msg.GoName)
	fmt.Fprintf(b, "if %s == nil { return nil }\n", recv)
	// Shallow copy covers all scalar/enum fields.
	fmt.Fprintf(b, "clone := *%s\n", recv)

	for _, f := range msg.Fields {
		writeDeepCloneField(b, recv, f)
	}

	b.WriteString("return &clone\n}\n\n")
}

// writeDeepCloneField emits the deep-copy override for a single field.
// Scalar and non-pointer enum fields are already correct after the shallow copy.
func writeDeepCloneField(b *strings.Builder, recv string, f transform.GoField) {
	accessor := fmt.Sprintf("%s.%s", recv, f.GoName)

	if f.Cardinality == model.CardinalityRepeated {
		writeDeepCloneRepeated(b, accessor, f)
		return
	}

	// HasPresence bytes (optional []byte, nil = absent): deep copy the slice.
	if f.HasPresence && f.Type.Kind == model.FieldKindScalar && f.Type.Scalar == model.ScalarBytes {
		fmt.Fprintf(b, "if %s != nil {\n", accessor)
		fmt.Fprintf(b, "clone.%s = make([]byte, len(%s))\n", f.GoName, accessor)
		fmt.Fprintf(b, "copy(clone.%s, %s)\n", f.GoName, accessor)
		b.WriteString("}\n")
		return
	}

	// Optional pointer fields: allocate a new pointer.
	// Note: Optional only applies to scalar and enum fields; message fields are
	// always pointers and handled by the FieldKindMessage branch below.
	if f.Optional {
		fmt.Fprintf(b, "if %s != nil {\n", accessor)
		// Both scalar and enum optional fields use the same new-pointer pattern.
		switch f.Type.Kind {
		case model.FieldKindScalar, model.FieldKindEnum:
			fmt.Fprintf(b, "v := *%s\n", accessor)
			fmt.Fprintf(b, "clone.%s = &v\n", f.GoName)
		default:
			panic(fmt.Sprintf("writeDeepCloneField: unexpected optional field kind %v for field %q", f.Type.Kind, f.GoName))
		}
		b.WriteString("}\n")
		return
	}

	// Singular message field: recursive DeepClone.
	if f.Type.Kind == model.FieldKindMessage {
		fmt.Fprintf(b, "if %s != nil {\n", accessor)
		fmt.Fprintf(b, "clone.%s = %s.DeepClone()\n", f.GoName, accessor)
		b.WriteString("}\n")
		return
	}

	// Singular scalar (non-pointer, non-bytes-presence) and singular enum:
	// already handled by the shallow copy — no extra code needed.
	// Exception: singular bytes ([]byte) shares the backing array and must be copied.
	if f.Type.Kind == model.FieldKindScalar && f.Type.Scalar == model.ScalarBytes {
		fmt.Fprintf(b, "if %s != nil {\n", accessor)
		fmt.Fprintf(b, "clone.%s = make([]byte, len(%s))\n", f.GoName, accessor)
		fmt.Fprintf(b, "copy(clone.%s, %s)\n", f.GoName, accessor)
		b.WriteString("}\n")
	}
}

// writeDeepCloneRepeated emits deep-copy code for repeated fields.
func writeDeepCloneRepeated(b *strings.Builder, accessor string, f transform.GoField) {
	fmt.Fprintf(b, "if %s != nil {\n", accessor)

	switch f.Type.Kind {
	case model.FieldKindScalar:
		if f.Type.Scalar == model.ScalarBytes {
			// repeated bytes: each element is a []byte that needs its own copy.
			fmt.Fprintf(b, "clone.%s = make([][]byte, len(%s))\n", f.GoName, accessor)
			fmt.Fprintf(b, "for i, v := range %s {\n", accessor)
			b.WriteString("if v != nil {\n")
			b.WriteString("tmp := make([]byte, len(v))\n")
			b.WriteString("copy(tmp, v)\n")
			fmt.Fprintf(b, "clone.%s[i] = tmp\n", f.GoName)
			b.WriteString("}\n")
			b.WriteString("}\n")
		} else {
			// repeated scalar (string, numeric): make + copy.
			fmt.Fprintf(b, "clone.%s = make([]%s, len(%s))\n", f.GoName, f.ElemGoType, accessor)
			fmt.Fprintf(b, "copy(clone.%s, %s)\n", f.GoName, accessor)
		}
	case model.FieldKindEnum:
		// repeated enum: make + copy (enum is an int32 alias, value type).
		fmt.Fprintf(b, "clone.%s = make([]%s, len(%s))\n", f.GoName, f.ElemGoType, accessor)
		fmt.Fprintf(b, "copy(clone.%s, %s)\n", f.GoName, accessor)
	case model.FieldKindMessage:
		// repeated message: recurse into each element.
		fmt.Fprintf(b, "clone.%s = make([]*%s, len(%s))\n", f.GoName, f.ElemGoType, accessor)
		fmt.Fprintf(b, "for i, v := range %s {\n", accessor)
		fmt.Fprintf(b, "clone.%s[i] = v.DeepClone()\n", f.GoName)
		b.WriteString("}\n")
	default:
		panic(fmt.Sprintf("writeDeepCloneRepeated: unexpected FieldKind %v for field %q", f.Type.Kind, f.GoName))
	}

	b.WriteString("}\n")
}
