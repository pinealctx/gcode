package parser

import (
	"context"
	"fmt"
	"strings"

	"github.com/bufbuild/protocompile"
	"github.com/bufbuild/protocompile/linker"
	"github.com/pinealctx/x/errorx"
	"google.golang.org/protobuf/reflect/protoreflect"
	"google.golang.org/protobuf/types/descriptorpb"

	"github.com/pinealctx/gcode/internal/model"
)

// fileDescriptorDependencyFieldNumber is the stable field number of
// google.protobuf.FileDescriptorProto.dependency in descriptor.proto.
const fileDescriptorDependencyFieldNumber = 3

// Parse compiles the given root proto files and maps them into the stage1 semantic model.
func Parse(ctx context.Context, importPaths []string, files []string) ([]model.File, error) {
	exts, err := buildGcodeExtensions()
	if err != nil {
		return nil, fmt.Errorf("init gcode extensions: %w", err)
	}

	validateExts, err := buildValidateExtensions()
	if err != nil {
		return nil, fmt.Errorf("init validate extensions: %w", err)
	}

	compiler := protocompile.Compiler{
		Resolver: &embeddedResolver{
			inner: protocompile.WithStandardImports(&protocompile.SourceResolver{
				ImportPaths: importPaths,
			}),
		},
		SourceInfoMode: protocompile.SourceInfoStandard,
		RetainASTs:     true,
	}

	compiledFiles, err := compiler.Compile(ctx, files...)
	if err != nil {
		return nil, fmt.Errorf("compile proto files: %w", err)
	}

	modelFiles := make([]model.File, 0, len(compiledFiles))
	for _, compiledFile := range compiledFiles {
		result, ok := compiledFile.(linker.Result)
		if !ok {
			return nil, fmt.Errorf("compile proto file %q: unexpected result type %T", compiledFile.Path(), compiledFile)
		}
		if result.AST() == nil {
			return nil, fmt.Errorf("compile proto file %q: ast was not retained", compiledFile.Path())
		}

		mappedFile, err := mapFile(result, exts, validateExts)
		if err != nil {
			return nil, fmt.Errorf("map proto file %q: %w", compiledFile.Path(), err)
		}
		modelFiles = append(modelFiles, mappedFile)
	}

	return modelFiles, nil
}

// mapFile maps a compiled proto file descriptor to a model.File.
// It extracts imports, messages, enums, services, and file-level options.
func mapFile(file protoreflect.FileDescriptor, exts *gcodeExtensions, validateExts *validateExtensions) (model.File, error) {
	syntax, err := mapSyntax(file.Syntax())
	if err != nil {
		return model.File{}, err
	}

	imports := make([]model.Import, 0, file.Imports().Len())
	for index := 0; index < file.Imports().Len(); index++ {
		fileImport := file.Imports().Get(index)
		imports = append(imports, model.Import{
			Path:           fileImport.Path(),
			IsPublic:       fileImport.IsPublic,
			LeadingComment: commentFromLocation(file.SourceLocations().ByPath(protoreflect.SourcePath{fileDescriptorDependencyFieldNumber, int32(index)})),
			Location:       locationFromSource(file.Path(), file.SourceLocations().ByPath(protoreflect.SourcePath{fileDescriptorDependencyFieldNumber, int32(index)})),
		})
	}

	messages := make([]model.Message, 0, file.Messages().Len())
	for index := 0; index < file.Messages().Len(); index++ {
		message, err := mapMessage(file.Messages().Get(index), file.Path(), file.SourceLocations(), exts, validateExts)
		if err != nil {
			return model.File{}, err
		}
		messages = append(messages, message)
	}

	enums := make([]model.Enum, 0, file.Enums().Len())
	for index := 0; index < file.Enums().Len(); index++ {
		enum, err := mapEnum(file.Enums().Get(index), file.Path(), file.SourceLocations())
		if err != nil {
			return model.File{}, err
		}
		enums = append(enums, enum)
	}

	services, err := mapServices(file)
	if err != nil {
		return model.File{}, err
	}

	isSchema := readSchemaFileOption(file.Options(), exts.schemaExt)

	return model.File{
		Path:           file.Path(),
		Syntax:         syntax,
		Package:        string(file.Package()),
		GoPackage:      fileGoPackage(file),
		Imports:        imports,
		Messages:       messages,
		Enums:          enums,
		Services:       services,
		IsSchema:       isSchema,
		LeadingComment: commentFromLocation(file.SourceLocations().ByDescriptor(file)),
		Location:       locationFromSource(file.Path(), file.SourceLocations().ByDescriptor(file)),
	}, nil
}

