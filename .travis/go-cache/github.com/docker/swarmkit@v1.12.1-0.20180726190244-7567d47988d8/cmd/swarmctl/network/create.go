package network

import (
	"errors"
	"fmt"
	"net"
	"strings"

	"github.com/docker/swarmkit/api"
	"github.com/docker/swarmkit/cmd/swarmctl/common"
	"github.com/spf13/cobra"
)

var (
	createCmd = &cobra.Command{
		Use:   "create",
		Short: "Create a network",
		RunE: func(cmd *cobra.Command, args []string) error {
			if len(args) != 0 {
				return errors.New("create command takes no arguments")
			}

			flags := cmd.Flags()
			if !flags.Changed("name") {
				return errors.New("--name is required")
			}

			name, err := flags.GetString("name")
			if err != nil {
				return err
			}

			// Process driver configurations
			var driver *api.Driver
			if flags.Changed("driver") {
				driver = new(api.Driver)

				driverName, err := flags.GetString("driver")
				if err != nil {
					return err
				}

				driver.Name = driverName

				opts, err := cmd.Flags().GetStringSlice("opts")
				if err != nil {
					return err
				}

				driver.Options = map[string]string{}
				for _, opt := range opts {
					optPair := strings.Split(opt, "=")
					if len(optPair) != 2 {
						return fmt.Errorf("Malformed opts: %s", opt)
					}
					driver.Options[optPair[0]] = optPair[1]
				}
			}

			ipamOpts, err := processIPAMOptions(cmd)
			if err != nil {
				return err
			}

			spec := &api.NetworkSpec{
				Annotations: api.Annotations{
					Name: name,
				},
				DriverConfig: driver,
				IPAM:         ipamOpts,
			}

			c, err := common.Dial(cmd)
			if err != nil {
				return err
			}
			r, err := c.CreateNetwork(common.Context(cmd), &api.CreateNetworkRequest{Spec: spec})
			if err != nil {
				return err
			}
			fmt.Println(r.Network.ID)
			return nil
		},
	}
)

func processIPAMOptions(cmd *cobra.Command) (*api.IPAMOptions, error) {
	flags := cmd.Flags()

	var ipamOpts *api.IPAMOptions
	if flags.Changed("ipam-driver") {
		driver, err := cmd.Flags().GetString("ipam-driver")
		if err != nil {
			return nil, err
		}

		ipamOpts = &api.IPAMOptions{
			Driver: &api.Driver{
				Name: driver,
			},
		}
	}

	if !flags.Changed("subnet") {
		return ipamOpts, nil
	}

	subnets, err := cmd.Flags().GetStringSlice("subnet")
	if err != nil {
		return nil, err
	}

	gateways, err := cmd.Flags().GetStringSlice("gateway")
	if err != nil {
		return nil, err
	}

	ranges, err := cmd.Flags().GetStringSlice("ip-range")
	if err != nil {
		return nil, err
	}

	ipamConfigs := make([]*api.IPAMConfig, 0, len(subnets))
	for _, s := range subnets {
		_, ipNet, err := net.ParseCIDR(s)
		if err != nil {
			return nil, err
		}

		family := api.IPAMConfig_IPV6
		if ipNet.IP.To4() != nil {
			family = api.IPAMConfig_IPV4
		}

		var gateway string
		for i, g := range gateways {
			if ipNet.Contains(net.ParseIP(g)) {
				gateways = append(gateways[:i], gateways[i+1:]...)
				gateway = g
				break
			}
		}

		var iprange string
		for i, r := range ranges {
			_, rangeNet, err := net.ParseCIDR(r)
			if err != nil {
				return nil, err
			}

			if ipNet.Contains(rangeNet.IP) {
				ranges = append(ranges[:i], ranges[i+1:]...)
				iprange = r
				break
			}
		}

		ipamConfigs = append(ipamConfigs, &api.IPAMConfig{
			Family:  family,
			Subnet:  s,
			Gateway: gateway,
			Range:   iprange,
		})
	}

	if ipamOpts == nil {
		ipamOpts = &api.IPAMOptions{}
	}

	ipamOpts.Configs = ipamConfigs
	return ipamOpts, nil
}

func init() {
	createCmd.Flags().String("name", "", "Network name")
	createCmd.Flags().String("driver", "", "Network driver")
	createCmd.Flags().String("ipam-driver", "", "IPAM driver")
	createCmd.Flags().StringSlice("subnet", []string{}, "Subnets in CIDR format that represents a network segments")
	createCmd.Flags().StringSlice("gateway", []string{}, "Gateway IP addresses for network segments")
	createCmd.Flags().StringSlice("ip-range", []string{}, "IP ranges to allocate from within the subnets")
	createCmd.Flags().StringSlice("opts", []string{}, "Network driver options")
}
