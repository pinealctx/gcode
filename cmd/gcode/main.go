package main

import (
	"context"
	"fmt"
	"log"
	"os"

	"github.com/pinealctx/gcode/internal/app"
	"github.com/pinealctx/gcode/internal/version"
	"github.com/urfave/cli/v2"
)

func main() {
	cliApp := &cli.App{
		Name:    "gcode",
		Usage:   "protobuf-based code generator for Go and TypeScript",
		Version: version.Version,
		Flags: []cli.Flag{
			&cli.StringFlag{
				Name:  "in",
				Usage: "input proto directory",
			},
			&cli.StringFlag{
				Name:  "out",
				Usage: "output directory",
			},
		},
		Action: func(c *cli.Context) error {
			if c.NumFlags() == 0 {
				return cli.ShowAppHelp(c)
			}
			args := daoFlags(c)
			return app.Run(c.Context, args)
		},
		Commands: []*cli.Command{
			{
				Name:  "version",
				Usage: "print version information",
				Action: func(*cli.Context) error {
					fmt.Println(version.String())
					return nil
				},
			},
			{
				Name:  "gen-proto",
				Usage: "generate entity/create/update proto files from schema (.meta.proto) files",
				Flags: []cli.Flag{
					&cli.StringFlag{
						Name:  "in",
						Usage: "input proto directory (generated files are written to the same directory)",
					},
				},
				Action: func(c *cli.Context) error {
					return app.RunGenProto(c.Context, subFlags(c))
				},
				HideHelp: true,
			},
			{
				Name:  "gen-ts",
				Usage: "generate TypeScript output from proto files",
				Flags: []cli.Flag{
					&cli.StringFlag{
						Name:  "in",
						Usage: "input proto directory",
					},
					&cli.StringFlag{
						Name:  "out",
						Usage: "output TypeScript directory",
					},
				},
				Action: func(c *cli.Context) error {
					return app.RunGenTS(c.Context, subFlags(c))
				},
				HideHelp: true,
			},
		},
	}

	if err := cliApp.RunContext(context.Background(), os.Args); err != nil {
		log.Fatal(err)
	}
}

// daoFlags reconstructs ["-in", value, "-out", value] from the parsed CLI context
// for the default gen-dao action, preserving compatibility with config.Parse.
func daoFlags(c *cli.Context) []string {
	var args []string
	if v := c.String("in"); v != "" {
		args = append(args, "-in", v)
	}
	if v := c.String("out"); v != "" {
		args = append(args, "-out", v)
	}
	return args
}

// subFlags reconstructs flag args from the CLI context for subcommands that
// still use the internal config.Parse flow.
func subFlags(c *cli.Context) []string {
	var args []string
	for _, name := range c.FlagNames() {
		if v := c.String(name); v != "" {
			args = append(args, "-"+name, v)
		}
	}
	return args
}
