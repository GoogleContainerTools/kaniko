package node

import (
	"bytes"
	"crypto/x509"
	"encoding/pem"
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/cloudflare/cfssl/helpers"
	"github.com/docker/swarmkit/agent"
	agentutils "github.com/docker/swarmkit/agent/testutils"
	"github.com/docker/swarmkit/api"
	"github.com/docker/swarmkit/ca"
	"github.com/docker/swarmkit/ca/keyutils"
	cautils "github.com/docker/swarmkit/ca/testutils"
	"github.com/docker/swarmkit/identity"
	"github.com/docker/swarmkit/log"
	"github.com/docker/swarmkit/manager/state/store"
	"github.com/docker/swarmkit/testutils"
	"github.com/pkg/errors"
	"github.com/stretchr/testify/require"
	"golang.org/x/net/context"
)

func getLoggingContext(t *testing.T) context.Context {
	return log.WithLogger(context.Background(), log.L.WithField("test", t.Name()))
}

// If there is nothing on disk and no join addr, we create a new CA and a new set of TLS certs.
// If AutoLockManagers is enabled, the TLS key is encrypted with a randomly generated lock key.
func TestLoadSecurityConfigNewNode(t *testing.T) {
	for _, autoLockManagers := range []bool{true, false} {
		tempdir, err := ioutil.TempDir("", "test-new-node")
		require.NoError(t, err)
		defer os.RemoveAll(tempdir)

		paths := ca.NewConfigPaths(filepath.Join(tempdir, "certificates"))

		node, err := New(&Config{
			StateDir:         tempdir,
			AutoLockManagers: autoLockManagers,
		})
		require.NoError(t, err)
		securityConfig, cancel, err := node.loadSecurityConfig(context.Background(), paths)
		require.NoError(t, err)
		defer cancel()
		require.NotNil(t, securityConfig)

		unencryptedReader := ca.NewKeyReadWriter(paths.Node, nil, nil)
		_, _, err = unencryptedReader.Read()
		if !autoLockManagers {
			require.NoError(t, err)
		} else {
			require.IsType(t, ca.ErrInvalidKEK{}, err)
		}
	}
}

// If there's only a root CA on disk (no TLS certs), and no join addr, we create a new CA
// and a new set of TLS certs.  Similarly if there's only a TLS cert and key, and no CA.
func TestLoadSecurityConfigPartialCertsOnDisk(t *testing.T) {
	tempdir, err := ioutil.TempDir("", "test-new-node")
	require.NoError(t, err)
	defer os.RemoveAll(tempdir)

	paths := ca.NewConfigPaths(filepath.Join(tempdir, "certificates"))
	rootCA, err := ca.CreateRootCA(ca.DefaultRootCN)
	require.NoError(t, err)
	require.NoError(t, ca.SaveRootCA(rootCA, paths.RootCA))

	node, err := New(&Config{
		StateDir: tempdir,
	})
	require.NoError(t, err)
	securityConfig, cancel, err := node.loadSecurityConfig(context.Background(), paths)
	require.NoError(t, err)
	defer cancel()
	require.NotNil(t, securityConfig)

	cert, key, err := securityConfig.KeyReader().Read()
	require.NoError(t, err)

	// a new CA was generated because no existing TLS certs were present
	require.NotEqual(t, rootCA.Certs, securityConfig.RootCA().Certs)

	// if the TLS key and cert are on disk, but there's no CA, a new CA and TLS
	// key+cert are generated
	require.NoError(t, os.RemoveAll(paths.RootCA.Cert))

	node, err = New(&Config{
		StateDir: tempdir,
	})
	require.NoError(t, err)
	securityConfig, cancel, err = node.loadSecurityConfig(context.Background(), paths)
	require.NoError(t, err)
	defer cancel()
	require.NotNil(t, securityConfig)

	newCert, newKey, err := securityConfig.KeyReader().Read()
	require.NoError(t, err)
	require.NotEqual(t, cert, newCert)
	require.NotEqual(t, key, newKey)
}

