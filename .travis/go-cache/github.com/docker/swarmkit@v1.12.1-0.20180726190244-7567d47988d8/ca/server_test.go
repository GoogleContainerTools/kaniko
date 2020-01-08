package ca_test

import (
	"bytes"
	"crypto/tls"
	"crypto/x509"
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"reflect"
	"testing"
	"time"

	"github.com/cloudflare/cfssl/helpers"
	"github.com/docker/swarmkit/api"
	"github.com/docker/swarmkit/api/equality"
	"github.com/docker/swarmkit/ca"
	cautils "github.com/docker/swarmkit/ca/testutils"
	"github.com/docker/swarmkit/log"
	"github.com/docker/swarmkit/manager/state/store"
	"github.com/docker/swarmkit/testutils"
	"github.com/opencontainers/go-digest"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"golang.org/x/net/context"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
)

var _ api.CAServer = &ca.Server{}
var _ api.NodeCAServer = &ca.Server{}

func TestGetRootCACertificate(t *testing.T) {
	tc := cautils.NewTestCA(t)
	defer tc.Stop()

	resp, err := tc.CAClients[0].GetRootCACertificate(tc.Context, &api.GetRootCACertificateRequest{})
	assert.NoError(t, err)
	assert.NotEmpty(t, resp.Certificate)
}

func TestRestartRootCA(t *testing.T) {
	tc := cautils.NewTestCA(t)
	defer tc.Stop()

	_, err := tc.NodeCAClients[0].NodeCertificateStatus(tc.Context, &api.NodeCertificateStatusRequest{NodeID: "foo"})
	assert.Error(t, err)
	assert.Equal(t, codes.NotFound, grpc.Code(err))

	tc.CAServer.Stop()
	go tc.CAServer.Run(tc.Context)

	<-tc.CAServer.Ready()

	_, err = tc.NodeCAClients[0].NodeCertificateStatus(tc.Context, &api.NodeCertificateStatusRequest{NodeID: "foo"})
	assert.Error(t, err)
	assert.Equal(t, codes.NotFound, grpc.Code(err))
}

func TestIssueNodeCertificate(t *testing.T) {
	tc := cautils.NewTestCA(t)
	defer tc.Stop()

	csr, _, err := ca.GenerateNewCSR()
	assert.NoError(t, err)

	issueRequest := &api.IssueNodeCertificateRequest{CSR: csr, Token: tc.WorkerToken}
	issueResponse, err := tc.NodeCAClients[0].IssueNodeCertificate(tc.Context, issueRequest)
	assert.NoError(t, err)
	assert.NotNil(t, issueResponse.NodeID)
	assert.Equal(t, api.NodeMembershipAccepted, issueResponse.NodeMembership)

	statusRequest := &api.NodeCertificateStatusRequest{NodeID: issueResponse.NodeID}
	statusResponse, err := tc.NodeCAClients[0].NodeCertificateStatus(tc.Context, statusRequest)
	require.NoError(t, err)
	assert.Equal(t, api.IssuanceStateIssued, statusResponse.Status.State)
	assert.NotNil(t, statusResponse.Certificate.Certificate)
	assert.Equal(t, api.NodeRoleWorker, statusResponse.Certificate.Role)
}

func TestForceRotationIsNoop(t *testing.T) {
	tc := cautils.NewTestCA(t)
	defer tc.Stop()

	// Get a new Certificate issued
	csr, _, err := ca.GenerateNewCSR()
	assert.NoError(t, err)

	issueRequest := &api.IssueNodeCertificateRequest{CSR: csr, Token: tc.WorkerToken}
	issueResponse, err := tc.NodeCAClients[0].IssueNodeCertificate(tc.Context, issueRequest)
	assert.NoError(t, err)
	assert.NotNil(t, issueResponse.NodeID)
	assert.Equal(t, api.NodeMembershipAccepted, issueResponse.NodeMembership)

	// Check that the Certificate is successfully issued
	statusRequest := &api.NodeCertificateStatusRequest{NodeID: issueResponse.NodeID}
	statusResponse, err := tc.NodeCAClients[0].NodeCertificateStatus(tc.Context, statusRequest)
	require.NoError(t, err)
	assert.Equal(t, api.IssuanceStateIssued, statusResponse.Status.State)
	assert.NotNil(t, statusResponse.Certificate.Certificate)
	assert.Equal(t, api.NodeRoleWorker, statusResponse.Certificate.Role)

	// Update the certificate status to IssuanceStateRotate which should be a server-side noop
	err = tc.MemoryStore.Update(func(tx store.Tx) error {
		// Attempt to retrieve the node with nodeID
		node := store.GetNode(tx, issueResponse.NodeID)
		assert.NotNil(t, node)

		node.Certificate.Status.State = api.IssuanceStateRotate
		return store.UpdateNode(tx, node)
	})
	assert.NoError(t, err)

	// Wait a bit and check that the certificate hasn't changed/been reissued
	time.Sleep(250 * time.Millisecond)

	statusNewResponse, err := tc.NodeCAClients[0].NodeCertificateStatus(tc.Context, statusRequest)
	require.NoError(t, err)
	assert.Equal(t, statusResponse.Certificate.Certificate, statusNewResponse.Certificate.Certificate)
	assert.Equal(t, api.IssuanceStateRotate, statusNewResponse.Certificate.Status.State)
	assert.Equal(t, api.NodeRoleWorker, statusNewResponse.Certificate.Role)
}

func TestIssueNodeCertificateBrokenCA(t *testing.T) {
	if !cautils.External {
		t.Skip("test only applicable for external CA configuration")
	}

	tc := cautils.NewTestCA(t)
	defer tc.Stop()

	csr, _, err := ca.GenerateNewCSR()
	assert.NoError(t, err)

	tc.ExternalSigningServer.Flake()

	go func() {
		time.Sleep(250 * time.Millisecond)
		tc.ExternalSigningServer.Deflake()
	}()
	issueRequest := &api.IssueNodeCertificateRequest{CSR: csr, Token: tc.WorkerToken}
	issueResponse, err := tc.NodeCAClients[0].IssueNodeCertificate(tc.Context, issueRequest)
	assert.NoError(t, err)
	assert.NotNil(t, issueResponse.NodeID)
	assert.Equal(t, api.NodeMembershipAccepted, issueResponse.NodeMembership)

	statusRequest := &api.NodeCertificateStatusRequest{NodeID: issueResponse.NodeID}
	statusResponse, err := tc.NodeCAClients[0].NodeCertificateStatus(tc.Context, statusRequest)
	require.NoError(t, err)
	assert.Equal(t, api.IssuanceStateIssued, statusResponse.Status.State)
	assert.NotNil(t, statusResponse.Certificate.Certificate)
	assert.Equal(t, api.NodeRoleWorker, statusResponse.Certificate.Role)

}

func TestIssueNodeCertificateWithInvalidCSR(t *testing.T) {
	tc := cautils.NewTestCA(t)
	defer tc.Stop()

	issueRequest := &api.IssueNodeCertificateRequest{CSR: []byte("random garbage"), Token: tc.WorkerToken}
	issueResponse, err := tc.NodeCAClients[0].IssueNodeCertificate(tc.Context, issueRequest)
	assert.NoError(t, err)
	assert.NotNil(t, issueResponse.NodeID)
	assert.Equal(t, api.NodeMembershipAccepted, issueResponse.NodeMembership)

	statusRequest := &api.NodeCertificateStatusRequest{NodeID: issueResponse.NodeID}
	statusResponse, err := tc.NodeCAClients[0].NodeCertificateStatus(tc.Context, statusRequest)
	require.NoError(t, err)
	assert.Equal(t, api.IssuanceStateFailed, statusResponse.Status.State)
	assert.Contains(t, statusResponse.Status.Err, "CSR Decode failed")
	assert.Nil(t, statusResponse.Certificate.Certificate)
}

