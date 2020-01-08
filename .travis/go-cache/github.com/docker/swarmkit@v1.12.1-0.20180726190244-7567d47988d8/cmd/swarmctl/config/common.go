package config

import (
	"fmt"

	"github.com/docker/swarmkit/api"
	"golang.org/x/net/context"
)

func getConfig(ctx context.Context, c api.ControlClient, input string) (*api.Config, error) {
	// not sure what it is, match by name or id prefix
	resp, err := c.ListConfigs(ctx,
		&api.ListConfigsRequest{
			Filters: &api.ListConfigsRequest_Filters{
				Names:      []string{input},
				IDPrefixes: []string{input},
			},
		},
	)
	if err != nil {
		return nil, err
	}

	switch len(resp.Configs) {
	case 0:
		return nil, fmt.Errorf("config %s not found", input)
	case 1:
		return resp.Configs[0], nil
	default:
		// ok, multiple matches.  Prefer exact ID over exact name.  If no exact matches, return an error
		for _, s := range resp.Configs {
			if s.ID == input {
				return s, nil
			}
		}
		for _, s := range resp.Configs {
			if s.Spec.Annotations.Name == input {
				return s, nil
			}
		}
		return nil, fmt.Errorf("config %s is ambiguous (%d matches found)", input, len(resp.Configs))
	}
}
