package naming

import "strings"

// GoTypeName returns the Go type name for a protobuf descriptor given its
// full name and the file's package name. Nested types use underscore
// separation (e.g., "Outer_Inner"), matching protoc-gen-go behavior.
//
// The algorithm: strip the package prefix from the full name, then apply
// GoCamelCase to the remainder. Since GoCamelCase converts '.' followed
// by an uppercase letter to '_', nested types naturally get the Outer_Inner
// format.
func GoTypeName(fullName, packageName string) string {
	name := strings.TrimPrefix(fullName, packageName+".")
	return GoCamelCase(name)
}

// GoFieldName returns the Go struct field name for a protobuf field name.
// It applies GoCamelCase to convert snake_case to CamelCase.
func GoFieldName(protoFieldName string) string {
	return GoCamelCase(protoFieldName)
}

// GoEnumValueName returns the Go constant name for a protobuf enum value.
// The format is ParentGoName_PROTO_VALUE_NAME (no CamelCase on the value
// name itself), matching protoc-gen-go behavior.
//
// parentGoName is the Go name of the containing type: the enum's Go name
// for top-level enums, or the message's Go name for enums nested in a
// message.
func GoEnumValueName(parentGoName, protoValueName string) string {
	return parentGoName + "_" + protoValueName
}
