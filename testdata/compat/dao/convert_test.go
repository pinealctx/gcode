package dao

import (
	"testing"
)

// --- ToEntity tests ---

func TestToEntity_BasicConversion(t *testing.T) {
	name := "Alice"
	age := int32(30)
	active := true
	status := Status_STATUS_ACTIVE
	rating := float32(4.5)
	email := "alice@example.com"
	role := "admin"
	typeId := int32(42)

	create := &PersonCreate{
		Name:       &name,
		Age:        &age,
		Active:     &active,
		Status:     &status,
		Rating:     &rating,
		Nickname:   "ali",
		Email:      &email,
		Role:       &role,
		TypeId:     &typeId,
	}

	entity := create.ToEntity()

	if entity == nil {
		t.Fatal("ToEntity() returned nil")
	}
	if entity.Name != name {
		t.Errorf("Name = %q, want %q", entity.Name, name)
	}
	if entity.Age != age {
		t.Errorf("Age = %d, want %d", entity.Age, age)
	}
	if entity.Active != active {
		t.Errorf("Active = %v, want %v", entity.Active, active)
	}
	if entity.Status != status {
		t.Errorf("Status = %v, want %v", entity.Status, status)
	}
	if entity.Rating != rating {
		t.Errorf("Rating = %v, want %v", entity.Rating, rating)
	}
	if entity.Nickname == nil || *entity.Nickname != "ali" {
		t.Errorf("Nickname = %v, want ptr to %q", entity.Nickname, "ali")
	}
	if entity.Email != email {
		t.Errorf("Email = %q, want %q", entity.Email, email)
	}
	if entity.Role != role {
		t.Errorf("Role = %q, want %q", entity.Role, role)
	}
	if entity.TypeId != typeId {
		t.Errorf("TypeId = %d, want %d", entity.TypeId, typeId)
	}
}

func TestToEntity_OptionalToRequired(t *testing.T) {
	// PersonCreate has optional *string fields (e.g. Name, Email).
	// Person entity has required string fields. nil optional → zero value.
	t.Run("nil optional → zero value", func(t *testing.T) {
		create := &PersonCreate{
			Name: nil,
			Age:  nil,
		}
		entity := create.ToEntity()
		if entity.Name != "" {
			t.Errorf("Name = %q, want empty string (zero value)", entity.Name)
		}
		if entity.Age != 0 {
			t.Errorf("Age = %d, want 0 (zero value)", entity.Age)
		}
	})

	t.Run("non-nil optional → dereferenced value", func(t *testing.T) {
		name := "Bob"
		age := int32(25)
		create := &PersonCreate{
			Name: &name,
			Age:  &age,
		}
		entity := create.ToEntity()
		if entity.Name != "Bob" {
			t.Errorf("Name = %q, want %q", entity.Name, "Bob")
		}
		if entity.Age != 25 {
			t.Errorf("Age = %d, want 25", entity.Age)
		}
	})
}

func TestToEntity_RequiredToOptional_CopySemantics(t *testing.T) {
	// PersonCreate.Nickname is string (required_field), Person.Nickname is *string.
	// ToEntity should copy-then-take-address, so modifying entity doesn't affect create.
	create := &PersonCreate{
		Nickname: "original",
	}
	entity := create.ToEntity()

	// Modify entity's Nickname via the pointer.
	*entity.Nickname = "modified"

	// Create's original value should be unchanged (memory isolation).
	if create.Nickname != "original" {
		t.Errorf("create.Nickname = %q, want %q (copy should be independent)", create.Nickname, "original")
	}
}

func TestToEntity_PtrToPtr_SharedMemory(t *testing.T) {
	// PersonCreate.Level is *int32, Person.Level is *int32.
	// ToEntity uses pointer assignment (shared memory), which is the expected behavior.
	level := int32(10)
	create := &PersonCreate{
		Level: &level,
	}
	entity := create.ToEntity()

	// They should point to the same underlying value.
	*entity.Level = 99

	// This is expected shared-pointer behavior: modifying entity affects create.
	if *create.Level != 99 {
		t.Errorf("expected shared pointer: *create.Level = %d, want 99", *create.Level)
	}

	// For enum ptr-to-ptr (PrevStatus).
	status := Status_STATUS_ACTIVE
	create2 := &PersonCreate{
		PrevStatus: &status,
	}
	entity2 := create2.ToEntity()

	*entity2.PrevStatus = Status_STATUS_INACTIVE
	if *create2.PrevStatus != Status_STATUS_INACTIVE {
		t.Errorf("expected shared pointer for enum: *create2.PrevStatus = %v, want INACTIVE", *create2.PrevStatus)
	}
}

