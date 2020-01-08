package main

import (
	"os"

	"github.com/docker/swarmkit/cmd/swarmctl/cluster"
	"github.com/docker/swarmkit/cmd/swarmctl/config"
	"github.com/docker/swarmkit/cmd/swarmctl/network"
	"github.com/docker/swarmkit/cmd/swarmctl/node"
	"github.com/docker/swarmkit/cmd/swarmctl/secret"
	"github.com/docker/swarmkit/cmd/swarmctl/service"
	"github.com/docker/swarmkit/cmd/swarmctl/task"
	"github.com/docker/swarmkit/cmd/swarmd/defaults"
	"github.com/docker/swarmkit/version"
	"github.com/spf13/cobra"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
)

func main() {
	if c, err := mainCmd.ExecuteC(); err != nil {
		c.Println("Error:", grpc.ErrorDesc(err))
		// if it's not a grpc, we assume it's a user error and we display the usage.
		if grpc.Code(err) == codes.Unknown {
			c.Println(c.UsageString())
		}

		os.Exit(-1)
	}
}

var (
	mainCmd = &cobra.Command{
		Use:           os.Args[0],
		Short:         "Control a swarm cluster",
		SilenceUsage:  true,
		SilenceErrors: true,
	}
)

func defaultSocket() string {
	swarmSocket := os.Getenv("SWARM_SOCKET")
	if swarmSocket != "" {
		return swarmSocket
	}
	return defaults.ControlAPISocket
}

func init() {
	mainCmd.PersistentFlags().StringP("socket", "s", defaultSocket(), "Socket to connect to the Swarm manager")
	mainCmd.PersistentFlags().BoolP("no-resolve", "n", false, "Do not try to map IDs to Names when displaying them")

	mainCmd.AddCommand(
		node.Cmd,
		service.Cmd,
		task.Cmd,
		version.Cmd,
		network.Cmd,
		cluster.Cmd,
		secret.Cmd,
		config.Cmd,
	)
}
