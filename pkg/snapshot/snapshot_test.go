package snapshot

import (
	"archive/tar"
	"github.com/GoogleCloudPlatform/k8s-container-builder/pkg/constants"
	"github.com/GoogleCloudPlatform/k8s-container-builder/pkg/util"
	"github.com/GoogleCloudPlatform/k8s-container-builder/testutil"
	"io"
	"io/ioutil"
	"os"
	"path/filepath"
	"testing"
)

func TestSnapshot(t *testing.T) {
	testDir, err := ioutil.TempDir("", "")
	if err != nil {
		t.Fatalf("Error creating tempdir: %s", err)
	}
	t.Log(testDir)
	defer os.RemoveAll(testDir)
	files := map[string]string{
		"foo":           "baz1",
		"bar/bat":       "baz2",
		"proc/baz":      "baz3",
		"work-dir/file": "file",
	}
	if err := testutil.SetupFiles(testDir, files); err != nil {
		t.Fatalf("Error setting up fs: %s", err)
	}
	l := NewLayeredMap(util.Hasher())
	snapshotter := NewSnapshotter(l, testDir)
	snapshotter.Init()

	newFiles := map[string]string{
		"foo":      "newbaz1",
		"proc/bat": "bat",
	}
	if err := testutil.SetupFiles(testDir, newFiles); err != nil {
		t.Fatalf("Error setting up fs: %s", err)
	}
	if err := snapshotter.TakeSnapshot(); err != nil {
		t.Fatalf("Error taking snapshot of fs: %s", err)
	}

	pathToTar := filepath.Join(testDir, constants.WorkDir, "layer-0.tar")
	tarFile, err := os.Open(pathToTar)
	if err != nil {
		t.Fatalf("Error finding %s: %s", pathToTar, err)
	}
	tr := tar.NewReader(tarFile)
	fooPath := filepath.Join(testDir, "foo")
	snapshotFiles := map[string]string{
		fooPath: "newbaz1",
	}
	for {
		hdr, err := tr.Next()
		if err == io.EOF {
			break
		}
		if _, isFile := snapshotFiles[hdr.Name]; !isFile {
			t.Fatalf("File %s unexpectedly in tar", hdr.Name)
		}
		contents, _ := ioutil.ReadAll(tr)
		if string(contents) != snapshotFiles[hdr.Name] {
			t.Fatalf("Contents of %s incorrect, expected: %s, actual: %s", hdr.Name, snapshotFiles[hdr.Name], string(contents))
		}

	}
}

func TestEmptySnapshot(t *testing.T) {
	testDir, err := ioutil.TempDir("", "")
	if err != nil {
		t.Fatalf("Error creating tempdir: %s", err)
	}
	t.Log(testDir)
	defer os.RemoveAll(testDir)
	files := map[string]string{
		"foo":           "baz1",
		"bar/bat":       "baz2",
		"proc/baz":      "baz3",
		"work-dir/file": "file",
	}
	if err := testutil.SetupFiles(testDir, files); err != nil {
		t.Fatalf("Error setting up fs: %s", err)
	}
	l := NewLayeredMap(util.Hasher())
	snapshotter := NewSnapshotter(l, testDir)
	snapshotter.Init()
	if err := snapshotter.TakeSnapshot(); err != nil {
		t.Fatalf("Error taking snapshot of fs: %s", err)
	}
	// Since we took a snapshot with no changes, no layer should be created
	// Make sure layer-0.tar does not exist
	if file, err := os.Open(filepath.Join(testDir, "layer-0.tar")); err == nil {
		t.Fatalf("File %s exists, and it should not.", file.Name())
	}
}
