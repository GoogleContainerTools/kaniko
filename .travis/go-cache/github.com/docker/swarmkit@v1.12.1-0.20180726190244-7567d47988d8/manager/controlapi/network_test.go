package controlapi

import (
	"testing"

	"golang.org/x/net/context"

	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"

	"github.com/docker/swarmkit/api"
	"github.com/docker/swarmkit/identity"
	"github.com/docker/swarmkit/manager/state/store"
	"github.com/stretchr/testify/assert"
)

func createNetworkSpec(name string) *api.NetworkSpec {
	return &api.NetworkSpec{
		Annotations: api.Annotations{
			Name: name,
		},
	}
}

// createInternalNetwork creates an internal network for testing. it is the same
// as Server.CreateNetwork except without the label check.
func (s *Server) createInternalNetwork(ctx context.Context, request *api.CreateNetworkRequest) (*api.CreateNetworkResponse, error) {
	if err := validateNetworkSpec(request.Spec, nil); err != nil {
		return nil, err
	}

	// TODO(mrjana): Consider using `Name` as a primary key to handle
	// duplicate creations. See #65
	n := &api.Network{
		ID:   identity.NewID(),
		Spec: *request.Spec,
	}

	err := s.store.Update(func(tx store.Tx) error {
		return store.CreateNetwork(tx, n)
	})
	if err != nil {
		return nil, err
	}

	return &api.CreateNetworkResponse{
		Network: n,
	}, nil
}

func createServiceInNetworkSpec(name, image string, nwid string, instances uint64) *api.ServiceSpec {
	return &api.ServiceSpec{
		Annotations: api.Annotations{
			Name: name,
			Labels: map[string]string{
				"common": "yes",
				"unique": name,
			},
		},
		Task: api.TaskSpec{
			Runtime: &api.TaskSpec_Container{
				Container: &api.ContainerSpec{
					Image: image,
				},
			},
		},
		Mode: &api.ServiceSpec_Replicated{
			Replicated: &api.ReplicatedService{
				Replicas: instances,
			},
		},
		Networks: []*api.NetworkAttachmentConfig{
			{
				Target: nwid,
			},
		},
	}
}

func createServiceInNetwork(t *testing.T, ts *testServer, name, image string, nwid string, instances uint64) *api.Service {
	spec := createServiceInNetworkSpec(name, image, nwid, instances)
	r, err := ts.Client.CreateService(context.Background(), &api.CreateServiceRequest{Spec: spec})
	assert.NoError(t, err)
	return r.Service
}

func TestValidateDriver(t *testing.T) {
	assert.NoError(t, validateDriver(nil, nil, ""))

	err := validateDriver(&api.Driver{Name: ""}, nil, "")
	assert.Error(t, err)
	assert.Equal(t, codes.InvalidArgument, grpc.Code(err))
}

func TestValidateIPAMConfiguration(t *testing.T) {
	err := validateIPAMConfiguration(nil)
	assert.Error(t, err)
	assert.Equal(t, codes.InvalidArgument, grpc.Code(err))

	IPAMConf := &api.IPAMConfig{
		Subnet: "",
	}

	err = validateIPAMConfiguration(IPAMConf)
	assert.Error(t, err)
	assert.Equal(t, codes.InvalidArgument, grpc.Code(err))

	IPAMConf.Subnet = "bad"
	err = validateIPAMConfiguration(IPAMConf)
	assert.Error(t, err)
	assert.Equal(t, codes.InvalidArgument, grpc.Code(err))

	IPAMConf.Subnet = "192.168.0.0/16"
	err = validateIPAMConfiguration(IPAMConf)
	assert.NoError(t, err)

	IPAMConf.Range = "bad"
	err = validateIPAMConfiguration(IPAMConf)
	assert.Error(t, err)
	assert.Equal(t, codes.InvalidArgument, grpc.Code(err))

	IPAMConf.Range = "192.169.1.0/24"
	err = validateIPAMConfiguration(IPAMConf)
	assert.Error(t, err)
	assert.Equal(t, codes.InvalidArgument, grpc.Code(err))

	IPAMConf.Range = "192.168.1.0/24"
	err = validateIPAMConfiguration(IPAMConf)
	assert.NoError(t, err)

	IPAMConf.Gateway = "bad"
	err = validateIPAMConfiguration(IPAMConf)
	assert.Error(t, err)
	assert.Equal(t, codes.InvalidArgument, grpc.Code(err))

	IPAMConf.Gateway = "192.169.1.1"
	err = validateIPAMConfiguration(IPAMConf)
	assert.Error(t, err)
	assert.Equal(t, codes.InvalidArgument, grpc.Code(err))

	IPAMConf.Gateway = "192.168.1.1"
	err = validateIPAMConfiguration(IPAMConf)
	assert.NoError(t, err)
}

