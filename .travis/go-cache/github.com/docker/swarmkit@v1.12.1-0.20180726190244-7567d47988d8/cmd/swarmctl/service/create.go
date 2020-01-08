package service

import (
	"errors"
	"fmt"

	"github.com/docker/swarmkit/api"
	"github.com/docker/swarmkit/cmd/swarmctl/common"
	"github.com/docker/swarmkit/cmd/swarmctl/service/flagparser"
	"github.com/spf13/cobra"
)

var (
	createCmd = &cobra.Command{
		Use:   "create",
		Short: "Create a service",
		RunE: func(cmd *cobra.Command, args []string) error {
			if len(args) != 0 {
				return errors.New("create command takes no arguments")
			}

			if !cmd.Flags().Changed("name") || !cmd.Flags().Changed("image") {
				return errors.New("--name and --image are mandatory")
			}

			c, err := common.Dial(cmd)
			if err != nil {
				return err
			}

			spec := &api.ServiceSpec{
				Mode: &api.ServiceSpec_Replicated{
					Replicated: &api.ReplicatedService{
						Replicas: 1,
					},
				},
				Task: api.TaskSpec{
					Runtime: &api.TaskSpec_Container{
						Container: &api.ContainerSpec{},
					},
				},
			}

			if err := flagparser.Merge(cmd, spec, c); err != nil {
				return err
			}

			if err := flagparser.ParseAddSecret(cmd, spec, "secret"); err != nil {
				return err
			}

			r, err := c.CreateService(common.Context(cmd), &api.CreateServiceRequest{Spec: spec})
			if err != nil {
				return err
			}
			fmt.Println(r.Service.ID)
			return nil
		},
	}
)

func init() {
	flags := createCmd.Flags()
	flagparser.AddServiceFlags(flags)
	flags.String("mode", "replicated", "one of replicated, global")
	flags.StringSlice("secret", nil, "add a secret from swarm")
}
