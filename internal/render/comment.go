package render

import "strings"

// writeComment writes leading comment lines to b as Go doc comment lines.
// Empty lines in c are written as "//" (no trailing space).
// No-op if c has no lines.
func writeComment(b *strings.Builder, lines []string) {
	for _, line := range lines {
		if line == "" {
			b.WriteString("//\n")
		} else {
			b.WriteString("// " + line + "\n")
		}
	}
}
