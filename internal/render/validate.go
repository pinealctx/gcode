package render

import (
	"fmt"
	"go/format"
	"maps"
	"strings"

	"github.com/pinealctx/x/ds"

	"github.com/pinealctx/gcode/internal/model"
	"github.com/pinealctx/gcode/internal/transform"
)

// ValidateFile renders a complete .pb.dao.validate.go source file from a GoFile.
// Every message gets a Validate() error method; messages without constraints get
// an empty method that returns nil, ensuring interface consistency.
// For update/create messages (UpdateSource/CreateSource non-empty), validate rules
// are read directly from the derived message's own fields (which carry validate
// annotations from gen-proto).
// The returned bytes are gofmt-formatted and ready to write to disk.
func ValidateFile(gf transform.GoFile, modulePath string, ctx Context) ([]byte, error) {
	// Build enum GoName → GoEnum map for defined_only lookups.
	enumByGoName := make(map[string]transform.GoEnum, len(gf.Enums))
	for _, e := range gf.Enums {
		enumByGoName[e.GoName] = e
	}

	var body strings.Builder
	for _, msg := range gf.Messages {
		writeValidateMethod(&body, msg, enumByGoName, ctx)
	}
	bodyStr := body.String()

	var b strings.Builder
	writeHeader(&b, gf.Source)
	writePackage(&b, gf.Package)

	needsValidateruntime := strings.Contains(bodyStr, "validateruntime.")
	needsFmt := strings.Contains(bodyStr, "fmt.")
	switch {
	case needsFmt:
		fmt.Fprintf(&b, "import (\n\"fmt\"\n\"%s/validateruntime\"\n)\n\n", modulePath)
	case needsValidateruntime:
		fmt.Fprintf(&b, "import (\n\"%s/validateruntime\"\n)\n\n", modulePath)
	}

	b.WriteString(bodyStr)

	src, err := format.Source([]byte(b.String()))
	if err != nil {
		return nil, fmt.Errorf("format validate source for %q: %w", gf.Source, err)
	}
	return src, nil
}

// mergeEnums combines local file enums with global enum index.
// Local file enums take precedence over global entries.
func mergeEnums(local, global map[string]transform.GoEnum) map[string]transform.GoEnum {
	if global == nil {
		return local
	}
	merged := make(map[string]transform.GoEnum, len(local)+len(global))
	maps.Copy(merged, global)
	maps.Copy(merged, local) // local takes precedence
	return merged
}

// writeValidateMethod writes the Validate() error method for a single GoMessage.
// For update/create messages (UpdateSource/CreateSource non-empty), validate rules are read directly
// from the derived message's own fields (which already carry validate annotations
// from gen-proto). Required fields are validated unconditionally; optional fields
// are skipped when nil.
func writeValidateMethod(b *strings.Builder, msg transform.GoMessage, enumByGoName map[string]transform.GoEnum, ctx Context) {
	recv := receiverName(msg.GoName)
	fmt.Fprintf(b, "func (%s *%s) Validate() error {\n", recv, msg.GoName)

	isDerived := msg.UpdateSource != "" || msg.CreateSource != ""

	if isDerived {
		// Derived message: validate rules are on the derived fields themselves.
		mergedEnums := mergeEnums(enumByGoName, ctx.EnumIndex)
		writeDerivedValidation(b, recv, msg, mergedEnums)
		b.WriteString("return nil\n}\n\n")
		return
	}

	// Standard validate: use the message's own field validate rules.
	effectiveEnums := mergeEnums(enumByGoName, ctx.EnumIndex)
	for _, f := range msg.Fields {
		if f.ValidateOptions == nil && f.Type.Kind != model.FieldKindMessage {
			continue
		}
		writeFieldValidation(b, recv, f, effectiveEnums)
	}
	b.WriteString("return nil\n}\n\n")
}

// writeDerivedValidation generates validate checks for an update/create message
// using the derived message's own field validate rules. These rules are already
// set by gen-proto when generating .create.proto/.update.proto files.
// Fields in the required set (ConditionFields + RequiredFields) are validated
// unconditionally; optional fields are skipped when nil.
func writeDerivedValidation(b *strings.Builder, recv string, msg transform.GoMessage, enumByGoName map[string]transform.GoEnum) {
	// Build required set: condition fields (update) + required fields (create).
	requiredSet := ds.NewSet(msg.ConditionFields...)
	for _, rf := range msg.RequiredFields {
		requiredSet.Add(rf)
	}

	for _, f := range msg.Fields {
		vo := f.ValidateOptions
		if vo == nil && f.Type.Kind != model.FieldKindMessage {
			continue
		}

		fieldExpr := recv + "." + f.GoName
		isRequired := requiredSet.Contains(f.Name)

		switch {
		case isRequired && f.Type.Kind == model.FieldKindMessage:
			writeMessageFieldValidation(b, fieldExpr, f.Name, f.ValidateMessage, vo)

		case isRequired:
			writeFieldValidationExpr(b, fieldExpr, f, enumByGoName, true)

		case strings.HasPrefix(f.GoType, "*"):
			if f.Type.Kind == model.FieldKindMessage {
				// Optional message field (*Message): writeMessageFieldValidation already
				// handles the nil guard and recursive Validate(). Do not dereference.
				writeMessageFieldValidation(b, fieldExpr, f.Name, f.ValidateMessage, vo)
			} else {
				fmt.Fprintf(b, "if %s != nil {\n", fieldExpr)
				writeFieldValidationExpr(b, "*"+fieldExpr, f, enumByGoName, false)
				b.WriteString("}\n")
			}

		default:
			// Non-required, non-pointer field (e.g. repeated, or a create field not in
			// RequiredFields whose GoType has no "*" prefix). Validate directly without
			// a nil guard; string zero-value guard is still applied by writeFieldValidationExpr.
			writeFieldValidationExpr(b, fieldExpr, f, enumByGoName, false)
		}
	}
}

