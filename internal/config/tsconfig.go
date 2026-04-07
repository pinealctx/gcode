package config

import (
	"flag"
	"fmt"
)

// GenTSConfig contains configuration for the gen-ts subcommand.
type GenTSConfig struct {
	InputDir  string
	OutputDir string
}

// ParseGenTS parses CLI arguments for the gen-ts subcommand.
func ParseGenTS(args []string) (GenTSConfig, error) {
	var cfg GenTSConfig

	fs := flag.NewFlagSet("gcode gen-ts", flag.ContinueOnError)
	fs.StringVar(&cfg.InputDir, "in", "", "input proto directory")
	fs.StringVar(&cfg.OutputDir, "out", "", "output TypeScript directory")

	if err := fs.Parse(args); err != nil {
		return GenTSConfig{}, fmt.Errorf("parse gen-ts flags: %w", err)
	}

	if remainingArgs := fs.Args(); len(remainingArgs) > 0 {
		return GenTSConfig{}, fmt.Errorf("parse gen-ts flags: unexpected positional arguments %q", remainingArgs)
	}

	if err := cfg.Validate(); err != nil {
		return GenTSConfig{}, err
	}

	return cfg, nil
}

// Validate validates gen-ts configuration values.
func (c GenTSConfig) Validate() error {
	if c.InputDir == "" {
		return ErrMissingTSInputDir
	}
	if c.OutputDir == "" {
		return ErrMissingTSOutputDir
	}
	return nil
}
