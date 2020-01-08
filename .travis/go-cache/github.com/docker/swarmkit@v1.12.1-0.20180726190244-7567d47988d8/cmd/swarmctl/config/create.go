package config

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
	Use:   "create <config name>",
	Short: "Create a config",
	RunE: func(cmd *cobra.Command, args []string) error {
		if len(args) != 1 {
			return errors.New(
				"create command takes a unique config name as an argument, and accepts config data via stdin or via a file")
		}

		flags := cmd.Flags()
		var (
			configData []byte
			err        error
		)

		if flags.Changed("file") {
			filename, err := flags.GetString("file")
			if err != nil {
				return err
			}
			configData, err = ioutil.ReadFile(filename)
			if err != nil {
				return fmt.Errorf("Error reading from file '%s': %s", filename, err.Error())
			}
		} else {
			configData, err = ioutil.ReadAll(os.Stdin)
			if err != nil {
				return fmt.Errorf("Error reading content from STDIN: %s", err.Error())
			}
		}

		client, err := common.Dial(cmd)
		if err != nil {
			return err
		}

		spec := &api.ConfigSpec{
			Annotations: api.Annotations{Name: args[0]},
			Data:        configData,
		}

		resp, err := client.CreateConfig(common.Context(cmd), &api.CreateConfigRequest{Spec: spec})
		if err != nil {
			return err
		}
		fmt.Println(resp.Config.ID)
		return nil
	},
}

func init() {
	createCmd.Flags().StringP("file", "f", "", "Rather than read the config from STDIN, read from the given file")
}