func TestIssueNodeCertificateWorkerRenewal(t *testing.T) {
	tc := cautils.NewTestCA(t)
	defer tc.Stop()

	csr, _, err := ca.GenerateNewCSR()
	assert.NoError(t, err)

	role := api.NodeRoleWorker
	issueRequest := &api.IssueNodeCertificateRequest{CSR: csr, Role: role}
	issueResponse, err := tc.NodeCAClients[1].IssueNodeCertificate(tc.Context, issueRequest)
	assert.NoError(t, err)
	assert.NotNil(t, issueResponse.NodeID)
	assert.Equal(t, api.NodeMembershipAccepted, issueResponse.NodeMembership)

	statusRequest := &api.NodeCertificateStatusRequest{NodeID: issueResponse.NodeID}
	statusResponse, err := tc.NodeCAClients[1].NodeCertificateStatus(tc.Context, statusRequest)
	require.NoError(t, err)
	assert.Equal(t, api.IssuanceStateIssued, statusResponse.Status.State)
	assert.NotNil(t, statusResponse.Certificate.Certificate)
	assert.Equal(t, role, statusResponse.Certificate.Role)
}

func TestIssueNodeCertificateManagerRenewal(t *testing.T) {
	tc := cautils.NewTestCA(t)
	defer tc.Stop()

	csr, _, err := ca.GenerateNewCSR()
	assert.NoError(t, err)
	assert.NotNil(t, csr)

	role := api.NodeRoleManager
	issueRequest := &api.IssueNodeCertificateRequest{CSR: csr, Role: role}
	issueResponse, err := tc.NodeCAClients[2].IssueNodeCertificate(tc.Context, issueRequest)
	require.NoError(t, err)
	assert.NotNil(t, issueResponse.NodeID)
	assert.Equal(t, api.NodeMembershipAccepted, issueResponse.NodeMembership)

	statusRequest := &api.NodeCertificateStatusRequest{NodeID: issueResponse.NodeID}
	statusResponse, err := tc.NodeCAClients[2].NodeCertificateStatus(tc.Context, statusRequest)
	require.NoError(t, err)
	assert.Equal(t, api.IssuanceStateIssued, statusResponse.Status.State)
	assert.NotNil(t, statusResponse.Certificate.Certificate)
	assert.Equal(t, role, statusResponse.Certificate.Role)
}

func TestIssueNodeCertificateWorkerFromDifferentOrgRenewal(t *testing.T) {
	tc := cautils.NewTestCA(t)
	defer tc.Stop()

	csr, _, err := ca.GenerateNewCSR()
	assert.NoError(t, err)

	// Since we're using a client that has a different Organization, this request will be treated
	// as a new certificate request, not allowing auto-renewal. Therefore, the request will fail.
	issueRequest := &api.IssueNodeCertificateRequest{CSR: csr}
	_, err = tc.NodeCAClients[3].IssueNodeCertificate(tc.Context, issueRequest)
	assert.Error(t, err)
}

func TestNodeCertificateRenewalsDoNotRequireToken(t *testing.T) {
	tc := cautils.NewTestCA(t)
	defer tc.Stop()

	csr, _, err := ca.GenerateNewCSR()
	assert.NoError(t, err)

	role := api.NodeRoleManager
	issueRequest := &api.IssueNodeCertificateRequest{CSR: csr, Role: role}
	issueResponse, err := tc.NodeCAClients[2].IssueNodeCertificate(tc.Context, issueRequest)
	assert.NoError(t, err)
	assert.NotNil(t, issueResponse.NodeID)
	assert.Equal(t, api.NodeMembershipAccepted, issueResponse.NodeMembership)

	statusRequest := &api.NodeCertificateStatusRequest{NodeID: issueResponse.NodeID}
	statusResponse, err := tc.NodeCAClients[2].NodeCertificateStatus(tc.Context, statusRequest)
	assert.NoError(t, err)
	assert.Equal(t, api.IssuanceStateIssued, statusResponse.Status.State)
	assert.NotNil(t, statusResponse.Certificate.Certificate)
	assert.Equal(t, role, statusResponse.Certificate.Role)

	role = api.NodeRoleWorker
	issueRequest = &api.IssueNodeCertificateRequest{CSR: csr, Role: role}
	issueResponse, err = tc.NodeCAClients[1].IssueNodeCertificate(tc.Context, issueRequest)
	require.NoError(t, err)
	assert.NotNil(t, issueResponse.NodeID)
	assert.Equal(t, api.NodeMembershipAccepted, issueResponse.NodeMembership)

	statusRequest = &api.NodeCertificateStatusRequest{NodeID: issueResponse.NodeID}
	statusResponse, err = tc.NodeCAClients[2].NodeCertificateStatus(tc.Context, statusRequest)
	require.NoError(t, err)
	assert.Equal(t, api.IssuanceStateIssued, statusResponse.Status.State)
	assert.NotNil(t, statusResponse.Certificate.Certificate)
	assert.Equal(t, role, statusResponse.Certificate.Role)
}

func TestNewNodeCertificateRequiresToken(t *testing.T) {
	t.Parallel()

	tc := cautils.NewTestCA(t)
	defer tc.Stop()

	csr, _, err := ca.GenerateNewCSR()
	assert.NoError(t, err)

	// Issuance fails if no secret is provided
	role := api.NodeRoleManager
	issueRequest := &api.IssueNodeCertificateRequest{CSR: csr, Role: role}
	_, err = tc.NodeCAClients[0].IssueNodeCertificate(tc.Context, issueRequest)
	assert.EqualError(t, err, "rpc error: code = InvalidArgument desc = A valid join token is necessary to join this cluster")

	role = api.NodeRoleWorker
	issueRequest = &api.IssueNodeCertificateRequest{CSR: csr, Role: role}
	_, err = tc.NodeCAClients[0].IssueNodeCertificate(tc.Context, issueRequest)
	assert.EqualError(t, err, "rpc error: code = InvalidArgument desc = A valid join token is necessary to join this cluster")

	// Issuance fails if wrong secret is provided
	role = api.NodeRoleManager
	issueRequest = &api.IssueNodeCertificateRequest{CSR: csr, Role: role, Token: "invalid-secret"}
	_, err = tc.NodeCAClients[0].IssueNodeCertificate(tc.Context, issueRequest)
	assert.EqualError(t, err, "rpc error: code = InvalidArgument desc = A valid join token is necessary to join this cluster")

	role = api.NodeRoleWorker
	issueRequest = &api.IssueNodeCertificateRequest{CSR: csr, Role: role, Token: "invalid-secret"}
	_, err = tc.NodeCAClients[0].IssueNodeCertificate(tc.Context, issueRequest)
	assert.EqualError(t, err, "rpc error: code = InvalidArgument desc = A valid join token is necessary to join this cluster")

	// Issuance succeeds if correct token is provided
	role = api.NodeRoleManager
	issueRequest = &api.IssueNodeCertificateRequest{CSR: csr, Role: role, Token: tc.ManagerToken}
	_, err = tc.NodeCAClients[0].IssueNodeCertificate(tc.Context, issueRequest)
	assert.NoError(t, err)

	role = api.NodeRoleWorker
	issueRequest = &api.IssueNodeCertificateRequest{CSR: csr, Role: role, Token: tc.WorkerToken}
	_, err = tc.NodeCAClients[0].IssueNodeCertificate(tc.Context, issueRequest)
	assert.NoError(t, err)

	// Rotate manager and worker tokens
	var (
		newManagerToken string
		newWorkerToken  string
	)
	assert.NoError(t, tc.MemoryStore.Update(func(tx store.Tx) error {
		clusters, _ := store.FindClusters(tx, store.ByName(store.DefaultClusterName))
		newWorkerToken = ca.GenerateJoinToken(&tc.RootCA, false)
		clusters[0].RootCA.JoinTokens.Worker = newWorkerToken
		newManagerToken = ca.GenerateJoinToken(&tc.RootCA, false)
		clusters[0].RootCA.JoinTokens.Manager = newManagerToken
		return store.UpdateCluster(tx, clusters[0])
	}))

	// updating the join token may take a little bit in order to register on the CA server, so poll
	assert.NoError(t, testutils.PollFunc(nil, func() error {
		// Old token should fail
		role = api.NodeRoleManager
		issueRequest = &api.IssueNodeCertificateRequest{CSR: csr, Role: role, Token: tc.ManagerToken}
		_, err = tc.NodeCAClients[0].IssueNodeCertificate(tc.Context, issueRequest)
		if err == nil {
			return fmt.Errorf("join token not updated yet")
		}
		return nil
	}))

	// Old token should fail
	assert.EqualError(t, err, "rpc error: code = InvalidArgument desc = A valid join token is necessary to join this cluster")

	role = api.NodeRoleWorker
	issueRequest = &api.IssueNodeCertificateRequest{CSR: csr, Role: role, Token: tc.WorkerToken}
	_, err = tc.NodeCAClients[0].IssueNodeCertificate(tc.Context, issueRequest)
	assert.EqualError(t, err, "rpc error: code = InvalidArgument desc = A valid join token is necessary to join this cluster")

	// New token should succeed
	role = api.NodeRoleManager
	issueRequest = &api.IssueNodeCertificateRequest{CSR: csr, Role: role, Token: newManagerToken}
	_, err = tc.NodeCAClients[0].IssueNodeCertificate(tc.Context, issueRequest)
	assert.NoError(t, err)

	role = api.NodeRoleWorker
	issueRequest = &api.IssueNodeCertificateRequest{CSR: csr, Role: role, Token: newWorkerToken}
	_, err = tc.NodeCAClients[0].IssueNodeCertificate(tc.Context, issueRequest)
	assert.NoError(t, err)
}

