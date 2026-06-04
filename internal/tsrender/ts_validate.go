package tsrender

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/pinealctx/gcode/internal/model"
	"github.com/pinealctx/gcode/internal/transform"
)

type tsConstraintValueMode int

const (
	tsConstraintValueNumber tsConstraintValueMode = iota
	tsConstraintValueIntegerString
)

// tsScalarValidationType maps a protobuf scalar kind to its TS validation "type" value.
func tsScalarValidationType(scalar model.ScalarKind) string {
	switch scalar {
	case model.ScalarString, model.ScalarBytes:
		return "string"
	case model.ScalarInt32, model.ScalarSint32, model.ScalarSfixed32,
		model.ScalarUint32, model.ScalarFixed32,
		model.ScalarInt64, model.ScalarSint64, model.ScalarSfixed64,
		model.ScalarUint64, model.ScalarFixed64:
		return "integer"
	case model.ScalarFloat, model.ScalarDouble:
		return "number"
	case model.ScalarBool:
		return "boolean"
	default:
		return "unknown"
	}
}

// tsValidationType returns the TS metadata "type" value for a GoField.
// This differs from tsScalarType/tsFieldType which produce interface types.
func tsValidationType(f transform.GoField) string {
	if f.Cardinality == model.CardinalityRepeated {
		return "array"
	}
	switch f.Type.Kind {
	case model.FieldKindScalar:
		if isIntegerStringField(f) {
			return "integerString"
		}
		return tsScalarValidationType(f.Type.Scalar)
	case model.FieldKindEnum:
		return "enum"
	case model.FieldKindMessage:
		return "object"
	default:
		return "unknown"
	}
}

func isIntegerStringField(f transform.GoField) bool {
	return f.Type.Kind == model.FieldKindScalar &&
		f.JSONOptions != nil &&
		f.JSONOptions.IntegerFormat == model.IntegerFormatString
}

func tsIntegerFormat(scalar model.ScalarKind) string {
	switch scalar {
	case model.ScalarInt64:
		return "int64"
	case model.ScalarUint64:
		return "uint64"
	case model.ScalarSint64:
		return "sint64"
	case model.ScalarFixed64:
		return "fixed64"
	case model.ScalarSfixed64:
		return "sfixed64"
	default:
		panic(fmt.Sprintf("tsIntegerFormat: unsupported integer_format scalar %q", scalar))
	}
}

func tsConstraintMode(f transform.GoField) tsConstraintValueMode {
	if isIntegerStringField(f) {
		return tsConstraintValueIntegerString
	}
	return tsConstraintValueNumber
}

func appendTSValidationTypeParts(parts []string, f transform.GoField) []string {
	parts = append(parts, fmt.Sprintf("type: %q", tsValidationType(f)))
	if isIntegerStringField(f) {
		parts = append(parts, fmt.Sprintf("integerFormat: %q", tsIntegerFormat(f.Type.Scalar)))
	}
	return parts
}

// writeTSValidationRules generates a validation rules constant for a message.
// For derived (create/update) messages (CreateSource/UpdateSource non-empty),
// rules are read directly from the derived message's own fields, which carry
// validate annotations copied by gen-proto. Required/optional is determined
// from ConditionFields and RequiredFields.
// For all other messages, only fields with ValidateOptions produce entries.
//
// Example output:
//
//	export const PersonRules = {
//	  name: { required: true, type: "string", minLength: 1 } },
//	} as const
func writeTSValidationRules(b *strings.Builder, msg transform.GoMessage) {
	// Derived message: use own field validate rules with required set from
	// ConditionFields + RequiredFields.
	if msg.UpdateSource != "" || msg.CreateSource != "" {
		writeTSDerivedValidationRules(b, msg)
		return
	}

	// Standard: use the message's own field validate rules.
	type fieldEntry struct {
		name  string
		field transform.GoField
	}
	var entries []fieldEntry
	for _, f := range msg.Fields {
		if f.ValidateOptions != nil {
			entries = append(entries, fieldEntry{name: f.JSONName, field: f})
		}
	}
	if len(entries) == 0 {
		return
	}

	fmt.Fprintf(b, "export const %sRules = {\n", msg.GoName)
	for i, e := range entries {
		writeTSFieldRules(b, e.name, e.field, "  ")
		if i < len(entries)-1 {
			b.WriteString(",\n")
		} else {
			b.WriteString("\n")
		}
	}
	b.WriteString("} as const\n\n")
}

