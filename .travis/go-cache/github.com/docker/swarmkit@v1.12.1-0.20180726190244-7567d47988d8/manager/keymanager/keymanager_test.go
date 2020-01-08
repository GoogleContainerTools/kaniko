package keymanager

import (
	"bytes"
	"testing"
	"time"

	"github.com/docker/swarmkit/api"
	"github.com/docker/swarmkit/manager/state/store"
	"github.com/stretchr/testify/assert"
	"golang.org/x/net/context"
)

func createClusterSpec(name string) *api.ClusterSpec {
	return &api.ClusterSpec{
		Annotations: api.Annotations{
			Name: name,
		},
	}
}

func createCluster(t *testing.T, s *store.MemoryStore, id, name string) *api.Cluster {
	spec := createClusterSpec(name)

	cluster := &api.Cluster{
		ID:   id,
		Spec: *spec,
	}
	assert.NoError(t, s.Update(func(tx store.Tx) error {
		return store.CreateCluster(tx, cluster)
	}))
	return cluster
}

// Verify the key generation and rotation for default subsystems
func TestKeyManagerDefaultSubsystem(t *testing.T) {
	st := store.NewMemoryStore(nil)
	defer st.Close()
	createCluster(t, st, "default", store.DefaultClusterName)

	k := New(st, DefaultConfig())

	ctx := context.Background()
	go k.Run(ctx)
	time.Sleep(250 * time.Millisecond)

	// verify the number of keys allocated matches the keyring size.
	var (
		clusters []*api.Cluster
		err      error
	)
	k.store.View(func(readTx store.ReadTx) {
		clusters, err = store.FindClusters(readTx, store.ByName(k.config.ClusterName))
	})

	assert.NoError(t, err)
	assert.Equal(t, len(clusters[0].NetworkBootstrapKeys), len(k.config.Subsystems)*keyringSize)

	key1 := clusters[0].NetworkBootstrapKeys[0].Key

	k.rotateKey(ctx)

	// verify that after a rotation oldest key has been removed from the keyring
	assert.Equal(t, len(k.keyRing.keys), len(k.config.Subsystems)*keyringSize)
	for _, key := range k.keyRing.keys {
		match := bytes.Equal(key.Key, key1)
		assert.False(t, match)
	}
}

// Verify the key generation and rotation for IPsec subsystem
func TestKeyManagerCustomSubsystem(t *testing.T) {
	st := store.NewMemoryStore(nil)
	defer st.Close()
	createCluster(t, st, "default", store.DefaultClusterName)

	config := &Config{
		ClusterName:      store.DefaultClusterName,
		Keylen:           DefaultKeyLen,
		RotationInterval: DefaultKeyRotationInterval,
		Subsystems:       []string{SubsystemIPSec},
	}
	k := New(st, config)

	ctx := context.Background()
	go k.Run(ctx)
	time.Sleep(250 * time.Millisecond)

	// verify the number of keys allocated matches the keyring size.
	var (
		clusters []*api.Cluster
		err      error
	)
	k.store.View(func(readTx store.ReadTx) {
		clusters, err = store.FindClusters(readTx, store.ByName(k.config.ClusterName))
	})

	assert.NoError(t, err)
	assert.Equal(t, len(clusters[0].NetworkBootstrapKeys), keyringSize)

	key1 := clusters[0].NetworkBootstrapKeys[0].Key

	k.rotateKey(ctx)

	// verify that after a rotation oldest key has been removed from the keyring
	// also verify that all keys are for the right subsystem
	assert.Equal(t, len(k.keyRing.keys), keyringSize)
	for _, key := range k.keyRing.keys {
		match := bytes.Equal(key.Key, key1)
		assert.False(t, match)
		match = key.Subsystem == SubsystemIPSec
		assert.True(t, match)
	}
}

// Verify that instantiating keymanager fails if an invalid subsystem is
// passed
func TestKeyManagerInvalidSubsystem(t *testing.T) {
	st := store.NewMemoryStore(nil)
	defer st.Close()
	createCluster(t, st, "default", store.DefaultClusterName)

	config := &Config{
		ClusterName:      store.DefaultClusterName,
		Keylen:           DefaultKeyLen,
		RotationInterval: DefaultKeyRotationInterval,
		Subsystems:       []string{"serf"},
	}
	k := New(st, config)

	assert.Nil(t, k)
}
