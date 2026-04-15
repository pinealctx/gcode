package app

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/pinealctx/gcode/internal/config"
	"github.com/pinealctx/gcode/internal/parser"
	"github.com/pinealctx/gcode/internal/source"
	"github.com/pinealctx/gcode/internal/transform"
	"github.com/pinealctx/gcode/internal/tsrender"
)

// RunGenTS implements the gen-ts subcommand: scans the input directory,
// parses proto files, flattens to GoFile IR, and generates TypeScript output.
func RunGenTS(ctx context.Context, args []string) error {
	cfg, err := config.ParseGenTS(args)
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

	// Exclude schema source files (.meta.proto) — only entity/create/update products are used.
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
	if err := checkOutputCollisions(files, tsOutputFileName); err != nil {
		return err
	}

	// First pass: flatten all files.
	goFiles := make([]transform.GoFile, len(files))
	for i, f := range files {
		goFiles[i] = transform.Flatten(f)
	}

	// Build type registry: GoName → source .pb.ts output file name.
	registry := make(tsrender.TypeRegistry)
	for i, gf := range goFiles {
		srcFile := tsOutputFileName(files[i].Path)
		for _, enum := range gf.Enums {
			registry[enum.GoName] = srcFile
		}
		for _, msg := range gf.Messages {
			registry[msg.GoName] = srcFile
		}
	}

	// Second pass: render TS files with type registry for cross-file imports.
	for i, gf := range goFiles {
		tsSrc, err := tsrender.TSFile(gf, registry)
		if err != nil {
			return fmt.Errorf("render ts %q: %w", files[i].Path, err)
		}

		outPath := filepath.Join(cfg.OutputDir, tsOutputFileName(files[i].Path))
		if err := os.WriteFile(outPath, tsSrc, 0o644); err != nil { //nolint:gosec // generated source files should be world-readable
			return fmt.Errorf("write %q: %w", outPath, err)
		}
	}

	return nil
}

// tsOutputFileName derives the .pb.ts output filename from a proto file path.
// e.g. "subdir/person.proto" → "person.pb.ts"
func tsOutputFileName(protoPath string) string {
	base := filepath.Base(protoPath)
	name := strings.TrimSuffix(base, ".proto")
	return name + ".pb.ts"
}
