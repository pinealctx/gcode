package config

import (
	"errors"
	"flag"
	"fmt"
)

const (
	DuplicateSingularError    = "error"
	DuplicateSingularLastWins = "last-wins"
)

// GenProtoConfig contains configuration for the gen-proto subcommand.
type GenProtoConfig struct {
	InputDir string
}

// ParseGenProto parses CLI arguments for the gen-proto subcommand.
func ParseGenProto(args []string) (GenProtoConfig, error) {
	var cfg GenProtoConfig

	fs := flag.NewFlagSet("gcode gen-proto", flag.ContinueOnError)
	fs.StringVar(&cfg.InputDir, "in", "", "input proto directory (generated files are written to the same directory)")

	if err := fs.Parse(args); err != nil {
		return GenProtoConfig{}, fmt.Errorf("parse gen-proto flags: %w", err)
	}

	if remainingArgs := fs.Args(); len(remainingArgs) > 0 {
		return GenProtoConfig{}, fmt.Errorf("parse gen-proto flags: unexpected positional arguments %q", remainingArgs)
	}

	if err := cfg.Validate(); err != nil {
		return GenProtoConfig{}, err
	}

	return cfg, nil
}

// Validate validates gen-proto configuration values.
func (c GenProtoConfig) Validate() error {
	if c.InputDir == "" {
		return errors.New("validate gen-proto config: missing -in")
	}
	return nil
}

type Config struct {
	InputDir                  string
	OutputDir                 string
	DuplicateSingularStrategy string
	AllowJSONUnknownFields    bool
}

// Parse parses CLI arguments into a validated configuration.
func Parse(args []string) (Config, error) {
	var cfg Config

	fs := flag.NewFlagSet("gcode", flag.ContinueOnError)
	fs.StringVar(&cfg.InputDir, "in", "", "input proto directory")
	fs.StringVar(&cfg.OutputDir, "out", "", "output directory")
	fs.StringVar(&cfg.DuplicateSingularStrategy, "duplicate-singular", DuplicateSingularError, "duplicate singular field strategy: error|last-wins")
	fs.BoolVar(&cfg.AllowJSONUnknownFields, "allow-json-unknown-fields", false, "allow unknown JSON fields during unmarshal")

	if err := fs.Parse(args); err != nil {
		return Config{}, fmt.Errorf("parse cli flags: %w", err)
	}

	if remainingArgs := fs.Args(); len(remainingArgs) > 0 {
		return Config{}, fmt.Errorf("parse cli flags: unexpected positional arguments %q", remainingArgs)
	}

	if err := cfg.Validate(); err != nil {
		return Config{}, err
	}

	return cfg, nil
}

// Validate validates configuration values.
func (c Config) Validate() error {
	if c.InputDir == "" {
		return errors.New("validate cli config: missing -in")
	}
	if c.OutputDir == "" {
		return errors.New("validate cli config: missing -out")
	}

	switch c.DuplicateSingularStrategy {
	case DuplicateSingularError, DuplicateSingularLastWins:
		return nil
	default:
		return fmt.Errorf("validate cli config: unsupported -duplicate-singular value %q", c.DuplicateSingularStrategy)
	}
}
