package app

import (
	"context"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"

	"github.com/pinealctx/x/ds"

	"github.com/pinealctx/gcode/internal/config"
	"github.com/pinealctx/gcode/internal/model"
	"github.com/pinealctx/gcode/internal/parser"
	"github.com/pinealctx/gcode/internal/source"
)

// RunGenProto implements the gen-proto subcommand: scans the input directory,
// generates *.entity.proto / *.create.proto / *.update.proto from schema files
// (those carrying option (gcode.schema) = {};) into the same directory.
func RunGenProto(ctx context.Context, args []string) error {
	cfg, err := config.ParseGenProto(args)
	if err != nil {
		return err
	}

	scanResult, err := source.Scan(cfg.InputDir)
	if err != nil {
		return fmt.Errorf("scan input directory: %w", err)
	}

	if len(scanResult.Files) == 0 {
		return fmt.Errorf("no .proto files found in %q: %w", cfg.InputDir, source.ErrNoProtoFiles)
	}

	// Exclude previously generated intermediate protos from parsing to avoid
	// symbol conflicts with their source schema files.
	inputFiles := filterSourceProtos(scanResult.Files)

	// Parse source files to discover schema files. Non-schema files that import
	// generated files not yet on disk will cause parse failures; in that case
	// fall back to parsing only .meta.proto files.
	firstPass, err := parser.Parse(ctx, []string{scanResult.ImportPath}, inputFiles)
	if err != nil {
		firstErr := err
		metaFiles := filterMetaProtos(inputFiles)
		if len(metaFiles) == 0 {
			return fmt.Errorf("parse proto files: %w", firstErr)
		}
		firstPass, err = parser.Parse(ctx, []string{scanResult.ImportPath}, metaFiles)
		if err != nil {
			return fmt.Errorf("full parse failed (%v); meta-only parse also failed: %w", firstErr, err)
		}
	}

	// Build type index and generate only from schema files.
	typeIdx := typeSourceIndex(firstPass)

	for _, f := range firstPass {
		if !f.IsSchema {
			continue
		}
		if err := generateIntermediateProtos(f, scanResult.ImportPath, typeIdx); err != nil {
			return fmt.Errorf("generate intermediate protos for %q: %w", f.Path, err)
		}
	}

	return nil
}

// generateIntermediateProtos generates *.entity.proto, *.create.proto, and
// *.update.proto for a schema file. Any previously generated intermediate proto
// files for this base name are removed first, so that deleting an option from
// the source proto does not leave stale generated files behind.
func generateIntermediateProtos(f model.File, outputDir string, typeIdx map[string]string) error {
	// Collect all messages with options (including nested).
	var updateMsgs []model.Message
	var createMsgs []model.Message
	collectMessages(f.Messages, &updateMsgs, &createMsgs)

	baseName := protoBaseName(f.Path)

	// Remove stale intermediate protos before (re)generating.
	entityPath := filepath.Join(outputDir, baseName+".entity.proto")
	updatePath := filepath.Join(outputDir, baseName+".update.proto")
	createPath := filepath.Join(outputDir, baseName+".create.proto")
	for _, stale := range []string{entityPath, updatePath, createPath} {
		if err := os.Remove(stale); err != nil && !errors.Is(err, fs.ErrNotExist) {
			return fmt.Errorf("remove stale %q: %w", stale, err)
		}
	}

	// Always generate entity proto from schema files.
	entityContent, err := buildEntityProto(f)
	if err != nil {
		return fmt.Errorf("build entity proto: %w", err)
	}
	if err := os.WriteFile(entityPath, []byte(entityContent), 0o644); err != nil { //nolint:gosec // generated source files should be world-readable
		return fmt.Errorf("write %q: %w", entityPath, err)
	}

	if len(updateMsgs) > 0 {
		extImports := collectExternalImports(updateMsgs, typeIdx, f.Path)
		content, err := buildUpdateProto(f, baseName, updateMsgs, extImports)
		if err != nil {
			return fmt.Errorf("build update proto: %w", err)
		}
		if err := os.WriteFile(updatePath, []byte(content), 0o644); err != nil { //nolint:gosec // generated source files should be world-readable
			return fmt.Errorf("write %q: %w", updatePath, err)
		}
	}

	if len(createMsgs) > 0 {
		extImports := collectExternalImports(createMsgs, typeIdx, f.Path)
		content, err := buildCreateProto(f, baseName, createMsgs, extImports)
		if err != nil {
			return fmt.Errorf("build create proto: %w", err)
		}
		if err := os.WriteFile(createPath, []byte(content), 0o644); err != nil { //nolint:gosec // generated source files should be world-readable
			return fmt.Errorf("write %q: %w", createPath, err)
		}
	}

	return nil
}