// writeFieldValidationExpr writes validation for a field using a custom expression
// (used for derived validation where the field expression may be dereferenced).
// noZeroGuard=true disables the zero-value skip for string fields (used for
// non-optional condition fields where empty string must be validated).
func writeFieldValidationExpr(b *strings.Builder, fieldExpr string, f transform.GoField, enumByGoName map[string]transform.GoEnum, noZeroGuard bool) {
	fieldName := f.Name
	vm := f.ValidateMessage
	vo := f.ValidateOptions

	switch {
	case f.Cardinality == model.CardinalityRepeated:
		writeRepeatedValidation(b, fieldExpr, fieldName, vm, vo, f, enumByGoName)
	case f.Type.Kind == model.FieldKindMessage:
		writeMessageFieldValidation(b, fieldExpr, fieldName, vm, vo)
	case f.Type.Kind == model.FieldKindEnum:
		writeEnumValidation(b, fieldExpr, fieldName, vm, vo, f, enumByGoName)
	case f.Type.Kind == model.FieldKindScalar:
		writeScalarValidation(b, fieldExpr, fieldName, vm, vo, f, noZeroGuard)
	default:
		panic(fmt.Sprintf("writeFieldValidationExpr: unhandled kind=%v cardinality=%v", f.Type.Kind, f.Cardinality))
	}
}

// writeFieldValidation writes validation checks for a single field.
// For optional scalar/enum fields (GoType starts with "*"), wraps checks in a
// nil guard and dereferences the pointer before validation.
func writeFieldValidation(b *strings.Builder, recv string, f transform.GoField, enumByGoName map[string]transform.GoEnum) {
	fieldExpr := recv + "." + f.GoName
	fieldName := f.Name
	vm := f.ValidateMessage
	vo := f.ValidateOptions

	switch {
	case f.Cardinality == model.CardinalityRepeated:
		writeRepeatedValidation(b, fieldExpr, fieldName, vm, vo, f, enumByGoName)

	case f.Type.Kind == model.FieldKindMessage:
		writeMessageFieldValidation(b, fieldExpr, fieldName, vm, vo)

	case f.Type.Kind == model.FieldKindEnum:
		if strings.HasPrefix(f.GoType, "*") {
			fmt.Fprintf(b, "if %s != nil {\n", fieldExpr)
			writeEnumValidation(b, "*"+fieldExpr, fieldName, vm, vo, f, enumByGoName)
			b.WriteString("}\n")
		} else {
			writeEnumValidation(b, fieldExpr, fieldName, vm, vo, f, enumByGoName)
		}

	case f.Type.Kind == model.FieldKindScalar:
		if strings.HasPrefix(f.GoType, "*") {
			fmt.Fprintf(b, "if %s != nil {\n", fieldExpr)
			writeScalarValidation(b, "*"+fieldExpr, fieldName, vm, vo, f, false)
			b.WriteString("}\n")
		} else {
			writeScalarValidation(b, fieldExpr, fieldName, vm, vo, f, false)
		}
	default:
		panic(fmt.Sprintf("writeFieldValidation: unhandled kind=%v cardinality=%v", f.Type.Kind, f.Cardinality))
	}
}

// writeScalarValidation writes validation for a scalar field.
// noZeroGuard=true disables the zero-value skip for string fields; it has no
// effect on other scalar types (integers, floats, bytes) which have no zero-value guard.
func writeScalarValidation(b *strings.Builder, fieldExpr, fieldName, vm string, vo *model.ValidateFieldOptions, f transform.GoField, noZeroGuard bool) {
	if vo == nil {
		return
	}
	scalar := f.Type.Scalar

	switch scalar {
	case model.ScalarString:
		writeStringValidation(b, fieldExpr, fieldName, vm, vo, noZeroGuard)
	case model.ScalarBytes:
		writeBytesValidation(b, fieldExpr, fieldName, vm, vo)
	case model.ScalarBool:
		// bool required not supported (parser rejects it)
	case model.ScalarFloat, model.ScalarDouble:
		writeFloatValidation(b, fieldExpr, fieldName, vm, vo)
	case model.ScalarInt32, model.ScalarInt64,
		model.ScalarUint32, model.ScalarUint64,
		model.ScalarSint32, model.ScalarSint64,
		model.ScalarFixed32, model.ScalarFixed64,
		model.ScalarSfixed32, model.ScalarSfixed64:
		writeIntValidation(b, fieldExpr, fieldName, vm, vo, scalar)
	default:
		panic(fmt.Sprintf("writeScalarValidation: unhandled scalar kind %v", scalar))
	}
}

