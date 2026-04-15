package parser

import (
	"context"
	"fmt"
	"io"
	"strings"
	"sync"

	"github.com/bufbuild/protocompile"
	"github.com/bufbuild/protocompile/linker"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/reflect/protodesc"
	"google.golang.org/protobuf/reflect/protoreflect"
	"google.golang.org/protobuf/reflect/protoregistry"
	"google.golang.org/protobuf/types/descriptorpb"
	"google.golang.org/protobuf/types/dynamicpb"

	"github.com/pinealctx/gcode/internal/model"
	"github.com/pinealctx/gcode/options"
)

// gcodeOptionsPath is the virtual import path exposed to user proto files.
const gcodeOptionsPath = "gcode/options.proto"

// validatePath is the canonical import path for buf/validate annotations.
const validatePath = "buf/validate/validate.proto"

// cachedGcodeExts and cachedValidateExts hold the compiled extension descriptors
// for the two embedded static proto files. Both are initialized at most once per
// process via sync.Once, since the embedded content never changes at runtime.
var (
	gcodeExtsOnce   sync.Once
	cachedGcodeExts *gcodeExtensions
	gcodeExtsErr    error

	validateExtsOnce   sync.Once
	cachedValidateExts *validateExtensions
	validateExtsErr    error
)

// embeddedResolver wraps an inner Resolver and intercepts requests for
// gcodeOptionsPath, serving the embedded gcode_options.proto content.
// All other paths are forwarded to the inner resolver unchanged.
type embeddedResolver struct {
	inner protocompile.Resolver
}

func (r *embeddedResolver) FindFileByPath(path string) (protocompile.SearchResult, error) {
	switch path {
	case gcodeOptionsPath:
		return protocompile.SearchResult{
			Source: io.NopCloser(strings.NewReader(string(options.GcodeOptionsProto))),
		}, nil
	case validatePath:
		return protocompile.SearchResult{
			Source: io.NopCloser(strings.NewReader(string(options.BufValidateProto))),
		}, nil
	}
	return r.inner.FindFileByPath(path)
}

// gcodeExtensions holds the compiled extension descriptors for gcode options.
// It is initialized once by buildGcodeExtensions and reused across Parse calls.
type gcodeExtensions struct {
	schemaExt           protoreflect.ExtensionType // extend google.protobuf.FileOptions    { SchemaFileOptions      schema            = 50000 }
	messageExt          protoreflect.ExtensionType // extend google.protobuf.MessageOptions { GcodeMessageOptions    message           = 50001 }
	fieldExt            protoreflect.ExtensionType // extend google.protobuf.FieldOptions  { GcodeFieldOptions      field             = 50002 }
	updateMessageExt    protoreflect.ExtensionType // extend google.protobuf.MessageOptions { repeated UpdateMessageOptions update_message = 50003 }
	createMessageExt    protoreflect.ExtensionType // extend google.protobuf.MessageOptions { repeated CreateMessageOptions create_message = 50004 }
	updateSourceOptsExt protoreflect.ExtensionType // extend google.protobuf.MessageOptions { UpdateSourceOptions update_source_opts = 50007 }
	createSourceExt     protoreflect.ExtensionType // extend google.protobuf.MessageOptions { string create_source   = 50006 }
}

// buildGcodeExtensions compiles gcode_options.proto in isolation and extracts
// the two extension descriptors using dynamicpb, avoiding any dependency on
// protoc-generated Go code.
// Results are cached after the first call; subsequent calls return the cached value.
func buildGcodeExtensions() (*gcodeExtensions, error) {
	// gcodeExtsOnce ensures extensions are compiled exactly once.
	// If compilation fails, the error is cached and all subsequent calls
	// return the same error — there is no retry mechanism. This is intentional:
	// extension compilation failure indicates a corrupted embed or a programming
	// error, not a transient condition.
	gcodeExtsOnce.Do(func() {
		cachedGcodeExts, gcodeExtsErr = compileGcodeExtensions()
	})
	return cachedGcodeExts, gcodeExtsErr
}