// If there are CAs and TLS certs on disk, it tries to load and fails if there
// are any errors, even if a join token is provided.
func TestLoadSecurityConfigLoadFromDisk(t *testing.T) {
	tempdir, err := ioutil.TempDir("", "test-load-node-tls")
	require.NoError(t, err)
	defer os.RemoveAll(tempdir)

	paths := ca.NewConfigPaths(filepath.Join(tempdir, "certificates"))

	tc := cautils.NewTestCA(t)
	defer tc.Stop()
	peer, err := tc.ConnBroker.Remotes().Select()
	require.NoError(t, err)

	// Load successfully with valid passphrase
	rootCA, err := ca.CreateRootCA(ca.DefaultRootCN)
	require.NoError(t, err)
	require.NoError(t, ca.SaveRootCA(rootCA, paths.RootCA))

	krw := ca.NewKeyReadWriter(paths.Node, []byte("passphrase"), nil)
	require.NoError(t, err)
	_, _, err = rootCA.IssueAndSaveNewCertificates(krw, identity.NewID(), ca.WorkerRole, identity.NewID())
	require.NoError(t, err)

	node, err := New(&Config{
		StateDir:  tempdir,
		JoinAddr:  peer.Addr,
		JoinToken: tc.ManagerToken,
		UnlockKey: []byte("passphrase"),
	})
	require.NoError(t, err)
	securityConfig, cancel, err := node.loadSecurityConfig(context.Background(), paths)
	require.NoError(t, err)
	defer cancel()
	require.NotNil(t, securityConfig)

	// Invalid passphrase
	node, err = New(&Config{
		StateDir:  tempdir,
		JoinAddr:  peer.Addr,
		JoinToken: tc.ManagerToken,
	})
	require.NoError(t, err)
	_, _, err = node.loadSecurityConfig(context.Background(), paths)
	require.Equal(t, ErrInvalidUnlockKey, err)

	// Invalid CA
	otherRootCA, err := ca.CreateRootCA(ca.DefaultRootCN)
	require.NoError(t, err)
	require.NoError(t, ca.SaveRootCA(otherRootCA, paths.RootCA))
	node, err = New(&Config{
		StateDir:  tempdir,
		JoinAddr:  peer.Addr,
		JoinToken: tc.ManagerToken,
		UnlockKey: []byte("passphrase"),
	})
	require.NoError(t, err)
	_, _, err = node.loadSecurityConfig(context.Background(), paths)
	require.IsType(t, x509.UnknownAuthorityError{}, errors.Cause(err))

	// Convert to PKCS1 and require FIPS
	require.NoError(t, krw.DowngradeKey())
	// go back to the previous root CA
	require.NoError(t, ca.SaveRootCA(rootCA, paths.RootCA))
	node, err = New(&Config{
		StateDir:  tempdir,
		JoinAddr:  peer.Addr,
		JoinToken: tc.ManagerToken,
		UnlockKey: []byte("passphrase"),
		FIPS:      true,
	})
	require.NoError(t, err)
	_, _, err = node.loadSecurityConfig(context.Background(), paths)
	require.Equal(t, keyutils.ErrFIPSUnsupportedKeyFormat, errors.Cause(err))
}

