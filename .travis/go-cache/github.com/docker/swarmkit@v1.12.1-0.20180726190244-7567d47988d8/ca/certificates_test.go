package ca_test

import (
	"crypto/ecdsa"
	"crypto/elliptic"
	cryptorand "crypto/rand"
	"crypto/sha256"
	"crypto/tls"
	"crypto/x509"
	"encoding/hex"
	"encoding/pem"
	"fmt"
	"io/ioutil"
	"net"
	"os"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"

	cfcsr "github.com/cloudflare/cfssl/csr"
	"github.com/cloudflare/cfssl/helpers"
	"github.com/cloudflare/cfssl/initca"
	"github.com/docker/go-events"
	"github.com/docker/swarmkit/api"
	"github.com/docker/swarmkit/ca"
	cautils "github.com/docker/swarmkit/ca/testutils"
	"github.com/docker/swarmkit/connectionbroker"
	"github.com/docker/swarmkit/identity"
	"github.com/docker/swarmkit/manager/state"
	"github.com/docker/swarmkit/manager/state/store"
	"github.com/docker/swarmkit/remotes"
	"github.com/docker/swarmkit/testutils"
	"github.com/opencontainers/go-digest"
	"github.com/phayes/permbits"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"golang.org/x/net/context"
)

func init() {
	ca.RenewTLSExponentialBackoff = events.ExponentialBackoffConfig{
		Base:   250 * time.Millisecond,
		Factor: 250 * time.Millisecond,
		Max:    1 * time.Hour,
	}
	ca.GetCertRetryInterval = 50 * time.Millisecond
}

func checkLeafCert(t *testing.T, certBytes []byte, issuerName, cn, ou, org string, additionalDNSNames ...string) []*x509.Certificate {
	certs, err := helpers.ParseCertificatesPEM(certBytes)
	require.NoError(t, err)
	require.NotEmpty(t, certs)
	require.Equal(t, issuerName, certs[0].Issuer.CommonName)
	require.Equal(t, cn, certs[0].Subject.CommonName)
	require.Equal(t, []string{ou}, certs[0].Subject.OrganizationalUnit)
	require.Equal(t, []string{org}, certs[0].Subject.Organization)

	require.Len(t, certs[0].DNSNames, len(additionalDNSNames)+2)
	for _, dnsName := range append(additionalDNSNames, cn, ou) {
		require.Contains(t, certs[0].DNSNames, dnsName)
	}
	return certs
}

// TestMain runs every test in this file twice - once with a local CA and
// again with an external CA server.
func TestMain(m *testing.M) {
	if status := m.Run(); status != 0 {
		os.Exit(status)
	}

	cautils.External = true
	os.Exit(m.Run())
}

func TestCreateRootCASaveRootCA(t *testing.T) {
	tempBaseDir, err := ioutil.TempDir("", "swarm-ca-test-")
	assert.NoError(t, err)
	defer os.RemoveAll(tempBaseDir)

	paths := ca.NewConfigPaths(tempBaseDir)

	rootCA, err := ca.CreateRootCA("rootCN")
	assert.NoError(t, err)

	err = ca.SaveRootCA(rootCA, paths.RootCA)
	assert.NoError(t, err)

	perms, err := permbits.Stat(paths.RootCA.Cert)
	assert.NoError(t, err)
	assert.False(t, perms.GroupWrite())
	assert.False(t, perms.OtherWrite())

	_, err = permbits.Stat(paths.RootCA.Key)
	assert.True(t, os.IsNotExist(err))

	// ensure that the cert that was written is already normalized
	written, err := ioutil.ReadFile(paths.RootCA.Cert)
	assert.NoError(t, err)
	assert.Equal(t, written, ca.NormalizePEMs(written))
}

func TestCreateRootCAExpiry(t *testing.T) {
	rootCA, err := ca.CreateRootCA("rootCN")
	assert.NoError(t, err)

	// Convert the certificate into an object to create a RootCA
	parsedCert, err := helpers.ParseCertificatePEM(rootCA.Certs)
	assert.NoError(t, err)
	duration, err := time.ParseDuration(ca.RootCAExpiration)
	assert.NoError(t, err)
	assert.True(t, time.Now().Add(duration).AddDate(0, -1, 0).Before(parsedCert.NotAfter))
}

func TestGetLocalRootCA(t *testing.T) {
	tempBaseDir, err := ioutil.TempDir("", "swarm-ca-test-")
	assert.NoError(t, err)
	defer os.RemoveAll(tempBaseDir)

	paths := ca.NewConfigPaths(tempBaseDir)

	// First, try to load the local Root CA with the certificate missing.
	_, err = ca.GetLocalRootCA(paths.RootCA)
	assert.Equal(t, ca.ErrNoLocalRootCA, err)

	// Create the local Root CA to ensure that we can reload it correctly.
	rootCA, err := ca.CreateRootCA("rootCN")
	assert.NoError(t, err)
	s, err := rootCA.Signer()
	assert.NoError(t, err)
	err = ca.SaveRootCA(rootCA, paths.RootCA)
	assert.NoError(t, err)

	// No private key here
	rootCA2, err := ca.GetLocalRootCA(paths.RootCA)
	assert.NoError(t, err)
	assert.Equal(t, rootCA.Certs, rootCA2.Certs)
	_, err = rootCA2.Signer()
	assert.Equal(t, err, ca.ErrNoValidSigner)

	// write private key and assert we can load it and sign
	assert.NoError(t, ioutil.WriteFile(paths.RootCA.Key, s.Key, os.FileMode(0600)))
	rootCA3, err := ca.GetLocalRootCA(paths.RootCA)
	assert.NoError(t, err)
	assert.Equal(t, rootCA.Certs, rootCA3.Certs)
	_, err = rootCA3.Signer()
	assert.NoError(t, err)

	// Try with a private key that does not match the CA cert public key.
	privKey, err := ecdsa.GenerateKey(elliptic.P256(), cryptorand.Reader)
	assert.NoError(t, err)
	privKeyBytes, err := x509.MarshalECPrivateKey(privKey)
	assert.NoError(t, err)
	privKeyPem := pem.EncodeToMemory(&pem.Block{
		Type:  "EC PRIVATE KEY",
		Bytes: privKeyBytes,
	})
	assert.NoError(t, ioutil.WriteFile(paths.RootCA.Key, privKeyPem, os.FileMode(0600)))

	_, err = ca.GetLocalRootCA(paths.RootCA)
	assert.EqualError(t, err, "certificate key mismatch")
}

func TestGetLocalRootCAInvalidCert(t *testing.T) {
	tempBaseDir, err := ioutil.TempDir("", "swarm-ca-test-")
	assert.NoError(t, err)
	defer os.RemoveAll(tempBaseDir)

	paths := ca.NewConfigPaths(tempBaseDir)

	// Write some garbage to the CA cert
	require.NoError(t, ioutil.WriteFile(paths.RootCA.Cert, []byte(`-----BEGIN CERTIFICATE-----\n
some random garbage\n
-----END CERTIFICATE-----`), 0644))

	_, err = ca.GetLocalRootCA(paths.RootCA)
	require.Error(t, err)
}

func TestGetLocalRootCAInvalidKey(t *testing.T) {
	tempBaseDir, err := ioutil.TempDir("", "swarm-ca-test-")
	assert.NoError(t, err)
	defer os.RemoveAll(tempBaseDir)

	paths := ca.NewConfigPaths(tempBaseDir)
	// Create the local Root CA to ensure that we can reload it correctly.
	rootCA, err := ca.CreateRootCA("rootCN")
	require.NoError(t, err)
	require.NoError(t, ca.SaveRootCA(rootCA, paths.RootCA))

	// Write some garbage to the root key - this will cause the loading to fail
	require.NoError(t, ioutil.WriteFile(paths.RootCA.Key, []byte(`-----BEGIN PRIVATE KEY-----\n
some random garbage\n
-----END PRIVATE KEY-----`), 0600))

	_, err = ca.GetLocalRootCA(paths.RootCA)
	require.Error(t, err)
}