func TestNewNodeCertificateBadToken(t *testing.T) {
	tc := cautils.NewTestCA(t)
	defer tc.Stop()

	csr, _, err := ca.GenerateNewCSR()
	assert.NoError(t, err)

	// Issuance fails if wrong secret is provided
	role := api.NodeRoleManager
	issueRequest := &api.IssueNodeCertificateRequest{CSR: csr, Role: role, Token: "invalid-secret"}
	_, err = tc.NodeCAClients[0].IssueNodeCertificate(tc.Context, issueRequest)
	assert.EqualError(t, err, "rpc error: code = InvalidArgument desc = A valid join token is necessary to join this cluster")

	role = api.NodeRoleWorker
	issueRequest = &api.IssueNodeCertificateRequest{CSR: csr, Role: role, Token: "invalid-secret"}
	_, err = tc.NodeCAClients[0].IssueNodeCertificate(tc.Context, issueRequest)
	assert.EqualError(t, err, "rpc error: code = InvalidArgument desc = A valid join token is necessary to join this cluster")
}

func TestGetUnlockKey(t *testing.T) {
	t.Parallel()

	tc := cautils.NewTestCA(t)
	defer tc.Stop()

	var cluster *api.Cluster
	tc.MemoryStore.View(func(tx store.ReadTx) {
		clusters, err := store.FindClusters(tx, store.ByName(store.DefaultClusterName))
		require.NoError(t, err)
		cluster = clusters[0]
	})

	resp, err := tc.CAClients[0].GetUnlockKey(tc.Context, &api.GetUnlockKeyRequest{})
	require.NoError(t, err)
	require.Nil(t, resp.UnlockKey)
	require.Equal(t, cluster.Meta.Version, resp.Version)

	// Update the unlock key
	require.NoError(t, tc.MemoryStore.Update(func(tx store.Tx) error {
		cluster = store.GetCluster(tx, cluster.ID)
		cluster.Spec.EncryptionConfig.AutoLockManagers = true
		cluster.UnlockKeys = []*api.EncryptionKey{{
			Subsystem: ca.ManagerRole,
			Key:       []byte("secret"),
		}}
		return store.UpdateCluster(tx, cluster)
	}))

	tc.MemoryStore.View(func(tx store.ReadTx) {
		cluster = store.GetCluster(tx, cluster.ID)
	})

	require.NoError(t, testutils.PollFuncWithTimeout(nil, func() error {
		resp, err = tc.CAClients[0].GetUnlockKey(tc.Context, &api.GetUnlockKeyRequest{})
		if err != nil {
			return fmt.Errorf("get unlock key: %v", err)
		}
		if !bytes.Equal(resp.UnlockKey, []byte("secret")) {
			return fmt.Errorf("secret hasn't rotated yet")
		}
		if cluster.Meta.Version.Index > resp.Version.Index {
			return fmt.Errorf("hasn't updated to the right version yet")
		}
		return nil
	}, 250*time.Millisecond))
}

type clusterObjToUpdate struct {
	clusterObj           *api.Cluster
	rootCARoots          []byte
	rootCASigningCert    []byte
	rootCASigningKey     []byte
	rootCAIntermediates  []byte
	externalCertSignedBy []byte
}

// When the SecurityConfig is updated with a new TLS keypair, the server automatically uses that keypair to contact
// the external CA
func TestServerExternalCAGetsTLSKeypairUpdates(t *testing.T) {
	t.Parallel()

	// this one needs the external CA server for testing
	if !cautils.External {
		return
	}

	tc := cautils.NewTestCA(t)
	defer tc.Stop()

	// show that we can connect to the external CA using our original creds
	csr, _, err := ca.GenerateNewCSR()
	require.NoError(t, err)
	req := ca.PrepareCSR(csr, "cn", ca.ManagerRole, tc.Organization)

	externalCA := tc.CAServer.ExternalCA()
	extSignedCert, err := externalCA.Sign(tc.Context, req)
	require.NoError(t, err)
	require.NotNil(t, extSignedCert)

	// get a new cert and make it expired
	_, issuerInfo, err := tc.RootCA.IssueAndSaveNewCertificates(
		tc.KeyReadWriter, tc.ServingSecurityConfig.ClientTLSCreds.NodeID(), ca.ManagerRole, tc.Organization)
	require.NoError(t, err)
	cert, key, err := tc.KeyReadWriter.Read()
	require.NoError(t, err)

	s, err := tc.RootCA.Signer()
	require.NoError(t, err)
	cert = cautils.ReDateCert(t, cert, s.Cert, s.Key, time.Now().Add(-5*time.Hour), time.Now().Add(-3*time.Hour))

	// we have to create the keypair and update the security config manually, because all the renew functions check for
	// expiry
	tlsKeyPair, err := tls.X509KeyPair(cert, key)
	require.NoError(t, err)
	require.NoError(t, tc.ServingSecurityConfig.UpdateTLSCredentials(&tlsKeyPair, issuerInfo))

	// show that we now cannot connect to the external CA using our original creds
	require.NoError(t, testutils.PollFuncWithTimeout(nil, func() error {
		externalCA := tc.CAServer.ExternalCA()
		// wait for the credentials for the external CA to update
		if _, err = externalCA.Sign(tc.Context, req); err == nil {
			return errors.New("external CA creds haven't updated yet to be invalid")
		}
		return nil
	}, 2*time.Second))
	require.Contains(t, errors.Cause(err).Error(), "remote error: tls: bad certificate")
}

