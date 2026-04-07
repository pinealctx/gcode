package transform

import (
	"errors"
	"strings"
	"testing"

	"github.com/pinealctx/gcode/internal/model"
)

func TestGoPackageName(t *testing.T) {
	t.Parallel()

	tests := []struct {
		input string
		want  string
	}{
		{"example.com/foo/bar;barpb", "barpb"},
		{"example.com/foo/bar", "bar"},
		{"mypkg", "mypkg"},
		{"", ""},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			got := goPackageName(tt.input)
			if got != tt.want {
				t.Errorf("goPackageName(%q) = %q, want %q", tt.input, got, tt.want)
			}
		})
	}
}

func TestFlattenBasic(t *testing.T) {
	t.Parallel()

	file := model.File{
		Path:      "person.proto",
		Syntax:    model.SyntaxProto3,
		Package:   "test.foo",
		GoPackage: "example.com/test/foo;foopb",
		Messages: []model.Message{
			{
				Name:     "Person",
				FullName: "test.foo.Person",
				Fields: []model.Field{
					{
						Name:        "name",
						Number:      1,
						Cardinality: model.CardinalitySingular,
						Type:        model.FieldType{Kind: model.FieldKindScalar, Scalar: model.ScalarString},
						JSONName:    "name",
					},
					{
						Name:        "age",
						Number:      2,
						Cardinality: model.CardinalitySingular,
						Type:        model.FieldType{Kind: model.FieldKindScalar, Scalar: model.ScalarInt32},
						JSONName:    "age",
					},
				},
			},
		},
	}

	got := Flatten(file)

	if got.Package != "foopb" {
		t.Errorf("Package = %q, want %q", got.Package, "foopb")
	}
	if got.Source != "person.proto" {
		t.Errorf("Source = %q, want %q", got.Source, "person.proto")
	}
	if len(got.Messages) != 1 {
		t.Fatalf("Messages count = %d, want 1", len(got.Messages))
	}

	msg := got.Messages[0]
	if msg.GoName != "Person" {
		t.Errorf("GoName = %q, want %q", msg.GoName, "Person")
	}
	if len(msg.Fields) != 2 {
		t.Fatalf("Fields count = %d, want 2", len(msg.Fields))
	}
	if msg.Fields[0].GoName != "Name" {
		t.Errorf("Fields[0].GoName = %q, want %q", msg.Fields[0].GoName, "Name")
	}
	if msg.Fields[0].GoType != "string" {
		t.Errorf("Fields[0].GoType = %q, want %q", msg.Fields[0].GoType, "string")
	}
	if msg.Fields[1].GoName != "Age" {
		t.Errorf("Fields[1].GoName = %q, want %q", msg.Fields[1].GoName, "Age")
	}
	if msg.Fields[1].GoType != "int32" {
		t.Errorf("Fields[1].GoType = %q, want %q", msg.Fields[1].GoType, "int32")
	}
}