// If there is no CA, and a join addr is provided, one is downloaded from the
// join server. If there is a CA, it is just loaded from disk.  The TLS key and
// cert are also downloaded.
func TestLoadSecurityConfigDownloadAllCerts(t *testing.T) {
	tempdir, err := ioutil.TempDir("", "test-join-node")
	require.NoError(t, err)
	defer os.RemoveAll(tempdir)

	paths := ca.NewConfigPaths(filepath.Join(tempdir, "certificates"))

	// join addr is invalid
	node, err := New(&Config{
		StateDir: tempdir,
		JoinAddr: "127.0.0.1:12",
	})
	require.NoError(t, err)
	_, _, err = node.loadSecurityConfig(context.Background(), paths)
	require.Error(t, err)

	tc := cautils.NewTestCA(t)
	defer tc.Stop()

	peer, err := tc.ConnBroker.Remotes().Select()
	require.NoError(t, err)

	node, err = New(&Config{
		StateDir:  tempdir,
		JoinAddr:  peer.Addr,
		JoinToken: tc.ManagerToken,
	})
	require.NoError(t, err)
	_, cancel, err := node.loadSecurityConfig(context.Background(), paths)
	require.NoError(t, err)
	cancel()

	// the TLS key and cert were written to disk unencrypted
	_, _, err = ca.NewKeyReadWriter(paths.Node, nil, nil).Read()
	require.NoError(t, err)

	// remove the TLS cert and key, and mark the root CA cert so that we will
	// know if it gets replaced
	require.NoError(t, os.Remove(paths.Node.Cert))
	require.NoError(t, os.Remove(paths.Node.Key))
	certBytes, err := ioutil.ReadFile(paths.RootCA.Cert)
	require.NoError(t, err)
	pemBlock, _ := pem.Decode(certBytes)
	require.NotNil(t, pemBlock)
	pemBlock.Headers["marked"] = "true"
	certBytes = pem.EncodeToMemory(pemBlock)
	require.NoError(t, ioutil.WriteFile(paths.RootCA.Cert, certBytes, 0644))

	// also make sure the new set gets downloaded and written to disk with a passphrase
	// by updating the memory store with manager autolock on and an unlock key
	require.NoError(t, tc.MemoryStore.Update(func(tx store.Tx) error {
		clusters, err := store.FindClusters(tx, store.All)
		require.NoError(t, err)
		require.Len(t, clusters, 1)

		newCluster := clusters[0].Copy()
		newCluster.Spec.EncryptionConfig.AutoLockManagers = true
		newCluster.UnlockKeys = []*api.EncryptionKey{{
			Subsystem: ca.ManagerRole,
			Key:       []byte("passphrase"),
		}}
		return store.UpdateCluster(tx, newCluster)
	}))

	// Join with without any passphrase - this should be fine, because the TLS
	// key is downloaded and then loaded just fine.  However, it *is* written
	// to disk encrypted.
	node, err = New(&Config{
		StateDir:  tempdir,
		JoinAddr:  peer.Addr,
		JoinToken: tc.ManagerToken,
	})
	require.NoError(t, err)
	_, cancel, err = node.loadSecurityConfig(context.Background(), paths)
	require.NoError(t, err)
	cancel()

	// make sure the CA cert has not been replaced
	readCertBytes, err := ioutil.ReadFile(paths.RootCA.Cert)
	require.NoError(t, err)
	require.Equal(t, certBytes, readCertBytes)

	// the TLS node cert and key were saved to disk encrypted, though
	_, _, err = ca.NewKeyReadWriter(paths.Node, nil, nil).Read()
	require.Error(t, err)
	_, _, err = ca.NewKeyReadWriter(paths.Node, []byte("passphrase"), nil).Read()
	require.NoError(t, err)
}

// If there is nothing on disk and no join addr, and FIPS is enabled, we create a cluster whose
// ID starts with 'FIPS.'
func TestLoadSecurityConfigNodeFIPSCreateCluster(t *testing.T) {
	tempdir, err := ioutil.TempDir("", "test-security-config-fips-new-cluster")
	require.NoError(t, err)
	defer os.RemoveAll(tempdir)

	paths := ca.NewConfigPaths(filepath.Join(tempdir, "certificates"))

	tc := cautils.NewTestCA(t)
	defer tc.Stop()

	config := &Config{
		StateDir: tempdir,
		FIPS:     true,
	}

	node, err := New(config)
	require.NoError(t, err)
	securityConfig, cancel, err := node.loadSecurityConfig(tc.Context, paths)
	require.NoError(t, err)
	defer cancel()
	require.NotNil(t, securityConfig)
	require.True(t, strings.HasPrefix(securityConfig.ClientTLSCreds.Organization(), "FIPS."))
}