// collectMessages recursively collects messages with update/create options.
func collectMessages(msgs []model.Message, updateOut, createOut *[]model.Message) {
	for _, msg := range msgs {
		if len(msg.UpdateOptions) > 0 {
			*updateOut = append(*updateOut, msg)
		}
		if len(msg.CreateOptions) > 0 {
			*createOut = append(*createOut, msg)
		}
		collectMessages(msg.Messages, updateOut, createOut)
	}
}

// buildProtoHeader writes the common header shared by all generated proto files:
// the generated-code notice, syntax declaration, package, go_package option, and imports.
func buildProtoHeader(sb *strings.Builder, f model.File, imports []string) {
	sb.WriteString("// Code generated by gcode. DO NOT EDIT.\n\n")
	sb.WriteString("syntax = \"proto3\";\n\n")
	if f.Package != "" {
		fmt.Fprintf(sb, "package %s;\n\n", f.Package)
	}
	if f.GoPackage != "" {
		fmt.Fprintf(sb, "option go_package = %q;\n\n", f.GoPackage)
	}
	for _, imp := range imports {
		fmt.Fprintf(sb, "import %q;\n", imp)
	}
	if len(imports) > 0 {
		sb.WriteString("\n")
	}
}

// buildEntityProto generates a *.entity.proto containing all enum and message
// definitions from the schema file, with gorm annotations but without validate
// annotations. Create/update/update_source/create_source annotations are also excluded.
func buildEntityProto(f model.File) (string, error) {
	var sb strings.Builder

	// Collect imports: gcode/options.proto + source imports (excluding buf/validate and gcode/options).
	// References to other .meta.proto files are rewritten to .entity.proto to avoid
	// symbol duplication when both entity and source protos are compiled together.
	imports := []string{"gcode/options.proto"}
	for _, imp := range f.Imports {
		if imp.Path == "buf/validate/validate.proto" || imp.Path == "gcode/options.proto" {
			continue
		}
		entityPath := metaToEntityImport(imp.Path)
		imports = append(imports, entityPath)
	}

	buildProtoHeader(&sb, f, imports)

	// Write all top-level enums.
	for _, enum := range f.Enums {
		writeEnum(&sb, enum)
	}

	// Write all top-level messages (with nested), flattening nested types.
	for _, msg := range f.Messages {
		if err := writeEntityMessage(&sb, msg); err != nil {
			return "", err
		}
	}

	return sb.String(), nil
}

// writeEnum writes an enum definition to the builder.
func writeEnum(sb *strings.Builder, enum model.Enum) {
	fmt.Fprintf(sb, "enum %s {\n", enum.Name)
	for _, v := range enum.Values {
		fmt.Fprintf(sb, "  %s = %d;\n", v.Name, v.Number)
	}
	sb.WriteString("}\n\n")
}

