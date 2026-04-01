package app

import (
	"errors"
	"os"
	"path/filepath"
	"testing"

	"github.com/pinealctx/gcode/internal/config"
)

func TestRunGenTS_BasicRouting(t *testing.T) {
	t.Parallel()

	inputDir := t.TempDir()
	outputDir := t.TempDir()

	writeFile(t, filepath.Join(inputDir, "user.proto"), `syntax = "proto3";
package test;
option go_package = "test";

message User {
  string name = 1;
  int32  age  = 2;
}
`)

	err := RunGenTS(t.Context(), []string{"-in", inputDir, "-out", outputDir})
	if err != nil {
		t.Fatalf("RunGenTS returned error: %v", err)
	}

	// Output directory should exist (created by RunGenTS).
	if _, err := os.Stat(outputDir); err != nil {
		t.Errorf("output directory should exist: %v", err)
	}
}

func TestRunGenTS_MissingFlags(t *testing.T) {
	t.Parallel()

	err := RunGenTS(t.Context(), []string{})
	if err == nil {
		t.Fatal("expected error for missing flags, got nil")
	}
	if !errors.Is(err, config.ErrMissingTSInputDir) {
		t.Errorf("error = %q, want ErrMissingTSInputDir", err.Error())
	}
}