// If FIPS is enabled and there is a join address, the cluster ID is whatever the CA set
// the cluster ID to.
func TestLoadSecurityConfigNodeFIPSJoinCluster(t *testing.T) {
	tempdir, err := ioutil.TempDir("", "test-security-config-fips-join-cluster")
	require.NoError(t, err)
	defer os.RemoveAll(tempdir)

	certDir := filepath.Join(tempdir, "certificates")
	paths := ca.NewConfigPaths(certDir)

	for _, fips := range []bool{true, false} {
		require.NoError(t, os.RemoveAll(certDir))

		var tc *cautils.TestCA
		if fips {
			tc = cautils.NewFIPSTestCA(t)
		} else {
			tc = cautils.NewTestCA(t)
		}
		defer tc.Stop()

		peer, err := tc.ConnBroker.Remotes().Select()
		require.NoError(t, err)

		node, err := New(&Config{
			StateDir:  tempdir,
			JoinAddr:  peer.Addr,
			JoinToken: tc.ManagerToken,
			FIPS:      true,
		})
		require.NoError(t, err)
		securityConfig, cancel, err := node.loadSecurityConfig(tc.Context, paths)
		require.NoError(t, err)
		defer cancel()
		require.NotNil(t, securityConfig)
		require.Equal(t, fips, strings.HasPrefix(securityConfig.ClientTLSCreds.Organization(), "FIPS."))
	}
}

// If the certificate specifies that the cluster requires FIPS mode, loading the security
// config will fail if the node is not FIPS enabled.
func TestLoadSecurityConfigRespectsFIPSCert(t *testing.T) {
	tempdir, err := ioutil.TempDir("", "test-security-config-fips-cert-on-disk")
	require.NoError(t, err)
	defer os.RemoveAll(tempdir)

	tc := cautils.NewFIPSTestCA(t)
	defer tc.Stop()

	certDir := filepath.Join(tempdir, "certificates")
	require.NoError(t, os.Mkdir(certDir, 0700))
	paths := ca.NewConfigPaths(certDir)

	// copy certs and keys from the test CA using a hard link
	_, err = tc.WriteNewNodeConfig(ca.ManagerRole)
	require.NoError(t, err)
	require.NoError(t, os.Link(tc.Paths.Node.Cert, paths.Node.Cert))
	require.NoError(t, os.Link(tc.Paths.Node.Key, paths.Node.Key))
	require.NoError(t, os.Link(tc.Paths.RootCA.Cert, paths.RootCA.Cert))

	node, err := New(&Config{StateDir: tempdir})
	require.NoError(t, err)
	_, _, err = node.loadSecurityConfig(tc.Context, paths)
	require.Equal(t, ErrMandatoryFIPS, err)

	node, err = New(&Config{
		StateDir: tempdir,
		FIPS:     true,
	})
	require.NoError(t, err)
	securityConfig, cancel, err := node.loadSecurityConfig(tc.Context, paths)
	require.NoError(t, err)
	defer cancel()
	require.NotNil(t, securityConfig)
	require.True(t, strings.HasPrefix(securityConfig.ClientTLSCreds.Organization(), "FIPS."))
}

