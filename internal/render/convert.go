package render

import (
	"fmt"
	"strings"

	"github.com/pinealctx/x/ds"

	"github.com/pinealctx/gcode/internal/model"
	"github.com/pinealctx/gcode/internal/transform"
)

// writeToEntityMethod generates the ToEntity() method for create-derived messages.
// It converts the create message to the source entity type, handling type
// differences between the derived and source fields (pointer vs non-pointer).
func writeToEntityMethod(b *strings.Builder, msg transform.GoMessage, srcMsg *transform.GoMessage) {
	recv := receiverName(msg.GoName)

	srcFieldByName := make(map[string]transform.GoField, len(srcMsg.Fields))
	for _, f := range srcMsg.Fields {
		srcFieldByName[f.Name] = f
	}

	fmt.Fprintf(b, "// ToEntity converts %s to %s.\n", msg.GoName, srcMsg.GoName)
	fmt.Fprintf(b, "func (%s *%s) ToEntity() *%s {\n", recv, msg.GoName, srcMsg.GoName)
	fmt.Fprintf(b, "var p %s\n", srcMsg.GoName)

	for _, f := range msg.Fields {
		src, ok := srcFieldByName[f.Name]
		if !ok {
			continue
		}
		srcExpr := recv + "." + f.GoName
		dstField := "p." + src.GoName

		fromPtr := strings.HasPrefix(f.GoType, "*")
		toPtr := strings.HasPrefix(src.GoType, "*")

		switch {
		case f.Cardinality == model.CardinalityRepeated || f.GoType == "[]byte":
			// Repeated/bytes: direct assign.
			fmt.Fprintf(b, "%s = %s\n", dstField, srcExpr)
		case fromPtr && !toPtr:
			// Optional pointer to non-pointer: nil-guard + deref.
			fmt.Fprintf(b, "if %s != nil {\n%s = *%s\n}\n", srcExpr, dstField, srcExpr)
		case fromPtr && toPtr:
			// Pointer to pointer: nil-guard + pointer assign.
			fmt.Fprintf(b, "if %s != nil {\n%s = %s\n}\n", srcExpr, dstField, srcExpr)
		case !fromPtr && toPtr:
			// Non-pointer (required) to pointer: copy value then take address,
			// so the entity field does not share memory with the create message.
			fmt.Fprintf(b, "tmp%s := %s\n%s = &tmp%s\n", src.GoName, srcExpr, dstField, src.GoName)
		default:
			// Non-pointer to non-pointer: direct assign.
			fmt.Fprintf(b, "%s = %s\n", dstField, srcExpr)
		}
	}

	b.WriteString("return &p\n}\n\n")
}

// writeApplyToMethod generates the ApplyTo() method for update-derived messages.
// It merges non-nil fields from the update message into the source entity.
// Condition fields (WHERE conditions) are skipped.
func writeApplyToMethod(b *strings.Builder, msg transform.GoMessage, srcMsg *transform.GoMessage) {
	recv := receiverName(msg.GoName)

	srcFieldByName := make(map[string]transform.GoField, len(srcMsg.Fields))
	for _, f := range srcMsg.Fields {
		srcFieldByName[f.Name] = f
	}

	conditionFields := ds.NewSet(msg.ConditionFields...)

	fmt.Fprintf(b, "// ApplyTo merges non-nil fields from %s into p.\n", msg.GoName)
	b.WriteString("// Condition fields are skipped.\n")
	fmt.Fprintf(b, "func (%s *%s) ApplyTo(p *%s) {\n", recv, msg.GoName, srcMsg.GoName)

	for _, f := range msg.Fields {
		if conditionFields.Contains(f.Name) {
			continue
		}
		src, ok := srcFieldByName[f.Name]
		if !ok {
			continue
		}
		srcExpr := recv + "." + f.GoName
		dstField := "p." + src.GoName

		fromPtr := strings.HasPrefix(f.GoType, "*")
		toPtr := strings.HasPrefix(src.GoType, "*")

		switch {
		case f.Cardinality == model.CardinalityRepeated || f.GoType == "[]byte":
			// Repeated/bytes: nil-guard to distinguish "not provided" from empty.
			fmt.Fprintf(b, "if %s != nil {\n%s = %s\n}\n", srcExpr, dstField, srcExpr)
		case fromPtr && !toPtr:
			// Optional pointer to non-pointer: nil-guard + deref.
			fmt.Fprintf(b, "if %s != nil {\n%s = *%s\n}\n", srcExpr, dstField, srcExpr)
		case fromPtr && toPtr:
			// Pointer to pointer: nil-guard + pointer assign.
			fmt.Fprintf(b, "if %s != nil {\n%s = %s\n}\n", srcExpr, dstField, srcExpr)
		case !fromPtr && toPtr:
			// Non-pointer to pointer: copy value then take address,
			// so the entity field does not share memory with the update message.
			fmt.Fprintf(b, "tmp%s := %s\n%s = &tmp%s\n", src.GoName, srcExpr, dstField, src.GoName)
		default:
			// Non-pointer to non-pointer: direct assign.
			fmt.Fprintf(b, "%s = %s\n", dstField, srcExpr)
		}
	}

	b.WriteString("}\n\n")
}