func TestParseValidateAndSignCSR(t *testing.T) {
	rootCA, err := ca.CreateRootCA("rootCN")
	assert.NoError(t, err)

	csr, _, err := ca.GenerateNewCSR()
	assert.NoError(t, err)

	signedCert, err := rootCA.ParseValidateAndSignCSR(csr, "CN", "OU", "ORG")
	assert.NoError(t, err)
	assert.NotNil(t, signedCert)

	assert.Len(t, checkLeafCert(t, signedCert, "rootCN", "CN", "OU", "ORG"), 1)
}

func TestParseValidateAndSignMaliciousCSR(t *testing.T) {
	rootCA, err := ca.CreateRootCA("rootCN")
	assert.NoError(t, err)

	req := &cfcsr.CertificateRequest{
		Names: []cfcsr.Name{
			{
				O:  "maliciousOrg",
				OU: "maliciousOU",
				L:  "maliciousLocality",
			},
		},
		CN:         "maliciousCN",
		Hosts:      []string{"docker.com"},
		KeyRequest: &cfcsr.BasicKeyRequest{A: "ecdsa", S: 256},
	}

	csr, _, err := cfcsr.ParseRequest(req)
	assert.NoError(t, err)

	signedCert, err := rootCA.ParseValidateAndSignCSR(csr, "CN", "OU", "ORG")
	assert.NoError(t, err)
	assert.NotNil(t, signedCert)

	assert.Len(t, checkLeafCert(t, signedCert, "rootCN", "CN", "OU", "ORG"), 1)
}

func TestGetRemoteCA(t *testing.T) {
	tc := cautils.NewTestCA(t)
	defer tc.Stop()

	shaHash := sha256.New()
	shaHash.Write(tc.RootCA.Certs)
	md := shaHash.Sum(nil)
	mdStr := hex.EncodeToString(md)

	d, err := digest.Parse("sha256:" + mdStr)
	require.NoError(t, err)

	downloadedRootCA, err := ca.GetRemoteCA(tc.Context, d, tc.ConnBroker)
	require.NoError(t, err)
	require.Equal(t, downloadedRootCA.Certs, tc.RootCA.Certs)

	// update the test CA to include a multi-certificate bundle as the root - the digest
	// we use to verify with must be the digest of the whole bundle
	tmpDir, err := ioutil.TempDir("", "GetRemoteCA")
	require.NoError(t, err)
	defer os.RemoveAll(tmpDir)
	paths := ca.NewConfigPaths(tmpDir)
	otherRootCA, err := ca.CreateRootCA("other")
	require.NoError(t, err)

	comboCertBundle := append(tc.RootCA.Certs, otherRootCA.Certs...)
	s, err := tc.RootCA.Signer()
	require.NoError(t, err)
	require.NoError(t, tc.MemoryStore.Update(func(tx store.Tx) error {
		cluster := store.GetCluster(tx, tc.Organization)
		cluster.RootCA.CACert = comboCertBundle
		cluster.RootCA.CAKey = s.Key
		return store.UpdateCluster(tx, cluster)
	}))
	require.NoError(t, testutils.PollFunc(nil, func() error {
		_, err := ca.GetRemoteCA(tc.Context, d, tc.ConnBroker)
		if err == nil {
			return fmt.Errorf("testca's rootca hasn't updated yet")
		}
		require.Contains(t, err.Error(), "remote CA does not match fingerprint")
		return nil
	}))

	// If we provide the right digest, the root CA is updated and we can validate
	// certs signed by either one
	d = digest.FromBytes(comboCertBundle)
	downloadedRootCA, err = ca.GetRemoteCA(tc.Context, d, tc.ConnBroker)
	require.NoError(t, err)
	require.Equal(t, comboCertBundle, downloadedRootCA.Certs)
	require.Equal(t, 2, len(downloadedRootCA.Pool.Subjects()))

	for _, rootCA := range []ca.RootCA{tc.RootCA, otherRootCA} {
		krw := ca.NewKeyReadWriter(paths.Node, nil, nil)
		_, _, err := rootCA.IssueAndSaveNewCertificates(krw, "cn", "ou", "org")
		require.NoError(t, err)

		certPEM, _, err := krw.Read()
		require.NoError(t, err)

		cert, err := helpers.ParseCertificatesPEM(certPEM)
		require.NoError(t, err)

		chains, err := cert[0].Verify(x509.VerifyOptions{
			Roots: downloadedRootCA.Pool,
		})
		require.NoError(t, err)
		require.Len(t, chains, 1)
	}
}

func TestGetRemoteCAInvalidHash(t *testing.T) {
	tc := cautils.NewTestCA(t)
	defer tc.Stop()

	_, err := ca.GetRemoteCA(tc.Context, "sha256:2d2f968475269f0dde5299427cf74348ee1d6115b95c6e3f283e5a4de8da445b", tc.ConnBroker)
	assert.Error(t, err)
}

// returns the issuer as well as all the parsed certs returned from the request
func testRequestAndSaveNewCertificates(t *testing.T, tc *cautils.TestCA) (*ca.IssuerInfo, []*x509.Certificate) {
	// Copy the current RootCA without the signer
	rca := ca.RootCA{Certs: tc.RootCA.Certs, Pool: tc.RootCA.Pool}
	tlsCert, issuerInfo, err := rca.RequestAndSaveNewCertificates(tc.Context, tc.KeyReadWriter,
		ca.CertificateRequestConfig{
			Token:      tc.ManagerToken,
			ConnBroker: tc.ConnBroker,
		})
	require.NoError(t, err)
	require.NotNil(t, tlsCert)
	require.NotNil(t, issuerInfo)
	perms, err := permbits.Stat(tc.Paths.Node.Cert)
	require.NoError(t, err)
	require.False(t, perms.GroupWrite())
	require.False(t, perms.OtherWrite())

	certs, err := ioutil.ReadFile(tc.Paths.Node.Cert)
	require.NoError(t, err)
	require.Equal(t, certs, ca.NormalizePEMs(certs))

	// ensure that the same number of certs was written
	parsedCerts, err := helpers.ParseCertificatesPEM(certs)
	require.NoError(t, err)
	return issuerInfo, parsedCerts
}

func TestRequestAndSaveNewCertificatesNoIntermediate(t *testing.T) {
	t.Parallel()

	tc := cautils.NewTestCA(t)
	defer tc.Stop()
	issuerInfo, parsedCerts := testRequestAndSaveNewCertificates(t, tc)
	require.Len(t, parsedCerts, 1)

	root, err := helpers.ParseCertificatePEM(tc.RootCA.Certs)
	require.NoError(t, err)
	require.Equal(t, root.RawSubject, issuerInfo.Subject)
}

func TestRequestAndSaveNewCertificatesWithIntermediates(t *testing.T) {
	t.Parallel()

	// use a RootCA with an intermediate
	apiRootCA := api.RootCA{
		CACert: cautils.ECDSACertChain[2],
		CAKey:  cautils.ECDSACertChainKeys[2],
		RootRotation: &api.RootRotation{
			CACert:            cautils.ECDSACertChain[1],
			CAKey:             cautils.ECDSACertChainKeys[1],
			CrossSignedCACert: concat([]byte("   "), cautils.ECDSACertChain[1]),
		},
	}
	tempdir, err := ioutil.TempDir("", "test-request-and-save-new-certificates")
	require.NoError(t, err)
	defer os.RemoveAll(tempdir)

	tc := cautils.NewTestCAFromAPIRootCA(t, tempdir, apiRootCA, nil)
	defer tc.Stop()
	issuerInfo, parsedCerts := testRequestAndSaveNewCertificates(t, tc)
	require.Len(t, parsedCerts, 2)

	intermediate, err := helpers.ParseCertificatePEM(tc.RootCA.Intermediates)
	require.NoError(t, err)
	require.Equal(t, intermediate, parsedCerts[1])
	require.Equal(t, intermediate.RawSubject, issuerInfo.Subject)
	require.Equal(t, intermediate.RawSubjectPublicKeyInfo, issuerInfo.PublicKey)
}

