package parser

import (
	"errors"
	"os"
	"path/filepath"
	"testing"

	"github.com/bufbuild/protocompile"
	"github.com/pinealctx/gcode/internal/model"
)

func TestParseMapsSemanticModel(t *testing.T) {
	t.Parallel()

	workspace := t.TempDir()
	writeProtoFile(t, workspace, "common.proto", `syntax = "proto3";

package test.common;

// Shared message comment.
message Shared {
	string id = 1;
}
`)
	writeProtoFile(t, workspace, "person.proto", `syntax = "proto3";

package test.foo;

option go_package = "example.com/test/foo;foopb";

// import comment.
import "common.proto";

// Person comment.
message Person {
	// name comment.
	string name = 1;

	// repeated scalar comment.
	repeated int32 lucky_numbers = 2;

	// enum field comment.
	Status status = 3;

	// imported message comment.
	test.common.Shared shared = 4;

	// nested message comment.
	Address address = 5;

	// Status comment.
	enum Status {
		STATUS_UNSPECIFIED = 0;
		// enum value comment.
		STATUS_ACTIVE = 1;
	}

	// Address comment.
	message Address {
		string city = 1;
	}
}
`)

	files, err := Parse(t.Context(), []string{workspace}, []string{"person.proto"})
	if err != nil {
		t.Fatalf("Parse returned error: %v", err)
	}
	if len(files) != 1 {
		t.Fatalf("expected 1 file, got %d", len(files))
	}

	file := files[0]
	if file.Path != "person.proto" {
		t.Fatalf("unexpected file path: %q", file.Path)
	}
	if file.Syntax != model.SyntaxProto3 {
		t.Fatalf("unexpected syntax: %q", file.Syntax)
	}
	if file.Package != "test.foo" {
		t.Fatalf("unexpected package: %q", file.Package)
	}
	if file.GoPackage != "example.com/test/foo;foopb" {
		t.Fatalf("unexpected go package: %q", file.GoPackage)
	}
	if len(file.Imports) != 1 {
		t.Fatalf("expected 1 import, got %d", len(file.Imports))
	}
	if file.Imports[0].Path != "common.proto" {
		t.Fatalf("unexpected import path: %q", file.Imports[0].Path)
	}
	if len(file.Imports[0].LeadingComment.Lines) != 1 || file.Imports[0].LeadingComment.Lines[0] != "import comment." {
		t.Fatalf("unexpected import comment: %#v", file.Imports[0].LeadingComment.Lines)
	}

	if len(file.Messages) != 1 {
		t.Fatalf("expected 1 top-level message, got %d", len(file.Messages))
	}

	person := file.Messages[0]
	if person.FullName != "test.foo.Person" {
		t.Fatalf("unexpected message full name: %q", person.FullName)
	}
	if len(person.Fields) != 5 {
		t.Fatalf("expected 5 fields, got %d", len(person.Fields))
	}
	if len(person.Messages) != 1 {
		t.Fatalf("expected 1 nested message, got %d", len(person.Messages))
	}
	if len(person.Enums) != 1 {
		t.Fatalf("expected 1 nested enum, got %d", len(person.Enums))
	}

	statusField := person.Fields[2]
	if statusField.Type.Kind != model.FieldKindEnum || statusField.Type.FullName != "test.foo.Person.Status" {
		t.Fatalf("unexpected enum field type: %#v", statusField.Type)
	}

	sharedField := person.Fields[3]
	if sharedField.Type.Kind != model.FieldKindMessage || sharedField.Type.FullName != "test.common.Shared" {
		t.Fatalf("unexpected imported message field type: %#v", sharedField.Type)
	}

	addressField := person.Fields[4]
	if addressField.Type.Kind != model.FieldKindMessage || addressField.Type.FullName != "test.foo.Person.Address" {
		t.Fatalf("unexpected nested message field type: %#v", addressField.Type)
	}

	repeatedField := person.Fields[1]
	if repeatedField.Cardinality != model.CardinalityRepeated || repeatedField.Type.Scalar != model.ScalarInt32 {
		t.Fatalf("unexpected repeated field mapping: %#v", repeatedField)
	}

	statusEnum := person.Enums[0]
	if statusEnum.FullName != "test.foo.Person.Status" {
		t.Fatalf("unexpected nested enum full name: %q", statusEnum.FullName)
	}
	if len(statusEnum.Values) != 2 {
		t.Fatalf("expected 2 enum values, got %d", len(statusEnum.Values))
	}
	if len(statusEnum.Values[1].LeadingComment.Lines) != 1 || statusEnum.Values[1].LeadingComment.Lines[0] != "enum value comment." {
		t.Fatalf("unexpected enum value comment: %#v", statusEnum.Values[1].LeadingComment.Lines)
	}

	addressMessage := person.Messages[0]
	if addressMessage.FullName != "test.foo.Person.Address" {
		t.Fatalf("unexpected nested message full name: %q", addressMessage.FullName)
	}
	if len(addressMessage.Fields) != 1 || addressMessage.Fields[0].Name != "city" {
		t.Fatalf("unexpected nested message fields: %#v", addressMessage.Fields)
	}
}

func TestParseRejectsProto2(t *testing.T) {
	t.Parallel()

	workspace := t.TempDir()
	writeProtoFile(t, workspace, "legacy.proto", `syntax = "proto2";

package test.legacy;

message Legacy {
	optional string name = 1;
}
`)

	_, err := Parse(t.Context(), []string{workspace}, []string{"legacy.proto"})
	if err == nil {
		t.Fatal("expected Parse to reject proto2 input")
	}
}

