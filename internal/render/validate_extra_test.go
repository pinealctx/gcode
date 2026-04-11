package render

import (
	"strings"
	"testing"

	"github.com/pinealctx/gcode/internal/model"
	"github.com/pinealctx/gcode/internal/transform"
)

// buildValidateFile generates a validate file for a message with the given fields.
func buildValidateFile(t *testing.T, fields []transform.GoField) string {
	t.Helper()
	gf := transform.GoFile{
		Source:  "test.proto",
		Package: "testpb",
		Messages: []transform.GoMessage{
			{GoName: "Msg", Fields: fields},
		},
	}
	got, err := ValidateFile(gf, testModule, Context{})
	if err != nil {
		t.Fatalf("ValidateFile() error: %v", err)
	}
	return string(got)
}

// buildScalarField builds a singular scalar GoField with the given validate options.
func buildScalarField(name string, number int, scalar model.ScalarKind, goType string, vo *model.ValidateFieldOptions) transform.GoField {
	return transform.GoField{
		Field: model.Field{
			Name:            name,
			Number:          number,
			Cardinality:     model.CardinalitySingular,
			Type:            model.FieldType{Kind: model.FieldKindScalar, Scalar: scalar},
			JSONName:        name,
			ValidateOptions: vo,
		},
		GoName: strings.ToUpper(name[:1]) + name[1:],
		GoType: goType,
	}
}

// TestFloatValidation verifies that float/double constraints generate correct code
// (writeFloatValidation).
func TestFloatValidation(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name   string
		scalar model.ScalarKind
		goType string
		vo     *model.ValidateFieldOptions
		want   string
	}{
		{
			name:   "float_gt",
			scalar: model.ScalarFloat,
			goType: "float32",
			vo:     &model.ValidateFieldOptions{GTFloat: ptr(0.0)},
			want:   `<= 0`,
		},
		{
			name:   "float_gte",
			scalar: model.ScalarFloat,
			goType: "float32",
			vo:     &model.ValidateFieldOptions{GTEFloat: ptr(1.5)},
			want:   `< 1.5`,
		},
		{
			name:   "double_lt",
			scalar: model.ScalarDouble,
			goType: "float64",
			vo:     &model.ValidateFieldOptions{LTFloat: ptr(100.0)},
			want:   `>= 100`,
		},
		{
			name:   "double_lte",
			scalar: model.ScalarDouble,
			goType: "float64",
			vo:     &model.ValidateFieldOptions{LTEFloat: ptr(99.9)},
			want:   `> 99.9`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			src := buildValidateFile(t, []transform.GoField{
				buildScalarField("val", 1, tt.scalar, tt.goType, tt.vo),
			})
			if !strings.Contains(src, tt.want) {
				t.Errorf("missing %q in generated output:\n%s", tt.want, src)
			}
		})
	}
}

// TestStringInCheck verifies that string in/not_in constraints generate correct code
// (writeStringInCheck, writeStringNotInCheck, writeStringSetCheck).
func TestStringInCheck(t *testing.T) {
	t.Parallel()

	// in check: must be one of ["a", "b"]
	src := buildValidateFile(t, []transform.GoField{
		buildScalarField("s", 1, model.ScalarString, "string", &model.ValidateFieldOptions{
			InStr: []string{"a", "b"},
		}),
	})
	containsAll(t, src, map[string]string{
		"found var":   "found := false",
		"in check":    `"a"`,
		"rule in":     `"in"`,
		"must be one": "must be one of",
	})
}

// TestStringNotInCheck verifies that string not_in constraints generate correct code.
func TestStringNotInCheck(t *testing.T) {
	t.Parallel()

	src := buildValidateFile(t, []transform.GoField{
		buildScalarField("s", 1, model.ScalarString, "string", &model.ValidateFieldOptions{
			NotInStr: []string{"x", "y"},
		}),
	})
	containsAll(t, src, map[string]string{
		"not_in rule":     `"not_in"`,
		"must not be one": "must not be one of",
		"value x":         `"x"`,
	})
}