func TestRequestAndSaveNewCertificatesWithKEKUpdate(t *testing.T) {
	t.Parallel()

	tc := cautils.NewTestCA(t)
	defer tc.Stop()

	// Copy the current RootCA without the signer
	rca := ca.RootCA{Certs: tc.RootCA.Certs, Pool: tc.RootCA.Pool}

	unencryptedKeyReader := ca.NewKeyReadWriter(tc.Paths.Node, nil, nil)

	// key for the manager and worker are both unencrypted
	for _, token := range []string{tc.ManagerToken, tc.WorkerToken} {
		_, _, err := rca.RequestAndSaveNewCertificates(tc.Context, tc.KeyReadWriter,
			ca.CertificateRequestConfig{
				Token:      token,
				ConnBroker: tc.ConnBroker,
			})
		require.NoError(t, err)

		// there was no encryption config in the remote, so the key should be unencrypted
		_, _, err = unencryptedKeyReader.Read()
		require.NoError(t, err)
	}

	// If there is a different kek in the remote store, when TLS certs are renewed the new key will
	// be encrypted with that kek
	require.NoError(t, tc.MemoryStore.Update(func(tx store.Tx) error {
		cluster := store.GetCluster(tx, tc.Organization)
		cluster.Spec.EncryptionConfig.AutoLockManagers = true
		cluster.UnlockKeys = []*api.EncryptionKey{{
			Subsystem: ca.ManagerRole,
			Key:       []byte("kek!"),
		}}
		return store.UpdateCluster(tx, cluster)
	}))
	require.NoError(t, os.RemoveAll(tc.Paths.Node.Cert))
	require.NoError(t, os.RemoveAll(tc.Paths.Node.Key))

	// key for the manager will be encrypted, but certs for the worker will not be
	for _, token := range []string{tc.ManagerToken, tc.WorkerToken} {
		_, _, err := rca.RequestAndSaveNewCertificates(tc.Context, tc.KeyReadWriter,
			ca.CertificateRequestConfig{
				Token:      token,
				ConnBroker: tc.ConnBroker,
			})
		require.NoError(t, err)

		// there was no encryption config in the remote, so the key should be unencrypted
		_, _, err = unencryptedKeyReader.Read()

		if token == tc.ManagerToken {
			require.Error(t, err)
			_, _, err = ca.NewKeyReadWriter(tc.Paths.Node, []byte("kek!"), nil).Read()
			require.NoError(t, err)
		} else {
			require.NoError(t, err)
		}
	}
}

// returns the issuer of the issued certificate and the parsed certs of the issued certificate
func testIssueAndSaveNewCertificates(t *testing.T, rca *ca.RootCA) {
	tempdir, err := ioutil.TempDir("", "test-issue-and-save-new-certificates")
	require.NoError(t, err)
	defer os.RemoveAll(tempdir)
	paths := ca.NewConfigPaths(tempdir)
	krw := ca.NewKeyReadWriter(paths.Node, nil, nil)

	var issuer *x509.Certificate
	if len(rca.Intermediates) > 0 {
		issuer, err = helpers.ParseCertificatePEM(rca.Intermediates)
		require.NoError(t, err)
	} else {
		issuer, err = helpers.ParseCertificatePEM(rca.Certs)
		require.NoError(t, err)
	}

	// Test the creation of a manager and worker certificate
	for _, role := range []string{ca.ManagerRole, ca.WorkerRole} {
		var additionalNames []string
		if role == ca.ManagerRole {
			additionalNames = []string{ca.CARole}
		}

		cert, issuerInfo, err := rca.IssueAndSaveNewCertificates(krw, "CN", role, "org")
		require.NoError(t, err)
		require.NotNil(t, cert)
		require.Equal(t, issuer.RawSubjectPublicKeyInfo, issuerInfo.PublicKey)
		require.Equal(t, issuer.RawSubject, issuerInfo.Subject)
		perms, err := permbits.Stat(paths.Node.Cert)
		require.NoError(t, err)
		require.False(t, perms.GroupWrite())
		require.False(t, perms.OtherWrite())

		certBytes, err := ioutil.ReadFile(paths.Node.Cert)
		require.NoError(t, err)
		parsed := checkLeafCert(t, certBytes, issuer.Subject.CommonName, "CN", role, "org", additionalNames...)
		if len(rca.Intermediates) > 0 {
			require.Len(t, parsed, 2)
			require.Equal(t, parsed[1], issuer)
		} else {
			require.Len(t, parsed, 1)
		}
	}
}

func TestIssueAndSaveNewCertificatesNoIntermediates(t *testing.T) {
	if cautils.External {
		return // this does not use the test CA at all
	}
	rca, err := ca.CreateRootCA("rootCN")
	require.NoError(t, err)
	testIssueAndSaveNewCertificates(t, &rca)
}

func TestIssueAndSaveNewCertificatesWithIntermediates(t *testing.T) {
	if cautils.External {
		return // this does not use the test CA at all
	}
	rca, err := ca.NewRootCA(cautils.ECDSACertChain[2], cautils.ECDSACertChain[1], cautils.ECDSACertChainKeys[1],
		ca.DefaultNodeCertExpiration, cautils.ECDSACertChain[1])
	require.NoError(t, err)
	testIssueAndSaveNewCertificates(t, &rca)
}

func TestGetRemoteSignedCertificate(t *testing.T) {
	tc := cautils.NewTestCA(t)
	defer tc.Stop()

	// Create a new CSR to be signed
	csr, _, err := ca.GenerateNewCSR()
	assert.NoError(t, err)

	certs, err := ca.GetRemoteSignedCertificate(tc.Context, csr, tc.RootCA.Pool,
		ca.CertificateRequestConfig{
			Token:      tc.ManagerToken,
			ConnBroker: tc.ConnBroker,
		})
	assert.NoError(t, err)
	assert.NotNil(t, certs)

	// Test the expiration for a manager certificate
	parsedCerts, err := helpers.ParseCertificatesPEM(certs)
	assert.NoError(t, err)
	assert.Len(t, parsedCerts, 1)
	assert.True(t, time.Now().Add(ca.DefaultNodeCertExpiration).AddDate(0, 0, -1).Before(parsedCerts[0].NotAfter))
	assert.True(t, time.Now().Add(ca.DefaultNodeCertExpiration).AddDate(0, 0, 1).After(parsedCerts[0].NotAfter))
	assert.Equal(t, parsedCerts[0].Subject.OrganizationalUnit[0], ca.ManagerRole)

	// Test the expiration for an worker certificate
	certs, err = ca.GetRemoteSignedCertificate(tc.Context, csr, tc.RootCA.Pool,
		ca.CertificateRequestConfig{
			Token:      tc.WorkerToken,
			ConnBroker: tc.ConnBroker,
		})
	assert.NoError(t, err)
	assert.NotNil(t, certs)
	parsedCerts, err = helpers.ParseCertificatesPEM(certs)
	assert.NoError(t, err)
	assert.Len(t, parsedCerts, 1)
	assert.True(t, time.Now().Add(ca.DefaultNodeCertExpiration).AddDate(0, 0, -1).Before(parsedCerts[0].NotAfter))
	assert.True(t, time.Now().Add(ca.DefaultNodeCertExpiration).AddDate(0, 0, 1).After(parsedCerts[0].NotAfter))
	assert.Equal(t, parsedCerts[0].Subject.OrganizationalUnit[0], ca.WorkerRole)
}

func TestGetRemoteSignedCertificateNodeInfo(t *testing.T) {
	tc := cautils.NewTestCA(t)
	defer tc.Stop()

	// Create a new CSR to be signed
	csr, _, err := ca.GenerateNewCSR()
	assert.NoError(t, err)

	cert, err := ca.GetRemoteSignedCertificate(tc.Context, csr, tc.RootCA.Pool,
		ca.CertificateRequestConfig{
			Token:      tc.WorkerToken,
			ConnBroker: tc.ConnBroker,
		})
	assert.NoError(t, err)
	assert.NotNil(t, cert)
}

