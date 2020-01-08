package ca_test

import (
	"bytes"
	"crypto/tls"
	"crypto/x509"
	"fmt"
	"io/ioutil"
	"net"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"

	"golang.org/x/net/context"

	cfconfig "github.com/cloudflare/cfssl/config"
	"github.com/cloudflare/cfssl/helpers"
	"github.com/docker/swarmkit/api"
	"github.com/docker/swarmkit/ca"
	cautils "github.com/docker/swarmkit/ca/testutils"
	"github.com/docker/swarmkit/log"
	"github.com/docker/swarmkit/manager/state"
	"github.com/docker/swarmkit/manager/state/store"
	"github.com/docker/swarmkit/testutils"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestDownloadRootCASuccess(t *testing.T) {
	for _, fips := range []bool{true, false} {
		testDownloadRootCASuccess(t, fips)
	}
}
func testDownloadRootCASuccess(t *testing.T, fips bool) {
	var tc *cautils.TestCA
	if fips {
		tc = cautils.NewFIPSTestCA(t)
	} else {
		tc = cautils.NewTestCA(t)
	}
	defer tc.Stop()

	token := ca.GenerateJoinToken(&tc.RootCA, fips)

	// if we require mandatory FIPS, the join token uses a new format.  otherwise
	// the join token should use the old format.
	prefix := "SWMTKN-1-"
	if fips {
		prefix = "SWMTKN-2-1-"
	}
	require.True(t, strings.HasPrefix(token, prefix))

	// Remove the CA cert
	os.RemoveAll(tc.Paths.RootCA.Cert)

	rootCA, err := ca.DownloadRootCA(tc.Context, tc.Paths.RootCA, token, tc.ConnBroker)
	require.NoError(t, err)
	require.NotNil(t, rootCA.Pool)
	require.NotNil(t, rootCA.Certs)
	_, err = rootCA.Signer()
	require.Equal(t, err, ca.ErrNoValidSigner)
	require.Equal(t, tc.RootCA.Certs, rootCA.Certs)

	// Remove the CA cert
	os.RemoveAll(tc.Paths.RootCA.Cert)

	// downloading without a join token also succeeds
	rootCA, err = ca.DownloadRootCA(tc.Context, tc.Paths.RootCA, "", tc.ConnBroker)
	require.NoError(t, err)
	require.NotNil(t, rootCA.Pool)
	require.NotNil(t, rootCA.Certs)
	_, err = rootCA.Signer()
	require.Equal(t, err, ca.ErrNoValidSigner)
	require.Equal(t, tc.RootCA.Certs, rootCA.Certs)
}

func TestDownloadRootCAWrongCAHash(t *testing.T) {
	tc := cautils.NewTestCA(t)
	defer tc.Stop()

	// Remove the CA cert
	os.RemoveAll(tc.Paths.RootCA.Cert)

	// invalid token
	for _, invalid := range []string{
		"invalidtoken", // completely invalid
		"SWMTKN-1-3wkodtpeoipd1u1hi0ykdcdwhw16dk73ulqqtn14b3indz68rf-4myj5xihyto11dg1cn55w8p6",  // mistyped
		"SWMTKN-2-1fhvpatk6ms36i3uc64tsv1ybyuxkb899zbjpq4ib64qwbibz4-1g3as27iwmko5yqh1byv868hx", // version 2 should have 5 tokens
		"SWMTKN-0-1fhvpatk6ms36i3uc64tsv1ybyuxkb899zbjpq4ib64qwbibz4-1g3as27iwmko5yqh1byv868hx", // invalid version
	} {
		_, err := ca.DownloadRootCA(tc.Context, tc.Paths.RootCA, invalid, tc.ConnBroker)
		require.Error(t, err)
		require.Contains(t, err.Error(), "invalid join token")
	}

	// invalid hash token - can get the wrong hash from both version 1 and version 2
	for _, wrongToken := range []string{
		"SWMTKN-1-1kxftv4ofnc6mt30lmgipg6ngf9luhwqopfk1tz6bdmnkubg0e-4myj5xihyto11dg1cn55w8p61",
		"SWMTKN-2-0-1kxftv4ofnc6mt30lmgipg6ngf9luhwqopfk1tz6bdmnkubg0e-4myj5xihyto11dg1cn55w8p61",
	} {
		_, err := ca.DownloadRootCA(tc.Context, tc.Paths.RootCA, wrongToken, tc.ConnBroker)
		require.Error(t, err)
		require.Contains(t, err.Error(), "remote CA does not match fingerprint.")
	}
}

func TestCreateSecurityConfigEmptyDir(t *testing.T) {
	if cautils.External {
		return // this doesn't require any servers at all
	}
	tc := cautils.NewTestCA(t)
	defer tc.Stop()
	assert.NoError(t, tc.CAServer.Stop())

	// Remove all the contents from the temp dir and try again with a new node
	for _, org := range []string{
		"",
		"my_org",
	} {
		os.RemoveAll(tc.TempDir)
		krw := ca.NewKeyReadWriter(tc.Paths.Node, nil, nil)
		nodeConfig, cancel, err := tc.RootCA.CreateSecurityConfig(tc.Context, krw,
			ca.CertificateRequestConfig{
				Token:        tc.WorkerToken,
				ConnBroker:   tc.ConnBroker,
				Organization: org,
			})
		assert.NoError(t, err)
		cancel()
		assert.NotNil(t, nodeConfig)
		assert.NotNil(t, nodeConfig.ClientTLSCreds)
		assert.NotNil(t, nodeConfig.ServerTLSCreds)
		assert.Equal(t, tc.RootCA, *nodeConfig.RootCA())
		if org != "" {
			assert.Equal(t, org, nodeConfig.ClientTLSCreds.Organization())
		}

		root, err := helpers.ParseCertificatePEM(tc.RootCA.Certs)
		assert.NoError(t, err)

		issuerInfo := nodeConfig.IssuerInfo()
		assert.NotNil(t, issuerInfo)
		assert.Equal(t, root.RawSubjectPublicKeyInfo, issuerInfo.PublicKey)
		assert.Equal(t, root.RawSubject, issuerInfo.Subject)
	}
}

func TestCreateSecurityConfigNoCerts(t *testing.T) {
	tc := cautils.NewTestCA(t)
	defer tc.Stop()

	krw := ca.NewKeyReadWriter(tc.Paths.Node, nil, nil)
	root, err := helpers.ParseCertificatePEM(tc.RootCA.Certs)
	assert.NoError(t, err)

	validateNodeConfig := func(rootCA *ca.RootCA) {
		nodeConfig, cancel, err := rootCA.CreateSecurityConfig(tc.Context, krw,
			ca.CertificateRequestConfig{
				Token:      tc.WorkerToken,
				ConnBroker: tc.ConnBroker,
			})
		assert.NoError(t, err)
		cancel()
		assert.NotNil(t, nodeConfig)
		assert.NotNil(t, nodeConfig.ClientTLSCreds)
		assert.NotNil(t, nodeConfig.ServerTLSCreds)
		// tc.RootCA can maybe sign, and the node root CA can also maybe sign, so we want to just compare the root
		// certs and intermediates
		assert.Equal(t, tc.RootCA.Certs, nodeConfig.RootCA().Certs)
		assert.Equal(t, tc.RootCA.Intermediates, nodeConfig.RootCA().Intermediates)

		issuerInfo := nodeConfig.IssuerInfo()
		assert.NotNil(t, issuerInfo)
		assert.Equal(t, root.RawSubjectPublicKeyInfo, issuerInfo.PublicKey)
		assert.Equal(t, root.RawSubject, issuerInfo.Subject)
	}

	// Remove only the node certificates form the directory, and attest that we get
	// new certificates that are locally signed
	os.RemoveAll(tc.Paths.Node.Cert)
	validateNodeConfig(&tc.RootCA)

	// Remove only the node certificates form the directory, get a new rootCA, and attest that we get
	// new certificates that are issued by the remote CA
	os.RemoveAll(tc.Paths.Node.Cert)
	rootCA, err := ca.GetLocalRootCA(tc.Paths.RootCA)
	assert.NoError(t, err)
	validateNodeConfig(&rootCA)
}

func testGRPCConnection(t *testing.T, secConfig *ca.SecurityConfig) {
	// set up a GRPC server using these credentials
	secConfig.ServerTLSCreds.Config().ClientAuth = tls.RequireAndVerifyClientCert
	l, err := net.Listen("tcp", "127.0.0.1:0")
	require.NoError(t, err)
	serverOpts := []grpc.ServerOption{grpc.Creds(secConfig.ServerTLSCreds)}
	grpcServer := grpc.NewServer(serverOpts...)
	go grpcServer.Serve(l)
	defer grpcServer.Stop()

	// we should be able to connect to the server using the client credentials
	dialOpts := []grpc.DialOption{
		grpc.WithBlock(),
		grpc.WithTimeout(10 * time.Second),
		grpc.WithTransportCredentials(secConfig.ClientTLSCreds),
	}
	conn, err := grpc.Dial(l.Addr().String(), dialOpts...)
	require.NoError(t, err)
	conn.Close()
}

func TestLoadSecurityConfigExpiredCert(t *testing.T) {
	if cautils.External {
		return // this doesn't require any servers at all
	}
	tc := cautils.NewTestCA(t)
	defer tc.Stop()
	s, err := tc.RootCA.Signer()
	require.NoError(t, err)

	krw := ca.NewKeyReadWriter(tc.Paths.Node, nil, nil)
	now := time.Now()

	_, _, err = tc.RootCA.IssueAndSaveNewCertificates(krw, "cn", "ou", "org")
	require.NoError(t, err)
	certBytes, _, err := krw.Read()
	require.NoError(t, err)

	// A cert that is not yet valid is not valid even if expiry is allowed
	invalidCert := cautils.ReDateCert(t, certBytes, tc.RootCA.Certs, s.Key, now.Add(time.Hour), now.Add(time.Hour*2))
	require.NoError(t, ioutil.WriteFile(tc.Paths.Node.Cert, invalidCert, 0700))

	_, _, err = ca.LoadSecurityConfig(tc.Context, tc.RootCA, krw, false)
	require.Error(t, err)
	require.IsType(t, x509.CertificateInvalidError{}, errors.Cause(err))

	_, _, err = ca.LoadSecurityConfig(tc.Context, tc.RootCA, krw, true)
	require.Error(t, err)
	require.IsType(t, x509.CertificateInvalidError{}, errors.Cause(err))

	// a cert that is expired is not valid if expiry is not allowed
	invalidCert = cautils.ReDateCert(t, certBytes, tc.RootCA.Certs, s.Key, now.Add(-2*time.Minute), now.Add(-1*time.Minute))
	require.NoError(t, ioutil.WriteFile(tc.Paths.Node.Cert, invalidCert, 0700))

	_, _, err = ca.LoadSecurityConfig(tc.Context, tc.RootCA, krw, false)
	require.Error(t, err)
	require.IsType(t, x509.CertificateInvalidError{}, errors.Cause(err))

	// but it is valid if expiry is allowed
	_, cancel, err := ca.LoadSecurityConfig(tc.Context, tc.RootCA, krw, true)
	require.NoError(t, err)
	cancel()
}

func TestLoadSecurityConfigInvalidCert(t *testing.T) {
	if cautils.External {
		return // this doesn't require any servers at all
	}
	tc := cautils.NewTestCA(t)
	defer tc.Stop()

	// Write some garbage to the cert
	ioutil.WriteFile(tc.Paths.Node.Cert, []byte(`-----BEGIN CERTIFICATE-----\n
some random garbage\n
-----END CERTIFICATE-----`), 0644)

	krw := ca.NewKeyReadWriter(tc.Paths.Node, nil, nil)

	_, _, err := ca.LoadSecurityConfig(tc.Context, tc.RootCA, krw, false)
	assert.Error(t, err)
}

func TestLoadSecurityConfigInvalidKey(t *testing.T) {
	if cautils.External {
		return // this doesn't require any servers at all
	}
	tc := cautils.NewTestCA(t)
	defer tc.Stop()

	// Write some garbage to the Key
	ioutil.WriteFile(tc.Paths.Node.Key, []byte(`-----BEGIN PRIVATE KEY-----\n
some random garbage\n
-----END PRIVATE KEY-----`), 0644)

	krw := ca.NewKeyReadWriter(tc.Paths.Node, nil, nil)

	_, _, err := ca.LoadSecurityConfig(tc.Context, tc.RootCA, krw, false)
	assert.Error(t, err)
}

func TestLoadSecurityConfigIncorrectPassphrase(t *testing.T) {
	if cautils.External {
		return // this doesn't require any servers at all
	}
	tc := cautils.NewTestCA(t)
	defer tc.Stop()

	paths := ca.NewConfigPaths(tc.TempDir)
	_, _, err := tc.RootCA.IssueAndSaveNewCertificates(ca.NewKeyReadWriter(paths.Node, []byte("kek"), nil),
		"nodeID", ca.WorkerRole, tc.Organization)
	require.NoError(t, err)

	_, _, err = ca.LoadSecurityConfig(tc.Context, tc.RootCA, ca.NewKeyReadWriter(paths.Node, nil, nil), false)
	require.IsType(t, ca.ErrInvalidKEK{}, err)
}

func TestLoadSecurityConfigIntermediates(t *testing.T) {
	if cautils.External {
		return // this doesn't require any servers at all
	}
	tempdir, err := ioutil.TempDir("", "test-load-config-with-intermediates")
	require.NoError(t, err)
	defer os.RemoveAll(tempdir)
	paths := ca.NewConfigPaths(tempdir)
	krw := ca.NewKeyReadWriter(paths.Node, nil, nil)

	rootCA, err := ca.NewRootCA(cautils.ECDSACertChain[2], nil, nil, ca.DefaultNodeCertExpiration, nil)
	require.NoError(t, err)

	ctx := log.WithLogger(context.Background(), log.L.WithFields(logrus.Fields{
		"testname":          t.Name(),
		"testHasExternalCA": false,
	}))

	// loading the incomplete chain fails
	require.NoError(t, krw.Write(cautils.ECDSACertChain[0], cautils.ECDSACertChainKeys[0], nil))
	_, _, err = ca.LoadSecurityConfig(ctx, rootCA, krw, false)
	require.Error(t, err)

	intermediate, err := helpers.ParseCertificatePEM(cautils.ECDSACertChain[1])
	require.NoError(t, err)

	// loading the complete chain succeeds
	require.NoError(t, krw.Write(append(cautils.ECDSACertChain[0], cautils.ECDSACertChain[1]...), cautils.ECDSACertChainKeys[0], nil))
	secConfig, cancel, err := ca.LoadSecurityConfig(ctx, rootCA, krw, false)
	require.NoError(t, err)
	defer cancel()
	require.NotNil(t, secConfig)
	issuerInfo := secConfig.IssuerInfo()
	require.NotNil(t, issuerInfo)
	require.Equal(t, intermediate.RawSubjectPublicKeyInfo, issuerInfo.PublicKey)
	require.Equal(t, intermediate.RawSubject, issuerInfo.Subject)

	testGRPCConnection(t, secConfig)
}

func TestLoadSecurityConfigKeyFormat(t *testing.T) {
	if cautils.External {
		return // this doesn't require any servers at all
	}
	tempdir, err := ioutil.TempDir("", "test-load-config")
	require.NoError(t, err)
	defer os.RemoveAll(tempdir)
	paths := ca.NewConfigPaths(tempdir)
	krw := ca.NewKeyReadWriter(paths.Node, nil, nil)

	rootCA, err := ca.NewRootCA(cautils.ECDSACertChain[1], nil, nil, ca.DefaultNodeCertExpiration, nil)
	require.NoError(t, err)

	ctx := log.WithLogger(context.Background(), log.L.WithFields(logrus.Fields{
		"testname":          t.Name(),
		"testHasExternalCA": false,
	}))

	// load leaf cert with its PKCS#1 format key
	require.NoError(t, krw.Write(cautils.ECDSACertChain[0], cautils.ECDSACertChainKeys[0], nil))
	secConfig, cancel, err := ca.LoadSecurityConfig(ctx, rootCA, krw, false)
	require.NoError(t, err)
	defer cancel()
	require.NotNil(t, secConfig)

	testGRPCConnection(t, secConfig)

	// load leaf cert with its PKCS#8 format key
	require.NoError(t, krw.Write(cautils.ECDSACertChain[0], cautils.ECDSACertChainPKCS8Keys[0], nil))
	secConfig, cancel, err = ca.LoadSecurityConfig(ctx, rootCA, krw, false)
	require.NoError(t, err)
	defer cancel()
	require.NotNil(t, secConfig)

	testGRPCConnection(t, secConfig)
}

// Custom GRPC dialer that does the TLS handshake itself, so that we can grab whatever
// TLS error comes out.  Otherwise, GRPC >=1.10.x attempts to load balance connections and dial
// asynchronously, thus eating whatever connection errors there are and returning nothing
// but a timeout error.  In theory, we can dial without the `WithBlock` option, and check
// the error from an RPC call instead, but that's racy: https://github.com/grpc/grpc-go/issues/1917
// Hopefully an API will be provided to check connection errors on the underlying connection:
// https://github.com/grpc/grpc-go/issues/2031.
func tlsGRPCDial(ctx context.Context, address string, creds credentials.TransportCredentials) (*grpc.ClientConn, chan error, error) {
	dialerErrChan := make(chan error, 1)
	conn, err := grpc.Dial(
		address,
		grpc.WithBlock(),
		grpc.WithTimeout(10*time.Second),
		grpc.WithInsecure(),
		grpc.WithDialer(func(address string, timeout time.Duration) (net.Conn, error) {
			ctx, cancel := context.WithTimeout(ctx, timeout)
			defer cancel()

			conn, err := (&net.Dialer{Cancel: ctx.Done()}).Dial("tcp", address)
			if err != nil {
				dialerErrChan <- err
				return nil, err
			}
			conn, _, err = creds.ClientHandshake(ctx, address, conn)
			if err != nil {
				dialerErrChan <- err
				return nil, err
			}
			return conn, nil
		}),
	)
	return conn, dialerErrChan, err
}

// When the root CA is updated on the security config, the root pools are updated
func TestSecurityConfigUpdateRootCA(t *testing.T) {
	t.Parallel()
	if cautils.External { // don't need an external CA server
		return
	}

	tc := cautils.NewTestCA(t)
	defer tc.Stop()
	tcConfig, err := tc.NewNodeConfig("worker")
	require.NoError(t, err)

	// create the "original" security config, and we'll update it to trust the test server's
	cert, key, err := cautils.CreateRootCertAndKey("root1")
	require.NoError(t, err)

	rootCA, err := ca.NewRootCA(cert, cert, key, ca.DefaultNodeCertExpiration, nil)
	require.NoError(t, err)

	tempdir, err := ioutil.TempDir("", "test-security-config-update")
	require.NoError(t, err)
	defer os.RemoveAll(tempdir)
	configPaths := ca.NewConfigPaths(tempdir)

	secConfig, cancel, err := rootCA.CreateSecurityConfig(tc.Context,
		ca.NewKeyReadWriter(configPaths.Node, nil, nil), ca.CertificateRequestConfig{})
	require.NoError(t, err)
	cancel()
	// update the server TLS to require certificates, otherwise this will all pass
	// even if the root pools aren't updated
	secConfig.ServerTLSCreds.Config().ClientAuth = tls.RequireAndVerifyClientCert

	// set up a GRPC server using these credentials
	l, err := net.Listen("tcp", "127.0.0.1:0")
	require.NoError(t, err)
	serverOpts := []grpc.ServerOption{grpc.Creds(secConfig.ServerTLSCreds)}
	grpcServer := grpc.NewServer(serverOpts...)
	go grpcServer.Serve(l)
	defer grpcServer.Stop()

	// We should not be able to connect to the test CA server using the original security config, and should not
	// be able to connect to new server using the test CA's client credentials.  We need to use our own
	// dialer, so that grpc does not attempt to load balance/retry the connection - this way the x509 errors can be
	// surfaced.
	_, actualErrChan, err := tlsGRPCDial(tc.Context, tc.Addr, secConfig.ClientTLSCreds)
	defer close(actualErrChan)
	require.Error(t, err)
	err = <-actualErrChan
	require.Error(t, err)
	require.IsType(t, x509.UnknownAuthorityError{}, err)

	_, actualErrChan, err = tlsGRPCDial(tc.Context, l.Addr().String(), tcConfig.ClientTLSCreds)
	defer close(actualErrChan)
	require.Error(t, err)
	err = <-actualErrChan
	require.Error(t, err)
	require.IsType(t, x509.UnknownAuthorityError{}, err)

	// update the root CA on the "original security config to support both the old root
	// and the "new root" (the testing CA root).  Also make sure this root CA has an
	// intermediate; we won't use it for anything, just make sure that newly generated TLS
	// certs have the intermediate appended.
	someOtherRootCA, err := ca.CreateRootCA("someOtherRootCA")
	require.NoError(t, err)
	intermediate, err := someOtherRootCA.CrossSignCACertificate(cert)
	require.NoError(t, err)
	rSigner, err := rootCA.Signer()
	require.NoError(t, err)
	updatedRootCA, err := ca.NewRootCA(concat(rootCA.Certs, tc.RootCA.Certs, someOtherRootCA.Certs), rSigner.Cert, rSigner.Key, ca.DefaultNodeCertExpiration, intermediate)
	require.NoError(t, err)
	err = secConfig.UpdateRootCA(&updatedRootCA)
	require.NoError(t, err)

	// can now connect to the test CA using our modified security config, and can cannect to our server using
	// the test CA config
	conn, err := grpc.Dial(
		tc.Addr,
		grpc.WithBlock(),
		grpc.WithTimeout(10*time.Second),
		grpc.WithTransportCredentials(tcConfig.ClientTLSCreds),
	)
	require.NoError(t, err)
	conn.Close()

	conn, err = grpc.Dial(
		tc.Addr,
		grpc.WithBlock(),
		grpc.WithTimeout(10*time.Second),
		grpc.WithTransportCredentials(secConfig.ClientTLSCreds),
	)
	require.NoError(t, err)
	conn.Close()

	// make sure any generated certs after updating contain the intermediate
	krw := ca.NewKeyReadWriter(configPaths.Node, nil, nil)
	_, _, err = secConfig.RootCA().IssueAndSaveNewCertificates(krw, "cn", "ou", "org")
	require.NoError(t, err)
	generatedCert, _, err := krw.Read()
	require.NoError(t, err)

	parsedCerts, err := helpers.ParseCertificatesPEM(generatedCert)
	require.NoError(t, err)
	require.Len(t, parsedCerts, 2)
	parsedIntermediate, err := helpers.ParseCertificatePEM(intermediate)
	require.NoError(t, err)
	require.Equal(t, parsedIntermediate, parsedCerts[1])
}

// You can't update the root CA to one that doesn't match the TLS certificates
func TestSecurityConfigUpdateRootCAUpdateConsistentWithTLSCertificates(t *testing.T) {
	t.Parallel()
	if cautils.External {
		return // we don't care about external CAs at all
	}
	tempdir, err := ioutil.TempDir("", "")
	require.NoError(t, err)
	krw := ca.NewKeyReadWriter(ca.NewConfigPaths(tempdir).Node, nil, nil)

	rootCA, err := ca.CreateRootCA("rootcn")
	require.NoError(t, err)
	tlsKeyPair, issuerInfo, err := rootCA.IssueAndSaveNewCertificates(krw, "cn", "ou", "org")
	require.NoError(t, err)

	otherRootCA, err := ca.CreateRootCA("otherCN")
	require.NoError(t, err)
	_, otherIssuerInfo, err := otherRootCA.IssueAndSaveNewCertificates(krw, "cn", "ou", "org")
	require.NoError(t, err)
	intermediate, err := rootCA.CrossSignCACertificate(otherRootCA.Certs)
	require.NoError(t, err)
	otherTLSCert, otherTLSKey, err := krw.Read()
	require.NoError(t, err)
	otherTLSKeyPair, err := tls.X509KeyPair(append(otherTLSCert, intermediate...), otherTLSKey)
	require.NoError(t, err)

	// Note - the validation only happens on UpdateRootCA for now, because the assumption is
	// that something else does the validation when loading the security config for the first
	// time and when getting new TLS credentials

	secConfig, cancel, err := ca.NewSecurityConfig(&rootCA, krw, tlsKeyPair, issuerInfo)
	require.NoError(t, err)
	cancel()

	// can't update the root CA to one that doesn't match the tls certs
	require.Error(t, secConfig.UpdateRootCA(&otherRootCA))

	// can update the secConfig's root CA to one that does match the certs
	combinedRootCA, err := ca.NewRootCA(append(otherRootCA.Certs, rootCA.Certs...), nil, nil,
		ca.DefaultNodeCertExpiration, nil)
	require.NoError(t, err)
	require.NoError(t, secConfig.UpdateRootCA(&combinedRootCA))

	// if there are intermediates, we can update to a root CA that signed the intermediate
	require.NoError(t, secConfig.UpdateTLSCredentials(&otherTLSKeyPair, otherIssuerInfo))
	require.NoError(t, secConfig.UpdateRootCA(&rootCA))

}

func TestSecurityConfigWatch(t *testing.T) {
	tc := cautils.NewTestCA(t)
	defer tc.Stop()

	secConfig, err := tc.NewNodeConfig(ca.ManagerRole)
	require.NoError(t, err)
	issuer := secConfig.IssuerInfo()

	configWatch, configCancel := secConfig.Watch()
	defer configCancel()

	require.NoError(t, ca.RenewTLSConfigNow(tc.Context, secConfig, tc.ConnBroker, tc.Paths.RootCA))
	select {
	case ev := <-configWatch:
		nodeTLSInfo, ok := ev.(*api.NodeTLSInfo)
		require.True(t, ok)
		require.Equal(t, &api.NodeTLSInfo{
			TrustRoot:           tc.RootCA.Certs,
			CertIssuerPublicKey: issuer.PublicKey,
			CertIssuerSubject:   issuer.Subject,
		}, nodeTLSInfo)
	case <-time.After(time.Second):
		require.FailNow(t, "on TLS certificate update, we should have gotten a security config update")
	}

	require.NoError(t, secConfig.UpdateRootCA(&tc.RootCA))
	select {
	case ev := <-configWatch:
		nodeTLSInfo, ok := ev.(*api.NodeTLSInfo)
		require.True(t, ok)
		require.Equal(t, &api.NodeTLSInfo{
			TrustRoot:           tc.RootCA.Certs,
			CertIssuerPublicKey: issuer.PublicKey,
			CertIssuerSubject:   issuer.Subject,
		}, nodeTLSInfo)
	case <-time.After(time.Second):
		require.FailNow(t, "on TLS certificate update, we should have gotten a security config update")
	}

	configCancel()

	// ensure that we can still update tls certs and roots without error even though the watch is closed
	require.NoError(t, secConfig.UpdateRootCA(&tc.RootCA))
	require.NoError(t, ca.RenewTLSConfigNow(tc.Context, secConfig, tc.ConnBroker, tc.Paths.RootCA))
}

// If we get an unknown authority error when trying to renew the TLS certificate, attempt to download the
// root certificate.  If it validates against the current TLS credentials, it will be used to download
// new ones, (only if the new certificate indicates that it's a worker, though).
func TestRenewTLSConfigUpdatesRootOnUnknownAuthError(t *testing.T) {
	tempdir, err := ioutil.TempDir("", "test-renew-tls-config-now-downloads-root")
	require.NoError(t, err)
	defer os.RemoveAll(tempdir)

	// make 3 CAs
	var (
		certs        = make([][]byte, 3)
		keys         = make([][]byte, 3)
		crossSigneds = make([][]byte, 3)
		cas          = make([]ca.RootCA, 3)
	)
	for i := 0; i < 3; i++ {
		certs[i], keys[i], err = cautils.CreateRootCertAndKey(fmt.Sprintf("CA%d", i))
		require.NoError(t, err)
		switch i {
		case 0:
			crossSigneds[i] = nil
			cas[i], err = ca.NewRootCA(certs[i], certs[i], keys[i], ca.DefaultNodeCertExpiration, nil)
			require.NoError(t, err)
		default:
			crossSigneds[i], err = cas[i-1].CrossSignCACertificate(certs[i])
			require.NoError(t, err)
			cas[i], err = ca.NewRootCA(certs[i-1], certs[i], keys[i], ca.DefaultNodeCertExpiration, crossSigneds[i])
			require.NoError(t, err)
		}
	}

	// the CA server is going to start off with a cert issued by the second CA, cross-signed by the first CA, and then
	// rotate to one issued by the third CA, cross-signed by the second.
	tc := cautils.NewTestCAFromAPIRootCA(t, tempdir, api.RootCA{
		CACert: certs[0],
		CAKey:  keys[0],
		RootRotation: &api.RootRotation{
			CACert:            certs[1],
			CAKey:             keys[1],
			CrossSignedCACert: crossSigneds[1],
		},
	}, nil)
	defer tc.Stop()
	require.NoError(t, tc.MemoryStore.Update(func(tx store.Tx) error {
		cluster := store.GetCluster(tx, tc.Organization)
		cluster.RootCA.CACert = certs[1]
		cluster.RootCA.CAKey = keys[1]
		cluster.RootCA.RootRotation = &api.RootRotation{
			CACert:            certs[2],
			CAKey:             keys[2],
			CrossSignedCACert: crossSigneds[2],
		}
		return store.UpdateCluster(tx, cluster)
	}))
	// wait until the CA is returning certs signed by the latest root
	rootCA, err := ca.NewRootCA(certs[1], nil, nil, ca.DefaultNodeCertExpiration, nil)
	require.NoError(t, err)
	expectedIssuer, err := helpers.ParseCertificatePEM(certs[2])
	require.NoError(t, err)
	require.NoError(t, testutils.PollFuncWithTimeout(nil, func() error {
		_, issuerInfo, err := rootCA.RequestAndSaveNewCertificates(tc.Context, tc.KeyReadWriter, ca.CertificateRequestConfig{
			Token:      tc.WorkerToken,
			ConnBroker: tc.ConnBroker,
		})
		if err != nil {
			return err
		}
		if !bytes.Equal(issuerInfo.PublicKey, expectedIssuer.RawSubjectPublicKeyInfo) {
			return errors.New("CA server hasn't finished updating yet")
		}
		return nil
	}, 2*time.Second))

	paths := ca.NewConfigPaths(tempdir)
	krw := ca.NewKeyReadWriter(paths.Node, nil, nil)
	for i, testCase := range []struct {
		role          api.NodeRole
		initialRootCA *ca.RootCA
		issuingRootCA *ca.RootCA
		expectedRoot  []byte
	}{
		{
			role:          api.NodeRoleWorker,
			initialRootCA: &cas[0],
			issuingRootCA: &cas[1],
			expectedRoot:  certs[1],
		},
		{
			role:          api.NodeRoleManager,
			initialRootCA: &cas[0],
			issuingRootCA: &cas[1],
		},
		// TODO(cyli): once signing root CA and serving root CA for the CA server are split up, so that the server can accept
		// requests from certs different than the cluster root CA, add another test case to make sure that the downloaded
		// root has to validate against both the old TLS creds and new TLS creds
	} {
		nodeID := fmt.Sprintf("node%d", i)
		tlsKeyPair, issuerInfo, err := testCase.issuingRootCA.IssueAndSaveNewCertificates(krw, nodeID, ca.ManagerRole, tc.Organization)
		require.NoError(t, err)
		// make sure the node is added to the memory store as a worker, so when we renew the cert the test CA will answer
		require.NoError(t, tc.MemoryStore.Update(func(tx store.Tx) error {
			return store.CreateNode(tx, &api.Node{
				Role: testCase.role,
				ID:   nodeID,
				Spec: api.NodeSpec{
					DesiredRole:  testCase.role,
					Membership:   api.NodeMembershipAccepted,
					Availability: api.NodeAvailabilityActive,
				},
			})
		}))
		secConfig, qClose, err := ca.NewSecurityConfig(testCase.initialRootCA, krw, tlsKeyPair, issuerInfo)
		require.NoError(t, err)
		defer qClose()

		paths := ca.NewConfigPaths(filepath.Join(tempdir, nodeID))
		err = ca.RenewTLSConfigNow(tc.Context, secConfig, tc.ConnBroker, paths.RootCA)

		// TODO(cyli): remove this role check once the codepaths for worker and manager are the same
		if testCase.expectedRoot != nil {
			// only rotate if we are a worker, and if the new cert validates against the old TLS creds
			require.NoError(t, err)
			downloadedRoot, err := ioutil.ReadFile(paths.RootCA.Cert)
			require.NoError(t, err)
			require.Equal(t, testCase.expectedRoot, downloadedRoot)
		} else {
			require.Error(t, err)
			require.IsType(t, x509.UnknownAuthorityError{}, err)
			_, err = ioutil.ReadFile(paths.RootCA.Cert) // we didn't download a file
			require.Error(t, err)
		}
	}
}

// If we get a not unknown authority error when trying to renew the TLS certificate, just return the
// error and do not attempt to download the root certificate.
func TestRenewTLSConfigUpdatesRootNonUnknownAuthError(t *testing.T) {
	tempdir, err := ioutil.TempDir("", "test-renew-tls-config-now-downloads-root")
	require.NoError(t, err)
	defer os.RemoveAll(tempdir)

	cert, key, err := cautils.CreateRootCertAndKey("rootCA")
	require.NoError(t, err)
	rootCA, err := ca.NewRootCA(cert, cert, key, ca.DefaultNodeCertExpiration, nil)
	require.NoError(t, err)

	tc := cautils.NewTestCAFromAPIRootCA(t, tempdir, api.RootCA{
		CACert: cert,
		CAKey:  key,
	}, nil)
	defer tc.Stop()

	fakeCAServer := newNonSigningCAServer(t, tc)
	defer fakeCAServer.stop(t)

	secConfig, err := tc.NewNodeConfig(ca.WorkerRole)
	require.NoError(t, err)
	tc.CAServer.Stop()

	signErr := make(chan error)
	go func() {
		updates, cancel := state.Watch(tc.MemoryStore.WatchQueue(), api.EventCreateNode{})
		defer cancel()
		select {
		case event := <-updates: // we want to skip the first node, which is the test CA
			n := event.(api.EventCreateNode).Node
			if n.Certificate.Status.State == api.IssuanceStatePending {
				signErr <- tc.MemoryStore.Update(func(tx store.Tx) error {
					node := store.GetNode(tx, n.ID)
					certChain, err := rootCA.ParseValidateAndSignCSR(node.Certificate.CSR, node.Certificate.CN, ca.WorkerRole, tc.Organization)
					if err != nil {
						return err
					}
					node.Certificate.Certificate = cautils.ReDateCert(t, certChain, cert, key, time.Now().Add(-5*time.Hour), time.Now().Add(-4*time.Hour))
					node.Certificate.Status = api.IssuanceStatus{
						State: api.IssuanceStateIssued,
					}
					return store.UpdateNode(tx, node)
				})
				return
			}
		}
	}()

	err = ca.RenewTLSConfigNow(tc.Context, secConfig, fakeCAServer.getConnBroker(), tc.Paths.RootCA)
	require.Error(t, err)
	require.IsType(t, x509.CertificateInvalidError{}, errors.Cause(err))
	require.NoError(t, <-signErr)
}

// enforce that no matter what order updating the root CA and updating TLS credential happens, we
// end up with a security config that has updated certs, and an updated root pool
func TestRenewTLSConfigUpdateRootCARace(t *testing.T) {
	tc := cautils.NewTestCA(t)
	defer tc.Stop()
	paths := ca.NewConfigPaths(tc.TempDir)

	secConfig, err := tc.WriteNewNodeConfig(ca.ManagerRole)
	require.NoError(t, err)

	leafCert, err := ioutil.ReadFile(paths.Node.Cert)
	require.NoError(t, err)

	for i := 0; i < 5; i++ {
		cert, _, err := cautils.CreateRootCertAndKey(fmt.Sprintf("root %d", i+2))
		require.NoError(t, err)

		ctx, cancel := context.WithCancel(tc.Context)
		defer cancel()

		done1, done2 := make(chan struct{}), make(chan struct{})
		rootCA := secConfig.RootCA()
		go func() {
			defer close(done1)
			s := ca.LocalSigner{}
			if signer, err := rootCA.Signer(); err == nil {
				s = *signer
			}
			updatedRootCA, err := ca.NewRootCA(append(rootCA.Certs, cert...), s.Cert, s.Key, ca.DefaultNodeCertExpiration, nil)
			require.NoError(t, err)
			require.NoError(t, secConfig.UpdateRootCA(&updatedRootCA))
		}()

		go func() {
			defer close(done2)
			require.NoError(t, ca.RenewTLSConfigNow(ctx, secConfig, tc.ConnBroker, tc.Paths.RootCA))
		}()

		<-done1
		<-done2

		newCert, err := ioutil.ReadFile(paths.Node.Cert)
		require.NoError(t, err)

		require.NotEqual(t, newCert, leafCert)
		leafCert = newCert

		// at the start of this loop had i+1 certs, afterward should have added one more
		require.Len(t, secConfig.ClientTLSCreds.Config().RootCAs.Subjects(), i+2)
		require.Len(t, secConfig.ServerTLSCreds.Config().RootCAs.Subjects(), i+2)
	}
}

func writeAlmostExpiringCertToDisk(t *testing.T, tc *cautils.TestCA, cn, ou, org string) {
	s, err := tc.RootCA.Signer()
	require.NoError(t, err)

	// Create a new RootCA, and change the policy to issue 6 minute certificates
	// Because of the default backdate of 5 minutes, this issues certificates
	// valid for 1 minute.
	newRootCA, err := ca.NewRootCA(tc.RootCA.Certs, s.Cert, s.Key, ca.DefaultNodeCertExpiration, nil)
	assert.NoError(t, err)
	newSigner, err := newRootCA.Signer()
	require.NoError(t, err)
	newSigner.SetPolicy(&cfconfig.Signing{
		Default: &cfconfig.SigningProfile{
			Usage:  []string{"signing", "key encipherment", "server auth", "client auth"},
			Expiry: 6 * time.Minute,
		},
	})

	// Issue a new certificate with the same details as the current config, but with 1 min expiration time, and
	// overwrite the existing cert on disk
	_, _, err = newRootCA.IssueAndSaveNewCertificates(ca.NewKeyReadWriter(tc.Paths.Node, nil, nil), cn, ou, org)
	assert.NoError(t, err)
}

func TestRenewTLSConfigWorker(t *testing.T) {
	t.Parallel()

	tc := cautils.NewTestCA(t)
	defer tc.Stop()

	ctx, cancel := context.WithCancel(tc.Context)
	defer cancel()

	// Get a new nodeConfig with a TLS cert that has the default Cert duration, but overwrite
	// the cert on disk with one that expires in 1 minute
	nodeConfig, err := tc.WriteNewNodeConfig(ca.WorkerRole)
	assert.NoError(t, err)
	c := nodeConfig.ClientTLSCreds
	writeAlmostExpiringCertToDisk(t, tc, c.NodeID(), c.Role(), c.Organization())

	renewer := ca.NewTLSRenewer(nodeConfig, tc.ConnBroker, tc.Paths.RootCA)
	updates := renewer.Start(ctx)
	select {
	case <-time.After(10 * time.Second):
		assert.Fail(t, "TestRenewTLSConfig timed-out")
	case certUpdate := <-updates:
		assert.NoError(t, certUpdate.Err)
		assert.NotNil(t, certUpdate)
		assert.Equal(t, ca.WorkerRole, certUpdate.Role)
	}

	root, err := helpers.ParseCertificatePEM(tc.RootCA.Certs)
	assert.NoError(t, err)

	issuerInfo := nodeConfig.IssuerInfo()
	assert.NotNil(t, issuerInfo)
	assert.Equal(t, root.RawSubjectPublicKeyInfo, issuerInfo.PublicKey)
	assert.Equal(t, root.RawSubject, issuerInfo.Subject)
}

func TestRenewTLSConfigManager(t *testing.T) {
	t.Parallel()

	tc := cautils.NewTestCA(t)
	defer tc.Stop()

	ctx, cancel := context.WithCancel(tc.Context)
	defer cancel()

	// Get a new nodeConfig with a TLS cert that has the default Cert duration, but overwrite
	// the cert on disk with one that expires in 1 minute
	nodeConfig, err := tc.WriteNewNodeConfig(ca.WorkerRole)
	assert.NoError(t, err)
	c := nodeConfig.ClientTLSCreds
	writeAlmostExpiringCertToDisk(t, tc, c.NodeID(), c.Role(), c.Organization())

	renewer := ca.NewTLSRenewer(nodeConfig, tc.ConnBroker, tc.Paths.RootCA)
	updates := renewer.Start(ctx)
	select {
	case <-time.After(10 * time.Second):
		assert.Fail(t, "TestRenewTLSConfig timed-out")
	case certUpdate := <-updates:
		assert.NoError(t, certUpdate.Err)
		assert.NotNil(t, certUpdate)
		assert.Equal(t, ca.WorkerRole, certUpdate.Role)
	}

	root, err := helpers.ParseCertificatePEM(tc.RootCA.Certs)
	assert.NoError(t, err)

	issuerInfo := nodeConfig.IssuerInfo()
	assert.NotNil(t, issuerInfo)
	assert.Equal(t, root.RawSubjectPublicKeyInfo, issuerInfo.PublicKey)
	assert.Equal(t, root.RawSubject, issuerInfo.Subject)
}

func TestRenewTLSConfigWithNoNode(t *testing.T) {
	t.Parallel()

	tc := cautils.NewTestCA(t)
	defer tc.Stop()

	ctx, cancel := context.WithCancel(tc.Context)
	defer cancel()

	// Get a new nodeConfig with a TLS cert that has the default Cert duration, but overwrite
	// the cert on disk with one that expires in 1 minute
	nodeConfig, err := tc.WriteNewNodeConfig(ca.WorkerRole)
	assert.NoError(t, err)
	c := nodeConfig.ClientTLSCreds
	writeAlmostExpiringCertToDisk(t, tc, c.NodeID(), c.Role(), c.Organization())

	// Delete the node from the backend store
	err = tc.MemoryStore.Update(func(tx store.Tx) error {
		node := store.GetNode(tx, nodeConfig.ClientTLSCreds.NodeID())
		assert.NotNil(t, node)
		return store.DeleteNode(tx, nodeConfig.ClientTLSCreds.NodeID())
	})
	assert.NoError(t, err)

	renewer := ca.NewTLSRenewer(nodeConfig, tc.ConnBroker, tc.Paths.RootCA)
	updates := renewer.Start(ctx)
	select {
	case <-time.After(10 * time.Second):
		assert.Fail(t, "TestRenewTLSConfig timed-out")
	case certUpdate := <-updates:
		assert.Error(t, certUpdate.Err)
		assert.Contains(t, certUpdate.Err.Error(), "not found when attempting to renew certificate")
	}
}