func TestFlattenNested(t *testing.T) {
	t.Parallel()

	file := model.File{
		Path:      "nested.proto",
		Syntax:    model.SyntaxProto3,
		Package:   "test.foo",
		GoPackage: "example.com/test/foo;foopb",
		Messages: []model.Message{
			{
				Name:     "Outer",
				FullName: "test.foo.Outer",
				Fields: []model.Field{
					{
						Name:        "inner",
						Number:      1,
						Cardinality: model.CardinalitySingular,
						Type: model.FieldType{
							Kind:     model.FieldKindMessage,
							Name:     "Inner",
							FullName: "test.foo.Outer.Inner",
						},
						JSONName: "inner",
					},
				},
				Messages: []model.Message{
					{
						Name:     "Inner",
						FullName: "test.foo.Outer.Inner",
						Fields: []model.Field{
							{
								Name:        "value",
								Number:      1,
								Cardinality: model.CardinalitySingular,
								Type:        model.FieldType{Kind: model.FieldKindScalar, Scalar: model.ScalarString},
								JSONName:    "value",
							},
						},
					},
				},
				Enums: []model.Enum{
					{
						Name:     "Status",
						FullName: "test.foo.Outer.Status",
						Values: []model.EnumValue{
							{Name: "STATUS_UNSPECIFIED", Number: 0},
							{Name: "STATUS_ACTIVE", Number: 1},
						},
					},
				},
			},
		},
	}

	got := Flatten(file)

	if len(got.Messages) != 2 {
		t.Fatalf("Messages count = %d, want 2", len(got.Messages))
	}

	// First message is Outer itself.
	if got.Messages[0].GoName != "Outer" {
		t.Errorf("Messages[0].GoName = %q, want %q", got.Messages[0].GoName, "Outer")
	}
	// Outer.inner field should be *Outer_Inner.
	if got.Messages[0].Fields[0].GoType != "*Outer_Inner" {
		t.Errorf("Outer.inner GoType = %q, want %q", got.Messages[0].Fields[0].GoType, "*Outer_Inner")
	}

	// Second message is the flattened Inner.
	if got.Messages[1].GoName != "Outer_Inner" {
		t.Errorf("Messages[1].GoName = %q, want %q", got.Messages[1].GoName, "Outer_Inner")
	}

	// Enum should be flattened.
	if len(got.Enums) != 1 {
		t.Fatalf("Enums count = %d, want 1", len(got.Enums))
	}
	if got.Enums[0].GoName != "Outer_Status" {
		t.Errorf("Enums[0].GoName = %q, want %q", got.Enums[0].GoName, "Outer_Status")
	}
	if got.Enums[0].Values[0].GoName != "Outer_Status_STATUS_UNSPECIFIED" {
		t.Errorf("Values[0].GoName = %q, want %q", got.Enums[0].Values[0].GoName, "Outer_Status_STATUS_UNSPECIFIED")
	}
}

func TestFlattenRepeatedAndEnum(t *testing.T) {
	t.Parallel()

	file := model.File{
		Path:      "types.proto",
		Syntax:    model.SyntaxProto3,
		Package:   "test.foo",
		GoPackage: "example.com/test/foo;foopb",
		Messages: []model.Message{
			{
				Name:     "Container",
				FullName: "test.foo.Container",
				Fields: []model.Field{
					{
						Name:        "tags",
						Number:      1,
						Cardinality: model.CardinalityRepeated,
						Type:        model.FieldType{Kind: model.FieldKindScalar, Scalar: model.ScalarString},
						JSONName:    "tags",
					},
					{
						Name:        "scores",
						Number:      2,
						Cardinality: model.CardinalityRepeated,
						Type:        model.FieldType{Kind: model.FieldKindScalar, Scalar: model.ScalarInt32},
						JSONName:    "scores",
					},
					{
						Name:        "status",
						Number:      3,
						Cardinality: model.CardinalitySingular,
						Type: model.FieldType{
							Kind:     model.FieldKindEnum,
							Name:     "Status",
							FullName: "test.foo.Status",
						},
						JSONName: "status",
					},
					{
						Name:        "items",
						Number:      4,
						Cardinality: model.CardinalityRepeated,
						Type: model.FieldType{
							Kind:     model.FieldKindMessage,
							Name:     "Item",
							FullName: "test.foo.Item",
						},
						JSONName: "items",
					},
					{
						Name:        "statuses",
						Number:      5,
						Cardinality: model.CardinalityRepeated,
						Type: model.FieldType{
							Kind:     model.FieldKindEnum,
							Name:     "Status",
							FullName: "test.foo.Status",
						},
						JSONName: "statuses",
					},
				},
			},
		},
		Enums: []model.Enum{
			{
				Name:     "Status",
				FullName: "test.foo.Status",
				Values: []model.EnumValue{
					{Name: "STATUS_UNSPECIFIED", Number: 0},
				},
			},
		},
	}

	got := Flatten(file)

	msg := got.Messages[0]
	// repeated string → []string
	if msg.Fields[0].GoType != "[]string" {
		t.Errorf("tags GoType = %q, want %q", msg.Fields[0].GoType, "[]string")
	}
	// repeated int32 → []int32
	if msg.Fields[1].GoType != "[]int32" {
		t.Errorf("scores GoType = %q, want %q", msg.Fields[1].GoType, "[]int32")
	}
	// enum → Status
	if msg.Fields[2].GoType != "Status" {
		t.Errorf("status GoType = %q, want %q", msg.Fields[2].GoType, "Status")
	}
	// repeated message → []*Item
	if msg.Fields[3].GoType != "[]*Item" {
		t.Errorf("items GoType = %q, want %q", msg.Fields[3].GoType, "[]*Item")
	}
	// repeated enum → []Status
	if msg.Fields[4].GoType != "[]Status" {
		t.Errorf("statuses GoType = %q, want %q", msg.Fields[4].GoType, "[]Status")
	}
}

