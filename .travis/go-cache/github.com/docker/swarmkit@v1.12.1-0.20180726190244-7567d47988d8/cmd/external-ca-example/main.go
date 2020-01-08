package main

import (
	"io/ioutil"
	"os"
	"os/signal"
	"path/filepath"
	"syscall"

	"github.com/docker/swarmkit/ca"
	"github.com/docker/swarmkit/ca/testutils"
	"github.com/docker/swarmkit/identity"
	"github.com/sirupsen/logrus"
)

func main() {
	// Create root material within the current directory.
	rootPaths := ca.CertPaths{
		Cert: filepath.Join("ca", "root.crt"),
		Key:  filepath.Join("ca", "root.key"),
	}

	// Initialize the Root CA.
	rootCA, err := ca.CreateRootCA("external-ca-example")
	if err != nil {
		logrus.Fatalf("unable to initialize Root CA: %s", err.Error())
	}
	if err := ca.SaveRootCA(rootCA, rootPaths); err != nil {
		logrus.Fatalf("unable to save Root CA: %s", err.Error())
	}

	// Create the initial manager node credentials.
	nodeConfigPaths := ca.NewConfigPaths("certificates")

	clusterID := identity.NewID()
	nodeID := identity.NewID()

	kw := ca.NewKeyReadWriter(nodeConfigPaths.Node, nil, nil)
	if _, _, err := rootCA.IssueAndSaveNewCertificates(kw, nodeID, ca.ManagerRole, clusterID); err != nil {
		logrus.Fatalf("unable to create initial manager node credentials: %s", err)
	}

	// And copy the Root CA certificate into the node config path for its
	// CA.
	ioutil.WriteFile(nodeConfigPaths.RootCA.Cert, rootCA.Certs, os.FileMode(0644))

	server, err := testutils.NewExternalSigningServer(rootCA, "ca")
	if err != nil {
		logrus.Fatalf("unable to start server: %s", err)
	}

	defer server.Stop()

	logrus.Infof("Now run: swarmd -d . --listen-control-api ./swarmd.sock --external-ca protocol=cfssl,url=%s", server.URL)

	sigC := make(chan os.Signal, 1)
	signal.Notify(sigC, syscall.SIGTERM, syscall.SIGINT)

	<-sigC
}
