package app

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// writeProtoFile writes a proto file into dir for testing.
func writeProtoFile(t *testing.T, dir, name, content string) {
	t.Helper()
	if err := os.WriteFile(filepath.Join(dir, name), []byte(content), 0o600); err != nil {
		t.Fatalf("write proto file %q: %v", name, err)
	}
}

// TestRunE2EAnnotations is the end-to-end acceptance test for phase 2.
// It exercises the full pipeline: proto with gcode annotations → Parse →
// Flatten → render.File → generated Go source, then verifies struct tags.
//
// This test corresponds to the acceptance sample in doc/phase2/checklist.md §8.
func TestRunE2EAnnotations(t *testing.T) {
	t.Parallel()

	inputDir := t.TempDir()
	outputDir := t.TempDir()

	writeProtoFile(t, inputDir, "user.proto", `syntax = "proto3";
package example;
option go_package = "example";

import "gcode/options.proto";

message User {
  option (gcode.message).gorm = { table: "users" };

  int32  id         = 1;
  string name       = 2;
  string email      = 3 [(gcode.field).gorm.column = "email_address"];
  string phone      = 4 [(gcode.field).json.omitempty = true];
  string secret_key = 5 [(gcode.field).json.ignore = true];
}

message Config {
  string key   = 1;
  string value = 2;
}
`)

	err := Run(t.Context(), []string{"-in", inputDir, "-out", outputDir})
	if err != nil {
		t.Fatalf("Run() error: %v", err)
	}

	outFile := filepath.Join(outputDir, "user.pb.dao.go")
	data, err := os.ReadFile(outFile)
	if err != nil {
		t.Fatalf("read output file: %v", err)
	}
	src := string(data)

	// --- User: has gorm annotation ---
	// id, name: default column name
	mustContain(t, src, "id gorm tag", `gorm:"column:id"`)
	mustContain(t, src, "name gorm tag", `gorm:"column:name"`)
	// email: column override
	mustContain(t, src, "email gorm override", `gorm:"column:email_address"`)
	// phone: omitempty, no gorm tag
	mustContain(t, src, "phone omitempty", `json:"phone,omitempty"`)
	mustNotContain(t, src, "phone no gorm", `phone" gorm:`)
	// secret_key: ignore, no gorm tag
	mustContain(t, src, "secret_key ignore", `json:"-"`)

	// --- Config: no gorm annotation — only json tags ---
	mustNotContain(t, src, "Config no gorm", `gorm:"column:key"`)
	mustContain(t, src, "Config key json", `json:"key"`)
	mustContain(t, src, "Config value json", `json:"value"`)
}

// TestRunE2EBackwardCompat verifies that a plain proto without any gcode
// annotations produces output identical in structure to phase 1 (json tags only,
// no gorm tags).
func TestRunE2EBackwardCompat(t *testing.T) {
	t.Parallel()

	inputDir := t.TempDir()
	outputDir := t.TempDir()

	writeProtoFile(t, inputDir, "plain.proto", `syntax = "proto3";
package plain;
option go_package = "plain";

message Person {
  string name = 1;
  int32  age  = 2;
}
`)

	err := Run(t.Context(), []string{"-in", inputDir, "-out", outputDir})
	if err != nil {
		t.Fatalf("Run() error: %v", err)
	}

	data, err := os.ReadFile(filepath.Join(outputDir, "plain.pb.dao.go"))
	if err != nil {
		t.Fatalf("read output file: %v", err)
	}
	src := string(data)

	mustContain(t, src, "name json", `json:"name"`)
	mustContain(t, src, "age json", `json:"age"`)
	mustNotContain(t, src, "no gorm tag", "gorm:")
}

