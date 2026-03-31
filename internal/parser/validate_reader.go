package parser

import (
	"fmt"
	"regexp"

	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/reflect/protoreflect"
	"google.golang.org/protobuf/types/descriptorpb"
	"google.golang.org/protobuf/types/dynamicpb"

	"github.com/pinealctx/gcode/internal/model"
)

// readValidateOptions extracts buf.validate field constraints from FieldOptions.
// Returns nil if no validate annotation is present.
// Returns an error if a constraint conflict is detected.
func readValidateOptions(
	opts proto.Message,
	ext protoreflect.ExtensionType,
	kind protoreflect.Kind,
	fieldFullName string,
) (*model.ValidateFieldOptions, error) {
	if opts == nil {
		return nil, nil
	}
	fieldOpts, ok := opts.(*descriptorpb.FieldOptions)
	if !ok || fieldOpts == nil {
		return nil, nil
	}
	if !proto.HasExtension(fieldOpts, ext) {
		return nil, nil
	}
	val := proto.GetExtension(fieldOpts, ext)
	fc, ok := val.(*dynamicpb.Message)
	if !ok || fc == nil {
		return nil, nil
	}
	return parseFieldConstraints(fc, kind, fieldFullName)
}

// placeholder to be replaced
// kind is the proto field kind, used to select the correct constraint group.
// fieldFullName is used in error messages.
func parseFieldConstraints(fc *dynamicpb.Message, kind protoreflect.Kind, fieldFullName string) (*model.ValidateFieldOptions, error) {
	if fc == nil {
		return nil, nil
	}

	opts := &model.ValidateFieldOptions{}
	hasAny := false

	// message-level required (top-level bool on FieldConstraints)
	if hasField(fc, "required") && getBoolField(fc, "required") {
		opts.Required = true
		hasAny = true
	}

	switch kind {
	case protoreflect.StringKind:
		strRules := getMessageField(fc, "string")
		if strRules != nil {
			if err := parseStringRules(strRules, opts, fieldFullName); err != nil {
				return nil, err
			}
			hasAny = true
		}

	case protoreflect.Int32Kind, protoreflect.Sint32Kind, protoreflect.Sfixed32Kind:
		rules := getMessageField(fc, "int32")
		if rules != nil {
			parseSignedIntRules(rules, opts)
			hasAny = true
		}

	case protoreflect.Int64Kind, protoreflect.Sint64Kind, protoreflect.Sfixed64Kind:
		rules := getMessageField(fc, "int64")
		if rules != nil {
			parseSignedIntRules(rules, opts)
			hasAny = true
		}

	case protoreflect.Uint32Kind, protoreflect.Fixed32Kind:
		rules := getMessageField(fc, "uint32")
		if rules != nil {
			parseUnsignedIntRules(rules, opts)
			hasAny = true
		}

	case protoreflect.Uint64Kind, protoreflect.Fixed64Kind:
		rules := getMessageField(fc, "uint64")
		if rules != nil {
			parseUnsignedIntRules(rules, opts)
			hasAny = true
		}

	case protoreflect.FloatKind:
		rules := getMessageField(fc, "float")
		if rules != nil {
			parseFloatRules(rules, opts)
			hasAny = true
		}

	case protoreflect.DoubleKind:
		rules := getMessageField(fc, "double")
		if rules != nil {
			parseFloatRules(rules, opts)
			hasAny = true
		}

	case protoreflect.BoolKind:
		// bool required is not supported
		if opts.Required {
			return nil, fmt.Errorf("field %q (bool): required constraint is not supported for bool fields", fieldFullName)
		}

	case protoreflect.BytesKind:
		bytesRules := getMessageField(fc, "bytes")
		if bytesRules != nil {
			if err := parseBytesRules(bytesRules, opts, fieldFullName); err != nil {
				return nil, err
			}
			hasAny = true
		}

	case protoreflect.EnumKind:
		enumRules := getMessageField(fc, "enum")
		if enumRules != nil {
			opts.DefinedOnly = getBoolField(enumRules, "defined_only")
			if opts.DefinedOnly {
				hasAny = true
			}
		}

	case protoreflect.MessageKind:
		// message required already handled above via top-level required field
		// repeated is handled separately below

	default:
		// unsupported kinds: no constraints
	}

	// repeated constraints
	repRules := getMessageField(fc, "repeated")
	if repRules != nil {
		if err := parseRepeatedRules(repRules, opts, fieldFullName); err != nil {
			return nil, err
		}
		hasAny = true
	}

	if !hasAny {
		return nil, nil
	}
	return opts, nil
}

