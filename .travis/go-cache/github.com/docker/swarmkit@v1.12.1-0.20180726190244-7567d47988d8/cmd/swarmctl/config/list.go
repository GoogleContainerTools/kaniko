package config

import (
	"errors"
	"fmt"
	"os"
	"sort"
	"text/tabwriter"

	"github.com/docker/swarmkit/api"
	"github.com/docker/swarmkit/cmd/swarmctl/common"
	"github.com/dustin/go-humanize"
	gogotypes "github.com/gogo/protobuf/types"
	"github.com/spf13/cobra"
)

type configSorter []*api.Config

func (k configSorter) Len() int      { return len(k) }
func (k configSorter) Swap(i, j int) { k[i], k[j] = k[j], k[i] }
func (k configSorter) Less(i, j int) bool {
	iTime, err := gogotypes.TimestampFromProto(k[i].Meta.CreatedAt)
	if err != nil {
		panic(err)
	}
	jTime, err := gogotypes.TimestampFromProto(k[j].Meta.CreatedAt)
	if err != nil {
		panic(err)
	}
	return jTime.Before(iTime)
}

var (
	listCmd = &cobra.Command{
		Use:   "ls",
		Short: "List configs",
		RunE: func(cmd *cobra.Command, args []string) error {
			if len(args) != 0 {
				return errors.New("ls command takes no arguments")
			}

			flags := cmd.Flags()
			quiet, err := flags.GetBool("quiet")
			if err != nil {
				return err
			}

			client, err := common.Dial(cmd)
			if err != nil {
				return err
			}

			resp, err := client.ListConfigs(common.Context(cmd), &api.ListConfigsRequest{})
			if err != nil {
				return err
			}

			var output func(*api.Config)

			if !quiet {
				w := tabwriter.NewWriter(os.Stdout, 0, 4, 4, ' ', 0)
				defer func() {
					// Ignore flushing errors - there's nothing we can do.
					_ = w.Flush()
				}()
				common.PrintHeader(w, "ID", "Name", "Created")
				output = func(s *api.Config) {
					created, err := gogotypes.TimestampFromProto(s.Meta.CreatedAt)
					if err != nil {
						panic(err)
					}
					fmt.Fprintf(w, "%s\t%s\t%s\n",
						s.ID,
						s.Spec.Annotations.Name,
						humanize.Time(created),
					)
				}
			} else {
				output = func(s *api.Config) { fmt.Println(s.ID) }
			}

			sorted := configSorter(resp.Configs)
			sort.Sort(sorted)
			for _, s := range sorted {
				output(s)
			}
			return nil
		},
	}
)

func init() {
	listCmd.Flags().BoolP("quiet", "q", false, "Only display config IDs")
}
