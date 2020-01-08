package task

import (
	"errors"
	"fmt"
	"os"
	"text/tabwriter"

	"github.com/docker/swarmkit/api"
	"github.com/docker/swarmkit/cmd/swarmctl/common"
	"github.com/spf13/cobra"
)

var (
	listCmd = &cobra.Command{
		Use:   "ls",
		Short: "List tasks",
		RunE: func(cmd *cobra.Command, args []string) error {
			if len(args) != 0 {
				return errors.New("ls command takes no arguments")
			}

			flags := cmd.Flags()

			all, err := flags.GetBool("all")
			if err != nil {
				return err
			}

			quiet, err := flags.GetBool("quiet")
			if err != nil {
				return err
			}

			c, err := common.Dial(cmd)
			if err != nil {
				return err
			}
			r, err := c.ListTasks(common.Context(cmd), &api.ListTasksRequest{})
			if err != nil {
				return err
			}
			res := common.NewResolver(cmd, c)

			var output func(t *api.Task)

			if !quiet {
				w := tabwriter.NewWriter(os.Stdout, 0, 4, 2, ' ', 0)
				defer func() {
					// Ignore flushing errors - there's nothing we can do.
					_ = w.Flush()
				}()
				common.PrintHeader(w, "ID", "Service", "Desired State", "Last State", "Node")
				output = func(t *api.Task) {
					fmt.Fprintf(w, "%s\t%s.%d\t%s\t%s %s\t%s\n",
						t.ID,
						res.Resolve(api.Service{}, t.ServiceID),
						t.Slot,
						t.DesiredState.String(),
						t.Status.State.String(),
						common.TimestampAgo(t.Status.Timestamp),
						res.Resolve(api.Node{}, t.NodeID),
					)
				}
			} else {
				output = func(t *api.Task) { fmt.Println(t.ID) }
			}

			for _, t := range r.Tasks {
				if all || t.DesiredState <= api.TaskStateRunning {
					output(t)
				}
			}
			return nil
		},
	}
)

func init() {
	listCmd.Flags().BoolP("all", "a", false, "Show all tasks (default shows just running)")
	listCmd.Flags().BoolP("quiet", "q", false, "Only display IDs")
}