// If FIPS is disabled and there is a join address and token, and the join token indicates
// the cluster requires fips, then loading the security config will fail.  However, if
// there are already certs on disk, it will load them and ignore the join token.
func TestLoadSecurityConfigNonFIPSNodeJoinCluster(t *testing.T) {
	tempdir, err := ioutil.TempDir("", "test-security-config-nonfips-join-cluster")
	require.NoError(t, err)
	defer os.RemoveAll(tempdir)

	certDir := filepath.Join(tempdir, "certificates")
	require.NoError(t, os.Mkdir(certDir, 0700))
	paths := ca.NewConfigPaths(certDir)

	tc := cautils.NewTestCA(t)
	defer tc.Stop()
	// copy certs and keys from the test CA using a hard link
	_, err = tc.WriteNewNodeConfig(ca.ManagerRole)
	require.NoError(t, err)
	require.NoError(t, os.Link(tc.Paths.Node.Cert, paths.Node.Cert))
	require.NoError(t, os.Link(tc.Paths.Node.Key, paths.Node.Key))
	require.NoError(t, os.Link(tc.Paths.RootCA.Cert, paths.RootCA.Cert))

	tcFIPS := cautils.NewFIPSTestCA(t)
	defer tcFIPS.Stop()

	peer, err := tcFIPS.ConnBroker.Remotes().Select()
	require.NoError(t, err)

	node, err := New(&Config{
		StateDir:  tempdir,
		JoinAddr:  peer.Addr,
		JoinToken: tcFIPS.ManagerToken,
	})
	require.NoError(t, err)
	securityConfig, cancel, err := node.loadSecurityConfig(tcFIPS.Context, paths)
	require.NoError(t, err)
	defer cancel()
	require.NotNil(t, securityConfig)
	require.False(t, strings.HasPrefix(securityConfig.ClientTLSCreds.Organization(), "FIPS."))

	// remove the node cert only - now that the node has to download the certs, it will check the
	// join address and fail
	require.NoError(t, os.Remove(paths.Node.Cert))

	_, _, err = node.loadSecurityConfig(tcFIPS.Context, paths)
	require.Equal(t, ErrMandatoryFIPS, err)

	// remove all the certs (CA and node) - the node will also check the join address and fail
	require.NoError(t, os.RemoveAll(certDir))

	_, _, err = node.loadSecurityConfig(tcFIPS.Context, paths)
	require.Equal(t, ErrMandatoryFIPS, err)
}

func TestManagerRespectsDispatcherRootCAUpdate(t *testing.T) {
	tmpDir, err := ioutil.TempDir("", "manager-root-ca-update")
	require.NoError(t, err)
	defer os.RemoveAll(tmpDir)

	// don't bother with a listening socket
	cAddr := filepath.Join(tmpDir, "control.sock")
	cfg := &Config{
		ListenControlAPI: cAddr,
		StateDir:         tmpDir,
		Executor:         &agentutils.TestExecutor{},
	}

	node, err := New(cfg)
	require.NoError(t, err)

	require.NoError(t, node.Start(context.Background()))

	select {
	case <-node.Ready():
	case <-time.After(5 * time.Second):
		require.FailNow(t, "node did not ready in time")
	}

	// ensure that we have a second dispatcher that we can connect to when we shut down ours
	paths := ca.NewConfigPaths(filepath.Join(tmpDir, certDirectory))
	rootCA, err := ca.GetLocalRootCA(paths.RootCA)
	require.NoError(t, err)
	managerSecConfig, cancel, err := ca.LoadSecurityConfig(context.Background(), rootCA, ca.NewKeyReadWriter(paths.Node, nil, nil), false)
	require.NoError(t, err)
	defer cancel()

	mockDispatcher, cleanup := agentutils.NewMockDispatcher(t, managerSecConfig, false)
	defer cleanup()
	node.remotes.Observe(api.Peer{Addr: mockDispatcher.Addr}, 1)

	currentCACerts := rootCA.Certs

	// shut down our current manager so that when the root CA changes, the manager doesn't "fix" it.
	node.manager.Stop(context.Background(), false)

	// fake an update from a remote dispatcher
	node.notifyNodeChange <- &agent.NodeChanges{
		RootCert: append(currentCACerts, cautils.ECDSA256SHA256Cert...),
	}

	// the node root CA certificates have changed now
	time.Sleep(250 * time.Millisecond)
	certPath := filepath.Join(tmpDir, certDirectory, "swarm-root-ca.crt")
	caCerts, err := ioutil.ReadFile(certPath)
	require.NoError(t, err)
	require.NotEqual(t, currentCACerts, caCerts)

	require.NoError(t, node.Stop(context.Background()))
}

