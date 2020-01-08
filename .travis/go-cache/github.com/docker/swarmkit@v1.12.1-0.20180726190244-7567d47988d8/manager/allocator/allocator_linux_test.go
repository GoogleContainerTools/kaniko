package allocator

import (
	"testing"

	"golang.org/x/net/context"

	"github.com/docker/swarmkit/api"
	"github.com/docker/swarmkit/manager/state"
	"github.com/docker/swarmkit/manager/state/store"
	"github.com/stretchr/testify/assert"
)

func TestIPAMNotNil(t *testing.T) {
	s := store.NewMemoryStore(nil)
	assert.NotNil(t, s)
	defer s.Close()

	a, err := New(s, nil)
	assert.NoError(t, err)
	assert.NotNil(t, a)

	// Predefined node-local network
	p := &api.Network{
		ID: "one_unIque_id",
		Spec: api.NetworkSpec{
			Annotations: api.Annotations{
				Name: "pred_bridge_network",
				Labels: map[string]string{
					"com.docker.swarm.predefined": "true",
				},
			},
			DriverConfig: &api.Driver{Name: "bridge"},
		},
	}

	// Node-local swarm scope network
	nln := &api.Network{
		ID: "another_unIque_id",
		Spec: api.NetworkSpec{
			Annotations: api.Annotations{
				Name: "swarm-macvlan",
			},
			DriverConfig: &api.Driver{Name: "macvlan"},
		},
	}

	// Try adding some objects to store before allocator is started
	assert.NoError(t, s.Update(func(tx store.Tx) error {
		// populate ingress network
		in := &api.Network{
			ID: "ingress-nw-id",
			Spec: api.NetworkSpec{
				Annotations: api.Annotations{
					Name: "default-ingress",
				},
				Ingress: true,
			},
		}
		assert.NoError(t, store.CreateNetwork(tx, in))

		// Create the predefined node-local network with one service
		assert.NoError(t, store.CreateNetwork(tx, p))

		// Create the the swarm level node-local network with one service
		assert.NoError(t, store.CreateNetwork(tx, nln))

		return nil
	}))

	netWatch, cancel := state.Watch(s.WatchQueue(), api.EventUpdateNetwork{}, api.EventDeleteNetwork{})
	defer cancel()

	// Start allocator
	go func() {
		assert.NoError(t, a.Run(context.Background()))
	}()
	defer a.Stop()

	// Now verify if we get network and tasks updated properly
	watchNetwork(t, netWatch, false, func(t assert.TestingT, n *api.Network) bool { return true })
	watchNetwork(t, netWatch, false, func(t assert.TestingT, n *api.Network) bool { return true })
	watchNetwork(t, netWatch, false, func(t assert.TestingT, n *api.Network) bool { return true })

	// Verify no allocation was done for the node-local networks
	var (
		ps *api.Network
		sn *api.Network
	)
	s.View(func(tx store.ReadTx) {
		ps = store.GetNetwork(tx, p.ID)
		sn = store.GetNetwork(tx, nln.ID)

	})
	assert.NotNil(t, ps)
	assert.NotNil(t, sn)
	assert.NotNil(t, ps.IPAM)
	assert.NotNil(t, sn.IPAM)
}
