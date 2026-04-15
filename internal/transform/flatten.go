// Package transform converts the protobuf semantic model into a
// Go-oriented intermediate representation suitable for code rendering.
package transform

import (
	"fmt"

	"github.com/pinealctx/gcode/internal/model"
	"github.com/pinealctx/gcode/internal/naming"
)

// GoFile is the Go-oriented view of a single proto file, ready for rendering.
type GoFile struct {
	// Source is the original proto file path (used in the generated header).
	Source string
	// Package is the Go package name for the generated file.
	Package string
	// ProtoPackage is the proto package name (e.g., "compat"). Used by the
	// TS renderer to compute correct short type names for cross-file references.
	ProtoPackage string
	// Messages contains all messages (including nested) flattened to top level.
	Messages []GoMessage
	// Enums contains all enums (including nested) flattened to top level.
	Enums []GoEnum
	// Services contains all service definitions with resolved Go names.
	Services []GoService
}

// GoMessage is a flattened message with resolved Go names.
type GoMessage struct {
	// GoName is the Go struct name (e.g. "Person_Address" for nested types).
	GoName string
	// Comment lines from the proto source.
	Comment model.Comment
	// Fields with resolved Go names and types.
	Fields []GoField
	// UpdateSource is non-empty when this message was generated from an update_message
	// annotation. Its value is the source message name, used to generate ToMap() and
	// inherit validate rules.
	UpdateSource string
	// CreateSource is non-empty when this message was generated from a create_message
	// annotation. Its value is the source message name, used to inherit validate rules.
	CreateSource string
	// ConditionFields lists the condition_fields declared in the update_message annotation.
	// These are WHERE-condition fields and must not appear in ToMap() output.
	// Only populated when UpdateSource is non-empty.
	ConditionFields []string
	// RequiredFields lists the required_fields declared in the create_message annotation,
	// excluding message-type fields (which are inherently nullable in proto and therefore
	// cannot be detected via the non-optional heuristic). Used by validate inheritance
	// to determine which scalar/enum fields must be present.
	// Only populated when CreateSource is non-empty.
	RequiredFields []string
	// GormMessageOptions carries the message-level GORM annotation.
	// Used by render to generate TableName() and decide whether to emit gorm struct tags.
	// Nil means no GORM annotation is present.
	GormMessageOptions *model.GormMessageOptions
}

// GoField is a message field with resolved Go naming and type info.
// It embeds model.Field to preserve proto-level metadata (field number,
// type kind, scalar kind, cardinality) needed by marshal/unmarshal generation.
// GormMessageOptions carries the owning message's GORM annotation so that
// tag providers can make per-field decisions without needing the parent GoMessage.
type GoField struct {
	model.Field
	// GoName is the Go struct field name after CamelCase and conflict resolution.
	GoName string
	// GoType is the fully resolved Go type string (e.g. "int32", "[]byte", "*Person").
	GoType string
	// ElemGoType is the Go type name of the element for repeated fields
	// (e.g. "Status" for []Status, "Address" for []*Address).
	// Empty for non-repeated fields.
	ElemGoType string
	// GormMessageOptions is copied from the owning message's GormOptions.
	// Nil means the message has no GORM annotation and no gorm tag should be generated.
	GormMessageOptions *model.GormMessageOptions
}

// GoEnum is a flattened enum with resolved Go names.
type GoEnum struct {
	// GoName is the Go type name (e.g. "Person_Status" for nested enums).
	GoName string
	// Comment lines from the proto source.
	Comment model.Comment
	// Values are the enum constants.
	Values []GoEnumValue
}

// GoEnumValue is a single enum constant.
type GoEnumValue struct {
	// GoName is the Go constant name (e.g. "Person_Status_STATUS_ACTIVE").
	GoName string
	// Number is the proto enum value number.
	Number int32
	// Comment lines from the proto source.
	Comment model.Comment
}

// GoService is a service definition with resolved Go names.
type GoService struct {
	// GoName is the Go interface name (e.g. "UserService").
	GoName string
	// Comment lines from the proto source.
	Comment model.Comment
	// Methods are the rpc methods of the service.
	Methods []GoRPCMethod
}

