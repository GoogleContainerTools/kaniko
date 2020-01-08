package main

import (
	"github.com/moby/buildkit/cmd/buildctl/debug"
	"github.com/urfave/cli"
)

var debugCommand = cli.Command{
	Name:  "debug",
	Usage: "debug utilities",
	Subcommands: []cli.Command{
		debug.DumpLLBCommand,
		debug.DumpMetadataCommand,
		debug.WorkersCommand,
	},
}