func TestParseService(t *testing.T) {
	t.Parallel()

	workspace := t.TempDir()
	writeProtoFile(t, workspace, "svc.proto", `syntax = "proto3";

package test.svc;

message Request {}
message Response {}

service Greeter {
	rpc SayHello (Request) returns (Response);
}
`)

	files, err := Parse(t.Context(), []string{workspace}, []string{"svc.proto"})
	if err != nil {
		t.Fatalf("Parse returned unexpected error: %v", err)
	}
	if len(files) != 1 {
		t.Fatalf("expected 1 file, got %d", len(files))
	}
	if len(files[0].Services) != 1 {
		t.Fatalf("expected 1 service, got %d", len(files[0].Services))
	}
	svc := files[0].Services[0]
	if svc.Name != "Greeter" {
		t.Errorf("service name = %q, want %q", svc.Name, "Greeter")
	}
	if svc.FullName != "test.svc.Greeter" {
		t.Errorf("service full name = %q, want %q", svc.FullName, "test.svc.Greeter")
	}
	if len(svc.RPCs) != 1 {
		t.Fatalf("expected 1 rpc, got %d", len(svc.RPCs))
	}
	rpc := svc.RPCs[0]
	if rpc.Name != "SayHello" {
		t.Errorf("rpc name = %q, want %q", rpc.Name, "SayHello")
	}
	if rpc.RequestType != "test.svc.Request" {
		t.Errorf("rpc request type = %q, want %q", rpc.RequestType, "test.svc.Request")
	}
	if rpc.ResponseType != "test.svc.Response" {
		t.Errorf("rpc response type = %q, want %q", rpc.ResponseType, "test.svc.Response")
	}
}

func TestParseServiceMultiple(t *testing.T) {
	t.Parallel()

	workspace := t.TempDir()
	writeProtoFile(t, workspace, "multi.proto", `syntax = "proto3";

package test.multi;

message AReq {}
message AResp {}
message BReq {}
message BResp {}

service ServiceA {
	rpc MethodA (AReq) returns (AResp);
}

service ServiceB {
	rpc MethodB1 (BReq) returns (BResp);
	rpc MethodB2 (BReq) returns (BResp);
}
`)

	files, err := Parse(t.Context(), []string{workspace}, []string{"multi.proto"})
	if err != nil {
		t.Fatalf("Parse returned unexpected error: %v", err)
	}
	if len(files[0].Services) != 2 {
		t.Fatalf("expected 2 services, got %d", len(files[0].Services))
	}
	if len(files[0].Services[1].RPCs) != 2 {
		t.Errorf("ServiceB expected 2 rpcs, got %d", len(files[0].Services[1].RPCs))
	}
}

func TestParseRejectsStreamingRPC(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name  string
		proto string
	}{
		{
			name: "server streaming",
			proto: `syntax = "proto3";
package test.stream;
message Req {}
message Resp {}
service S { rpc Watch (Req) returns (stream Resp); }`,
		},
		{
			name: "client streaming",
			proto: `syntax = "proto3";
package test.stream;
message Req {}
message Resp {}
service S { rpc Upload (stream Req) returns (Resp); }`,
		},
		{
			name: "bidi streaming",
			proto: `syntax = "proto3";
package test.stream;
message Req {}
message Resp {}
service S { rpc Chat (stream Req) returns (stream Resp); }`,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			workspace := t.TempDir()
			writeProtoFile(t, workspace, "stream.proto", tc.proto)
			_, err := Parse(t.Context(), []string{workspace}, []string{"stream.proto"})
			if err == nil {
				t.Fatal("expected Parse to reject streaming rpc")
			}
			var pe ParseError
			if !errors.As(err, &pe) {
				t.Errorf("expected ParseError domain type, got %T: %v", err, err)
			}
		})
	}
}

func TestParseNoServiceNoServicesField(t *testing.T) {
	t.Parallel()

	workspace := t.TempDir()
	writeProtoFile(t, workspace, "msg.proto", `syntax = "proto3";
package test.msg;
message Foo { string id = 1; }
`)

	files, err := Parse(t.Context(), []string{workspace}, []string{"msg.proto"})
	if err != nil {
		t.Fatalf("Parse returned unexpected error: %v", err)
	}
	if len(files[0].Services) != 0 {
		t.Errorf("expected no services, got %d", len(files[0].Services))
	}
}

func TestParseRejectsOneof(t *testing.T) {
	t.Parallel()

	workspace := t.TempDir()
	writeProtoFile(t, workspace, "oneof.proto", `syntax = "proto3";

package test.oneof;

message Event {
	oneof payload {
		string text = 1;
		int32 number = 2;
	}
}
`)

	_, err := Parse(t.Context(), []string{workspace}, []string{"oneof.proto"})
	if err == nil {
		t.Fatal("expected Parse to reject oneof fields")
	}
}

func TestParseRejectsMapField(t *testing.T) {
	t.Parallel()

	workspace := t.TempDir()
	writeProtoFile(t, workspace, "mapfield.proto", `syntax = "proto3";

package test.mapfield;

message Config {
	map<string, string> labels = 1;
}
`)

	_, err := Parse(t.Context(), []string{workspace}, []string{"mapfield.proto"})
	if err == nil {
		t.Fatal("expected Parse to reject map fields")
	}
}

func TestParseRejectsWellKnownType(t *testing.T) {
	t.Parallel()

	workspace := t.TempDir()
	writeProtoFile(t, workspace, "wkt.proto", `syntax = "proto3";

package test.wkt;

import "google/protobuf/timestamp.proto";

message Event {
	google.protobuf.Timestamp created_at = 1;
}
`)

	_, err := Parse(t.Context(), []string{workspace}, []string{"wkt.proto"})
	if err == nil {
		t.Fatal("expected Parse to reject well-known type references")
	}
}

func writeProtoFile(t *testing.T, dir string, name string, content string) {
	t.Helper()

	path := filepath.Join(dir, name)
	if err := os.WriteFile(path, []byte(content), 0o600); err != nil {
		t.Fatalf("write proto file %q: %v", path, err)
	}
}

