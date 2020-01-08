package network

import (
	"fmt"

	"github.com/docker/swarmkit/api"
	"golang.org/x/net/context"
)

// GetNetwork tries to query for a network as an ID and if it can't be
// found tries to query as a name. If the name query returns exactly
// one entry then it is returned to the caller. Otherwise an error is
// returned.
func GetNetwork(ctx context.Context, c api.ControlClient, input string) (*api.Network, error) {
	// GetService to match via full ID.
	rg, err := c.GetNetwork(ctx, &api.GetNetworkRequest{NetworkID: input})
	if err != nil {
		// If any error (including NotFound), ListServices to match via full name.
		rl, err := c.ListNetworks(ctx,
			&api.ListNetworksRequest{
				Filters: &api.ListNetworksRequest_Filters{
					Names: []string{input},
				},
			},
		)
		if err != nil {
			return nil, err
		}

		if len(rl.Networks) == 0 {
			return nil, fmt.Errorf("network %s not found", input)
		}

		if l := len(rl.Networks); l > 1 {
			return nil, fmt.Errorf("network %s is ambiguous (%d matches found)", input, l)
		}

		return rl.Networks[0], nil
	}

	return rg.Network, nil
}

// ResolveServiceNetworks takes a service spec and resolves network names to network IDs.
func ResolveServiceNetworks(ctx context.Context, c api.ControlClient, spec *api.ServiceSpec) error {
	if len(spec.Task.Networks) == 0 {
		return nil
	}
	networks := make([]*api.NetworkAttachmentConfig, 0, len(spec.Task.Networks))
	for _, na := range spec.Task.Networks {
		n, err := GetNetwork(ctx, c, na.Target)
		if err != nil {
			return err
		}

		networks = append(networks, &api.NetworkAttachmentConfig{
			Target: n.ID,
		})
	}

	spec.Task.Networks = networks
	return nil
}
