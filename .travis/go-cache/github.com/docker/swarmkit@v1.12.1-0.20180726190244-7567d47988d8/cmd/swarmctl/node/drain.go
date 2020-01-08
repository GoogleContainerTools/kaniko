package node

import (
	"fmt"

	"github.com/docker/swarmkit/api"
	"github.com/spf13/cobra"
)

var (
	drainCmd = &cobra.Command{
		Use:   "drain <node ID>",
		Short: "Drain a node",
		RunE: func(cmd *cobra.Command, args []string) error {
			if err := changeNodeAvailability(cmd, args, api.NodeAvailabilityDrain); err != nil {
				if err == errNoChange {
					return fmt.Errorf("Node %s was already drained", args[0])
				}
				return err
			}
			return nil
		},
	}
)