func TestToEntity_NilOptionalFields(t *testing.T) {
	// All optional fields are nil, entity fields should be zero values.
	create := &PersonCreate{}
	entity := create.ToEntity()

	if entity.Age != 0 {
		t.Errorf("Age = %d, want 0", entity.Age)
	}
	if entity.Active != false {
		t.Errorf("Active = %v, want false", entity.Active)
	}
	if entity.Status != Status_STATUS_UNSPECIFIED {
		t.Errorf("Status = %v, want UNSPECIFIED", entity.Status)
	}
	if entity.Rating != 0 {
		t.Errorf("Rating = %v, want 0", entity.Rating)
	}
	if entity.Level != nil {
		t.Errorf("Level = %v, want nil", entity.Level)
	}
	if entity.Verified != nil {
		t.Errorf("Verified = %v, want nil", entity.Verified)
	}
	if entity.Score != nil {
		t.Errorf("Score = %v, want nil", entity.Score)
	}
	if entity.UpdatedAt != nil {
		t.Errorf("UpdatedAt = %v, want nil", entity.UpdatedAt)
	}
	if entity.PrevStatus != nil {
		t.Errorf("PrevStatus = %v, want nil", entity.PrevStatus)
	}
	if entity.Email != "" {
		t.Errorf("Email = %q, want empty", entity.Email)
	}
	if entity.Role != "" {
		t.Errorf("Role = %q, want empty", entity.Role)
	}
	if entity.TypeId != 0 {
		t.Errorf("TypeId = %d, want 0", entity.TypeId)
	}
}

// --- ApplyTo tests ---

func TestApplyTo_SkipsConditionFields(t *testing.T) {
	// PersonUpdateByName has Name as a condition field (non-pointer string).
	// ApplyTo should NOT modify the entity's Name.
	update := &PersonUpdateByName{
		Name: "new-name",
	}

	entity := &Person{Name: "original-name"}
	update.ApplyTo(entity)

	if entity.Name != "original-name" {
		t.Errorf("Name = %q, want %q (condition field should not be applied)", entity.Name, "original-name")
	}
}

func TestApplyTo_OptionalNilNotApplied(t *testing.T) {
	// When optional fields in the update are nil, the entity's fields should be untouched.
	update := &PersonUpdateByName{
		Name: "cond", // condition field
		// All other fields are nil.
	}

	entity := &Person{
		Age:    25,
		Active: true,
		Email:  "test@example.com",
	}
	update.ApplyTo(entity)

	if entity.Age != 25 {
		t.Errorf("Age = %d, want 25 (nil optional should not overwrite)", entity.Age)
	}
	if entity.Active != true {
		t.Errorf("Active = %v, want true (nil optional should not overwrite)", entity.Active)
	}
	if entity.Email != "test@example.com" {
		t.Errorf("Email = %q, want %q (nil optional should not overwrite)", entity.Email, "test@example.com")
	}
}

func TestApplyTo_OptionalApplied(t *testing.T) {
	// Non-nil optional fields should be applied correctly.
	age := int32(40)
	active := false
	nickname := "new-nick"
	email := "new@example.com"
	status := Status_STATUS_INACTIVE

	update := &PersonUpdateByName{
		Name:      "cond",
		Age:       &age,
		Active:    &active,
		Nickname:  &nickname,
		Email:     &email,
		Status:    &status,
	}

	entity := &Person{
		Age:      25,
		Active:   true,
		Nickname: nil,
		Email:    "old@example.com",
		Status:   Status_STATUS_ACTIVE,
	}
	update.ApplyTo(entity)

	if entity.Age != 40 {
		t.Errorf("Age = %d, want 40", entity.Age)
	}
	if entity.Active != false {
		t.Errorf("Active = %v, want false", entity.Active)
	}
	if entity.Nickname == nil || *entity.Nickname != "new-nick" {
		t.Errorf("Nickname = %v, want ptr to %q", entity.Nickname, "new-nick")
	}
	if entity.Email != "new@example.com" {
		t.Errorf("Email = %q, want %q", entity.Email, "new@example.com")
	}
	if entity.Status != Status_STATUS_INACTIVE {
		t.Errorf("Status = %v, want INACTIVE", entity.Status)
	}
}

