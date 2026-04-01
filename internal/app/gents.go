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

	for _, f := range files {
		gf := transform.Flatten(f)

		tsSrc, err := tsrender.TSFile(gf)
		if err != nil {
			return fmt.Errorf("render ts %q: %w", f.Path, err)
		}

		outPath := filepath.Join(cfg.OutputDir, tsOutputFileName(f.Path))
		if err := os.WriteFile(outPath, tsSrc, 0o600); err != nil {
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
