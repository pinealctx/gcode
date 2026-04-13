package render

import (
	"strings"
	"testing"

	"github.com/pinealctx/gcode/internal/model"
	"github.com/pinealctx/gcode/internal/transform"
)

const testModulePhase4 = "github.com/pinealctx/gcode"

// buildUpdateGoMessage builds a GoMessage simulating an update message generated
// by gen-proto: id is non-optional (condition field), name/email are optional.
func buildUpdateGoMessage() transform.GoMessage {
	return transform.GoMessage{
		GoName:          "UserUpdateByID",
		UpdateSource:    "User",
		ConditionFields: []string{"id"},
		Fields: []transform.GoField{
			{
				Field:  model.Field{Name: "id", Number: 1, Cardinality: model.CardinalitySingular, Type: model.FieldType{Kind: model.FieldKindScalar, Scalar: model.ScalarInt64}, JSONName: "id"},
				GoName: "Id",
				GoType: "int64",
			},
			{
				Field:  model.Field{Name: "name", Number: 2, Cardinality: model.CardinalitySingular, Optional: true, Type: model.FieldType{Kind: model.FieldKindScalar, Scalar: model.ScalarString}, JSONName: "name"},
				GoName: "Name",
				GoType: "*string",
			},
			{
				Field:  model.Field{Name: "email", Number: 3, Cardinality: model.CardinalitySingular, Optional: true, Type: model.FieldType{Kind: model.FieldKindScalar, Scalar: model.ScalarString}, JSONName: "email"},
				GoName: "Email",
				GoType: "*string",
			},
		},
	}
}

// buildSourceGoMessage builds the original User GoMessage with validate rules.
func buildSourceGoMessage() transform.GoMessage {
	minLen1 := uint64(1)
	maxLen100 := uint64(100)
	return transform.GoMessage{
		GoName: "User",
		Fields: []transform.GoField{
			{
				Field:  model.Field{Name: "id", Number: 1, Cardinality: model.CardinalitySingular, Type: model.FieldType{Kind: model.FieldKindScalar, Scalar: model.ScalarInt64}, JSONName: "id"},
				GoName: "Id",
				GoType: "int64",
			},
			{
				Field: model.Field{
					Name: "name", Number: 2, Cardinality: model.CardinalitySingular,
					Type:            model.FieldType{Kind: model.FieldKindScalar, Scalar: model.ScalarString},
					JSONName:        "name",
					ValidateOptions: &model.ValidateFieldOptions{MinLen: &minLen1, MaxLen: &maxLen100},
				},
				GoName: "Name",
				GoType: "string",
			},
			{
				Field: model.Field{
					Name: "email", Number: 3, Cardinality: model.CardinalitySingular,
					Type:            model.FieldType{Kind: model.FieldKindScalar, Scalar: model.ScalarString},
					JSONName:        "email",
					ValidateOptions: &model.ValidateFieldOptions{Email: true},
				},
				GoName: "Email",
				GoType: "string",
			},
		},
	}
}

func TestToMapGeneration(t *testing.T) {
	t.Parallel()

	updateMsg := buildUpdateGoMessage()
	srcMsg := buildSourceGoMessage()
	msgIndex := map[string]*transform.GoMessage{
		"User": &srcMsg,
	}
	ctx := Context{MessageIndex: msgIndex}

	gf := transform.GoFile{
		Source:   "user.update.proto",
		Package:  "testpkg",
		Messages: []transform.GoMessage{updateMsg},
	}

	src, err := File(gf, testModulePhase4, ctx)
	if err != nil {
		t.Fatalf("File() error: %v", err)
	}
	s := string(src)

	// ToMap method should be generated.
	if !strings.Contains(s, "func (x *UserUpdateByID) ToMap()") {
		t.Errorf("missing ToMap method in:\n%s", s)
	}
	// condition field (id, non-optional) should NOT be in map.
	if strings.Contains(s, `um["id"]`) {
		t.Errorf("condition field 'id' should not be in ToMap in:\n%s", s)
	}
	// optional fields should be in map with nil check.
	if !strings.Contains(s, `um["name"] = *`) {
		t.Errorf("optional field 'name' should be in ToMap with deref in:\n%s", s)
	}
	if !strings.Contains(s, `um["email"] = *`) {
		t.Errorf("optional field 'email' should be in ToMap with deref in:\n%s", s)
	}
	// nil check for optional fields.
	if !strings.Contains(s, "if x.Name != nil") {
		t.Errorf("missing nil check for Name in:\n%s", s)
	}
}

