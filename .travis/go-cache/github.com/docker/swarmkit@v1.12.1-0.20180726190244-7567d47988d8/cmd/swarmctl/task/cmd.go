package task

import "github.com/spf13/cobra"

var (
	// Cmd exposes the top-level task command.
	Cmd = &cobra.Command{
		Use:   "task",
		Short: "Task management",
	}
)

func init() {
	Cmd.AddCommand(
		listCmd,
		inspectCmd,
		removeCmd,
	)
}
