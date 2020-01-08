package network

import (
	"errors"
	"fmt"

	"github.com/docker/swarmkit/api"
	"github.com/docker/swarmkit/cmd/swarmctl/common"
	"github.com/spf13/cobra"
)

var (
	removeCmd = &cobra.Command{
		Use:     "remove <network ID>",
		Short:   "Remove a network",
		Aliases: []string{"rm"},
		RunE: func(cmd *cobra.Command, args []string) error {
			if len(args) == 0 {
				return errors.New("network ID missing")
			}

			if len(args) > 1 {
				return errors.New("remove command takes exactly 1 argument")
			}

			c, err := common.Dial(cmd)
			if err != nil {
				return err
			}

			network, err := GetNetwork(common.Context(cmd), c, args[0])
			if err != nil {
				return err
			}
			_, err = c.RemoveNetwork(common.Context(cmd), &api.RemoveNetworkRequest{NetworkID: network.ID})
			if err != nil {
				return err
			}
			fmt.Println(args[0])
			return nil
		},
	}
)