// compileGcodeExtensions performs the actual compilation of gcode_options.proto
// and extracts all six extension descriptors. Called at most once via buildGcodeExtensions.
func compileGcodeExtensions() (*gcodeExtensions, error) {
	compiler := protocompile.Compiler{
		Resolver: &embeddedResolver{
			inner: protocompile.WithStandardImports(&protocompile.SourceResolver{}),
		},
	}

	compiled, err := compiler.Compile(context.Background(), gcodeOptionsPath)
	if err != nil {
		return nil, fmt.Errorf("compile gcode options proto: %w", err)
	}

	// Find the gcode_options.proto result among compiled files.
	var gcodeResult linker.Result
	for _, f := range compiled {
		r, ok := f.(linker.Result)
		if !ok {
			return nil, fmt.Errorf("unexpected result type %T", f)
		}
		if r.Path() == gcodeOptionsPath {
			gcodeResult = r
		}
	}
	if gcodeResult == nil {
		return nil, fmt.Errorf("gcode options proto not found in compiled results")
	}

	// Use protoregistry.GlobalFiles as the dependency resolver so that
	// google/protobuf/descriptor.proto (already registered globally) is found.
	fd, err := protodesc.NewFile(gcodeResult.FileDescriptorProto(), protoregistry.GlobalFiles)
	if err != nil {
		return nil, fmt.Errorf("build file descriptor for gcode options: %w", err)
	}

	exts := fd.Extensions()
	var schemaExt, msgExt, fieldExt, updateMsgExt, createMsgExt, updateSrcOptsExt, createSrcExt protoreflect.ExtensionDescriptor
	// protobuf descriptor API uses Len()/Get(i); no range iterator available.
	for i := 0; i < exts.Len(); i++ {
		ext := exts.Get(i)
		switch ext.Name() {
		case "schema":
			schemaExt = ext
		case "message":
			msgExt = ext
		case "field":
			fieldExt = ext
		case "update_message":
			updateMsgExt = ext
		case "create_message":
			createMsgExt = ext
		case "update_source_opts":
			updateSrcOptsExt = ext
		case "create_source":
			createSrcExt = ext
		}
	}
	if schemaExt == nil {
		return nil, fmt.Errorf("gcode options proto: missing expected extension 'schema'")
	}
	if msgExt == nil || fieldExt == nil {
		return nil, fmt.Errorf("gcode options proto: missing expected extensions (message=%v, field=%v)", msgExt, fieldExt)
	}
	if updateMsgExt == nil || createMsgExt == nil || updateSrcOptsExt == nil || createSrcExt == nil {
		return nil, fmt.Errorf("gcode options proto: missing phase4 extensions (update_message=%v, create_message=%v, update_source_opts=%v, create_source=%v)",
			updateMsgExt, createMsgExt, updateSrcOptsExt, createSrcExt)
	}

	return &gcodeExtensions{
		schemaExt:           dynamicpb.NewExtensionType(schemaExt),
		messageExt:          dynamicpb.NewExtensionType(msgExt),
		fieldExt:            dynamicpb.NewExtensionType(fieldExt),
		updateMessageExt:    dynamicpb.NewExtensionType(updateMsgExt),
		createMessageExt:    dynamicpb.NewExtensionType(createMsgExt),
		updateSourceOptsExt: dynamicpb.NewExtensionType(updateSrcOptsExt),
		createSourceExt:     dynamicpb.NewExtensionType(createSrcExt),
	}, nil
}

// getStringField retrieves a string field by name from a *dynamicpb.Message.
// Returns empty string if the message is nil or the field is not set.
func getStringField(msg *dynamicpb.Message, name protoreflect.Name) string {
	if msg == nil {
		return ""
	}
	fd := msg.Descriptor().Fields().ByName(name)
	if fd == nil {
		return ""
	}
	return msg.Get(fd).String()
}

// getBoolField retrieves a bool field by name from a *dynamicpb.Message.
func getBoolField(msg *dynamicpb.Message, name protoreflect.Name) bool {
	if msg == nil {
		return false
	}
	fd := msg.Descriptor().Fields().ByName(name)
	if fd == nil {
		return false
	}
	return msg.Get(fd).Bool()
}

