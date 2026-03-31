package render

import "github.com/pinealctx/gcode/internal/transform"

// GormTagProvider generates gorm struct tags for fields belonging to
// messages that have a GORM annotation (GormMessageOptions != nil).
type GormTagProvider struct{}

// Key returns the struct tag key "gorm".
func (p *GormTagProvider) Key() string { return "gorm" }

// Value returns the gorm tag value for the field, or empty string if the
// owning message has no GORM annotation.
// The column name defaults to the proto field name; it can be overridden
// via the field-level GormFieldOptions.Column annotation.
func (p *GormTagProvider) Value(f transform.GoField) string {
	if f.GormMessageOptions == nil {
		return ""
	}
	col := f.Name // proto field name as default column name
	if f.GormOptions != nil && f.GormOptions.Column != "" {
		col = f.GormOptions.Column
	}
	return "column:" + col
}