func TestCAServerUpdateRootCA(t *testing.T) {
	// this one needs both external CA servers for testing
	if !cautils.External {
		return
	}

	fakeClusterSpec := func(rootCerts, key []byte, rotation *api.RootRotation, externalCAs []*api.ExternalCA) *api.Cluster {
		return &api.Cluster{
			RootCA: api.RootCA{
				CACert:     rootCerts,
				CAKey:      key,
				CACertHash: "hash",
				JoinTokens: api.JoinTokens{
					Worker:  "SWMTKN-1-worker",
					Manager: "SWMTKN-1-manager",
				},
				RootRotation: rotation,
			},
			Spec: api.ClusterSpec{
				CAConfig: api.CAConfig{
					ExternalCAs: externalCAs,
				},
			},
		}
	}

	tc := cautils.NewTestCA(t)
	require.NoError(t, tc.CAServer.Stop())
	defer tc.Stop()

	cert, key, err := cautils.CreateRootCertAndKey("new root to rotate to")
	require.NoError(t, err)
	newRootCA, err := ca.NewRootCA(append(tc.RootCA.Certs, cert...), cert, key, ca.DefaultNodeCertExpiration, nil)
	require.NoError(t, err)
	externalServer, err := cautils.NewExternalSigningServer(newRootCA, tc.TempDir)
	require.NoError(t, err)
	defer externalServer.Stop()
	crossSigned, err := tc.RootCA.CrossSignCACertificate(cert)
	require.NoError(t, err)

	for i, testCase := range []clusterObjToUpdate{
		{
			clusterObj: fakeClusterSpec(tc.RootCA.Certs, nil, nil, []*api.ExternalCA{{
				Protocol: api.ExternalCA_CAProtocolCFSSL,
				URL:      tc.ExternalSigningServer.URL,
				// without a CA cert, the URL gets successfully added, and there should be no error connecting to it
			}}),
			rootCARoots:          tc.RootCA.Certs,
			externalCertSignedBy: tc.RootCA.Certs,
		},
		{
			clusterObj: fakeClusterSpec(tc.RootCA.Certs, nil, &api.RootRotation{
				CACert:            cert,
				CAKey:             key,
				CrossSignedCACert: crossSigned,
			}, []*api.ExternalCA{
				{
					Protocol: api.ExternalCA_CAProtocolCFSSL,
					URL:      tc.ExternalSigningServer.URL,
					// without a CA cert, we count this as the old tc.RootCA.Certs, and this should be ignored because we want the new root
				},
			}),
			rootCARoots:         tc.RootCA.Certs,
			rootCASigningCert:   crossSigned,
			rootCASigningKey:    key,
			rootCAIntermediates: crossSigned,
		},
		{
			clusterObj: fakeClusterSpec(tc.RootCA.Certs, nil, &api.RootRotation{
				CACert:            cert,
				CrossSignedCACert: crossSigned,
			}, []*api.ExternalCA{
				{
					Protocol: api.ExternalCA_CAProtocolCFSSL,
					URL:      tc.ExternalSigningServer.URL,
					// without a CA cert, we count this as the old tc.RootCA.Certs
				},
				{
					Protocol: api.ExternalCA_CAProtocolCFSSL,
					URL:      externalServer.URL,
					CACert:   append(cert, '\n'),
				},
			}),
			rootCARoots:          tc.RootCA.Certs,
			rootCAIntermediates:  crossSigned,
			externalCertSignedBy: cert,
		},
	} {
		require.NoError(t, tc.CAServer.UpdateRootCA(tc.Context, testCase.clusterObj, nil))

		rootCA := tc.CAServer.RootCA()
		require.Equal(t, testCase.rootCARoots, rootCA.Certs)
		var signingCert, signingKey []byte
		if s, err := rootCA.Signer(); err == nil {
			signingCert, signingKey = s.Cert, s.Key
		}
		require.Equal(t, testCase.rootCARoots, rootCA.Certs)
		require.Equal(t, testCase.rootCASigningCert, signingCert, "%d", i)
		require.Equal(t, testCase.rootCASigningKey, signingKey, "%d", i)
		require.Equal(t, testCase.rootCAIntermediates, rootCA.Intermediates)

		externalCA := tc.CAServer.ExternalCA()
		csr, _, err := ca.GenerateNewCSR()
		require.NoError(t, err)
		signedCert, err := externalCA.Sign(tc.Context, ca.PrepareCSR(csr, "cn", ca.ManagerRole, tc.Organization))

		if testCase.externalCertSignedBy != nil {
			require.NoError(t, err)
			parsed, err := helpers.ParseCertificatesPEM(signedCert)
			require.NoError(t, err)
			rootPool := x509.NewCertPool()
			rootPool.AppendCertsFromPEM(testCase.externalCertSignedBy)
			var intermediatePool *x509.CertPool
			if len(parsed) > 1 {
				intermediatePool = x509.NewCertPool()
				for _, cert := range parsed[1:] {
					intermediatePool.AddCert(cert)
				}
			}
			_, err = parsed[0].Verify(x509.VerifyOptions{Roots: rootPool, Intermediates: intermediatePool})
			require.NoError(t, err)
		} else {
			require.Equal(t, ca.ErrNoExternalCAURLs, err)
		}
	}
}

type rootRotationTester struct {
	tc *cautils.TestCA
	t  *testing.T
}

// go through all the nodes and update/create the ones we want, and delete the ones
// we don't
func (r *rootRotationTester) convergeWantedNodes(wantNodes map[string]*api.Node, descr string) {
	// update existing and create new nodes first before deleting nodes, else a root rotation
	// may finish early if all the nodes get deleted when the root rotation happens
	require.NoError(r.t, r.tc.MemoryStore.Update(func(tx store.Tx) error {
		for nodeID, wanted := range wantNodes {
			node := store.GetNode(tx, nodeID)
			if node == nil {
				if err := store.CreateNode(tx, wanted); err != nil {
					return err
				}
				continue
			}
			node.Description = wanted.Description
			node.Certificate = wanted.Certificate
			if err := store.UpdateNode(tx, node); err != nil {
				return err
			}
		}
		nodes, err := store.FindNodes(tx, store.All)
		if err != nil {
			return err
		}
		for _, node := range nodes {
			if _, inWanted := wantNodes[node.ID]; !inWanted {
				if err := store.DeleteNode(tx, node.ID); err != nil {
					return err
				}
			}
		}
		return nil
	}), descr)
}

func (r *rootRotationTester) convergeRootCA(wantRootCA *api.RootCA, descr string) {
	require.NoError(r.t, r.tc.MemoryStore.Update(func(tx store.Tx) error {
		clusters, err := store.FindClusters(tx, store.All)
		if err != nil || len(clusters) != 1 {
			return errors.Wrap(err, "unable to find cluster")
		}
		clusters[0].RootCA = *wantRootCA
		return store.UpdateCluster(tx, clusters[0])
	}), descr)
}

func getFakeAPINode(t *testing.T, id string, state api.IssuanceStatus_State, tlsInfo *api.NodeTLSInfo, member bool) *api.Node {
	node := &api.Node{
		ID: id,
		Certificate: api.Certificate{
			Status: api.IssuanceStatus{
				State: state,
			},
		},
		Spec: api.NodeSpec{
			Membership: api.NodeMembershipAccepted,
		},
	}
	if !member {
		node.Spec.Membership = api.NodeMembershipPending
	}
	// the CA server will immediately pick these up, so generate CSRs for the CA server to sign
	if state == api.IssuanceStateRenew || state == api.IssuanceStatePending {
		csr, _, err := ca.GenerateNewCSR()
		require.NoError(t, err)
		node.Certificate.CSR = csr
	}
	if tlsInfo != nil {
		node.Description = &api.NodeDescription{TLSInfo: tlsInfo}
	}
	return node
}

func startCAServer(ctx context.Context, caServer *ca.Server) {
	alreadyRunning := make(chan struct{})
	go func() {
		if err := caServer.Run(ctx); err != nil {
			close(alreadyRunning)
		}
	}()
	select {
	case <-caServer.Ready():
	case <-alreadyRunning:
	}
}

func getRotationInfo(t *testing.T, rotationCert []byte, rootCA *ca.RootCA) ([]byte, *api.NodeTLSInfo) {
	parsedNewRoot, err := helpers.ParseCertificatePEM(rotationCert)
	require.NoError(t, err)
	crossSigned, err := rootCA.CrossSignCACertificate(rotationCert)
	require.NoError(t, err)
	return crossSigned, &api.NodeTLSInfo{
		TrustRoot:           rootCA.Certs,
		CertIssuerPublicKey: parsedNewRoot.RawSubjectPublicKeyInfo,
		CertIssuerSubject:   parsedNewRoot.RawSubject,
	}
}

