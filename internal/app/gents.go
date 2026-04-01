package app

import (
	"context"
	"fmt"
	"os"

	"github.com/pinealctx/gcode/internal/config"
	"github.com/pinealctx/gcode/internal/parser"
	"github.com/pinealctx/gcode/internal/source"
	"github.com/pinealctx/gcode/internal/transform"
)

// RunGenTS implements the gen-ts subcommand: scans the input directory,
// parses proto files, flattens to GoFile IR, and generates TypeScript output.
// The current skeleton validates config and runs the pipeline up to Flatten,
// but does not yet render TS files.
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

	// Flatten all files (TS renderer will consume GoFile IR in subtask_2).
	for _, f := range files {
		_ = transform.Flatten(f)
	}

	return nil
}