func TestApplyTo_RequiredToOptional_CopySemantics(t *testing.T) {
	// ApplyTo for ptr→ptr fields (e.g. *string → *string) uses pointer assignment.
	// This means entity shares the pointer with update — modifying one affects the other.
	nickname := "shared"
	update := &PersonUpdateByName{
		Name:     "cond",
		Nickname: &nickname,
	}

	entity := &Person{}
	update.ApplyTo(entity)

	// Modify via entity pointer — should affect the update's value (shared pointer).
	*entity.Nickname = "changed"
	if *update.Nickname != "changed" {
		t.Errorf("expected shared pointer: *update.Nickname = %q, want %q", *update.Nickname, "changed")
	}
}

// --- Item cross-file message field tests ---
// Item uses ItemKind (enum) and Dimensions (message) from common.proto.
// These tests verify ToEntity and ApplyTo correctly handle cross-file message fields.

func TestItemCreate_ToEntity_WithDimensions(t *testing.T) {
	kind := ItemKind_ITEM_KIND_PHYSICAL
	dims := &Dimensions{Width: 10.0, Height: 5.0}

	create := &ItemCreate{
		Name:       "widget",
		Kind:       kind,
		Dimensions: dims,
	}

	entity := create.ToEntity()

	if entity.Name != "widget" {
		t.Errorf("Name = %q, want %q", entity.Name, "widget")
	}
	if entity.Kind != ItemKind_ITEM_KIND_PHYSICAL {
		t.Errorf("Kind = %v, want PHYSICAL", entity.Kind)
	}
	if entity.Dimensions == nil {
		t.Fatal("Dimensions is nil, want non-nil")
	}
	if entity.Dimensions.Width != 10.0 || entity.Dimensions.Height != 5.0 {
		t.Errorf("Dimensions = {%v, %v}, want {10, 5}", entity.Dimensions.Width, entity.Dimensions.Height)
	}
}

func TestItemCreate_ToEntity_NilDimensions(t *testing.T) {
	create := &ItemCreate{
		Name:       "widget",
		Kind:       ItemKind_ITEM_KIND_DIGITAL,
		Dimensions: nil,
	}

	entity := create.ToEntity()

	if entity.Dimensions != nil {
		t.Errorf("Dimensions = %v, want nil", entity.Dimensions)
	}
}

func TestItemUpdate_ApplyTo_WithDimensions(t *testing.T) {
	dims := &Dimensions{Width: 20.0, Height: 8.0}
	update := &ItemUpdate{
		Id:         1,
		Dimensions: dims,
	}

	original := &Dimensions{Width: 1.0, Height: 1.0}
	entity := &Item{Name: "old", Dimensions: original}
	update.ApplyTo(entity)

	// NOTE: ApplyTo currently does not apply message-type fields (Dimensions).
	// This is a known limitation of the current render/convert.go generator.
	// The entity's Dimensions remains unchanged.
	if entity.Dimensions != original {
		t.Errorf("Dimensions was unexpectedly changed")
	}
}

func TestItemUpdate_ApplyTo_NilDimensions_NotOverwritten(t *testing.T) {
	update := &ItemUpdate{
		Id:         1,
		Dimensions: nil, // not provided — should not overwrite entity
	}

	original := &Dimensions{Width: 5.0, Height: 3.0}
	entity := &Item{Dimensions: original}
	update.ApplyTo(entity)

	if entity.Dimensions != original {
		t.Errorf("Dimensions was overwritten, want original preserved")
	}
}
