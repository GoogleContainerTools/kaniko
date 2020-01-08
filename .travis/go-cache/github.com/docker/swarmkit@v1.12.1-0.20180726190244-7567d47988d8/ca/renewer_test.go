package ca_test

import (
	"testing"
	"time"

	"github.com/docker/swarmkit/api"
	"github.com/docker/swarmkit/ca"
	"github.com/docker/swarmkit/ca/testutils"
	"github.com/docker/swarmkit/manager/state/store"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"golang.org/x/net/context"
)

func TestForceRenewTLSConfig(t *testing.T) {
	t.Parallel()

	tc := testutils.NewTestCA(t)
	defer tc.Stop()

	ctx, cancel := context.WithCancel(tc.Context)
	defer cancel()

	// Get a new managerConfig with a TLS cert that has 15 minutes to live
	nodeConfig, err := tc.WriteNewNodeConfig(ca.ManagerRole)
	assert.NoError(t, err)

	renewer := ca.NewTLSRenewer(nodeConfig, tc.ConnBroker, tc.Paths.RootCA)
	updates := renewer.Start(ctx)
	renewer.Renew()
	select {
	case <-time.After(10 * time.Second):
		assert.Fail(t, "TestForceRenewTLSConfig timed-out")
	case certUpdate := <-updates:
		assert.NoError(t, certUpdate.Err)
		assert.NotNil(t, certUpdate)
		assert.Equal(t, certUpdate.Role, ca.ManagerRole)
	}
}

func TestForceRenewExpectedRole(t *testing.T) {
	t.Parallel()

	tc := testutils.NewTestCA(t)
	defer tc.Stop()

	ctx, cancel := context.WithCancel(tc.Context)
	defer cancel()

	// Get a new managerConfig with a TLS cert that has 15 minutes to live
	nodeConfig, err := tc.WriteNewNodeConfig(ca.ManagerRole)
	assert.NoError(t, err)

	go func() {
		time.Sleep(750 * time.Millisecond)

		err := tc.MemoryStore.Update(func(tx store.Tx) error {
			node := store.GetNode(tx, nodeConfig.ClientTLSCreds.NodeID())
			require.NotNil(t, node)

			node.Spec.DesiredRole = api.NodeRoleWorker
			node.Role = api.NodeRoleWorker

			return store.UpdateNode(tx, node)
		})
		assert.NoError(t, err)
	}()

	renewer := ca.NewTLSRenewer(nodeConfig, tc.ConnBroker, tc.Paths.RootCA)
	updates := renewer.Start(ctx)
	renewer.SetExpectedRole(ca.WorkerRole)
	renewer.Renew()
	for {
		select {
		case <-time.After(10 * time.Second):
			t.Fatal("timed out")
		case certUpdate := <-updates:
			assert.NoError(t, certUpdate.Err)
			assert.NotNil(t, certUpdate)
			if certUpdate.Role == ca.WorkerRole {
				return
			}
		}
	}
}