// writeTSFieldRules writes all validation rules for a single field.
func writeTSFieldRules(b *strings.Builder, jsonName string, f transform.GoField, indent string) {
	vo := f.ValidateOptions
	var parts []string

	// required and type are always emitted
	parts = append(parts, fmt.Sprintf("required: %t", vo.Required))
	parts = appendTSValidationTypeParts(parts, f)

	parts = appendConstraintParts(parts, vo, tsConstraintMode(f))

	// Enum constraints
	if vo.DefinedOnly {
		parts = append(parts, "definedOnly: true")
	}

	// Repeated constraints
	if vo.MinItems != nil {
		parts = append(parts, fmt.Sprintf("minItems: %d", *vo.MinItems))
	}
	if vo.MaxItems != nil {
		parts = append(parts, fmt.Sprintf("maxItems: %d", *vo.MaxItems))
	}

	fmt.Fprintf(b, "%s%s: { %s", indent, jsonName, strings.Join(parts, ", "))

	if vo.Items != nil {
		b.WriteString(", items: { ")
		writeItemRules(b, vo.Items, f)
		b.WriteString(" }")
	}

	b.WriteString(" }")
}

// writeTSDerivedValidationRules generates validation rules for a create/update
// message using the derived message's own field ValidateOptions (copied by
// gen-proto from the source). Required/optional is determined from
// ConditionFields (update) and RequiredFields (create).
func writeTSDerivedValidationRules(b *strings.Builder, msg transform.GoMessage) {
	// Build required set from ConditionFields + RequiredFields.
	requiredSet := make(map[string]bool, len(msg.ConditionFields)+len(msg.RequiredFields))
	for _, cf := range msg.ConditionFields {
		requiredSet[cf] = true
	}
	for _, rf := range msg.RequiredFields {
		requiredSet[rf] = true
	}

	// Collect fields with validate rules.
	type fieldEntry struct {
		jsonName string
		field    transform.GoField
		required bool
	}
	var entries []fieldEntry
	for _, f := range msg.Fields {
		vo := f.ValidateOptions
		if vo == nil && f.Type.Kind != model.FieldKindMessage {
			continue
		}
		entries = append(entries, fieldEntry{
			jsonName: f.JSONName,
			field:    f,
			required: requiredSet[f.Name],
		})
	}
	if len(entries) == 0 {
		return
	}

	fmt.Fprintf(b, "export const %sRules = {\n", msg.GoName)
	for i, e := range entries {
		writeTSDerivedFieldRules(b, e.jsonName, e.field, e.required, "  ")
		if i < len(entries)-1 {
			b.WriteString(",\n")
		} else {
			b.WriteString("\n")
		}
	}
	b.WriteString("} as const\n\n")
}

// writeTSDerivedFieldRules writes validation rules for a single field in a
// derived (create/update) message. Constraint rules come from the field's own
// ValidateOptions (copied by gen-proto); required is determined by the derived
// message's ConditionFields/RequiredFields context.
func writeTSDerivedFieldRules(b *strings.Builder, jsonName string, f transform.GoField, required bool, indent string) {
	vo := f.ValidateOptions

	var parts []string
	parts = append(parts, fmt.Sprintf("required: %t", required))
	parts = appendTSValidationTypeParts(parts, f)

	if vo != nil {
		parts = appendConstraintParts(parts, vo, tsConstraintMode(f))
		if vo.DefinedOnly {
			parts = append(parts, "definedOnly: true")
		}
		if vo.MinItems != nil {
			parts = append(parts, fmt.Sprintf("minItems: %d", *vo.MinItems))
		}
		if vo.MaxItems != nil {
			parts = append(parts, fmt.Sprintf("maxItems: %d", *vo.MaxItems))
		}
	}

	fmt.Fprintf(b, "%s%s: { %s", indent, jsonName, strings.Join(parts, ", "))

	if vo != nil && vo.Items != nil {
		b.WriteString(", items: { ")
		writeItemRules(b, vo.Items, f)
		b.WriteString(" }")
	}

	b.WriteString(" }")
}

