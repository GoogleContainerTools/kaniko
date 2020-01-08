package node

import "github.com/spf13/cobra"

var (
	// Cmd exposes the top-level node command.
	Cmd = &cobra.Command{
		Use:   "node",
		Short: "Node management",
	}
)

func init() {
	Cmd.AddCommand(
		activateCmd,
		demoteCmd,
		drainCmd,
		inspectCmd,
		listCmd,
		pauseCmd,
		promoteCmd,
		removeCmd,
		updateCmd,
	)
}