// These are the root rotation test cases where we expect there to be a change in the FindNodes
// or root CA values after converging.
func TestRootRotationReconciliationWithChanges(t *testing.T) {
	t.Parallel()
	if cautils.External {
		// the external CA functionality is unrelated to testing the reconciliation loop
		return
	}

	tc := cautils.NewTestCA(t)
	defer tc.Stop()
	rt := rootRotationTester{
		tc: tc,
		t:  t,
	}

	rotationCerts := [][]byte{cautils.ECDSA256SHA256Cert, cautils.ECDSACertChain[2]}
	rotationKeys := [][]byte{cautils.ECDSA256Key, cautils.ECDSACertChainKeys[2]}
	var (
		rotationCrossSigned [][]byte
		rotationTLSInfo     []*api.NodeTLSInfo
	)
	for _, cert := range rotationCerts {
		cross, info := getRotationInfo(t, cert, &tc.RootCA)
		rotationCrossSigned = append(rotationCrossSigned, cross)
		rotationTLSInfo = append(rotationTLSInfo, info)
	}

	oldNodeTLSInfo := &api.NodeTLSInfo{
		TrustRoot:           tc.RootCA.Certs,
		CertIssuerPublicKey: tc.ServingSecurityConfig.IssuerInfo().PublicKey,
		CertIssuerSubject:   tc.ServingSecurityConfig.IssuerInfo().Subject,
	}

	var startCluster *api.Cluster
	tc.MemoryStore.View(func(tx store.ReadTx) {
		startCluster = store.GetCluster(tx, tc.Organization)
	})
	require.NotNil(t, startCluster)

	testcases := []struct {
		nodes           map[string]*api.Node // what nodes we should start with
		rootCA          *api.RootCA          // what root CA we should start with
		expectedNodes   map[string]*api.Node // what nodes we expect in the end, if nil, then unchanged from the start
		expectedRootCA  *api.RootCA          // what root CA we expect in the end, if nil, then unchanged from the start
		caServerRestart bool                 // whether to stop the CA server before making the node and root changes and restart after
		descr           string
	}{
		{
			descr: ("If there is no TLS info, the reconciliation cycle tells the nodes to rotate if they're not already getting " +
				"a new cert.  Any renew/pending nodes will have certs issued, but because the TLS info is nil, they will " +
				`go "rotate" state`),
			nodes: map[string]*api.Node{
				"0": getFakeAPINode(t, "0", api.IssuanceStatePending, nil, false),
				"1": getFakeAPINode(t, "1", api.IssuanceStateIssued, nil, true),
				"2": getFakeAPINode(t, "2", api.IssuanceStateRenew, nil, true),
				"3": getFakeAPINode(t, "3", api.IssuanceStateRotate, nil, true),
				"4": getFakeAPINode(t, "4", api.IssuanceStatePending, nil, true),
				"5": getFakeAPINode(t, "5", api.IssuanceStateFailed, nil, true),
				"6": getFakeAPINode(t, "6", api.IssuanceStateIssued, nil, false),
			},
			rootCA: &api.RootCA{
				CACert:     startCluster.RootCA.CACert,
				CAKey:      startCluster.RootCA.CAKey,
				CACertHash: startCluster.RootCA.CACertHash,
				RootRotation: &api.RootRotation{
					CACert:            rotationCerts[0],
					CAKey:             rotationKeys[0],
					CrossSignedCACert: rotationCrossSigned[0],
				},
			},
			expectedNodes: map[string]*api.Node{
				"0": getFakeAPINode(t, "0", api.IssuanceStatePending, nil, false),
				"1": getFakeAPINode(t, "1", api.IssuanceStateRotate, nil, true),
				"2": getFakeAPINode(t, "2", api.IssuanceStateRotate, nil, true),
				"3": getFakeAPINode(t, "3", api.IssuanceStateRotate, nil, true),
				"4": getFakeAPINode(t, "4", api.IssuanceStateRotate, nil, true),
				"5": getFakeAPINode(t, "5", api.IssuanceStateRotate, nil, true),
				"6": getFakeAPINode(t, "6", api.IssuanceStateRotate, nil, false),
			},
		},
		{
			descr: ("Assume all of the nodes have gotten certs, but some of them are the wrong cert " +
				"(going by the TLS info), which shouldn't really happen.  the rotation reconciliation " +
				"will tell the wrong ones to rotate a second time"),
			nodes: map[string]*api.Node{
				"0": getFakeAPINode(t, "0", api.IssuanceStatePending, nil, false),
				"1": getFakeAPINode(t, "1", api.IssuanceStateIssued, rotationTLSInfo[0], true),
				"2": getFakeAPINode(t, "2", api.IssuanceStateIssued, oldNodeTLSInfo, true),
				"3": getFakeAPINode(t, "3", api.IssuanceStateIssued, rotationTLSInfo[0], true),
				"4": getFakeAPINode(t, "4", api.IssuanceStateIssued, rotationTLSInfo[0], true),
				"5": getFakeAPINode(t, "5", api.IssuanceStateIssued, oldNodeTLSInfo, true),
				"6": getFakeAPINode(t, "6", api.IssuanceStateIssued, oldNodeTLSInfo, false),
			},
			rootCA: &api.RootCA{ // no change in root CA from previous
				CACert:     startCluster.RootCA.CACert,
				CAKey:      startCluster.RootCA.CAKey,
				CACertHash: startCluster.RootCA.CACertHash,
				RootRotation: &api.RootRotation{
					CACert:            rotationCerts[0],
					CAKey:             rotationKeys[0],
					CrossSignedCACert: rotationCrossSigned[0],
				},
			},
			expectedNodes: map[string]*api.Node{
				"0": getFakeAPINode(t, "0", api.IssuanceStatePending, nil, false),
				"1": getFakeAPINode(t, "1", api.IssuanceStateIssued, rotationTLSInfo[0], true),
				"2": getFakeAPINode(t, "2", api.IssuanceStateRotate, oldNodeTLSInfo, true),
				"3": getFakeAPINode(t, "3", api.IssuanceStateIssued, rotationTLSInfo[0], true),
				"4": getFakeAPINode(t, "4", api.IssuanceStateIssued, rotationTLSInfo[0], true),
				"5": getFakeAPINode(t, "5", api.IssuanceStateRotate, oldNodeTLSInfo, true),
				"6": getFakeAPINode(t, "6", api.IssuanceStateRotate, oldNodeTLSInfo, false),
			},
		},
		{
			descr: ("New nodes that are added will also be picked up and told to rotate"),
			nodes: map[string]*api.Node{
				"0": getFakeAPINode(t, "0", api.IssuanceStatePending, nil, false),
				"1": getFakeAPINode(t, "1", api.IssuanceStateIssued, rotationTLSInfo[0], true),
				"3": getFakeAPINode(t, "3", api.IssuanceStateIssued, rotationTLSInfo[0], true),
				"4": getFakeAPINode(t, "4", api.IssuanceStateIssued, rotationTLSInfo[0], true),
				"5": getFakeAPINode(t, "5", api.IssuanceStateIssued, rotationTLSInfo[0], true),
				"6": getFakeAPINode(t, "6", api.IssuanceStateIssued, rotationTLSInfo[0], false),
				"7": getFakeAPINode(t, "7", api.IssuanceStateRenew, nil, true),
			},
			rootCA: &api.RootCA{ // no change in root CA from previous
				CACert:     startCluster.RootCA.CACert,
				CAKey:      startCluster.RootCA.CAKey,
				CACertHash: startCluster.RootCA.CACertHash,
				RootRotation: &api.RootRotation{
					CACert:            rotationCerts[0],
					CAKey:             rotationKeys[0],
					CrossSignedCACert: rotationCrossSigned[0],
				},
			},
			expectedNodes: map[string]*api.Node{
				"0": getFakeAPINode(t, "0", api.IssuanceStatePending, nil, false),
				"1": getFakeAPINode(t, "1", api.IssuanceStateIssued, rotationTLSInfo[0], true),
				"3": getFakeAPINode(t, "3", api.IssuanceStateIssued, rotationTLSInfo[0], true),
				"4": getFakeAPINode(t, "4", api.IssuanceStateIssued, rotationTLSInfo[0], true),
				"5": getFakeAPINode(t, "5", api.IssuanceStateIssued, rotationTLSInfo[0], true),
				"6": getFakeAPINode(t, "6", api.IssuanceStateIssued, rotationTLSInfo[0], false),
				"7": getFakeAPINode(t, "7", api.IssuanceStateRotate, nil, true),
			},
		},
		{
			descr: ("Even if root rotation isn't finished, if the root changes again to a " +
				"different cert, all the nodes with the old root rotation cert will be told " +
				"to rotate again."),
			nodes: map[string]*api.Node{
				"0": getFakeAPINode(t, "0", api.IssuanceStatePending, nil, false),
				"1": getFakeAPINode(t, "1", api.IssuanceStateIssued, rotationTLSInfo[0], true),
				"3": getFakeAPINode(t, "3", api.IssuanceStateIssued, rotationTLSInfo[1], true),
				"4": getFakeAPINode(t, "4", api.IssuanceStateIssued, rotationTLSInfo[0], true),
				"5": getFakeAPINode(t, "5", api.IssuanceStateIssued, oldNodeTLSInfo, true),
				"6": getFakeAPINode(t, "6", api.IssuanceStateIssued, rotationTLSInfo[0], true),
				"7": getFakeAPINode(t, "7", api.IssuanceStateIssued, rotationTLSInfo[0], false),
			},
			rootCA: &api.RootCA{ // new root rotation
				CACert:     startCluster.RootCA.CACert,
				CAKey:      startCluster.RootCA.CAKey,
				CACertHash: startCluster.RootCA.CACertHash,
				RootRotation: &api.RootRotation{
					CACert:            rotationCerts[1],
					CAKey:             rotationKeys[1],
					CrossSignedCACert: rotationCrossSigned[1],
				},
			},
			expectedNodes: map[string]*api.Node{
				"0": getFakeAPINode(t, "0", api.IssuanceStatePending, nil, false),
				"1": getFakeAPINode(t, "1", api.IssuanceStateRotate, rotationTLSInfo[0], true),
				"3": getFakeAPINode(t, "3", api.IssuanceStateIssued, rotationTLSInfo[1], true),
				"4": getFakeAPINode(t, "4", api.IssuanceStateRotate, rotationTLSInfo[0], true),
				"5": getFakeAPINode(t, "5", api.IssuanceStateRotate, oldNodeTLSInfo, true),
				"6": getFakeAPINode(t, "6", api.IssuanceStateRotate, rotationTLSInfo[0], true),
				"7": getFakeAPINode(t, "7", api.IssuanceStateRotate, rotationTLSInfo[0], false),
			},
		},
		{
			descr: ("Once all nodes have rotated to their desired TLS info (even if it's because " +
				"a node with the wrong TLS info has been removed, the root rotation is completed."),
			nodes: map[string]*api.Node{
				"0": getFakeAPINode(t, "0", api.IssuanceStateIssued, rotationTLSInfo[1], false),
				"1": getFakeAPINode(t, "1", api.IssuanceStateIssued, rotationTLSInfo[1], true),
				"3": getFakeAPINode(t, "3", api.IssuanceStateIssued, rotationTLSInfo[1], true),
				"4": getFakeAPINode(t, "4", api.IssuanceStateIssued, rotationTLSInfo[1], true),
				"6": getFakeAPINode(t, "6", api.IssuanceStateIssued, rotationTLSInfo[1], true),
			},
			rootCA: &api.RootCA{
				// no change in root CA from previous - even if root rotation gets completed after
				// the nodes are first set, and we just add the root rotation again because of this
				// test order, because the TLS info is correct for all nodes it will be completed again
				// anyway)
				CACert:     startCluster.RootCA.CACert,
				CAKey:      startCluster.RootCA.CAKey,
				CACertHash: startCluster.RootCA.CACertHash,
				RootRotation: &api.RootRotation{
					CACert:            rotationCerts[1],
					CAKey:             rotationKeys[1],
					CrossSignedCACert: rotationCrossSigned[1],
				},
			},
			expectedRootCA: &api.RootCA{
				CACert:     rotationCerts[1],
				CAKey:      rotationKeys[1],
				CACertHash: digest.FromBytes(rotationCerts[1]).String(),
				// ignore the join tokens - we aren't comparing them
			},
		},
		{
			descr: ("If a root rotation happens when the CA server is down, so long as it saw the change " +
				"it will start reconciling the nodes as soon as it's started up again"),
			caServerRestart: true,
			nodes: map[string]*api.Node{
				"0": getFakeAPINode(t, "0", api.IssuanceStatePending, nil, false),
				"1": getFakeAPINode(t, "1", api.IssuanceStateIssued, rotationTLSInfo[1], true),
				"3": getFakeAPINode(t, "3", api.IssuanceStateIssued, rotationTLSInfo[1], true),
				"4": getFakeAPINode(t, "4", api.IssuanceStateIssued, rotationTLSInfo[1], true),
				"6": getFakeAPINode(t, "6", api.IssuanceStateIssued, rotationTLSInfo[1], true),
				"7": getFakeAPINode(t, "7", api.IssuanceStateIssued, rotationTLSInfo[1], false),
			},
			rootCA: &api.RootCA{
				CACert:     startCluster.RootCA.CACert,
				CAKey:      startCluster.RootCA.CAKey,
				CACertHash: startCluster.RootCA.CACertHash,
				RootRotation: &api.RootRotation{
					CACert:            rotationCerts[0],
					CAKey:             rotationKeys[0],
					CrossSignedCACert: rotationCrossSigned[0],
				},
			},
			expectedNodes: map[string]*api.Node{
				"0": getFakeAPINode(t, "0", api.IssuanceStatePending, nil, false),
				"1": getFakeAPINode(t, "1", api.IssuanceStateRotate, rotationTLSInfo[1], true),
				"3": getFakeAPINode(t, "3", api.IssuanceStateRotate, rotationTLSInfo[1], true),
				"4": getFakeAPINode(t, "4", api.IssuanceStateRotate, rotationTLSInfo[1], true),
				"6": getFakeAPINode(t, "6", api.IssuanceStateRotate, rotationTLSInfo[1], true),
				"7": getFakeAPINode(t, "7", api.IssuanceStateRotate, rotationTLSInfo[1], false),
			},
		},
	}

	for _, testcase := range testcases {
		// stop the CA server, get the cluster to the state we want (correct root CA, correct nodes, etc.)
		rt.tc.CAServer.Stop()
		rt.convergeWantedNodes(testcase.nodes, testcase.descr)

		if testcase.caServerRestart {
			// if we want to simulate restarting the CA server with a root rotation already done, set the rootCA to
			// have a root rotation, then start the CA
			rt.convergeRootCA(testcase.rootCA, testcase.descr)
			startCAServer(rt.tc.Context, rt.tc.CAServer)
		} else {
			// otherwise, start the CA in the state where there is no root rotation, and start a root rotation
			rt.convergeRootCA(&startCluster.RootCA, testcase.descr) // no root rotation
			startCAServer(rt.tc.Context, rt.tc.CAServer)
			rt.convergeRootCA(testcase.rootCA, testcase.descr)
		}

		if testcase.expectedNodes == nil {
			testcase.expectedNodes = testcase.nodes
		}
		if testcase.expectedRootCA == nil {
			testcase.expectedRootCA = testcase.rootCA
		}

		require.NoError(t, testutils.PollFuncWithTimeout(nil, func() error {
			var (
				nodes   []*api.Node
				cluster *api.Cluster
				err     error
			)
			tc.MemoryStore.View(func(tx store.ReadTx) {
				nodes, err = store.FindNodes(tx, store.All)
				cluster = store.GetCluster(tx, tc.Organization)
			})
			if err != nil {
				return err
			}
			if cluster == nil {
				return errors.New("no cluster found")
			}

			if !equality.RootCAEqualStable(&cluster.RootCA, testcase.expectedRootCA) {
				return fmt.Errorf("root CAs not equal:\n\texpected: %v\n\tactual: %v", *testcase.expectedRootCA, cluster.RootCA)
			}
			if len(nodes) != len(testcase.expectedNodes) {
				return fmt.Errorf("number of expected nodes (%d) does not equal number of actual nodes (%d)",
					len(testcase.expectedNodes), len(nodes))
			}
			for _, node := range nodes {
				expected, ok := testcase.expectedNodes[node.ID]
				if !ok {
					return fmt.Errorf("node %s is present and was unexpected", node.ID)
				}
				if !reflect.DeepEqual(expected.Description, node.Description) {
					return fmt.Errorf("the node description of node %s is not expected:\n\texpected: %v\n\tactual: %v", node.ID,
						expected.Description, node.Description)
				}
				if !reflect.DeepEqual(expected.Certificate.Status, node.Certificate.Status) {
					return fmt.Errorf("the certificate status of node %s is not expected:\n\texpected: %v\n\tactual: %v", node.ID,
						expected.Certificate, node.Certificate)
				}

				// ensure that the security config's root CA object has the same expected key
				expectedKey := testcase.expectedRootCA.CAKey
				if testcase.expectedRootCA.RootRotation != nil {
					expectedKey = testcase.expectedRootCA.RootRotation.CAKey
				}
				s, err := rt.tc.CAServer.RootCA().Signer()
				if err != nil {
					return err
				}
				if !bytes.Equal(s.Key, expectedKey) {
					return fmt.Errorf("the CA Server's root CA has not been updated correctly")
				}
			}
			return nil
		}, 5*time.Second), testcase.descr)
	}
}

