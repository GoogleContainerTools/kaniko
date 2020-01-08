package cluster

import "github.com/spf13/cobra"

var (
	// Cmd exposes the top-level cluster command.
	Cmd = &cobra.Command{
		Use:   "cluster",
		Short: "Cluster management",
	}
)

func init() {
	Cmd.AddCommand(
		inspectCmd,
		listCmd,
		updateCmd,
		unlockKeyCmd,
	)
}