// A CA Server implementation that doesn't actually sign anything - something else
// will have to update the memory store to have a valid value for a node
type nonSigningCAServer struct {
	tc               *cautils.TestCA
	server           *grpc.Server
	addr             string
	nodeStatusCalled int64
}

func newNonSigningCAServer(t *testing.T, tc *cautils.TestCA) *nonSigningCAServer {
	secConfig, err := tc.NewNodeConfig(ca.ManagerRole)
	require.NoError(t, err)
	serverOpts := []grpc.ServerOption{grpc.Creds(secConfig.ServerTLSCreds)}
	grpcServer := grpc.NewServer(serverOpts...)
	l, err := net.Listen("tcp", "127.0.0.1:0")
	require.NoError(t, err)

	n := &nonSigningCAServer{
		tc:     tc,
		addr:   l.Addr().String(),
		server: grpcServer,
	}

	api.RegisterNodeCAServer(grpcServer, n)
	go grpcServer.Serve(l)
	return n
}

func (n *nonSigningCAServer) stop(t *testing.T) {
	n.server.Stop()
}

func (n *nonSigningCAServer) getConnBroker() *connectionbroker.Broker {
	return connectionbroker.New(remotes.NewRemotes(api.Peer{Addr: n.addr}))
}

// only returns the status in the store
func (n *nonSigningCAServer) NodeCertificateStatus(ctx context.Context, request *api.NodeCertificateStatusRequest) (*api.NodeCertificateStatusResponse, error) {
	atomic.AddInt64(&n.nodeStatusCalled, 1)
	for {
		var node *api.Node
		n.tc.MemoryStore.View(func(tx store.ReadTx) {
			node = store.GetNode(tx, request.NodeID)
		})
		if node != nil && node.Certificate.Status.State == api.IssuanceStateIssued {
			return &api.NodeCertificateStatusResponse{
				Status:      &node.Certificate.Status,
				Certificate: &node.Certificate,
			}, nil
		}
		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		case <-time.After(500 * time.Millisecond):
		}
	}
}

func (n *nonSigningCAServer) IssueNodeCertificate(ctx context.Context, request *api.IssueNodeCertificateRequest) (*api.IssueNodeCertificateResponse, error) {
	nodeID := identity.NewID()
	role := api.NodeRoleWorker
	if n.tc.ManagerToken == request.Token {
		role = api.NodeRoleManager
	}

	// Create a new node
	err := n.tc.MemoryStore.Update(func(tx store.Tx) error {
		node := &api.Node{
			Role: role,
			ID:   nodeID,
			Certificate: api.Certificate{
				CSR:  request.CSR,
				CN:   nodeID,
				Role: role,
				Status: api.IssuanceStatus{
					State: api.IssuanceStatePending,
				},
			},
			Spec: api.NodeSpec{
				DesiredRole:  role,
				Membership:   api.NodeMembershipAccepted,
				Availability: request.Availability,
			},
		}

		return store.CreateNode(tx, node)
	})
	if err != nil {
		return nil, err
	}
	return &api.IssueNodeCertificateResponse{
		NodeID:         nodeID,
		NodeMembership: api.NodeMembershipAccepted,
	}, nil
}

func TestGetRemoteSignedCertificateWithPending(t *testing.T) {
	t.Parallel()
	if cautils.External {
		// we don't actually need an external signing server, since we're faking a CA server which doesn't really sign
		return
	}

	tc := cautils.NewTestCA(t)
	defer tc.Stop()
	require.NoError(t, tc.CAServer.Stop())

	// Create a new CSR to be signed
	csr, _, err := ca.GenerateNewCSR()
	require.NoError(t, err)

	updates, cancel := state.Watch(tc.MemoryStore.WatchQueue(), api.EventCreateNode{})
	defer cancel()

	fakeCAServer := newNonSigningCAServer(t, tc)
	defer fakeCAServer.stop(t)

	completed := make(chan error)
	defer close(completed)
	go func() {
		_, err := ca.GetRemoteSignedCertificate(tc.Context, csr, tc.RootCA.Pool,
			ca.CertificateRequestConfig{
				Token:      tc.WorkerToken,
				ConnBroker: fakeCAServer.getConnBroker(),
				// ensure the RPC call to get state is cancelled after 500 milliseconds
				NodeCertificateStatusRequestTimeout: 500 * time.Millisecond,
			})
		completed <- err
	}()

	var node *api.Node
	// wait for a new node to show up
	for node == nil {
		select {
		case event := <-updates: // we want to skip the first node, which is the test CA
			n := event.(api.EventCreateNode).Node.Copy()
			if n.Certificate.Status.State == api.IssuanceStatePending {
				node = n
			}
		}
	}

	// wait for the calls to NodeCertificateStatus to begin on the first signing server before we start timing
	require.NoError(t, testutils.PollFuncWithTimeout(nil, func() error {
		if atomic.LoadInt64(&fakeCAServer.nodeStatusCalled) == 0 {
			return fmt.Errorf("waiting for NodeCertificateStatus to be called")
		}
		return nil
	}, time.Second*2))

	// wait for 2.5 seconds and ensure that GetRemoteSignedCertificate has not returned with an error yet -
	// the first attempt to get the certificate status should have timed out after 500 milliseconds, but
	// it should have tried to poll again.  Add a few seconds for fudge time to make sure it's actually
	// still polling.
	select {
	case <-completed:
		require.FailNow(t, "GetRemoteSignedCertificate should wait at least 500 milliseconds")
	case <-time.After(2500 * time.Millisecond):
		// good, it's still polling so we can proceed with the test
	}
	require.True(t, atomic.LoadInt64(&fakeCAServer.nodeStatusCalled) > 1, "expected NodeCertificateStatus to have been polled more than once")

	// Directly update the status of the store
	err = tc.MemoryStore.Update(func(tx store.Tx) error {
		node.Certificate.Status.State = api.IssuanceStateIssued
		return store.UpdateNode(tx, node)
	})
	require.NoError(t, err)

	// Make sure GetRemoteSignedCertificate didn't return an error
	require.NoError(t, <-completed)

	// make sure if we time out the GetRemoteSignedCertificate call, it cancels immediately and doesn't keep
	// polling the status
	go func() {
		ctx, _ := context.WithTimeout(tc.Context, 1*time.Second)
		_, err := ca.GetRemoteSignedCertificate(ctx, csr, tc.RootCA.Pool,
			ca.CertificateRequestConfig{
				Token:      tc.WorkerToken,
				ConnBroker: fakeCAServer.getConnBroker(),
			})
		completed <- err
	}()

	// wait for 3 seconds and ensure that GetRemoteSignedCertificate has returned with a context DeadlineExceeded
	// error - it should have returned after 1 second, but add some more for rudge time.
	select {
	case err = <-completed:
		require.Equal(t, grpc.Code(err), codes.DeadlineExceeded)
	case <-time.After(3 * time.Second):
		require.FailNow(t, "GetRemoteSignedCertificate should have been canceled after 1 second, and it has been 3")
	}
}

// fake remotes interface that just selects the remotes in order
type fakeRemotes struct {
	mu    sync.Mutex
	peers []api.Peer
}

func (f *fakeRemotes) Weights() map[api.Peer]int {
	panic("this is not called")
}

func (f *fakeRemotes) Select(...string) (api.Peer, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	if len(f.peers) > 0 {
		return f.peers[0], nil
	}
	return api.Peer{}, fmt.Errorf("no more peers")
}

func (f *fakeRemotes) Observe(peer api.Peer, weight int) {
	panic("this is not called")
}

// just removes a peer if the weight is negative
func (f *fakeRemotes) ObserveIfExists(peer api.Peer, weight int) {
	f.mu.Lock()
	defer f.mu.Unlock()
	if weight < 0 {
		var newPeers []api.Peer
		for _, p := range f.peers {
			if p != peer {
				newPeers = append(newPeers, p)
			}
		}
		f.peers = newPeers
	}
}

func (f *fakeRemotes) Remove(addrs ...api.Peer) {
	panic("this is not called")
}

var _ remotes.Remotes = &fakeRemotes{}

