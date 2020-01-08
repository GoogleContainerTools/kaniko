package service

import (
	"errors"
	"fmt"

	"github.com/docker/swarmkit/api"
	"github.com/docker/swarmkit/cmd/swarmctl/common"
	"github.com/spf13/cobra"
)

var (
	removeCmd = &cobra.Command{
		Use:     "remove <service ID>",
		Short:   "Remove a service",
		Aliases: []string{"rm"},
		RunE: func(cmd *cobra.Command, args []string) error {
			if len(args) == 0 {
				return errors.New("service ID missing")
			}

			if len(args) > 1 {
				return errors.New("remove command takes exactly 1 argument")
			}

			c, err := common.Dial(cmd)
			if err != nil {
				return err
			}

			for _, serviceName := range args {
				service, err := getService(common.Context(cmd), c, serviceName)
				if err != nil {
					return err
				}
				_, err = c.RemoveService(common.Context(cmd), &api.RemoveServiceRequest{ServiceID: service.ID})
				if err != nil {
					return err
				}
				fmt.Println(serviceName)
			}
			return nil
		},
	}
)