// mapServices maps all service definitions in a proto file to model.Service values.
// It returns an error if any rpc method uses streaming, which is not supported.
func mapServices(file protoreflect.FileDescriptor) ([]model.Service, error) {
	if file.Services().Len() == 0 {
		return nil, nil
	}
	services := make([]model.Service, 0, file.Services().Len())
	// protobuf descriptor API uses Len()/Get(i); no range iterator available.
	for i := 0; i < file.Services().Len(); i++ {
		svc := file.Services().Get(i)
		rpcs := make([]model.RPC, 0, svc.Methods().Len())
		for j := 0; j < svc.Methods().Len(); j++ {
			method := svc.Methods().Get(j)
			if method.IsStreamingClient() || method.IsStreamingServer() {
				return nil, errorx.NewSentinelf[parserTag]("service %q: rpc %q uses streaming, which is not supported",
					svc.Name(), method.Name())
			}
			rpcs = append(rpcs, model.RPC{
				Name:         string(method.Name()),
				RequestType:  string(method.Input().FullName()),
				ResponseType: string(method.Output().FullName()),
				LeadingComment: commentFromLocation(
					file.SourceLocations().ByDescriptor(method),
				),
				Location: locationFromSource(file.Path(),
					file.SourceLocations().ByDescriptor(method),
				),
			})
		}
		services = append(services, model.Service{
			Name:           string(svc.Name()),
			FullName:       string(svc.FullName()),
			RPCs:           rpcs,
			LeadingComment: commentFromLocation(file.SourceLocations().ByDescriptor(svc)),
			Location:       locationFromSource(file.Path(), file.SourceLocations().ByDescriptor(svc)),
		})
	}
	return services, nil
}

func mapMessage(message protoreflect.MessageDescriptor, filePath string, locations protoreflect.SourceLocations, exts *gcodeExtensions, validateExts *validateExtensions) (model.Message, error) {
	if message.IsMapEntry() {
		return model.Message{}, fmt.Errorf("message %q: map entry messages are not supported", message.FullName())
	}

	// Stage1 does not support oneof fields. Synthetic oneofs generated by the
	// proto3 compiler for optional fields are permitted; only real oneofs are
	// rejected.
	// protobuf descriptor API uses Len()/Get(i); no range iterator available.
	for i := 0; i < message.Oneofs().Len(); i++ {
		if !message.Oneofs().Get(i).IsSynthetic() {
			return model.Message{}, fmt.Errorf("message %q: oneof fields are not supported", message.FullName())
		}
	}

	fields := make([]model.Field, 0, message.Fields().Len())
	for index := 0; index < message.Fields().Len(); index++ {
		field, err := mapField(message.Fields().Get(index), filePath, locations, exts, validateExts)
		if err != nil {
			return model.Message{}, fmt.Errorf("message %q field %q: %w", message.FullName(), message.Fields().Get(index).Name(), err)
		}
		fields = append(fields, field)
	}

	nestedMessages := make([]model.Message, 0, message.Messages().Len())
	for index := 0; index < message.Messages().Len(); index++ {
		nestedMessage, err := mapMessage(message.Messages().Get(index), filePath, locations, exts, validateExts)
		if err != nil {
			return model.Message{}, fmt.Errorf("message %q nested[%d]: %w", message.FullName(), index, err)
		}
		nestedMessages = append(nestedMessages, nestedMessage)
	}

	nestedEnums := make([]model.Enum, 0, message.Enums().Len())
	for index := 0; index < message.Enums().Len(); index++ {
		nestedEnum, err := mapEnum(message.Enums().Get(index), filePath, locations)
		if err != nil {
			return model.Message{}, err
		}
		nestedEnums = append(nestedEnums, nestedEnum)
	}

	var gormOpts *model.GormMessageOptions
	if table, ok := readMessageOptions(message.Options(), exts.messageExt); ok {
		gormOpts = &model.GormMessageOptions{Table: table}
	}

	updateOpts, err := readUpdateMessageOptions(message.Options(), exts.updateMessageExt)
	if err != nil {
		return model.Message{}, fmt.Errorf("message %q: %w", message.FullName(), err)
	}
	createOpts, err := readCreateMessageOptions(message.Options(), exts.createMessageExt)
	if err != nil {
		return model.Message{}, fmt.Errorf("message %q: %w", message.FullName(), err)
	}

	updateSource, conditionFields := readUpdateSourceOptions(message.Options(), exts.updateSourceOptsExt)
	createSource := readStringMessageOption(message.Options(), exts.createSourceExt)

	return model.Message{
		Name:            string(message.Name()),
		FullName:        string(message.FullName()),
		Fields:          fields,
		Messages:        nestedMessages,
		Enums:           nestedEnums,
		GormOptions:     gormOpts,
		UpdateOptions:   updateOpts,
		CreateOptions:   createOpts,
		UpdateSource:    updateSource,
		ConditionFields: conditionFields,
		CreateSource:    createSource,
		LeadingComment:  commentFromLocation(locations.ByDescriptor(message)),
		Location:        locationFromSource(filePath, locations.ByDescriptor(message)),
	}, nil
}