// writeStringValidation writes string field constraints.
// noZeroGuard=true disables the "if field != """ zero-value skip (used for
// non-optional condition fields where empty string must be validated).
func writeStringValidation(b *strings.Builder, fieldExpr, fieldName, vm string, vo *model.ValidateFieldOptions, noZeroGuard bool) {
	// required first
	if vo.Required {
		fmt.Fprintf(b, "if %s == \"\" {\nreturn &validateruntime.ValidationError{Field: %q, Rule: \"required\", Message: validateruntime.MsgOr(%q, \"is required\")}\n}\n",
			fieldExpr, fieldName, vm)
	}

	// non-required constraints: skip if empty string (unless noZeroGuard)
	hasNonRequired := vo.MinLen != nil || vo.MaxLen != nil || vo.Pattern != "" || vo.Email || vo.URI || len(vo.InStr) > 0 || len(vo.NotInStr) > 0
	if !hasNonRequired {
		return
	}

	// wrap non-required checks in "if field != """ unless required or noZeroGuard
	useZeroGuard := !vo.Required && !noZeroGuard
	if useZeroGuard {
		fmt.Fprintf(b, "if %s != \"\" {\n", fieldExpr)
	}

	if vo.MinLen != nil {
		fmt.Fprintf(b, "if len(%s) < %d {\nreturn &validateruntime.ValidationError{Field: %q, Rule: \"min_len\", Message: validateruntime.MsgOr(%q, \"length must be >= %d\")}\n}\n",
			fieldExpr, *vo.MinLen, fieldName, vm, *vo.MinLen)
	}
	if vo.MaxLen != nil {
		fmt.Fprintf(b, "if len(%s) > %d {\nreturn &validateruntime.ValidationError{Field: %q, Rule: \"max_len\", Message: validateruntime.MsgOr(%q, \"length must be <= %d\")}\n}\n",
			fieldExpr, *vo.MaxLen, fieldName, vm, *vo.MaxLen)
	}
	if vo.Pattern != "" {
		fmt.Fprintf(b, "if !validateruntime.MatchPattern(%s, %q) {\nreturn &validateruntime.ValidationError{Field: %q, Rule: \"pattern\", Message: validateruntime.MsgOr(%q, %q)}\n}\n",
			fieldExpr, vo.Pattern, fieldName, vm, "must match pattern "+vo.Pattern)
	}
	if vo.Email {
		fmt.Fprintf(b, "if !validateruntime.IsEmail(%s) {\nreturn &validateruntime.ValidationError{Field: %q, Rule: \"email\", Message: validateruntime.MsgOr(%q, \"must be a valid email address\")}\n}\n",
			fieldExpr, fieldName, vm)
	}
	if vo.URI {
		fmt.Fprintf(b, "if !validateruntime.IsURI(%s) {\nreturn &validateruntime.ValidationError{Field: %q, Rule: \"uri\", Message: validateruntime.MsgOr(%q, \"must be a valid URI\")}\n}\n",
			fieldExpr, fieldName, vm)
	}
	if len(vo.InStr) > 0 {
		writeStringInCheck(b, fieldExpr, fieldName, vm, vo.InStr)
	}
	if len(vo.NotInStr) > 0 {
		writeStringNotInCheck(b, fieldExpr, fieldName, vm, vo.NotInStr)
	}

	if useZeroGuard {
		b.WriteString("}\n")
	}
}

// writeStringInCheck writes an "in" membership check for a string field.
func writeStringInCheck(b *strings.Builder, fieldExpr, fieldName, vm string, vals []string) {
	writeStringSetCheck(b, fieldExpr, fieldName, vm, vals, false)
}

// writeStringNotInCheck writes a "not_in" membership check for a string field.
func writeStringNotInCheck(b *strings.Builder, fieldExpr, fieldName, vm string, vals []string) {
	writeStringSetCheck(b, fieldExpr, fieldName, vm, vals, true)
}

// writeStringSetCheck writes an in/not_in membership check for a string field.
// negate=false generates an "in" check; negate=true generates a "not_in" check.
func writeStringSetCheck(b *strings.Builder, fieldExpr, fieldName, vm string, vals []string, negate bool) {
	display := buildStringList(vals)
	if !negate {
		b.WriteString("{\nfound := false\n")
		b.WriteString("for _, v := range []string{")
		for i, s := range vals {
			if i > 0 {
				b.WriteString(", ")
			}
			fmt.Fprintf(b, "%q", s)
		}
		b.WriteString("} {\n")
		fmt.Fprintf(b, "if %s == v {\nfound = true\nbreak\n}\n}\n", fieldExpr)
		fmt.Fprintf(b, "if !found {\nreturn &validateruntime.ValidationError{Field: %q, Rule: \"in\", Message: validateruntime.MsgOr(%q, %q)}\n}\n}\n",
			fieldName, vm, "must be one of "+display)
	} else {
		b.WriteString("for _, v := range []string{")
		for i, s := range vals {
			if i > 0 {
				b.WriteString(", ")
			}
			fmt.Fprintf(b, "%q", s)
		}
		b.WriteString("} {\n")
		fmt.Fprintf(b, "if %s == v {\nreturn &validateruntime.ValidationError{Field: %q, Rule: \"not_in\", Message: validateruntime.MsgOr(%q, %q)}\n}\n}\n",
			fieldExpr, fieldName, vm, "must not be one of "+display)
	}
}

// writeBytesValidation writes bytes field constraints.
func writeBytesValidation(b *strings.Builder, fieldExpr, fieldName, vm string, vo *model.ValidateFieldOptions) {
	if vo.Required {
		fmt.Fprintf(b, "if %s == nil {\nreturn &validateruntime.ValidationError{Field: %q, Rule: \"required\", Message: validateruntime.MsgOr(%q, \"is required\")}\n}\n",
			fieldExpr, fieldName, vm)
	}
	if vo.MinLen != nil || vo.MaxLen != nil {
		if !vo.Required {
			fmt.Fprintf(b, "if %s != nil {\n", fieldExpr)
		}
		if vo.MinLen != nil {
			fmt.Fprintf(b, "if len(%s) < %d {\nreturn &validateruntime.ValidationError{Field: %q, Rule: \"min_len\", Message: validateruntime.MsgOr(%q, \"length must be >= %d\")}\n}\n",
				fieldExpr, *vo.MinLen, fieldName, vm, *vo.MinLen)
		}
		if vo.MaxLen != nil {
			fmt.Fprintf(b, "if len(%s) > %d {\nreturn &validateruntime.ValidationError{Field: %q, Rule: \"max_len\", Message: validateruntime.MsgOr(%q, \"length must be <= %d\")}\n}\n",
				fieldExpr, *vo.MaxLen, fieldName, vm, *vo.MaxLen)
		}
		if !vo.Required {
			b.WriteString("}\n")
		}
	}
}

