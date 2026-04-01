package tsrender

import (
	"testing"

	"github.com/pinealctx/gcode/internal/model"
)

func TestTSScalarType(t *testing.T) {
	t.Parallel()

	tests := []struct {
		kind model.ScalarKind
		want string
	}{
		// 32-bit integers → number
		{model.ScalarInt32, "number"},
		{model.ScalarSint32, "number"},
		{model.ScalarSfixed32, "number"},
		{model.ScalarUint32, "number"},
		{model.ScalarFixed32, "number"},
		// 64-bit integers → string
		{model.ScalarInt64, "string"},
		{model.ScalarSint64, "string"},
		{model.ScalarSfixed64, "string"},
		{model.ScalarUint64, "string"},
		{model.ScalarFixed64, "string"},
		// Floating point → number
		{model.ScalarFloat, "number"},
		{model.ScalarDouble, "number"},
		// Other scalars
		{model.ScalarBool, "boolean"},
		{model.ScalarString, "string"},
		{model.ScalarBytes, "string"},
	}

	for _, tt := range tests {
		t.Run(string(tt.kind), func(t *testing.T) {
			t.Parallel()
			got := tsScalarType(tt.kind)
			if got != tt.want {
				t.Errorf("tsScalarType(%q) = %q, want %q", tt.kind, got, tt.want)
			}
		})
	}
}
