package network

import "github.com/spf13/cobra"

var (
	// Cmd exposes the top-level network command
	Cmd = &cobra.Command{
		Use:   "network",
		Short: "Network management",
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
