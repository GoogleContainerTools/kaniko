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
	"io"
	"io/ioutil"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"testing"

	"github.com/GoogleContainerTools/kaniko/pkg/util"
	"github.com/GoogleContainerTools/kaniko/testutil"
	"github.com/pkg/errors"
)

func TestSnapshotFSFileChange(t *testing.T) {
	testDir, snapshotter, cleanup, err := setUpTestDir()
	testDirWithoutLeadingSlash := strings.TrimLeft(testDir, "/")
	defer cleanup()
	if err != nil {
		t.Fatal(err)
	}
	// Make some changes to the filesystem
	newFiles := map[string]string{
		"foo":     "newbaz1",
		"bar/bat": "baz",
	}
	if err := testutil.SetupFiles(testDir, newFiles); err != nil {
		t.Fatalf("Error setting up fs: %s", err)
	}
	// Take another snapshot
	tarPath, err := snapshotter.TakeSnapshotFS()
	if err != nil {
		t.Fatalf("Error taking snapshot of fs: %s", err)
	}

	f, err := os.Open(tarPath)
	if err != nil {
		t.Fatal(err)
	}
	// Check contents of the snapshot, make sure contents is equivalent to snapshotFiles
	tr := tar.NewReader(f)
	fooPath := filepath.Join(testDirWithoutLeadingSlash, "foo")
	batPath := filepath.Join(testDirWithoutLeadingSlash, "bar/bat")
	snapshotFiles := map[string]string{
		fooPath: "newbaz1",
		batPath: "baz",
	}
	for _, dir := range util.ParentDirectoriesWithoutLeadingSlash(fooPath) {
		snapshotFiles[dir] = ""
	}
	for _, dir := range util.ParentDirectoriesWithoutLeadingSlash(batPath) {
		snapshotFiles[dir] = ""
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
	if numFiles != len(snapshotFiles) {
		t.Fatalf("Incorrect number of files were added, expected: 2, actual: %v", numFiles)
	}
}

func TestSnapshotFSIsReproducible(t *testing.T) {
	testDir, snapshotter, cleanup, err := setUpTestDir()
	defer cleanup()
	if err != nil {
		t.Fatal(err)
	}
	// Make some changes to the filesystem
	newFiles := map[string]string{
		"foo":     "newbaz1",
		"bar/bat": "baz",
	}
	if err := testutil.SetupFiles(testDir, newFiles); err != nil {
		t.Fatalf("Error setting up fs: %s", err)
	}
	// Take another snapshot
	tarPath, err := snapshotter.TakeSnapshotFS()
	if err != nil {
		t.Fatalf("Error taking snapshot of fs: %s", err)
	}

	f, err := os.Open(tarPath)
	if err != nil {
		t.Fatal(err)
	}
	// Check contents of the snapshot, make sure contents are sorted by name
	tr := tar.NewReader(f)
	var filesInTar []string
	for {
		hdr, err := tr.Next()
		if err == io.EOF {
			break
		}
		filesInTar = append(filesInTar, hdr.Name)
	}
	if !sort.StringsAreSorted(filesInTar) {
		t.Fatalf("Expected the file in the tar archive were sorted, actual list was not sorted: %v", filesInTar)
	}
}

func TestSnapshotFSChangePermissions(t *testing.T) {
	testDir, snapshotter, cleanup, err := setUpTestDir()
	testDirWithoutLeadingSlash := strings.TrimLeft(testDir, "/")
	defer cleanup()
	if err != nil {
		t.Fatal(err)
	}
	// Change permissions on a file
	batPath := filepath.Join(testDir, "bar/bat")
	batPathWithoutLeadingSlash := filepath.Join(testDirWithoutLeadingSlash, "bar/bat")
	if err := os.Chmod(batPath, 0600); err != nil {
		t.Fatalf("Error changing permissions on %s: %v", batPath, err)
	}
	// Take another snapshot
	tarPath, err := snapshotter.TakeSnapshotFS()
	if err != nil {
		t.Fatalf("Error taking snapshot of fs: %s", err)
	}
	f, err := os.Open(tarPath)
	if err != nil {
		t.Fatal(err)
	}
	// Check contents of the snapshot, make sure contents is equivalent to snapshotFiles
	tr := tar.NewReader(f)
	snapshotFiles := map[string]string{
		batPathWithoutLeadingSlash: "baz2",
	}
	for _, dir := range util.ParentDirectoriesWithoutLeadingSlash(batPath) {
		snapshotFiles[dir] = ""
	}
	numFiles := 0
	for {
		hdr, err := tr.Next()
		if err == io.EOF {
			break
		}
		t.Logf("Info %s in tar", hdr.Name)
		numFiles++
		if _, isFile := snapshotFiles[hdr.Name]; !isFile {
			t.Fatalf("File %s unexpectedly in tar", hdr.Name)
		}
		contents, _ := ioutil.ReadAll(tr)
		if string(contents) != snapshotFiles[hdr.Name] {
			t.Fatalf("Contents of %s incorrect, expected: %s, actual: %s", hdr.Name, snapshotFiles[hdr.Name], string(contents))
		}
	}
	if numFiles != len(snapshotFiles) {
		t.Fatalf("Incorrect number of files were added, expected: 1, got: %v", numFiles)
	}
}

func TestSnapshotFiles(t *testing.T) {
	testDir, snapshotter, cleanup, err := setUpTestDir()
	testDirWithoutLeadingSlash := strings.TrimLeft(testDir, "/")
	defer cleanup()
	if err != nil {
		t.Fatal(err)
	}
	// Make some changes to the filesystem
	newFiles := map[string]string{
		"foo": "newbaz1",
	}
	if err := testutil.SetupFiles(testDir, newFiles); err != nil {
		t.Fatalf("Error setting up fs: %s", err)
	}
	filesToSnapshot := []string{
		filepath.Join(testDir, "foo"),
	}
	tarPath, err := snapshotter.TakeSnapshot(filesToSnapshot)
	if err != nil {
		t.Fatal(err)
	}
	defer os.Remove(tarPath)

	expectedFiles := []string{
		filepath.Join(testDirWithoutLeadingSlash, "foo"),
	}
	expectedFiles = append(expectedFiles, util.ParentDirectoriesWithoutLeadingSlash(filepath.Join(testDir, "foo"))...)

	f, err := os.Open(tarPath)
	if err != nil {
		t.Fatal(err)
	}
	// Check contents of the snapshot, make sure contents is equivalent to snapshotFiles
	tr := tar.NewReader(f)
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
	sort.Strings(expectedFiles)
	sort.Strings(actualFiles)
	testutil.CheckErrorAndDeepEqual(t, false, nil, expectedFiles, actualFiles)
}

func TestEmptySnapshotFS(t *testing.T) {
	_, snapshotter, cleanup, err := setUpTestDir()
	if err != nil {
		t.Fatal(err)
	}
	defer cleanup()

	// Take snapshot with no changes
	tarPath, err := snapshotter.TakeSnapshotFS()
	if err != nil {
		t.Fatalf("Error taking snapshot of fs: %s", err)
	}

	f, err := os.Open(tarPath)
	if err != nil {
		t.Fatal(err)
	}
	tr := tar.NewReader(f)

	if _, err := tr.Next(); err != io.EOF {
		t.Fatal("no files expected in tar, found files.")
	}
}

func setUpTestDir() (string, *Snapshotter, func(), error) {
	testDir, err := ioutil.TempDir("", "")
	if err != nil {
		return "", nil, nil, errors.Wrap(err, "setting up temp dir")
	}

	snapshotPath, err := ioutil.TempDir("", "")
	if err != nil {
		return "", nil, nil, errors.Wrap(err, "setting up temp dir")
	}

	snapshotPathPrefix = snapshotPath

	files := map[string]string{
		"foo":         "baz1",
		"bar/bat":     "baz2",
		"kaniko/file": "file",
	}
	// Set up initial files
	if err := testutil.SetupFiles(testDir, files); err != nil {
		return "", nil, nil, errors.Wrap(err, "setting up file system")
	}

	// Take the initial snapshot
	l := NewLayeredMap(util.Hasher(), util.CacheHasher())
	snapshotter := NewSnapshotter(l, testDir)
	if err := snapshotter.Init(); err != nil {
		return "", nil, nil, errors.Wrap(err, "initializing snapshotter")
	}

	cleanup := func() {
		os.RemoveAll(snapshotPath)
		os.RemoveAll(testDir)
	}

	return testDir, snapshotter, cleanup, nil
}
