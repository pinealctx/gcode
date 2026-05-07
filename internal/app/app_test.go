package app

import (
	"errors"
	"io/fs"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

// TestRunE2EAnnotations is the end-to-end acceptance test for phase 2.
// It exercises the full pipeline: proto with gcode annotations → Parse →
// Flatten → render.File → generated Go source, then verifies struct tags.
//
// This test corresponds to the acceptance sample in doc/phase2/checklist.md §8.
func TestRunE2EAnnotations(t *testing.T) {
	t.Parallel()

	inputDir := t.TempDir()
	outputDir := t.TempDir()

	writeFile(t, filepath.Join(inputDir, "user.proto"), `syntax = "proto3";
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
	mustNotContain(t, src, "secret_key no gorm", `secret_key" gorm:`)

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

	writeFile(t, filepath.Join(inputDir, "plain.proto"), `syntax = "proto3";
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

	writeFile(t, filepath.Join(inputDir, "prec.proto"), `syntax = "proto3";
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

	writeFile(t, filepath.Join(inputDir, "combo.proto"), `syntax = "proto3";
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
	if !errors.Is(err, fs.ErrNotExist) {
		t.Errorf("expected fs.ErrNotExist, got %T: %v", err, err)
	}
}

func TestRun_ReadOnlyOutputDir(t *testing.T) {
	t.Parallel()

	if os.Getuid() == 0 {
		t.Skip("cannot test permission errors as root")
	}

	inputDir := t.TempDir()
	outputDir := t.TempDir()

	writeFile(t, filepath.Join(inputDir, "plain.proto"), `syntax = "proto3";
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
	if !errors.Is(err, fs.ErrPermission) {
		t.Errorf("expected fs.ErrPermission, got %T: %v", err, err)
	}
}

func TestRun_EnumNameCollision(t *testing.T) {
	t.Parallel()

	inputDir := t.TempDir()
	outputDir := t.TempDir()

	// Two different proto packages define the same enum name "Status".
	// The proto compiler allows this (different packages), but gcode's
	// cross-file GoName index must detect the collision.
	writeFile(t, filepath.Join(inputDir, "a.proto"), `syntax = "proto3";
package pkg_a;
option go_package = "example.com/test;testpb";
enum Status { STATUS_UNKNOWN = 0; STATUS_ACTIVE = 1; }
message A { Status status = 1; }
`)
	writeFile(t, filepath.Join(inputDir, "b.proto"), `syntax = "proto3";
package pkg_b;
option go_package = "example.com/test;testpb";
enum Status { STATUS_UNKNOWN = 0; STATUS_INACTIVE = 1; }
message B { Status status = 1; }
`)

	err := Run(t.Context(), []string{"-in", inputDir, "-out", outputDir})
	if err == nil {
		t.Fatal("expected error for enum name collision, got nil")
	}
	var ae AppError
	if !errors.As(err, &ae) {
		t.Errorf("expected AppError domain type, got %T: %v", err, err)
	}
	entries, readErr := os.ReadDir(outputDir)
	if readErr != nil {
		t.Fatalf("ReadDir: %v", readErr)
	}
	if len(entries) != 0 {
		t.Errorf("expected empty output dir on collision, got %d entries", len(entries))
	}
}

func TestRun_EmptyDirectory(t *testing.T) {
	t.Parallel()

	inputDir := t.TempDir()
	outputDir := t.TempDir()

	err := Run(t.Context(), []string{"-in", inputDir, "-out", outputDir})
	if err == nil {
		t.Fatal("expected error for empty directory, got nil")
	}
	if !errors.Is(err, ErrNoProtoFiles) {
		t.Errorf("expected ErrNoProtoFiles, got %T: %v", err, err)
	}
}

// TestRun_MessageEnumNameCollision verifies that a message and an enum with the
// same Go name across different proto files are detected and rejected with an
// AppError before any output is written.
func TestRun_MessageEnumNameCollision(t *testing.T) {
	t.Parallel()

	inputDir := t.TempDir()
	outputDir := t.TempDir()

	writeFile(t, filepath.Join(inputDir, "a.proto"), `syntax = "proto3";
package pkg_a;
option go_package = "example.com/test;testpb";
message Status { int32 code = 1; }
`)
	writeFile(t, filepath.Join(inputDir, "b.proto"), `syntax = "proto3";
package pkg_b;
option go_package = "example.com/test;testpb";
enum Status { STATUS_UNKNOWN = 0; STATUS_ACTIVE = 1; }
message B { Status status = 1; }
`)

	err := Run(t.Context(), []string{"-in", inputDir, "-out", outputDir})
	if err == nil {
		t.Fatal("expected error for message/enum name collision, got nil")
	}
	var ae AppError
	if !errors.As(err, &ae) {
		t.Errorf("expected AppError domain type, got %T: %v", err, err)
	}
	// Collision is detected before render; output directory must be empty.
	entries, readErr := os.ReadDir(outputDir)
	if readErr != nil {
		t.Fatalf("ReadDir: %v", readErr)
	}
	if len(entries) != 0 {
		t.Errorf("expected empty output dir on collision, got %d entries", len(entries))
	}
}

// TestRun_MessageNameCollision verifies that two proto files defining a message
// with the same Go name are detected and rejected with an AppError.
func TestRun_MessageNameCollision(t *testing.T) {
	t.Parallel()

	inputDir := t.TempDir()
	outputDir := t.TempDir()

	writeFile(t, filepath.Join(inputDir, "a.proto"), `syntax = "proto3";
package pkg_a;
option go_package = "example.com/test;testpb";
message User { string name = 1; }
`)
	writeFile(t, filepath.Join(inputDir, "b.proto"), `syntax = "proto3";
package pkg_b;
option go_package = "example.com/test;testpb";
message User { int32 id = 1; }
`)

	err := Run(t.Context(), []string{"-in", inputDir, "-out", outputDir})
	if err == nil {
		t.Fatal("expected error for message name collision, got nil")
	}
	var ae AppError
	if !errors.As(err, &ae) {
		t.Errorf("expected AppError domain type, got %T: %v", err, err)
	}
	entries, readErr := os.ReadDir(outputDir)
	if readErr != nil {
		t.Fatalf("ReadDir: %v", readErr)
	}
	if len(entries) != 0 {
		t.Errorf("expected empty output dir on collision, got %d entries", len(entries))
	}
}

// TestRun_SubdirectorySameBasenameCollision verifies that Go generation keeps a
// flat output directory and rejects same-basename proto files before writing.
func TestRun_SubdirectorySameBasenameCollision(t *testing.T) {
	t.Parallel()

	inputDir := t.TempDir()
	outputDir := t.TempDir()

	sub1 := filepath.Join(inputDir, "sub1")
	sub2 := filepath.Join(inputDir, "sub2")
	if err := os.MkdirAll(sub1, 0o755); err != nil {
		t.Fatalf("mkdir sub1: %v", err)
	}
	if err := os.MkdirAll(sub2, 0o755); err != nil {
		t.Fatalf("mkdir sub2: %v", err)
	}

	writeFile(t, filepath.Join(sub1, "user.proto"), `syntax = "proto3";
package pkg_a;
option go_package = "example.com/test;testpb";
message UserA { string name = 1; }
`)
	writeFile(t, filepath.Join(sub2, "user.proto"), `syntax = "proto3";
package pkg_b;
option go_package = "example.com/test;testpb";
message UserB { int32 id = 1; }
`)

	err := Run(t.Context(), []string{"-in", inputDir, "-out", outputDir})
	if err == nil {
		t.Fatal("expected output filename collision, got nil")
	}
	if !errors.Is(err, ErrOutputFilenameCollision) {
		t.Errorf("expected ErrOutputFilenameCollision, got %T: %v", err, err)
	}
	entries, readErr := os.ReadDir(outputDir)
	if readErr != nil {
		t.Fatalf("ReadDir: %v", readErr)
	}
	if len(entries) != 0 {
		t.Errorf("expected empty output dir on collision, got %d entries", len(entries))
	}
}

func TestRun_SubdirectoryCrossFileReferenceCompiles(t *testing.T) {
	t.Parallel()

	inputDir := t.TempDir()
	outputDir := t.TempDir()

	commonDir := filepath.Join(inputDir, "common")
	userDir := filepath.Join(inputDir, "user")
	if err := os.MkdirAll(commonDir, 0o755); err != nil {
		t.Fatalf("mkdir common: %v", err)
	}
	if err := os.MkdirAll(userDir, 0o755); err != nil {
		t.Fatalf("mkdir user: %v", err)
	}

	writeFile(t, filepath.Join(commonDir, "common.proto"), `syntax = "proto3";
package demo;
option go_package = "example.com/demo;demo";
message Address { string city = 1; }
`)
	writeFile(t, filepath.Join(userDir, "user.proto"), `syntax = "proto3";
package demo;
option go_package = "example.com/demo;demo";
import "common/common.proto";
message User { Address address = 1; }
`)

	if err := Run(t.Context(), []string{"-in", inputDir, "-out", outputDir}); err != nil {
		t.Fatalf("Run() error: %v", err)
	}
	for _, name := range []string{"common.pb.dao.go", "common.pb.dao.validate.go", "user.pb.dao.go", "user.pb.dao.validate.go"} {
		if _, err := os.Stat(filepath.Join(outputDir, name)); err != nil {
			t.Fatalf("expected generated file %q: %v", name, err)
		}
	}
	if _, err := os.Stat(filepath.Join(outputDir, "common", "common.pb.dao.go")); !errors.Is(err, fs.ErrNotExist) {
		t.Fatalf("unexpected nested Go output file, stat error = %v", err)
	}

	moduleRoot, err := filepath.Abs("../..")
	if err != nil {
		t.Fatalf("module root: %v", err)
	}
	writeFile(t, filepath.Join(outputDir, "go.mod"), `module example.com/demo

go 1.23

require github.com/pinealctx/gcode v0.0.0

replace github.com/pinealctx/gcode => `+filepath.ToSlash(moduleRoot)+`
`)

	for _, args := range [][]string{{"mod", "tidy"}, {"test", "./..."}} {
		cmd := exec.CommandContext(t.Context(), "go", args...)
		cmd.Dir = outputDir
		cmd.Env = append(os.Environ(), "GOWORK=off")
		out, err := cmd.CombinedOutput()
		if err != nil {
			t.Fatalf("go %s failed: %v\n%s", strings.Join(args, " "), err, out)
		}
	}
}