func TestToMapNotGeneratedForNonUpdate(t *testing.T) {
	t.Parallel()

	// A regular message (no UpdateSource) should not get ToMap.
	msg := transform.GoMessage{
		GoName: "User",
		Fields: []transform.GoField{
			{
				Field:  model.Field{Name: "id", Type: model.FieldType{Kind: model.FieldKindScalar, Scalar: model.ScalarInt64}, JSONName: "id"},
				GoName: "Id", GoType: "int64",
			},
		},
	}
	gf := transform.GoFile{
		Source:   "user.proto",
		Package:  "testpkg",
		Messages: []transform.GoMessage{msg},
	}

	src, err := File(gf, testModulePhase4, Context{})
	if err != nil {
		t.Fatalf("File() error: %v", err)
	}
	if strings.Contains(string(src), "ToMap") {
		t.Errorf("ToMap should not be generated for non-update message")
	}
}

func TestValidateInheritance_UpdateMessage(t *testing.T) {
	t.Parallel()

	updateMsg := buildUpdateGoMessage()
	srcMsg := buildSourceGoMessage()
	msgIndex := map[string]*transform.GoMessage{
		"User": &srcMsg,
	}
	ctx := Context{MessageIndex: msgIndex}

	gf := transform.GoFile{
		Source:   "user.update.proto",
		Package:  "testpkg",
		Messages: []transform.GoMessage{updateMsg},
	}

	src, err := ValidateFile(gf, testModulePhase4, ctx)
	if err != nil {
		t.Fatalf("ValidateFile() error: %v", err)
	}
	s := string(src)

	// Should generate Validate() for UserUpdateByID.
	if !strings.Contains(s, "func (x *UserUpdateByID) Validate()") {
		t.Errorf("missing Validate method in:\n%s", s)
	}
	// Optional fields should have nil check before validation.
	if !strings.Contains(s, "if x.Name != nil") {
		t.Errorf("missing nil check for optional Name field in:\n%s", s)
	}
	if !strings.Contains(s, "if x.Email != nil") {
		t.Errorf("missing nil check for optional Email field in:\n%s", s)
	}
	// Validate rules from source should be inherited (email check).
	if !strings.Contains(s, "IsEmail") {
		t.Errorf("email validate rule should be inherited in:\n%s", s)
	}
	// min_len rule for name.
	if !strings.Contains(s, "min_len") {
		t.Errorf("min_len validate rule should be inherited in:\n%s", s)
	}
}

func TestValidateInheritance_CreateMessage(t *testing.T) {
	t.Parallel()

	minLen1 := uint64(1)
	srcMsg := transform.GoMessage{
		GoName: "Product",
		Fields: []transform.GoField{
			{
				Field: model.Field{
					Name: "sku", Type: model.FieldType{Kind: model.FieldKindScalar, Scalar: model.ScalarString},
					JSONName:        "sku",
					ValidateOptions: &model.ValidateFieldOptions{MinLen: &minLen1},
				},
				GoName: "Sku", GoType: "string",
			},
		},
	}
	// Create message: sku is required (non-optional), title is optional.
	createMsg := transform.GoMessage{
		GoName:       "ProductCreate",
		CreateSource: "Product",
		Fields: []transform.GoField{
			{
				Field:  model.Field{Name: "sku", Type: model.FieldType{Kind: model.FieldKindScalar, Scalar: model.ScalarString}, JSONName: "sku"},
				GoName: "Sku", GoType: "string", // non-optional (required_fields)
			},
		},
	}
	msgIndex := map[string]*transform.GoMessage{"Product": &srcMsg}
	ctx := Context{MessageIndex: msgIndex}

	gf := transform.GoFile{
		Source:   "product.create.proto",
		Package:  "testpkg",
		Messages: []transform.GoMessage{createMsg},
	}

	src, err := ValidateFile(gf, testModulePhase4, ctx)
	if err != nil {
		t.Fatalf("ValidateFile() error: %v", err)
	}
	s := string(src)

	// Non-optional field should be validated directly (no nil check).
	if strings.Contains(s, "if x.Sku != nil") {
		t.Errorf("non-optional field should not have nil check in:\n%s", s)
	}
	if !strings.Contains(s, "min_len") {
		t.Errorf("min_len rule should be inherited for non-optional field in:\n%s", s)
	}
}