func TestParseGcodeOptionsImport(t *testing.T) {
	t.Parallel()

	workspace := t.TempDir()
	writeProtoFile(t, workspace, "annotated.proto", `syntax = "proto3";

package test.annotated;

import "gcode/options.proto";

message User {
  option (gcode.message).gorm = { table: "users" };

  string email = 1 [(gcode.field).gorm.column = "email_address"];
  string phone = 2 [(gcode.field).json.omitempty = true];
  string token = 3 [(gcode.field).json.ignore = true];
  string name  = 4;
}

message Config {
  string key   = 1;
  string value = 2;
}
`)

	files, err := Parse(t.Context(), []string{workspace}, []string{"annotated.proto"})
	if err != nil {
		t.Fatalf("Parse returned error: %v", err)
	}
	if len(files) != 1 {
		t.Fatalf("expected 1 file, got %d", len(files))
	}

	msgs := files[0].Messages
	if len(msgs) != 2 {
		t.Fatalf("expected 2 messages, got %d", len(msgs))
	}

	// --- User: has gorm annotation ---
	user := msgs[0]
	if user.GormOptions == nil {
		t.Fatal("User.GormOptions should not be nil")
	}
	if user.GormOptions.Table != "users" {
		t.Errorf("User.GormOptions.Table = %q, want %q", user.GormOptions.Table, "users")
	}

	// email: gorm column override
	email := user.Fields[0]
	if email.GormOptions == nil || email.GormOptions.Column != "email_address" {
		t.Errorf("email.GormOptions = %+v, want column=email_address", email.GormOptions)
	}
	if email.JSONOptions != nil {
		t.Errorf("email.JSONOptions should be nil, got %+v", email.JSONOptions)
	}

	// phone: json omitempty
	phone := user.Fields[1]
	if phone.GormOptions != nil {
		t.Errorf("phone.GormOptions should be nil, got %+v", phone.GormOptions)
	}
	if phone.JSONOptions == nil || !phone.JSONOptions.Omitempty || phone.JSONOptions.Ignore {
		t.Errorf("phone.JSONOptions = %+v, want omitempty=true ignore=false", phone.JSONOptions)
	}

	// token: json ignore
	token := user.Fields[2]
	if token.JSONOptions == nil || !token.JSONOptions.Ignore {
		t.Errorf("token.JSONOptions = %+v, want ignore=true", token.JSONOptions)
	}

	// name: no annotations
	name := user.Fields[3]
	if name.GormOptions != nil || name.JSONOptions != nil {
		t.Errorf("name should have no annotations, got gorm=%+v json=%+v", name.GormOptions, name.JSONOptions)
	}

	// --- Config: no gorm annotation ---
	config := msgs[1]
	if config.GormOptions != nil {
		t.Errorf("Config.GormOptions should be nil, got %+v", config.GormOptions)
	}
}

func TestBuildGcodeExtensions(t *testing.T) {
	t.Parallel()

	exts, err := buildGcodeExtensions()
	if err != nil {
		t.Fatalf("buildGcodeExtensions returned error: %v", err)
	}
	if exts.messageExt == nil {
		t.Error("messageExt should not be nil")
	}
	if exts.fieldExt == nil {
		t.Error("fieldExt should not be nil")
	}
}

func TestParseOptionalFields(t *testing.T) {
	t.Parallel()

	workspace := t.TempDir()
	writeProtoFile(t, workspace, "optional.proto", `syntax = "proto3";

package test.optional;

enum Status {
  STATUS_UNSPECIFIED = 0;
  STATUS_ACTIVE = 1;
}

message Item {
  optional string  name   = 1;
  optional int32   count  = 2;
  optional bool    active = 3;
  optional Status  status = 4;
  optional bytes   data   = 5;
  string           label  = 6;
  repeated string  tags   = 7;
}
`)

	files, err := Parse(t.Context(), []string{workspace}, []string{"optional.proto"})
	if err != nil {
		t.Fatalf("Parse returned error: %v", err)
	}

	item := files[0].Messages[0]
	cases := []struct {
		name        string
		optional    bool
		hasPresence bool
	}{
		{"name", true, false},   // optional string → Optional=true
		{"count", true, false},  // optional int32  → Optional=true
		{"active", true, false}, // optional bool   → Optional=true
		{"status", true, false}, // optional enum   → Optional=true
		{"data", false, true},   // optional bytes  → Optional=false, HasPresence=true ([]byte, nil=absent)
		{"label", false, false}, // plain string    → Optional=false
		{"tags", false, false},  // repeated        → Optional=false
	}

	if len(item.Fields) != len(cases) {
		t.Fatalf("expected %d fields, got %d", len(cases), len(item.Fields))
	}
	for i, tc := range cases {
		f := item.Fields[i]
		if f.Name != tc.name {
			t.Errorf("field[%d]: name = %q, want %q", i, f.Name, tc.name)
		}
		if f.Optional != tc.optional {
			t.Errorf("field %q: Optional = %v, want %v", tc.name, f.Optional, tc.optional)
		}
		if f.HasPresence != tc.hasPresence {
			t.Errorf("field %q: HasPresence = %v, want %v", tc.name, f.HasPresence, tc.hasPresence)
		}
	}
}

func TestParseRejectsRealOneof(t *testing.T) {
	t.Parallel()

	workspace := t.TempDir()
	writeProtoFile(t, workspace, "realOneof.proto", `syntax = "proto3";

package test.realoneof;

message Event {
  oneof payload {
    string text   = 1;
    int32  number = 2;
  }
}
`)

	_, err := Parse(t.Context(), []string{workspace}, []string{"realOneof.proto"})
	if err == nil {
		t.Fatal("expected Parse to reject real oneof fields")
	}
}

func TestBuildValidateExtensions(t *testing.T) {
	t.Parallel()

	exts, err := buildValidateExtensions()
	if err != nil {
		t.Fatalf("buildValidateExtensions returned error: %v", err)
	}
	if exts.fieldExt == nil {
		t.Error("fieldExt should not be nil")
	}

	// Verify the extension targets google.protobuf.FieldOptions.
	extendee := exts.fieldExt.TypeDescriptor().ParentFile().FullName()
	if extendee == "" {
		t.Error("fieldExt parent file should not be empty")
	}
	if exts.fieldExt.TypeDescriptor().ContainingMessage().FullName() != "google.protobuf.FieldOptions" {
		t.Errorf("fieldExt extendee = %q, want google.protobuf.FieldOptions",
			exts.fieldExt.TypeDescriptor().ContainingMessage().FullName())
	}
	// Verify the extension number matches the buf.validate spec (field number 1159).
	if exts.fieldExt.TypeDescriptor().Number() != 1159 {
		t.Errorf("fieldExt number = %d, want 1159", exts.fieldExt.TypeDescriptor().Number())
	}
}

func TestEmbeddedResolverUnknownPathFallsThrough(t *testing.T) {
	t.Parallel()

	// An unknown path should be forwarded to the inner resolver and return an error
	// (since there's no file at that path in the standard imports).
	r := &embeddedResolver{
		inner: protocompile.WithStandardImports(&protocompile.SourceResolver{}),
	}
	_, err := r.FindFileByPath("nonexistent/unknown.proto")
	if err == nil {
		t.Error("expected error for unknown proto path, got nil")
	}
}

