package integration

import (
	"github.com/docker/swarmkit/api"
	"golang.org/x/net/context"
)

type dummyAPI struct {
	c *testCluster
}

func (a *dummyAPI) GetNode(ctx context.Context, r *api.GetNodeRequest) (*api.GetNodeResponse, error) {
	ctx, cancel := context.WithTimeout(ctx, opsTimeout)
	defer cancel()
	cli, err := a.c.RandomManager().ControlClient(ctx)
	if err != nil {
		return nil, err
	}
	return cli.GetNode(ctx, r)
}

func (a *dummyAPI) ListNodes(ctx context.Context, r *api.ListNodesRequest) (*api.ListNodesResponse, error) {
	ctx, cancel := context.WithTimeout(ctx, opsTimeout)
	defer cancel()
	m := a.c.RandomManager()
	cli, err := m.ControlClient(ctx)
	if err != nil {
		return nil, err
	}
	resp, err := cli.ListNodes(ctx, r)
	return resp, err
}

func (a *dummyAPI) UpdateNode(ctx context.Context, r *api.UpdateNodeRequest) (*api.UpdateNodeResponse, error) {
	ctx, cancel := context.WithTimeout(ctx, opsTimeout)
	defer cancel()
	cli, err := a.c.RandomManager().ControlClient(ctx)
	if err != nil {
		return nil, err
	}
	return cli.UpdateNode(ctx, r)
}

func (a *dummyAPI) RemoveNode(ctx context.Context, r *api.RemoveNodeRequest) (*api.RemoveNodeResponse, error) {
	ctx, cancel := context.WithTimeout(ctx, opsTimeout)
	defer cancel()
	cli, err := a.c.RandomManager().ControlClient(ctx)
	if err != nil {
		return nil, err
	}
	return cli.RemoveNode(ctx, r)
}

func (a *dummyAPI) GetTask(context.Context, *api.GetTaskRequest) (*api.GetTaskResponse, error) {
	panic("not implemented")
}

func (a *dummyAPI) ListTasks(ctx context.Context, r *api.ListTasksRequest) (*api.ListTasksResponse, error) {
	ctx, cancel := context.WithTimeout(ctx, opsTimeout)
	defer cancel()
	cli, err := a.c.RandomManager().ControlClient(ctx)
	if err != nil {
		return nil, err
	}
	return cli.ListTasks(ctx, r)
}

func (a *dummyAPI) RemoveTask(context.Context, *api.RemoveTaskRequest) (*api.RemoveTaskResponse, error) {
	panic("not implemented")
}

func (a *dummyAPI) GetService(context.Context, *api.GetServiceRequest) (*api.GetServiceResponse, error) {
	panic("not implemented")
}

func (a *dummyAPI) ListServices(context.Context, *api.ListServicesRequest) (*api.ListServicesResponse, error) {
	panic("not implemented")
}

func (a *dummyAPI) CreateService(ctx context.Context, r *api.CreateServiceRequest) (*api.CreateServiceResponse, error) {
	ctx, cancel := context.WithTimeout(ctx, opsTimeout)
	defer cancel()
	cli, err := a.c.RandomManager().ControlClient(ctx)
	if err != nil {
		return nil, err
	}
	return cli.CreateService(ctx, r)
}

func (a *dummyAPI) UpdateService(context.Context, *api.UpdateServiceRequest) (*api.UpdateServiceResponse, error) {
	panic("not implemented")
}

func (a *dummyAPI) RemoveService(context.Context, *api.RemoveServiceRequest) (*api.RemoveServiceResponse, error) {
	panic("not implemented")
}

func (a *dummyAPI) GetNetwork(context.Context, *api.GetNetworkRequest) (*api.GetNetworkResponse, error) {
	panic("not implemented")
}

func (a *dummyAPI) ListNetworks(context.Context, *api.ListNetworksRequest) (*api.ListNetworksResponse, error) {
	panic("not implemented")
}

func (a *dummyAPI) CreateNetwork(context.Context, *api.CreateNetworkRequest) (*api.CreateNetworkResponse, error) {
	panic("not implemented")
}

func (a *dummyAPI) RemoveNetwork(context.Context, *api.RemoveNetworkRequest) (*api.RemoveNetworkResponse, error) {
	panic("not implemented")
}

func (a *dummyAPI) GetCluster(ctx context.Context, r *api.GetClusterRequest) (*api.GetClusterResponse, error) {
	ctx, cancel := context.WithTimeout(ctx, opsTimeout)
	defer cancel()
	cli, err := a.c.RandomManager().ControlClient(ctx)
	if err != nil {
		return nil, err
	}
	return cli.GetCluster(ctx, r)
}

func (a *dummyAPI) ListClusters(ctx context.Context, r *api.ListClustersRequest) (*api.ListClustersResponse, error) {
	ctx, cancel := context.WithTimeout(ctx, opsTimeout)
	defer cancel()
	cli, err := a.c.RandomManager().ControlClient(ctx)
	if err != nil {
		return nil, err
	}
	return cli.ListClusters(ctx, r)
}

func (a *dummyAPI) UpdateCluster(ctx context.Context, r *api.UpdateClusterRequest) (*api.UpdateClusterResponse, error) {
	ctx, cancel := context.WithTimeout(ctx, opsTimeout)
	defer cancel()
	cli, err := a.c.RandomManager().ControlClient(ctx)
	if err != nil {
		return nil, err
	}
	return cli.UpdateCluster(ctx, r)
}