// writeEntityMessage writes a message definition for entity proto (with gorm, without validate).
func writeEntityMessage(sb *strings.Builder, msg model.Message) error {
	fmt.Fprintf(sb, "message %s {\n", msg.Name)

	// Write gorm message option (table name).
	if msg.GormOptions != nil && msg.GormOptions.Table != "" {
		fmt.Fprintf(sb, "  option (gcode.message) = { gorm: { table: %q } };\n", msg.GormOptions.Table)
	}

	// Write fields with gorm but without validate.
	fieldNum := 1
	for _, f := range msg.Fields {
		line, err := entityFieldLine(f, fieldNum)
		if err != nil {
			return fmt.Errorf("field %q: %w", f.Name, err)
		}
		fmt.Fprintf(sb, "  %s\n", line)
		fieldNum++
	}

	sb.WriteString("}\n\n")

	// Write nested enums and messages (flattened to top level).
	for _, nested := range msg.Enums {
		writeEnum(sb, nested)
	}
	for _, nested := range msg.Messages {
		if err := writeEntityMessage(sb, nested); err != nil {
			return err
		}
	}

	return nil
}

// entityFieldLine renders a field for entity proto: gorm annotations, no validate.
// Preserves optional/repeated/HasPresence from the source field.
func entityFieldLine(f model.Field, fieldNum int) (string, error) {
	opts := gormFieldOpts(f)
	return formatFieldDecl(f, f.Optional || f.HasPresence, fieldNum, opts), nil
}

// formatFieldDecl renders a proto field declaration with the given options string.
// It dispatches on cardinality, kind, and optionality to produce correct syntax.
func formatFieldDecl(f model.Field, optional bool, fieldNum int, optStr string) string {
	typeStr := protoTypeName(f)
	if f.Cardinality == model.CardinalityRepeated {
		return fmt.Sprintf("repeated %s %s = %d%s;", typeStr, f.Name, fieldNum, optStr)
	}
	if f.Type.Kind == model.FieldKindMessage {
		return fmt.Sprintf("%s %s = %d%s;", typeStr, f.Name, fieldNum, optStr)
	}
	if optional {
		return fmt.Sprintf("optional %s %s = %d%s;", typeStr, f.Name, fieldNum, optStr)
	}
	return fmt.Sprintf("%s %s = %d%s;", typeStr, f.Name, fieldNum, optStr)
}

// gormFieldOpts returns the formatted gorm annotation option string for a field,
// or empty string if no gorm annotation is present.
func gormFieldOpts(f model.Field) string {
	if f.GormOptions != nil && f.GormOptions.Column != "" {
		return " [(gcode.field).gorm.column = " + strconv.Quote(f.GormOptions.Column) + "]"
	}
	return ""
}

// buildUpdateProto generates the content of a *.update.proto file.
// It imports the entity proto, buf/validate, and gcode/options.
func buildUpdateProto(f model.File, baseName string, msgs []model.Message, extraImports []string) (string, error) {
	imports := []string{
		baseName + ".entity.proto",
		"buf/validate/validate.proto",
		"gcode/options.proto",
	}
	imports = append(imports, extraImports...)

	var sb strings.Builder
	buildProtoHeader(&sb, f, imports)
	for _, msg := range msgs {
		for _, opt := range msg.UpdateOptions {
			content, err := buildUpdateMessage(msg, opt)
			if err != nil {
				return "", fmt.Errorf("message %q update_message %q: %w", msg.FullName, opt.Name, err)
			}
			sb.WriteString(content)
			sb.WriteString("\n")
		}
	}
	return sb.String(), nil
}

// buildCreateProto generates the content of a *.create.proto file.
// It imports the entity proto, buf/validate, and gcode/options.
func buildCreateProto(f model.File, baseName string, msgs []model.Message, extraImports []string) (string, error) {
	imports := []string{
		baseName + ".entity.proto",
		"buf/validate/validate.proto",
		"gcode/options.proto",
	}
	imports = append(imports, extraImports...)

	var sb strings.Builder
	buildProtoHeader(&sb, f, imports)
	for _, msg := range msgs {
		for _, opt := range msg.CreateOptions {
			content, err := buildCreateMessage(msg, opt)
			if err != nil {
				return "", fmt.Errorf("message %q create_message %q: %w", msg.FullName, opt.Name, err)
			}
			sb.WriteString(content)
			sb.WriteString("\n")
		}
	}

	return sb.String(), nil
}

