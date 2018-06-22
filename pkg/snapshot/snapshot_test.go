/*
Copyright 2018 Google LLC

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/
package snapshot

import (
	"archive/tar"
	"bytes"
	"io"
	"io/ioutil"
	"os"
	"path/filepath"
	"testing"

	"github.com/GoogleContainerTools/kaniko/pkg/util"
	"github.com/GoogleContainerTools/kaniko/testutil"
	"github.com/pkg/errors"
)

func TestSnapshotFileChange(t *testing.T) {

	testDir, snapshotter, err := setUpTestDir()
	defer os.RemoveAll(testDir)
	if err != nil {
		t.Fatal(err)
	}
	// Make some changes to the filesystem
	newFiles := map[string]string{
		"foo":        "newbaz1",
		"bar/bat":    "baz",
		"kaniko/bat": "bat",
	}
	if err := testutil.SetupFiles(testDir, newFiles); err != nil {
		t.Fatalf("Error setting up fs: %s", err)
	}
	// Take another snapshot
	contents, err := snapshotter.TakeSnapshot(nil)
	if err != nil {
		t.Fatalf("Error taking snapshot of fs: %s", err)
	}
	if contents == nil {
		t.Fatal("No files added to snapshot.")
	}
	// Check contents of the snapshot, make sure contents is equivalent to snapshotFiles
	reader := bytes.NewReader(contents)
	tr := tar.NewReader(reader)
	fooPath := filepath.Join(testDir, "foo")
	batPath := filepath.Join(testDir, "bar/bat")
	snapshotFiles := map[string]string{
		fooPath: "newbaz1",
		batPath: "baz",
	}
	numFiles := 0
	for {
		hdr, err := tr.Next()
		if err == io.EOF {
			break
		}
		numFiles++
		if _, isFile := snapshotFiles[hdr.Name]; !isFile {
			t.Fatalf("File %s unexpectedly in tar", hdr.Name)
		}
		contents, _ := ioutil.ReadAll(tr)
		if string(contents) != snapshotFiles[hdr.Name] {
			t.Fatalf("Contents of %s incorrect, expected: %s, actual: %s", hdr.Name, snapshotFiles[hdr.Name], string(contents))
		}
	}
	if numFiles != 2 {
		t.Fatalf("Incorrect number of files were added, expected: 2, actual: %v", numFiles)
	}
}

func TestSnapshotChangePermissions(t *testing.T) {
	testDir, snapshotter, err := setUpTestDir()
	defer os.RemoveAll(testDir)
	if err != nil {
		t.Fatal(err)
	}
	// Change permissions on a file
	batPath := filepath.Join(testDir, "bar/bat")
	if err := os.Chmod(batPath, 0600); err != nil {
		t.Fatalf("Error changing permissions on %s: %v", batPath, err)
	}
	// Take another snapshot
	contents, err := snapshotter.TakeSnapshot(nil)
	if err != nil {
		t.Fatalf("Error taking snapshot of fs: %s", err)
	}
	if contents == nil {
		t.Fatal("No files added to snapshot.")
	}
	// Check contents of the snapshot, make sure contents is equivalent to snapshotFiles
	reader := bytes.NewReader(contents)
	tr := tar.NewReader(reader)
	snapshotFiles := map[string]string{
		batPath: "baz2",
	}
	numFiles := 0
	for {
		hdr, err := tr.Next()
		if err == io.EOF {
			break
		}
		numFiles++
		if _, isFile := snapshotFiles[hdr.Name]; !isFile {
			t.Fatalf("File %s unexpectedly in tar", hdr.Name)
		}
		contents, _ := ioutil.ReadAll(tr)
		if string(contents) != snapshotFiles[hdr.Name] {
			t.Fatalf("Contents of %s incorrect, expected: %s, actual: %s", hdr.Name, snapshotFiles[hdr.Name], string(contents))
		}
	}
	if numFiles != 1 {
		t.Fatalf("Incorrect number of files were added, expected: 1, got: %v", numFiles)
	}
}

func TestSnapshotFiles(t *testing.T) {
	testDir, snapshotter, err := setUpTestDir()
	defer os.RemoveAll(testDir)
	if err != nil {
		t.Fatal(err)
	}
	// Make some changes to the filesystem
	newFiles := map[string]string{
		"foo":         "newbaz1",
		"kaniko/file": "bat",
	}
	if err := testutil.SetupFiles(testDir, newFiles); err != nil {
		t.Fatalf("Error setting up fs: %s", err)
	}
	filesToSnapshot := []string{
		filepath.Join(testDir, "foo"),
		filepath.Join(testDir, "kaniko/file"),
	}
	contents, err := snapshotter.TakeSnapshot(filesToSnapshot)
	if err != nil {
		t.Fatal(err)
	}
	expectedFiles := []string{"/", "/tmp", filepath.Join(testDir, "foo")}

	// Check contents of the snapshot, make sure contents is equivalent to snapshotFiles
	reader := bytes.NewReader(contents)
	tr := tar.NewReader(reader)
	var actualFiles []string
	for {
		hdr, err := tr.Next()
		if err == io.EOF {
			break
		}
		if err != nil {
			t.Fatal(err)
		}
		actualFiles = append(actualFiles, hdr.Name)
	}
	testutil.CheckErrorAndDeepEqual(t, false, nil, expectedFiles, actualFiles)
}

func TestEmptySnapshot(t *testing.T) {
	testDir, snapshotter, err := setUpTestDir()
	defer os.RemoveAll(testDir)
	if err != nil {
		t.Fatal(err)
	}
	// Take snapshot with no changes
	contents, err := snapshotter.TakeSnapshot(nil)
	if err != nil {
		t.Fatalf("Error taking snapshot of fs: %s", err)
	}
	// Since we took a snapshot with no changes, contents should be nil
	if contents != nil {
		t.Fatal("Files added even though no changes to file system were made.")
	}
}

func setUpTestDir() (string, *Snapshotter, error) {
	testDir, err := ioutil.TempDir("", "")
	if err != nil {
		return testDir, nil, errors.Wrap(err, "setting up temp dir")
	}
	files := map[string]string{
		"foo":         "baz1",
		"bar/bat":     "baz2",
		"kaniko/file": "file",
	}
	// Set up initial files
	if err := testutil.SetupFiles(testDir, files); err != nil {
		return testDir, nil, errors.Wrap(err, "setting up file system")
	}

	// Take the initial snapshot
	l := NewLayeredMap(util.Hasher())
	snapshotter := NewSnapshotter(l, testDir)
	if err := snapshotter.Init(); err != nil {
		return testDir, nil, errors.Wrap(err, "initializing snapshotter")
	}
	return testDir, snapshotter, nil
}