func TestEmbeddedResolverServesValidateProto(t *testing.T) {
	t.Parallel()

	// Verify that protocompile can compile a proto3 file that imports
	// buf/validate/validate.proto via the embeddedResolver.
	workspace := t.TempDir()
	writeProtoFile(t, workspace, "validated.proto", `syntax = "proto3";

package test.validated;

import "buf/validate/validate.proto";

message Person {
  string name = 1 [(buf.validate.field).string.min_len = 1];
  int32  age  = 2 [(buf.validate.field).int32.gte = 0];
}
`)

	compiler := protocompile.Compiler{
		Resolver: protocompile.WithStandardImports(&embeddedResolver{
			inner: &protocompile.SourceResolver{
				ImportPaths: []string{workspace},
			},
		}),
	}

	compiled, err := compiler.Compile(t.Context(), "validated.proto")
	if err != nil {
		t.Fatalf("compile proto with buf/validate import failed: %v", err)
	}
	if len(compiled) == 0 {
		t.Fatal("expected at least one compiled file")
	}
}

func TestParseValidateOptions_StringConstraints(t *testing.T) {
	t.Parallel()

	workspace := t.TempDir()
	writeProtoFile(t, workspace, "validate_str.proto", `syntax = "proto3";
package test.validate;
import "buf/validate/validate.proto";
import "gcode/options.proto";
message User {
  string email = 1 [
    (buf.validate.field).string.email = true,
    (gcode.field).validate_message = "invalid email"
  ];
  string name = 2 [
    (buf.validate.field).string.min_len = 1,
    (buf.validate.field).string.max_len = 100
  ];
  string code = 3 [(buf.validate.field).string.pattern = "^[A-Z]{3}$"];
  string status = 4 [(buf.validate.field).required = true];
  string kind = 5;
}
`)

	files, err := Parse(t.Context(), []string{workspace}, []string{"validate_str.proto"})
	if err != nil {
		t.Fatalf("Parse returned error: %v", err)
	}
	fields := files[0].Messages[0].Fields

	// email: Email=true, ValidateMessage set
	if fields[0].ValidateOptions == nil || !fields[0].ValidateOptions.Email {
		t.Errorf("email: expected Email=true, got %+v", fields[0].ValidateOptions)
	}
	if fields[0].ValidateMessage != "invalid email" {
		t.Errorf("email: ValidateMessage = %q, want %q", fields[0].ValidateMessage, "invalid email")
	}

	// name: MinLen=1, MaxLen=100
	if fields[1].ValidateOptions == nil {
		t.Fatal("name: ValidateOptions should not be nil")
	}
	if fields[1].ValidateOptions.MinLen == nil || *fields[1].ValidateOptions.MinLen != 1 {
		t.Errorf("name: MinLen = %v, want 1", fields[1].ValidateOptions.MinLen)
	}
	if fields[1].ValidateOptions.MaxLen == nil || *fields[1].ValidateOptions.MaxLen != 100 {
		t.Errorf("name: MaxLen = %v, want 100", fields[1].ValidateOptions.MaxLen)
	}

	// code: Pattern set
	if fields[2].ValidateOptions == nil || fields[2].ValidateOptions.Pattern != "^[A-Z]{3}$" {
		t.Errorf("code: Pattern = %q, want ^[A-Z]{3}$", fields[2].ValidateOptions.Pattern)
	}

	// status: Required=true
	if fields[3].ValidateOptions == nil || !fields[3].ValidateOptions.Required {
		t.Errorf("status: expected Required=true, got %+v", fields[3].ValidateOptions)
	}

	// kind: no annotations → nil
	if fields[4].ValidateOptions != nil {
		t.Errorf("kind: expected nil ValidateOptions, got %+v", fields[4].ValidateOptions)
	}
}

func TestParseValidateOptions_NumericConstraints(t *testing.T) {
	t.Parallel()

	workspace := t.TempDir()
	writeProtoFile(t, workspace, "validate_num.proto", `syntax = "proto3";
package test.validate;
import "buf/validate/validate.proto";
message Product {
  int32  age    = 1 [(buf.validate.field).int32.gte = 0, (buf.validate.field).int32.lte = 150];
  uint64 price  = 2 [(buf.validate.field).uint64.gt = 0];
  double score  = 3 [(buf.validate.field).double.gte = 0.0, (buf.validate.field).double.lte = 1.0];
}
`)

	files, err := Parse(t.Context(), []string{workspace}, []string{"validate_num.proto"})
	if err != nil {
		t.Fatalf("Parse returned error: %v", err)
	}
	fields := files[0].Messages[0].Fields

	// age: GTEInt=0, LTEInt=150
	if fields[0].ValidateOptions == nil {
		t.Fatal("age: ValidateOptions should not be nil")
	}
	if fields[0].ValidateOptions.GTEInt == nil || *fields[0].ValidateOptions.GTEInt != 0 {
		t.Errorf("age: GTEInt = %v, want 0", fields[0].ValidateOptions.GTEInt)
	}
	if fields[0].ValidateOptions.LTEInt == nil || *fields[0].ValidateOptions.LTEInt != 150 {
		t.Errorf("age: LTEInt = %v, want 150", fields[0].ValidateOptions.LTEInt)
	}

	// price: GTUint=0
	if fields[1].ValidateOptions == nil || fields[1].ValidateOptions.GTUint == nil || *fields[1].ValidateOptions.GTUint != 0 {
		t.Errorf("price: GTUint = %v, want 0", fields[1].ValidateOptions.GTUint)
	}

	// score: GTEFloat=0.0, LTEFloat=1.0
	if fields[2].ValidateOptions == nil {
		t.Fatal("score: ValidateOptions should not be nil")
	}
	if fields[2].ValidateOptions.GTEFloat == nil || *fields[2].ValidateOptions.GTEFloat != 0.0 {
		t.Errorf("score: GTEFloat = %v, want 0.0", fields[2].ValidateOptions.GTEFloat)
	}
	if fields[2].ValidateOptions.LTEFloat == nil || *fields[2].ValidateOptions.LTEFloat != 1.0 {
		t.Errorf("score: LTEFloat = %v, want 1.0", fields[2].ValidateOptions.LTEFloat)
	}
}

