// +build linux darwin freebsd

package commands

import (
	"io/ioutil"
	"log"
	"os"
	"os/signal"
	"path/filepath"
	"syscall"
	"time"

	"bazil.org/fuse"
	"bazil.org/fuse/fs"

	"github.com/containerd/continuity"
	"github.com/containerd/continuity/continuityfs"
	"github.com/containerd/continuity/driver"
	"github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
)

var MountCmd = &cobra.Command{
	Use:   "mount <mountpoint> [<manifest>] [<source directory>]",
	Short: "Mount the manifest to the provided mountpoint using content from a source directory",
	Run: func(cmd *cobra.Command, args []string) {
		if len(args) != 3 {
			log.Fatal("Must specify mountpoint, manifest, and source directory")
		}
		mountpoint := args[0]
		manifest, source := args[1], args[2]

		manifestName := filepath.Base(manifest)

		p, err := ioutil.ReadFile(manifest)
		if err != nil {
			log.Fatalf("error reading manifest: %v", err)
		}

		m, err := continuity.Unmarshal(p)
		if err != nil {
			log.Fatalf("error unmarshaling manifest: %v", err)
		}

		driver, err := driver.NewSystemDriver()
		if err != nil {
			logrus.Fatal(err)
		}

		provider := continuityfs.NewFSFileContentProvider(source, driver)

		contfs, err := continuityfs.NewFSFromManifest(m, mountpoint, provider)
		if err != nil {
			logrus.Fatal(err)
		}

		c, err := fuse.Mount(
			mountpoint,
			fuse.ReadOnly(),
			fuse.FSName(manifestName),
			fuse.Subtype("continuity"),
			// OSX Only options
			fuse.LocalVolume(),
			fuse.VolumeName("Continuity FileSystem"),
		)
		if err != nil {
			logrus.Fatal(err)
		}

		<-c.Ready
		if err := c.MountError; err != nil {
			c.Close()
			logrus.Fatal(err)
		}

		errChan := make(chan error, 1)
		go func() {
			// TODO: Create server directory to use context
			err = fs.Serve(c, contfs)
			if err != nil {
				errChan <- err
			}
		}()

		sigChan := make(chan os.Signal, 1)
		signal.Notify(sigChan, os.Interrupt)
		signal.Notify(sigChan, syscall.SIGTERM)

		select {
		case <-sigChan:
			logrus.Infof("Shutting down")
		case err = <-errChan:
		}

		go func() {
			if err := c.Close(); err != nil {
				logrus.Errorf("Unable to close connection %s", err)
			}
		}()

		// Wait for any inprogress requests to be handled
		time.Sleep(time.Second)

		logrus.Infof("Attempting unmount")
		if err := fuse.Unmount(mountpoint); err != nil {
			logrus.Errorf("Error unmounting %s: %v", mountpoint, err)
		}

		// Handle server error
		if err != nil {
			logrus.Fatalf("Error serving fuse server: %v", err)
		}
	},
}