func TestAgentRespectsDispatcherRootCAUpdate(t *testing.T) {
	tmpDir, err := ioutil.TempDir("", "manager-root-ca-update")
	require.NoError(t, err)
	defer os.RemoveAll(tmpDir)

	// bootstrap worker TLS certificates
	paths := ca.NewConfigPaths(filepath.Join(tmpDir, certDirectory))
	rootCA, err := ca.CreateRootCA("rootCN")
	require.NoError(t, err)
	require.NoError(t, ca.SaveRootCA(rootCA, paths.RootCA))
	managerSecConfig, cancel, err := rootCA.CreateSecurityConfig(context.Background(),
		ca.NewKeyReadWriter(paths.Node, nil, nil), ca.CertificateRequestConfig{})
	require.NoError(t, err)
	defer cancel()

	_, _, err = rootCA.IssueAndSaveNewCertificates(ca.NewKeyReadWriter(paths.Node, nil, nil), "workerNode",
		ca.WorkerRole, managerSecConfig.ServerTLSCreds.Organization())
	require.NoError(t, err)

	mockDispatcher, cleanup := agentutils.NewMockDispatcher(t, managerSecConfig, false)
	defer cleanup()

	cfg := &Config{
		StateDir: tmpDir,
		Executor: &agentutils.TestExecutor{},
		JoinAddr: mockDispatcher.Addr,
	}
	node, err := New(cfg)
	require.NoError(t, err)

	require.NoError(t, node.Start(context.Background()))

	select {
	case <-node.Ready():
	case <-time.After(5 * time.Second):
		require.FailNow(t, "node did not ready in time")
	}

	currentCACerts, err := ioutil.ReadFile(paths.RootCA.Cert)
	require.NoError(t, err)
	parsedCerts, err := helpers.ParseCertificatesPEM(currentCACerts)
	require.NoError(t, err)
	require.Len(t, parsedCerts, 1)

	// fake an update from the dispatcher
	node.notifyNodeChange <- &agent.NodeChanges{
		RootCert: append(currentCACerts, cautils.ECDSA256SHA256Cert...),
	}

	require.NoError(t, testutils.PollFuncWithTimeout(nil, func() error {
		caCerts, err := ioutil.ReadFile(paths.RootCA.Cert)
		require.NoError(t, err)
		if bytes.Equal(currentCACerts, caCerts) {
			return errors.New("new certificates have not been replaced yet")
		}
		parsedCerts, err := helpers.ParseCertificatesPEM(caCerts)
		if err != nil {
			return err
		}
		if len(parsedCerts) != 2 {
			return fmt.Errorf("expecting 2 new certificates, got %d", len(parsedCerts))
		}
		return nil
	}, time.Second))

	require.NoError(t, node.Stop(context.Background()))
}

func TestCertRenewals(t *testing.T) {
	tmpDir, err := ioutil.TempDir("", "no-top-level-role")
	require.NoError(t, err)
	defer os.RemoveAll(tmpDir)

	paths := ca.NewConfigPaths(filepath.Join(tmpDir, "certificates"))

	// don't bother with a listening socket
	cAddr := filepath.Join(tmpDir, "control.sock")
	cfg := &Config{
		ListenControlAPI: cAddr,
		StateDir:         tmpDir,
		Executor:         &agentutils.TestExecutor{},
	}
	node, err := New(cfg)
	require.NoError(t, err)

	require.NoError(t, node.Start(context.Background()))

	select {
	case <-node.Ready():
	case <-time.After(5 * time.Second):
		require.FailNow(t, "node did not ready in time")
	}

	currentNodeCert, err := ioutil.ReadFile(paths.Node.Cert)
	require.NoError(t, err)

	// Fake an update from the dispatcher. Make sure the Role field is
	// ignored when DesiredRole has not changed.
	node.notifyNodeChange <- &agent.NodeChanges{
		Node: &api.Node{
			Spec: api.NodeSpec{
				DesiredRole: api.NodeRoleManager,
			},
			Role: api.NodeRoleWorker,
		},
	}

	time.Sleep(500 * time.Millisecond)

	nodeCert, err := ioutil.ReadFile(paths.Node.Cert)
	require.NoError(t, err)
	if !bytes.Equal(currentNodeCert, nodeCert) {
		t.Fatal("Certificate should not have been renewed")
	}

	// Fake an update from the dispatcher. When DesiredRole doesn't match
	// the current role, a cert renewal should be triggered.
	node.notifyNodeChange <- &agent.NodeChanges{
		Node: &api.Node{
			Spec: api.NodeSpec{
				DesiredRole: api.NodeRoleWorker,
			},
			Role: api.NodeRoleWorker,
		},
	}

	require.NoError(t, testutils.PollFuncWithTimeout(nil, func() error {
		nodeCert, err := ioutil.ReadFile(paths.Node.Cert)
		require.NoError(t, err)
		if bytes.Equal(currentNodeCert, nodeCert) {
			return errors.New("certificate has not been replaced yet")
		}
		currentNodeCert = nodeCert
		return nil
	}, 5*time.Second))

	require.NoError(t, node.Stop(context.Background()))
}

