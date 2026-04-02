package tsrender

import (
	"strings"
	"testing"

	"github.com/pinealctx/gcode/internal/model"
	"github.com/pinealctx/gcode/internal/transform"
)

// uintPtr returns a pointer to the given uint64 value.
func uintPtr(v uint64) *uint64 { return &v }

// intPtr returns a pointer to the given int64 value.
func intPtr(v int64) *int64 { return &v }

// floatPtr returns a pointer to the given float64 value.
func floatPtr(v float64) *float64 { return &v }

// renderRules is a test helper that renders validation rules for a single message
// and returns the output as a string.
func renderRules(msg transform.GoMessage) string {
	var b strings.Builder
	writeTSValidationRules(&b, msg)
	return b.String()
}

func TestValidationStringRules(t *testing.T) {
	t.Parallel()

	msg := transform.GoMessage{
		GoName: "User",
		Fields: []transform.GoField{
			{
				Field: model.Field{
					JSONName:    "name",
					Cardinality: model.CardinalitySingular,
					Type:        model.FieldType{Kind: model.FieldKindScalar, Scalar: model.ScalarString},
					ValidateOptions: &model.ValidateFieldOptions{
						Required: true,
						MinLen:   uintPtr(1),
						MaxLen:   uintPtr(100),
						Pattern:  `^[a-z]+$`,
					},
				},
			},
			{
				Field: model.Field{
					JSONName:    "email",
					Cardinality: model.CardinalitySingular,
					Type:        model.FieldType{Kind: model.FieldKindScalar, Scalar: model.ScalarString},
					ValidateOptions: &model.ValidateFieldOptions{
						Required: true,
						Email:    true,
					},
				},
			},
			{
				Field: model.Field{
					JSONName:    "website",
					Cardinality: model.CardinalitySingular,
					Type:        model.FieldType{Kind: model.FieldKindScalar, Scalar: model.ScalarString},
					ValidateOptions: &model.ValidateFieldOptions{
						Required: false,
						URI:      true,
					},
				},
			},
			{
				Field: model.Field{
					JSONName:    "role",
					Cardinality: model.CardinalitySingular,
					Type:        model.FieldType{Kind: model.FieldKindScalar, Scalar: model.ScalarString},
					ValidateOptions: &model.ValidateFieldOptions{
						InStr: []string{"admin", "user", "guest"},
					},
				},
			},
			{
				Field: model.Field{
					JSONName:    "ban",
					Cardinality: model.CardinalitySingular,
					Type:        model.FieldType{Kind: model.FieldKindScalar, Scalar: model.ScalarString},
					ValidateOptions: &model.ValidateFieldOptions{
						NotInStr: []string{"root", "system"},
					},
				},
			},
		},
	}

	s := renderRules(msg)
	assertContains(t, s, `export const UserRules = {`)
	assertContains(t, s, `name: { required: true, type: "string", minLength: 1, maxLength: 100, pattern: "^[a-z]+$" }`)
	assertContains(t, s, `email: { required: true, type: "string", format: "email" }`)
	assertContains(t, s, `website: { required: false, type: "string", format: "uri" }`)
	assertContains(t, s, `role: { required: false, type: "string", enum: ["admin", "user", "guest"] }`)
	assertContains(t, s, `ban: { required: false, type: "string", notIn: ["root", "system"] }`)
	assertContains(t, s, `} as const`)
}

