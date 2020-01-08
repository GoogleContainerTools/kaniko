package config

import "github.com/spf13/cobra"

var (
	// Cmd exposes the top-level service command.
	Cmd = &cobra.Command{
		Use:     "config",
		Aliases: nil,
		Short:   "Config management",
	}
)

func init() {
	Cmd.AddCommand(
		inspectCmd,
		listCmd,
		createCmd,
		removeCmd,
	)
}