func TestFlattenGormAnnotationPassthrough(t *testing.T) {
	t.Parallel()

	file := model.File{
		Path:      "annotated.proto",
		Syntax:    model.SyntaxProto3,
		Package:   "test.foo",
		GoPackage: "example.com/test/foo;foopb",
		Messages: []model.Message{
			{
				Name:        "User",
				FullName:    "test.foo.User",
				GormOptions: &model.GormMessageOptions{Table: "users"},
				Fields: []model.Field{
					{
						Name:        "email",
						Number:      1,
						Cardinality: model.CardinalitySingular,
						Type:        model.FieldType{Kind: model.FieldKindScalar, Scalar: model.ScalarString},
						JSONName:    "email",
						GormOptions: &model.GormFieldOptions{Column: "email_address"},
					},
					{
						Name:        "phone",
						Number:      2,
						Cardinality: model.CardinalitySingular,
						Type:        model.FieldType{Kind: model.FieldKindScalar, Scalar: model.ScalarString},
						JSONName:    "phone",
					},
				},
			},
			{
				Name:     "Config",
				FullName: "test.foo.Config",
				// No GormOptions — GormMessageOptions should be nil on all fields.
				Fields: []model.Field{
					{
						Name:        "key",
						Number:      1,
						Cardinality: model.CardinalitySingular,
						Type:        model.FieldType{Kind: model.FieldKindScalar, Scalar: model.ScalarString},
						JSONName:    "key",
					},
				},
			},
		},
	}

	got := Flatten(file)

	if len(got.Messages) != 2 {
		t.Fatalf("Messages count = %d, want 2", len(got.Messages))
	}

	// --- User: GormMessageOptions should be propagated to all fields ---
	user := got.Messages[0]
	for _, f := range user.Fields {
		if f.GormMessageOptions == nil {
			t.Errorf("User.%s: GormMessageOptions should not be nil", f.GoName)
			continue
		}
		if f.GormMessageOptions.Table != "users" {
			t.Errorf("User.%s: GormMessageOptions.Table = %q, want %q", f.GoName, f.GormMessageOptions.Table, "users")
		}
	}

	// email field-level GormOptions should also be preserved via embedded model.Field
	email := user.Fields[0]
	if email.GormOptions == nil || email.GormOptions.Column != "email_address" {
		t.Errorf("email.GormOptions = %+v, want column=email_address", email.GormOptions)
	}

	// phone has no field-level GormOptions
	phone := user.Fields[1]
	if phone.GormOptions != nil {
		t.Errorf("phone.GormOptions should be nil, got %+v", phone.GormOptions)
	}

	// --- Config: GormMessageOptions should be nil on all fields ---
	config := got.Messages[1]
	for _, f := range config.Fields {
		if f.GormMessageOptions != nil {
			t.Errorf("Config.%s: GormMessageOptions should be nil, got %+v", f.GoName, f.GormMessageOptions)
		}
	}
}