func TestValidationIntRules(t *testing.T) {
	t.Parallel()

	msg := transform.GoMessage{
		GoName: "Limits",
		Fields: []transform.GoField{
			{
				Field: model.Field{
					JSONName:    "value",
					Cardinality: model.CardinalitySingular,
					Type:        model.FieldType{Kind: model.FieldKindScalar, Scalar: model.ScalarInt32},
					ValidateOptions: &model.ValidateFieldOptions{
						GTEInt: intPtr(0),
						LTEInt: intPtr(100),
					},
				},
			},
			{
				Field: model.Field{
					JSONName:    "exclusive",
					Cardinality: model.CardinalitySingular,
					Type:        model.FieldType{Kind: model.FieldKindScalar, Scalar: model.ScalarInt32},
					ValidateOptions: &model.ValidateFieldOptions{
						GTInt: intPtr(0),
						LTInt: intPtr(100),
					},
				},
			},
			{
				Field: model.Field{
					JSONName:    "code",
					Cardinality: model.CardinalitySingular,
					Type:        model.FieldType{Kind: model.FieldKindScalar, Scalar: model.ScalarInt32},
					ValidateOptions: &model.ValidateFieldOptions{
						InInt:    []int64{1, 2, 3},
						NotInInt: []int64{0, -1},
					},
				},
			},
			{
				Field: model.Field{
					JSONName:    "uid",
					Cardinality: model.CardinalitySingular,
					Type:        model.FieldType{Kind: model.FieldKindScalar, Scalar: model.ScalarUint32},
					ValidateOptions: &model.ValidateFieldOptions{
						GTEUint: uintPtr(1),
						LTEUint: uintPtr(1000),
					},
				},
			},
			{
				Field: model.Field{
					JSONName:    "xuid",
					Cardinality: model.CardinalitySingular,
					Type:        model.FieldType{Kind: model.FieldKindScalar, Scalar: model.ScalarUint32},
					ValidateOptions: &model.ValidateFieldOptions{
						GTUint: uintPtr(0),
						LTUint: uintPtr(100),
					},
				},
			},
			{
				Field: model.Field{
					JSONName:    "ucode",
					Cardinality: model.CardinalitySingular,
					Type:        model.FieldType{Kind: model.FieldKindScalar, Scalar: model.ScalarUint32},
					ValidateOptions: &model.ValidateFieldOptions{
						InUint:    []uint64{10, 20, 30},
						NotInUint: []uint64{0, 99},
					},
				},
			},
		},
	}

	s := renderRules(msg)
	assertContains(t, s, `value: { required: false, type: "integer", minimum: 0, maximum: 100 }`)
	assertContains(t, s, `exclusive: { required: false, type: "integer", exclusiveMinimum: 0, exclusiveMaximum: 100 }`)
	assertContains(t, s, `code: { required: false, type: "integer", enum: [1, 2, 3], notIn: [0, -1] }`)
	assertContains(t, s, `uid: { required: false, type: "integer", minimum: 1, maximum: 1000 }`)
	assertContains(t, s, `xuid: { required: false, type: "integer", exclusiveMinimum: 0, exclusiveMaximum: 100 }`)
	assertContains(t, s, `ucode: { required: false, type: "integer", enum: [10, 20, 30], notIn: [0, 99] }`)
}

func TestValidationFloatRules(t *testing.T) {
	t.Parallel()

	msg := transform.GoMessage{
		GoName: "Range",
		Fields: []transform.GoField{
			{
				Field: model.Field{
					JSONName:    "ratio",
					Cardinality: model.CardinalitySingular,
					Type:        model.FieldType{Kind: model.FieldKindScalar, Scalar: model.ScalarDouble},
					ValidateOptions: &model.ValidateFieldOptions{
						GTEFloat: floatPtr(0),
						LTEFloat: floatPtr(1),
					},
				},
			},
			{
				Field: model.Field{
					JSONName:    "exclusive",
					Cardinality: model.CardinalitySingular,
					Type:        model.FieldType{Kind: model.FieldKindScalar, Scalar: model.ScalarFloat},
					ValidateOptions: &model.ValidateFieldOptions{
						GTFloat: floatPtr(0),
						LTFloat: floatPtr(100),
					},
				},
			},
		},
	}

	s := renderRules(msg)
	assertContains(t, s, `ratio: { required: false, type: "number", minimum: 0, maximum: 1 }`)
	assertContains(t, s, `exclusive: { required: false, type: "number", exclusiveMinimum: 0, exclusiveMaximum: 100 }`)
}

