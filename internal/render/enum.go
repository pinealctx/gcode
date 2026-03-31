package render

import (
	"fmt"
	"strings"

	"github.com/pinealctx/gcode/internal/transform"
)

func writeEnum(b *strings.Builder, enum transform.GoEnum) {
	writeComment(b, enum.Comment.Lines)
	fmt.Fprintf(b, "type %s int32\n\n", enum.GoName)

	if len(enum.Values) == 0 {
		return
	}

	b.WriteString("const (\n")
	for _, v := range enum.Values {
		writeComment(b, v.Comment.Lines)
		fmt.Fprintf(b, "%s %s = %d\n", v.GoName, enum.GoName, v.Number)
	}
	b.WriteString(")\n\n")
}
