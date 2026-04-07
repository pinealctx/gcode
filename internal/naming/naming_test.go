package naming

import (
	"strings"
	"testing"

	"github.com/pinealctx/gcode/internal/model"
)

func TestGoCamelCase(t *testing.T) {
	t.Parallel()

	tests := []struct {
		input string
		want  string
	}{
		// Basic snake_case conversions.
		{"foo_bar", "FooBar"},
		{"foo_bar_baz", "FooBarBaz"},
		{"lucky_numbers", "LuckyNumbers"},
		{"user_id", "UserId"},

		// Already CamelCase (message/enum names).
		{"Person", "Person"},
		{"MyMessage", "MyMessage"},

		// Nested type names with dots.
		{"Person.Address", "Person_Address"},
		{"Outer.Middle.Inner", "Outer_Middle_Inner"},

		// Leading underscore.
		{"_field", "XField"},

		// Digits treated as words.
		{"field2name", "Field2Name"},
		{"x2y", "X2Y"},

		// Single character.
		{"x", "X"},

		// Empty string.
		{"", ""},

		// All uppercase.
		{"FOO", "FOO"},
		{"FOO_BAR", "FOO_BAR"},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			got := GoCamelCase(tt.input)
			if got != tt.want {
				t.Errorf("GoCamelCase(%q) = %q, want %q", tt.input, got, tt.want)
			}
		})
	}
}

func TestGoTypeName(t *testing.T) {
	t.Parallel()

	tests := []struct {
		fullName    string
		packageName string
		want        string
	}{
		{"test.foo.Person", "test.foo", "Person"},
		{"test.foo.Person.Address", "test.foo", "Person_Address"},
		{"test.foo.Person.Status", "test.foo", "Person_Status"},
		{"test.foo.Outer.Middle.Inner", "test.foo", "Outer_Middle_Inner"},
		// Cross-package: fullName does not start with packageName, so TrimPrefix
		// is a no-op and GoCamelCase converts the full dotted name.
		{"other.pkg.Person", "test.foo", "OtherPkg_Person"},
	}

	for _, tt := range tests {
		t.Run(tt.fullName, func(t *testing.T) {
			got := GoTypeName(tt.fullName, tt.packageName)
			if got != tt.want {
				t.Errorf("GoTypeName(%q, %q) = %q, want %q", tt.fullName, tt.packageName, got, tt.want)
			}
		})
	}
}

func TestGoFieldName(t *testing.T) {
	t.Parallel()

	tests := []struct {
		input string
		want  string
	}{
		{"name", "Name"},
		{"lucky_numbers", "LuckyNumbers"},
		{"user_id", "UserId"},
		{"created_at", "CreatedAt"},
		{"status", "Status"},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			got := GoFieldName(tt.input)
			if got != tt.want {
				t.Errorf("GoFieldName(%q) = %q, want %q", tt.input, got, tt.want)
			}
		})
	}
}

func TestGoEnumValueName(t *testing.T) {
	t.Parallel()

	tests := []struct {
		parentGoName   string
		protoValueName string
		want           string
	}{
		{"Status", "STATUS_UNSPECIFIED", "Status_STATUS_UNSPECIFIED"},
		{"Status", "STATUS_ACTIVE", "Status_STATUS_ACTIVE"},
		{"Person_Status", "STATUS_ACTIVE", "Person_Status_STATUS_ACTIVE"},
		{"Person", "STATUS_ACTIVE", "Person_STATUS_ACTIVE"},
	}

	for _, tt := range tests {
		t.Run(tt.want, func(t *testing.T) {
			got := GoEnumValueName(tt.parentGoName, tt.protoValueName)
			if got != tt.want {
				t.Errorf("GoEnumValueName(%q, %q) = %q, want %q", tt.parentGoName, tt.protoValueName, got, tt.want)
			}
		})
	}
}

func TestGoScalarType(t *testing.T) {
	t.Parallel()

	tests := []struct {
		kind model.ScalarKind
		want string
	}{
		{model.ScalarDouble, "float64"},
		{model.ScalarFloat, "float32"},
		{model.ScalarInt32, "int32"},
		{model.ScalarInt64, "int64"},
		{model.ScalarUint32, "uint32"},
		{model.ScalarUint64, "uint64"},
		{model.ScalarSint32, "int32"},
		{model.ScalarSint64, "int64"},
		{model.ScalarFixed32, "uint32"},
		{model.ScalarFixed64, "uint64"},
		{model.ScalarSfixed32, "int32"},
		{model.ScalarSfixed64, "int64"},
		{model.ScalarBool, "bool"},
		{model.ScalarString, "string"},
		{model.ScalarBytes, "[]byte"},
	}

	for _, tt := range tests {
		t.Run(string(tt.kind), func(t *testing.T) {
			got := GoScalarType(tt.kind)
			if got != tt.want {
				t.Errorf("GoScalarType(%q) = %q, want %q", tt.kind, got, tt.want)
			}
		})
	}
}

