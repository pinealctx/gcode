package render

import (
	"strings"
	"testing"

	"github.com/pinealctx/gcode/internal/model"
	"github.com/pinealctx/gcode/internal/transform"
)

// buildRepeatedField builds a GoField with repeated cardinality for the given scalar.
func buildRepeatedField(name string, number int, scalar model.ScalarKind, goType string) transform.GoField {
	return transform.GoField{
		Field: model.Field{
			Name:        name,
			Number:      number,
			Cardinality: model.CardinalityRepeated,
			Type:        model.FieldType{Kind: model.FieldKindScalar, Scalar: scalar},
			JSONName:    name,
		},
		GoName: strings.ToUpper(name[:1]) + name[1:],
		GoType: goType,
	}
}

// buildRepeatedEnumField builds a GoField with repeated enum cardinality.
func buildRepeatedEnumField(name string, number int, goType string) transform.GoField {
	return transform.GoField{
		Field: model.Field{
			Name:        name,
			Number:      number,
			Cardinality: model.CardinalityRepeated,
			Type:        model.FieldType{Kind: model.FieldKindEnum},
			JSONName:    name,
		},
		GoName:     strings.ToUpper(name[:1]) + name[1:],
		GoType:     goType,
		ElemGoType: goType[2:], // strip "[]"
	}
}

// buildRepeatedMessageField builds a GoField with repeated message cardinality.
func buildRepeatedMessageField(name string, number int, goType string) transform.GoField {
	return transform.GoField{
		Field: model.Field{
			Name:        name,
			Number:      number,
			Cardinality: model.CardinalityRepeated,
			Type:        model.FieldType{Kind: model.FieldKindMessage},
			JSONName:    name,
		},
		GoName:     strings.ToUpper(name[:1]) + name[1:],
		GoType:     goType,
		ElemGoType: goType[3:], // strip "[]*"
	}
}

// buildFileWithFields generates a complete dao file for a message with the given fields.
func buildFileWithFields(t *testing.T, fields []transform.GoField) string {
	t.Helper()
	gf := transform.GoFile{
		Source:  "test.proto",
		Package: "testpb",
		Messages: []transform.GoMessage{
			{GoName: "Msg", Fields: fields},
		},
	}
	got, err := File(gf, testModule, Context{})
	if err != nil {
		t.Fatalf("File() error: %v", err)
	}
	return string(got)
}

// TestMarshalPackedFixed32 verifies that fixed32/sfixed32/float repeated fields
// generate AppendFixed32/AppendFloat calls (writeMarshalPackedElement fixed32 branch).
func TestMarshalPackedFixed32(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name   string
		scalar model.ScalarKind
		goType string
		want   string
	}{
		{"fixed32", model.ScalarFixed32, "[]uint32", "AppendFixed32(b, v)"},
		{"sfixed32", model.ScalarSfixed32, "[]int32", "AppendFixed32(b, uint32(v))"},
		{"float", model.ScalarFloat, "[]float32", "AppendFloat(b, v)"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			src := buildFileWithFields(t, []transform.GoField{
				buildRepeatedField("vals", 1, tt.scalar, tt.goType),
			})
			if !strings.Contains(src, tt.want) {
				t.Errorf("missing %q in generated output", tt.want)
			}
		})
	}
}

// TestMarshalPackedFixed64 verifies that fixed64/sfixed64/double repeated fields
// generate AppendFixed64/AppendDouble calls (writeMarshalPackedElement fixed64 branch).
func TestMarshalPackedFixed64(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name   string
		scalar model.ScalarKind
		goType string
		want   string
	}{
		{"fixed64", model.ScalarFixed64, "[]uint64", "AppendFixed64(b, v)"},
		{"sfixed64", model.ScalarSfixed64, "[]int64", "AppendFixed64(b, uint64(v))"},
		{"double", model.ScalarDouble, "[]float64", "AppendDouble(b, v)"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			src := buildFileWithFields(t, []transform.GoField{
				buildRepeatedField("vals", 1, tt.scalar, tt.goType),
			})
			if !strings.Contains(src, tt.want) {
				t.Errorf("missing %q in generated output", tt.want)
			}
		})
	}
}

// TestMarshalPackedEnum verifies that repeated enum fields generate packed enum encoding
// (writeMarshalPackedEnum) and decoding (writeUnmarshalPackedEnum).
func TestMarshalPackedEnum(t *testing.T) {
	t.Parallel()

	src := buildFileWithFields(t, []transform.GoField{
		buildRepeatedEnumField("statuses", 1, "[]Status"),
	})

	containsAll(t, src, map[string]string{
		"size enum":   "SizeEnum(int32(v))",
		"append enum": "AppendVarint(b, uint64(v))",
		"decode enum": "Status(int32(v))",
	})
}

// TestUnmarshalPackedFixed32Assign verifies that packed fixed32/sfixed32/float fields
// generate correct decode assignment (writePackedFixed32Assign).
func TestUnmarshalPackedFixed32Assign(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name   string
		scalar model.ScalarKind
		goType string
		want   string
	}{
		{"fixed32", model.ScalarFixed32, "[]uint32", "append(x.Vals, v)"},
		{"sfixed32", model.ScalarSfixed32, "[]int32", "append(x.Vals, int32(v))"},
		{"float", model.ScalarFloat, "[]float32", "Float32frombits(v)"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			src := buildFileWithFields(t, []transform.GoField{
				buildRepeatedField("vals", 1, tt.scalar, tt.goType),
			})
			if !strings.Contains(src, tt.want) {
				t.Errorf("missing %q in generated output", tt.want)
			}
		})
	}
}

// TestUnmarshalPackedFixed64Assign verifies that packed fixed64/sfixed64/double fields
// generate correct decode assignment (writePackedFixed64Assign).
func TestUnmarshalPackedFixed64Assign(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name   string
		scalar model.ScalarKind
		goType string
		want   string
	}{
		{"fixed64", model.ScalarFixed64, "[]uint64", "append(x.Vals, v)"},
		{"sfixed64", model.ScalarSfixed64, "[]int64", "append(x.Vals, int64(v))"},
		{"double", model.ScalarDouble, "[]float64", "Float64frombits(v)"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			src := buildFileWithFields(t, []transform.GoField{
				buildRepeatedField("vals", 1, tt.scalar, tt.goType),
			})
			if !strings.Contains(src, tt.want) {
				t.Errorf("missing %q in generated output", tt.want)
			}
		})
	}
}

// TestUnmarshalRepeatedMessage verifies that repeated message fields generate
// correct decode logic (writeUnmarshalRepeatedMessage).
func TestUnmarshalRepeatedMessage(t *testing.T) {
	t.Parallel()

	src := buildFileWithFields(t, []transform.GoField{
		buildRepeatedMessageField("items", 1, "[]*Item"),
	})

	containsAll(t, src, map[string]string{
		"wire type check": "WireBytes",
		"new element":     "new(Item)",
		"unmarshal call":  "UnmarshalBinary",
		"append":          "append(x.Items",
	})
}