func TestValidateInheritance_NoSourceInIndex(t *testing.T) {
	t.Parallel()

	// When source message is not in index, fall back to own validate rules.
	updateMsg := transform.GoMessage{
		GoName:       "UserUpdateByID",
		UpdateSource: "User",
		Fields: []transform.GoField{
			{
				Field:  model.Field{Name: "id", Type: model.FieldType{Kind: model.FieldKindScalar, Scalar: model.ScalarInt64}, JSONName: "id"},
				GoName: "Id", GoType: "int64",
			},
		},
	}
	// Empty index — source not found.
	ctx := Context{MessageIndex: map[string]*transform.GoMessage{}}

	gf := transform.GoFile{
		Source:   "user.update.proto",
		Package:  "testpkg",
		Messages: []transform.GoMessage{updateMsg},
	}

	src, err := ValidateFile(gf, testModulePhase4, ctx)
	if err != nil {
		t.Fatalf("ValidateFile() error: %v", err)
	}
	// Should still generate a Validate() method (empty body).
	if !strings.Contains(string(src), "func (x *UserUpdateByID) Validate()") {
		t.Errorf("Validate method should still be generated when source not in index")
	}
}

func TestValidateInheritance_NilContext(t *testing.T) {
	t.Parallel()

	// Zero-value Context should not panic.
	updateMsg := transform.GoMessage{
		GoName:       "UserUpdateByID",
		UpdateSource: "User",
		Fields: []transform.GoField{
			{
				Field:  model.Field{Name: "id", Type: model.FieldType{Kind: model.FieldKindScalar, Scalar: model.ScalarInt64}, JSONName: "id"},
				GoName: "Id", GoType: "int64",
			},
		},
	}
	gf := transform.GoFile{
		Source:   "user.update.proto",
		Package:  "testpkg",
		Messages: []transform.GoMessage{updateMsg},
	}

	_, err := ValidateFile(gf, testModulePhase4, Context{})
	if err != nil {
		t.Fatalf("ValidateFile() with zero Context should not error: %v", err)
	}
}

func TestGlobalIndexBuiltInApp(t *testing.T) {
	t.Parallel()

	// Verify that GoMessage.UpdateSource/CreateSource are propagated through Flatten.
	msg := model.Message{
		Name:         "UserUpdateByID",
		FullName:     "test.UserUpdateByID",
		UpdateSource: "User",
		Fields: []model.Field{
			{Name: "id", Number: 1, Cardinality: model.CardinalitySingular, Type: model.FieldType{Kind: model.FieldKindScalar, Scalar: model.ScalarInt64}, JSONName: "id"},
		},
	}
	file := model.File{
		Path:     "user.update.proto",
		Package:  "test",
		Messages: []model.Message{msg},
	}

	gf := transform.Flatten(file)
	if len(gf.Messages) != 1 {
		t.Fatalf("expected 1 message, got %d", len(gf.Messages))
	}
	if gf.Messages[0].UpdateSource != "User" {
		t.Errorf("UpdateSource = %q, want User", gf.Messages[0].UpdateSource)
	}
	if gf.Messages[0].CreateSource != "" {
		t.Errorf("CreateSource should be empty, got %q", gf.Messages[0].CreateSource)
	}
}