func TestValidationEnumRules(t *testing.T) {
	t.Parallel()

	msg := transform.GoMessage{
		GoName: "Item",
		Fields: []transform.GoField{
			{
				Field: model.Field{
					JSONName:    "status",
					Cardinality: model.CardinalitySingular,
					Type:        model.FieldType{Kind: model.FieldKindEnum, FullName: "test.Status"},
					ValidateOptions: &model.ValidateFieldOptions{
						DefinedOnly: true,
					},
				},
			},
		},
	}

	s := renderRules(msg)
	assertContains(t, s, `status: { required: false, type: "enum", definedOnly: true }`)
}

func TestValidationRepeatedRules(t *testing.T) {
	t.Parallel()

	minItems := uint64(1)
	maxItems := uint64(100)

	msg := transform.GoMessage{
		GoName: "List",
		Fields: []transform.GoField{
			{
				Field: model.Field{
					JSONName:    "scores",
					Cardinality: model.CardinalityRepeated,
					Type:        model.FieldType{Kind: model.FieldKindScalar, Scalar: model.ScalarInt32},
					ValidateOptions: &model.ValidateFieldOptions{
						MinItems: &minItems,
						MaxItems: &maxItems,
						Items: &model.ValidateFieldOptions{
							GTEInt: intPtr(0),
							LTEInt: intPtr(100),
						},
					},
				},
			},
		},
	}

	s := renderRules(msg)
	assertContains(t, s, `export const ListRules = {`)
	assertContains(t, s, `scores: { required: false, type: "array", minItems: 1, maxItems: 100, items: { minimum: 0, maximum: 100 } }`)
	assertContains(t, s, `} as const`)
}

func TestValidationNoRules(t *testing.T) {
	t.Parallel()

	msg := transform.GoMessage{
		GoName: "Empty",
		Fields: []transform.GoField{
			scalarField("name", "name", model.ScalarString),
			scalarField("age", "age", model.ScalarInt32),
		},
	}

	s := renderRules(msg)
	if s != "" {
		t.Errorf("expected no output for message with no validation rules, got:\n%s", s)
	}
}

func TestValidationMixedWithNoRules(t *testing.T) {
	t.Parallel()

	msg := transform.GoMessage{
		GoName: "Mixed",
		Fields: []transform.GoField{
			{
				Field: model.Field{
					JSONName:    "name",
					Cardinality: model.CardinalitySingular,
					Type:        model.FieldType{Kind: model.FieldKindScalar, Scalar: model.ScalarString},
					ValidateOptions: &model.ValidateFieldOptions{
						Required: true,
					},
				},
			},
			// No ValidateOptions — should not appear in output.
			scalarField("description", "description", model.ScalarString),
			{
				Field: model.Field{
					JSONName:    "count",
					Cardinality: model.CardinalitySingular,
					Type:        model.FieldType{Kind: model.FieldKindScalar, Scalar: model.ScalarInt32},
					ValidateOptions: &model.ValidateFieldOptions{
						GTEInt: intPtr(0),
					},
				},
			},
		},
	}

	s := renderRules(msg)
	assertContains(t, s, `name: { required: true, type: "string" }`)
	assertContains(t, s, `count: { required: false, type: "integer", minimum: 0 }`)
	if strings.Contains(s, "description") {
		t.Errorf("fields without ValidateOptions should not appear in rules, got:\n%s", s)
	}
}

func TestValidationRequiredPlusMinLen(t *testing.T) {
	t.Parallel()

	msg := transform.GoMessage{
		GoName: "CreateReq",
		Fields: []transform.GoField{
			{
				Field: model.Field{
					JSONName:    "title",
					Cardinality: model.CardinalitySingular,
					Type:        model.FieldType{Kind: model.FieldKindScalar, Scalar: model.ScalarString},
					ValidateOptions: &model.ValidateFieldOptions{
						Required: true,
						MinLen:   uintPtr(1),
						MaxLen:   uintPtr(200),
					},
				},
			},
		},
	}

	s := renderRules(msg)
	assertContains(t, s, `title: { required: true, type: "string", minLength: 1, maxLength: 200 }`)
}

