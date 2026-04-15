package app

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/pinealctx/x/errorx"

	"github.com/pinealctx/gcode/internal/config"
	"github.com/pinealctx/gcode/internal/model"
	"github.com/pinealctx/gcode/internal/parser"
	"github.com/pinealctx/gcode/internal/render"
	"github.com/pinealctx/gcode/internal/source"
	"github.com/pinealctx/gcode/internal/transform"
)

// modulePath is the Go module path of gcode itself. It is used to generate
// import paths for the runtime packages (runtime, validateruntime, httpruntime)
// in generated files. This is intentionally hardcoded: the runtime packages
// are part of the gcode module and their import paths are stable public API.
// If the module path ever changes (e.g. major version bump), generated files
// must be regenerated.
const modulePath = "github.com/pinealctx/gcode"

// Run is the process entry used by the CLI main package for the default
// gen-dao action.
func Run(ctx context.Context, args []string) error {
	cfg, err := config.Parse(args)
	if err != nil {
		return err
	}

	scanResult, err := source.Scan(cfg.InputDir)
	if err != nil {
		return fmt.Errorf("scan input directory: %w", err)
	}

	if len(scanResult.Files) == 0 {
		return fmt.Errorf("no .proto files found in %q: %w", cfg.InputDir, ErrNoProtoFiles)
	}

	// Exclude schema source files (.meta.proto) from gen-dao processing.
	// These are only consumed by gen-proto; their entity/create/update products
	// are what gen-dao reads.
	inputFiles := filterMetaProtoSources(scanResult.Files)

	files, err := parser.Parse(ctx, []string{scanResult.ImportPath}, inputFiles)
	if err != nil {
		return fmt.Errorf("parse proto files: %w", err)
	}

	if err := transform.ValidateCreateOptions(files); err != nil {
		return fmt.Errorf("validate create options: %w", err)
	}

	// Ensure output directory exists.
	if err := os.MkdirAll(cfg.OutputDir, 0o755); err != nil {
		return fmt.Errorf("create output directory %q: %w", cfg.OutputDir, err)
	}

	// Check for output filename collisions before writing anything.
	// .pb.dao.go, .pb.rpc.go, and .pb.http.go names are checked to catch same-basename proto files.
	if err := checkOutputCollisions(files, outputFileName, rpcOutputFileName, httpOutputFileName); err != nil {
		return err
	}

	// Flatten all files and build global message and enum indexes for cross-file lookups.
	type flattenedFile struct {
		src model.File
		gf  transform.GoFile
	}
	// indexedMsg bundles a GoMessage pointer with its source proto path for collision error messages.
	// Pointer is used because GoMessage contains a []GoField slice; value copy would be expensive.
	type indexedMsg struct {
		msg *transform.GoMessage
		src string
	}
	// indexedEnum bundles a GoEnum value with its source proto path for collision error messages.
	// Value copy is fine: GoEnum is small (GoName + Values slice header).
	type indexedEnum struct {
		enum transform.GoEnum
		src  string
	}
	flattened := make([]flattenedFile, 0, len(files))
	msgIndex := make(map[string]indexedMsg)
	enumIndex := make(map[string]indexedEnum)
	for _, f := range files {
		gf := transform.Flatten(f)
		for i := range gf.Messages {
			m := &gf.Messages[i]
			if prev, exists := msgIndex[m.GoName]; exists {
				return errorx.NewSentinelf[appTag]("message name collision: %q defined in both %q and %q; cross-file same-name messages are not supported", m.GoName, prev.src, f.Path)
			}
			msgIndex[m.GoName] = indexedMsg{msg: m, src: f.Path}
		}
		for _, e := range gf.Enums {
			if prev, exists := enumIndex[e.GoName]; exists {
				return errorx.NewSentinelf[appTag]("enum name collision: %q defined in both %q and %q; cross-file same-name enums are not supported", e.GoName, prev.src, f.Path)
			}
			enumIndex[e.GoName] = indexedEnum{enum: e, src: f.Path}
		}
		flattened = append(flattened, flattenedFile{src: f, gf: gf})
	}

	// Build render indexes from the merged index maps.
	renderMsgIndex := make(map[string]*transform.GoMessage, len(msgIndex))
	renderEnumIndex := make(map[string]transform.GoEnum, len(enumIndex))
	for name, im := range msgIndex {
		renderMsgIndex[name] = im.msg
	}
	for name, ie := range enumIndex {
		renderEnumIndex[name] = ie.enum
	}
	rctx := render.Context{MessageIndex: renderMsgIndex, EnumIndex: renderEnumIndex}

	// Check for cross-type name collisions: a message and an enum with the same
	// GoName would produce two Go types with the same name in the same package.
	for goName, im := range msgIndex {
		if ie, exists := enumIndex[goName]; exists {
			return errorx.NewSentinelf[appTag]("name collision: %q defined as message in %q and as enum in %q; cross-type same-name types are not supported", goName, im.src, ie.src)
		}
	}

	// Generate and write each file.
	for _, ff := range flattened {
		src, err := render.File(ff.gf, modulePath, rctx)
		if err != nil {
			return fmt.Errorf("render %q: %w", ff.src.Path, err)
		}

		outPath := filepath.Join(cfg.OutputDir, outputFileName(ff.src.Path))
		if err := writeFileMkdir(outPath, src); err != nil {
			return fmt.Errorf("write %q: %w", outPath, err)
		}

		validateSrc, err := render.ValidateFile(ff.gf, modulePath, rctx)
		if err != nil {
			return fmt.Errorf("render validate %q: %w", ff.src.Path, err)
		}

		validateOutPath := filepath.Join(cfg.OutputDir, validateOutputFileName(ff.src.Path))
		if err := writeFileMkdir(validateOutPath, validateSrc); err != nil {
			return fmt.Errorf("write validate %q: %w", validateOutPath, err)
		}

		if len(ff.gf.Services) > 0 {
			rpcSrc, err := render.RPCFile(ff.gf)
			if err != nil {
				return fmt.Errorf("render rpc %q: %w", ff.src.Path, err)
			}
			rpcOutPath := filepath.Join(cfg.OutputDir, rpcOutputFileName(ff.src.Path))
			if err := writeFileMkdir(rpcOutPath, rpcSrc); err != nil {
				return fmt.Errorf("write rpc %q: %w", rpcOutPath, err)
			}

			httpSrc, err := render.HTTPFile(ff.gf, modulePath)
			if err != nil {
				return fmt.Errorf("render http %q: %w", ff.src.Path, err)
			}
			httpOutPath := filepath.Join(cfg.OutputDir, httpOutputFileName(ff.src.Path))
			if err := writeFileMkdir(httpOutPath, httpSrc); err != nil {
				return fmt.Errorf("write http %q: %w", httpOutPath, err)
			}
		}
	}

	return nil
}