// parseStringRules fills string constraints from a StringRules dynamicpb.Message.
// Note: StringRules does not have a required field; required is a top-level FieldConstraints field.
func parseStringRules(r *dynamicpb.Message, opts *model.ValidateFieldOptions, fieldFullName string) error {
	if hasField(r, "min_len") {
		v := getUint64Field(r, "min_len")
		opts.MinLen = &v
	}
	if hasField(r, "max_len") {
		v := getUint64Field(r, "max_len")
		opts.MaxLen = &v
	}
	// validate min_len <= max_len
	if opts.MinLen != nil && opts.MaxLen != nil && *opts.MinLen > *opts.MaxLen {
		return fmt.Errorf("field %q (string): min_len (%d) must be <= max_len (%d)", fieldFullName, *opts.MinLen, *opts.MaxLen)
	}
	if hasField(r, "pattern") {
		p := getStringField(r, "pattern")
		if p != "" {
			if _, err := regexp.Compile(p); err != nil {
				return fmt.Errorf("field %q (string): pattern %q is not a valid RE2 regexp: %w", fieldFullName, p, err)
			}
			opts.Pattern = p
		}
	}
	opts.Email = getBoolField(r, "email")
	opts.URI = getBoolField(r, "uri")

	// in set — empty in set is a user error (always fails)
	if hasField(r, "in") {
		inVals := getListField(r, "in")
		if len(inVals) == 0 {
			return fmt.Errorf("field %q (string): in set is empty, constraint will always fail", fieldFullName)
		}
		strs := make([]string, len(inVals))
		for i, v := range inVals {
			strs[i] = v.String()
		}
		// check empty string + required conflict
		if opts.Required {
			for _, s := range strs {
				if s == "" {
					return fmt.Errorf("field %q (string): in set contains empty string, which conflicts with required=true", fieldFullName)
				}
			}
		}
		opts.InStr = strs
	}

	// not_in set (empty = silent skip)
	notInVals := getListField(r, "not_in")
	if len(notInVals) > 0 {
		strs := make([]string, len(notInVals))
		for i, v := range notInVals {
			strs[i] = v.String()
		}
		opts.NotInStr = strs
	}
	return nil
}

// parseSignedIntRules fills signed integer constraints (int32/int64 rules share same field names).
func parseSignedIntRules(r *dynamicpb.Message, opts *model.ValidateFieldOptions) {
	if hasField(r, "gt") {
		v := getInt64Field(r, "gt")
		opts.GTInt = &v
	}
	if hasField(r, "gte") {
		v := getInt64Field(r, "gte")
		opts.GTEInt = &v
	}
	if hasField(r, "lt") {
		v := getInt64Field(r, "lt")
		opts.LTInt = &v
	}
	if hasField(r, "lte") {
		v := getInt64Field(r, "lte")
		opts.LTEInt = &v
	}
	if hasField(r, "in") {
		inVals := getListField(r, "in")
		if len(inVals) > 0 {
			vals := make([]int64, len(inVals))
			for i, v := range inVals {
				vals[i] = v.Int()
			}
			opts.InInt = vals
		}
	}
	notInVals := getListField(r, "not_in")
	if len(notInVals) > 0 {
		vals := make([]int64, len(notInVals))
		for i, v := range notInVals {
			vals[i] = v.Int()
		}
		opts.NotInInt = vals
	}
}

// parseUnsignedIntRules fills unsigned integer constraints.
func parseUnsignedIntRules(r *dynamicpb.Message, opts *model.ValidateFieldOptions) {
	if hasField(r, "gt") {
		v := getUint64Field(r, "gt")
		opts.GTUint = &v
	}
	if hasField(r, "gte") {
		v := getUint64Field(r, "gte")
		opts.GTEUint = &v
	}
	if hasField(r, "lt") {
		v := getUint64Field(r, "lt")
		opts.LTUint = &v
	}
	if hasField(r, "lte") {
		v := getUint64Field(r, "lte")
		opts.LTEUint = &v
	}
	if hasField(r, "in") {
		inVals := getListField(r, "in")
		if len(inVals) > 0 {
			vals := make([]uint64, len(inVals))
			for i, v := range inVals {
				vals[i] = v.Uint()
			}
			opts.InUint = vals
		}
	}
	notInVals := getListField(r, "not_in")
	if len(notInVals) > 0 {
		vals := make([]uint64, len(notInVals))
		for i, v := range notInVals {
			vals[i] = v.Uint()
		}
		opts.NotInUint = vals
	}
}

