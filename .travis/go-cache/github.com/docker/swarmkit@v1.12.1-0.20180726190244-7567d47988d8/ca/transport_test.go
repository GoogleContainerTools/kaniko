package ca

import (
	"crypto/tls"
	"io/ioutil"
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewMutableTLS(t *testing.T) {
	tempdir, err := ioutil.TempDir("", "test-transport")
	require.NoError(t, err)
	defer os.RemoveAll(tempdir)
	paths := NewConfigPaths(tempdir)
	krw := NewKeyReadWriter(paths.Node, nil, nil)

	rootCA, err := CreateRootCA("rootCN")
	require.NoError(t, err)

	cert, _, err := rootCA.IssueAndSaveNewCertificates(krw, "CN", ManagerRole, "org")
	assert.NoError(t, err)

	tlsConfig, err := NewServerTLSConfig([]tls.Certificate{*cert}, rootCA.Pool)
	assert.NoError(t, err)
	creds, err := NewMutableTLS(tlsConfig)
	assert.NoError(t, err)
	assert.Equal(t, ManagerRole, creds.Role())
	assert.Equal(t, "CN", creds.NodeID())
}

func TestGetAndValidateCertificateSubject(t *testing.T) {
	tempdir, err := ioutil.TempDir("", "test-transport")
	require.NoError(t, err)
	defer os.RemoveAll(tempdir)
	paths := NewConfigPaths(tempdir)
	krw := NewKeyReadWriter(paths.Node, nil, nil)

	rootCA, err := CreateRootCA("rootCN")
	require.NoError(t, err)

	cert, _, err := rootCA.IssueAndSaveNewCertificates(krw, "CN", ManagerRole, "org")
	assert.NoError(t, err)

	name, err := GetAndValidateCertificateSubject([]tls.Certificate{*cert})
	assert.NoError(t, err)
	assert.Equal(t, "CN", name.CommonName)
	assert.Len(t, name.OrganizationalUnit, 1)
	assert.Equal(t, ManagerRole, name.OrganizationalUnit[0])
}

func TestLoadNewTLSConfig(t *testing.T) {
	tempdir, err := ioutil.TempDir("", "test-transport")
	require.NoError(t, err)
	defer os.RemoveAll(tempdir)
	paths := NewConfigPaths(tempdir)
	krw := NewKeyReadWriter(paths.Node, nil, nil)

	rootCA, err := CreateRootCA("rootCN")
	require.NoError(t, err)

	// Create two different certs and two different TLS configs
	cert1, _, err := rootCA.IssueAndSaveNewCertificates(krw, "CN1", ManagerRole, "org")
	assert.NoError(t, err)
	cert2, _, err := rootCA.IssueAndSaveNewCertificates(krw, "CN2", WorkerRole, "org")
	assert.NoError(t, err)
	tlsConfig1, err := NewServerTLSConfig([]tls.Certificate{*cert1}, rootCA.Pool)
	assert.NoError(t, err)
	tlsConfig2, err := NewServerTLSConfig([]tls.Certificate{*cert2}, rootCA.Pool)
	assert.NoError(t, err)

	// Load the first TLS config into a MutableTLS
	creds, err := NewMutableTLS(tlsConfig1)
	assert.NoError(t, err)
	assert.Equal(t, ManagerRole, creds.Role())
	assert.Equal(t, "CN1", creds.NodeID())

	// Load the new Config and assert it changed
	err = creds.loadNewTLSConfig(tlsConfig2)
	assert.NoError(t, err)
	assert.Equal(t, WorkerRole, creds.Role())
	assert.Equal(t, "CN2", creds.NodeID())
}