func TestFlattenNestedGormAnnotationPassthrough(t *testing.T) {
	t.Parallel()

	// Nested messages should each carry their own GormMessageOptions independently.
	file := model.File{
		Path:      "nested_annotated.proto",
		Syntax:    model.SyntaxProto3,
		Package:   "test.foo",
		GoPackage: "example.com/test/foo;foopb",
		Messages: []model.Message{
			{
				Name:        "Outer",
				FullName:    "test.foo.Outer",
				GormOptions: &model.GormMessageOptions{Table: "outers"},
				Fields: []model.Field{
					{
						Name:        "value",
						Number:      1,
						Cardinality: model.CardinalitySingular,
						Type:        model.FieldType{Kind: model.FieldKindScalar, Scalar: model.ScalarString},
						JSONName:    "value",
					},
				},
				Messages: []model.Message{
					{
						Name:     "Inner",
						FullName: "test.foo.Outer.Inner",
						// Inner has no GormOptions.
						Fields: []model.Field{
							{
								Name:        "id",
								Number:      1,
								Cardinality: model.CardinalitySingular,
								Type:        model.FieldType{Kind: model.FieldKindScalar, Scalar: model.ScalarInt32},
								JSONName:    "id",
							},
						},
					},
				},
			},
		},
	}

	got := Flatten(file)

	if len(got.Messages) != 2 {
		t.Fatalf("Messages count = %d, want 2", len(got.Messages))
	}

	// Outer fields should carry Outer's GormMessageOptions.
	outer := got.Messages[0]
	if outer.Fields[0].GormMessageOptions == nil || outer.Fields[0].GormMessageOptions.Table != "outers" {
		t.Errorf("Outer.value GormMessageOptions = %+v, want table=outers", outer.Fields[0].GormMessageOptions)
	}

	// Inner fields should have nil GormMessageOptions (Inner has no gorm annotation).
	inner := got.Messages[1]
	if inner.Fields[0].GormMessageOptions != nil {
		t.Errorf("Inner.id GormMessageOptions should be nil, got %+v", inner.Fields[0].GormMessageOptions)
	}
}

func TestFlattenOptionalGoType(t *testing.T) {
	t.Parallel()

	statusEnum := model.FieldType{Kind: model.FieldKindEnum, Name: "Status", FullName: "test.foo.Status"}

	cases := []struct {
		name     string
		field    model.Field
		wantType string
	}{
		{
			name: "optional string → *string",
			field: model.Field{
				Name: "name", Number: 1, Cardinality: model.CardinalitySingular,
				Optional: true,
				Type:     model.FieldType{Kind: model.FieldKindScalar, Scalar: model.ScalarString},
			},
			wantType: "*string",
		},
		{
			name: "optional int32 → *int32",
			field: model.Field{
				Name: "count", Number: 2, Cardinality: model.CardinalitySingular,
				Optional: true,
				Type:     model.FieldType{Kind: model.FieldKindScalar, Scalar: model.ScalarInt32},
			},
			wantType: "*int32",
		},
		{
			name: "optional bool → *bool",
			field: model.Field{
				Name: "active", Number: 3, Cardinality: model.CardinalitySingular,
				Optional: true,
				Type:     model.FieldType{Kind: model.FieldKindScalar, Scalar: model.ScalarBool},
			},
			wantType: "*bool",
		},
		{
			name: "optional enum → *Status",
			field: model.Field{
				Name: "status", Number: 4, Cardinality: model.CardinalitySingular,
				Optional: true,
				Type:     statusEnum,
			},
			wantType: "*Status",
		},
		{
			name: "optional bytes → *[]byte",
			field: model.Field{
				Name: "data", Number: 5, Cardinality: model.CardinalitySingular,
				Optional: true,
				Type:     model.FieldType{Kind: model.FieldKindScalar, Scalar: model.ScalarBytes},
			},
			wantType: "*[]byte",
		},
		{
			name: "plain string → string (backward compat)",
			field: model.Field{
				Name: "label", Number: 6, Cardinality: model.CardinalitySingular,
				Optional: false,
				Type:     model.FieldType{Kind: model.FieldKindScalar, Scalar: model.ScalarString},
			},
			wantType: "string",
		},
		{
			name: "repeated string → []string (optional ignored for repeated)",
			field: model.Field{
				Name: "tags", Number: 7, Cardinality: model.CardinalityRepeated,
				Optional: false,
				Type:     model.FieldType{Kind: model.FieldKindScalar, Scalar: model.ScalarString},
			},
			wantType: "[]string",
		},
		{
			// proto3 message fields are always pointers; Optional is always false
			// for message kinds (parser excludes MessageKind from optional detection).
			// This case documents that plain message fields produce *T, not **T.
			name: "plain message → *Address",
			field: model.Field{
				Name: "address", Number: 8, Cardinality: model.CardinalitySingular,
				Optional: false,
				Type:     model.FieldType{Kind: model.FieldKindMessage, FullName: "test.foo.Address"},
			},
			wantType: "*Address",
		},
		{
			// If Optional were somehow true for a message field (which the parser
			// never produces), resolveGoType must still return *T, not **T.
			// This guards against a future "fix" that would incorrectly add another
			// pointer layer for optional message fields.
			name: "optional=true message → *Address (not **Address)",
			field: model.Field{
				Name: "address", Number: 9, Cardinality: model.CardinalitySingular,
				Optional: true,
				Type:     model.FieldType{Kind: model.FieldKindMessage, FullName: "test.foo.Address"},
			},
			wantType: "*Address",
		},
		{
			// repeated message fields produce []*T; this verifies the repeated
			// branch inside the FieldKindMessage case is not broken by the optional fix.
			name: "repeated message → []*Address",
			field: model.Field{
				Name: "addresses", Number: 10, Cardinality: model.CardinalityRepeated,
				Optional: false,
				Type:     model.FieldType{Kind: model.FieldKindMessage, FullName: "test.foo.Address"},
			},
			wantType: "[]*Address",
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := resolveGoType(tc.field, "test.foo")
			if got != tc.wantType {
				t.Errorf("resolveGoType = %q, want %q", got, tc.wantType)
			}
		})
	}
}

