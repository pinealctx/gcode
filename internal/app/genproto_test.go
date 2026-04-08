package app

import (
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/pinealctx/gcode/internal/config"
	"github.com/pinealctx/gcode/internal/model"
	"github.com/pinealctx/gcode/internal/parser"
	"github.com/pinealctx/gcode/internal/source"
)

// writeFile writes content to path, creating parent dirs as needed.
func writeFile(t *testing.T, path, content string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatalf("mkdir %q: %v", filepath.Dir(path), err)
	}
	if err := os.WriteFile(path, []byte(content), 0o600); err != nil {
		t.Fatalf("write %q: %v", path, err)
	}
}

// compileProtoDir verifies that all .proto files in dir can be compiled by protocompile.
func compileProtoDir(t *testing.T, dir string) {
	t.Helper()
	entries, err := os.ReadDir(dir)
	if err != nil {
		t.Fatalf("read dir %q: %v", dir, err)
	}
	var files []string
	for _, e := range entries {
		if strings.HasSuffix(e.Name(), ".proto") {
			files = append(files, e.Name())
		}
	}
	if len(files) == 0 {
		return
	}
	_, err = parser.Parse(t.Context(), []string{dir}, files)
	if err != nil {
		t.Errorf("protocompile failed on generated protos: %v", err)
	}
}

func TestRunGenProto_UpdateMessage(t *testing.T) {
	t.Parallel()

	inDir := t.TempDir()

	writeFile(t, filepath.Join(inDir, "user.proto"), `syntax = "proto3";
package test.user;
import "gcode/options.proto";

message User {
  int64           id         = 1;
  string          name       = 2;
  optional string email      = 3;
  int64           created_at = 4;

  option (gcode.update_message) = {
    name: "UserUpdateByID"
    condition_fields: ["id"]
    ignore_fields: ["created_at"]
  };
}
`)

	if err := RunGenProto(t.Context(), []string{"-in", inDir}); err != nil {
		t.Fatalf("RunGenProto returned error: %v", err)
	}

	// Update proto should be generated in the same directory.
	updatePath := filepath.Join(inDir, "user.update.proto")
	content, err := os.ReadFile(updatePath)
	if err != nil {
		t.Fatalf("user.update.proto not generated: %v", err)
	}
	s := string(content)

	if !strings.Contains(s, "message UserUpdateByID") {
		t.Errorf("missing message UserUpdateByID in:\n%s", s)
	}
	if !strings.Contains(s, `(gcode.update_source) = "User"`) {
		t.Errorf("missing update_source annotation in:\n%s", s)
	}
	// id is condition field → non-optional
	if !strings.Contains(s, "int64 id = 1") {
		t.Errorf("id should be non-optional in:\n%s", s)
	}
	// name is optional
	if !strings.Contains(s, "optional string name") {
		t.Errorf("name should be optional in:\n%s", s)
	}
	// created_at is ignored
	if strings.Contains(s, "created_at") {
		t.Errorf("created_at should be ignored in:\n%s", s)
	}

	// No create proto should be generated.
	if _, err := os.Stat(filepath.Join(inDir, "user.create.proto")); err == nil {
		t.Errorf("user.create.proto should not be generated")
	}

	compileProtoDir(t, inDir)
}

func TestRunGenProto_CreateMessage(t *testing.T) {
	t.Parallel()

	inDir := t.TempDir()

	writeFile(t, filepath.Join(inDir, "product.proto"), `syntax = "proto3";
package test.product;
import "gcode/options.proto";

message Product {
  int64           id         = 1;
  string          title      = 2;
  optional string sku        = 3;
  int64           created_at = 4;

  option (gcode.create_message) = {
    name: "ProductCreate"
    ignore_fields: ["id", "created_at"]
    required_fields: ["sku"]
  };
}
`)

	if err := RunGenProto(t.Context(), []string{"-in", inDir}); err != nil {
		t.Fatalf("RunGenProto returned error: %v", err)
	}

	createPath := filepath.Join(inDir, "product.create.proto")
	content, err := os.ReadFile(createPath)
	if err != nil {
		t.Fatalf("product.create.proto not generated: %v", err)
	}
	s := string(content)

	if !strings.Contains(s, "message ProductCreate") {
		t.Errorf("missing message ProductCreate in:\n%s", s)
	}
	if !strings.Contains(s, `(gcode.create_source) = "Product"`) {
		t.Errorf("missing create_source annotation in:\n%s", s)
	}
	// sku is required_fields → non-optional (should NOT have "optional" prefix)
	if strings.Contains(s, "optional string sku") {
		t.Errorf("sku should be non-optional in:\n%s", s)
	}
	if !strings.Contains(s, "string sku") {
		t.Errorf("sku field missing in:\n%s", s)
	}
	// title is optional
	if !strings.Contains(s, "optional string title") {
		t.Errorf("title should be optional in:\n%s", s)
	}
	// id and created_at are ignored
	if strings.Contains(s, "id") || strings.Contains(s, "created_at") {
		t.Errorf("id/created_at should be ignored in:\n%s", s)
	}

	compileProtoDir(t, inDir)
}