// tsItemValidationType returns the validation "type" value for items of a repeated field.
// parentField must be a repeated field.
func tsItemValidationType(parentField transform.GoField) string {
	switch parentField.Type.Kind {
	case model.FieldKindScalar:
		return tsScalarValidationType(parentField.Type.Scalar)
	case model.FieldKindEnum:
		return "enum"
	case model.FieldKindMessage:
		return "object"
	default:
		return "unknown"
	}
}

// appendConstraintParts appends TS validation rule key-value pairs for the
// constraint fields of vo that are shared between field-level and item-level rules:
// string, signed integer, unsigned integer, and float constraints.
func appendConstraintParts(parts []string, vo *model.ValidateFieldOptions, mode tsConstraintValueMode) []string {
	// String constraints
	if vo.MinLen != nil {
		parts = append(parts, fmt.Sprintf("minLength: %d", *vo.MinLen))
	}
	if vo.MaxLen != nil {
		parts = append(parts, fmt.Sprintf("maxLength: %d", *vo.MaxLen))
	}
	if vo.Pattern != "" {
		parts = append(parts, fmt.Sprintf("pattern: %q", vo.Pattern))
	}
	if vo.Email {
		parts = append(parts, "format: \"email\"")
	}
	if vo.URI {
		parts = append(parts, "format: \"uri\"")
	}
	if len(vo.InStr) > 0 {
		elems := make([]string, len(vo.InStr))
		for i, v := range vo.InStr {
			elems[i] = fmt.Sprintf("%q", v)
		}
		parts = append(parts, "enum: ["+strings.Join(elems, ", ")+"]")
	}
	if len(vo.NotInStr) > 0 {
		elems := make([]string, len(vo.NotInStr))
		for i, v := range vo.NotInStr {
			elems[i] = fmt.Sprintf("%q", v)
		}
		parts = append(parts, "notIn: ["+strings.Join(elems, ", ")+"]")
	}

	// Signed integer constraints
	if vo.GTEInt != nil {
		parts = append(parts, tsSignedIntConstraintPart("minimum", *vo.GTEInt, mode))
	}
	if vo.LTEInt != nil {
		parts = append(parts, tsSignedIntConstraintPart("maximum", *vo.LTEInt, mode))
	}
	if vo.GTInt != nil {
		parts = append(parts, tsSignedIntConstraintPart("exclusiveMinimum", *vo.GTInt, mode))
	}
	if vo.LTInt != nil {
		parts = append(parts, tsSignedIntConstraintPart("exclusiveMaximum", *vo.LTInt, mode))
	}
	if len(vo.InInt) > 0 {
		elems := make([]string, len(vo.InInt))
		for i, v := range vo.InInt {
			elems[i] = tsSignedIntValue(v, mode)
		}
		parts = append(parts, "enum: ["+strings.Join(elems, ", ")+"]")
	}
	if len(vo.NotInInt) > 0 {
		elems := make([]string, len(vo.NotInInt))
		for i, v := range vo.NotInInt {
			elems[i] = tsSignedIntValue(v, mode)
		}
		parts = append(parts, "notIn: ["+strings.Join(elems, ", ")+"]")
	}

	// Unsigned integer constraints
	if vo.GTEUint != nil {
		parts = append(parts, tsUnsignedIntConstraintPart("minimum", *vo.GTEUint, mode))
	}
	if vo.LTEUint != nil {
		parts = append(parts, tsUnsignedIntConstraintPart("maximum", *vo.LTEUint, mode))
	}
	if vo.GTUint != nil {
		parts = append(parts, tsUnsignedIntConstraintPart("exclusiveMinimum", *vo.GTUint, mode))
	}
	if vo.LTUint != nil {
		parts = append(parts, tsUnsignedIntConstraintPart("exclusiveMaximum", *vo.LTUint, mode))
	}
	if len(vo.InUint) > 0 {
		elems := make([]string, len(vo.InUint))
		for i, v := range vo.InUint {
			elems[i] = tsUnsignedIntValue(v, mode)
		}
		parts = append(parts, "enum: ["+strings.Join(elems, ", ")+"]")
	}
	if len(vo.NotInUint) > 0 {
		elems := make([]string, len(vo.NotInUint))
		for i, v := range vo.NotInUint {
			elems[i] = tsUnsignedIntValue(v, mode)
		}
		parts = append(parts, "notIn: ["+strings.Join(elems, ", ")+"]")
	}

	// Float constraints
	if vo.GTEFloat != nil {
		parts = append(parts, fmt.Sprintf("minimum: %g", *vo.GTEFloat))
	}
	if vo.LTEFloat != nil {
		parts = append(parts, fmt.Sprintf("maximum: %g", *vo.LTEFloat))
	}
	if vo.GTFloat != nil {
		parts = append(parts, fmt.Sprintf("exclusiveMinimum: %g", *vo.GTFloat))
	}
	if vo.LTFloat != nil {
		parts = append(parts, fmt.Sprintf("exclusiveMaximum: %g", *vo.LTFloat))
	}

	// Enum constraints
	if len(vo.NotInEnum) > 0 {
		elems := make([]string, len(vo.NotInEnum))
		for i, v := range vo.NotInEnum {
			elems[i] = fmt.Sprintf("%d", v)
		}
		parts = append(parts, "notIn: ["+strings.Join(elems, ", ")+"]")
	}

	return parts
}

