package ca_test

import (
	"context"
	"crypto/x509"
	"net"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/cloudflare/cfssl/helpers"
	"github.com/docker/swarmkit/ca"
	"github.com/docker/swarmkit/ca/testutils"
	"github.com/docker/swarmkit/log"
	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/require"
)

// Tests ExternalCA.CrossSignRootCA can produce an intermediate that can be used to
// validate a leaf certificate
func TestExternalCACrossSign(t *testing.T) {
	t.Parallel()

	if !testutils.External {
		return // this is only tested using the external CA
	}

	tc := testutils.NewTestCA(t)
	defer tc.Stop()
	paths := ca.NewConfigPaths(tc.TempDir)

	secConfig, cancel, err := tc.RootCA.CreateSecurityConfig(tc.Context,
		ca.NewKeyReadWriter(paths.Node, nil, nil), ca.CertificateRequestConfig{})
	require.NoError(t, err)
	cancel()

	externalCA := ca.NewExternalCA(nil,
		ca.NewExternalCATLSConfig(secConfig.ClientTLSCreds.Config().Certificates, tc.RootCA.Pool),
		tc.ExternalSigningServer.URL)

	for _, testcase := range []struct{ cert, key []byte }{
		{
			cert: testutils.ECDSA256SHA256Cert,
			key:  testutils.ECDSA256Key,
		},
		{
			cert: testutils.RSA2048SHA256Cert,
			key:  testutils.RSA2048Key,
		},
	} {
		rootCA2, err := ca.NewRootCA(testcase.cert, testcase.cert, testcase.key, ca.DefaultNodeCertExpiration, nil)
		require.NoError(t, err)

		krw := ca.NewKeyReadWriter(paths.Node, nil, nil)

		_, _, err = rootCA2.IssueAndSaveNewCertificates(krw, "cn", "ou", "org")
		require.NoError(t, err)
		certBytes, _, err := krw.Read()
		require.NoError(t, err)
		leafCert, err := helpers.ParseCertificatePEM(certBytes)
		require.NoError(t, err)

		// we have not enabled CA signing on the external server
		tc.ExternalSigningServer.DisableCASigning()
		_, err = externalCA.CrossSignRootCA(tc.Context, rootCA2)
		require.Error(t, err)

		require.NoError(t, tc.ExternalSigningServer.EnableCASigning())

		intermediate, err := externalCA.CrossSignRootCA(tc.Context, rootCA2)
		require.NoError(t, err)

		parsedIntermediate, err := helpers.ParseCertificatePEM(intermediate)
		require.NoError(t, err)
		parsedRoot2, err := helpers.ParseCertificatePEM(testcase.cert)
		require.NoError(t, err)
		require.Equal(t, parsedRoot2.RawSubject, parsedIntermediate.RawSubject)
		require.Equal(t, parsedRoot2.RawSubjectPublicKeyInfo, parsedIntermediate.RawSubjectPublicKeyInfo)
		require.True(t, parsedIntermediate.IsCA)

		intermediatePool := x509.NewCertPool()
		intermediatePool.AddCert(parsedIntermediate)

		// we can validate a chain from the leaf to the first root through the intermediate,
		// or from the leaf cert to the second root with or without the intermediate
		_, err = leafCert.Verify(x509.VerifyOptions{Roots: tc.RootCA.Pool})
		require.Error(t, err)
		_, err = leafCert.Verify(x509.VerifyOptions{Roots: tc.RootCA.Pool, Intermediates: intermediatePool})
		require.NoError(t, err)

		_, err = leafCert.Verify(x509.VerifyOptions{Roots: rootCA2.Pool})
		require.NoError(t, err)
		_, err = leafCert.Verify(x509.VerifyOptions{Roots: rootCA2.Pool, Intermediates: intermediatePool})
		require.NoError(t, err)
	}
}

func TestExternalCASignRequestTimesOut(t *testing.T) {
	t.Parallel()

	if testutils.External {
		return // this does not require the external CA in any way
	}

	ctx := log.WithLogger(context.Background(), log.L.WithFields(logrus.Fields{
		"testname":          t.Name(),
		"testHasExternalCA": false,
	}))

	signDone, allDone := make(chan error), make(chan struct{})
	defer close(signDone)
	mux := http.NewServeMux()
	mux.HandleFunc("/", func(http.ResponseWriter, *http.Request) {
		// hang forever
		select {
		case <-allDone:
		}
	})

	server := httptest.NewServer(mux)
	defer server.Close()
	defer server.CloseClientConnections()
	defer close(allDone)

	csr, _, err := ca.GenerateNewCSR()
	require.NoError(t, err)

	externalCA := ca.NewExternalCA(nil, nil, server.URL)
	externalCA.ExternalRequestTimeout = time.Second
	go func() {
		_, err := externalCA.Sign(ctx, ca.PrepareCSR(csr, "cn", "ou", "org"))
		select {
		case <-allDone:
		case signDone <- err:
		}
	}()

	select {
	case err = <-signDone:
		require.Contains(t, err.Error(), context.DeadlineExceeded.Error())
	case <-time.After(3 * time.Second):
		require.FailNow(t, "call to external CA signing should have timed out after 1 second - it's been 3")
	}
}

// The ExternalCA object will stop reading the response from the server past a
// a certain size
func TestExternalCASignRequestSizeLimit(t *testing.T) {
	t.Parallel()

	if testutils.External {
		return // this does not require the external CA in any way
	}

	ctx := log.WithLogger(context.Background(), log.L.WithFields(logrus.Fields{
		"testname":          t.Name(),
		"testHasExternalCA": false,
	}))

	rootCA, err := ca.CreateRootCA("rootCN")
	require.NoError(t, err)

	signDone, allDone, writeDone := make(chan error), make(chan struct{}), make(chan error)
	defer close(signDone)
	mux := http.NewServeMux()

	mux.HandleFunc("/", func(w http.ResponseWriter, _ *http.Request) {
		garbage := []byte("abcdefghijklmnopqrstuvwxyz")
		// keep writing until done
		for {
			select {
			case <-allDone:
				return
			default:
				if _, err := w.Write(garbage); err != nil {
					writeDone <- err
					return
				}
			}
		}
	})

	server := httptest.NewServer(mux)
	defer server.Close()
	defer server.CloseClientConnections()
	defer close(allDone)

	csr, _, err := ca.GenerateNewCSR()
	require.NoError(t, err)

	externalCA := ca.NewExternalCA(rootCA.Intermediates, nil, server.URL)
	externalCA.ExternalRequestTimeout = time.Second
	go func() {
		_, err := externalCA.Sign(ctx, ca.PrepareCSR(csr, "cn", "ou", "org"))
		select {
		case <-allDone:
		case signDone <- err:
		}
	}()

	select {
	case err = <-signDone:
		require.Error(t, err)
		require.Contains(t, err.Error(), "unable to parse JSON response")
	case <-time.After(2 * time.Second):
		require.FailNow(t, "call to external CA signing should have failed by now")
	}

	select {
	case err := <-writeDone:
		// due to buffering/client disconnecting, we don't know how much was written to the TCP socket,
		// but the client should have terminated the connection after receiving the max amount, so the
		// request should have finished and the write to the socket failed.
		require.Error(t, err)
		require.IsType(t, &net.OpError{}, err)
	case <-time.After(time.Second):
		require.FailNow(t, "the client connection to the server should have been closed by now")
	}
}
