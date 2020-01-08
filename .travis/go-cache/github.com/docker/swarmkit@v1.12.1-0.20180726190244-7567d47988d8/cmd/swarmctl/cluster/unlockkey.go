package cluster

import (
	"errors"
	"fmt"

	"github.com/docker/swarmkit/api"
	"github.com/docker/swarmkit/cmd/swarmctl/common"
	"github.com/docker/swarmkit/manager/encryption"
	"github.com/spf13/cobra"
)

// get the unlock key

func displayUnlockKey(cmd *cobra.Command) error {
	conn, err := common.DialConn(cmd)
	if err != nil {
		return err
	}
	defer conn.Close()

	resp, err := api.NewCAClient(conn).GetUnlockKey(common.Context(cmd), &api.GetUnlockKeyRequest{})
	if err != nil {
		return err
	}

	if len(resp.UnlockKey) == 0 {
		fmt.Printf("Managers not auto-locked")
	}
	fmt.Printf("Managers auto-locked.  Unlock key: %s\n", encryption.HumanReadableKey(resp.UnlockKey))
	return nil
}

var (
	unlockKeyCmd = &cobra.Command{
		Use:   "unlock-key <cluster name>",
		Short: "Get the unlock key for a cluster",
		RunE: func(cmd *cobra.Command, args []string) error {
			if len(args) == 0 {
				return errors.New("cluster name missing")
			}

			if len(args) > 1 {
				return errors.New("unlock-key command takes exactly 1 argument")
			}

			return displayUnlockKey(cmd)
		},
	}
)
