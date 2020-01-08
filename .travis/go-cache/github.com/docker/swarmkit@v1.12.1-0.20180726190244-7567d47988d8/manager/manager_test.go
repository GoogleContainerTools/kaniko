package manager

import (
	"bytes"
	"crypto/tls"
	"encoding/pem"
	"errors"
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"testing"
	"time"

	"golang.org/x/net/context"

	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"

	"github.com/docker/swarmkit/api"
	"github.com/docker/swarmkit/ca"
	"github.com/docker/swarmkit/ca/keyutils"
	cautils "github.com/docker/swarmkit/ca/testutils"
	"github.com/docker/swarmkit/manager/dispatcher"
	"github.com/docker/swarmkit/manager/encryption"
	"github.com/docker/swarmkit/manager/state/raft/storage"
	"github.com/docker/swarmkit/manager/state/store"
	"github.com/docker/swarmkit/testutils"
	"github.com/stretchr/testify/require"
)

func TestManager(t *testing.T) {
	temp, err := ioutil.TempFile("", "test-socket")
	require.NoError(t, err)
	require.NoError(t, temp.Close())
	require.NoError(t, os.Remove(temp.Name()))

	defer os.RemoveAll(temp.Name())

	stateDir, err := ioutil.TempDir("", "test-raft")
	require.NoError(t, err)
	defer os.RemoveAll(stateDir)

	tc := cautils.NewTestCA(t, func(p ca.CertPaths) *ca.KeyReadWriter {
		return ca.NewKeyReadWriter(p, []byte("kek"), nil)
	})
	defer tc.Stop()

	agentSecurityConfig, err := tc.NewNodeConfig(ca.WorkerRole)
	require.NoError(t, err)
	agentDiffOrgSecurityConfig, err := tc.NewNodeConfigOrg(ca.WorkerRole, "another-org")
	require.NoError(t, err)
	managerSecurityConfig, err := tc.NewNodeConfig(ca.ManagerRole)
	require.NoError(t, err)

	m, err := New(&Config{
		RemoteAPI:        &RemoteAddrs{ListenAddr: "127.0.0.1:0"},
		ControlAPI:       temp.Name(),
		StateDir:         stateDir,
		SecurityConfig:   managerSecurityConfig,
		AutoLockManagers: true,
		UnlockKey:        []byte("kek"),
		RootCAPaths:      tc.Paths.RootCA,
	})
	require.NoError(t, err)
	require.NotNil(t, m)

	tcpAddr := m.Addr()

	done := make(chan error)
	defer close(done)
	go func() {
		done <- m.Run(tc.Context)
	}()

	opts := []grpc.DialOption{
		grpc.WithTimeout(10 * time.Second),
		grpc.WithTransportCredentials(agentSecurityConfig.ClientTLSCreds),
	}

	conn, err := grpc.Dial(tcpAddr, opts...)
	require.NoError(t, err)
	defer func() {
		require.NoError(t, conn.Close())
	}()

	// We have to send a dummy request to verify if the connection is actually up.
	client := api.NewDispatcherClient(conn)
	require.NoError(t, testutils.PollFuncWithTimeout(nil, func() error {
		_, err = client.Heartbeat(tc.Context, &api.HeartbeatRequest{})
		if dispatcher.ErrNodeNotRegistered.Error() != grpc.ErrorDesc(err) {
			return err
		}
		_, err = client.Session(tc.Context, &api.SessionRequest{})
		return err
	}, 1*time.Second))

	// Try to have a client in a different org access this manager
	opts = []grpc.DialOption{
		grpc.WithTimeout(10 * time.Second),
		grpc.WithTransportCredentials(agentDiffOrgSecurityConfig.ClientTLSCreds),
	}

	conn2, err := grpc.Dial(tcpAddr, opts...)
	require.NoError(t, err)
	defer func() {
		require.NoError(t, conn2.Close())
	}()

	client = api.NewDispatcherClient(conn2)
	_, err = client.Heartbeat(context.Background(), &api.HeartbeatRequest{})
	require.Contains(t, grpc.ErrorDesc(err), "Permission denied: unauthorized peer role: rpc error: code = PermissionDenied desc = Permission denied: remote certificate not part of organization")

	// Verify that requests to the various GRPC services running on TCP
	// are rejected if they don't have certs.
	opts = []grpc.DialOption{
		grpc.WithTimeout(10 * time.Second),
		grpc.WithTransportCredentials(credentials.NewTLS(&tls.Config{InsecureSkipVerify: true})),
	}

	noCertConn, err := grpc.Dial(tcpAddr, opts...)
	require.NoError(t, err)
	defer func() {
		require.NoError(t, noCertConn.Close())
	}()

	client = api.NewDispatcherClient(noCertConn)
	_, err = client.Heartbeat(context.Background(), &api.HeartbeatRequest{})
	require.EqualError(t, err, "rpc error: code = PermissionDenied desc = Permission denied: unauthorized peer role: rpc error: code = PermissionDenied desc = no client certificates in request")

	controlClient := api.NewControlClient(noCertConn)
	_, err = controlClient.ListNodes(context.Background(), &api.ListNodesRequest{})
	require.EqualError(t, err, "rpc error: code = PermissionDenied desc = Permission denied: unauthorized peer role: rpc error: code = PermissionDenied desc = no client certificates in request")

	raftClient := api.NewRaftMembershipClient(noCertConn)
	_, err = raftClient.Join(context.Background(), &api.JoinRequest{})
	require.EqualError(t, err, "rpc error: code = PermissionDenied desc = Permission denied: unauthorized peer role: rpc error: code = PermissionDenied desc = no client certificates in request")

	opts = []grpc.DialOption{
		grpc.WithTimeout(10 * time.Second),
		grpc.WithTransportCredentials(managerSecurityConfig.ClientTLSCreds),
	}

	controlConn, err := grpc.Dial(tcpAddr, opts...)
	require.NoError(t, err)
	defer func() {
		require.NoError(t, controlConn.Close())
	}()

	// check that the kek is added to the config
	var cluster api.Cluster
	require.NoError(t, testutils.PollFunc(nil, func() error {
		var (
			err      error
			clusters []*api.Cluster
		)
		m.raftNode.MemoryStore().View(func(tx store.ReadTx) {
			clusters, err = store.FindClusters(tx, store.All)
		})
		if err != nil {
			return err
		}
		if len(clusters) != 1 {
			return errors.New("wrong number of clusters")
		}
		cluster = *clusters[0]
		return nil

	}))
	require.NotNil(t, cluster)
	require.Len(t, cluster.UnlockKeys, 1)
	require.Equal(t, &api.EncryptionKey{
		Subsystem: ca.ManagerRole,
		Key:       []byte("kek"),
	}, cluster.UnlockKeys[0])

	// Test removal of the agent node
	agentID := agentSecurityConfig.ClientTLSCreds.NodeID()
	require.NoError(t, m.raftNode.MemoryStore().Update(func(tx store.Tx) error {
		return store.CreateNode(tx,
			&api.Node{
				ID: agentID,
				Certificate: api.Certificate{
					Role: api.NodeRoleWorker,
					CN:   agentID,
				},
			},
		)
	}))
	controlClient = api.NewControlClient(controlConn)
	_, err = controlClient.CreateNetwork(context.Background(), &api.CreateNetworkRequest{
		Spec: &api.NetworkSpec{
			Annotations: api.Annotations{
				Name: "test-network-bad-driver",
			},
			DriverConfig: &api.Driver{
				Name: "invalid-must-never-exist",
			},
		},
	})
	require.Error(t, err)

	_, err = controlClient.RemoveNode(context.Background(),
		&api.RemoveNodeRequest{
			NodeID: agentID,
			Force:  true,
		},
	)
	require.NoError(t, err)

	client = api.NewDispatcherClient(conn)
	_, err = client.Heartbeat(context.Background(), &api.HeartbeatRequest{})
	require.Contains(t, grpc.ErrorDesc(err), "removed from swarm")

	m.Stop(tc.Context, false)

	// After stopping we should MAY receive an error from ListenAndServe if
	// all this happened before WaitForLeader completed, so don't check the
	// error.
	<-done
}