func TestRunGenProto_MultipleOptions(t *testing.T) {
	t.Parallel()

	inDir := t.TempDir()

	writeFile(t, filepath.Join(inDir, "user.proto"), `syntax = "proto3";
package test.user;
import "gcode/options.proto";

message User {
  int64           id    = 1;
  string          name  = 2;
  optional string email = 3;

  option (gcode.update_message) = {
    name: "UserUpdateByID"
    condition_fields: ["id"]
  };
  option (gcode.update_message) = {
    name: "UserUpdateByEmail"
    condition_fields: ["email"]
  };
  option (gcode.create_message) = {
    name: "UserCreate"
    ignore_fields: ["id"]
    required_fields: ["email"]
  };
}
`)

	if err := RunGenProto(t.Context(), []string{"-in", inDir}); err != nil {
		t.Fatalf("RunGenProto returned error: %v", err)
	}

	updateContent, err := os.ReadFile(filepath.Join(inDir, "user.update.proto"))
	if err != nil {
		t.Fatalf("user.update.proto not generated: %v", err)
	}
	us := string(updateContent)
	if !strings.Contains(us, "message UserUpdateByID") {
		t.Errorf("missing UserUpdateByID in update proto")
	}
	if !strings.Contains(us, "message UserUpdateByEmail") {
		t.Errorf("missing UserUpdateByEmail in update proto")
	}

	createContent, err := os.ReadFile(filepath.Join(inDir, "user.create.proto"))
	if err != nil {
		t.Fatalf("user.create.proto not generated: %v", err)
	}
	if !strings.Contains(string(createContent), "message UserCreate") {
		t.Errorf("missing UserCreate in create proto")
	}

	compileProtoDir(t, inDir)
}

func TestRunGenProto_NoOptions(t *testing.T) {
	t.Parallel()

	inDir := t.TempDir()

	writeFile(t, filepath.Join(inDir, "plain.proto"), `syntax = "proto3";
package test;

message Plain {
  string name = 1;
}
`)

	if err := RunGenProto(t.Context(), []string{"-in", inDir}); err != nil {
		t.Fatalf("RunGenProto returned error: %v", err)
	}

	// No intermediate protos generated.
	if _, err := os.Stat(filepath.Join(inDir, "plain.update.proto")); err == nil {
		t.Errorf("plain.update.proto should not be generated")
	}
	if _, err := os.Stat(filepath.Join(inDir, "plain.create.proto")); err == nil {
		t.Errorf("plain.create.proto should not be generated")
	}
}

func TestRunGenProto_MessageTypeFieldRejected(t *testing.T) {
	t.Parallel()

	inDir := t.TempDir()

	writeFile(t, filepath.Join(inDir, "order.proto"), `syntax = "proto3";
package test;
import "gcode/options.proto";

message Address {
  string city = 1;
}

message Order {
  int64   id      = 1;
  Address address = 2;

  option (gcode.update_message) = {
    name: "OrderUpdate"
    condition_fields: ["id"]
  };
}
`)

	err := RunGenProto(t.Context(), []string{"-in", inDir})
	if err == nil {
		t.Fatal("expected error for message-type field, got nil")
	}
	if !strings.Contains(err.Error(), "message-type fields are not allowed") {
		t.Errorf("error = %q, want to contain 'message-type fields are not allowed'", err.Error())
	}
}

