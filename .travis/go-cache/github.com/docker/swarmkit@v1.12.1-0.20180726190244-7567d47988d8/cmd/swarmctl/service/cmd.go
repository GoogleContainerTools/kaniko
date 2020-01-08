package service

import "github.com/spf13/cobra"

var (
	// Cmd exposes the top-level service command.
	Cmd = &cobra.Command{
		Use:     "service",
		Aliases: []string{"svc"},
		Short:   "Service management",
	}
)

func init() {
	Cmd.AddCommand(
		inspectCmd,
		listCmd,
		createCmd,
		updateCmd,
		removeCmd,
		logsCmd,
	)
}