// GoRPCMethod is a single rpc method with resolved Go type names.
type GoRPCMethod struct {
	// GoName is the Go method name (e.g. "CreateUser").
	GoName string
	// RequestType is the Go type name of the request message (e.g. "CreateUserRequest").
	RequestType string
	// ResponseType is the Go type name of the response message (e.g. "CreateUserResponse").
	ResponseType string
	// Comment lines from the proto source.
	Comment model.Comment
}

// Flatten converts a model.File into a GoFile by:
//  1. Extracting the Go package name from go_package option.
//  2. Recursively flattening nested messages and enums to top level.
//  3. Resolving Go type names, field names, and enum value names via the naming package.
//  4. Resolving field name conflicts within each message.
func Flatten(file model.File) GoFile {
	pkg := goPackageName(file.GoPackage)

	var messages []GoMessage
	var enums []GoEnum
	flattenMessages(file.Messages, file.Package, &messages, &enums)
	flattenEnums(file.Enums, file.Package, &enums)

	return GoFile{
		Source:       file.Path,
		Package:      pkg,
		ProtoPackage: file.Package,
		Messages:     messages,
		Enums:        enums,
		Services:     flattenServices(file.Services, file.Package),
	}
}

// goPackageName extracts the package name from a go_package option value.
// go_package can be "path/to/pkg;name" or "path/to/pkg" — we want the name part.
func goPackageName(goPackage string) string {
	if goPackage == "" {
		return ""
	}
	// If there's a semicolon, the part after it is the package name.
	for i := len(goPackage) - 1; i >= 0; i-- {
		if goPackage[i] == ';' {
			return goPackage[i+1:]
		}
	}
	// Otherwise, use the last path element.
	for i := len(goPackage) - 1; i >= 0; i-- {
		if goPackage[i] == '/' {
			return goPackage[i+1:]
		}
	}
	return goPackage
}

func flattenMessages(msgs []model.Message, pkgName string, outMsgs *[]GoMessage, outEnums *[]GoEnum) {
	for _, msg := range msgs {
		goName := naming.GoTypeName(msg.FullName, pkgName)

		// Collect raw field names for conflict resolution.
		rawFieldNames := make([]string, len(msg.Fields))
		for i, f := range msg.Fields {
			rawFieldNames[i] = naming.GoFieldName(f.Name)
		}
		resolvedNames := naming.ResolveFieldNames(rawFieldNames)

		fields := make([]GoField, len(msg.Fields))
		for i, f := range msg.Fields {
			fields[i] = GoField{
				Field:              f,
				GoName:             resolvedNames[i],
				GoType:             resolveGoType(f, pkgName),
				ElemGoType:         resolveElemGoType(f, pkgName),
				GormMessageOptions: msg.GormOptions,
			}
		}

		*outMsgs = append(*outMsgs, GoMessage{
			GoName:             goName,
			Comment:            msg.LeadingComment,
			Fields:             fields,
			UpdateSource:       msg.UpdateSource,
			CreateSource:       msg.CreateSource,
			ConditionFields:    conditionFieldsFor(msg),
			RequiredFields:     requiredFieldsFor(msg),
			GormMessageOptions: msg.GormOptions,
		})

		// Recurse into nested messages and enums.
		flattenMessages(msg.Messages, pkgName, outMsgs, outEnums)
		flattenEnums(msg.Enums, pkgName, outEnums)
	}
}

// conditionFieldsFor returns the condition field names for an update message.
// These are carried from the original update_message annotation via update_source_opts,
// and stored directly in msg.ConditionFields by the parser.
// Returns nil for non-update messages (UpdateSource == "").
func conditionFieldsFor(msg model.Message) []string {
	if msg.UpdateSource == "" {
		return nil
	}
	return msg.ConditionFields
}

