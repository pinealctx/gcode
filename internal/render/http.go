package render

import (
	"fmt"
	"go/format"
	"strings"

	"github.com/pinealctx/gcode/internal/transform"
)

// HTTPFile renders a Go source file containing gin HTTP handler factory functions
// for all services in gf. Each rpc method gets a XxxHandler(svc XxxService) gin.HandlerFunc
// function. The returned bytes are gofmt-formatted and ready to write to a .pb.http.go file.
// Callers should only invoke this when len(gf.Services) > 0.
func HTTPFile(gf transform.GoFile, modulePath string) ([]byte, error) {
	var body strings.Builder
	for _, svc := range gf.Services {
		writeServiceHandlers(&body, svc)
	}
	bodyStr := body.String()

	var b strings.Builder
	writeHeader(&b, gf.Source)
	writePackage(&b, gf.Package)

	if bodyStr != "" {
		fmt.Fprintf(&b, "import (\n")
		if strings.Contains(bodyStr, "http.") {
			fmt.Fprintf(&b, "\t\"net/http\"\n\n")
		}
		fmt.Fprintf(&b, "\t\"github.com/gin-gonic/gin\"\n")
		fmt.Fprintf(&b, "\t\"%s/httpruntime\"\n", modulePath)
		fmt.Fprintf(&b, ")\n\n")
	}

	b.WriteString(bodyStr)

	src, err := format.Source([]byte(b.String()))
	if err != nil {
		return nil, fmt.Errorf("format generated http source for %q: %w", gf.Source, err)
	}
	return src, nil
}

func writeServiceHandlers(b *strings.Builder, svc transform.GoService) {
	for _, m := range svc.Methods {
		writeComment(b, m.Comment.Lines)
		fmt.Fprintf(b, "func %sHandler(svc %s) gin.HandlerFunc {\n", m.GoName, svc.GoName)
		b.WriteString("\treturn func(c *gin.Context) {\n")
		fmt.Fprintf(b, "\t\tvar req %s\n", m.RequestType)
		b.WriteString("\t\tif err := c.ShouldBind(&req); err != nil {\n")
		b.WriteString("\t\t\t_ = c.Error(err)\n")
		b.WriteString("\t\t\treturn\n")
		b.WriteString("\t\t}\n")
		b.WriteString("\t\tif err := req.Validate(); err != nil {\n")
		b.WriteString("\t\t\t_ = c.Error(err)\n")
		b.WriteString("\t\t\treturn\n")
		b.WriteString("\t\t}\n")
		fmt.Fprintf(b, "\t\tresp, err := svc.%s(c.Request.Context(), &req)\n", m.GoName)
		b.WriteString("\t\tif err != nil {\n")
		b.WriteString("\t\t\t_ = c.Error(err)\n")
		b.WriteString("\t\t\treturn\n")
		b.WriteString("\t\t}\n")
		b.WriteString("\t\tc.JSON(http.StatusOK, httpruntime.OKResponse(resp))\n")
		b.WriteString("\t}\n")
		b.WriteString("}\n\n")
	}
}
