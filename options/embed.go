// Package options provides the embedded proto files for injection into
// protocompile's resolver, enabling users to import "gcode/options.proto"
// and "buf/validate/validate.proto" without any additional configuration.
package options

import _ "embed"

// GcodeOptionsProto is the raw content of gcode_options.proto, embedded at
// compile time. It is used by the parser to register a virtual file under the
// path "gcode/options.proto".
//
//go:embed gcode_options.proto
var GcodeOptionsProto []byte

// BufValidateProto is the raw content of buf/validate/validate.proto, embedded
// at compile time. It is used by the parser to resolve the import path
// "buf/validate/validate.proto" without requiring an external proto installation.
//
//go:embed buf_validate_validate.proto
var BufValidateProto []byte