func TestRunGenProto_EnumField(t *testing.T) {
	t.Parallel()

	inDir := t.TempDir()

	writeFile(t, filepath.Join(inDir, "item.proto"), `syntax = "proto3";
package test.item;
import "gcode/options.proto";

enum Status {
  STATUS_UNSPECIFIED = 0;
  STATUS_ACTIVE = 1;
}

message Item {
  int64  id     = 1;
  string title  = 2;
  Status status = 3;

  option (gcode.update_message) = {
    name: "ItemUpdate"
    condition_fields: ["id"]
  };
}
`)

	if err := RunGenProto(t.Context(), []string{"-in", inDir}); err != nil {
		t.Fatalf("RunGenProto returned error: %v", err)
	}

	content, err := os.ReadFile(filepath.Join(inDir, "item.update.proto"))
	if err != nil {
		t.Fatalf("item.update.proto not generated: %v", err)
	}
	s := string(content)

	// Enum field should appear as optional Status.
	if !strings.Contains(s, "optional Status status") {
		t.Errorf("enum field should be optional Status in:\n%s", s)
	}
	// Original proto should be imported.
	if !strings.Contains(s, `import "item.proto"`) {
		t.Errorf("missing import of item.proto in:\n%s", s)
	}

	compileProtoDir(t, inDir)
}

func TestRunGenProto_RejectsOutFlag(t *testing.T) {
	t.Parallel()

	inDir := t.TempDir()
	writeFile(t, filepath.Join(inDir, "plain.proto"), `syntax = "proto3";
package test;
message Plain { string name = 1; }
`)

	err := RunGenProto(t.Context(), []string{"-in", inDir, "-out", "somewhere"})
	if err == nil {
		t.Fatal("expected error for unknown -out flag, got nil")
	}
	if !strings.Contains(err.Error(), "out") {
		t.Errorf("error = %q, want to mention 'out'", err.Error())
	}
}

func TestRunGenProto_MissingInFlag(t *testing.T) {
	t.Parallel()

	err := RunGenProto(t.Context(), []string{})
	if err == nil {
		t.Fatal("expected error for missing -in flag, got nil")
	}
	if !errors.Is(err, config.ErrMissingProtoInputDir) {
		t.Errorf("expected config.ErrMissingProtoInputDir, got %T: %v", err, err)
	}
}

func TestRunGenProto_EmptyDirectory(t *testing.T) {
	t.Parallel()

	inDir := t.TempDir()

	err := RunGenProto(t.Context(), []string{"-in", inDir})
	if err == nil {
		t.Fatal("expected error for empty directory, got nil")
	}
	if !errors.Is(err, source.ErrNoProtoFiles) {
		t.Errorf("expected source.ErrNoProtoFiles, got %T: %v", err, err)
	}
}

func TestRunGenProto_BackwardCompatible(t *testing.T) {
	t.Parallel()

	// Verify that the existing gcode pipeline (Run, not RunGenProto) is unaffected.
	inDir := t.TempDir()
	outDir := t.TempDir()

	writeFile(t, filepath.Join(inDir, "plain.proto"), `syntax = "proto3";
package test;
option go_package = "example.com/test;testpb";

message Plain {
  string name = 1;
  int32  age  = 2;
}
`)

	if err := Run(t.Context(), []string{"-in", inDir, "-out", outDir}); err != nil {
		t.Fatalf("Run (existing pipeline) returned error: %v", err)
	}

	if _, err := os.Stat(filepath.Join(outDir, "plain.pb.dao.go")); err != nil {
		t.Errorf("plain.pb.dao.go not generated: %v", err)
	}
}

func TestProtoFieldLine_RepeatedField(t *testing.T) {
	t.Parallel()

	f := model.Field{
		Name:        "tags",
		Cardinality: model.CardinalityRepeated,
		Type:        model.FieldType{Kind: model.FieldKindScalar, Scalar: model.ScalarString},
	}
	line, err := protoFieldLine(f, true, 1)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if line != "repeated string tags = 1;" {
		t.Errorf("got %q, want %q", line, "repeated string tags = 1;")
	}
}

func TestProtoFieldLine_MessageTypeRejected(t *testing.T) {
	t.Parallel()

	f := model.Field{
		Name:        "addr",
		Cardinality: model.CardinalitySingular,
		Type:        model.FieldType{Kind: model.FieldKindMessage, Name: "Address"},
	}
	_, err := protoFieldLine(f, true, 1)
	if err == nil {
		t.Fatal("expected error for message-type field")
	}
	if !strings.Contains(err.Error(), "message-type fields are not allowed") {
		t.Errorf("error = %q", err.Error())
	}
}

