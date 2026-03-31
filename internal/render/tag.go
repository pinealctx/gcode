package render

import (
	"fmt"
	"strings"

	"github.com/pinealctx/gcode/internal/transform"
)

// TagProvider generates a struct tag value for a single field.
// Return empty string to skip this tag for the field.
type TagProvider interface {
	Key() string
	Value(f transform.GoField) string
}

// buildTag constructs the full struct tag string for a field.
// The json tag is always generated (built-in, not via provider).
// Additional tags are appended in provider order, skipping empty values.
func buildTag(f transform.GoField, providers []TagProvider) string {
	jsonVal := jsonTagValue(f)
	parts := []string{fmt.Sprintf("json:%q", jsonVal)}
	for _, p := range providers {
		v := p.Value(f)
		if v != "" {
			parts = append(parts, fmt.Sprintf("%s:%q", p.Key(), v))
		}
	}
	return "`" + strings.Join(parts, " ") + "`"
}

// jsonTagValue returns the json tag value for a field.
// ignore takes precedence over omitempty when both are set.
func jsonTagValue(f transform.GoField) string {
	if f.JSONOptions != nil && f.JSONOptions.Ignore {
		return "-"
	}
	name := f.JSONName
	if f.JSONOptions != nil && f.JSONOptions.Omitempty {
		return name + ",omitempty"
	}
	return name
}
