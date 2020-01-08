package network

import (
	"errors"
	"fmt"
	"os"
	"text/tabwriter"

	"github.com/docker/swarmkit/api"
	"github.com/docker/swarmkit/cmd/swarmctl/common"
	"github.com/spf13/cobra"
)

var (
	inspectCmd = &cobra.Command{
		Use:   "inspect <network ID>",
		Short: "Inspect a network",
		RunE: func(cmd *cobra.Command, args []string) error {
			if len(args) == 0 {
				return errors.New("network ID missing")
			}

			if len(args) > 1 {
				return errors.New("inspect command takes exactly 1 argument")
			}

			c, err := common.Dial(cmd)
			if err != nil {
				return err
			}
			network, err := GetNetwork(common.Context(cmd), c, args[0])
			if err != nil {
				return err
			}

			printNetworkSummary(network)

			return nil
		},
	}
)

func printNetworkSummary(network *api.Network) {
	w := tabwriter.NewWriter(os.Stdout, 8, 8, 8, ' ', 0)
	defer func() {
		// Ignore flushing errors - there's nothing we can do.
		_ = w.Flush()
	}()

	spec := &network.Spec
	common.FprintfIfNotEmpty(w, "ID\t: %s\n", network.ID)
	common.FprintfIfNotEmpty(w, "Name\t: %s\n", spec.Annotations.Name)

	fmt.Fprintln(w, "Spec:\t")
	if len(spec.Annotations.Labels) > 0 {
		fmt.Fprintln(w, "  Labels:\t")
		for k, v := range spec.Annotations.Labels {
			fmt.Fprintf(w, "    %s = %s\n", k, v)
		}
	}
	fmt.Fprintf(w, "  IPv6Enabled\t: %t\n", spec.Ipv6Enabled)
	fmt.Fprintf(w, "  Internal\t: %t\n", spec.Internal)

	driver := network.DriverState
	if driver != nil {
		fmt.Fprintln(w, "Driver:\t")
		common.FprintfIfNotEmpty(w, "  Name\t: %s\n", driver.Name)
		if len(driver.Options) > 0 {
			fmt.Fprintln(w, "  Options:\t")
			for k, v := range driver.Options {
				fmt.Fprintf(w, "    %s = %s\n", k, v)
			}
		}
	}

	ipam := network.IPAM
	if ipam != nil {
		fmt.Fprintln(w, "IPAM:\t")
		if ipam.Driver != nil {
			fmt.Fprintln(w, "  Driver:\t")
			common.FprintfIfNotEmpty(w, "    Name\t: %s\n", ipam.Driver.Name)
			if len(ipam.Driver.Options) > 0 {
				fmt.Fprintln(w, "    Options:\t")
				for k, v := range ipam.Driver.Options {
					fmt.Fprintf(w, "      %s = %s\n", k, v)
				}
			}
		}

		if len(ipam.Configs) > 0 {
			for _, config := range ipam.Configs {
				fmt.Fprintln(w, "  IPAM Config:\t")
				common.FprintfIfNotEmpty(w, "    Family\t: %s\n", config.Family.String())
				common.FprintfIfNotEmpty(w, "    Subnet\t: %s\n", config.Subnet)
				common.FprintfIfNotEmpty(w, "    Range\t: %s\n", config.Range)
				common.FprintfIfNotEmpty(w, "    Range\t: %s\n", config.Range)
				common.FprintfIfNotEmpty(w, "    Gateway\t: %s\n", config.Gateway)
				if len(config.Reserved) > 0 {
					fmt.Fprintln(w, "    Reserved:\t")
					for k, v := range config.Reserved {
						fmt.Fprintf(w, "      %s = %s\n", k, v)
					}
				}
			}
		}
	}
}