func TestGlobalIndexBuiltInApp_CreateSource(t *testing.T) {
	t.Parallel()

	// Verify CreateSource is also propagated through Flatten.
	msg := model.Message{
		Name:         "ProductCreate",
		FullName:     "test.ProductCreate",
		CreateSource: "Product",
		Fields: []model.Field{
			{Name: "title", Number: 1, Cardinality: model.CardinalitySingular, Type: model.FieldType{Kind: model.FieldKindScalar, Scalar: model.ScalarString}, JSONName: "title"},
		},
	}
	file := model.File{
		Path:     "product.create.proto",
		Package:  "test",
		Messages: []model.Message{msg},
	}

	gf := transform.Flatten(file)
	if len(gf.Messages) != 1 {
		t.Fatalf("expected 1 message, got %d", len(gf.Messages))
	}
	if gf.Messages[0].CreateSource != "Product" {
		t.Errorf("CreateSource = %q, want Product", gf.Messages[0].CreateSource)
	}
	if gf.Messages[0].UpdateSource != "" {
		t.Errorf("UpdateSource should be empty, got %q", gf.Messages[0].UpdateSource)
	}
}

func TestToMapRepeatedField(t *testing.T) {
	t.Parallel()

	updateMsg := transform.GoMessage{
		GoName:          "ItemUpdate",
		UpdateSource:    "Item",
		ConditionFields: []string{"id"},
		Fields: []transform.GoField{
			{
				Field:  model.Field{Name: "id", Cardinality: model.CardinalitySingular, Type: model.FieldType{Kind: model.FieldKindScalar, Scalar: model.ScalarInt64}, JSONName: "id"},
				GoName: "Id", GoType: "int64",
			},
			{
				Field:  model.Field{Name: "tags", Cardinality: model.CardinalityRepeated, Type: model.FieldType{Kind: model.FieldKindScalar, Scalar: model.ScalarString}, JSONName: "tags"},
				GoName: "Tags", GoType: "[]string",
			},
		},
	}
	gf := transform.GoFile{
		Source:   "item.update.proto",
		Package:  "testpkg",
		Messages: []transform.GoMessage{updateMsg},
	}

	src, err := File(gf, testModulePhase4, Context{})
	if err != nil {
		t.Fatalf("File() error: %v", err)
	}
	s := string(src)

	// repeated field should use len() check.
	if !strings.Contains(s, "len(x.Tags) > 0") {
		t.Errorf("repeated field should use len() check in:\n%s", s)
	}
	if !strings.Contains(s, `um["tags"] = x.Tags`) {
		t.Errorf("repeated field should be written to map in:\n%s", s)
	}
	// condition field should not be in map.
	if strings.Contains(s, `um["id"]`) {
		t.Errorf("condition field id should not be in map in:\n%s", s)
	}
}

func TestToMapNoConditionFields(t *testing.T) {
	t.Parallel()

	// Update message with no condition fields — all fields go into map.
	updateMsg := transform.GoMessage{
		GoName:          "ItemUpdate",
		UpdateSource:    "Item",
		ConditionFields: nil,
		Fields: []transform.GoField{
			{
				Field:  model.Field{Name: "name", Cardinality: model.CardinalitySingular, Optional: true, Type: model.FieldType{Kind: model.FieldKindScalar, Scalar: model.ScalarString}, JSONName: "name"},
				GoName: "Name", GoType: "*string",
			},
		},
	}
	gf := transform.GoFile{
		Source:   "item.update.proto",
		Package:  "testpkg",
		Messages: []transform.GoMessage{updateMsg},
	}

	src, err := File(gf, testModulePhase4, Context{})
	if err != nil {
		t.Fatalf("File() error: %v", err)
	}
	if !strings.Contains(string(src), `um["name"] = *`) {
		t.Errorf("name should be in map when no condition fields")
	}
}