func TestManagerFailedStartup(t *testing.T) {
	tmpDir, err := ioutil.TempDir("", "manager-root-ca-update")
	require.NoError(t, err)
	defer os.RemoveAll(tmpDir)

	paths := ca.NewConfigPaths(filepath.Join(tmpDir, certDirectory))

	rootCA, err := ca.CreateRootCA(ca.DefaultRootCN)
	require.NoError(t, err)
	require.NoError(t, ca.SaveRootCA(rootCA, paths.RootCA))

	krw := ca.NewKeyReadWriter(paths.Node, nil, nil)
	require.NoError(t, err)
	_, _, err = rootCA.IssueAndSaveNewCertificates(krw, identity.NewID(), ca.ManagerRole, identity.NewID())
	require.NoError(t, err)

	// don't bother with a listening socket
	cAddr := filepath.Join(tmpDir, "control.sock")
	cfg := &Config{
		ListenControlAPI: cAddr,
		StateDir:         tmpDir,
		Executor:         &agentutils.TestExecutor{},
		JoinAddr:         "127.0.0.1",
	}

	node, err := New(cfg)
	require.NoError(t, err)

	require.NoError(t, node.Start(context.Background()))

	select {
	case <-node.Ready():
		require.FailNow(t, "node should not become ready")
	case <-time.After(5 * time.Second):
		require.FailNow(t, "node neither became ready nor encountered an error")
	case <-node.closed:
		require.EqualError(t, node.err, "manager stopped: can't initialize raft node: attempted to join raft cluster without knowing own address")
	}
}

// TestFIPSConfiguration ensures that new keys will be stored in PKCS8 format.
func TestFIPSConfiguration(t *testing.T) {
	ctx := getLoggingContext(t)
	tmpDir, err := ioutil.TempDir("", "fips")
	require.NoError(t, err)
	defer os.RemoveAll(tmpDir)

	paths := ca.NewConfigPaths(filepath.Join(tmpDir, "certificates"))

	// don't bother with a listening socket
	cAddr := filepath.Join(tmpDir, "control.sock")
	cfg := &Config{
		ListenControlAPI: cAddr,
		StateDir:         tmpDir,
		Executor:         &agentutils.TestExecutor{},
		FIPS:             true,
	}
	node, err := New(cfg)
	require.NoError(t, err)
	require.NoError(t, node.Start(ctx))
	defer func() {
		require.NoError(t, node.Stop(ctx))
	}()

	select {
	case <-node.Ready():
	case <-time.After(5 * time.Second):
		require.FailNow(t, "node did not ready in time")
	}

	nodeKey, err := ioutil.ReadFile(paths.Node.Key)
	require.NoError(t, err)
	pemBlock, _ := pem.Decode(nodeKey)
	require.NotNil(t, pemBlock)
	require.True(t, keyutils.IsPKCS8(pemBlock.Bytes))
}