// writeIntValidation writes signed/unsigned integer field constraints.
func writeIntValidation(b *strings.Builder, fieldExpr, fieldName, vm string, vo *model.ValidateFieldOptions, scalar model.ScalarKind) {
	isSigned := isSignedScalar(scalar)

	if isSigned {
		if vo.GTInt != nil {
			fmt.Fprintf(b, "if %s <= %d {\nreturn &validateruntime.ValidationError{Field: %q, Rule: \"gt\", Message: validateruntime.MsgOr(%q, \"must be > %d\")}\n}\n",
				fieldExpr, *vo.GTInt, fieldName, vm, *vo.GTInt)
		}
		if vo.GTEInt != nil {
			fmt.Fprintf(b, "if %s < %d {\nreturn &validateruntime.ValidationError{Field: %q, Rule: \"gte\", Message: validateruntime.MsgOr(%q, \"must be >= %d\")}\n}\n",
				fieldExpr, *vo.GTEInt, fieldName, vm, *vo.GTEInt)
		}
		if vo.LTInt != nil {
			fmt.Fprintf(b, "if %s >= %d {\nreturn &validateruntime.ValidationError{Field: %q, Rule: \"lt\", Message: validateruntime.MsgOr(%q, \"must be < %d\")}\n}\n",
				fieldExpr, *vo.LTInt, fieldName, vm, *vo.LTInt)
		}
		if vo.LTEInt != nil {
			fmt.Fprintf(b, "if %s > %d {\nreturn &validateruntime.ValidationError{Field: %q, Rule: \"lte\", Message: validateruntime.MsgOr(%q, \"must be <= %d\")}\n}\n",
				fieldExpr, *vo.LTEInt, fieldName, vm, *vo.LTEInt)
		}
		if len(vo.InInt) > 0 {
			writeSignedInCheck(b, fieldExpr, fieldName, vm, vo.InInt, scalar)
		}
		if len(vo.NotInInt) > 0 {
			writeSignedNotInCheck(b, fieldExpr, fieldName, vm, vo.NotInInt, scalar)
		}
	} else {
		if vo.GTUint != nil {
			fmt.Fprintf(b, "if %s <= %d {\nreturn &validateruntime.ValidationError{Field: %q, Rule: \"gt\", Message: validateruntime.MsgOr(%q, \"must be > %d\")}\n}\n",
				fieldExpr, *vo.GTUint, fieldName, vm, *vo.GTUint)
		}
		if vo.GTEUint != nil {
			fmt.Fprintf(b, "if %s < %d {\nreturn &validateruntime.ValidationError{Field: %q, Rule: \"gte\", Message: validateruntime.MsgOr(%q, \"must be >= %d\")}\n}\n",
				fieldExpr, *vo.GTEUint, fieldName, vm, *vo.GTEUint)
		}
		if vo.LTUint != nil {
			fmt.Fprintf(b, "if %s >= %d {\nreturn &validateruntime.ValidationError{Field: %q, Rule: \"lt\", Message: validateruntime.MsgOr(%q, \"must be < %d\")}\n}\n",
				fieldExpr, *vo.LTUint, fieldName, vm, *vo.LTUint)
		}
		if vo.LTEUint != nil {
			fmt.Fprintf(b, "if %s > %d {\nreturn &validateruntime.ValidationError{Field: %q, Rule: \"lte\", Message: validateruntime.MsgOr(%q, \"must be <= %d\")}\n}\n",
				fieldExpr, *vo.LTEUint, fieldName, vm, *vo.LTEUint)
		}
		if len(vo.InUint) > 0 {
			writeUnsignedInCheck(b, fieldExpr, fieldName, vm, vo.InUint, scalar)
		}
		if len(vo.NotInUint) > 0 {
			writeUnsignedNotInCheck(b, fieldExpr, fieldName, vm, vo.NotInUint, scalar)
		}
	}
}

func writeSignedInCheck(b *strings.Builder, fieldExpr, fieldName, vm string, vals []int64, scalar model.ScalarKind) {
	goType := signedGoType(scalar)
	b.WriteString("{\nfound := false\n")
	fmt.Fprintf(b, "for _, v := range []%s{", goType)
	for i, v := range vals {
		if i > 0 {
			b.WriteString(", ")
		}
		fmt.Fprintf(b, "%d", v)
	}
	b.WriteString("} {\n")
	fmt.Fprintf(b, "if %s == v {\nfound = true\nbreak\n}\n}\n", fieldExpr)
	display := buildInt64List(vals)
	fmt.Fprintf(b, "if !found {\nreturn &validateruntime.ValidationError{Field: %q, Rule: \"in\", Message: validateruntime.MsgOr(%q, \"must be one of %s\")}\n}\n}\n",
		fieldName, vm, display)
}

func writeSignedNotInCheck(b *strings.Builder, fieldExpr, fieldName, vm string, vals []int64, scalar model.ScalarKind) {
	goType := signedGoType(scalar)
	fmt.Fprintf(b, "for _, v := range []%s{", goType)
	for i, v := range vals {
		if i > 0 {
			b.WriteString(", ")
		}
		fmt.Fprintf(b, "%d", v)
	}
	b.WriteString("} {\n")
	display := buildInt64List(vals)
	fmt.Fprintf(b, "if %s == v {\nreturn &validateruntime.ValidationError{Field: %q, Rule: \"not_in\", Message: validateruntime.MsgOr(%q, \"must not be one of %s\")}\n}\n}\n",
		fieldExpr, fieldName, vm, display)
}

