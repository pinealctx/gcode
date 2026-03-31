package naming

// reservedFieldNames contains method names that protoc-gen-go reserves on
// generated message types. Field names that collide with these (or their
// Get* accessors) are suffixed with '_' until unique.
//
// Ported from google.golang.org/protobuf/compiler/protogen.newMessage.
var reservedFieldNames = map[string]bool{
	"Reset":               true,
	"String":              true,
	"ProtoMessage":        true,
	"Marshal":             true,
	"Unmarshal":           true,
	"ExtensionRangeArray": true,
	"ExtensionMap":        true,
	"Descriptor":          true,
}

// ResolveFieldNames takes a slice of raw Go field names (already
// CamelCased) and returns a new slice with conflicts resolved. Each name
// is made unique against reserved names and previously assigned names,
// including their Get* accessors.
//
// The order of the input slice matters: earlier fields claim names first,
// matching protoc-gen-go behavior.
func ResolveFieldNames(rawNames []string) []string {
	used := make(map[string]bool, len(reservedFieldNames)+len(rawNames))
	for k, v := range reservedFieldNames {
		used[k] = v
	}

	resolved := make([]string, len(rawNames))
	for i, name := range rawNames {
		for used[name] || used["Get"+name] {
			name += "_"
		}
		used[name] = true
		used["Get"+name] = true
		resolved[i] = name
	}
	return resolved
}