// TestValidateCreateOptions_Errors verifies that required_fields referencing a non-optional
// or non-existent field is rejected by the transform layer.
func TestValidateCreateOptions_Errors(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name  string
		files []model.File
	}{
		{
			name: "required_fields non-optional field",
			files: []model.File{{
				Messages: []model.Message{{
					FullName: "test.M",
					Fields: []model.Field{
						{Name: "name", Optional: false},
					},
					CreateOptions: []model.CreateMessageOptions{{
						Name:           "MCreate",
						RequiredFields: []string{"name"},
					}},
				}},
			}},
		},
		{
			name: "required_fields unknown field",
			files: []model.File{{
				Messages: []model.Message{{
					FullName: "test.M",
					Fields: []model.Field{
						{Name: "name", Optional: true},
					},
					CreateOptions: []model.CreateMessageOptions{{
						Name:           "MCreate",
						RequiredFields: []string{"nonexistent"},
					}},
				}},
			}},
		},
		{
			name: "required_fields in nested message",
			files: []model.File{{
				Messages: []model.Message{{
					FullName: "test.Outer",
					Fields:   []model.Field{{Name: "x", Optional: false}},
					Messages: []model.Message{{
						FullName: "test.Outer.Inner",
						Fields:   []model.Field{{Name: "val", Optional: false}},
						CreateOptions: []model.CreateMessageOptions{{
							Name:           "InnerCreate",
							RequiredFields: []string{"val"},
						}},
					}},
				}},
			}},
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			err := ValidateCreateOptions(tc.files)
			if err == nil {
				t.Fatal("expected TransformError, got nil")
			}
			var te TransformError
			if !errors.As(err, &te) {
				t.Errorf("expected TransformError domain type, got %T: %v", err, err)
			}
		})
	}
}

// TestValidateCreateOptions_Valid verifies that valid create options pass without error.
func TestValidateCreateOptions_Valid(t *testing.T) {
	t.Parallel()

	files := []model.File{{
		Messages: []model.Message{{
			FullName: "test.User",
			Fields: []model.Field{
				{Name: "id", Optional: false},
				{Name: "email", Optional: true},
				{Name: "name", Optional: true},
			},
			CreateOptions: []model.CreateMessageOptions{{
				Name:           "UserCreate",
				IgnoreFields:   []string{"id"},
				RequiredFields: []string{"email"},
			}},
		}},
	}}

	if err := ValidateCreateOptions(files); err != nil {
		t.Errorf("unexpected error: %v", err)
	}
}

