package app

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/pinealctx/gcode/internal/config"
	"github.com/pinealctx/gcode/internal/model"
	"github.com/pinealctx/gcode/internal/parser"
	"github.com/pinealctx/gcode/internal/render"
	"github.com/pinealctx/gcode/internal/source"
	"github.com/pinealctx/gcode/internal/transform"
)

const modulePath = "github.com/pinealctx/gcode"

// Run is the process entry used by the CLI main package.
func Run(ctx context.Context, args []string) error {
	if len(args) > 0 && args[0] == "gen-proto" {
		return RunGenProto(ctx, args[1:])
	}
	cfg, err := config.Parse(args)
	if err != nil {
		return err
	}

	scanResult, err := source.Scan(cfg.InputDir)
	if err != nil {
		return fmt.Errorf("scan input directory: %w", err)
	}

	if len(scanResult.Files) == 0 {
		return fmt.Errorf("no .proto files found in %q", cfg.InputDir)
	}

	files, err := parser.Parse(ctx, []string{scanResult.ImportPath}, scanResult.Files)
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
	seen := make(map[string]string, len(files)*3)
	for _, f := range files {
		for _, name := range []string{outputFileName(f.Path), rpcOutputFileName(f.Path), httpOutputFileName(f.Path)} {
			if prev, ok := seen[name]; ok {
				return fmt.Errorf("output filename collision: %q and %q both produce %q", prev, f.Path, name)
			}
			seen[name] = f.Path
		}
	}

	// Flatten all files and build global message and enum indexes for cross-file lookups.
	type flattenedFile struct {
		src model.File
		gf  transform.GoFile
	}
	flattened := make([]flattenedFile, 0, len(files))
	msgIndex := make(map[string]*transform.GoMessage)
	enumIndex := make(map[string]transform.GoEnum)
	for _, f := range files {
		gf := transform.Flatten(f)
		for i := range gf.Messages {
			m := &gf.Messages[i]
			if _, exists := msgIndex[m.GoName]; exists {
				return fmt.Errorf("message name collision: %q appears in multiple proto files; cross-file same-name messages are not supported", m.GoName)
			}
			msgIndex[m.GoName] = m
		}
		for _, e := range gf.Enums {
			enumIndex[e.GoName] = e
		}
		flattened = append(flattened, flattenedFile{src: f, gf: gf})
	}
	rctx := render.Context{MessageIndex: msgIndex, EnumIndex: enumIndex}

	// Generate and write each file.
	for _, ff := range flattened {
		src, err := render.File(ff.gf, modulePath, rctx)
		if err != nil {
			return fmt.Errorf("render %q: %w", ff.src.Path, err)
		}

		outPath := filepath.Join(cfg.OutputDir, outputFileName(ff.src.Path))
		if err := os.WriteFile(outPath, src, 0o600); err != nil {
			return fmt.Errorf("write %q: %w", outPath, err)
		}

		validateSrc, err := render.ValidateFile(ff.gf, modulePath, rctx)
		if err != nil {
			return fmt.Errorf("render validate %q: %w", ff.src.Path, err)
		}

		validateOutPath := filepath.Join(cfg.OutputDir, validateOutputFileName(ff.src.Path))
		if err := os.WriteFile(validateOutPath, validateSrc, 0o600); err != nil {
			return fmt.Errorf("write validate %q: %w", validateOutPath, err)
		}

		if len(ff.gf.Services) > 0 {
			rpcSrc, err := render.RPCFile(ff.gf)
			if err != nil {
				return fmt.Errorf("render rpc %q: %w", ff.src.Path, err)
			}
			rpcOutPath := filepath.Join(cfg.OutputDir, rpcOutputFileName(ff.src.Path))
			if err := os.WriteFile(rpcOutPath, rpcSrc, 0o600); err != nil {
				return fmt.Errorf("write rpc %q: %w", rpcOutPath, err)
			}

			httpSrc, err := render.HTTPFile(ff.gf, modulePath)
			if err != nil {
				return fmt.Errorf("render http %q: %w", ff.src.Path, err)
			}
			httpOutPath := filepath.Join(cfg.OutputDir, httpOutputFileName(ff.src.Path))
			if err := os.WriteFile(httpOutPath, httpSrc, 0o600); err != nil {
				return fmt.Errorf("write http %q: %w", httpOutPath, err)
			}
		}
	}

	return nil
}

// outputFileName derives the .pb.dao.go output filename from a proto file path.
// e.g. "subdir/person.proto" → "person.pb.dao.go"
func outputFileName(protoPath string) string {
	base := filepath.Base(protoPath)
	name := strings.TrimSuffix(base, ".proto")
	return name + ".pb.dao.go"
}

// validateOutputFileName derives the .pb.dao.validate.go output filename from a proto file path.
// e.g. "subdir/person.proto" → "person.pb.dao.validate.go"
func validateOutputFileName(protoPath string) string {
	base := filepath.Base(protoPath)
	name := strings.TrimSuffix(base, ".proto")
	return name + ".pb.dao.validate.go"
}

// rpcOutputFileName derives the .pb.rpc.go output filename from a proto file path.
// e.g. "subdir/user_service.proto" → "user_service.pb.rpc.go"
func rpcOutputFileName(protoPath string) string {
	base := filepath.Base(protoPath)
	name := strings.TrimSuffix(base, ".proto")
	return name + ".pb.rpc.go"
}

// httpOutputFileName derives the .pb.http.go output filename from a proto file path.
// e.g. "subdir/user_service.proto" → "user_service.pb.http.go"
func httpOutputFileName(protoPath string) string {
	base := filepath.Base(protoPath)
	name := strings.TrimSuffix(base, ".proto")
	return name + ".pb.http.go"
}
