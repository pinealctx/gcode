package tsrender

import (
	"fmt"
	"strings"

	"github.com/pinealctx/gcode/internal/model"
	"github.com/pinealctx/gcode/internal/transform"
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
		return tsScalarValidationType(f.Type.Scalar)
	case model.FieldKindEnum:
		return "enum"
	case model.FieldKindMessage:
		return "object"
	default:
		return "unknown"
	}
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
	parts = append(parts, fmt.Sprintf("type: %q", tsValidationType(f)))

	parts = appendConstraintParts(parts, vo)

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
	parts = append(parts, fmt.Sprintf("type: %q", tsValidationType(f)))

	if vo != nil {
		parts = appendConstraintParts(parts, vo)
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
func appendConstraintParts(parts []string, vo *model.ValidateFieldOptions) []string {
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
		parts = append(parts, fmt.Sprintf("minimum: %d", *vo.GTEInt))
	}
	if vo.LTEInt != nil {
		parts = append(parts, fmt.Sprintf("maximum: %d", *vo.LTEInt))
	}
	if vo.GTInt != nil {
		parts = append(parts, fmt.Sprintf("exclusiveMinimum: %d", *vo.GTInt))
	}
	if vo.LTInt != nil {
		parts = append(parts, fmt.Sprintf("exclusiveMaximum: %d", *vo.LTInt))
	}
	if len(vo.InInt) > 0 {
		elems := make([]string, len(vo.InInt))
		for i, v := range vo.InInt {
			elems[i] = fmt.Sprintf("%d", v)
		}
		parts = append(parts, "enum: ["+strings.Join(elems, ", ")+"]")
	}
	if len(vo.NotInInt) > 0 {
		elems := make([]string, len(vo.NotInInt))
		for i, v := range vo.NotInInt {
			elems[i] = fmt.Sprintf("%d", v)
		}
		parts = append(parts, "notIn: ["+strings.Join(elems, ", ")+"]")
	}

	// Unsigned integer constraints
	if vo.GTEUint != nil {
		parts = append(parts, fmt.Sprintf("minimum: %d", *vo.GTEUint))
	}
	if vo.LTEUint != nil {
		parts = append(parts, fmt.Sprintf("maximum: %d", *vo.LTEUint))
	}
	if vo.GTUint != nil {
		parts = append(parts, fmt.Sprintf("exclusiveMinimum: %d", *vo.GTUint))
	}
	if vo.LTUint != nil {
		parts = append(parts, fmt.Sprintf("exclusiveMaximum: %d", *vo.LTUint))
	}
	if len(vo.InUint) > 0 {
		elems := make([]string, len(vo.InUint))
		for i, v := range vo.InUint {
			elems[i] = fmt.Sprintf("%d", v)
		}
		parts = append(parts, "enum: ["+strings.Join(elems, ", ")+"]")
	}
	if len(vo.NotInUint) > 0 {
		elems := make([]string, len(vo.NotInUint))
		for i, v := range vo.NotInUint {
			elems[i] = fmt.Sprintf("%d", v)
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

// writeItemRules writes validation rules for repeated field items (inner constraints).
// parentField is the repeated field whose items are being described; it is used to
// emit the "type" property that identifies the element kind.
func writeItemRules(b *strings.Builder, vo *model.ValidateFieldOptions, parentField transform.GoField) {
	var parts []string

	// type is always emitted first for items
	parts = append(parts, fmt.Sprintf("type: %q", tsItemValidationType(parentField)))

	parts = appendConstraintParts(parts, vo)

	// DefinedOnly for enum items
	if vo.DefinedOnly {
		parts = append(parts, "definedOnly: true")
	}

	b.WriteString(strings.Join(parts, ", "))
}
