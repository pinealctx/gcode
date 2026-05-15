package model

// Syntax identifies the protobuf syntax version for an input file.
type Syntax string

const (
	SyntaxProto3 Syntax = "proto3"
)

// Cardinality describes whether a field is singular or repeated.
type Cardinality string

const (
	CardinalitySingular Cardinality = "singular"
	CardinalityRepeated Cardinality = "repeated"
)

// FieldKind identifies the high-level category of a field type.
type FieldKind string

const (
	FieldKindScalar  FieldKind = "scalar"
	FieldKindEnum    FieldKind = "enum"
	FieldKindMessage FieldKind = "message"
)

// ScalarKind identifies a protobuf scalar type.
type ScalarKind string

const (
	ScalarDouble   ScalarKind = "double"
	ScalarFloat    ScalarKind = "float"
	ScalarInt32    ScalarKind = "int32"
	ScalarInt64    ScalarKind = "int64"
	ScalarUint32   ScalarKind = "uint32"
	ScalarUint64   ScalarKind = "uint64"
	ScalarSint32   ScalarKind = "sint32"
	ScalarSint64   ScalarKind = "sint64"
	ScalarFixed32  ScalarKind = "fixed32"
	ScalarFixed64  ScalarKind = "fixed64"
	ScalarSfixed32 ScalarKind = "sfixed32"
	ScalarSfixed64 ScalarKind = "sfixed64"
	ScalarBool     ScalarKind = "bool"
	ScalarString   ScalarKind = "string"
	ScalarBytes    ScalarKind = "bytes"
)

// File represents a parsed protobuf file after normalization into the stage1 semantic model.
type File struct {
	Path           string
	Syntax         Syntax
	Package        string
	GoPackage      string
	Imports        []Import
	Messages       []Message
	Enums          []Enum
	Services       []Service
	IsSchema       bool // true when the file carries option (gcode.schema) = {};
	LeadingComment Comment
	Location       Location
}

// Import represents a file import statement.
type Import struct {
	Path           string
	IsPublic       bool
	LeadingComment Comment
	Location       Location
}

// Message represents a protobuf message declaration.
type Message struct {
	Name        string
	FullName    string
	Fields      []Field
	Messages    []Message
	Enums       []Enum
	GormOptions *GormMessageOptions
	// UpdateOptions holds zero or more update_message annotations declared on this message.
	// Each entry drives generation of one derived update proto message.
	// nil and empty slice are equivalent; callers should use len() to check.
	UpdateOptions []UpdateMessageOptions
	// CreateOptions holds zero or more create_message annotations declared on this message.
	// Each entry drives generation of one derived create proto message.
	// nil and empty slice are equivalent; callers should use len() to check.
	CreateOptions []CreateMessageOptions

	// UpdateSource is non-empty when this message was generated from an update_message annotation.
	// Its value is the source message name (e.g. "User"), used by stage 2 to generate ToMap()
	// and inherit validate rules. Written by the gen-proto pipeline; parser does not set this.
	UpdateSource string
	// ConditionFields lists the condition_fields from the update_source_opts annotation.
	// These are the WHERE-condition fields carried from the original update_message annotation.
	// Only populated when UpdateSource is non-empty. When nil/empty, no fields are treated
	// as condition fields, meaning all fields appear in ToMap() output and ApplyTo().
	ConditionFields []string
	// CreateSource is non-empty when this message was generated from a create_message annotation.
	// Its value is the source message name (e.g. "User"), used by stage 2 to inherit validate rules.
	// Written by the gen-proto pipeline; parser does not set this.
	CreateSource   string
	LeadingComment Comment
	Location       Location
}

// Enum represents a protobuf enum declaration.
type Enum struct {
	Name           string
	FullName       string
	Values         []EnumValue
	LeadingComment Comment
	Location       Location
}

// EnumValue represents a protobuf enum value declaration.
type EnumValue struct {
	Name           string
	Number         int32
	LeadingComment Comment
	Location       Location
}

// Service represents a protobuf service declaration.
type Service struct {
	Name           string
	FullName       string
	RPCs           []RPC
	LeadingComment Comment
	Location       Location
}

// RPC represents a single rpc method within a service.
// RequestType and ResponseType hold the proto full names of the request and
// response message types (e.g. "example.CreateUserRequest"), which the
// transform layer resolves to Go type names via naming.GoTypeName.
type RPC struct {
	Name           string
	RequestType    string // proto message full name
	ResponseType   string // proto message full name
	LeadingComment Comment
	Location       Location
}

