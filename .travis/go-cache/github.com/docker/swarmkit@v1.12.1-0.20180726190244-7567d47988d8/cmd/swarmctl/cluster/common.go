package cluster

import (
	"fmt"

	"golang.org/x/net/context"

	"github.com/docker/swarmkit/api"
)

func getCluster(ctx context.Context, c api.ControlClient, input string) (*api.Cluster, error) {
	rg, err := c.GetCluster(ctx, &api.GetClusterRequest{ClusterID: input})
	if err == nil {
		return rg.Cluster, nil
	}
	rl, err := c.ListClusters(ctx,
		&api.ListClustersRequest{
			Filters: &api.ListClustersRequest_Filters{
				Names: []string{input},
			},
		},
	)
	if err != nil {
		return nil, err
	}

	if len(rl.Clusters) == 0 {
		return nil, fmt.Errorf("cluster %s not found", input)
	}

	if l := len(rl.Clusters); l > 1 {
		return nil, fmt.Errorf("cluster %s is ambiguous (%d matches found)", input, l)
	}

	return rl.Clusters[0], nil
}
