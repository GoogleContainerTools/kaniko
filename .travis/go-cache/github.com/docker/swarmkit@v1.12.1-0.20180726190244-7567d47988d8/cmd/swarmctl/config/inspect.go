package config

import (
	"errors"
	"fmt"
	"os"
	"text/tabwriter"

	"github.com/docker/swarmkit/api"
	"github.com/docker/swarmkit/cmd/swarmctl/common"
	gogotypes "github.com/gogo/protobuf/types"
	"github.com/spf13/cobra"
)

func printConfigSummary(config *api.Config) {
	w := tabwriter.NewWriter(os.Stdout, 8, 8, 8, ' ', 0)
	defer w.Flush()

	common.FprintfIfNotEmpty(w, "ID\t: %s\n", config.ID)
	common.FprintfIfNotEmpty(w, "Name\t: %s\n", config.Spec.Annotations.Name)
	if len(config.Spec.Annotations.Labels) > 0 {
		fmt.Fprintln(w, "Labels\t")
		for k, v := range config.Spec.Annotations.Labels {
			fmt.Fprintf(w, "  %s\t: %s\n", k, v)
		}
	}

	common.FprintfIfNotEmpty(w, "Created\t: %s\n", gogotypes.TimestampString(config.Meta.CreatedAt))

	fmt.Print(w, "Payload:\n\n")
	fmt.Println(w, config.Spec.Data)
}

var (
	inspectCmd = &cobra.Command{
		Use:   "inspect <config ID or name>",
		Short: "Inspect a config",
		RunE: func(cmd *cobra.Command, args []string) error {
			if len(args) != 1 {
				return errors.New("inspect command takes a single config ID or name")
			}

			client, err := common.Dial(cmd)
			if err != nil {
				return err
			}

			config, err := getConfig(common.Context(cmd), client, args[0])
			if err != nil {
				return err
			}

			printConfigSummary(config)
			return nil
		},
	}
)