// buildUpdateMessage generates a single update message block with validate annotations.
func buildUpdateMessage(msg model.Message, opt model.UpdateMessageOptions) (string, error) {
	if opt.Name == "" {
		return "", fmt.Errorf("update_message annotation on %q: %w", msg.FullName, ErrNameEmpty)
	}
	if !isProtoIdentifier(opt.Name) {
		return "", fmt.Errorf("update_message annotation on %q: name %q: %w", msg.FullName, opt.Name, ErrInvalidName)
	}
	ignoreSet := ds.NewSet(opt.IgnoreFields...)
	conditionSet := ds.NewSet(opt.ConditionFields...)

	var sb strings.Builder
	fmt.Fprintf(&sb, "message %s {\n", opt.Name)
	fmt.Fprintf(&sb, "  option (gcode.update_source) = %q;\n\n", msg.Name)

	fieldNum := 1
	for _, f := range msg.Fields {
		if ignoreSet.Contains(f.Name) {
			continue
		}
		line, err := derivedFieldLine(f, !conditionSet.Contains(f.Name), fieldNum)
		if err != nil {
			return "", fmt.Errorf("field %q: %w", f.Name, err)
		}
		fmt.Fprintf(&sb, "  %s\n", line)
		fieldNum++
	}
	sb.WriteString("}\n")
	return sb.String(), nil
}

// buildCreateMessage generates a single create message block with validate annotations.
func buildCreateMessage(msg model.Message, opt model.CreateMessageOptions) (string, error) {
	if opt.Name == "" {
		return "", fmt.Errorf("create_message annotation on %q: %w", msg.FullName, ErrNameEmpty)
	}
	if !isProtoIdentifier(opt.Name) {
		return "", fmt.Errorf("create_message annotation on %q: name %q: %w", msg.FullName, opt.Name, ErrInvalidName)
	}
	ignoreSet := ds.NewSet(opt.IgnoreFields...)
	requiredSet := ds.NewSet(opt.RequiredFields...)

	var sb strings.Builder
	fmt.Fprintf(&sb, "message %s {\n", opt.Name)
	fmt.Fprintf(&sb, "  option (gcode.create_source) = %q;\n\n", msg.Name)

	fieldNum := 1
	for _, f := range msg.Fields {
		if ignoreSet.Contains(f.Name) {
			continue
		}
		// required_fields forces non-optional; otherwise use original optionality.
		makeOptional := !requiredSet.Contains(f.Name)
		line, err := derivedFieldLine(f, makeOptional, fieldNum)
		if err != nil {
			return "", fmt.Errorf("field %q: %w", f.Name, err)
		}
		fmt.Fprintf(&sb, "  %s\n", line)
		fieldNum++
	}
	sb.WriteString("}\n")
	return sb.String(), nil
}

// derivedFieldLine renders a field for create/update proto with validate and
// gorm annotations. Gorm annotations are preserved so that generated Go code
// can use gorm column names in ToMap().
func derivedFieldLine(f model.Field, makeOptional bool, fieldNum int) (string, error) {
	var opts []string
	if f.GormOptions != nil && f.GormOptions.Column != "" {
		opts = append(opts, fmt.Sprintf("(gcode.field).gorm.column = %q", f.GormOptions.Column))
	}
	opts = appendValidateAnnotations(opts, f)

	optStr := ""
	if len(opts) > 0 {
		optStr = " [" + strings.Join(opts, ", ") + "]"
	}
	return formatFieldDecl(f, makeOptional, fieldNum, optStr), nil
}

// protoFieldLine renders a single proto field declaration (used by tests).
// makeOptional=true adds the "optional" keyword for scalar and enum fields.
// Message-type fields never get "optional" (they are inherently nullable in proto3).
// Repeated fields keep "repeated" regardless of makeOptional.
// Field-level gcode annotations (gorm.column) are preserved in the output.
func protoFieldLine(f model.Field, makeOptional bool, fieldNum int) (string, error) {
	return formatFieldDecl(f, makeOptional, fieldNum, gormFieldOpts(f)), nil
}

