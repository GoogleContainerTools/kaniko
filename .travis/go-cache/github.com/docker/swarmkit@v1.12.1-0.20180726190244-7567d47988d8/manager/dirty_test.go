package manager

import (
	"io/ioutil"
	"os"
	"testing"

	"golang.org/x/net/context"

	"github.com/docker/swarmkit/api"
	"github.com/docker/swarmkit/ca"
	"github.com/docker/swarmkit/ca/testutils"
	"github.com/docker/swarmkit/manager/state"
	"github.com/docker/swarmkit/manager/state/store"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestIsStateDirty(t *testing.T) {
	ctx := context.Background()

	temp, err := ioutil.TempFile("", "test-socket")
	assert.NoError(t, err)
	assert.NoError(t, temp.Close())
	assert.NoError(t, os.Remove(temp.Name()))

	defer os.RemoveAll(temp.Name())

	stateDir, err := ioutil.TempDir("", "test-raft")
	assert.NoError(t, err)
	defer os.RemoveAll(stateDir)

	tc := testutils.NewTestCA(t, func(p ca.CertPaths) *ca.KeyReadWriter {
		return ca.NewKeyReadWriter(p, []byte("kek"), nil)
	})
	defer tc.Stop()

	managerSecurityConfig, err := tc.NewNodeConfig(ca.ManagerRole)
	assert.NoError(t, err)

	m, err := New(&Config{
		RemoteAPI:        &RemoteAddrs{ListenAddr: "127.0.0.1:0"},
		ControlAPI:       temp.Name(),
		StateDir:         stateDir,
		SecurityConfig:   managerSecurityConfig,
		AutoLockManagers: true,
		UnlockKey:        []byte("kek"),
		RootCAPaths:      tc.Paths.RootCA,
	})
	assert.NoError(t, err)
	assert.NotNil(t, m)

	go m.Run(ctx)
	defer m.Stop(ctx, false)

	// State should never be dirty just after creating the manager
	isDirty, err := m.IsStateDirty()
	assert.NoError(t, err)
	assert.False(t, isDirty)

	// Wait for cluster and node to be created.
	watch, cancel := state.Watch(m.raftNode.MemoryStore().WatchQueue())
	defer cancel()
	<-watch
	<-watch

	// Updating the node should not cause the state to become dirty
	assert.NoError(t, m.raftNode.MemoryStore().Update(func(tx store.Tx) error {
		node := store.GetNode(tx, m.config.SecurityConfig.ClientTLSCreds.NodeID())
		require.NotNil(t, node)
		node.Spec.Availability = api.NodeAvailabilityPause
		return store.UpdateNode(tx, node)
	}))
	isDirty, err = m.IsStateDirty()
	assert.NoError(t, err)
	assert.False(t, isDirty)

	// Adding a service should cause the state to become dirty
	assert.NoError(t, m.raftNode.MemoryStore().Update(func(tx store.Tx) error {
		return store.CreateService(tx, &api.Service{ID: "foo"})
	}))
	isDirty, err = m.IsStateDirty()
	assert.NoError(t, err)
	assert.True(t, isDirty)
}
