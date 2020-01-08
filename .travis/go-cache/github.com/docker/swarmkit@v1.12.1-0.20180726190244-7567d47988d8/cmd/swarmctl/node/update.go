package node

import (
	"fmt"

	"github.com/spf13/cobra"
)

var (
	updateCmd = &cobra.Command{
		Use:   "update <node ID>",
		Short: "Update a node",
		RunE: func(cmd *cobra.Command, args []string) error {
			if err := updateNode(cmd, args); err != nil {
				if err == errNoChange {
					return fmt.Errorf("No change for node %s", args[0])
				}
				return err
			}
			return nil
		},
	}
)

func init() {
	flags := updateCmd.Flags()
	flags.StringSlice(flagLabel, nil, "node label (key=value)")
}
