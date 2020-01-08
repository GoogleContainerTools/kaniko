package cluster

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
		Short: "List clusters",
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
			r, err := c.ListClusters(common.Context(cmd), &api.ListClustersRequest{})
			if err != nil {
				return err
			}

			var output func(j *api.Cluster)

			if !quiet {
				w := tabwriter.NewWriter(os.Stdout, 0, 4, 2, ' ', 0)
				defer func() {
					// Ignore flushing errors - there's nothing we can do.
					_ = w.Flush()
				}()
				common.PrintHeader(w, "ID", "Name")
				output = func(s *api.Cluster) {
					spec := s.Spec

					fmt.Fprintf(w, "%s\t%s\n",
						s.ID,
						spec.Annotations.Name,
					)
				}

			} else {
				output = func(j *api.Cluster) { fmt.Println(j.ID) }
			}

			for _, j := range r.Clusters {
				output(j)
			}
			return nil
		},
	}
)

func init() {
	listCmd.Flags().BoolP("quiet", "q", false, "Only display IDs")
}