func TestToEntityGeneration(t *testing.T) {
	t.Parallel()

	// Source: User with id (int64), name (string), nickname (*string), tags ([]string).
	srcMsg := transform.GoMessage{
		GoName: "User",
		Fields: []transform.GoField{
			{Field: model.Field{Name: "id", Type: model.FieldType{Kind: model.FieldKindScalar, Scalar: model.ScalarInt64}}, GoName: "Id", GoType: "int64"},
			{Field: model.Field{Name: "name", Type: model.FieldType{Kind: model.FieldKindScalar, Scalar: model.ScalarString}}, GoName: "Name", GoType: "string"},
			{Field: model.Field{Name: "nickname", Type: model.FieldType{Kind: model.FieldKindScalar, Scalar: model.ScalarString}}, GoName: "Nickname", GoType: "*string"},
			{Field: model.Field{Name: "tags", Cardinality: model.CardinalityRepeated, Type: model.FieldType{Kind: model.FieldKindScalar, Scalar: model.ScalarString}}, GoName: "Tags", GoType: "[]string"},
		},
	}
	// Create: id is required (non-pointer), name/nickname are optional (*T), tags is repeated.
	createMsg := transform.GoMessage{
		GoName:       "UserCreate",
		CreateSource: "User",
		Fields: []transform.GoField{
			{Field: model.Field{Name: "id", Type: model.FieldType{Kind: model.FieldKindScalar, Scalar: model.ScalarInt64}}, GoName: "Id", GoType: "int64"},
			{Field: model.Field{Name: "name", Type: model.FieldType{Kind: model.FieldKindScalar, Scalar: model.ScalarString}}, GoName: "Name", GoType: "*string"},
			{Field: model.Field{Name: "nickname", Type: model.FieldType{Kind: model.FieldKindScalar, Scalar: model.ScalarString}}, GoName: "Nickname", GoType: "*string"},
			{Field: model.Field{Name: "tags", Cardinality: model.CardinalityRepeated, Type: model.FieldType{Kind: model.FieldKindScalar, Scalar: model.ScalarString}}, GoName: "Tags", GoType: "[]string"},
		},
	}
	msgIndex := map[string]*transform.GoMessage{"User": &srcMsg}
	ctx := Context{MessageIndex: msgIndex}

	gf := transform.GoFile{
		Source:   "user.create.proto",
		Package:  "testpkg",
		Messages: []transform.GoMessage{createMsg},
	}

	src, err := File(gf, testModulePhase4, ctx)
	if err != nil {
		t.Fatalf("File() error: %v", err)
	}
	s := string(src)

	// ToEntity method should exist.
	if !strings.Contains(s, "func (x *UserCreate) ToEntity() User {") {
		t.Errorf("missing ToEntity method in:\n%s", s)
	}
	// Required field (id): non-ptr→non-ptr, direct assign.
	if !strings.Contains(s, "p.Id = x.Id\n") {
		t.Errorf("required field id should be direct assign in:\n%s", s)
	}
	// Optional field (name): ptr→non-ptr, nil-guard + deref.
	if !strings.Contains(s, "p.Name = *x.Name") {
		t.Errorf("optional field name should have deref assign in:\n%s", s)
	}
	if !strings.Contains(s, "if x.Name != nil") {
		t.Errorf("optional field name should have nil-guard in:\n%s", s)
	}
	// Optional ptr→ptr (nickname): nil-guard + pointer assign (no deref).
	if !strings.Contains(s, "p.Nickname = x.Nickname") {
		t.Errorf("optional ptr→ptr field nickname should have pointer assign in:\n%s", s)
	}
	if strings.Contains(s, "p.Nickname = *x.Nickname") {
		t.Errorf("optional ptr→ptr field nickname should NOT deref in:\n%s", s)
	}
	// Repeated field (tags): direct assign.
	if !strings.Contains(s, "p.Tags = x.Tags\n") {
		t.Errorf("repeated field tags should be direct assign in:\n%s", s)
	}
}