// On connection errors, so long as they happen after IssueNodeCertificate is successful, GetRemoteSignedCertificate
// tries to open a new connection and continue polling for NodeCertificateStatus.  If there are no more connections,
// then fail.
func TestGetRemoteSignedCertificateConnectionErrors(t *testing.T) {
	t.Parallel()
	if cautils.External {
		// we don't actually need an external signing server, since we're faking a CA server which doesn't really sign
		return
	}

	tc := cautils.NewTestCA(t)
	defer tc.Stop()
	require.NoError(t, tc.CAServer.Stop())

	// Create a new CSR to be signed
	csr, _, err := ca.GenerateNewCSR()
	require.NoError(t, err)

	// create 2 CA servers referencing the same memory store, so we can have multiple connections
	fakeSigningServers := []*nonSigningCAServer{newNonSigningCAServer(t, tc), newNonSigningCAServer(t, tc)}
	defer fakeSigningServers[0].stop(t)
	defer fakeSigningServers[1].stop(t)
	multiBroker := connectionbroker.New(&fakeRemotes{
		peers: []api.Peer{
			{Addr: fakeSigningServers[0].addr},
			{Addr: fakeSigningServers[1].addr},
		},
	})

	completed, done := make(chan error), make(chan struct{})
	defer close(completed)
	defer close(done)
	go func() {
		_, err := ca.GetRemoteSignedCertificate(tc.Context, csr, tc.RootCA.Pool,
			ca.CertificateRequestConfig{
				Token:      tc.WorkerToken,
				ConnBroker: multiBroker,
			})
		select {
		case <-done:
		case completed <- err:
		}
	}()

	// wait for the calls to NodeCertificateStatus to begin on the first signing server
	require.NoError(t, testutils.PollFuncWithTimeout(nil, func() error {
		if atomic.LoadInt64(&fakeSigningServers[0].nodeStatusCalled) == 0 {
			return fmt.Errorf("waiting for NodeCertificateStatus to be called")
		}
		return nil
	}, time.Second*2))

	// stop 1 server, because it will have been the remote GetRemoteSignedCertificate first connected to, and ensure
	// that GetRemoteSignedCertificate is still going
	fakeSigningServers[0].stop(t)
	select {
	case <-completed:
		require.FailNow(t, "GetRemoteSignedCertificate should still be going after 2.5 seconds")
	case <-time.After(2500 * time.Millisecond):
		// good, it's still polling so we can proceed with the test
	}

	// wait for the calls to NodeCertificateStatus to begin on the second signing server
	require.NoError(t, testutils.PollFuncWithTimeout(nil, func() error {
		if atomic.LoadInt64(&fakeSigningServers[1].nodeStatusCalled) == 0 {
			return fmt.Errorf("waiting for NodeCertificateStatus to be called")
		}
		return nil
	}, time.Second*2))

	// kill the last server - this should cause GetRemoteSignedCertificate to fail because there are no more peers
	fakeSigningServers[1].stop(t)
	// wait for 5 seconds and ensure that GetRemoteSignedCertificate has returned with an error.
	select {
	case err = <-completed:
		require.Contains(t, err.Error(), "no more peers")
	case <-time.After(5 * time.Second):
		require.FailNow(t, "GetRemoteSignedCertificate should errored after 5 seconds")
	}

	// calling GetRemoteSignedCertificate with a connection that doesn't work with IssueNodeCertificate will fail
	// immediately without retrying with a new connection
	fakeSigningServers[1] = newNonSigningCAServer(t, tc)
	defer fakeSigningServers[1].stop(t)
	multiBroker = connectionbroker.New(&fakeRemotes{
		peers: []api.Peer{
			{Addr: fakeSigningServers[0].addr},
			{Addr: fakeSigningServers[1].addr},
		},
	})
	_, err = ca.GetRemoteSignedCertificate(tc.Context, csr, tc.RootCA.Pool,
		ca.CertificateRequestConfig{
			Token:      tc.WorkerToken,
			ConnBroker: multiBroker,
		})
	require.Error(t, err)
}

func TestNewRootCA(t *testing.T) {
	for _, pair := range []struct{ cert, key []byte }{
		{cert: cautils.ECDSA256SHA256Cert, key: cautils.ECDSA256Key},
		{cert: cautils.RSA2048SHA256Cert, key: cautils.RSA2048Key},
	} {
		rootCA, err := ca.NewRootCA(pair.cert, pair.cert, pair.key, ca.DefaultNodeCertExpiration, nil)
		require.NoError(t, err, string(pair.key))
		require.Equal(t, pair.cert, rootCA.Certs)
		s, err := rootCA.Signer()
		require.NoError(t, err)
		require.Equal(t, pair.key, s.Key)
		_, err = rootCA.Digest.Verifier().Write(pair.cert)
		require.NoError(t, err)
	}
}

func TestNewRootCABundle(t *testing.T) {
	tempBaseDir, err := ioutil.TempDir("", "swarm-ca-test-")
	assert.NoError(t, err)
	defer os.RemoveAll(tempBaseDir)

	paths := ca.NewConfigPaths(tempBaseDir)

	// make one rootCA
	firstRootCA, err := ca.CreateRootCA("rootCN1")
	assert.NoError(t, err)

	// make a second root CA
	secondRootCA, err := ca.CreateRootCA("rootCN2")
	assert.NoError(t, err)
	s, err := firstRootCA.Signer()
	require.NoError(t, err)

	// Overwrite the bytes of the second Root CA with the bundle, creating a valid 2 cert bundle
	bundle := append(firstRootCA.Certs, secondRootCA.Certs...)
	err = ioutil.WriteFile(paths.RootCA.Cert, bundle, 0644)
	assert.NoError(t, err)

	newRootCA, err := ca.NewRootCA(bundle, firstRootCA.Certs, s.Key, ca.DefaultNodeCertExpiration, nil)
	assert.NoError(t, err)
	assert.Equal(t, bundle, newRootCA.Certs)
	assert.Equal(t, 2, len(newRootCA.Pool.Subjects()))

	// If I use newRootCA's IssueAndSaveNewCertificates to sign certs, I'll get the correct CA in the chain
	kw := ca.NewKeyReadWriter(paths.Node, nil, nil)
	_, _, err = newRootCA.IssueAndSaveNewCertificates(kw, "CN", "OU", "ORG")
	assert.NoError(t, err)

	certBytes, err := ioutil.ReadFile(paths.Node.Cert)
	assert.NoError(t, err)
	assert.Len(t, checkLeafCert(t, certBytes, "rootCN1", "CN", "OU", "ORG"), 1)
}

func TestNewRootCANonDefaultExpiry(t *testing.T) {
	rootCA, err := ca.CreateRootCA("rootCN")
	assert.NoError(t, err)
	s, err := rootCA.Signer()
	require.NoError(t, err)

	newRootCA, err := ca.NewRootCA(rootCA.Certs, rootCA.Certs, s.Key, 1*time.Hour, nil)
	assert.NoError(t, err)

	// Create and sign a new CSR
	csr, _, err := ca.GenerateNewCSR()
	assert.NoError(t, err)
	cert, err := newRootCA.ParseValidateAndSignCSR(csr, "CN", ca.ManagerRole, "ORG")
	assert.NoError(t, err)

	parsedCerts, err := helpers.ParseCertificatesPEM(cert)
	assert.NoError(t, err)
	assert.Len(t, parsedCerts, 1)
	assert.True(t, time.Now().Add(time.Minute*59).Before(parsedCerts[0].NotAfter))
	assert.True(t, time.Now().Add(time.Hour).Add(time.Minute).After(parsedCerts[0].NotAfter))

	// Sign the same CSR again, this time with a 59 Minute expiration RootCA (under the 60 minute minimum).
	// This should use the default of 3 months
	newRootCA, err = ca.NewRootCA(rootCA.Certs, rootCA.Certs, s.Key, 59*time.Minute, nil)
	assert.NoError(t, err)

	cert, err = newRootCA.ParseValidateAndSignCSR(csr, "CN", ca.ManagerRole, "ORG")
	assert.NoError(t, err)

	parsedCerts, err = helpers.ParseCertificatesPEM(cert)
	assert.NoError(t, err)
	assert.Len(t, parsedCerts, 1)
	assert.True(t, time.Now().Add(ca.DefaultNodeCertExpiration).AddDate(0, 0, -1).Before(parsedCerts[0].NotAfter))
	assert.True(t, time.Now().Add(ca.DefaultNodeCertExpiration).AddDate(0, 0, 1).After(parsedCerts[0].NotAfter))
}