func writeUnsignedInCheck(b *strings.Builder, fieldExpr, fieldName, vm string, vals []uint64, scalar model.ScalarKind) {
	goType := unsignedGoType(scalar)
	b.WriteString("{\nfound := false\n")
	fmt.Fprintf(b, "for _, v := range []%s{", goType)
	for i, v := range vals {
		if i > 0 {
			b.WriteString(", ")
		}
		fmt.Fprintf(b, "%d", v)
	}
	b.WriteString("} {\n")
	fmt.Fprintf(b, "if %s == v {\nfound = true\nbreak\n}\n}\n", fieldExpr)
	display := buildUint64List(vals)
	fmt.Fprintf(b, "if !found {\nreturn &validateruntime.ValidationError{Field: %q, Rule: \"in\", Message: validateruntime.MsgOr(%q, \"must be one of %s\")}\n}\n}\n",
		fieldName, vm, display)
}

func writeUnsignedNotInCheck(b *strings.Builder, fieldExpr, fieldName, vm string, vals []uint64, scalar model.ScalarKind) {
	goType := unsignedGoType(scalar)
	fmt.Fprintf(b, "for _, v := range []%s{", goType)
	for i, v := range vals {
		if i > 0 {
			b.WriteString(", ")
		}
		fmt.Fprintf(b, "%d", v)
	}
	b.WriteString("} {\n")
	display := buildUint64List(vals)
	fmt.Fprintf(b, "if %s == v {\nreturn &validateruntime.ValidationError{Field: %q, Rule: \"not_in\", Message: validateruntime.MsgOr(%q, \"must not be one of %s\")}\n}\n}\n",
		fieldExpr, fieldName, vm, display)
}

// writeFloatValidation writes float/double field constraints.
func writeFloatValidation(b *strings.Builder, fieldExpr, fieldName, vm string, vo *model.ValidateFieldOptions) {
	if vo.GTFloat != nil {
		fmt.Fprintf(b, "if %s <= %g {\nreturn &validateruntime.ValidationError{Field: %q, Rule: \"gt\", Message: validateruntime.MsgOr(%q, \"must be > %g\")}\n}\n",
			fieldExpr, *vo.GTFloat, fieldName, vm, *vo.GTFloat)
	}
	if vo.GTEFloat != nil {
		fmt.Fprintf(b, "if %s < %g {\nreturn &validateruntime.ValidationError{Field: %q, Rule: \"gte\", Message: validateruntime.MsgOr(%q, \"must be >= %g\")}\n}\n",
			fieldExpr, *vo.GTEFloat, fieldName, vm, *vo.GTEFloat)
	}
	if vo.LTFloat != nil {
		fmt.Fprintf(b, "if %s >= %g {\nreturn &validateruntime.ValidationError{Field: %q, Rule: \"lt\", Message: validateruntime.MsgOr(%q, \"must be < %g\")}\n}\n",
			fieldExpr, *vo.LTFloat, fieldName, vm, *vo.LTFloat)
	}
	if vo.LTEFloat != nil {
		fmt.Fprintf(b, "if %s > %g {\nreturn &validateruntime.ValidationError{Field: %q, Rule: \"lte\", Message: validateruntime.MsgOr(%q, \"must be <= %g\")}\n}\n",
			fieldExpr, *vo.LTEFloat, fieldName, vm, *vo.LTEFloat)
	}
}

// writeEnumValidation writes enum field constraints.
func writeEnumValidation(b *strings.Builder, fieldExpr, fieldName, vm string, vo *model.ValidateFieldOptions, f transform.GoField, enumByGoName map[string]transform.GoEnum) {
	if vo == nil {
		return
	}

	if len(vo.NotInEnum) > 0 {
		writeEnumNotInCheck(b, fieldExpr, fieldName, vm, vo.NotInEnum)
	}

	if vo.DefinedOnly {
		enum, ok := lookupEnum(f, enumByGoName)
		if !ok {
			return
		}
		b.WriteString("switch " + fieldExpr + " {\n")
		b.WriteString("case ")
		for i, v := range enum.Values {
			if i > 0 {
				b.WriteString(", ")
			}
			b.WriteString(v.GoName)
		}
		b.WriteString(":\n// ok\n")
		fmt.Fprintf(b, "default:\nreturn &validateruntime.ValidationError{Field: %q, Rule: \"defined_only\", Message: validateruntime.MsgOr(%q, \"must be a defined enum value\")}\n}\n",
			fieldName, vm)
	}
}

func writeEnumNotInCheck(b *strings.Builder, fieldExpr, fieldName, vm string, vals []int32) {
	fmt.Fprintf(b, "for _, v := range []int32{")
	for i, v := range vals {
		if i > 0 {
			b.WriteString(", ")
		}
		fmt.Fprintf(b, "%d", v)
	}
	b.WriteString("} {\n")
	display := buildInt32List(vals)
	fmt.Fprintf(b, "if int32(%s) == v {\nreturn &validateruntime.ValidationError{Field: %q, Rule: \"not_in\", Message: validateruntime.MsgOr(%q, \"must not be one of %s\")}\n}\n}\n",
		fieldExpr, fieldName, vm, display)
}

// writeMessageFieldValidation writes message field required + recursive Validate().
func writeMessageFieldValidation(b *strings.Builder, fieldExpr, fieldName, vm string, vo *model.ValidateFieldOptions) {
	if vo != nil && vo.Required {
		fmt.Fprintf(b, "if %s == nil {\nreturn &validateruntime.ValidationError{Field: %q, Rule: \"required\", Message: validateruntime.MsgOr(%q, \"is required\")}\n}\n",
			fieldExpr, fieldName, vm)
		// Field is guaranteed non-nil here; recurse unconditionally.
		fmt.Fprintf(b, "if err := %s.Validate(); err != nil {\nreturn err\n}\n", fieldExpr)
		return
	}
	// No required constraint: recurse only if non-nil.
	fmt.Fprintf(b, "if %s != nil {\nif err := %s.Validate(); err != nil {\nreturn err\n}\n}\n",
		fieldExpr, fieldExpr)
}

