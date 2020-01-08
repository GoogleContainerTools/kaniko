package service

import (
	"fmt"

	"golang.org/x/net/context"

	"github.com/docker/swarmkit/api"
)

func getService(ctx context.Context, c api.ControlClient, input string) (*api.Service, error) {
	// GetService to match via full ID.
	rg, err := c.GetService(ctx, &api.GetServiceRequest{ServiceID: input})
	if err != nil {
		// If any error (including NotFound), ListServices to match via full name.
		rl, err := c.ListServices(ctx,
			&api.ListServicesRequest{
				Filters: &api.ListServicesRequest_Filters{
					Names: []string{input},
				},
			},
		)
		if err != nil {
			return nil, err
		}

		if len(rl.Services) == 0 {
			return nil, fmt.Errorf("service %s not found", input)
		}

		if l := len(rl.Services); l > 1 {
			return nil, fmt.Errorf("service %s is ambiguous (%d matches found)", input, l)
		}

		return rl.Services[0], nil
	}
	return rg.Service, nil
}

func getServiceReplicasTxt(s *api.Service, running int) string {
	switch t := s.Spec.GetMode().(type) {
	case *api.ServiceSpec_Global:
		return "global"
	case *api.ServiceSpec_Replicated:
		return fmt.Sprintf("%d/%d", running, t.Replicated.Replicas)
	}
	return ""
}