// These are the root rotation test cases where we expect there to be no changes made to either
// the nodes or the root CA object, although the server's signing root CA may change.
func TestRootRotationReconciliationNoChanges(t *testing.T) {
	t.Parallel()
	if cautils.External {
		// the external CA functionality is unrelated to testing the reconciliation loop
		return
	}

	tc := cautils.NewTestCA(t)
	defer tc.Stop()
	rt := rootRotationTester{
		tc: tc,
		t:  t,
	}

	rotationCert := cautils.ECDSA256SHA256Cert
	rotationKey := cautils.ECDSA256Key
	rotationCrossSigned, rotationTLSInfo := getRotationInfo(t, rotationCert, &tc.RootCA)

	oldNodeTLSInfo := &api.NodeTLSInfo{
		TrustRoot:           tc.RootCA.Certs,
		CertIssuerPublicKey: tc.ServingSecurityConfig.IssuerInfo().PublicKey,
		CertIssuerSubject:   tc.ServingSecurityConfig.IssuerInfo().Subject,
	}

	var startCluster *api.Cluster
	tc.MemoryStore.View(func(tx store.ReadTx) {
		startCluster = store.GetCluster(tx, tc.Organization)
	})
	require.NotNil(t, startCluster)

	testcases := []struct {
		nodes  map[string]*api.Node // what nodes we should start with
		rootCA *api.RootCA          // what root CA we should start with
		descr  string
	}{
		{
			descr: ("If all nodes have the right TLS info or are already rotated, rotating, or pending, " +
				"there will be no changes needed"),
			nodes: map[string]*api.Node{
				"0": getFakeAPINode(t, "0", api.IssuanceStatePending, nil, false),
				"1": getFakeAPINode(t, "1", api.IssuanceStateIssued, rotationTLSInfo, true),
				"2": getFakeAPINode(t, "2", api.IssuanceStateRotate, oldNodeTLSInfo, true),
				"3": getFakeAPINode(t, "3", api.IssuanceStateRotate, rotationTLSInfo, false),
			},
			rootCA: &api.RootCA{ // no change in root CA from previous
				CACert:     startCluster.RootCA.CACert,
				CAKey:      startCluster.RootCA.CAKey,
				CACertHash: startCluster.RootCA.CACertHash,
				RootRotation: &api.RootRotation{
					CACert:            rotationCert,
					CAKey:             rotationKey,
					CrossSignedCACert: rotationCrossSigned,
				},
			},
		},
		{
			descr: ("Nodes already in rotate state, even if they currently have the correct TLS issuer, will be " +
				"left in the rotate state even if root rotation is aborted because we don't know if they're already " +
				"in the process of getting a new cert.  Even if they're issued by a different issuer, they will be " +
				"left alone because they'll have an interemdiate that chains up to the old issuer."),
			nodes: map[string]*api.Node{
				"0": getFakeAPINode(t, "0", api.IssuanceStatePending, nil, false),
				"1": getFakeAPINode(t, "1", api.IssuanceStateIssued, rotationTLSInfo, true),
				"2": getFakeAPINode(t, "2", api.IssuanceStateRotate, oldNodeTLSInfo, true),
				"3": getFakeAPINode(t, "3", api.IssuanceStateRotate, oldNodeTLSInfo, false),
			},
			rootCA: &api.RootCA{ // no change in root CA from previous
				CACert:     startCluster.RootCA.CACert,
				CAKey:      startCluster.RootCA.CAKey,
				CACertHash: startCluster.RootCA.CACertHash,
			},
		},
	}

	for _, testcase := range testcases {
		// stop the CA server, get the cluster to the state we want (correct root CA, correct nodes, etc.)
		rt.tc.CAServer.Stop()
		rt.convergeWantedNodes(testcase.nodes, testcase.descr)
		rt.convergeRootCA(&startCluster.RootCA, testcase.descr) // no root rotation
		startCAServer(rt.tc.Context, rt.tc.CAServer)
		rt.convergeRootCA(testcase.rootCA, testcase.descr)

		time.Sleep(500 * time.Millisecond)

		var (
			nodes   []*api.Node
			cluster *api.Cluster
			err     error
		)

		tc.MemoryStore.View(func(tx store.ReadTx) {
			nodes, err = store.FindNodes(tx, store.All)
			cluster = store.GetCluster(tx, tc.Organization)
		})
		require.NoError(t, err)
		require.NotNil(t, cluster)
		require.Equal(t, cluster.RootCA, *testcase.rootCA, testcase.descr)
		require.Len(t, nodes, len(testcase.nodes), testcase.descr)
		for _, node := range nodes {
			expected, ok := testcase.nodes[node.ID]
			require.True(t, ok, "node %s: %s", node.ID, testcase.descr)
			require.Equal(t, expected.Description, node.Description, "node %s: %s", node.ID, testcase.descr)
			require.Equal(t, expected.Certificate.Status, node.Certificate.Status, "node %s: %s", node.ID, testcase.descr)
		}

		// ensure that the server's root CA object has the same expected key
		expectedKey := testcase.rootCA.CAKey
		if testcase.rootCA.RootRotation != nil {
			expectedKey = testcase.rootCA.RootRotation.CAKey
		}
		s, err := rt.tc.CAServer.RootCA().Signer()
		require.NoError(t, err, testcase.descr)
		require.Equal(t, s.Key, expectedKey, testcase.descr)
	}
}