// writeRepeatedValidation writes repeated field constraints.
func writeRepeatedValidation(b *strings.Builder, fieldExpr, fieldName, vm string, vo *model.ValidateFieldOptions, f transform.GoField, enumByGoName map[string]transform.GoEnum) {
	if vo == nil {
		// No validate options, but repeated message fields still need element-level Validate() calls.
		if f.Type.Kind == model.FieldKindMessage {
			fmt.Fprintf(b, "for _, v := range %s {\nif err := v.Validate(); err != nil {\nreturn err\n}\n}\n", fieldExpr)
		}
		return
	}
	if vo.MinItems != nil {
		fmt.Fprintf(b, "if len(%s) < %d {\nreturn &validateruntime.ValidationError{Field: %q, Rule: \"min_items\", Message: validateruntime.MsgOr(%q, \"must have at least %d item(s)\")}\n}\n",
			fieldExpr, *vo.MinItems, fieldName, vm, *vo.MinItems)
	}
	if vo.MaxItems != nil {
		fmt.Fprintf(b, "if len(%s) > %d {\nreturn &validateruntime.ValidationError{Field: %q, Rule: \"max_items\", Message: validateruntime.MsgOr(%q, \"must have at most %d item(s)\")}\n}\n",
			fieldExpr, *vo.MaxItems, fieldName, vm, *vo.MaxItems)
	}
	if vo.Items != nil {
		writeItemsValidation(b, fieldExpr, fieldName, vm, vo.Items, f, enumByGoName)
	}
}

// writeItemsValidation writes element-level validation for repeated fields.
// It dispatches to type-specific sub-writers based on the element scalar kind.
func writeItemsValidation(b *strings.Builder, fieldExpr, fieldName, vm string, items *model.ValidateFieldOptions, f transform.GoField, enumByGoName map[string]transform.GoEnum) {
	fmt.Fprintf(b, "for i, v := range %s {\n", fieldExpr)
	elemField := fmt.Sprintf("fmt.Sprintf(\"%s[%%d]\", i)", fieldName)

	writeItemsStringValidation(b, elemField, vm, items)
	writeItemsSignedIntValidation(b, elemField, vm, items, f)
	writeItemsUnsignedIntValidation(b, elemField, vm, items, f)
	writeItemsFloatValidation(b, elemField, vm, items)
	writeItemsEnumValidation(b, elemField, vm, items, f, enumByGoName)

	b.WriteString("}\n")
}

// writeItemsStringValidation writes string element constraints (min_len, max_len, pattern, email, uri, in, not_in).
func writeItemsStringValidation(b *strings.Builder, elemField, vm string, items *model.ValidateFieldOptions) {
	if items.MinLen != nil {
		fmt.Fprintf(b, "if len(v) < %d {\nreturn &validateruntime.ValidationError{Field: %s, Rule: \"min_len\", Message: validateruntime.MsgOr(%q, \"length must be >= %d\")}\n}\n",
			*items.MinLen, elemField, vm, *items.MinLen)
	}
	if items.MaxLen != nil {
		fmt.Fprintf(b, "if len(v) > %d {\nreturn &validateruntime.ValidationError{Field: %s, Rule: \"max_len\", Message: validateruntime.MsgOr(%q, \"length must be <= %d\")}\n}\n",
			*items.MaxLen, elemField, vm, *items.MaxLen)
	}
	if items.Pattern != "" {
		fmt.Fprintf(b, "if !validateruntime.MatchPattern(v, %q) {\nreturn &validateruntime.ValidationError{Field: %s, Rule: \"pattern\", Message: validateruntime.MsgOr(%q, %q)}\n}\n",
			items.Pattern, elemField, vm, "must match pattern "+items.Pattern)
	}
	if items.Email {
		fmt.Fprintf(b, "if !validateruntime.IsEmail(v) {\nreturn &validateruntime.ValidationError{Field: %s, Rule: \"email\", Message: validateruntime.MsgOr(%q, \"must be a valid email address\")}\n}\n",
			elemField, vm)
	}
	if items.URI {
		fmt.Fprintf(b, "if !validateruntime.IsURI(v) {\nreturn &validateruntime.ValidationError{Field: %s, Rule: \"uri\", Message: validateruntime.MsgOr(%q, \"must be a valid URI\")}\n}\n",
			elemField, vm)
	}
	if len(items.InStr) > 0 {
		display := buildStringList(items.InStr)
		b.WriteString("{\nfound := false\n")
		b.WriteString("for _, sv := range []string{")
		for i, s := range items.InStr {
			if i > 0 {
				b.WriteString(", ")
			}
			fmt.Fprintf(b, "%q", s)
		}
		b.WriteString("} {\n")
		fmt.Fprintf(b, "if v == sv {\nfound = true\nbreak\n}\n}\n")
		fmt.Fprintf(b, "if !found {\nreturn &validateruntime.ValidationError{Field: %s, Rule: \"in\", Message: validateruntime.MsgOr(%q, %q)}\n}\n}\n",
			elemField, vm, "must be one of "+display)
	}
	if len(items.NotInStr) > 0 {
		display := buildStringList(items.NotInStr)
		b.WriteString("for _, sv := range []string{")
		for i, s := range items.NotInStr {
			if i > 0 {
				b.WriteString(", ")
			}
			fmt.Fprintf(b, "%q", s)
		}
		b.WriteString("} {\n")
		fmt.Fprintf(b, "if v == sv {\nreturn &validateruntime.ValidationError{Field: %s, Rule: \"not_in\", Message: validateruntime.MsgOr(%q, %q)}\n}\n}\n",
			elemField, vm, "must not be one of "+display)
	}
}

