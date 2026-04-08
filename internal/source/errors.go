package source

import "github.com/pinealctx/x/errorx"

// sourceTag is the domain discriminator for source-level errors.
type sourceTag struct{}

// ErrNoProtoFiles is returned when a directory scan finds no .proto files.
var ErrNoProtoFiles = errorx.Sentinel[sourceTag]("no .proto files found")