// Tests if the root rotation changes while the reconciliation loop is going, eventually the root rotation will finish
// successfully (even if there's a competing reconciliation loop, for instance if there's a bug during leadership handoff).
func TestRootRotationReconciliationRace(t *testing.T) {
	t.Parallel()
	if cautils.External {
		// the external CA functionality is unrelated to testing the reconciliation loop
		return
	}

	tc := cautils.NewTestCA(t)
	defer tc.Stop()
	tc.CAServer.Stop() // we can't use the testCA's CA server because we need to inject extra behavior into the control loop
	rt := rootRotationTester{
		tc: tc,
		t:  t,
	}

	tempDir, err := ioutil.TempDir("", "competing-ca-server")
	require.NoError(t, err)
	defer os.RemoveAll(tempDir)

	var (
		otherServers   = make([]*ca.Server, 5)
		serverContexts = make([]context.Context, 5)
		paths          = make([]*ca.SecurityConfigPaths, 5)
	)

	for i := 0; i < 5; i++ { // to make sure we get some collision
		// start a competing CA server
		paths[i] = ca.NewConfigPaths(filepath.Join(tempDir, fmt.Sprintf("%d", i)))

		// the sec config is only used to get the organization, the initial root CA copy, and any updates to
		// TLS certificates, so all the servers can share the same one
		otherServers[i] = ca.NewServer(tc.MemoryStore, tc.ServingSecurityConfig)

		// offset each server's reconciliation interval somewhat so that some will
		// pre-empt others
		otherServers[i].SetRootReconciliationInterval(time.Millisecond * time.Duration((i+1)*10))
		serverContexts[i] = log.WithLogger(tc.Context, log.G(tc.Context).WithFields(logrus.Fields{
			"otherCAServer": i,
		}))
		startCAServer(serverContexts[i], otherServers[i])
		defer otherServers[i].Stop()
	}

	oldNodeTLSInfo := &api.NodeTLSInfo{
		TrustRoot:           tc.RootCA.Certs,
		CertIssuerPublicKey: tc.ServingSecurityConfig.IssuerInfo().PublicKey,
		CertIssuerSubject:   tc.ServingSecurityConfig.IssuerInfo().Subject,
	}

	nodes := make(map[string]*api.Node)
	for i := 0; i < 5; i++ {
		nodeID := fmt.Sprintf("%d", i)
		nodes[nodeID] = getFakeAPINode(t, nodeID, api.IssuanceStateIssued, oldNodeTLSInfo, true)
	}
	rt.convergeWantedNodes(nodes, "setting up nodes for root rotation race condition test")

	var rotationCert, rotationKey []byte
	for i := 0; i < 10; i++ {
		var (
			rotationCrossSigned []byte
			rotationTLSInfo     *api.NodeTLSInfo
			caRootCA            ca.RootCA
		)
		rotationCert, rotationKey, err = cautils.CreateRootCertAndKey(fmt.Sprintf("root cn %d", i))
		require.NoError(t, err)
		require.NoError(t, tc.MemoryStore.Update(func(tx store.Tx) error {
			cluster := store.GetCluster(tx, tc.Organization)
			if cluster == nil {
				return errors.New("cluster has disappeared")
			}
			rootCA := cluster.RootCA.Copy()
			caRootCA, err = ca.NewRootCA(rootCA.CACert, rootCA.CACert, rootCA.CAKey, ca.DefaultNodeCertExpiration, nil)
			if err != nil {
				return err
			}
			rotationCrossSigned, rotationTLSInfo = getRotationInfo(t, rotationCert, &caRootCA)
			rootCA.RootRotation = &api.RootRotation{
				CACert:            rotationCert,
				CAKey:             rotationKey,
				CrossSignedCACert: rotationCrossSigned,
			}
			cluster.RootCA = *rootCA
			return store.UpdateCluster(tx, cluster)
		}))
		for _, node := range nodes {
			node.Description.TLSInfo = rotationTLSInfo
		}
		rt.convergeWantedNodes(nodes, fmt.Sprintf("iteration %d", i))
	}

	require.NoError(t, testutils.PollFuncWithTimeout(nil, func() error {
		var cluster *api.Cluster
		tc.MemoryStore.View(func(tx store.ReadTx) {
			cluster = store.GetCluster(tx, tc.Organization)
		})
		if cluster == nil {
			return errors.New("cluster has disappeared")
		}
		if cluster.RootCA.RootRotation != nil {
			return errors.New("root rotation is still present")
		}
		if !bytes.Equal(cluster.RootCA.CACert, rotationCert) {
			return errors.New("expected root cert is wrong")
		}
		if !bytes.Equal(cluster.RootCA.CAKey, rotationKey) {
			return errors.New("expected root key is wrong")
		}
		for i, server := range otherServers {
			s, err := server.RootCA().Signer()
			if err != nil {
				return err
			}
			if !bytes.Equal(s.Key, rotationKey) {
				return errors.Errorf("server %d's root CAs hasn't been updated yet", i)
			}
		}
		return nil
	}, 5*time.Second))

	// all of the ca servers have the appropriate cert and key
}

