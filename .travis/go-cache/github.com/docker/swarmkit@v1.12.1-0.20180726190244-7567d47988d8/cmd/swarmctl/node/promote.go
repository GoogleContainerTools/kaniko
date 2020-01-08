package node

import (
	"fmt"

	"github.com/docker/swarmkit/api"
	"github.com/spf13/cobra"
)

var (
	promoteCmd = &cobra.Command{
		Use:   "promote <node ID>",
		Short: "Promote a node to a manager",
		RunE: func(cmd *cobra.Command, args []string) error {
			if err := changeNodeRole(cmd, args, api.NodeRoleManager); err != nil {
				if err == errNoChange {
					return fmt.Errorf("Node %s is already a manager", args[0])
				}
				return err
			}
			return nil
		},
	}
)
