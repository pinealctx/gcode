// Package dao_test verifies gorm annotation behavior of generated structs.
// Tests cover TableName() generation, gorm struct tag correctness, and
// the absence of gorm tags on messages without gorm annotations.
package dao_test

import (
	"reflect"
	"testing"

	"github.com/pinealctx/gcode/testdata/compat/dao"
)

// TestPersonTableName verifies that Person.TableName() returns the table name
// configured via (gcode.message).gorm.table annotation.
func TestPersonTableName(t *testing.T) {
	t.Parallel()

	var p dao.Person
	if got := p.TableName(); got != "persons" {
		t.Errorf("Person.TableName() = %q, want %q", got, "persons")
	}
}

// TestPersonCreateTableName verifies that PersonCreate inherits the source
// message's table name and generates the same TableName() method.
func TestPersonCreateTableName(t *testing.T) {
	t.Parallel()

	var p dao.PersonCreate
	if got := p.TableName(); got != "persons" {
		t.Errorf("PersonCreate.TableName() = %q, want %q", got, "persons")
	}
}

// TestPersonUpdateByNameNoTableName verifies at compile time that
// PersonUpdateByName does not implement the TableName() method.
// Update derived messages use db.Model(&Person{}) for table routing.
func TestPersonUpdateByNameNoTableName(t *testing.T) {
	t.Parallel()

	type tableNamer interface{ TableName() string }
	_, ok := any(dao.PersonUpdateByName{}).(tableNamer)
	if ok {
		t.Error("PersonUpdateByName must not implement TableName()")
	}
}

// TestPersonGormColumnOverride verifies that a field with (gcode.field).gorm.column
// annotation generates the overridden column name in the gorm struct tag.
func TestPersonGormColumnOverride(t *testing.T) {
	t.Parallel()

	rt := reflect.TypeOf(dao.Person{})
	f, ok := rt.FieldByName("CreatedAt")
	if !ok {
		t.Fatal("Person.CreatedAt field not found")
	}
	got := f.Tag.Get("gorm")
	want := "column:created_ts"
	if got != want {
		t.Errorf("Person.CreatedAt gorm tag = %q, want %q", got, want)
	}
}

// TestPersonGormDefaultColumn verifies that a field without gorm.column override
// uses the proto field name as the default column name.
func TestPersonGormDefaultColumn(t *testing.T) {
	t.Parallel()

	rt := reflect.TypeOf(dao.Person{})
	f, ok := rt.FieldByName("Name")
	if !ok {
		t.Fatal("Person.Name field not found")
	}
	got := f.Tag.Get("gorm")
	want := "column:name"
	if got != want {
		t.Errorf("Person.Name gorm tag = %q, want %q", got, want)
	}
}

// TestAddressNoGormTag verifies that Address, which has no gorm annotation,
// does not generate any gorm struct tags.
func TestAddressNoGormTag(t *testing.T) {
	t.Parallel()

	rt := reflect.TypeOf(dao.Address{})
	for i := range rt.NumField() {
		f := rt.Field(i)
		if tag := f.Tag.Get("gorm"); tag != "" {
			t.Errorf("Address.%s must not have gorm tag, got %q", f.Name, tag)
		}
	}
}

// TestPersonUpdateByNameToMapUsesGormColumn verifies that ToMap() uses the
// gorm column name as the map key when a field has a gorm.column override.
func TestPersonUpdateByNameToMapUsesGormColumn(t *testing.T) {
	t.Parallel()

	createdAt := int64(1234567890)
	req := &dao.PersonUpdateByName{
		Name:      "Alice",
		CreatedAt: &createdAt,
	}
	m := req.ToMap()

	if _, ok := m["created_ts"]; !ok {
		t.Errorf("ToMap() must use gorm column name 'created_ts' as key, got keys: %v", mapKeys(m))
	}
	if _, ok := m["created_at"]; ok {
		t.Error("ToMap() must not use proto field name 'created_at' as key")
	}
}

func mapKeys(m map[string]any) []string {
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	return keys
}