// TestValidateCreateOptions_NoOptions verifies that messages without create options pass.
func TestValidateCreateOptions_NoOptions(t *testing.T) {
	t.Parallel()

	files := []model.File{{
		Messages: []model.Message{{
			FullName: "test.Plain",
			Fields:   []model.Field{{Name: "name", Optional: false}},
		}},
	}}

	if err := ValidateCreateOptions(files); err != nil {
		t.Errorf("unexpected error: %v", err)
	}
}

func TestFlattenServices(t *testing.T) {
	t.Parallel()

	file := model.File{
		Path:      "user_service.proto",
		Package:   "example.svc",
		GoPackage: "example.com/svc;svc",
		Services: []model.Service{
			{
				Name:     "UserService",
				FullName: "example.svc.UserService",
				RPCs: []model.RPC{
					{
						Name:         "CreateUser",
						RequestType:  "example.svc.CreateUserRequest",
						ResponseType: "example.svc.CreateUserResponse",
					},
					{
						Name:         "GetUser",
						RequestType:  "example.svc.GetUserRequest",
						ResponseType: "example.svc.GetUserResponse",
					},
				},
			},
		},
	}

	gf := Flatten(file)

	if len(gf.Services) != 1 {
		t.Fatalf("expected 1 service, got %d", len(gf.Services))
	}
	svc := gf.Services[0]
	if svc.GoName != "UserService" {
		t.Errorf("GoName = %q, want %q", svc.GoName, "UserService")
	}
	if len(svc.Methods) != 2 {
		t.Fatalf("expected 2 methods, got %d", len(svc.Methods))
	}

	m0 := svc.Methods[0]
	if m0.GoName != "CreateUser" {
		t.Errorf("method[0].GoName = %q, want %q", m0.GoName, "CreateUser")
	}
	if m0.RequestType != "CreateUserRequest" {
		t.Errorf("method[0].RequestType = %q, want %q", m0.RequestType, "CreateUserRequest")
	}
	if m0.ResponseType != "CreateUserResponse" {
		t.Errorf("method[0].ResponseType = %q, want %q", m0.ResponseType, "CreateUserResponse")
	}

	m1 := svc.Methods[1]
	if m1.GoName != "GetUser" {
		t.Errorf("method[1].GoName = %q, want %q", m1.GoName, "GetUser")
	}
	if m1.RequestType != "GetUserRequest" {
		t.Errorf("method[1].RequestType = %q, want %q", m1.RequestType, "GetUserRequest")
	}
	if m1.ResponseType != "GetUserResponse" {
		t.Errorf("method[1].ResponseType = %q, want %q", m1.ResponseType, "GetUserResponse")
	}
}

func TestFlattenServicesMultiple(t *testing.T) {
	t.Parallel()

	file := model.File{
		Package:   "example.multi",
		GoPackage: "example.com/multi;multi",
		Services: []model.Service{
			{FullName: "example.multi.ServiceA", RPCs: []model.RPC{{Name: "MethodA", RequestType: "example.multi.AReq", ResponseType: "example.multi.AResp"}}},
			{FullName: "example.multi.ServiceB", RPCs: []model.RPC{{Name: "MethodB1", RequestType: "example.multi.BReq", ResponseType: "example.multi.BResp"}, {Name: "MethodB2", RequestType: "example.multi.BReq", ResponseType: "example.multi.BResp"}}},
		},
	}

	gf := Flatten(file)

	if len(gf.Services) != 2 {
		t.Fatalf("expected 2 services, got %d", len(gf.Services))
	}
	if len(gf.Services[1].Methods) != 2 {
		t.Errorf("ServiceB expected 2 methods, got %d", len(gf.Services[1].Methods))
	}
}

