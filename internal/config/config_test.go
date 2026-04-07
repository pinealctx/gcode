package config

import (
	"errors"
	"testing"
)

func TestParse_MissingIn(t *testing.T) {
	t.Parallel()

	_, err := Parse([]string{"-out", "/out"})
	if err == nil {
		t.Fatal("expected error for missing -in, got nil")
	}
	if !errors.Is(err, ErrMissingInputDir) {
		t.Errorf("expected ErrMissingInputDir, got %T: %v", err, err)
	}
}

func TestParse_MissingOut(t *testing.T) {
	t.Parallel()

	_, err := Parse([]string{"-in", "/proto"})
	if err == nil {
		t.Fatal("expected error for missing -out, got nil")
	}
	if !errors.Is(err, ErrMissingOutputDir) {
		t.Errorf("expected ErrMissingOutputDir, got %T: %v", err, err)
	}
}

func TestParseGenProto_MissingIn(t *testing.T) {
	t.Parallel()

	_, err := ParseGenProto([]string{})
	if err == nil {
		t.Fatal("expected error for missing -in, got nil")
	}
	if !errors.Is(err, ErrMissingProtoInputDir) {
		t.Errorf("expected ErrMissingProtoInputDir, got %T: %v", err, err)
	}
}