// getMessageField retrieves a nested message field by name from a *dynamicpb.Message.
// Returns nil if the field is not set or is the default (empty) message.
func getMessageField(msg *dynamicpb.Message, name protoreflect.Name) *dynamicpb.Message {
	if msg == nil {
		return nil
	}
	fd := msg.Descriptor().Fields().ByName(name)
	if fd == nil {
		return nil
	}
	if !msg.Has(fd) {
		return nil
	}
	nested, ok := msg.Get(fd).Message().(*dynamicpb.Message)
	if !ok {
		return nil
	}
	return nested
}

// readMessageOptions extracts gcode message-level annotations from a compiled
// MessageOptions proto using the provided extension type.
// Returns table="", ok=false if no gcode annotation is present.
func readMessageOptions(opts proto.Message, ext protoreflect.ExtensionType) (table string, ok bool) {
	if opts == nil {
		return "", false
	}
	msgOpts, castOK := opts.(*descriptorpb.MessageOptions)
	if !castOK || msgOpts == nil {
		return "", false
	}
	if !proto.HasExtension(msgOpts, ext) {
		return "", false
	}
	val := proto.GetExtension(msgOpts, ext)
	gcodeMsg, dynOK := val.(*dynamicpb.Message)
	if !dynOK || gcodeMsg == nil {
		return "", false
	}
	gormMsg := getMessageField(gcodeMsg, "gorm")
	if gormMsg == nil {
		return "", false
	}
	table = getStringField(gormMsg, "table")
	return table, table != ""
}

// readFieldOptions extracts gcode field-level annotations from a compiled
// FieldOptions proto using the provided extension type.
func readFieldOptions(opts proto.Message, ext protoreflect.ExtensionType) (gormColumn string, jsonOmitempty, jsonIgnore bool, validateMessage string) {
	if opts == nil {
		return "", false, false, ""
	}
	fieldOpts, castOK := opts.(*descriptorpb.FieldOptions)
	if !castOK || fieldOpts == nil {
		return "", false, false, ""
	}
	if !proto.HasExtension(fieldOpts, ext) {
		return "", false, false, ""
	}
	val := proto.GetExtension(fieldOpts, ext)
	gcodeMsg, dynOK := val.(*dynamicpb.Message)
	if !dynOK || gcodeMsg == nil {
		return "", false, false, ""
	}

	if gormMsg := getMessageField(gcodeMsg, "gorm"); gormMsg != nil {
		gormColumn = getStringField(gormMsg, "column")
	}
	if jsonMsg := getMessageField(gcodeMsg, "json"); jsonMsg != nil {
		jsonOmitempty = getBoolField(jsonMsg, "omitempty")
		jsonIgnore = getBoolField(jsonMsg, "ignore")
	}
	validateMessage = getStringField(gcodeMsg, "validate_message")
	return gormColumn, jsonOmitempty, jsonIgnore, validateMessage
}

// readSchemaFileOption reports whether the file carries (gcode.schema) = {};.
func readSchemaFileOption(opts proto.Message, ext protoreflect.ExtensionType) bool {
	if opts == nil {
		return false
	}
	fileOpts, ok := opts.(*descriptorpb.FileOptions)
	if !ok || fileOpts == nil {
		return false
	}
	return proto.HasExtension(fileOpts, ext)
}

// readUpdateMessageOptions extracts repeated update_message options from MessageOptions.
// Returns nil if no update_message annotations are present.
func readUpdateMessageOptions(opts proto.Message, ext protoreflect.ExtensionType) ([]model.UpdateMessageOptions, error) {
	if opts == nil {
		return nil, nil
	}
	msgOpts, ok := opts.(*descriptorpb.MessageOptions)
	if !ok || msgOpts == nil {
		return nil, nil
	}
	if !proto.HasExtension(msgOpts, ext) {
		return nil, nil
	}
	// For repeated extensions, use ProtoReflect to get the list value directly.
	listVal := msgOpts.ProtoReflect().Get(ext.TypeDescriptor()).List()
	if listVal.Len() == 0 {
		return nil, nil
	}
	result := make([]model.UpdateMessageOptions, 0, listVal.Len())
	// protobuf descriptor API uses Len()/Get(i); no range iterator available.
	for i := 0; i < listVal.Len(); i++ {
		item, ok := listVal.Get(i).Message().(*dynamicpb.Message)
		if !ok || item == nil {
			return nil, fmt.Errorf("update_message[%d]: unexpected message type %T", i, listVal.Get(i).Message())
		}
		opt := model.UpdateMessageOptions{
			Name:            getStringField(item, "name"),
			ConditionFields: getStringListField(item, "condition_fields"),
			IgnoreFields:    getStringListField(item, "ignore_fields"),
		}
		result = append(result, opt)
	}
	if len(result) == 0 {
		return nil, nil
	}
	return result, nil
}