// Field represents a protobuf message field declaration.
type Field struct {
	Name        string
	Number      int
	Cardinality Cardinality
	// Optional is true when the field carries an explicit proto3 optional keyword,
	// meaning the field has presence semantics and should be represented as a
	// pointer type in Go. Only applies to scalar and enum fields; bytes and
	// message fields are excluded.
	Optional bool
	// HasPresence is true when the field has explicit presence (proto3 optional)
	// but is NOT represented as a pointer (i.e. optional bytes). Such fields
	// must be serialized when non-nil even if their length is zero.
	HasPresence bool
	Type        FieldType
	JSONName    string
	GormOptions *GormFieldOptions
	JSONOptions *JSONFieldOptions
	// ValidateOptions holds buf/validate field constraints parsed from proto annotations.
	// nil means no constraints are declared for this field.
	ValidateOptions *ValidateFieldOptions
	// ValidateMessage overrides the default error message for all constraints on this field
	// when non-empty. Corresponds to the (gcode.field).validate_message annotation.
	ValidateMessage string
	LeadingComment  Comment
	Location        Location
}

// FieldType describes the type of a field without binding it to a target language.
type FieldType struct {
	Kind     FieldKind
	Scalar   ScalarKind
	Name     string
	FullName string
}

// GormMessageOptions holds GORM-related annotations for a message.
type GormMessageOptions struct {
	Table string
}

// UpdateMessageOptions holds the parameters of a single update_message annotation.
// It drives generation of one derived update proto message from the annotated message.
type UpdateMessageOptions struct {
	// Name is the generated message name (e.g. "UserUpdateByID").
	Name string
	// ConditionFields are the WHERE-condition fields; generated without optional wrapper
	// (always non-pointer in the generated Go struct).
	ConditionFields []string
	// IgnoreFields are excluded from the generated message.
	IgnoreFields []string
}

// CreateMessageOptions holds the parameters of a single create_message annotation.
// It drives generation of one derived create proto message from the annotated message.
type CreateMessageOptions struct {
	// Name is the generated message name (e.g. "UserCreate").
	Name string
	// IgnoreFields are excluded from the generated message.
	IgnoreFields []string
	// RequiredFields are forced to non-optional even if the source field is optional.
	RequiredFields []string
}

// GormFieldOptions holds GORM-related annotations for a field.
type GormFieldOptions struct {
	Column string // empty means use proto field name as default column name
}

// JSONFieldOptions holds JSON tag annotations for a field.
type JSONFieldOptions struct {
	Omitempty bool
	Ignore    bool
}

// Comment preserves normalized comment lines associated with a declaration.
type Comment struct {
	Lines []string
}

// Location identifies a source position in a proto file.
type Location struct {
	Path   string
	Line   int
	Column int
}

// ValidateFieldOptions holds buf/validate field constraints extracted from proto annotations.
// Numeric constraints are stored by scalar kind to avoid precision loss:
//   - signed integers (int32/int64/sint32/sint64/sfixed32/sfixed64) → int64
//   - unsigned integers (uint32/uint64/fixed32/fixed64)             → uint64
//   - floats (float/double)                                         → float64 (no in/not_in)
//
// nil pointer fields mean the constraint is not set.
// A nil ValidateFieldOptions on Field means no constraints are declared.
type ValidateFieldOptions struct {
	// General
	Required bool

	// String constraints (len semantics: byte length, i.e. len(s))
	MinLen   *uint64
	MaxLen   *uint64
	Pattern  string
	Email    bool
	URI      bool
	InStr    []string // string in set
	NotInStr []string // string not_in set

	// Signed integer constraints
	GTInt    *int64
	GTEInt   *int64
	LTInt    *int64
	LTEInt   *int64
	InInt    []int64
	NotInInt []int64

	// Unsigned integer constraints
	GTUint    *uint64
	GTEUint   *uint64
	LTUint    *uint64
	LTEUint   *uint64
	InUint    []uint64
	NotInUint []uint64

	// Float constraints (in/not_in not supported for float/double)
	GTFloat  *float64
	GTEFloat *float64
	LTFloat  *float64
	LTEFloat *float64

	// Repeated field constraints
	MinItems *uint64
	MaxItems *uint64
	Items    *ValidateFieldOptions // element-level constraints

	// Enum constraints
	DefinedOnly bool
	NotInEnum   []int32 // enum not_in set
}