// appendValidateAnnotations renders buf.validate field constraints as proto
// annotation text and appends them to opts. Returns the modified slice.
func appendValidateAnnotations(opts []string, f model.Field) []string {
	v := f.ValidateOptions
	if v == nil {
		return opts
	}

	// Top-level required constraint applies to any field type (bytes, message, etc.).
	if v.Required {
		opts = append(opts, "(buf.validate.field).required = true")
	}

	switch f.Type.Kind {
	case model.FieldKindScalar:
		switch f.Type.Scalar {
		case model.ScalarString:
			opts = appendStringValidate(opts, v)
		case model.ScalarBytes:
			opts = appendBytesValidate(opts, v)
		case model.ScalarInt32, model.ScalarInt64, model.ScalarSint32, model.ScalarSint64,
			model.ScalarSfixed32, model.ScalarSfixed64:
			opts = appendSignedIntValidate(opts, v, f.Type.Scalar)
		case model.ScalarUint32, model.ScalarUint64, model.ScalarFixed32, model.ScalarFixed64:
			opts = appendUnsignedIntValidate(opts, v, f.Type.Scalar)
		case model.ScalarFloat:
			opts = appendFloatValidate(opts, v, "float")
		case model.ScalarDouble:
			opts = appendFloatValidate(opts, v, "double")
		}
	case model.FieldKindEnum:
		if v.DefinedOnly {
			opts = append(opts, "(buf.validate.field).enum.defined_only = true")
		}
	case model.FieldKindMessage:
		// required is already handled above.
	}

	// Repeated constraints apply regardless of element type.
	if v.MinItems != nil {
		opts = append(opts, fmt.Sprintf("(buf.validate.field).repeated.min_items = %d", *v.MinItems))
	}
	if v.MaxItems != nil {
		opts = append(opts, fmt.Sprintf("(buf.validate.field).repeated.max_items = %d", *v.MaxItems))
	}
	if v.Items != nil && f.Cardinality == model.CardinalityRepeated {
		opts = appendItemsValidate(opts, v.Items)
	}

	return opts
}

// appendStringValidate appends string constraint annotations.
func appendStringValidate(opts []string, v *model.ValidateFieldOptions) []string {
	if v.MinLen != nil {
		opts = append(opts, fmt.Sprintf("(buf.validate.field).string.min_len = %d", *v.MinLen))
	}
	if v.MaxLen != nil {
		opts = append(opts, fmt.Sprintf("(buf.validate.field).string.max_len = %d", *v.MaxLen))
	}
	if v.Pattern != "" {
		opts = append(opts, fmt.Sprintf("(buf.validate.field).string.pattern = %q", v.Pattern))
	}
	if v.Email {
		opts = append(opts, "(buf.validate.field).string.email = true")
	}
	if v.URI {
		opts = append(opts, "(buf.validate.field).string.uri = true")
	}
	for _, s := range v.InStr {
		opts = append(opts, fmt.Sprintf("(buf.validate.field).string.in = %q", s))
	}
	for _, s := range v.NotInStr {
		opts = append(opts, fmt.Sprintf("(buf.validate.field).string.not_in = %q", s))
	}
	return opts
}

// appendBytesValidate appends bytes constraint annotations.
func appendBytesValidate(opts []string, v *model.ValidateFieldOptions) []string {
	if v.MinLen != nil {
		opts = append(opts, fmt.Sprintf("(buf.validate.field).bytes.min_len = %d", *v.MinLen))
	}
	if v.MaxLen != nil {
		opts = append(opts, fmt.Sprintf("(buf.validate.field).bytes.max_len = %d", *v.MaxLen))
	}
	return opts
}

