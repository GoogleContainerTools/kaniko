package version

import (
	"errors"

	"github.com/spf13/cobra"
)

var (
	// Cmd can be added to other commands to provide a version subcommand with
	// the correct version of swarm.
	Cmd = &cobra.Command{
		Use:   "version",
		Short: "Print version number of swarm",
		RunE: func(cmd *cobra.Command, args []string) error {
			if len(args) != 0 {
				return errors.New("version command takes no arguments")
			}

			PrintVersion()
			return nil
		},
	}
)
