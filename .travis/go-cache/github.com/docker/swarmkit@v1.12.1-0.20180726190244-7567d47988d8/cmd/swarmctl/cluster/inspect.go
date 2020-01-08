package cluster

import (
	"errors"
	"fmt"
	"os"
	"sort"
	"text/tabwriter"

	"github.com/docker/swarmkit/api"
	"github.com/docker/swarmkit/cmd/swarmctl/common"
	gogotypes "github.com/gogo/protobuf/types"
	"github.com/spf13/cobra"
)

func printClusterSummary(cluster *api.Cluster) {
	w := tabwriter.NewWriter(os.Stdout, 8, 8, 8, ' ', 0)
	defer w.Flush()

	common.FprintfIfNotEmpty(w, "ID\t: %s\n", cluster.ID)
	common.FprintfIfNotEmpty(w, "Name\t: %s\n", cluster.Spec.Annotations.Name)
	fmt.Fprintln(w, "Orchestration settings:")
	fmt.Fprintf(w, "  Task history entries: %d\n", cluster.Spec.Orchestration.TaskHistoryRetentionLimit)

	heartbeatPeriod, err := gogotypes.DurationFromProto(cluster.Spec.Dispatcher.HeartbeatPeriod)
	if err == nil {
		fmt.Fprintln(w, "Dispatcher settings:")
		fmt.Fprintf(w, "  Dispatcher heartbeat period: %s\n", heartbeatPeriod.String())
	}

	fmt.Fprintln(w, "Certificate Authority settings:")
	if cluster.Spec.CAConfig.NodeCertExpiry != nil {
		clusterDuration, err := gogotypes.DurationFromProto(cluster.Spec.CAConfig.NodeCertExpiry)
		if err != nil {
			fmt.Fprintln(w, "  Certificate Validity Duration: [ERROR PARSING DURATION]")
		} else {
			fmt.Fprintf(w, "  Certificate Validity Duration: %s\n", clusterDuration.String())
		}
	}
	if len(cluster.Spec.CAConfig.ExternalCAs) > 0 {
		fmt.Fprintln(w, "  External CAs:")
		for _, ca := range cluster.Spec.CAConfig.ExternalCAs {
			fmt.Fprintf(w, "    %s: %s\n", ca.Protocol, ca.URL)
		}
	}

	fmt.Fprintln(w, "  Join Tokens:")
	fmt.Fprintln(w, "    Worker:", cluster.RootCA.JoinTokens.Worker)
	fmt.Fprintln(w, "    Manager:", cluster.RootCA.JoinTokens.Manager)

	if cluster.Spec.TaskDefaults.LogDriver != nil {
		fmt.Fprintf(w, "Default Log Driver\t: %s\n", cluster.Spec.TaskDefaults.LogDriver.Name)
		var keys []string

		if len(cluster.Spec.TaskDefaults.LogDriver.Options) != 0 {
			for k := range cluster.Spec.TaskDefaults.LogDriver.Options {
				keys = append(keys, k)
			}
			sort.Strings(keys)

			for _, k := range keys {
				v := cluster.Spec.TaskDefaults.LogDriver.Options[k]
				if v != "" {
					fmt.Fprintf(w, "  %s\t: %s\n", k, v)
				} else {
					fmt.Fprintf(w, "  %s\t\n", k)
				}
			}
		}
	}
}

var (
	inspectCmd = &cobra.Command{
		Use:   "inspect <cluster name>",
		Short: "Inspect a cluster",
		RunE: func(cmd *cobra.Command, args []string) error {
			if len(args) == 0 {
				return errors.New("cluster name missing")
			}

			if len(args) > 1 {
				return errors.New("inspect command takes exactly 1 argument")
			}

			c, err := common.Dial(cmd)
			if err != nil {
				return err
			}

			cluster, err := getCluster(common.Context(cmd), c, args[0])
			if err != nil {
				return err
			}

			printClusterSummary(cluster)

			return nil
		},
	}
)