// readCreateMessageOptions extracts repeated create_message options from MessageOptions.
// Returns nil if no create_message annotations are present.
func readCreateMessageOptions(opts proto.Message, ext protoreflect.ExtensionType) ([]model.CreateMessageOptions, error) {
	if opts == nil {
		return nil, nil
	}
	msgOpts, ok := opts.(*descriptorpb.MessageOptions)
	if !ok || msgOpts == nil {
		return nil, nil
	}
	if !proto.HasExtension(msgOpts, ext) {
		return nil, nil
	}
	listVal := msgOpts.ProtoReflect().Get(ext.TypeDescriptor()).List()
	if listVal.Len() == 0 {
		return nil, nil
	}
	result := make([]model.CreateMessageOptions, 0, listVal.Len())
	// protobuf descriptor API uses Len()/Get(i); no range iterator available.
	for i := 0; i < listVal.Len(); i++ {
		item, ok := listVal.Get(i).Message().(*dynamicpb.Message)
		if !ok || item == nil {
			return nil, fmt.Errorf("create_message[%d]: unexpected message type %T", i, listVal.Get(i).Message())
		}
		opt := model.CreateMessageOptions{
			Name:           getStringField(item, "name"),
			IgnoreFields:   getStringListField(item, "ignore_fields"),
			RequiredFields: getStringListField(item, "required_fields"),
		}
		result = append(result, opt)
	}
	if len(result) == 0 {
		return nil, nil
	}
	return result, nil
}

// readStringMessageOption extracts a string message-level option (e.g. create_source).
// Returns empty string if not set.
func readStringMessageOption(opts proto.Message, ext protoreflect.ExtensionType) string {
	if opts == nil {
		return ""
	}
	msgOpts, ok := opts.(*descriptorpb.MessageOptions)
	if !ok || msgOpts == nil {
		return ""
	}
	if !proto.HasExtension(msgOpts, ext) {
		return ""
	}
	val := proto.GetExtension(msgOpts, ext)
	s, ok := val.(string)
	if !ok {
		return ""
	}
	return s
}

// readUpdateSourceOptions extracts the UpdateSourceOptions message-level option.
// Returns the source name and condition_fields. Returns ("", nil) if not set.
func readUpdateSourceOptions(opts proto.Message, ext protoreflect.ExtensionType) (string, []string) {
	if opts == nil {
		return "", nil
	}
	msgOpts, ok := opts.(*descriptorpb.MessageOptions)
	if !ok || msgOpts == nil {
		return "", nil
	}
	if !proto.HasExtension(msgOpts, ext) {
		return "", nil
	}
	val := proto.GetExtension(msgOpts, ext)
	msg, ok := val.(*dynamicpb.Message)
	if !ok || msg == nil {
		return "", nil
	}
	source := getStringField(msg, "source")
	conditionFields := getStringListField(msg, "condition_fields")
	return source, conditionFields
}

// getStringListField retrieves a repeated string field by name from a *dynamicpb.Message.
// Returns nil if not set or empty.
func getStringListField(msg *dynamicpb.Message, name protoreflect.Name) []string {
	vals := getListField(msg, name)
	if len(vals) == 0 {
		return nil
	}
	result := make([]string, len(vals))
	for i, v := range vals {
		result[i] = v.String()
	}
	return result
}

// validateExtensions holds the compiled extension descriptor for buf.validate.field.
// It is initialized once by buildValidateExtensions and reused across Parse calls.
type validateExtensions struct {
	fieldExt protoreflect.ExtensionType // extend google.protobuf.FieldOptions { FieldConstraints field = 1 }
}

