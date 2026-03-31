package render

import (
	"fmt"
	"strings"

	"github.com/pinealctx/gcode/internal/transform"
)

func writeMessage(b *strings.Builder, msg transform.GoMessage, providers []TagProvider) {
	writeComment(b, msg.Comment.Lines)
	fmt.Fprintf(b, "type %s struct {\n", msg.GoName)
	for _, f := range msg.Fields {
		writeField(b, f, providers)
	}
	b.WriteString("}\n\n")
	if msg.GormMessageOptions != nil && msg.GormMessageOptions.Table != "" {
		fmt.Fprintf(b, "func (%s) TableName() string { return %q }\n\n", msg.GoName, msg.GormMessageOptions.Table)
	}
}

func writeField(b *strings.Builder, f transform.GoField, providers []TagProvider) {
	writeComment(b, f.LeadingComment.Lines)
	tag := buildTag(f, providers)
	fmt.Fprintf(b, "%s %s %s\n", f.GoName, f.GoType, tag)
}