func TestToEntityRequiredToPointer(t *testing.T) {
	t.Parallel()

	// Edge case: required non-pointer in create → pointer in source.
	srcMsg := transform.GoMessage{
		GoName: "Item",
		Fields: []transform.GoField{
			{Field: model.Field{Name: "label", Type: model.FieldType{Kind: model.FieldKindScalar, Scalar: model.ScalarString}}, GoName: "Label", GoType: "*string"},
		},
	}
	createMsg := transform.GoMessage{
		GoName:       "ItemCreate",
		CreateSource: "Item",
		Fields: []transform.GoField{
			{Field: model.Field{Name: "label", Type: model.FieldType{Kind: model.FieldKindScalar, Scalar: model.ScalarString}}, GoName: "Label", GoType: "string"},
		},
	}
	msgIndex := map[string]*transform.GoMessage{"Item": &srcMsg}
	ctx := Context{MessageIndex: msgIndex}

	gf := transform.GoFile{Source: "item.create.proto", Package: "testpkg", Messages: []transform.GoMessage{createMsg}}
	src, err := File(gf, testModulePhase4, ctx)
	if err != nil {
		t.Fatalf("File() error: %v", err)
	}
	s := string(src)

	// Non-ptr→ptr: take address.
	if !strings.Contains(s, "p.Label = &x.Label\n") {
		t.Errorf("required non-ptr to ptr should take address in:\n%s", s)
	}
}

func TestToEntityNotGeneratedForNonCreate(t *testing.T) {
	t.Parallel()

	msg := transform.GoMessage{
		GoName: "User",
		Fields: []transform.GoField{
			{Field: model.Field{Name: "id", Type: model.FieldType{Kind: model.FieldKindScalar, Scalar: model.ScalarInt64}}, GoName: "Id", GoType: "int64"},
		},
	}
	gf := transform.GoFile{Source: "user.proto", Package: "testpkg", Messages: []transform.GoMessage{msg}}

	src, err := File(gf, testModulePhase4, Context{})
	if err != nil {
		t.Fatalf("File() error: %v", err)
	}
	if strings.Contains(string(src), "ToEntity") {
		t.Errorf("ToEntity should not be generated for non-create message")
	}
}

func TestApplyToGeneration(t *testing.T) {
	t.Parallel()

	srcMsg := transform.GoMessage{
		GoName: "User",
		Fields: []transform.GoField{
			{Field: model.Field{Name: "id", Type: model.FieldType{Kind: model.FieldKindScalar, Scalar: model.ScalarInt64}}, GoName: "Id", GoType: "int64"},
			{Field: model.Field{Name: "name", Type: model.FieldType{Kind: model.FieldKindScalar, Scalar: model.ScalarString}}, GoName: "Name", GoType: "string"},
			{Field: model.Field{Name: "nickname", Type: model.FieldType{Kind: model.FieldKindScalar, Scalar: model.ScalarString}}, GoName: "Nickname", GoType: "*string"},
			{Field: model.Field{Name: "tags", Cardinality: model.CardinalityRepeated, Type: model.FieldType{Kind: model.FieldKindScalar, Scalar: model.ScalarString}}, GoName: "Tags", GoType: "[]string"},
		},
	}
	updateMsg := transform.GoMessage{
		GoName:          "UserUpdateByID",
		UpdateSource:    "User",
		ConditionFields: []string{"id"},
		Fields: []transform.GoField{
			{Field: model.Field{Name: "id", Type: model.FieldType{Kind: model.FieldKindScalar, Scalar: model.ScalarInt64}}, GoName: "Id", GoType: "int64"},
			{Field: model.Field{Name: "name", Type: model.FieldType{Kind: model.FieldKindScalar, Scalar: model.ScalarString}}, GoName: "Name", GoType: "*string"},
			{Field: model.Field{Name: "nickname", Type: model.FieldType{Kind: model.FieldKindScalar, Scalar: model.ScalarString}}, GoName: "Nickname", GoType: "*string"},
			{Field: model.Field{Name: "tags", Cardinality: model.CardinalityRepeated, Type: model.FieldType{Kind: model.FieldKindScalar, Scalar: model.ScalarString}}, GoName: "Tags", GoType: "[]string"},
		},
	}
	msgIndex := map[string]*transform.GoMessage{"User": &srcMsg}
	ctx := Context{MessageIndex: msgIndex}

	gf := transform.GoFile{Source: "user.update.proto", Package: "testpkg", Messages: []transform.GoMessage{updateMsg}}
	src, err := File(gf, testModulePhase4, ctx)
	if err != nil {
		t.Fatalf("File() error: %v", err)
	}
	s := string(src)

	// ApplyTo method should exist.
	if !strings.Contains(s, "func (x *UserUpdateByID) ApplyTo(p *User) {") {
		t.Errorf("missing ApplyTo method in:\n%s", s)
	}
	// Condition field (id) should be skipped — no assignment to p.Id.
	if strings.Contains(s, "p.Id =") {
		t.Errorf("condition field id should be skipped in:\n%s", s)
	}
	// Optional ptr→non-ptr (name): nil-guard + deref.
	if !strings.Contains(s, "p.Name = *x.Name") {
		t.Errorf("optional field name should have deref assign in:\n%s", s)
	}
	if !strings.Contains(s, "if x.Name != nil") {
		t.Errorf("optional field name should have nil-guard in:\n%s", s)
	}
	// Optional ptr→ptr (nickname): nil-guard + pointer assign.
	if !strings.Contains(s, "p.Nickname = x.Nickname") {
		t.Errorf("optional ptr→ptr field nickname should have pointer assign in:\n%s", s)
	}
	// Repeated field (tags): nil-guard.
	if !strings.Contains(s, "p.Tags = x.Tags") {
		t.Errorf("repeated field tags should be assigned in:\n%s", s)
	}
}