// buildValidateExtensions compiles buf/validate/validate.proto in isolation and
// extracts the field-level extension descriptor using dynamicpb.
// Results are cached after the first call; subsequent calls return the cached value.
func buildValidateExtensions() (*validateExtensions, error) {
	validateExtsOnce.Do(func() {
		cachedValidateExts, validateExtsErr = compileValidateExtensions()
	})
	return cachedValidateExts, validateExtsErr
}

// compileValidateExtensions performs the actual compilation of buf/validate/validate.proto
// and extracts the field-level extension descriptor. Called at most once via buildValidateExtensions.
func compileValidateExtensions() (*validateExtensions, error) {
	compiler := protocompile.Compiler{
		Resolver: &embeddedResolver{
			inner: protocompile.WithStandardImports(&protocompile.SourceResolver{}),
		},
	}

	compiled, err := compiler.Compile(context.Background(), validatePath)
	if err != nil {
		return nil, fmt.Errorf("compile buf/validate proto: %w", err)
	}

	var validateResult linker.Result
	for _, f := range compiled {
		r, ok := f.(linker.Result)
		if !ok {
			return nil, fmt.Errorf("unexpected result type %T", f)
		}
		if r.Path() == validatePath {
			validateResult = r
		}
	}
	if validateResult == nil {
		return nil, fmt.Errorf("buf/validate proto not found in compiled results")
	}

	fd, err := protodesc.NewFile(validateResult.FileDescriptorProto(), protoregistry.GlobalFiles)
	if err != nil {
		return nil, fmt.Errorf("build file descriptor for buf/validate: %w", err)
	}

	exts := fd.Extensions()
	var fieldExt protoreflect.ExtensionDescriptor
	// protobuf descriptor API uses Len()/Get(i); no range iterator available.
	for i := 0; i < exts.Len(); i++ {
		ext := exts.Get(i)
		if ext.Name() == "field" {
			fieldExt = ext
			break
		}
	}
	if fieldExt == nil {
		return nil, fmt.Errorf("buf/validate proto: missing expected extension 'field'")
	}

	return &validateExtensions{
		fieldExt: dynamicpb.NewExtensionType(fieldExt),
	}, nil
}

// getInt64Field retrieves a signed integer field by name from a *dynamicpb.Message.
// Returns 0 if the message is nil or the field is not set.
func getInt64Field(msg *dynamicpb.Message, name protoreflect.Name) int64 {
	if msg == nil {
		return 0
	}
	fd := msg.Descriptor().Fields().ByName(name)
	if fd == nil {
		return 0
	}
	return msg.Get(fd).Int()
}

// getUint64Field retrieves an unsigned integer field by name from a *dynamicpb.Message.
func getUint64Field(msg *dynamicpb.Message, name protoreflect.Name) uint64 {
	if msg == nil {
		return 0
	}
	fd := msg.Descriptor().Fields().ByName(name)
	if fd == nil {
		return 0
	}
	return msg.Get(fd).Uint()
}

// getFloat64Field retrieves a float/double field by name from a *dynamicpb.Message.
func getFloat64Field(msg *dynamicpb.Message, name protoreflect.Name) float64 {
	if msg == nil {
		return 0
	}
	fd := msg.Descriptor().Fields().ByName(name)
	if fd == nil {
		return 0
	}
	return msg.Get(fd).Float()
}

// hasField reports whether a field is explicitly set in a *dynamicpb.Message.
func hasField(msg *dynamicpb.Message, name protoreflect.Name) bool {
	if msg == nil {
		return false
	}
	fd := msg.Descriptor().Fields().ByName(name)
	if fd == nil {
		return false
	}
	return msg.Has(fd)
}

// getListField retrieves a repeated field by name from a *dynamicpb.Message,
// returning its elements as a slice. Returns nil if not set or empty.
func getListField(msg *dynamicpb.Message, name protoreflect.Name) []protoreflect.Value {
	if msg == nil {
		return nil
	}
	fd := msg.Descriptor().Fields().ByName(name)
	if fd == nil || !msg.Has(fd) {
		return nil
	}
	list := msg.Get(fd).List()
	if list.Len() == 0 {
		return nil
	}
	result := make([]protoreflect.Value, list.Len())
	// protobuf descriptor API uses Len()/Get(i); no range iterator available.
	for i := 0; i < list.Len(); i++ {
		result[i] = list.Get(i)
	}
	return result
}
