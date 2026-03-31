// gen regenerates all static testdata snapshots under testdata/compat/dao/.
// Run from the module root:
//
//	go run ./testdata/compat/gen/
//
// Step 1: gen-proto generates intermediate update/create proto files into proto/.
// Step 2: gcode generates all Go DAO/RPC files from proto/ into dao/.
package main

import (
	"context"
	"fmt"
	"os"

	"github.com/pinealctx/gcode/internal/app"
)

func main() {
	if err := run(); err != nil {
		fmt.Fprintf(os.Stderr, "gen: %v\n", err)
		os.Exit(1)
	}
}

func run() error {
	ctx := context.Background()

	// Step 1: generate intermediate update/create proto files into proto/.
	if err := app.RunGenProto(ctx, []string{"-in", "testdata/compat/proto"}); err != nil {
		return fmt.Errorf("gen-proto: %w", err)
	}

	// Step 2: generate all Go DAO/RPC files from proto/ into dao/.
	if err := app.Run(ctx, []string{"-in", "testdata/compat/proto", "-out", "testdata/compat/dao"}); err != nil {
		return fmt.Errorf("gcode: %w", err)
	}

	return nil
}