func mapEnum(enum protoreflect.EnumDescriptor, filePath string, locations protoreflect.SourceLocations) (model.Enum, error) {
	values := make([]model.EnumValue, 0, enum.Values().Len())
	for index := 0; index < enum.Values().Len(); index++ {
		value := enum.Values().Get(index)
		values = append(values, model.EnumValue{
			Name:           string(value.Name()),
			Number:         int32(value.Number()),
			LeadingComment: commentFromLocation(locations.ByDescriptor(value)),
			Location:       locationFromSource(filePath, locations.ByDescriptor(value)),
		})
	}

	return model.Enum{
		Name:           string(enum.Name()),
		FullName:       string(enum.FullName()),
		Values:         values,
		LeadingComment: commentFromLocation(locations.ByDescriptor(enum)),
		Location:       locationFromSource(filePath, locations.ByDescriptor(enum)),
	}, nil
}

func mapField(field protoreflect.FieldDescriptor, filePath string, locations protoreflect.SourceLocations, exts *gcodeExtensions, validateExts *validateExtensions) (model.Field, error) {
	if field.IsMap() {
		return model.Field{}, fmt.Errorf("field %q: map fields are not supported", field.FullName())
	}

	fieldType, err := mapFieldType(field)
	if err != nil {
		return model.Field{}, err
	}

	cardinality := model.CardinalitySingular
	if field.IsList() {
		cardinality = model.CardinalityRepeated
	}

	// A field is optional when it carries an explicit proto3 optional keyword.
	// bytes fields are excluded: protoc-gen-go uses []byte (not *[]byte) for
	// optional bytes because nil already represents "not set" for slice types.
	optional := field.HasPresence() &&
		!field.IsList() &&
		field.Kind() != protoreflect.MessageKind &&
		field.Kind() != protoreflect.BytesKind

	// HasPresence is set for optional bytes: the field uses []byte (not *[]byte)
	// but still needs nil-vs-empty distinction in wire encoding.
	hasPresence := field.HasPresence() &&
		!field.IsList() &&
		field.Kind() == protoreflect.BytesKind

	var gormOpts *model.GormFieldOptions
	var jsonOpts *model.JSONFieldOptions
	var validateMsg string
	if col, omitempty, ignore, vmsg := readFieldOptions(field.Options(), exts.fieldExt); col != "" || omitempty || ignore || vmsg != "" {
		if col != "" {
			gormOpts = &model.GormFieldOptions{Column: col}
		}
		if omitempty || ignore {
			jsonOpts = &model.JSONFieldOptions{Omitempty: omitempty, Ignore: ignore}
		}
		validateMsg = vmsg
	}

	var validateOpts *model.ValidateFieldOptions
	if validateExts != nil {
		opts, err := readValidateOptions(field.Options(), validateExts.fieldExt, field.Kind(), string(field.FullName()))
		if err != nil {
			return model.Field{}, err
		}
		validateOpts = opts
	}

	return model.Field{
		Name:            string(field.Name()),
		Number:          int(field.Number()),
		Cardinality:     cardinality,
		Optional:        optional,
		HasPresence:     hasPresence,
		Type:            fieldType,
		JSONName:        field.JSONName(),
		GormOptions:     gormOpts,
		JSONOptions:     jsonOpts,
		ValidateOptions: validateOpts,
		ValidateMessage: validateMsg,
		LeadingComment:  commentFromLocation(locations.ByDescriptor(field)),
		Location:        locationFromSource(filePath, locations.ByDescriptor(field)),
	}, nil
}