func tsSignedIntConstraintPart(name string, value int64, mode tsConstraintValueMode) string {
	return fmt.Sprintf("%s: %s", name, tsSignedIntValue(value, mode))
}

func tsUnsignedIntConstraintPart(name string, value uint64, mode tsConstraintValueMode) string {
	return fmt.Sprintf("%s: %s", name, tsUnsignedIntValue(value, mode))
}

func tsSignedIntValue(value int64, mode tsConstraintValueMode) string {
	switch mode {
	case tsConstraintValueNumber:
		return strconv.FormatInt(value, 10)
	case tsConstraintValueIntegerString:
		return fmt.Sprintf("%q", strconv.FormatInt(value, 10))
	default:
		panic(fmt.Sprintf("tsSignedIntValue: unhandled constraint value mode %d", mode))
	}
}

func tsUnsignedIntValue(value uint64, mode tsConstraintValueMode) string {
	switch mode {
	case tsConstraintValueNumber:
		return strconv.FormatUint(value, 10)
	case tsConstraintValueIntegerString:
		return fmt.Sprintf("%q", strconv.FormatUint(value, 10))
	default:
		panic(fmt.Sprintf("tsUnsignedIntValue: unhandled constraint value mode %d", mode))
	}
}

// writeItemRules writes validation rules for repeated field items (inner constraints).
// parentField is the repeated field whose items are being described; it is used to
// emit the "type" property that identifies the element kind.
func writeItemRules(b *strings.Builder, vo *model.ValidateFieldOptions, parentField transform.GoField) {
	var parts []string

	// type is always emitted first for items
	parts = append(parts, fmt.Sprintf("type: %q", tsItemValidationType(parentField)))

	parts = appendConstraintParts(parts, vo, tsConstraintValueNumber)

	// DefinedOnly for enum items
	if vo.DefinedOnly {
		parts = append(parts, "definedOnly: true")
	}

	b.WriteString(strings.Join(parts, ", "))
}