// TestSignedInCheck verifies that signed int in constraints generate correct code
// (writeSignedInCheck).
func TestSignedInCheck(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name   string
		scalar model.ScalarKind
		goType string
		want   string
	}{
		{
			name:   "int32_in",
			scalar: model.ScalarInt32,
			goType: "int32",
			want:   "[]int32{",
		},
		{
			name:   "int64_in",
			scalar: model.ScalarInt64,
			goType: "int64",
			want:   "[]int64{",
		},
		{
			name:   "sint32_in",
			scalar: model.ScalarSint32,
			goType: "int32",
			want:   "[]int32{",
		},
		{
			name:   "sint64_in",
			scalar: model.ScalarSint64,
			goType: "int64",
			want:   "[]int64{",
		},
		{
			name:   "sfixed32_in",
			scalar: model.ScalarSfixed32,
			goType: "int32",
			want:   "[]int32{",
		},
		{
			name:   "sfixed64_in",
			scalar: model.ScalarSfixed64,
			goType: "int64",
			want:   "[]int64{",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			src := buildValidateFile(t, []transform.GoField{
				buildScalarField("v", 1, tt.scalar, tt.goType, &model.ValidateFieldOptions{
					InInt: []int64{1, 2, 3},
				}),
			})
			if !strings.Contains(src, tt.want) {
				t.Errorf("missing %q in generated output", tt.want)
			}
			if !strings.Contains(src, "found := false") {
				t.Errorf("missing found var in generated output")
			}
		})
	}
}

// TestUnsignedInCheck verifies that unsigned int in/not_in constraints generate correct code
// (writeUnsignedInCheck, writeUnsignedNotInCheck, unsignedGoType).
func TestUnsignedInCheck(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name   string
		scalar model.ScalarKind
		goType string
		want   string
	}{
		{
			name:   "uint32_in",
			scalar: model.ScalarUint32,
			goType: "uint32",
			want:   "[]uint32{",
		},
		{
			name:   "uint64_in",
			scalar: model.ScalarUint64,
			goType: "uint64",
			want:   "[]uint64{",
		},
		{
			name:   "fixed32_in",
			scalar: model.ScalarFixed32,
			goType: "uint32",
			want:   "[]uint32{",
		},
		{
			name:   "fixed64_in",
			scalar: model.ScalarFixed64,
			goType: "uint64",
			want:   "[]uint64{",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			src := buildValidateFile(t, []transform.GoField{
				buildScalarField("v", 1, tt.scalar, tt.goType, &model.ValidateFieldOptions{
					InUint: []uint64{1, 2, 3},
				}),
			})
			if !strings.Contains(src, tt.want) {
				t.Errorf("missing %q in generated output", tt.want)
			}
		})
	}
}

// TestUnsignedNotInCheck verifies that unsigned int not_in constraints generate correct code.
func TestUnsignedNotInCheck(t *testing.T) {
	t.Parallel()

	src := buildValidateFile(t, []transform.GoField{
		buildScalarField("v", 1, model.ScalarUint32, "uint32", &model.ValidateFieldOptions{
			NotInUint: []uint64{0, 99},
		}),
	})
	containsAll(t, src, map[string]string{
		"uint32 type":     "[]uint32{",
		"not_in rule":     `"not_in"`,
		"must not be one": "must not be one of",
	})
}

// TestBuildStringList verifies buildStringList helper output.
func TestBuildStringList(t *testing.T) {
	t.Parallel()

	got := buildStringList([]string{"a", "b", "c"})
	if got != "[a, b, c]" {
		t.Errorf("buildStringList = %q, want %q", got, "[a, b, c]")
	}
	got = buildStringList([]string{"x"})
	if got != "[x]" {
		t.Errorf("buildStringList single = %q, want %q", got, "[x]")
	}
}

// TestBuildUint64List verifies buildUint64List helper output.
func TestBuildUint64List(t *testing.T) {
	t.Parallel()

	got := buildUint64List([]uint64{1, 2, 3})
	if got != "[1, 2, 3]" {
		t.Errorf("buildUint64List = %q, want %q", got, "[1, 2, 3]")
	}
	got = buildUint64List([]uint64{42})
	if got != "[42]" {
		t.Errorf("buildUint64List single = %q, want %q", got, "[42]")
	}
}

// TestUnsignedGoType verifies unsignedGoType returns correct Go type strings.
func TestUnsignedGoType(t *testing.T) {
	t.Parallel()

	if got := unsignedGoType(model.ScalarUint64); got != "uint64" {
		t.Errorf("unsignedGoType(uint64) = %q, want uint64", got)
	}
	if got := unsignedGoType(model.ScalarFixed64); got != "uint64" {
		t.Errorf("unsignedGoType(fixed64) = %q, want uint64", got)
	}
	if got := unsignedGoType(model.ScalarUint32); got != "uint32" {
		t.Errorf("unsignedGoType(uint32) = %q, want uint32", got)
	}
	if got := unsignedGoType(model.ScalarFixed32); got != "uint32" {
		t.Errorf("unsignedGoType(fixed32) = %q, want uint32", got)
	}
}
