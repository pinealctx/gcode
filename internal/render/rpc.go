package render

import (
	"fmt"
	"go/format"
	"strings"

	"github.com/pinealctx/gcode/internal/transform"
)

// RPCFile renders a Go source file containing interface definitions for all
// services in gf. The returned bytes are gofmt-formatted and ready to write
// to a .pb.rpc.go file. Callers should only invoke this when len(gf.Services) > 0.
func RPCFile(gf transform.GoFile) ([]byte, error) {
	var body strings.Builder
	for _, svc := range gf.Services {
		writeServiceInterface(&body, svc)
	}
	bodyStr := body.String()

	var b strings.Builder
	writeHeader(&b, gf.Source)
	writePackage(&b, gf.Package)

	if strings.Contains(bodyStr, "context.") {
		b.WriteString("import \"context\"\n\n")
	}

	b.WriteString(bodyStr)

	src, err := format.Source([]byte(b.String()))
	if err != nil {
		return nil, fmt.Errorf("format generated rpc source for %q: %w", gf.Source, err)
	}
	return src, nil
}

func writeServiceInterface(b *strings.Builder, svc transform.GoService) {
	writeComment(b, svc.Comment.Lines)
	fmt.Fprintf(b, "type %s interface {\n", svc.GoName)
	for _, m := range svc.Methods {
		writeComment(b, m.Comment.Lines)
		fmt.Fprintf(b, "\t%s(ctx context.Context, req *%s) (*%s, error)\n",
			m.GoName, m.RequestType, m.ResponseType)
	}
	b.WriteString("}\n\n")
}