// appendSignedIntValidate appends signed integer constraint annotations.
func appendSignedIntValidate(opts []string, v *model.ValidateFieldOptions, scalar model.ScalarKind) []string {
	typeName := string(scalar)
	if v.GTInt != nil {
		opts = append(opts, fmt.Sprintf("(buf.validate.field).%s.gt = %d", typeName, *v.GTInt))
	}
	if v.GTEInt != nil {
		opts = append(opts, fmt.Sprintf("(buf.validate.field).%s.gte = %d", typeName, *v.GTEInt))
	}
	if v.LTInt != nil {
		opts = append(opts, fmt.Sprintf("(buf.validate.field).%s.lt = %d", typeName, *v.LTInt))
	}
	if v.LTEInt != nil {
		opts = append(opts, fmt.Sprintf("(buf.validate.field).%s.lte = %d", typeName, *v.LTEInt))
	}
	for _, n := range v.InInt {
		opts = append(opts, fmt.Sprintf("(buf.validate.field).%s.in = %d", typeName, n))
	}
	for _, n := range v.NotInInt {
		opts = append(opts, fmt.Sprintf("(buf.validate.field).%s.not_in = %d", typeName, n))
	}
	return opts
}

// appendUnsignedIntValidate appends unsigned integer constraint annotations.
func appendUnsignedIntValidate(opts []string, v *model.ValidateFieldOptions, scalar model.ScalarKind) []string {
	typeName := string(scalar)
	if v.GTUint != nil {
		opts = append(opts, fmt.Sprintf("(buf.validate.field).%s.gt = %d", typeName, *v.GTUint))
	}
	if v.GTEUint != nil {
		opts = append(opts, fmt.Sprintf("(buf.validate.field).%s.gte = %d", typeName, *v.GTEUint))
	}
	if v.LTUint != nil {
		opts = append(opts, fmt.Sprintf("(buf.validate.field).%s.lt = %d", typeName, *v.LTUint))
	}
	if v.LTEUint != nil {
		opts = append(opts, fmt.Sprintf("(buf.validate.field).%s.lte = %d", typeName, *v.LTEUint))
	}
	for _, n := range v.InUint {
		opts = append(opts, fmt.Sprintf("(buf.validate.field).%s.in = %d", typeName, n))
	}
	for _, n := range v.NotInUint {
		opts = append(opts, fmt.Sprintf("(buf.validate.field).%s.not_in = %d", typeName, n))
	}
	return opts
}

// appendFloatValidate appends float/double constraint annotations.
func appendFloatValidate(opts []string, v *model.ValidateFieldOptions, typeName string) []string {
	if v.GTFloat != nil {
		opts = append(opts, fmt.Sprintf("(buf.validate.field).%s.gt = %s", typeName, formatFloat(*v.GTFloat)))
	}
	if v.GTEFloat != nil {
		opts = append(opts, fmt.Sprintf("(buf.validate.field).%s.gte = %s", typeName, formatFloat(*v.GTEFloat)))
	}
	if v.LTFloat != nil {
		opts = append(opts, fmt.Sprintf("(buf.validate.field).%s.lt = %s", typeName, formatFloat(*v.LTFloat)))
	}
	if v.LTEFloat != nil {
		opts = append(opts, fmt.Sprintf("(buf.validate.field).%s.lte = %s", typeName, formatFloat(*v.LTEFloat)))
	}
	return opts
}

// appendItemsValidate appends repeated items constraint annotations.
func appendItemsValidate(opts []string, items *model.ValidateFieldOptions) []string {
	if items.MinLen != nil {
		opts = append(opts, fmt.Sprintf("(buf.validate.field).repeated.items.string.min_len = %d", *items.MinLen))
	}
	if items.GTEInt != nil {
		opts = append(opts, fmt.Sprintf("(buf.validate.field).repeated.items.int32.gte = %d", *items.GTEInt))
	}
	if items.DefinedOnly {
		opts = append(opts, "(buf.validate.field).repeated.items.enum.defined_only = true")
	}
	return opts
}

// formatFloat formats a float64 value, using integer representation for whole numbers
// (e.g. 1.0 instead of 1) to match proto syntax expectations.
func formatFloat(f float64) string {
	if f == float64(int64(f)) {
		return strconv.FormatFloat(f, 'f', 1, 64)
	}
	return strconv.FormatFloat(f, 'f', -1, 64)
}