func TestProtoFieldLine_NonOptionalScalar(t *testing.T) {
	t.Parallel()

	f := model.Field{
		Name:        "id",
		Cardinality: model.CardinalitySingular,
		Type:        model.FieldType{Kind: model.FieldKindScalar, Scalar: model.ScalarInt64},
	}
	line, err := protoFieldLine(f, false, 1)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if line != "int64 id = 1;" {
		t.Errorf("got %q, want %q", line, "int64 id = 1;")
	}
}

func TestProtoBaseName_Subdirectory(t *testing.T) {
	t.Parallel()

	got := protoBaseName("subdir/user.proto")
	if got != "user" {
		t.Errorf("got %q, want %q", got, "user")
	}
}

func TestRunGenProto_NonExistentDirectory(t *testing.T) {
	t.Parallel()

	err := RunGenProto(t.Context(), []string{"-in", "/nonexistent/path/that/does/not/exist"})
	if err == nil {
		t.Fatal("expected error for non-existent directory, got nil")
	}
}

func TestRunGenProto_StaleIntermediateProtosCleaned(t *testing.T) {
	t.Parallel()

	inDir := t.TempDir()

	// Write a proto with update_message option.
	writeFile(t, filepath.Join(inDir, "user.proto"), `syntax = "proto3";
package test.user;
import "gcode/options.proto";

message User {
  int64  id   = 1;
  string name = 2;

  option (gcode.update_message) = {
    name: "UserUpdateByID"
    condition_fields: ["id"]
  };
}
`)

	// First run: generates user.update.proto.
	if err := RunGenProto(t.Context(), []string{"-in", inDir}); err != nil {
		t.Fatalf("first RunGenProto: %v", err)
	}
	if _, err := os.Stat(filepath.Join(inDir, "user.update.proto")); err != nil {
		t.Fatalf("user.update.proto not generated on first run: %v", err)
	}

	// Overwrite with a proto that has no options.
	writeFile(t, filepath.Join(inDir, "user.proto"), `syntax = "proto3";
package test.user;

message User {
  int64  id   = 1;
  string name = 2;
}
`)

	// Second run: stale user.update.proto must be removed.
	if err := RunGenProto(t.Context(), []string{"-in", inDir}); err != nil {
		t.Fatalf("second RunGenProto: %v", err)
	}
	if _, err := os.Stat(filepath.Join(inDir, "user.update.proto")); err == nil {
		t.Errorf("user.update.proto should have been cleaned up after option was removed")
	}
}

func TestRunGenProto_StaleCreateProtosCleaned(t *testing.T) {
	t.Parallel()

	inDir := t.TempDir()

	// Write a proto with create_message option.
	writeFile(t, filepath.Join(inDir, "product.proto"), `syntax = "proto3";
package test.product;
import "gcode/options.proto";

message Product {
  int64  id    = 1;
  string title = 2;

  option (gcode.create_message) = {
    name: "ProductCreate"
    ignore_fields: ["id"]
  };
}
`)

	// First run: generates product.create.proto.
	if err := RunGenProto(t.Context(), []string{"-in", inDir}); err != nil {
		t.Fatalf("first RunGenProto: %v", err)
	}
	if _, err := os.Stat(filepath.Join(inDir, "product.create.proto")); err != nil {
		t.Fatalf("product.create.proto not generated on first run: %v", err)
	}

	// Overwrite with a proto that has no options.
	writeFile(t, filepath.Join(inDir, "product.proto"), `syntax = "proto3";
package test.product;

message Product {
  int64  id    = 1;
  string title = 2;
}
`)

	// Second run: stale product.create.proto must be removed.
	if err := RunGenProto(t.Context(), []string{"-in", inDir}); err != nil {
		t.Fatalf("second RunGenProto: %v", err)
	}
	if _, err := os.Stat(filepath.Join(inDir, "product.create.proto")); err == nil {
		t.Errorf("product.create.proto should have been cleaned up after option was removed")
	}
}