// parseFloatRules fills float/double constraints (no in/not_in).
func parseFloatRules(r *dynamicpb.Message, opts *model.ValidateFieldOptions) {
	if hasField(r, "gt") {
		v := getFloat64Field(r, "gt")
		opts.GTFloat = &v
	}
	if hasField(r, "gte") {
		v := getFloat64Field(r, "gte")
		opts.GTEFloat = &v
	}
	if hasField(r, "lt") {
		v := getFloat64Field(r, "lt")
		opts.LTFloat = &v
	}
	if hasField(r, "lte") {
		v := getFloat64Field(r, "lte")
		opts.LTEFloat = &v
	}
}

// parseBytesRules fills bytes constraints.
func parseBytesRules(r *dynamicpb.Message, opts *model.ValidateFieldOptions, fieldFullName string) error {
	if hasField(r, "min_len") {
		v := getUint64Field(r, "min_len")
		opts.MinLen = &v
	}
	if hasField(r, "max_len") {
		v := getUint64Field(r, "max_len")
		opts.MaxLen = &v
	}
	if opts.MinLen != nil && opts.MaxLen != nil && *opts.MinLen > *opts.MaxLen {
		return fmt.Errorf("field %q (bytes): min_len (%d) must be <= max_len (%d)", fieldFullName, *opts.MinLen, *opts.MaxLen)
	}
	return nil
}

// parseRepeatedRules fills repeated field constraints, including recursive items constraints.
func parseRepeatedRules(r *dynamicpb.Message, opts *model.ValidateFieldOptions, fieldFullName string) error {
	if hasField(r, "min_items") {
		v := getUint64Field(r, "min_items")
		opts.MinItems = &v
	}
	if hasField(r, "max_items") {
		v := getUint64Field(r, "max_items")
		opts.MaxItems = &v
	}
	if opts.MinItems != nil && opts.MaxItems != nil && *opts.MinItems > *opts.MaxItems {
		return fmt.Errorf("field %q (repeated): min_items (%d) must be <= max_items (%d)", fieldFullName, *opts.MinItems, *opts.MaxItems)
	}
	// items: element-level FieldConstraints; kind is unknown here, so we probe
	// which sub-rule message is present and dispatch accordingly.
	itemsFC := getMessageField(r, "items")
	if itemsFC != nil {
		itemOpts, err := parseItemConstraints(itemsFC, fieldFullName+"[items]")
		if err != nil {
			return err
		}
		opts.Items = itemOpts
	}
	return nil
}

// parseItemConstraints parses element-level FieldConstraints for repeated field items.
// Since the element kind is not available at this level, we probe which sub-rule
// message is present and dispatch to the appropriate parser.
func parseItemConstraints(fc *dynamicpb.Message, fieldFullName string) (*model.ValidateFieldOptions, error) {
	if fc == nil {
		return nil, nil
	}
	opts := &model.ValidateFieldOptions{}
	hasAny := false

	if r := getMessageField(fc, "string"); r != nil {
		if err := parseStringRules(r, opts, fieldFullName); err != nil {
			return nil, err
		}
		hasAny = true
	}
	if r := getMessageField(fc, "int32"); r != nil {
		parseSignedIntRules(r, opts)
		hasAny = true
	}
	if r := getMessageField(fc, "int64"); r != nil {
		parseSignedIntRules(r, opts)
		hasAny = true
	}
	if r := getMessageField(fc, "uint32"); r != nil {
		parseUnsignedIntRules(r, opts)
		hasAny = true
	}
	if r := getMessageField(fc, "uint64"); r != nil {
		parseUnsignedIntRules(r, opts)
		hasAny = true
	}
	if r := getMessageField(fc, "float"); r != nil {
		parseFloatRules(r, opts)
		hasAny = true
	}
	if r := getMessageField(fc, "double"); r != nil {
		parseFloatRules(r, opts)
		hasAny = true
	}
	if r := getMessageField(fc, "bytes"); r != nil {
		if err := parseBytesRules(r, opts, fieldFullName); err != nil {
			return nil, err
		}
		hasAny = true
	}
	if r := getMessageField(fc, "enum"); r != nil {
		opts.DefinedOnly = getBoolField(r, "defined_only")
		if opts.DefinedOnly {
			hasAny = true
		}
	}

	if !hasAny {
		return nil, nil
	}
	return opts, nil
}