func TestValidationBytesRules(t *testing.T) {
	t.Parallel()

	msg := transform.GoMessage{
		GoName: "Upload",
		Fields: []transform.GoField{
			{
				Field: model.Field{
					JSONName:    "data",
					Cardinality: model.CardinalitySingular,
					Type:        model.FieldType{Kind: model.FieldKindScalar, Scalar: model.ScalarBytes},
					ValidateOptions: &model.ValidateFieldOptions{
						Required: true,
						MinLen:   uintPtr(1),
						MaxLen:   uintPtr(1048576),
					},
				},
			},
		},
	}

	s := renderRules(msg)
	assertContains(t, s, `data: { required: true, type: "string", minLength: 1, maxLength: 1048576 }`)
}

func TestValidationMessageType(t *testing.T) {
	t.Parallel()

	msg := transform.GoMessage{
		GoName: "Req",
		Fields: []transform.GoField{
			{
				Field: model.Field{
					JSONName:    "address",
					Cardinality: model.CardinalitySingular,
					Type:        model.FieldType{Kind: model.FieldKindMessage, FullName: "test.Address"},
					ValidateOptions: &model.ValidateFieldOptions{
						Required: true,
					},
				},
			},
		},
	}

	s := renderRules(msg)
	assertContains(t, s, `address: { required: true, type: "object" }`)
}

func TestValidationBoolType(t *testing.T) {
	t.Parallel()

	msg := transform.GoMessage{
		GoName: "Flags",
		Fields: []transform.GoField{
			{
				Field: model.Field{
					JSONName:    "active",
					Cardinality: model.CardinalitySingular,
					Type:        model.FieldType{Kind: model.FieldKindScalar, Scalar: model.ScalarBool},
					ValidateOptions: &model.ValidateFieldOptions{
						Required: true,
					},
				},
			},
		},
	}

	s := renderRules(msg)
	assertContains(t, s, `active: { required: true, type: "boolean" }`)
}

func TestValidationInt64Type(t *testing.T) {
	t.Parallel()

	msg := transform.GoMessage{
		GoName: "BigID",
		Fields: []transform.GoField{
			{
				Field: model.Field{
					JSONName:    "id",
					Cardinality: model.CardinalitySingular,
					Type:        model.FieldType{Kind: model.FieldKindScalar, Scalar: model.ScalarInt64},
					ValidateOptions: &model.ValidateFieldOptions{
						Required: true,
					},
				},
			},
		},
	}

	s := renderRules(msg)
	assertContains(t, s, `id: { required: true, type: "integer" }`)
}

func TestValidationRepeatedWithEnumItems(t *testing.T) {
	t.Parallel()

	msg := transform.GoMessage{
		GoName: "Selector",
		Fields: []transform.GoField{
			{
				Field: model.Field{
					JSONName:    "statuses",
					Cardinality: model.CardinalityRepeated,
					Type:        model.FieldType{Kind: model.FieldKindEnum, FullName: "test.Status"},
					ValidateOptions: &model.ValidateFieldOptions{
						MinItems: uintPtr(1),
						MaxItems: uintPtr(10),
						Items: &model.ValidateFieldOptions{
							DefinedOnly: true,
						},
					},
				},
			},
		},
	}

	s := renderRules(msg)
	assertContains(t, s, `statuses: { required: false, type: "array", minItems: 1, maxItems: 10, items: { definedOnly: true } }`)
}