func TestParseValidateOptions_RepeatedConstraints(t *testing.T) {
	t.Parallel()

	workspace := t.TempDir()
	writeProtoFile(t, workspace, "validate_rep.proto", `syntax = "proto3";
package test.validate;
import "buf/validate/validate.proto";
message Order {
  repeated string tags = 1 [
    (buf.validate.field).repeated.min_items = 1,
    (buf.validate.field).repeated.max_items = 10,
    (buf.validate.field).repeated.items.string.min_len = 1
  ];
}
`)

	files, err := Parse(t.Context(), []string{workspace}, []string{"validate_rep.proto"})
	if err != nil {
		t.Fatalf("Parse returned error: %v", err)
	}
	f := files[0].Messages[0].Fields[0]
	if f.ValidateOptions == nil {
		t.Fatal("tags: ValidateOptions should not be nil")
	}
	if f.ValidateOptions.MinItems == nil || *f.ValidateOptions.MinItems != 1 {
		t.Errorf("tags: MinItems = %v, want 1", f.ValidateOptions.MinItems)
	}
	if f.ValidateOptions.MaxItems == nil || *f.ValidateOptions.MaxItems != 10 {
		t.Errorf("tags: MaxItems = %v, want 10", f.ValidateOptions.MaxItems)
	}
	if f.ValidateOptions.Items == nil {
		t.Fatal("tags: Items should not be nil")
	}
	if f.ValidateOptions.Items.MinLen == nil || *f.ValidateOptions.Items.MinLen != 1 {
		t.Errorf("tags.items: MinLen = %v, want 1", f.ValidateOptions.Items.MinLen)
	}
}

func TestParseValidateOptions_ConflictErrors(t *testing.T) {
	t.Parallel()

	// ParseError cases: errors that should be of type ParseError (domain-typed).
	parseCases := []struct {
		name  string
		proto string
	}{
		{
			name: "bool required",
			proto: `syntax = "proto3";
package test;
import "buf/validate/validate.proto";
message M { bool active = 1 [(buf.validate.field).required = true]; }`,
		},
		{
			name: "min_len > max_len",
			proto: `syntax = "proto3";
package test;
import "buf/validate/validate.proto";
message M { string name = 1 [(buf.validate.field).string.min_len = 10, (buf.validate.field).string.max_len = 5]; }`,
		},
		{
			name: "min_items > max_items",
			proto: `syntax = "proto3";
package test;
import "buf/validate/validate.proto";
message M { repeated string tags = 1 [(buf.validate.field).repeated.min_items = 5, (buf.validate.field).repeated.max_items = 2]; }`,
		},
		{
			name: "string in + required conflict",
			proto: `syntax = "proto3";
package test;
import "buf/validate/validate.proto";
message M { string s = 1 [(buf.validate.field).required = true, (buf.validate.field).string.in = "", (buf.validate.field).string.in = "active"]; }`,
		},
		{
			name: "repeated bytes items required",
			proto: `syntax = "proto3";
package test;
import "buf/validate/validate.proto";
message M { repeated bytes data = 1 [(buf.validate.field).repeated.items.required = true]; }`,
		},
		{
			name: "invalid pattern",
			proto: `syntax = "proto3";
package test;
import "buf/validate/validate.proto";
message M { string code = 1 [(buf.validate.field).string.pattern = "[invalid"]; }`,
		},
		{
			name: "bytes min_len > max_len",
			proto: `syntax = "proto3";
package test;
import "buf/validate/validate.proto";
message M { bytes b = 1 [(buf.validate.field).bytes.min_len = 10, (buf.validate.field).bytes.max_len = 5]; }`,
		},
	}

	for _, tc := range parseCases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			workspace := t.TempDir()
			writeProtoFile(t, workspace, "conflict.proto", tc.proto)
			_, err := Parse(t.Context(), []string{workspace}, []string{"conflict.proto"})
			if err == nil {
				t.Fatalf("expected ParseError, got nil")
			}
			var pe ParseError
			if !errors.As(err, &pe) {
				t.Errorf("expected ParseError domain type, got %T: %v", err, err)
			}
		})
	}
}

func TestParseValidateOptions_NoAnnotation(t *testing.T) {
	t.Parallel()

	workspace := t.TempDir()
	writeProtoFile(t, workspace, "no_validate.proto", `syntax = "proto3";
package test;
message Plain { string name = 1; int32 age = 2; }
`)

	files, err := Parse(t.Context(), []string{workspace}, []string{"no_validate.proto"})
	if err != nil {
		t.Fatalf("Parse returned error: %v", err)
	}
	for _, f := range files[0].Messages[0].Fields {
		if f.ValidateOptions != nil {
			t.Errorf("field %q: expected nil ValidateOptions, got %+v", f.Name, f.ValidateOptions)
		}
	}
}