// requiredFieldsFor returns the required field names for a create message.
// In the generated create proto, required_fields are rendered as non-optional
// scalar/enum fields. Message-type fields are excluded because they are
// inherently nullable (always *T in Go, no optional keyword needed) and
// cannot be reliably detected as required via the non-optional heuristic.
// Returns nil for non-create messages (CreateSource == "").
func requiredFieldsFor(msg model.Message) []string {
	if msg.CreateSource == "" {
		return nil
	}
	var result []string
	for _, f := range msg.Fields {
		if !f.Optional && f.Cardinality != model.CardinalityRepeated && f.Type.Kind != model.FieldKindMessage {
			result = append(result, f.Name)
		}
	}
	return result
}

// flattenServices converts model.Service definitions to GoService values with
// resolved Go names. Service names and rpc method names use naming.GoTypeName
// for consistency with message and enum name resolution.
func flattenServices(services []model.Service, pkgName string) []GoService {
	if len(services) == 0 {
		return nil
	}
	result := make([]GoService, len(services))
	for i, svc := range services {
		methods := make([]GoRPCMethod, len(svc.RPCs))
		for j, rpc := range svc.RPCs {
			methods[j] = GoRPCMethod{
				GoName:       rpc.Name,
				RequestType:  naming.GoTypeName(rpc.RequestType, pkgName),
				ResponseType: naming.GoTypeName(rpc.ResponseType, pkgName),
				Comment:      rpc.LeadingComment,
			}
		}
		result[i] = GoService{
			GoName:  naming.GoTypeName(svc.FullName, pkgName),
			Comment: svc.LeadingComment,
			Methods: methods,
		}
	}
	return result
}

func flattenEnums(enums []model.Enum, pkgName string, out *[]GoEnum) {
	for _, e := range enums {
		goName := naming.GoTypeName(e.FullName, pkgName)

		values := make([]GoEnumValue, len(e.Values))
		for i, v := range e.Values {
			values[i] = GoEnumValue{
				GoName:  naming.GoEnumValueName(goName, v.Name),
				Number:  v.Number,
				Comment: v.LeadingComment,
			}
		}

		*out = append(*out, GoEnum{
			GoName:  goName,
			Comment: e.LeadingComment,
			Values:  values,
		})
	}
}

// resolveGoType returns the Go type string for a field, including pointer
// prefix for message types and slice prefix for repeated fields.
func resolveGoType(f model.Field, pkgName string) string {
	var base string
	switch f.Type.Kind {
	case model.FieldKindScalar:
		base = naming.GoScalarType(f.Type.Scalar)
	case model.FieldKindEnum:
		base = naming.GoTypeName(f.Type.FullName, pkgName)
	case model.FieldKindMessage:
		// Message fields are always pointers; the optional keyword has no additional
		// effect on message types (nil already represents "not set"). We handle both
		// cardinalities here and return early to skip the Optional branch below,
		// avoiding **T for optional message fields.
		base = "*" + naming.GoTypeName(f.Type.FullName, pkgName)
		if f.Cardinality == model.CardinalityRepeated {
			return "[]" + base
		}
		return base
	default:
		panic(fmt.Sprintf("resolveGoType: unexpected FieldKind %v for field %q", f.Type.Kind, f.Name))
	}

	if f.Cardinality == model.CardinalityRepeated {
		return "[]" + base
	}
	if f.Optional {
		return "*" + base
	}
	return base
}

// resolveElemGoType returns the Go element type name for repeated fields
// (e.g. "Status" for []Status, "Address" for []*Address).
// Returns empty string for non-repeated fields.
func resolveElemGoType(f model.Field, pkgName string) string {
	if f.Cardinality != model.CardinalityRepeated {
		return ""
	}
	switch f.Type.Kind {
	case model.FieldKindScalar:
		return naming.GoScalarType(f.Type.Scalar)
	case model.FieldKindEnum:
		return naming.GoTypeName(f.Type.FullName, pkgName)
	case model.FieldKindMessage:
		return naming.GoTypeName(f.Type.FullName, pkgName)
	default:
		panic(fmt.Sprintf("resolveElemGoType: unexpected FieldKind %v for field %q", f.Type.Kind, f.Name))
	}
}
