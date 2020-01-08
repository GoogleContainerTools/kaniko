package secret

import (
	"errors"
	"fmt"
	"io/ioutil"
	"os"

	"github.com/docker/swarmkit/api"
	"github.com/docker/swarmkit/cmd/swarmctl/common"
	"github.com/spf13/cobra"
)

var createCmd = &cobra.Command{
	Use:   "create <secret name>",
	Short: "Create a secret",
	RunE: func(cmd *cobra.Command, args []string) error {
		if len(args) != 1 {
			return errors.New(
				"create command takes a unique secret name as an argument, and accepts secret data via stdin or via a file")
		}

		flags := cmd.Flags()
		var (
			secretData []byte
			err        error
			driver     string
		)

		driver, err = flags.GetString("driver")
		if err != nil {
			return fmt.Errorf("Error reading secret driver %s", err.Error())
		}
		if flags.Changed("file") {
			filename, err := flags.GetString("file")
			if err != nil {
				return err
			}
			secretData, err = ioutil.ReadFile(filename)
			if err != nil {
				return fmt.Errorf("Error reading from file '%s': %s", filename, err.Error())
			}
		} else if driver == "" {
			secretData, err = ioutil.ReadAll(os.Stdin)
			if err != nil {
				return fmt.Errorf("Error reading content from STDIN: %s", err.Error())
			}
		}

		client, err := common.Dial(cmd)
		if err != nil {
			return err
		}

		spec := &api.SecretSpec{
			Annotations: api.Annotations{Name: args[0]},
			Data:        secretData,
		}
		if driver != "" {
			spec.Driver = &api.Driver{Name: driver}
		}

		resp, err := client.CreateSecret(common.Context(cmd), &api.CreateSecretRequest{Spec: spec})
		if err != nil {
			return err
		}
		fmt.Println(resp.Secret.ID)
		return nil
	},
}

func init() {
	createCmd.Flags().StringP("file", "f", "", "Rather than read the secret from STDIN, read from the given file")
	createCmd.Flags().StringP("driver", "d", "", "Rather than read the secret from STDIN, read the value from an external secret driver")
}