func TestValidationTypeUnknown(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		field    transform.GoField
		wantType string
	}{
		{
			name: "unknown scalar kind",
			field: transform.GoField{
				Field: model.Field{
					JSONName:    "x",
					Cardinality: model.CardinalitySingular,
					Type:        model.FieldType{Kind: model.FieldKindScalar, Scalar: model.ScalarKind("timestamp")},
					ValidateOptions: &model.ValidateFieldOptions{
						Required: true,
					},
				},
			},
			wantType: `"unknown"`,
		},
		{
			name: "unknown field kind",
			field: transform.GoField{
				Field: model.Field{
					JSONName:    "y",
					Cardinality: model.CardinalitySingular,
					Type:        model.FieldType{Kind: model.FieldKind("map")},
					ValidateOptions: &model.ValidateFieldOptions{
						Required: true,
					},
				},
			},
			wantType: `"unknown"`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			msg := transform.GoMessage{
				GoName: "UnknownTest",
				Fields: []transform.GoField{tt.field},
			}
			s := renderRules(msg)
			assertContains(t, s, "type: "+tt.wantType)
		})
	}
}

func TestValidationRepeatedItemsStringRules(t *testing.T) {
	t.Parallel()

	msg := transform.GoMessage{
		GoName: "Tags",
		Fields: []transform.GoField{
			{
				Field: model.Field{
					JSONName:    "items",
					Cardinality: model.CardinalityRepeated,
					Type:        model.FieldType{Kind: model.FieldKindScalar, Scalar: model.ScalarString},
					ValidateOptions: &model.ValidateFieldOptions{
						MinItems: uintPtr(1),
						MaxItems: uintPtr(50),
						Items: &model.ValidateFieldOptions{
							MinLen:  uintPtr(1),
							MaxLen:  uintPtr(100),
							Pattern: `^[a-z]+$`,
							Email:   true,
						},
					},
				},
			},
		},
	}

	s := renderRules(msg)
	assertContains(t, s, `items: { required: false, type: "array", minItems: 1, maxItems: 50, items: { minLength: 1, maxLength: 100, pattern: "^[a-z]+$", format: "email" } }`)
}

func TestValidationRepeatedItemsUintAndFloatRules(t *testing.T) {
	t.Parallel()

	msg := transform.GoMessage{
		GoName: "Data",
		Fields: []transform.GoField{
			{
				Field: model.Field{
					JSONName:    "uids",
					Cardinality: model.CardinalityRepeated,
					Type:        model.FieldType{Kind: model.FieldKindScalar, Scalar: model.ScalarUint32},
					ValidateOptions: &model.ValidateFieldOptions{
						MinItems: uintPtr(1),
						Items: &model.ValidateFieldOptions{
							GTEUint: uintPtr(1),
							LTEUint: uintPtr(100),
							InUint:  []uint64{10, 20},
						},
					},
				},
			},
			{
				Field: model.Field{
					JSONName:    "ratios",
					Cardinality: model.CardinalityRepeated,
					Type:        model.FieldType{Kind: model.FieldKindScalar, Scalar: model.ScalarDouble},
					ValidateOptions: &model.ValidateFieldOptions{
						MinItems: uintPtr(1),
						Items: &model.ValidateFieldOptions{
							GTEFloat: floatPtr(0),
							LTEFloat: floatPtr(1),
						},
					},
				},
			},
		},
	}

	s := renderRules(msg)
	assertContains(t, s, `uids: { required: false, type: "array", minItems: 1, items: { minimum: 1, maximum: 100, enum: [10, 20] } }`)
	assertContains(t, s, `ratios: { required: false, type: "array", minItems: 1, items: { minimum: 0, maximum: 1 } }`)
}

func TestValidationRepeatedItemsURIAndNotIn(t *testing.T) {
	t.Parallel()

	msg := transform.GoMessage{
		GoName: "URLs",
		Fields: []transform.GoField{
			{
				Field: model.Field{
					JSONName:    "links",
					Cardinality: model.CardinalityRepeated,
					Type:        model.FieldType{Kind: model.FieldKindScalar, Scalar: model.ScalarString},
					ValidateOptions: &model.ValidateFieldOptions{
						Items: &model.ValidateFieldOptions{
							MinLen: uintPtr(1),
							URI:    true,
						},
					},
				},
			},
		},
	}

	s := renderRules(msg)
	assertContains(t, s, `links: { required: false, type: "array", items: { minLength: 1, format: "uri" } }`)
}

