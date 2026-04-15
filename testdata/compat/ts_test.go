// Package compat_test — snapshot tests for generated TypeScript files.
// Verifies that gen-ts output matches the committed golden files in ts/.
package compat_test

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	"github.com/pinealctx/gcode/internal/app"
)

// _goldenFiles lists every expected .pb.ts golden file relative to testdata/compat/ts/.
var _goldenFiles = []string{
	"all_types.entity.pb.ts",
	"all_types.create.pb.ts",
	"all_types.update.pb.ts",
	"all_validate.pb.ts",
	"common.pb.ts",
	"item.entity.pb.ts",
	"item.create.pb.ts",
	"item.update.pb.ts",
	"person.entity.pb.ts",
	"person.create.pb.ts",
	"person.update.pb.ts",
	"person_service.pb.ts",
}

// TestTSSnapshot verifies that the TS generator output matches committed golden files.
func TestTSSnapshot(t *testing.T) {
	t.Parallel()

	protoDir := "proto"
	goldenDir := "ts"

	// Generate TS into a temp directory.
	outDir := t.TempDir()
	if err := app.RunGenTS(context.Background(), []string{"-in", protoDir, "-out", outDir}); err != nil {
		t.Fatalf("RunGenTS: %v", err)
	}

	// Verify no unexpected files were generated.
	entries, err := os.ReadDir(outDir)
	if err != nil {
		t.Fatalf("ReadDir output: %v", err)
	}
	expected := make(map[string]bool, len(_goldenFiles))
	for _, name := range _goldenFiles {
		expected[name] = true
	}
	for _, e := range entries {
		if !expected[e.Name()] {
			t.Errorf("unexpected generated file: %s", e.Name())
		}
	}

	for _, name := range _goldenFiles {
		t.Run(name, func(t *testing.T) {
			generated, err := os.ReadFile(filepath.Join(outDir, name))
			if err != nil {
				t.Fatalf("read generated file: %v", err)
			}

			golden, err := os.ReadFile(filepath.Join(goldenDir, name))
			if err != nil {
				t.Fatalf("read golden file: %v", err)
			}

			genStr := string(generated)
			goldStr := string(golden)

			if genStr != goldStr {
				// Show a compact line diff.
				genLines := strings.Split(genStr, "\n")
				goldLines := strings.Split(goldStr, "\n")

				var diff strings.Builder
				maxLines := len(genLines)
				if len(goldLines) > maxLines {
					maxLines = len(goldLines)
				}
				for i := 0; i < maxLines; i++ {
					var gl, dl string
					if i < len(goldLines) {
						gl = goldLines[i]
					}
					if i < len(genLines) {
						dl = genLines[i]
					}
					if gl != dl {
						fmt.Fprintf(&diff, "line %d:\n  golden:   %q\n  generated: %q\n", i+1, gl, dl)
					}
				}

				t.Errorf("generated %q does not match golden file:\n%s", name, diff.String())
			}
		})
	}
}

// _tsTestDir is the directory containing the TS runtime test project.
const _tsTestDir = "ts-test"

// requireNode skips the test if Node.js is not available.
func requireNode(t *testing.T) {
	t.Helper()
	if _, err := exec.LookPath("node"); err != nil {
		t.Skip("node not found in PATH, skipping TS runtime test")
	}
}

// runNpm runs an npm command in the ts-test directory.
func runNpm(t *testing.T, dir string, args ...string) {
	t.Helper()
	cmd := exec.Command("npm", args...)
	cmd.Dir = dir
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	if err := cmd.Run(); err != nil {
		t.Fatalf("npm %s: %v", strings.Join(args, " "), err)
	}
}

// TestTSTypeCheck runs tsc --noEmit on the generated TS files.
func TestTSTypeCheck(t *testing.T) {
	requireNode(t)
	if runtime.GOOS == "windows" {
		t.Skip("skipping npm-based test on Windows")
	}

	tsDir, err := filepath.Abs(_tsTestDir)
	if err != nil {
		t.Fatalf("resolve ts-test dir: %v", err)
	}

	runNpm(t, tsDir, "install")
	runNpm(t, tsDir, "run", "typecheck")
}

// TestTSRuntime runs the TS runtime test suite via tsx.
func TestTSRuntime(t *testing.T) {
	requireNode(t)
	if runtime.GOOS == "windows" {
		t.Skip("skipping npm-based test on Windows")
	}

	tsDir, err := filepath.Abs(_tsTestDir)
	if err != nil {
		t.Fatalf("resolve ts-test dir: %v", err)
	}

	runNpm(t, tsDir, "install")
	runNpm(t, tsDir, "test")
}