type invalidNewRootCATestCase struct {
	roots, cert, key, intermediates []byte
	errorStr                        string
}

func TestNewRootCAInvalidCertAndKeys(t *testing.T) {
	now := time.Now()

	expiredIntermediate := cautils.ReDateCert(t, cautils.ECDSACertChain[1],
		cautils.ECDSACertChain[2], cautils.ECDSACertChainKeys[2], now.Add(-10*time.Hour), now.Add(-1*time.Minute))
	notYetValidIntermediate := cautils.ReDateCert(t, cautils.ECDSACertChain[1],
		cautils.ECDSACertChain[2], cautils.ECDSACertChainKeys[2], now.Add(time.Hour), now.Add(2*time.Hour))

	certChainRootCA, err := ca.NewRootCA(cautils.ECDSACertChain[2], cautils.ECDSACertChain[2], cautils.ECDSACertChainKeys[2],
		ca.DefaultNodeCertExpiration, nil)
	require.NoError(t, err)

	cert, _, _ := cautils.CreateRootCertAndKey("alternateIntermediate")
	alternateIntermediate, err := certChainRootCA.CrossSignCACertificate(cert)
	require.NoError(t, err)

	invalids := []invalidNewRootCATestCase{
		// invalid root or signer cert
		{
			roots:    []byte("malformed"),
			cert:     cautils.ECDSA256SHA256Cert,
			key:      cautils.ECDSA256Key,
			errorStr: "Failed to decode certificate",
		},
		{
			roots:    cautils.ECDSA256SHA256Cert,
			cert:     []byte("malformed"),
			key:      cautils.ECDSA256Key,
			errorStr: "Failed to decode certificate",
		},
		{
			roots:    []byte("  "),
			cert:     cautils.ECDSA256SHA256Cert,
			key:      cautils.ECDSA256Key,
			errorStr: "no valid root CA certificates found",
		},
		{
			roots:    cautils.ECDSA256SHA256Cert,
			cert:     []byte("  "),
			key:      cautils.ECDSA256Key,
			errorStr: "no valid signing CA certificates found",
		},
		{
			roots:    cautils.NotYetValidCert,
			cert:     cautils.ECDSA256SHA256Cert,
			key:      cautils.ECDSA256Key,
			errorStr: "not yet valid",
		},
		{
			roots:    cautils.ECDSA256SHA256Cert,
			cert:     cautils.NotYetValidCert,
			key:      cautils.NotYetValidKey,
			errorStr: "not yet valid",
		},
		{
			roots:    cautils.ExpiredCert,
			cert:     cautils.ECDSA256SHA256Cert,
			key:      cautils.ECDSA256Key,
			errorStr: "expired",
		},
		{
			roots:    cautils.ExpiredCert,
			cert:     cautils.ECDSA256SHA256Cert,
			key:      cautils.ECDSA256Key,
			errorStr: "expired",
		},
		{
			roots:    cautils.RSA2048SHA1Cert,
			cert:     cautils.ECDSA256SHA256Cert,
			key:      cautils.ECDSA256Key,
			errorStr: "unsupported signature algorithm",
		},
		{
			roots:    cautils.ECDSA256SHA256Cert,
			cert:     cautils.RSA2048SHA1Cert,
			key:      cautils.RSA2048Key,
			errorStr: "unsupported signature algorithm",
		},
		{
			roots:    cautils.ECDSA256SHA256Cert,
			cert:     cautils.ECDSA256SHA1Cert,
			key:      cautils.ECDSA256Key,
			errorStr: "unsupported signature algorithm",
		},
		{
			roots:    cautils.ECDSA256SHA1Cert,
			cert:     cautils.ECDSA256SHA256Cert,
			key:      cautils.ECDSA256Key,
			errorStr: "unsupported signature algorithm",
		},
		{
			roots:    cautils.ECDSA256SHA256Cert,
			cert:     cautils.DSA2048Cert,
			key:      cautils.DSA2048Key,
			errorStr: "unsupported signature algorithm",
		},
		{
			roots:    cautils.DSA2048Cert,
			cert:     cautils.ECDSA256SHA256Cert,
			key:      cautils.ECDSA256Key,
			errorStr: "unsupported signature algorithm",
		},
		// invalid signer
		{
			roots:    cautils.ECDSA256SHA256Cert,
			cert:     cautils.ECDSA256SHA256Cert,
			key:      []byte("malformed"),
			errorStr: "malformed private key",
		},
		{
			roots:    cautils.RSA1024Cert,
			cert:     cautils.RSA1024Cert,
			key:      cautils.RSA1024Key,
			errorStr: "unsupported RSA key parameters",
		},
		{
			roots:    cautils.ECDSA224Cert,
			cert:     cautils.ECDSA224Cert,
			key:      cautils.ECDSA224Key,
			errorStr: "unsupported ECDSA key parameters",
		},
		{
			roots:    cautils.ECDSA256SHA256Cert,
			cert:     cautils.ECDSA256SHA256Cert,
			key:      cautils.ECDSA224Key,
			errorStr: "certificate key mismatch",
		},
		{
			roots:    cautils.ECDSA256SHA256Cert,
			cert:     cautils.ECDSACertChain[1],
			key:      cautils.ECDSACertChainKeys[1],
			errorStr: "unknown authority", // signer cert doesn't chain up to the root
		},
		// invalid intermediates
		{
			roots:         cautils.ECDSACertChain[2],
			cert:          cautils.ECDSACertChain[1],
			key:           cautils.ECDSACertChainKeys[1],
			intermediates: []byte("malformed"),
			errorStr:      "Failed to decode certificate",
		},
		{
			roots:         cautils.ECDSACertChain[2],
			cert:          cautils.ECDSACertChain[1],
			key:           cautils.ECDSACertChainKeys[1],
			intermediates: expiredIntermediate,
			errorStr:      "expired",
		},
		{
			roots:         cautils.ECDSACertChain[2],
			cert:          cautils.ECDSACertChain[1],
			key:           cautils.ECDSACertChainKeys[1],
			intermediates: notYetValidIntermediate,
			errorStr:      "expired",
		},
		{
			roots:         cautils.ECDSACertChain[2],
			cert:          cautils.ECDSACertChain[1],
			key:           cautils.ECDSACertChainKeys[1],
			intermediates: append(cautils.ECDSACertChain[1], cautils.ECDSA256SHA256Cert...),
			errorStr:      "do not form a chain",
		},
		{
			roots:         cautils.ECDSACertChain[2],
			cert:          cautils.ECDSACertChain[1],
			key:           cautils.ECDSACertChainKeys[1],
			intermediates: cautils.ECDSA256SHA256Cert,
			errorStr:      "unknown authority", // intermediates don't chain up to root
		},
		{
			roots:         cautils.ECDSACertChain[2],
			cert:          cautils.ECDSACertChain[1],
			key:           cautils.ECDSACertChainKeys[1],
			intermediates: alternateIntermediate,
			errorStr:      "the first intermediate must have the same subject and public key as the signing cert",
		},
	}

	for i, invalid := range invalids {
		_, err := ca.NewRootCA(invalid.roots, invalid.cert, invalid.key, ca.DefaultNodeCertExpiration, invalid.intermediates)
		require.Error(t, err, fmt.Sprintf("expected error containing: \"%s\", test case (%d)", invalid.errorStr, i))
		require.Contains(t, err.Error(), invalid.errorStr, fmt.Sprintf("%d", i))
	}
}