func TestValidationEmptyInStrSlice(t *testing.T) {
	t.Parallel()

	msg := transform.GoMessage{
		GoName: "Edge",
		Fields: []transform.GoField{
			{
				Field: model.Field{
					JSONName:    "val",
					Cardinality: model.CardinalitySingular,
					Type:        model.FieldType{Kind: model.FieldKindScalar, Scalar: model.ScalarString},
					ValidateOptions: &model.ValidateFieldOptions{
						InStr: []string{},
					},
				},
			},
		},
	}

	s := renderRules(msg)
	assertContains(t, s, `val: { required: false, type: "string" }`)
	if strings.Contains(s, "enum:") {
		t.Errorf("empty InStr slice should not produce enum key, got:\n%s", s)
	}
}

func TestValidationFullFile(t *testing.T) {
	t.Parallel()

	gf := transform.GoFile{
		Source:  "person.proto",
		Package: "test",
		Enums: []transform.GoEnum{
			{
				GoName: "Status",
				Values: []transform.GoEnumValue{
					{GoName: "Status_UNKNOWN", Number: 0},
					{GoName: "Status_ACTIVE", Number: 1},
				},
			},
		},
		Messages: []transform.GoMessage{
			{
				GoName: "Person",
				Fields: []transform.GoField{
					{
						Field: model.Field{
							JSONName:    "name",
							Cardinality: model.CardinalitySingular,
							Type:        model.FieldType{Kind: model.FieldKindScalar, Scalar: model.ScalarString},
							ValidateOptions: &model.ValidateFieldOptions{
								Required: true,
								MinLen:   uintPtr(1),
								MaxLen:   uintPtr(100),
							},
						},
					},
					{
						Field: model.Field{
							JSONName:    "age",
							Cardinality: model.CardinalitySingular,
							Type:        model.FieldType{Kind: model.FieldKindScalar, Scalar: model.ScalarInt32},
							ValidateOptions: &model.ValidateFieldOptions{
								Required: false,
								GTEInt:   intPtr(0),
								LTEInt:   intPtr(150),
							},
						},
					},
					{
						Field: model.Field{
							JSONName:    "email",
							Cardinality: model.CardinalitySingular,
							Type:        model.FieldType{Kind: model.FieldKindScalar, Scalar: model.ScalarString},
							ValidateOptions: &model.ValidateFieldOptions{
								Required: true,
								Email:    true,
							},
						},
					},
					{
						Field: model.Field{
							JSONName:    "status",
							Cardinality: model.CardinalitySingular,
							Type:        model.FieldType{Kind: model.FieldKindEnum, FullName: "test.Status"},
							ValidateOptions: &model.ValidateFieldOptions{
								DefinedOnly: true,
							},
						},
					},
					// No ValidateOptions — should not appear in rules.
					scalarField("note", "note", model.ScalarString),
				},
			},
		},
	}

	out, err := TSFile(gf, nil)
	if err != nil {
		t.Fatalf("TSFile returned error: %v", err)
	}

	s := string(out)
	// Verify interface and enum still present.
	assertContains(t, s, "export interface Person {")
	assertContains(t, s, "export enum Status {")
	// Verify rules present.
	assertContains(t, s, "export const PersonRules = {")
	assertContains(t, s, `name: { required: true, type: "string", minLength: 1, maxLength: 100 }`)
	assertContains(t, s, `age: { required: false, type: "integer", minimum: 0, maximum: 150 }`)
	assertContains(t, s, `email: { required: true, type: "string", format: "email" }`)
	assertContains(t, s, `status: { required: false, type: "enum", definedOnly: true }`)
	// Verify "note" (no rules) not in rules.
	rulesStart := strings.Index(s, "export const PersonRules")
	rulesEnd := strings.Index(s[rulesStart:], "} as const")
	rulesSection := s[rulesStart : rulesStart+rulesEnd]
	if strings.Contains(rulesSection, "note") {
		t.Errorf("field 'note' without ValidateOptions should not appear in rules, got:\n%s", rulesSection)
	}
}
