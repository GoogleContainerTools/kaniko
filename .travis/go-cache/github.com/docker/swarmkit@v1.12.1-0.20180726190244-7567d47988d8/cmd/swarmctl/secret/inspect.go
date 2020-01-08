package secret

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

func printSecretSummary(secret *api.Secret) {
	w := tabwriter.NewWriter(os.Stdout, 8, 8, 8, ' ', 0)
	defer w.Flush()

	common.FprintfIfNotEmpty(w, "ID\t: %s\n", secret.ID)
	common.FprintfIfNotEmpty(w, "Name\t: %s\n", secret.Spec.Annotations.Name)
	if len(secret.Spec.Annotations.Labels) > 0 {
		fmt.Fprintln(w, "Labels\t")
		for k, v := range secret.Spec.Annotations.Labels {
			fmt.Fprintf(w, "  %s\t: %s\n", k, v)
		}
	}

	common.FprintfIfNotEmpty(w, "Created\t: %s\n", gogotypes.TimestampString(secret.Meta.CreatedAt))
}

var (
	inspectCmd = &cobra.Command{
		Use:   "inspect <secret ID or name>",
		Short: "Inspect a secret",
		RunE: func(cmd *cobra.Command, args []string) error {
			if len(args) != 1 {
				return errors.New("inspect command takes a single secret ID or name")
			}

			client, err := common.Dial(cmd)
			if err != nil {
				return err
			}

			secret, err := getSecret(common.Context(cmd), client, args[0])
			if err != nil {
				return err
			}

			printSecretSummary(secret)
			return nil
		},
	}
)
