package task

import (
	"errors"
	"fmt"

	"github.com/docker/swarmkit/api"
	"github.com/docker/swarmkit/cmd/swarmctl/common"
	"github.com/spf13/cobra"
)

var (
	removeCmd = &cobra.Command{
		Use:     "remove <task ID>",
		Short:   "Remove a task",
		Aliases: []string{"rm"},
		RunE: func(cmd *cobra.Command, args []string) error {
			if len(args) == 0 {
				return errors.New("task ID missing")
			}

			if len(args) > 1 {
				return errors.New("remove command takes exactly 1 argument")
			}

			c, err := common.Dial(cmd)
			if err != nil {
				return err
			}

			_, err = c.RemoveTask(common.Context(cmd), &api.RemoveTaskRequest{TaskID: args[0]})
			if err != nil {
				return err
			}
			fmt.Println(args[0])
			return nil
		},
	}
)