func TestRunGenProto_StaleCleanupPartial(t *testing.T) {
	t.Parallel()

	inDir := t.TempDir()

	// First run: proto has both update and create options.
	writeFile(t, filepath.Join(inDir, "user.proto"), `syntax = "proto3";
package test.user;
import "gcode/options.proto";

message User {
  int64  id    = 1;
  string name  = 2;
  string email = 3;

  option (gcode.update_message) = {
    name: "UserUpdateByID"
    condition_fields: ["id"]
  };
  option (gcode.create_message) = {
    name: "UserCreate"
    ignore_fields: ["id"]
  };
}
`)
	if err := RunGenProto(t.Context(), []string{"-in", inDir}); err != nil {
		t.Fatalf("first RunGenProto: %v", err)
	}
	if _, err := os.Stat(filepath.Join(inDir, "user.update.proto")); err != nil {
		t.Fatalf("user.update.proto not generated: %v", err)
	}
	if _, err := os.Stat(filepath.Join(inDir, "user.create.proto")); err != nil {
		t.Fatalf("user.create.proto not generated: %v", err)
	}

	// Second run: remove only create_message option, keep update_message.
	writeFile(t, filepath.Join(inDir, "user.proto"), `syntax = "proto3";
package test.user;
import "gcode/options.proto";

message User {
  int64  id    = 1;
  string name  = 2;
  string email = 3;

  option (gcode.update_message) = {
    name: "UserUpdateByID"
    condition_fields: ["id"]
  };
}
`)
	if err := RunGenProto(t.Context(), []string{"-in", inDir}); err != nil {
		t.Fatalf("second RunGenProto: %v", err)
	}
	// update proto must still exist.
	if _, err := os.Stat(filepath.Join(inDir, "user.update.proto")); err != nil {
		t.Errorf("user.update.proto should still exist: %v", err)
	}
	// create proto must be cleaned up.
	if _, err := os.Stat(filepath.Join(inDir, "user.create.proto")); err == nil {
		t.Errorf("user.create.proto should have been cleaned up after create_message was removed")
	}
}

func TestRunGenProto_RemoveFailureReturnsError(t *testing.T) {
	t.Parallel()

	if os.Getuid() == 0 {
		t.Skip("cannot test permission errors as root")
	}

	inDir := t.TempDir()

	// First run: generate user.update.proto.
	writeFile(t, filepath.Join(inDir, "user.proto"), `syntax = "proto3";
package test.user;
import "gcode/options.proto";

message User {
  int64  id   = 1;
  string name = 2;

  option (gcode.update_message) = {
    name: "UserUpdateByID"
    condition_fields: ["id"]
  };
}
`)
	if err := RunGenProto(t.Context(), []string{"-in", inDir}); err != nil {
		t.Fatalf("first RunGenProto: %v", err)
	}

	// Make the generated file read-only so os.Remove fails.
	updatePath := filepath.Join(inDir, "user.update.proto")
	if err := os.Chmod(updatePath, 0o444); err != nil {
		t.Fatalf("chmod: %v", err)
	}
	// Also make the directory read-only to prevent removal.
	if err := os.Chmod(inDir, 0o555); err != nil {
		t.Fatalf("chmod dir: %v", err)
	}
	t.Cleanup(func() {
		_ = os.Chmod(inDir, 0o755)
		_ = os.Chmod(updatePath, 0o644)
	})

	// Second run must fail because os.Remove is blocked.
	err := RunGenProto(t.Context(), []string{"-in", inDir})
	if err == nil {
		t.Fatal("expected error when os.Remove fails, got nil")
	}
	if !strings.Contains(err.Error(), "remove stale") {
		t.Errorf("error = %q, want to contain 'remove stale'", err.Error())
	}
}

func TestBuildUpdateMessage_EmptyName(t *testing.T) {
	t.Parallel()

	msg := model.Message{FullName: "pkg.Person"}
	opt := model.UpdateMessageOptions{Name: ""}
	_, err := buildUpdateMessage(msg, opt)
	if err == nil {
		t.Fatal("expected error for empty opt.Name, got nil")
	}
	if !strings.Contains(err.Error(), "name must not be empty") {
		t.Errorf("error = %q, want to contain 'name must not be empty'", err.Error())
	}
}

func TestBuildCreateMessage_EmptyName(t *testing.T) {
	t.Parallel()

	msg := model.Message{FullName: "pkg.Person"}
	opt := model.CreateMessageOptions{Name: ""}
	_, err := buildCreateMessage(msg, opt)
	if err == nil {
		t.Fatal("expected error for empty opt.Name, got nil")
	}
	if !strings.Contains(err.Error(), "name must not be empty") {
		t.Errorf("error = %q, want to contain 'name must not be empty'", err.Error())
	}
}