func TestParseValidateOptions_InNotIn(t *testing.T) {
	t.Parallel()

	workspace := t.TempDir()
	writeProtoFile(t, workspace, "validate_in.proto", `syntax = "proto3";
package test.validate;
import "buf/validate/validate.proto";
message Filter {
  string status = 1 [
    (buf.validate.field).string.in = "active",
    (buf.validate.field).string.in = "inactive"
  ];
  string kind = 2 [
    (buf.validate.field).string.not_in = "deleted",
    (buf.validate.field).string.not_in = "archived"
  ];
  int32 level = 3 [
    (buf.validate.field).int32.in = 1,
    (buf.validate.field).int32.in = 2,
    (buf.validate.field).int32.in = 3
  ];
  int32 priority = 4 [
    (buf.validate.field).int32.not_in = 0
  ];
  uint64 code = 5 [
    (buf.validate.field).uint64.in = 100,
    (buf.validate.field).uint64.in = 200
  ];
  uint64 flags = 6 [
    (buf.validate.field).uint64.not_in = 0
  ];
}
`)

	files, err := Parse(t.Context(), []string{workspace}, []string{"validate_in.proto"})
	if err != nil {
		t.Fatalf("Parse returned error: %v", err)
	}
	fields := files[0].Messages[0].Fields

	// status: InStr = ["active", "inactive"]
	if fields[0].ValidateOptions == nil {
		t.Fatal("status: ValidateOptions should not be nil")
	}
	if len(fields[0].ValidateOptions.InStr) != 2 {
		t.Errorf("status: InStr len = %d, want 2", len(fields[0].ValidateOptions.InStr))
	}

	// kind: NotInStr = ["deleted", "archived"]
	if fields[1].ValidateOptions == nil {
		t.Fatal("kind: ValidateOptions should not be nil")
	}
	if len(fields[1].ValidateOptions.NotInStr) != 2 {
		t.Errorf("kind: NotInStr len = %d, want 2", len(fields[1].ValidateOptions.NotInStr))
	}

	// level: InInt = [1, 2, 3]
	if fields[2].ValidateOptions == nil {
		t.Fatal("level: ValidateOptions should not be nil")
	}
	if len(fields[2].ValidateOptions.InInt) != 3 {
		t.Errorf("level: InInt len = %d, want 3", len(fields[2].ValidateOptions.InInt))
	}

	// priority: NotInInt = [0]
	if fields[3].ValidateOptions == nil {
		t.Fatal("priority: ValidateOptions should not be nil")
	}
	if len(fields[3].ValidateOptions.NotInInt) != 1 || fields[3].ValidateOptions.NotInInt[0] != 0 {
		t.Errorf("priority: NotInInt = %v, want [0]", fields[3].ValidateOptions.NotInInt)
	}

	// code: InUint = [100, 200]
	if fields[4].ValidateOptions == nil {
		t.Fatal("code: ValidateOptions should not be nil")
	}
	if len(fields[4].ValidateOptions.InUint) != 2 {
		t.Errorf("code: InUint len = %d, want 2", len(fields[4].ValidateOptions.InUint))
	}

	// flags: NotInUint = [0]
	if fields[5].ValidateOptions == nil {
		t.Fatal("flags: ValidateOptions should not be nil")
	}
	if len(fields[5].ValidateOptions.NotInUint) != 1 || fields[5].ValidateOptions.NotInUint[0] != 0 {
		t.Errorf("flags: NotInUint = %v, want [0]", fields[5].ValidateOptions.NotInUint)
	}
}

// TestParseUpdateCreateOptions_ProtoCompiles verifies that protocompile can parse proto files
// using the new update_message and create_message options without errors.
// Full option reading is tested in subtask_3; this test only validates proto compilation.
func TestParseUpdateCreateOptions_ProtoCompiles(t *testing.T) {
	t.Parallel()

	workspace := t.TempDir()
	writeProtoFile(t, workspace, "user.proto", `syntax = "proto3";
package test.user;
import "gcode/options.proto";

message User {
  int64           id         = 1;
  string          name       = 2;
  optional string email      = 3;
  int64           created_at = 4;
  int64           updated_at = 5;

  option (gcode.update_message) = {
    name: "UserUpdateByID"
    condition_fields: ["id"]
    ignore_fields: ["created_at", "updated_at"]
  };
  option (gcode.update_message) = {
    name: "UserUpdateByEmail"
    condition_fields: ["email"]
    ignore_fields: ["created_at", "updated_at"]
  };
  option (gcode.create_message) = {
    name: "UserCreate"
    ignore_fields: ["id", "created_at", "updated_at"]
    required_fields: ["email"]
  };
}
`)

	files, err := Parse(t.Context(), []string{workspace}, []string{"user.proto"})
	if err != nil {
		t.Fatalf("Parse returned error: %v", err)
	}
	if len(files) != 1 || len(files[0].Messages) != 1 {
		t.Fatalf("expected 1 file with 1 message, got %d files", len(files))
	}
}

// TestParseUpdateSourceCreateSource_ProtoCompiles verifies that update_source and create_source
// options (written into generated intermediate proto files) can be parsed.
func TestParseUpdateSourceCreateSource_ProtoCompiles(t *testing.T) {
	t.Parallel()

	workspace := t.TempDir()
	writeProtoFile(t, workspace, "user.update.proto", `syntax = "proto3";
package test.user;
import "gcode/options.proto";

message UserUpdateByID {
  option (gcode.update_source) = "User";

  int64           id    = 1;
  optional string name  = 2;
  optional string email = 3;
}
`)
	writeProtoFile(t, workspace, "user.create.proto", `syntax = "proto3";
package test.user;
import "gcode/options.proto";

message UserCreate {
  option (gcode.create_source) = "User";

  string name  = 1;
  optional string email = 2;
}
`)

	for _, f := range []string{"user.update.proto", "user.create.proto"} {
		files, err := Parse(t.Context(), []string{workspace}, []string{f})
		if err != nil {
			t.Fatalf("Parse(%s) returned error: %v", f, err)
		}
		if len(files) != 1 || len(files[0].Messages) != 1 {
			t.Fatalf("%s: expected 1 file with 1 message, got %d files", f, len(files))
		}
	}
}

// TestParseUpdateMessageOptions verifies that update_message options are correctly read into model.
func TestParseUpdateMessageOptions(t *testing.T) {
	t.Parallel()

	workspace := t.TempDir()
	writeProtoFile(t, workspace, "user.proto", `syntax = "proto3";
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
  option (gcode.update_message) = {
    name: "UserUpdateByEmail"
    condition_fields: ["email"]
    ignore_fields: ["created_at"]
  };
}
`)

	files, err := Parse(t.Context(), []string{workspace}, []string{"user.proto"})
	if err != nil {
		t.Fatalf("Parse returned error: %v", err)
	}
	msg := files[0].Messages[0]

	if len(msg.UpdateOptions) != 2 {
		t.Fatalf("expected 2 UpdateOptions, got %d", len(msg.UpdateOptions))
	}
	if msg.CreateOptions != nil {
		t.Errorf("expected nil CreateOptions, got %v", msg.CreateOptions)
	}

	u0 := msg.UpdateOptions[0]
	if u0.Name != "UserUpdateByID" {
		t.Errorf("UpdateOptions[0].Name = %q, want UserUpdateByID", u0.Name)
	}
	if len(u0.ConditionFields) != 1 || u0.ConditionFields[0] != "id" {
		t.Errorf("UpdateOptions[0].ConditionFields = %v, want [id]", u0.ConditionFields)
	}
	if len(u0.IgnoreFields) != 1 || u0.IgnoreFields[0] != "created_at" {
		t.Errorf("UpdateOptions[0].IgnoreFields = %v, want [created_at]", u0.IgnoreFields)
	}

	u1 := msg.UpdateOptions[1]
	if u1.Name != "UserUpdateByEmail" {
		t.Errorf("UpdateOptions[1].Name = %q, want UserUpdateByEmail", u1.Name)
	}
	if len(u1.ConditionFields) != 1 || u1.ConditionFields[0] != "email" {
		t.Errorf("UpdateOptions[1].ConditionFields = %v, want [email]", u1.ConditionFields)
	}
}

