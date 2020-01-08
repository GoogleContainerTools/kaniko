package network

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
		Short: "List networks",
		RunE: func(cmd *cobra.Command, args []string) error {
			if len(args) != 0 {
				return errors.New("ls command takes no arguments")
			}

			flags := cmd.Flags()

			quiet, err := flags.GetBool("quiet")
			if err != nil {
				return err
			}

			c, err := common.Dial(cmd)
			if err != nil {
				return err
			}
			r, err := c.ListNetworks(common.Context(cmd), &api.ListNetworksRequest{})
			if err != nil {
				return err
			}

			var output func(*api.Network)

			if !quiet {
				w := tabwriter.NewWriter(os.Stdout, 0, 4, 2, ' ', 0)
				defer func() {
					// Ignore flushing errors - there's nothing we can do.
					_ = w.Flush()
				}()
				common.PrintHeader(w, "ID", "Name", "Driver")
				output = func(n *api.Network) {
					spec := n.Spec
					fmt.Fprintf(w, "%s\t%s\t%s\n",
						n.ID,
						spec.Annotations.Name,
						n.DriverState.Name,
					)
				}

			} else {
				output = func(n *api.Network) { fmt.Println(n.ID) }
			}

			for _, j := range r.Networks {
				output(j)
			}
			return nil
		},
	}
)

func init() {
	listCmd.Flags().BoolP("quiet", "q", false, "Only display IDs")
}