func mapFieldType(field protoreflect.FieldDescriptor) (model.FieldType, error) {
	scalarKind, ok := mapScalarKind(field.Kind())
	if ok {
		return model.FieldType{
			Kind:   model.FieldKindScalar,
			Scalar: scalarKind,
			Name:   field.Kind().String(),
		}, nil
	}

	switch field.Kind() {
	case protoreflect.EnumKind:
		return model.FieldType{
			Kind:     model.FieldKindEnum,
			Name:     string(field.Enum().Name()),
			FullName: string(field.Enum().FullName()),
		}, nil
	case protoreflect.MessageKind:
		if isWellKnownType(string(field.Message().FullName())) {
			return model.FieldType{}, fmt.Errorf("field %q: well-known type %q is not supported",
				field.FullName(), field.Message().FullName())
		}
		return model.FieldType{
			Kind:     model.FieldKindMessage,
			Name:     string(field.Message().Name()),
			FullName: string(field.Message().FullName()),
		}, nil
	default:
		return model.FieldType{}, fmt.Errorf("field %q: unsupported kind %q", field.FullName(), field.Kind().String())
	}
}

func mapScalarKind(kind protoreflect.Kind) (model.ScalarKind, bool) {
	switch kind {
	case protoreflect.DoubleKind:
		return model.ScalarDouble, true
	case protoreflect.FloatKind:
		return model.ScalarFloat, true
	case protoreflect.Int32Kind:
		return model.ScalarInt32, true
	case protoreflect.Int64Kind:
		return model.ScalarInt64, true
	case protoreflect.Uint32Kind:
		return model.ScalarUint32, true
	case protoreflect.Uint64Kind:
		return model.ScalarUint64, true
	case protoreflect.Sint32Kind:
		return model.ScalarSint32, true
	case protoreflect.Sint64Kind:
		return model.ScalarSint64, true
	case protoreflect.Fixed32Kind:
		return model.ScalarFixed32, true
	case protoreflect.Fixed64Kind:
		return model.ScalarFixed64, true
	case protoreflect.Sfixed32Kind:
		return model.ScalarSfixed32, true
	case protoreflect.Sfixed64Kind:
		return model.ScalarSfixed64, true
	case protoreflect.BoolKind:
		return model.ScalarBool, true
	case protoreflect.StringKind:
		return model.ScalarString, true
	case protoreflect.BytesKind:
		return model.ScalarBytes, true
	default:
		return "", false
	}
}

func mapSyntax(syntax protoreflect.Syntax) (model.Syntax, error) {
	if syntax != protoreflect.Proto3 {
		return "", fmt.Errorf("unsupported proto syntax %q", syntax)
	}
	return model.SyntaxProto3, nil
}

func fileGoPackage(file protoreflect.FileDescriptor) string {
	options, ok := file.Options().(*descriptorpb.FileOptions)
	if !ok || options == nil {
		return ""
	}
	return options.GetGoPackage()
}

// wellKnownTypePrefix is the protobuf package prefix for all well-known types.
const wellKnownTypePrefix = "google.protobuf."

// isWellKnownType reports whether fullName belongs to the google.protobuf
// well-known types package.
func isWellKnownType(fullName string) bool {
	return strings.HasPrefix(fullName, wellKnownTypePrefix)
}

func commentFromLocation(location protoreflect.SourceLocation) model.Comment {
	lines := make([]string, 0, len(location.LeadingDetachedComments)+1)
	for index, block := range location.LeadingDetachedComments {
		lines = appendCommentBlock(lines, block)
		if index < len(location.LeadingDetachedComments)-1 || strings.TrimSpace(location.LeadingComments) != "" {
			lines = append(lines, "")
		}
	}
	lines = appendCommentBlock(lines, location.LeadingComments)

	return model.Comment{Lines: lines}
}

func appendCommentBlock(lines []string, block string) []string {
	trimmed := strings.TrimSuffix(block, "\n")
	if strings.TrimSpace(trimmed) == "" {
		return lines
	}

	for _, line := range strings.Split(trimmed, "\n") {
		lines = append(lines, strings.TrimRight(strings.TrimLeft(line, " \t"), "\r\t "))
	}
	return lines
}

func locationFromSource(path string, location protoreflect.SourceLocation) model.Location {
	return model.Location{
		Path:   path,
		Line:   location.StartLine + 1,
		Column: location.StartColumn + 1,
	}
}