func TestFlattenNoServicesEmpty(t *testing.T) {
	t.Parallel()

	file := model.File{
		Package:   "example.msg",
		GoPackage: "example.com/msg;msg",
		Messages: []model.Message{
			{Name: "Foo", FullName: "example.msg.Foo"},
		},
	}

	gf := Flatten(file)

	if len(gf.Services) != 0 {
		t.Errorf("expected no services, got %d", len(gf.Services))
	}
}

func TestFlattenServiceTypeNameStripsPackage(t *testing.T) {
	t.Parallel()

	// Request/response types from the same package should have the package prefix stripped.
	file := model.File{
		Package:   "example.svc",
		GoPackage: "example.com/svc;svc",
		Services: []model.Service{
			{
				FullName: "example.svc.MyService",
				RPCs: []model.RPC{
					{
						Name:         "Do",
						RequestType:  "example.svc.DoRequest",
						ResponseType: "example.svc.DoResponse",
					},
				},
			},
		},
	}

	gf := Flatten(file)
	m := gf.Services[0].Methods[0]

	// Same-package types have the package prefix stripped by GoTypeName.
	if m.RequestType != "DoRequest" {
		t.Errorf("same-package type should strip prefix, got %q", m.RequestType)
	}
	if m.ResponseType != "DoResponse" {
		t.Errorf("same-package type should strip prefix, got %q", m.ResponseType)
	}
}

// TestFlattenCreateMessageGormOptionsNotInherited verifies that Flatten does NOT
// inherit GormMessageOptions across files — that is the render layer's responsibility
// via Context.MessageIndex. Within a single file, a create derived message that has
// no gorm annotation of its own will have GormMessageOptions == nil after Flatten.
func TestFlattenCreateMessageGormOptionsNotInherited(t *testing.T) {
	t.Parallel()

	file := model.File{
		Path:      "person.create.proto",
		Package:   "compat",
		GoPackage: "example.com/dao;dao",
		Messages: []model.Message{
			{
				Name:         "PersonCreate",
				FullName:     "compat.PersonCreate",
				CreateSource: "Person",
				Fields: []model.Field{
					{Name: "name", Number: 1, Cardinality: model.CardinalitySingular, Type: model.FieldType{Kind: model.FieldKindScalar, Scalar: model.ScalarString}, JSONName: "name"},
				},
			},
		},
	}

	gf := Flatten(file)

	if len(gf.Messages) == 0 {
		t.Fatal("expected at least one message after Flatten")
	}
	msg := gf.Messages[0]
	if msg.GormMessageOptions != nil {
		t.Error("PersonCreate.GormMessageOptions must be nil before render-layer inheritance")
	}
}

// TestResolveGoType_UnknownKindPanics verifies that resolveGoType panics with a
// descriptive message when it encounters an unknown FieldKind. This is a
// programming-error guard: the parser only produces known kinds, so this branch
// should never be reached in production.
func TestResolveGoType_UnknownKindPanics(t *testing.T) {
	t.Parallel()

	field := model.Field{
		Name:        "bad_field",
		Number:      1,
		Cardinality: model.CardinalitySingular,
		Type:        model.FieldType{Kind: model.FieldKind("nonexistent")},
	}
	file := model.File{
		Path:      "test.proto",
		Syntax:    model.SyntaxProto3,
		Package:   "test",
		GoPackage: "example.com/test;testpb",
		Messages: []model.Message{
			{
				Name:     "Bad",
				FullName: "test.Bad",
				Fields:   []model.Field{field},
			},
		},
	}

	defer func() {
		r := recover()
		if r == nil {
			t.Fatal("expected panic for unknown FieldKind, got none")
		}
		msg, ok := r.(string)
		if !ok {
			t.Fatalf("expected panic value to be string, got %T: %v", r, r)
		}
		if !strings.Contains(msg, "resolveGoType") || !strings.Contains(msg, "unexpected FieldKind") {
			t.Errorf("panic message %q does not contain expected text", msg)
		}
		if !strings.Contains(msg, "bad_field") {
			t.Errorf("panic message %q should contain field name %q", msg, "bad_field")
		}
	}()
	Flatten(file)
}
