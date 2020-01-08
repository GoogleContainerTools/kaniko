package config

import (
	"errors"
	"fmt"

	"github.com/docker/swarmkit/api"
	"github.com/docker/swarmkit/cmd/swarmctl/common"
	"github.com/spf13/cobra"
)

var removeCmd = &cobra.Command{
	Use:     "remove <config ID or name>",
	Short:   "Remove a config",
	Aliases: []string{"rm"},
	RunE: func(cmd *cobra.Command, args []string) error {
		if len(args) == 0 {
			return errors.New("remove command takes a single config ID or name")
		}

		client, err := common.Dial(cmd)
		if err != nil {
			return err
		}

		config, err := getConfig(common.Context(cmd), client, args[0])
		if err != nil {
			return err
		}

		_, err = client.RemoveConfig(common.Context(cmd), &api.RemoveConfigRequest{ConfigID: config.ID})
		if err != nil {
			return err
		}
		fmt.Println(config.ID)
		return nil
	},
}
