package node

import (
	"fmt"

	"github.com/docker/swarmkit/api"
	"github.com/spf13/cobra"
)

var (
	demoteCmd = &cobra.Command{
		Use:   "demote <node ID>",
		Short: "Demote a node from a manager to a worker",
		RunE: func(cmd *cobra.Command, args []string) error {
			if err := changeNodeRole(cmd, args, api.NodeRoleWorker); err != nil {
				if err == errNoChange {
					return fmt.Errorf("Node %s is already a worker", args[0])
				}
				return err
			}
			return nil
		},
	}
)