func TestApplyToNotGeneratedForNonUpdate(t *testing.T) {
	t.Parallel()

	msg := transform.GoMessage{
		GoName: "User",
		Fields: []transform.GoField{
			{Field: model.Field{Name: "id", Type: model.FieldType{Kind: model.FieldKindScalar, Scalar: model.ScalarInt64}}, GoName: "Id", GoType: "int64"},
		},
	}
	gf := transform.GoFile{Source: "user.proto", Package: "testpkg", Messages: []transform.GoMessage{msg}}

	src, err := File(gf, testModulePhase4, Context{})
	if err != nil {
		t.Fatalf("File() error: %v", err)
	}
	if strings.Contains(string(src), "ApplyTo") {
		t.Errorf("ApplyTo should not be generated for non-update message")
	}
}

func TestApplyToNoContext(t *testing.T) {
	t.Parallel()

	// Update message without context — ApplyTo should not be generated.
	updateMsg := transform.GoMessage{
		GoName:       "UserUpdateByID",
		UpdateSource: "User",
		Fields: []transform.GoField{
			{Field: model.Field{Name: "name", Type: model.FieldType{Kind: model.FieldKindScalar, Scalar: model.ScalarString}}, GoName: "Name", GoType: "*string"},
		},
	}
	gf := transform.GoFile{Source: "user.update.proto", Package: "testpkg", Messages: []transform.GoMessage{updateMsg}}

	src, err := File(gf, testModulePhase4, Context{})
	if err != nil {
		t.Fatalf("File() error: %v", err)
	}
	if strings.Contains(string(src), "ApplyTo") {
		t.Errorf("ApplyTo should not be generated without MessageIndex context")
	}
}

func TestConditionFieldsPopulatedByFlatten(t *testing.T) {
	t.Parallel()

	// Verify conditionFieldsFor correctly identifies non-optional fields.
	msg := model.Message{
		Name:         "UserUpdateByID",
		FullName:     "test.UserUpdateByID",
		UpdateSource: "User",
		Fields: []model.Field{
			{Name: "id", Cardinality: model.CardinalitySingular, Optional: false, Type: model.FieldType{Kind: model.FieldKindScalar, Scalar: model.ScalarInt64}},
			{Name: "name", Cardinality: model.CardinalitySingular, Optional: true, Type: model.FieldType{Kind: model.FieldKindScalar, Scalar: model.ScalarString}},
			{Name: "tags", Cardinality: model.CardinalityRepeated, Type: model.FieldType{Kind: model.FieldKindScalar, Scalar: model.ScalarString}},
		},
	}
	file := model.File{Path: "user.update.proto", Package: "test", Messages: []model.Message{msg}}
	gf := transform.Flatten(file)

	goMsg := gf.Messages[0]
	if len(goMsg.ConditionFields) != 1 || goMsg.ConditionFields[0] != "id" {
		t.Errorf("ConditionFields = %v, want [id]", goMsg.ConditionFields)
	}
}