// filterMetaProtoSources excludes schema source files (.meta.proto) from the
// input list. These are only consumed by gen-proto; their generated entity/create/update
// products are what gen-dao and gen-ts read.
func filterMetaProtoSources(files []string) []string {
	var result []string
	for _, f := range files {
		if !strings.HasSuffix(f, ".meta.proto") {
			result = append(result, f)
		}
	}
	return result
}

// outputFileName derives the .pb.dao.go output filename from a proto file path,
// preserving the directory structure to avoid collisions from same-name protos
// in different subdirectories.
// e.g. "subdir/person.proto" → "subdir/person.pb.dao.go"
func outputFileName(protoPath string) string {
	name := strings.TrimSuffix(protoPath, ".proto")
	return name + ".pb.dao.go"
}

// validateOutputFileName derives the .pb.dao.validate.go output filename from a proto file path,
// preserving the directory structure to avoid collisions from same-name protos
// in different subdirectories.
// e.g. "subdir/person.proto" → "subdir/person.pb.dao.validate.go"
func validateOutputFileName(protoPath string) string {
	name := strings.TrimSuffix(protoPath, ".proto")
	return name + ".pb.dao.validate.go"
}

// rpcOutputFileName derives the .pb.rpc.go output filename from a proto file path,
// preserving the directory structure to avoid collisions from same-name protos
// in different subdirectories.
// e.g. "subdir/user_service.proto" → "subdir/user_service.pb.rpc.go"
func rpcOutputFileName(protoPath string) string {
	name := strings.TrimSuffix(protoPath, ".proto")
	return name + ".pb.rpc.go"
}

// httpOutputFileName derives the .pb.http.go output filename from a proto file path,
// preserving the directory structure to avoid collisions from same-name protos
// in different subdirectories.
// e.g. "subdir/user_service.proto" → "subdir/user_service.pb.http.go"
func httpOutputFileName(protoPath string) string {
	name := strings.TrimSuffix(protoPath, ".proto")
	return name + ".pb.http.go"
}

// checkOutputCollisions checks that no two proto files produce the same output filename.
// nameFuncs is a list of functions that derive output filenames from a proto file path.
func checkOutputCollisions(files []model.File, nameFuncs ...func(string) string) error {
	seen := make(map[string]string, len(files)*len(nameFuncs))
	for _, f := range files {
		for _, fn := range nameFuncs {
			name := fn(f.Path)
			if prev, ok := seen[name]; ok {
				return fmt.Errorf("%w: %q and %q both produce %q", ErrOutputFilenameCollision, prev, f.Path, name)
			}
			seen[name] = f.Path
		}
	}
	return nil
}

// writeFileMkdir writes data to path, creating parent directories as needed.
func writeFileMkdir(path string, data []byte) error {
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	return os.WriteFile(path, data, 0o644) //nolint:gosec // generated source files should be world-readable
}