func TestCreateNetwork(t *testing.T) {
	ts := newTestServer(t)
	defer ts.Stop()
	nr, err := ts.Client.CreateNetwork(context.Background(), &api.CreateNetworkRequest{
		Spec: createNetworkSpec("testnet1"),
	})
	assert.NoError(t, err)
	assert.NotEqual(t, nr.Network, nil)
	assert.NotEqual(t, nr.Network.ID, "")
}

func TestGetNetwork(t *testing.T) {
	ts := newTestServer(t)
	defer ts.Stop()
	nr, err := ts.Client.CreateNetwork(context.Background(), &api.CreateNetworkRequest{
		Spec: createNetworkSpec("testnet2"),
	})
	assert.NoError(t, err)
	assert.NotEqual(t, nr.Network, nil)
	assert.NotEqual(t, nr.Network.ID, "")

	_, err = ts.Client.GetNetwork(context.Background(), &api.GetNetworkRequest{NetworkID: nr.Network.ID})
	assert.NoError(t, err)
}

func TestRemoveNetwork(t *testing.T) {
	ts := newTestServer(t)
	defer ts.Stop()
	nr, err := ts.Client.CreateNetwork(context.Background(), &api.CreateNetworkRequest{
		Spec: createNetworkSpec("testnet3"),
	})
	assert.NoError(t, err)
	assert.NotEqual(t, nr.Network, nil)
	assert.NotEqual(t, nr.Network.ID, "")

	_, err = ts.Client.RemoveNetwork(context.Background(), &api.RemoveNetworkRequest{NetworkID: nr.Network.ID})
	assert.NoError(t, err)
}

func TestRemoveNetworkWithAttachedService(t *testing.T) {
	ts := newTestServer(t)
	defer ts.Stop()
	nr, err := ts.Client.CreateNetwork(context.Background(), &api.CreateNetworkRequest{
		Spec: createNetworkSpec("testnet4"),
	})
	assert.NoError(t, err)
	assert.NotEqual(t, nr.Network, nil)
	assert.NotEqual(t, nr.Network.ID, "")
	createServiceInNetwork(t, ts, "name", "image", nr.Network.ID, 1)
	_, err = ts.Client.RemoveNetwork(context.Background(), &api.RemoveNetworkRequest{NetworkID: nr.Network.ID})
	assert.Error(t, err)
}

func TestCreateNetworkInvalidDriver(t *testing.T) {
	ts := newTestServer(t)
	defer ts.Stop()
	spec := createNetworkSpec("baddrivernet")
	spec.DriverConfig = &api.Driver{
		Name: "invalid-must-never-exist",
	}
	_, err := ts.Client.CreateNetwork(context.Background(), &api.CreateNetworkRequest{
		Spec: spec,
	})
	assert.Error(t, err)
}

func TestListNetworks(t *testing.T) {
	ts := newTestServer(t)
	defer ts.Stop()

	nr1, err := ts.Client.CreateNetwork(context.Background(), &api.CreateNetworkRequest{
		Spec: createNetworkSpec("listtestnet1"),
	})
	assert.NoError(t, err)
	assert.NotEqual(t, nr1.Network, nil)
	assert.NotEqual(t, nr1.Network.ID, "")

	nr2, err := ts.Client.CreateNetwork(context.Background(), &api.CreateNetworkRequest{
		Spec: createNetworkSpec("listtestnet2"),
	})
	assert.NoError(t, err)
	assert.NotEqual(t, nr2.Network, nil)
	assert.NotEqual(t, nr2.Network.ID, "")

	r, err := ts.Client.ListNetworks(context.Background(), &api.ListNetworksRequest{})
	assert.NoError(t, err)
	assert.Equal(t, 3, len(r.Networks)) // Account ingress network
	for _, nw := range r.Networks {
		if nw.Spec.Ingress {
			continue
		}
		assert.True(t, nw.ID == nr1.Network.ID || nw.ID == nr2.Network.ID)
	}
}
