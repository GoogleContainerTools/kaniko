package secret

import (
	"errors"
	"fmt"

	"github.com/docker/swarmkit/api"
	"github.com/docker/swarmkit/cmd/swarmctl/common"
	"github.com/spf13/cobra"
)

var removeCmd = &cobra.Command{
	Use:     "remove <secret ID or name>",
	Short:   "Remove a secret",
	Aliases: []string{"rm"},
	RunE: func(cmd *cobra.Command, args []string) error {
		if len(args) == 0 {
			return errors.New("remove command takes a single secret ID or name")
		}

		client, err := common.Dial(cmd)
		if err != nil {
			return err
		}

		secret, err := getSecret(common.Context(cmd), client, args[0])
		if err != nil {
			return err
		}

		_, err = client.RemoveSecret(common.Context(cmd), &api.RemoveSecretRequest{SecretID: secret.ID})
		if err != nil {
			return err
		}
		fmt.Println(secret.ID)
		return nil
	},
}