// writeItemsSignedIntValidation writes signed integer element constraints (gt, gte, lt, lte, in, not_in).
func writeItemsSignedIntValidation(b *strings.Builder, elemField, vm string, items *model.ValidateFieldOptions, f transform.GoField) {
	if items.GTInt != nil {
		fmt.Fprintf(b, "if v <= %d {\nreturn &validateruntime.ValidationError{Field: %s, Rule: \"gt\", Message: validateruntime.MsgOr(%q, \"must be > %d\")}\n}\n",
			*items.GTInt, elemField, vm, *items.GTInt)
	}
	if items.GTEInt != nil {
		fmt.Fprintf(b, "if v < %d {\nreturn &validateruntime.ValidationError{Field: %s, Rule: \"gte\", Message: validateruntime.MsgOr(%q, \"must be >= %d\")}\n}\n",
			*items.GTEInt, elemField, vm, *items.GTEInt)
	}
	if items.LTInt != nil {
		fmt.Fprintf(b, "if v >= %d {\nreturn &validateruntime.ValidationError{Field: %s, Rule: \"lt\", Message: validateruntime.MsgOr(%q, \"must be < %d\")}\n}\n",
			*items.LTInt, elemField, vm, *items.LTInt)
	}
	if items.LTEInt != nil {
		fmt.Fprintf(b, "if v > %d {\nreturn &validateruntime.ValidationError{Field: %s, Rule: \"lte\", Message: validateruntime.MsgOr(%q, \"must be <= %d\")}\n}\n",
			*items.LTEInt, elemField, vm, *items.LTEInt)
	}
	if len(items.InInt) > 0 {
		goType := signedGoType(f.Type.Scalar)
		display := buildInt64List(items.InInt)
		b.WriteString("{\nfound := false\n")
		fmt.Fprintf(b, "for _, iv := range []%s{", goType)
		for i, n := range items.InInt {
			if i > 0 {
				b.WriteString(", ")
			}
			fmt.Fprintf(b, "%d", n)
		}
		b.WriteString("} {\n")
		b.WriteString("if v == iv {\nfound = true\nbreak\n}\n}\n")
		fmt.Fprintf(b, "if !found {\nreturn &validateruntime.ValidationError{Field: %s, Rule: \"in\", Message: validateruntime.MsgOr(%q, \"must be one of %s\")}\n}\n}\n",
			elemField, vm, display)
	}
	if len(items.NotInInt) > 0 {
		goType := signedGoType(f.Type.Scalar)
		display := buildInt64List(items.NotInInt)
		fmt.Fprintf(b, "for _, iv := range []%s{", goType)
		for i, n := range items.NotInInt {
			if i > 0 {
				b.WriteString(", ")
			}
			fmt.Fprintf(b, "%d", n)
		}
		b.WriteString("} {\n")
		fmt.Fprintf(b, "if v == iv {\nreturn &validateruntime.ValidationError{Field: %s, Rule: \"not_in\", Message: validateruntime.MsgOr(%q, \"must not be one of %s\")}\n}\n}\n",
			elemField, vm, display)
	}
}

// writeItemsUnsignedIntValidation writes unsigned integer element constraints (gt, gte, lt, lte, in, not_in).
func writeItemsUnsignedIntValidation(b *strings.Builder, elemField, vm string, items *model.ValidateFieldOptions, f transform.GoField) {
	if items.GTUint != nil {
		fmt.Fprintf(b, "if v <= %d {\nreturn &validateruntime.ValidationError{Field: %s, Rule: \"gt\", Message: validateruntime.MsgOr(%q, \"must be > %d\")}\n}\n",
			*items.GTUint, elemField, vm, *items.GTUint)
	}
	if items.GTEUint != nil {
		fmt.Fprintf(b, "if v < %d {\nreturn &validateruntime.ValidationError{Field: %s, Rule: \"gte\", Message: validateruntime.MsgOr(%q, \"must be >= %d\")}\n}\n",
			*items.GTEUint, elemField, vm, *items.GTEUint)
	}
	if items.LTUint != nil {
		fmt.Fprintf(b, "if v >= %d {\nreturn &validateruntime.ValidationError{Field: %s, Rule: \"lt\", Message: validateruntime.MsgOr(%q, \"must be < %d\")}\n}\n",
			*items.LTUint, elemField, vm, *items.LTUint)
	}
	if items.LTEUint != nil {
		fmt.Fprintf(b, "if v > %d {\nreturn &validateruntime.ValidationError{Field: %s, Rule: \"lte\", Message: validateruntime.MsgOr(%q, \"must be <= %d\")}\n}\n",
			*items.LTEUint, elemField, vm, *items.LTEUint)
	}
	if len(items.InUint) > 0 {
		goType := unsignedGoType(f.Type.Scalar)
		display := buildUint64List(items.InUint)
		b.WriteString("{\nfound := false\n")
		fmt.Fprintf(b, "for _, uv := range []%s{", goType)
		for i, n := range items.InUint {
			if i > 0 {
				b.WriteString(", ")
			}
			fmt.Fprintf(b, "%d", n)
		}
		b.WriteString("} {\n")
		b.WriteString("if v == uv {\nfound = true\nbreak\n}\n}\n")
		fmt.Fprintf(b, "if !found {\nreturn &validateruntime.ValidationError{Field: %s, Rule: \"in\", Message: validateruntime.MsgOr(%q, \"must be one of %s\")}\n}\n}\n",
			elemField, vm, display)
	}
	if len(items.NotInUint) > 0 {
		goType := unsignedGoType(f.Type.Scalar)
		display := buildUint64List(items.NotInUint)
		fmt.Fprintf(b, "for _, uv := range []%s{", goType)
		for i, n := range items.NotInUint {
			if i > 0 {
				b.WriteString(", ")
			}
			fmt.Fprintf(b, "%d", n)
		}
		b.WriteString("} {\n")
		fmt.Fprintf(b, "if v == uv {\nreturn &validateruntime.ValidationError{Field: %s, Rule: \"not_in\", Message: validateruntime.MsgOr(%q, \"must not be one of %s\")}\n}\n}\n",
			elemField, vm, display)
	}
}