func TestRootCAWithCrossSignedIntermediates(t *testing.T) {
	tempdir, err := ioutil.TempDir("", "swarm-ca-test-")
	require.NoError(t, err)
	defer os.RemoveAll(tempdir)

	// re-generate the intermediate to be a self-signed root, and use that as the second root
	parsedKey, err := helpers.ParsePrivateKeyPEM(cautils.ECDSACertChainKeys[1])
	require.NoError(t, err)
	parsedIntermediate, err := helpers.ParseCertificatePEM(cautils.ECDSACertChain[1])
	require.NoError(t, err)
	fauxRootDER, err := x509.CreateCertificate(cryptorand.Reader, parsedIntermediate, parsedIntermediate, parsedKey.Public(), parsedKey)
	require.NoError(t, err)
	fauxRootCert := pem.EncodeToMemory(&pem.Block{
		Type:  "CERTIFICATE",
		Bytes: fauxRootDER,
	})

	// It is not required, but not wrong, for the intermediate chain to terminate with a self-signed root
	signWithIntermediate, err := ca.NewRootCA(cautils.ECDSACertChain[2], cautils.ECDSACertChain[1], cautils.ECDSACertChainKeys[1],
		ca.DefaultNodeCertExpiration, append(cautils.ECDSACertChain[1], cautils.ECDSACertChain[2]...))
	require.NoError(t, err)

	// just the intermediate, without a terminating self-signed root, is also ok
	signWithIntermediate, err = ca.NewRootCA(cautils.ECDSACertChain[2], cautils.ECDSACertChain[1], cautils.ECDSACertChainKeys[1],
		ca.DefaultNodeCertExpiration, cautils.ECDSACertChain[1])
	require.NoError(t, err)

	paths := ca.NewConfigPaths(tempdir)
	krw := ca.NewKeyReadWriter(paths.Node, nil, nil)
	_, _, err = signWithIntermediate.IssueAndSaveNewCertificates(krw, "cn", "ou", "org")
	require.NoError(t, err)
	tlsCert, _, err := krw.Read()
	require.NoError(t, err)

	parsedCerts, chains, err := ca.ValidateCertChain(signWithIntermediate.Pool, tlsCert, false)
	require.NoError(t, err)
	require.Len(t, parsedCerts, 2)
	require.Len(t, chains, 1)
	require.Equal(t, parsedIntermediate.Raw, parsedCerts[1].Raw)
	require.Equal(t, parsedCerts, chains[0][:len(chains[0])-1]) // the last one is the root

	oldRoot, err := ca.NewRootCA(cautils.ECDSACertChain[2], cautils.ECDSACertChain[2], cautils.ECDSACertChainKeys[2], ca.DefaultNodeCertExpiration, nil)
	require.NoError(t, err)

	newRoot, err := ca.NewRootCA(fauxRootCert, fauxRootCert, cautils.ECDSACertChainKeys[1], ca.DefaultNodeCertExpiration, nil)
	require.NoError(t, err)
	apiNewRoot := api.RootCA{
		CACert: fauxRootCert,
		CAKey:  cautils.ECDSACertChainKeys[1],
	}

	checkValidateAgainstAllRoots := func(cert []byte) {
		for i, root := range []ca.RootCA{signWithIntermediate, oldRoot, newRoot} {
			parsedCerts, chains, err := ca.ValidateCertChain(root.Pool, cert, false)
			require.NoError(t, err)
			require.Len(t, parsedCerts, 2)
			require.Len(t, chains, 1)
			require.True(t, len(chains[0]) >= 2) // there are always at least 2 certs at minimum: the leaf and the root
			require.Equal(t, parsedCerts[0], chains[0][0])
			require.Equal(t, parsedIntermediate.Raw, parsedCerts[1].Raw)

			chainWithoutRoot := chains[0][:len(chains[0])-1]
			if i == 2 {
				// against the new root, the cert can chain directly up to the root without the intermediate
				require.Equal(t, parsedCerts[0:1], chainWithoutRoot)
			} else {
				require.Equal(t, parsedCerts, chainWithoutRoot)
			}
		}
	}
	checkValidateAgainstAllRoots(tlsCert)

	if !cautils.External {
		return
	}

	// create an external signing server that generates leaf certs with the new root (but does not append the intermediate)
	tc := cautils.NewTestCAFromAPIRootCA(t, tempdir, apiNewRoot, nil)
	defer tc.Stop()

	// we need creds that trust both the old and new root in order to connect to the test CA, and we want this root CA to
	// append certificates
	connectToExternalRootCA, err := ca.NewRootCA(append(cautils.ECDSACertChain[2], fauxRootCert...), cautils.ECDSACertChain[1],
		cautils.ECDSACertChainKeys[1], ca.DefaultNodeCertExpiration, cautils.ECDSACertChain[1])
	require.NoError(t, err)
	tlsKeyPair, _, err := connectToExternalRootCA.IssueAndSaveNewCertificates(krw, "cn", ca.ManagerRole, tc.Organization)
	require.NoError(t, err)
	externalCA := ca.NewExternalCA(cautils.ECDSACertChain[1],
		ca.NewExternalCATLSConfig([]tls.Certificate{*tlsKeyPair}, connectToExternalRootCA.Pool), tc.ExternalSigningServer.URL)

	newCSR, _, err := ca.GenerateNewCSR()
	require.NoError(t, err)

	tlsCert, err = externalCA.Sign(tc.Context, ca.PrepareCSR(newCSR, "cn", ca.ManagerRole, tc.Organization))
	require.NoError(t, err)

	checkValidateAgainstAllRoots(tlsCert)
}

type certTestCase struct {
	cert        []byte
	errorStr    string
	root        []byte
	allowExpiry bool
}

