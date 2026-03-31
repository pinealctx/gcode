package naming

import (
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
