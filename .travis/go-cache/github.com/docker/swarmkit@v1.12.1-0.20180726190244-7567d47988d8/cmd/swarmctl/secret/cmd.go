package secret

import "github.com/spf13/cobra"

var (
	// Cmd exposes the top-level service command.
	Cmd = &cobra.Command{
		Use:     "secret",
		Aliases: nil,
		Short:   "Secrets management",
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
