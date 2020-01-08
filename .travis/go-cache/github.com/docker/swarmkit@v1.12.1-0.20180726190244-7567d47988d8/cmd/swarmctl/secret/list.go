package secret

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

type secretSorter []*api.Secret

func (k secretSorter) Len() int      { return len(k) }
func (k secretSorter) Swap(i, j int) { k[i], k[j] = k[j], k[i] }
func (k secretSorter) Less(i, j int) bool {
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
		Short: "List secrets",
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

			resp, err := client.ListSecrets(common.Context(cmd), &api.ListSecretsRequest{})
			if err != nil {
				return err
			}

			var output func(*api.Secret)

			if !quiet {
				w := tabwriter.NewWriter(os.Stdout, 0, 4, 4, ' ', 0)
				defer func() {
					// Ignore flushing errors - there's nothing we can do.
					_ = w.Flush()
				}()
				common.PrintHeader(w, "ID", "Name", "Driver", "Created")
				output = func(s *api.Secret) {
					created, err := gogotypes.TimestampFromProto(s.Meta.CreatedAt)
					if err != nil {
						panic(err)
					}
					var driver string
					if s.Spec.Driver != nil {
						driver = s.Spec.Driver.Name
					}

					fmt.Fprintf(w, "%s\t%s\t%s\t%s\n",
						s.ID,
						s.Spec.Annotations.Name,
						driver,
						humanize.Time(created),
					)
				}

			} else {
				output = func(s *api.Secret) { fmt.Println(s.ID) }
			}

			sorted := secretSorter(resp.Secrets)
			sort.Sort(sorted)
			for _, s := range sorted {
				output(s)
			}
			return nil
		},
	}
)

func init() {
	listCmd.Flags().BoolP("quiet", "q", false, "Only display secret IDs")
}
