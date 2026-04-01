package config

import (
	"errors"
	"testing"
)

func TestParseGenTS(t *testing.T) {
	t.Parallel()

	cfg, err := ParseGenTS([]string{"-in", "/proto", "-out", "/ts"})
	if err != nil {
		t.Fatalf("ParseGenTS returned error: %v", err)
	}
	if cfg.InputDir != "/proto" {
		t.Errorf("InputDir = %q, want %q", cfg.InputDir, "/proto")
	}
	if cfg.OutputDir != "/ts" {
		t.Errorf("OutputDir = %q, want %q", cfg.OutputDir, "/ts")
	}
}

func TestParseGenTS_MissingIn(t *testing.T) {
	t.Parallel()

	_, err := ParseGenTS([]string{"-out", "/ts"})
	if err == nil {
		t.Fatal("expected error for missing -in, got nil")
	}
	if !errors.Is(err, ErrMissingTSInputDir) {
		t.Errorf("error = %q, want ErrMissingTSInputDir", err.Error())
	}
}

func TestParseGenTS_MissingOut(t *testing.T) {
	t.Parallel()

	_, err := ParseGenTS([]string{"-in", "/proto"})
	if err == nil {
		t.Fatal("expected error for missing -out, got nil")
	}
	if !errors.Is(err, ErrMissingTSOutputDir) {
		t.Errorf("error = %q, want ErrMissingTSOutputDir", err.Error())
	}
}

func TestParseGenTS_UnexpectedArgs(t *testing.T) {
	t.Parallel()

	_, err := ParseGenTS([]string{"-in", "/proto", "-out", "/ts", "extra"})
	if err == nil {
		t.Fatal("expected error for unexpected positional arguments, got nil")
	}
	// Unexpected args error comes from fmt.Errorf with dynamic content,
	// so we check the message prefix which is stable.
	if err.Error()[:len("parse gen-ts flags")] != "parse gen-ts flags" {
		t.Errorf("error = %q, want to start with 'parse gen-ts flags'", err.Error())
	}
}