// TestRunE2EJSONIgnorePrecedence verifies that when both omitempty and ignore
// are set on the same field, ignore wins and produces json:"-".
func TestRunE2EJSONIgnorePrecedence(t *testing.T) {
	t.Parallel()

	inputDir := t.TempDir()
	outputDir := t.TempDir()

	writeProtoFile(t, inputDir, "prec.proto", `syntax = "proto3";
package prec;
option go_package = "prec";

import "gcode/options.proto";

message Msg {
  string secret = 1 [(gcode.field).json = {omitempty: true, ignore: true}];
}
`)

	err := Run(t.Context(), []string{"-in", inputDir, "-out", outputDir})
	if err != nil {
		t.Fatalf("Run() error: %v", err)
	}

	data, err := os.ReadFile(filepath.Join(outputDir, "prec.pb.dao.go"))
	if err != nil {
		t.Fatalf("read output file: %v", err)
	}
	src := string(data)

	mustContain(t, src, "ignore wins", `json:"-"`)
	mustNotContain(t, src, "no omitempty", "omitempty")
}

// TestRunE2EGormWithJSONIgnore verifies that a field with both gorm (via message
// annotation) and json.ignore generates both tags correctly.
func TestRunE2EGormWithJSONIgnore(t *testing.T) {
	t.Parallel()

	inputDir := t.TempDir()
	outputDir := t.TempDir()

	writeProtoFile(t, inputDir, "combo.proto", `syntax = "proto3";
package combo;
option go_package = "combo";

import "gcode/options.proto";

message Row {
  option (gcode.message).gorm = { table: "rows" };

  string token = 1 [(gcode.field).json.ignore = true];
}
`)

	err := Run(t.Context(), []string{"-in", inputDir, "-out", outputDir})
	if err != nil {
		t.Fatalf("Run() error: %v", err)
	}

	data, err := os.ReadFile(filepath.Join(outputDir, "combo.pb.dao.go"))
	if err != nil {
		t.Fatalf("read output file: %v", err)
	}
	src := string(data)

	// json:"-" and gorm tag must both be present
	mustContain(t, src, "json ignore", `json:"-"`)
	mustContain(t, src, "gorm column", `gorm:"column:token"`)
}

func mustContain(t *testing.T, src, label, substr string) {
	t.Helper()
	if !strings.Contains(src, substr) {
		t.Errorf("missing %s: %q not found in output:\n%s", label, substr, src)
	}
}

func mustNotContain(t *testing.T, src, label, substr string) {
	t.Helper()
	if strings.Contains(src, substr) {
		t.Errorf("unexpected %s: %q found in output:\n%s", label, substr, src)
	}
}

func TestRun_NonExistentInputDir(t *testing.T) {
	t.Parallel()

	err := Run(t.Context(), []string{"-in", "/nonexistent/path/that/does/not/exist", "-out", t.TempDir()})
	if err == nil {
		t.Fatal("expected error for non-existent input directory, got nil")
	}
	if !strings.Contains(err.Error(), "scan input directory") {
		t.Errorf("error = %q, want to contain 'scan input directory'", err.Error())
	}
}

func TestRun_ReadOnlyOutputDir(t *testing.T) {
	t.Parallel()

	if os.Getuid() == 0 {
		t.Skip("cannot test permission errors as root")
	}

	inputDir := t.TempDir()
	outputDir := t.TempDir()

	writeProtoFile(t, inputDir, "plain.proto", `syntax = "proto3";
package test;
option go_package = "example.com/test;testpb";
message Plain { string name = 1; }
`)

	if err := os.Chmod(outputDir, 0o555); err != nil {
		t.Fatalf("chmod: %v", err)
	}
	t.Cleanup(func() { _ = os.Chmod(outputDir, 0o755) })

	err := Run(t.Context(), []string{"-in", inputDir, "-out", outputDir})
	if err == nil {
		t.Fatal("expected error for read-only output directory, got nil")
	}
	if !strings.Contains(err.Error(), "permission denied") {
		t.Errorf("error = %q, want to contain 'permission denied'", err.Error())
	}
}