func TestValidateCertificateChain(t *testing.T) {
	leaf, intermediate, root := cautils.ECDSACertChain[0], cautils.ECDSACertChain[1], cautils.ECDSACertChain[2]
	intermediateKey, rootKey := cautils.ECDSACertChainKeys[1], cautils.ECDSACertChainKeys[2] // we don't care about the leaf key

	chain := func(certs ...[]byte) []byte {
		var all []byte
		for _, cert := range certs {
			all = append(all, cert...)
		}
		return all
	}

	now := time.Now()
	expiredLeaf := cautils.ReDateCert(t, leaf, intermediate, intermediateKey, now.Add(-10*time.Hour), now.Add(-1*time.Minute))
	expiredIntermediate := cautils.ReDateCert(t, intermediate, root, rootKey, now.Add(-10*time.Hour), now.Add(-1*time.Minute))
	notYetValidLeaf := cautils.ReDateCert(t, leaf, intermediate, intermediateKey, now.Add(time.Hour), now.Add(2*time.Hour))
	notYetValidIntermediate := cautils.ReDateCert(t, intermediate, root, rootKey, now.Add(time.Hour), now.Add(2*time.Hour))

	rootPool := x509.NewCertPool()
	rootPool.AppendCertsFromPEM(root)

	invalids := []certTestCase{
		{
			cert:     nil,
			root:     root,
			errorStr: "no certificates to validate",
		},
		{
			cert:     []byte("malformed"),
			root:     root,
			errorStr: "Failed to decode certificate",
		},
		{
			cert:     chain(leaf, intermediate, leaf),
			root:     root,
			errorStr: "certificates do not form a chain",
		},
		{
			cert:     chain(leaf, intermediate),
			root:     cautils.ECDSA256SHA256Cert,
			errorStr: "unknown authority",
		},
		{
			cert:     chain(expiredLeaf, intermediate),
			root:     root,
			errorStr: "not valid after",
		},
		{
			cert:     chain(leaf, expiredIntermediate),
			root:     root,
			errorStr: "not valid after",
		},
		{
			cert:     chain(notYetValidLeaf, intermediate),
			root:     root,
			errorStr: "not valid before",
		},
		{
			cert:     chain(leaf, notYetValidIntermediate),
			root:     root,
			errorStr: "not valid before",
		},

		// if we allow expiry, we still don't allow not yet valid certs or expired certs that don't chain up to the root
		{
			cert:        chain(notYetValidLeaf, intermediate),
			root:        root,
			allowExpiry: true,
			errorStr:    "not valid before",
		},
		{
			cert:        chain(leaf, notYetValidIntermediate),
			root:        root,
			allowExpiry: true,
			errorStr:    "not valid before",
		},
		{
			cert:        chain(expiredLeaf, intermediate),
			root:        cautils.ECDSA256SHA256Cert,
			allowExpiry: true,
			errorStr:    "unknown authority",
		},

		// construct a weird cases where one cert is expired, we allow expiry, but the other cert is not yet valid at the first cert's expiry
		// (this is not something that can happen unless we allow expiry, because if the cert periods don't overlap, one or the other will
		// be either not yet valid or already expired)
		{
			cert: chain(
				cautils.ReDateCert(t, leaf, intermediate, intermediateKey, now.Add(-3*helpers.OneDay), now.Add(-2*helpers.OneDay)),
				cautils.ReDateCert(t, intermediate, root, rootKey, now.Add(-1*helpers.OneDay), now.Add(helpers.OneDay))),
			root:        root,
			allowExpiry: true,
			errorStr:    "there is no time span",
		},
		// similarly, but for root pool
		{
			cert:        chain(expiredLeaf, expiredIntermediate),
			root:        cautils.ReDateCert(t, root, root, rootKey, now.Add(-3*helpers.OneYear), now.Add(-2*helpers.OneYear)),
			allowExpiry: true,
			errorStr:    "there is no time span",
		},
	}

	for _, invalid := range invalids {
		pool := x509.NewCertPool()
		pool.AppendCertsFromPEM(invalid.root)
		_, _, err := ca.ValidateCertChain(pool, invalid.cert, invalid.allowExpiry)
		require.Error(t, err, invalid.errorStr)
		require.Contains(t, err.Error(), invalid.errorStr)
	}

	// these will default to using the root pool, so we don't have to specify the root pool
	valids := []certTestCase{
		{cert: chain(leaf, intermediate, root)},
		{cert: chain(leaf, intermediate)},
		{cert: intermediate},
		{
			cert:        chain(expiredLeaf, intermediate),
			allowExpiry: true,
		},
		{
			cert:        chain(leaf, expiredIntermediate),
			allowExpiry: true,
		},
		{
			cert:        chain(expiredLeaf, expiredIntermediate),
			allowExpiry: true,
		},
	}

	for _, valid := range valids {
		parsedCerts, chains, err := ca.ValidateCertChain(rootPool, valid.cert, valid.allowExpiry)
		require.NoError(t, err)
		require.NotEmpty(t, chain)
		for _, chain := range chains {
			require.Equal(t, parsedCerts[0], chain[0]) // the leaf certs are equal
			require.True(t, len(chain) >= 2)
		}
	}
}

// Tests cross-signing an RSA cert with an ECDSA cert and vice versa, and an ECDSA
// cert with another ECDSA cert and a RSA cert with another RSA cert
func TestRootCACrossSignCACertificate(t *testing.T) {
	t.Parallel()
	if cautils.External {
		return
	}

	oldCAs := []struct {
		cert, key []byte
	}{
		{
			cert: cautils.ECDSA256SHA256Cert,
			key:  cautils.ECDSA256Key,
		},
		{
			cert: cautils.RSA2048SHA256Cert,
			key:  cautils.RSA2048Key,
		},
	}

	cert1, key1, err := cautils.CreateRootCertAndKey("rootCNECDSA")
	require.NoError(t, err)

	rsaReq := cfcsr.CertificateRequest{
		CN: "rootCNRSA",
		KeyRequest: &cfcsr.BasicKeyRequest{
			A: "rsa",
			S: 2048,
		},
		CA: &cfcsr.CAConfig{Expiry: ca.RootCAExpiration},
	}

	// Generate the CA and get the certificate and private key
	cert2, _, key2, err := initca.New(&rsaReq)
	require.NoError(t, err)

	newCAs := []struct {
		cert, key []byte
	}{
		{
			cert: cert1,
			key:  key1,
		},
		{
			cert: cert2,
			key:  key2,
		},
	}

	tempdir, err := ioutil.TempDir("", "cross-sign-cert")
	require.NoError(t, err)
	defer os.RemoveAll(tempdir)
	paths := ca.NewConfigPaths(tempdir)
	krw := ca.NewKeyReadWriter(paths.Node, nil, nil)

	for _, oldRoot := range oldCAs {
		for _, newRoot := range newCAs {
			rootCA1, err := ca.NewRootCA(oldRoot.cert, oldRoot.cert, oldRoot.key, ca.DefaultNodeCertExpiration, nil)
			require.NoError(t, err)

			rootCA2, err := ca.NewRootCA(newRoot.cert, newRoot.cert, newRoot.key, ca.DefaultNodeCertExpiration, nil)
			require.NoError(t, err)

			_, _, err = rootCA2.IssueAndSaveNewCertificates(krw, "cn", "ou", "org")
			require.NoError(t, err)
			certBytes, keyBytes, err := krw.Read()
			require.NoError(t, err)
			leafCert, err := helpers.ParseCertificatePEM(certBytes)
			require.NoError(t, err)

			// cross-signing a non-CA fails
			_, err = rootCA1.CrossSignCACertificate(certBytes)
			require.Error(t, err)

			// cross-signing some non-cert PEM bytes fail
			_, err = rootCA1.CrossSignCACertificate(keyBytes)
			require.Error(t, err)

			intermediate, err := rootCA1.CrossSignCACertificate(newRoot.cert)
			require.NoError(t, err)
			parsedIntermediate, err := helpers.ParseCertificatePEM(intermediate)
			require.NoError(t, err)
			parsedRoot2, err := helpers.ParseCertificatePEM(newRoot.cert)
			require.NoError(t, err)
			require.Equal(t, parsedRoot2.RawSubject, parsedIntermediate.RawSubject)
			require.Equal(t, parsedRoot2.RawSubjectPublicKeyInfo, parsedIntermediate.RawSubjectPublicKeyInfo)
			require.True(t, parsedIntermediate.IsCA)

			intermediatePool := x509.NewCertPool()
			intermediatePool.AddCert(parsedIntermediate)

			// we can validate a chain from the leaf to the first root through the intermediate,
			// or from the leaf cert to the second root with or without the intermediate
			_, err = leafCert.Verify(x509.VerifyOptions{Roots: rootCA1.Pool})
			require.Error(t, err)
			_, err = leafCert.Verify(x509.VerifyOptions{Roots: rootCA1.Pool, Intermediates: intermediatePool})
			require.NoError(t, err)

			_, err = leafCert.Verify(x509.VerifyOptions{Roots: rootCA2.Pool})
			require.NoError(t, err)
			_, err = leafCert.Verify(x509.VerifyOptions{Roots: rootCA2.Pool, Intermediates: intermediatePool})
			require.NoError(t, err)
		}
	}
}

func concat(byteSlices ...[]byte) []byte {
	var results []byte
	for _, slice := range byteSlices {
		results = append(results, slice...)
	}
	return results
}

func TestNormalizePEMs(t *testing.T) {
	pemBlock, _ := pem.Decode(cautils.ECDSA256SHA256Cert)
	pemBlock.Headers = map[string]string{
		"hello": "world",
	}
	withHeaders := pem.EncodeToMemory(pemBlock)
	for _, testcase := range []struct{ input, expect []byte }{
		{
			input:  nil,
			expect: nil,
		},
		{
			input:  []byte("garbage"),
			expect: nil,
		},
		{
			input:  concat([]byte("garbage\n\t\n\n"), cautils.ECDSA256SHA256Cert, []byte("   \n")),
			expect: ca.NormalizePEMs(cautils.ECDSA256SHA256Cert),
		},
		{
			input:  concat([]byte("\n\t\n     "), withHeaders, []byte("\t\n\n"), cautils.ECDSACertChain[0]),
			expect: ca.NormalizePEMs(append(cautils.ECDSA256SHA256Cert, cautils.ECDSACertChain[0]...)),
		},
	} {
		require.Equal(t, testcase.expect, ca.NormalizePEMs(testcase.input))
	}
}
