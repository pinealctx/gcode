package naming

import "github.com/pinealctx/gcode/internal/model"

// GoScalarType returns the Go type string for a protobuf scalar kind.
func GoScalarType(kind model.ScalarKind) string {
	switch kind {
	case model.ScalarDouble:
		return "float64"
	case model.ScalarFloat:
		return "float32"
	case model.ScalarInt32, model.ScalarSint32, model.ScalarSfixed32:
		return "int32"
	case model.ScalarInt64, model.ScalarSint64, model.ScalarSfixed64:
		return "int64"
	case model.ScalarUint32, model.ScalarFixed32:
		return "uint32"
	case model.ScalarUint64, model.ScalarFixed64:
		return "uint64"
	case model.ScalarBool:
		return "bool"
	case model.ScalarString:
		return "string"
	case model.ScalarBytes:
		return "[]byte"
	default:
		return "UNKNOWN"
	}
}