// TestParseCreateMessageOptions verifies that create_message options are correctly read into model.
func TestParseCreateMessageOptions(t *testing.T) {
	t.Parallel()

	workspace := t.TempDir()
	writeProtoFile(t, workspace, "product.proto", `syntax = "proto3";
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

	files, err := Parse(t.Context(), []string{workspace}, []string{"product.proto"})
	if err != nil {
		t.Fatalf("Parse returned error: %v", err)
	}
	msg := files[0].Messages[0]

	if msg.UpdateOptions != nil {
		t.Errorf("expected nil UpdateOptions, got %v", msg.UpdateOptions)
	}
	if len(msg.CreateOptions) != 1 {
		t.Fatalf("expected 1 CreateOptions, got %d", len(msg.CreateOptions))
	}

	c0 := msg.CreateOptions[0]
	if c0.Name != "ProductCreate" {
		t.Errorf("CreateOptions[0].Name = %q, want ProductCreate", c0.Name)
	}
	if len(c0.IgnoreFields) != 2 {
		t.Errorf("CreateOptions[0].IgnoreFields = %v, want [id created_at]", c0.IgnoreFields)
	}
	if len(c0.RequiredFields) != 1 || c0.RequiredFields[0] != "sku" {
		t.Errorf("CreateOptions[0].RequiredFields = %v, want [sku]", c0.RequiredFields)
	}
}

// TestParseUpdateSourceCreateSource verifies that update_source and create_source options
// are correctly read into model.Message.UpdateSource and CreateSource.
func TestParseUpdateSourceCreateSource(t *testing.T) {
	t.Parallel()

	workspace := t.TempDir()
	writeProtoFile(t, workspace, "user.update.proto", `syntax = "proto3";
package test.user;
import "gcode/options.proto";

message UserUpdateByID {
  option (gcode.update_source) = "User";

  int64           id    = 1;
  optional string name  = 2;
  optional string email = 3;
}
`)
	writeProtoFile(t, workspace, "user.create.proto", `syntax = "proto3";
package test.user;
import "gcode/options.proto";

message UserCreate {
  option (gcode.create_source) = "User";

  string          name  = 1;
  optional string email = 2;
}
`)

	updateFiles, err := Parse(t.Context(), []string{workspace}, []string{"user.update.proto"})
	if err != nil {
		t.Fatalf("Parse(user.update.proto) returned error: %v", err)
	}
	updateMsg := updateFiles[0].Messages[0]
	if updateMsg.UpdateSource != "User" {
		t.Errorf("UpdateSource = %q, want User", updateMsg.UpdateSource)
	}
	if updateMsg.CreateSource != "" {
		t.Errorf("CreateSource should be empty, got %q", updateMsg.CreateSource)
	}

	createFiles, err := Parse(t.Context(), []string{workspace}, []string{"user.create.proto"})
	if err != nil {
		t.Fatalf("Parse(user.create.proto) returned error: %v", err)
	}
	createMsg := createFiles[0].Messages[0]
	if createMsg.CreateSource != "User" {
		t.Errorf("CreateSource = %q, want User", createMsg.CreateSource)
	}
	if createMsg.UpdateSource != "" {
		t.Errorf("UpdateSource should be empty, got %q", createMsg.UpdateSource)
	}
}

// TestParseRequiredFieldsConstraint verifies that required_fields validation has moved to
// the transform layer: parser accepts these proto files without error.
func TestParseRequiredFieldsConstraint(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name  string
		proto string
	}{
		{
			name: "required_fields non-optional field",
			proto: `syntax = "proto3";
package test;
import "gcode/options.proto";
message M {
  string name = 1;
  option (gcode.create_message) = {
    name: "MCreate"
    required_fields: ["name"]
  };
}`,
		},
		{
			name: "required_fields unknown field",
			proto: `syntax = "proto3";
package test;
import "gcode/options.proto";
message M {
  optional string name = 1;
  option (gcode.create_message) = {
    name: "MCreate"
    required_fields: ["nonexistent"]
  };
}`,
		},
	}

	for _, tc := range cases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			workspace := t.TempDir()
			writeProtoFile(t, workspace, "constraint.proto", tc.proto)
			// Parser no longer validates required_fields; transform layer does.
			_, err := Parse(t.Context(), []string{workspace}, []string{"constraint.proto"})
			if err != nil {
				t.Errorf("Parse returned unexpected error: %v", err)
			}
		})
	}
}

// TestParseNoUpdateCreateOptions verifies that messages without update/create options
// have nil UpdateOptions, CreateOptions, and empty source fields.
func TestParseNoUpdateCreateOptions(t *testing.T) {
	t.Parallel()

	workspace := t.TempDir()
	writeProtoFile(t, workspace, "plain.proto", `syntax = "proto3";
package test;
message Plain {
  string name = 1;
  int32  age  = 2;
}
`)

	files, err := Parse(t.Context(), []string{workspace}, []string{"plain.proto"})
	if err != nil {
		t.Fatalf("Parse returned error: %v", err)
	}
	msg := files[0].Messages[0]
	if msg.UpdateOptions != nil {
		t.Errorf("expected nil UpdateOptions, got %v", msg.UpdateOptions)
	}
	if msg.CreateOptions != nil {
		t.Errorf("expected nil CreateOptions, got %v", msg.CreateOptions)
	}
	if msg.UpdateSource != "" {
		t.Errorf("expected empty UpdateSource, got %q", msg.UpdateSource)
	}
	if msg.CreateSource != "" {
		t.Errorf("expected empty CreateSource, got %q", msg.CreateSource)
	}
}

// TestParseUpdateAndCreateCoexist verifies that a message with both update_message and
// create_message options has both correctly populated and independent.
func TestParseUpdateAndCreateCoexist(t *testing.T) {
	t.Parallel()

	workspace := t.TempDir()
	writeProtoFile(t, workspace, "item.proto", `syntax = "proto3";
package test.item;
import "gcode/options.proto";

message Item {
  int64           id         = 1;
  string          title      = 2;
  optional string sku        = 3;
  int64           created_at = 4;

  option (gcode.update_message) = {
    name: "ItemUpdateByID"
    condition_fields: ["id"]
    ignore_fields: ["created_at"]
  };
  option (gcode.create_message) = {
    name: "ItemCreate"
    ignore_fields: ["id", "created_at"]
    required_fields: ["sku"]
  };
}
`)

	files, err := Parse(t.Context(), []string{workspace}, []string{"item.proto"})
	if err != nil {
		t.Fatalf("Parse returned error: %v", err)
	}
	msg := files[0].Messages[0]

	if len(msg.UpdateOptions) != 1 {
		t.Fatalf("expected 1 UpdateOptions, got %d", len(msg.UpdateOptions))
	}
	if len(msg.CreateOptions) != 1 {
		t.Fatalf("expected 1 CreateOptions, got %d", len(msg.CreateOptions))
	}

	// Verify they are independent.
	if msg.UpdateOptions[0].Name != "ItemUpdateByID" {
		t.Errorf("UpdateOptions[0].Name = %q, want ItemUpdateByID", msg.UpdateOptions[0].Name)
	}
	if msg.CreateOptions[0].Name != "ItemCreate" {
		t.Errorf("CreateOptions[0].Name = %q, want ItemCreate", msg.CreateOptions[0].Name)
	}
	// UpdateOptions should not have RequiredFields.
	if len(msg.UpdateOptions[0].ConditionFields) != 1 || msg.UpdateOptions[0].ConditionFields[0] != "id" {
		t.Errorf("UpdateOptions[0].ConditionFields = %v, want [id]", msg.UpdateOptions[0].ConditionFields)
	}
	if len(msg.CreateOptions[0].RequiredFields) != 1 || msg.CreateOptions[0].RequiredFields[0] != "sku" {
		t.Errorf("CreateOptions[0].RequiredFields = %v, want [sku]", msg.CreateOptions[0].RequiredFields)
	}
}

// TestParseCreateMessageMultiple verifies that multiple create_message options on one message
// are all correctly read.
func TestParseCreateMessageMultiple(t *testing.T) {
	t.Parallel()

	workspace := t.TempDir()
	writeProtoFile(t, workspace, "order.proto", `syntax = "proto3";
package test.order;
import "gcode/options.proto";

message Order {
  int64           id         = 1;
  string          ref        = 2;
  optional string note       = 3;
  int64           created_at = 4;

  option (gcode.create_message) = {
    name: "OrderCreate"
    ignore_fields: ["id", "created_at"]
  };
  option (gcode.create_message) = {
    name: "OrderDraftCreate"
    ignore_fields: ["id", "created_at"]
    required_fields: ["note"]
  };
}
`)

	files, err := Parse(t.Context(), []string{workspace}, []string{"order.proto"})
	if err != nil {
		t.Fatalf("Parse returned error: %v", err)
	}
	msg := files[0].Messages[0]

	if len(msg.CreateOptions) != 2 {
		t.Fatalf("expected 2 CreateOptions, got %d", len(msg.CreateOptions))
	}
	if msg.CreateOptions[0].Name != "OrderCreate" {
		t.Errorf("CreateOptions[0].Name = %q, want OrderCreate", msg.CreateOptions[0].Name)
	}
	if msg.CreateOptions[1].Name != "OrderDraftCreate" {
		t.Errorf("CreateOptions[1].Name = %q, want OrderDraftCreate", msg.CreateOptions[1].Name)
	}
	if len(msg.CreateOptions[1].RequiredFields) != 1 || msg.CreateOptions[1].RequiredFields[0] != "note" {
		t.Errorf("CreateOptions[1].RequiredFields = %v, want [note]", msg.CreateOptions[1].RequiredFields)
	}
}

// TestParseUpdateMessageEmptyRepeatedFields verifies that when condition_fields and
// ignore_fields are omitted, they map to nil (not empty slice).
func TestParseUpdateMessageEmptyRepeatedFields(t *testing.T) {
	t.Parallel()

	workspace := t.TempDir()
	writeProtoFile(t, workspace, "minimal.proto", `syntax = "proto3";
package test.minimal;
import "gcode/options.proto";

message Thing {
  int64  id    = 1;
  string title = 2;

  option (gcode.update_message) = {
    name: "ThingUpdate"
  };
}
`)

	files, err := Parse(t.Context(), []string{workspace}, []string{"minimal.proto"})
	if err != nil {
		t.Fatalf("Parse returned error: %v", err)
	}
	msg := files[0].Messages[0]

	if len(msg.UpdateOptions) != 1 {
		t.Fatalf("expected 1 UpdateOptions, got %d", len(msg.UpdateOptions))
	}
	u := msg.UpdateOptions[0]
	if u.Name != "ThingUpdate" {
		t.Errorf("Name = %q, want ThingUpdate", u.Name)
	}
	// Omitted repeated fields should be nil, not empty slice.
	if u.ConditionFields != nil {
		t.Errorf("ConditionFields should be nil when omitted, got %v", u.ConditionFields)
	}
	if u.IgnoreFields != nil {
		t.Errorf("IgnoreFields should be nil when omitted, got %v", u.IgnoreFields)
	}
}

func TestAppendCommentBlock(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name  string
		block string
		want  []string
	}{
		{
			name:  "empty block",
			block: "",
			want:  nil,
		},
		{
			name:  "whitespace only",
			block: "   \n",
			want:  nil,
		},
		{
			name:  "single line with leading space",
			block: " CreatePerson creates a new person record.\n",
			want:  []string{"CreatePerson creates a new person record."},
		},
		{
			name:  "single line with leading tab",
			block: "\tTabIndented comment.\n",
			want:  []string{"TabIndented comment."},
		},
		{
			name:  "trailing whitespace stripped",
			block: " text with trailing spaces   \n",
			want:  []string{"text with trailing spaces"},
		},
		{
			name:  "multi-line",
			block: " First line.\n Second line.\n",
			want:  []string{"First line.", "Second line."},
		},
		{
			name:  "empty line preserved",
			block: " First.\n\n After blank.\n",
			want:  []string{"First.", "", "After blank."},
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			got := appendCommentBlock(nil, tc.block)
			if len(got) != len(tc.want) {
				t.Fatalf("got %v, want %v", got, tc.want)
			}
			for i := range got {
				if got[i] != tc.want[i] {
					t.Errorf("line %d: got %q, want %q", i, got[i], tc.want[i])
				}
			}
		})
	}
}