// If there are a lot of nodes, we only update a small number of them at once.
func TestRootRotationReconciliationThrottled(t *testing.T) {
	t.Parallel()
	if cautils.External {
		// the external CA functionality is unrelated to testing the reconciliation loop
		return
	}

	tc := cautils.NewTestCA(t)
	defer tc.Stop()
	// immediately stop the CA server - we want to run our own
	tc.CAServer.Stop()

	caServer := ca.NewServer(tc.MemoryStore, tc.ServingSecurityConfig)
	// set the reconciliation interval to something ridiculous, so we can make sure the first
	// batch does update all of them
	caServer.SetRootReconciliationInterval(time.Hour)
	startCAServer(tc.Context, caServer)
	defer caServer.Stop()

	var (
		nodes []*api.Node
		err   error
	)
	tc.MemoryStore.View(func(tx store.ReadTx) {
		nodes, err = store.FindNodes(tx, store.All)
	})
	require.NoError(t, err)

	// create twice the batch size of nodes
	err = tc.MemoryStore.Batch(func(batch *store.Batch) error {
		for i := len(nodes); i < ca.IssuanceStateRotateMaxBatchSize*2; i++ {
			nodeID := fmt.Sprintf("%d", i)
			err := batch.Update(func(tx store.Tx) error {
				return store.CreateNode(tx, getFakeAPINode(t, nodeID, api.IssuanceStateIssued, nil, true))
			})
			if err != nil {
				return err
			}
		}
		return nil
	})
	require.NoError(t, err)

	rotationCert := cautils.ECDSA256SHA256Cert
	rotationKey := cautils.ECDSA256Key
	rotationCrossSigned, _ := getRotationInfo(t, rotationCert, &tc.RootCA)

	require.NoError(t, tc.MemoryStore.Update(func(tx store.Tx) error {
		cluster := store.GetCluster(tx, tc.Organization)
		if cluster == nil {
			return errors.New("cluster has disappeared")
		}
		rootCA := cluster.RootCA.Copy()
		rootCA.RootRotation = &api.RootRotation{
			CACert:            rotationCert,
			CAKey:             rotationKey,
			CrossSignedCACert: rotationCrossSigned,
		}
		cluster.RootCA = *rootCA
		return store.UpdateCluster(tx, cluster)
	}))

	checkRotationNumber := func() error {
		tc.MemoryStore.View(func(tx store.ReadTx) {
			nodes, err = store.FindNodes(tx, store.All)
		})
		var issuanceRotate int
		for _, n := range nodes {
			if n.Certificate.Status.State == api.IssuanceStateRotate {
				issuanceRotate += 1
			}
		}
		if issuanceRotate != ca.IssuanceStateRotateMaxBatchSize {
			return fmt.Errorf("expected %d, got %d", ca.IssuanceStateRotateMaxBatchSize, issuanceRotate)
		}
		return nil
	}

	require.NoError(t, testutils.PollFuncWithTimeout(nil, checkRotationNumber, 5*time.Second))
	// prove that it's not just because the updates haven't finished
	time.Sleep(time.Second)
	require.NoError(t, checkRotationNumber())
}
