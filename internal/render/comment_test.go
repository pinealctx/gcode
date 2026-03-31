package render

import (
	"strings"
	"testing"
)

func TestWriteComment(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name  string
		lines []string
		want  string
	}{
		{
			name:  "nil lines",
			lines: nil,
			want:  "",
		},
		{
			name:  "empty slice",
			lines: []string{},
			want:  "",
		},
		{
			name:  "single line",
			lines: []string{"Person represents a human."},
			want:  "// Person represents a human.\n",
		},
		{
			name:  "multi line",
			lines: []string{"First line.", "Second line."},
			want:  "// First line.\n// Second line.\n",
		},
		{
			name:  "empty line between text",
			lines: []string{"First.", "", "After blank."},
			want:  "// First.\n//\n// After blank.\n",
		},
		{
			name:  "blank line no trailing space",
			lines: []string{""},
			want:  "//\n", // must be exactly "//\n", not "// \n"
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			var b strings.Builder
			writeComment(&b, tc.lines)
			if got := b.String(); got != tc.want {
				t.Errorf("got %q, want %q", got, tc.want)
			}
		})
	}
}