// writeItemsFloatValidation writes float/double element constraints (gt, gte, lt, lte).
func writeItemsFloatValidation(b *strings.Builder, elemField, vm string, items *model.ValidateFieldOptions) {
	if items.GTFloat != nil {
		fmt.Fprintf(b, "if v <= %g {\nreturn &validateruntime.ValidationError{Field: %s, Rule: \"gt\", Message: validateruntime.MsgOr(%q, \"must be > %g\")}\n}\n",
			*items.GTFloat, elemField, vm, *items.GTFloat)
	}
	if items.GTEFloat != nil {
		fmt.Fprintf(b, "if v < %g {\nreturn &validateruntime.ValidationError{Field: %s, Rule: \"gte\", Message: validateruntime.MsgOr(%q, \"must be >= %g\")}\n}\n",
			*items.GTEFloat, elemField, vm, *items.GTEFloat)
	}
	if items.LTFloat != nil {
		fmt.Fprintf(b, "if v >= %g {\nreturn &validateruntime.ValidationError{Field: %s, Rule: \"lt\", Message: validateruntime.MsgOr(%q, \"must be < %g\")}\n}\n",
			*items.LTFloat, elemField, vm, *items.LTFloat)
	}
	if items.LTEFloat != nil {
		fmt.Fprintf(b, "if v > %g {\nreturn &validateruntime.ValidationError{Field: %s, Rule: \"lte\", Message: validateruntime.MsgOr(%q, \"must be <= %g\")}\n}\n",
			*items.LTEFloat, elemField, vm, *items.LTEFloat)
	}
}

// writeItemsEnumValidation writes enum element constraints.
func writeItemsEnumValidation(b *strings.Builder, elemField, vm string, items *model.ValidateFieldOptions, f transform.GoField, enumByGoName map[string]transform.GoEnum) {
	if len(items.NotInEnum) > 0 {
		display := buildInt32List(items.NotInEnum)
		b.WriteString("for _, ev := range []int32{")
		for i, n := range items.NotInEnum {
			if i > 0 {
				b.WriteString(", ")
			}
			fmt.Fprintf(b, "%d", n)
		}
		b.WriteString("} {\n")
		fmt.Fprintf(b, "if int32(v) == ev {\nreturn &validateruntime.ValidationError{Field: %s, Rule: \"not_in\", Message: validateruntime.MsgOr(%q, \"must not be one of %s\")}\n}\n}\n",
			elemField, vm, display)
	}

	if !items.DefinedOnly {
		return
	}
	enum, ok := lookupEnum(f, enumByGoName)
	if !ok {
		return
	}
	b.WriteString("switch v {\n case ")
	for i, v := range enum.Values {
		if i > 0 {
			b.WriteString(", ")
		}
		b.WriteString(v.GoName)
	}
	b.WriteString(":\n// ok\n")
	fmt.Fprintf(b, "default:\nreturn &validateruntime.ValidationError{Field: %s, Rule: \"defined_only\", Message: validateruntime.MsgOr(%q, \"must be a defined enum value\")}\n}\n",
		elemField, vm)
}

func lookupEnum(f transform.GoField, enumByGoName map[string]transform.GoEnum) (transform.GoEnum, bool) {
	// Look up enum by stripping pointer/slice prefix from GoType.
	enumGoName := stripTypePrefix(f.GoType)
	enum, ok := enumByGoName[enumGoName]
	if !ok || len(enum.Values) == 0 {
		return transform.GoEnum{}, false
	}
	return enum, true
}

// --- helpers ---

// stripTypePrefix removes leading "*" and "[]" from a Go type string.
func stripTypePrefix(goType string) string {
	s := goType
	for strings.HasPrefix(s, "*") || strings.HasPrefix(s, "[]") {
		if strings.HasPrefix(s, "*") {
			s = s[1:]
		} else {
			s = s[2:]
		}
	}
	return s
}

func isSignedScalar(s model.ScalarKind) bool {
	switch s {
	case model.ScalarInt32, model.ScalarInt64,
		model.ScalarSint32, model.ScalarSint64,
		model.ScalarSfixed32, model.ScalarSfixed64:
		return true
	}
	return false
}

func signedGoType(s model.ScalarKind) string {
	switch s {
	case model.ScalarInt64, model.ScalarSint64, model.ScalarSfixed64:
		return "int64"
	default:
		return "int32"
	}
}

func unsignedGoType(s model.ScalarKind) string {
	switch s {
	case model.ScalarUint64, model.ScalarFixed64:
		return "uint64"
	default:
		return "uint32"
	}
}

func buildStringList(vals []string) string {
	return "[" + strings.Join(vals, ", ") + "]"
}

func buildInt64List(vals []int64) string {
	parts := make([]string, len(vals))
	for i, v := range vals {
		parts[i] = fmt.Sprintf("%d", v)
	}
	return "[" + strings.Join(parts, ", ") + "]"
}

func buildInt32List(vals []int32) string {
	parts := make([]string, len(vals))
	for i, v := range vals {
		parts[i] = fmt.Sprintf("%d", v)
	}
	return "[" + strings.Join(parts, ", ") + "]"
}

func buildUint64List(vals []uint64) string {
	parts := make([]string, len(vals))
	for i, v := range vals {
		parts[i] = fmt.Sprintf("%d", v)
	}
	return "[" + strings.Join(parts, ", ") + "]"
}