// Tests locking and unlocking the manager and key rotations
func TestManagerLockUnlock(t *testing.T) {
	temp, err := ioutil.TempFile("", "test-manager-lock")
	require.NoError(t, err)
	require.NoError(t, temp.Close())
	require.NoError(t, os.Remove(temp.Name()))

	defer os.RemoveAll(temp.Name())

	stateDir, err := ioutil.TempDir("", "test-raft")
	require.NoError(t, err)
	defer os.RemoveAll(stateDir)

	tc := cautils.NewTestCA(t)
	defer tc.Stop()

	managerSecurityConfig, err := tc.NewNodeConfig(ca.ManagerRole)
	require.NoError(t, err)

	_, _, err = managerSecurityConfig.KeyReader().Read()
	require.NoError(t, err)

	m, err := New(&Config{
		RemoteAPI:      &RemoteAddrs{ListenAddr: "127.0.0.1:0"},
		ControlAPI:     temp.Name(),
		StateDir:       stateDir,
		SecurityConfig: managerSecurityConfig,
		RootCAPaths:    tc.Paths.RootCA,
		// start off without any encryption
	})
	require.NoError(t, err)
	require.NotNil(t, m)

	done := make(chan error)
	defer close(done)
	go func() {
		done <- m.Run(tc.Context)
	}()

	opts := []grpc.DialOption{
		grpc.WithTimeout(10 * time.Second),
		grpc.WithTransportCredentials(managerSecurityConfig.ClientTLSCreds),
	}

	conn, err := grpc.Dial(m.Addr(), opts...)
	require.NoError(t, err)
	defer func() {
		require.NoError(t, conn.Close())
	}()

	// check that there is no kek currently - we are using the API because this
	// lets us wait until the manager is up and listening, as well
	var cluster *api.Cluster
	client := api.NewControlClient(conn)

	require.NoError(t, testutils.PollFuncWithTimeout(nil, func() error {
		resp, err := client.ListClusters(tc.Context, &api.ListClustersRequest{})
		if err != nil {
			return err
		}
		if len(resp.Clusters) == 0 {
			return fmt.Errorf("no clusters yet")
		}
		cluster = resp.Clusters[0]
		return nil
	}, 1*time.Second))

	require.Nil(t, cluster.UnlockKeys)

	// tls key is unencrypted, but there is a DEK
	unencryptedKey, err := ioutil.ReadFile(tc.Paths.Node.Key)
	require.NoError(t, err)
	keyBlock, _ := pem.Decode(unencryptedKey)
	require.NotNil(t, keyBlock)
	require.False(t, keyutils.IsEncryptedPEMBlock(keyBlock))
	require.Len(t, keyBlock.Headers, 2)
	currentDEK, err := decodePEMHeaderValue(keyBlock.Headers[pemHeaderRaftDEK], nil, false)
	require.NoError(t, err)
	require.NotEmpty(t, currentDEK)

	// update the lock key - this may fail due to update out of sequence errors, so try again
	for {
		getResp, err := client.GetCluster(tc.Context, &api.GetClusterRequest{ClusterID: cluster.ID})
		require.NoError(t, err)
		cluster = getResp.Cluster

		spec := cluster.Spec.Copy()
		spec.EncryptionConfig.AutoLockManagers = true
		updateResp, err := client.UpdateCluster(tc.Context, &api.UpdateClusterRequest{
			ClusterID:      cluster.ID,
			ClusterVersion: &cluster.Meta.Version,
			Spec:           spec,
		})
		if grpc.ErrorDesc(err) == "update out of sequence" {
			continue
		}
		// if there is any other type of error, this should fail
		if err == nil {
			cluster = updateResp.Cluster
		}
		break
	}
	require.NoError(t, err)

	caConn := api.NewCAClient(conn)
	unlockKeyResp, err := caConn.GetUnlockKey(tc.Context, &api.GetUnlockKeyRequest{})
	require.NoError(t, err)

	// this should update the TLS key, rotate the DEK, and finish snapshotting
	var encryptedKey []byte
	require.NoError(t, testutils.PollFuncWithTimeout(nil, func() error {
		encryptedKey, err = ioutil.ReadFile(tc.Paths.Node.Key)
		require.NoError(t, err) // this should never error due to atomic writes

		if bytes.Equal(unencryptedKey, encryptedKey) {
			return fmt.Errorf("TLS key should have been re-encrypted at least")
		}

		keyBlock, _ = pem.Decode(encryptedKey)
		require.NotNil(t, keyBlock) // this should never error due to atomic writes

		if !keyutils.IsEncryptedPEMBlock(keyBlock) {
			return fmt.Errorf("Key not encrypted")
		}

		// we don't check that the TLS key has been rotated, because that may take
		// a little bit, and is best effort only
		currentDEKString, ok := keyBlock.Headers[pemHeaderRaftDEK]
		require.True(t, ok) // there should never NOT be a current header
		nowCurrentDEK, err := decodePEMHeaderValue(currentDEKString, unlockKeyResp.UnlockKey, false)
		require.NoError(t, err) // it should always be encrypted
		if bytes.Equal(currentDEK, nowCurrentDEK) {
			return fmt.Errorf("snapshot has not been finished yet")
		}

		currentDEK = nowCurrentDEK
		return nil
	}, 1*time.Second))

	_, ok := keyBlock.Headers[pemHeaderRaftPendingDEK]
	require.False(t, ok) // once the snapshot is done, the pending DEK should have been deleted

	_, ok = keyBlock.Headers[pemHeaderRaftDEKNeedsRotation]
	require.False(t, ok)

	// verify that the snapshot is readable with the new DEK
	encrypter, decrypter := encryption.Defaults(currentDEK, false)
	// we can't use the raftLogger, because the WALs are still locked while the raft node is up.  And once we remove
	// the manager, they'll be deleted.
	snapshot, err := storage.NewSnapFactory(encrypter, decrypter).New(filepath.Join(stateDir, "raft", "snap-v3-encrypted")).Load()
	require.NoError(t, err)
	require.NotNil(t, snapshot)

	// update the lock key to nil
	for i := 0; i < 3; i++ {
		getResp, err := client.GetCluster(tc.Context, &api.GetClusterRequest{ClusterID: cluster.ID})
		require.NoError(t, err)
		cluster = getResp.Cluster

		spec := cluster.Spec.Copy()
		spec.EncryptionConfig.AutoLockManagers = false
		_, err = client.UpdateCluster(tc.Context, &api.UpdateClusterRequest{
			ClusterID:      cluster.ID,
			ClusterVersion: &cluster.Meta.Version,
			Spec:           spec,
		})
		if grpc.ErrorDesc(err) == "update out of sequence" {
			continue
		}
		require.NoError(t, err)
	}

	// this should update the TLS key
	var unlockedKey []byte
	require.NoError(t, testutils.PollFuncWithTimeout(nil, func() error {
		unlockedKey, err = ioutil.ReadFile(tc.Paths.Node.Key)
		if err != nil {
			return err
		}

		if bytes.Equal(unlockedKey, encryptedKey) {
			return fmt.Errorf("TLS key should have been rotated")
		}

		// Previously, we did not check that the TLS key got rotated after going from
		// unlocked -> locked, because it might take a while for the snapshot to be done,
		// and the rotation happens on a best effort basis.  However, that *could*
		// have happened, in which case the encrypted key may have changed, so we have
		// to poll to make sure that the key is eventually decrypted, rather than
		// just waiting for it to look different.

		// the new key should not be encrypted, and the DEK should also be unencrypted
		keyBlock, _ = pem.Decode(unlockedKey)
		if keyBlock == nil {
			return fmt.Errorf("keyblock is nil")
		}
		if keyutils.IsEncryptedPEMBlock(keyBlock) {
			return fmt.Errorf("key is still encrypted")
		}
		return nil
	}, 1*time.Second))

	// the new key should not be encrypted, and the DEK should also be unencrypted
	// but not rotated
	keyBlock, _ = pem.Decode(unlockedKey)
	require.NotNil(t, keyBlock)
	require.False(t, keyutils.IsEncryptedPEMBlock(keyBlock))

	unencryptedDEK, err := decodePEMHeaderValue(keyBlock.Headers[pemHeaderRaftDEK], nil, false)
	require.NoError(t, err)
	require.NotNil(t, unencryptedDEK)
	require.Equal(t, currentDEK, unencryptedDEK)

	m.Stop(tc.Context, false)

	// After stopping we should MAY receive an error from ListenAndServe if
	// all this happened before WaitForLeader completed, so don't check the
	// error.
	<-done
}