// protoTypeName returns the proto type string for a field.
func protoTypeName(f model.Field) string {
	switch f.Type.Kind {
	case model.FieldKindScalar:
		return string(f.Type.Scalar)
	case model.FieldKindEnum:
		return f.Type.Name
	default:
		return f.Type.Name
	}
}

// typeSourceIndex builds a FullName → source proto file path mapping from all
// parsed files. Used by collectExternalImports to resolve cross-file type
// references in derived messages.
func typeSourceIndex(files []model.File) map[string]string {
	idx := make(map[string]string)
	for _, f := range files {
		collectTypeSources(f.Messages, f.Enums, f.Path, idx)
	}
	return idx
}

// collectTypeSources recursively maps message and enum FullNames to their
// source proto file path.
func collectTypeSources(msgs []model.Message, enums []model.Enum, filePath string, idx map[string]string) {
	for _, msg := range msgs {
		idx[msg.FullName] = filePath
		collectTypeSources(msg.Messages, msg.Enums, filePath, idx)
	}
	for _, enum := range enums {
		idx[enum.FullName] = filePath
	}
}

// collectExternalImports scans the fields of derived messages and returns
// the import paths of proto files that define referenced enum or message types
// from other files.
func collectExternalImports(msgs []model.Message, typeIdx map[string]string, selfPath string) []string {
	seen := make(map[string]bool)
	var imports []string
	for _, msg := range msgs {
		for _, f := range msg.Fields {
			if f.Type.Kind != model.FieldKindEnum && f.Type.Kind != model.FieldKindMessage {
				continue
			}
			srcFile, ok := typeIdx[f.Type.FullName]
			if !ok || srcFile == selfPath {
				continue
			}
			if !seen[srcFile] {
				seen[srcFile] = true
				imports = append(imports, srcFile)
			}
		}
	}
	return imports
}

// protoBaseName strips the .proto (and optional .meta) suffix from a relative
// path, returning just the base name used for generated file naming.
// e.g. "person.meta.proto" → "person", "user.proto" → "user".
func protoBaseName(relPath string) string {
	base := filepath.Base(relPath)
	name := strings.TrimSuffix(base, ".proto")
	return strings.TrimSuffix(name, ".meta")
}

// protoIdentifierRe matches a valid proto identifier: a letter or underscore
// followed by zero or more letters, digits, or underscores.
var protoIdentifierRe = regexp.MustCompile(`^[A-Za-z_][A-Za-z0-9_]*$`)

// isProtoIdentifier reports whether s is a valid proto identifier.
// Used to validate annotation-supplied message names before writing them
// into generated proto source.
func isProtoIdentifier(s string) bool {
	return protoIdentifierRe.MatchString(s)
}

// generatedProtoSuffixes lists filename suffixes of intermediate protos generated
// by gen-proto. These are excluded from the parser input to avoid symbol conflicts
// with their source schema files.
var generatedProtoSuffixes = []string{
	".entity.proto",
	".create.proto",
	".update.proto",
}

// filterSourceProtos removes previously generated intermediate proto files
// from the input list, keeping only source proto files for parsing.
func filterSourceProtos(files []string) []string {
	var result []string
	for _, f := range files {
		excluded := false
		for _, suffix := range generatedProtoSuffixes {
			if strings.HasSuffix(f, suffix) {
				excluded = true
				break
			}
		}
		if !excluded {
			result = append(result, f)
		}
	}
	return result
}

// filterMetaProtos returns only files ending in .meta.proto.
func filterMetaProtos(files []string) []string {
	var result []string
	for _, f := range files {
		if strings.HasSuffix(f, ".meta.proto") {
			result = append(result, f)
		}
	}
	return result
}

// metaToEntityImport rewrites a .meta.proto import path to .entity.proto.
// Non-meta imports are returned unchanged.
func metaToEntityImport(importPath string) string {
	if base, ok := strings.CutSuffix(importPath, ".meta.proto"); ok {
		return base + ".entity.proto"
	}
	return importPath
}