func TestGoScalarType_UnknownKindPanics(t *testing.T) {
	t.Parallel()

	for _, kind := range []model.ScalarKind{"nonexistent", ""} {
		kind := kind
		t.Run(string(kind)+"_panics", func(t *testing.T) {
			t.Parallel()
			defer func() {
				r := recover()
				if r == nil {
					t.Fatal("expected panic for unknown ScalarKind, got none")
				}
				msg, ok := r.(string)
				if !ok {
					t.Fatalf("expected panic value to be string, got %T: %v", r, r)
				}
				if !strings.Contains(msg, "GoScalarType") || !strings.Contains(msg, "unexpected ScalarKind") {
					t.Errorf("panic message %q does not contain expected text", msg)
				}
			}()
			GoScalarType(kind)
		})
	}
}

func TestResolveFieldNames(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name  string
		input []string
		want  []string
	}{
		{
			name:  "no conflicts",
			input: []string{"Name", "Age", "Email"},
			want:  []string{"Name", "Age", "Email"},
		},
		{
			name:  "empty input",
			input: []string{},
			want:  []string{},
		},
		{
			name:  "nil input",
			input: nil,
			want:  []string{},
		},
		{
			name:  "reserved name collision",
			input: []string{"Reset", "Name"},
			want:  []string{"Reset_", "Name"},
		},
		{
			name:  "getter collision",
			input: []string{"GetName", "Name"},
			want:  []string{"GetName", "Name_"},
		},
		{
			name:  "duplicate field names",
			input: []string{"Name", "Name"},
			want:  []string{"Name", "Name_"},
		},
		{
			name:  "String reserved",
			input: []string{"String"},
			want:  []string{"String_"},
		},
		{
			name:  "Marshal reserved",
			input: []string{"Marshal", "Unmarshal"},
			want:  []string{"Marshal_", "Unmarshal_"},
		},
		{
			name:  "gcode generated method names",
			input: []string{"Size", "Validate", "ToMap", "MarshalBinary", "MarshalAppend", "UnmarshalBinary", "UnmarshalBinaryLenient"},
			want:  []string{"Size_", "Validate_", "ToMap_", "MarshalBinary_", "MarshalAppend_", "UnmarshalBinary_", "UnmarshalBinaryLenient_"},
		},
		{
			// Chain collision: "Validate" is reserved, so it becomes "Validate_".
			// Then "Validate_" is claimed by the first field, so the second field
			// must keep appending '_' until it finds a free name.
			name:  "chain collision with reserved name",
			input: []string{"Validate", "Validate_"},
			want:  []string{"Validate_", "Validate__"},
		},
		{
			// "GetReset" is processed first (index 0) and passes immediately —
			// neither "GetReset" nor "GetGetReset" is reserved or used.
			// "Reset" is processed second: it is reserved, so it becomes "Reset_".
			// Result: ["GetReset", "Reset_"].
			name:  "getter chain collision with reserved name",
			input: []string{"GetReset", "Reset"},
			want:  []string{"GetReset", "Reset_"},
		},
		{
			// Multi-step getter chain: "GetFoo" (field 0) occupies used["GetFoo"].
			// "Foo" (field 1) fails the loop condition used["GetFoo"]=true, so it
			// escapes to "Foo_", which claims used["Foo_"] and used["GetFoo_"].
			// "Foo_" (field 2) finds used["Foo_"]=true, escapes to "Foo__".
			name:  "getter multi-step chain collision",
			input: []string{"GetFoo", "Foo", "Foo_"},
			want:  []string{"GetFoo", "Foo_", "Foo__"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := ResolveFieldNames(tt.input)
			if len(got) != len(tt.want) {
				t.Fatalf("ResolveFieldNames(%v) returned %d names, want %d", tt.input, len(got), len(tt.want))
			}
			for i := range got {
				if got[i] != tt.want[i] {
					t.Errorf("ResolveFieldNames(%v)[%d] = %q, want %q", tt.input, i, got[i], tt.want[i])
				}
			}
		})
	}
}
