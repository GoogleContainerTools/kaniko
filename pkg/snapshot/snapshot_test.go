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
	"reflect"
	"sort"
	"strings"
	"testing"

	"github.com/GoogleContainerTools/kaniko/pkg/util"
	"github.com/GoogleContainerTools/kaniko/testutil"
	"github.com/pkg/errors"
)

func Test_Snapshotter_SnapshotFiles(t *testing.T) {
	type testcase struct {
		desc                    string
		files                   []string
		expectErr               bool
		expectedLayeredMap      *LayeredMap
		expectedWriterFiles     []string
		expectedWriterWhiteouts []string
		expectedTarPath         string
	}

	validateResults := func(t *testing.T, tc testcase, snap *Snapshotter, tarPath string, err error) {
		if tc.expectErr {
		} else {
			if err != nil {
				t.Errorf("expected err to be nil but was %s", err)
			}

			if tarPath != tc.expectedTarPath {
				t.Errorf("expected tar path to equal %s but was %s",
					tc.expectedTarPath, tarPath,
				)
			}

			if !layeredMapsMatch(snap.l, tc.expectedLayeredMap) {
				t.Errorf("expected\n%+v\nto equal\n%+v",
					snap.l, tc.expectedLayeredMap)
			}
		}
	}

	t.Run("list of files", func(t *testing.T) {
		dir, err := ioutil.TempDir("", "snapshot-test")
		if err != nil {
			t.Fatal(err)
		}

		defer os.RemoveAll(dir)

		files := []string{
			"/foo/bar.txt",
			"/baz/boom.txt",
		}

		hasher := func(x string) (string, error) {
			return x, nil
		}

		cacheHasher := func(x string) (string, error) {
			return x, nil
		}

		t.Run("all are symlinks", func(t *testing.T) {
			for _, f := range files {
				fLink := filepath.Join(dir, "link", f)
				fTarget := filepath.Join(dir, "target", f)

				if err := os.MkdirAll(filepath.Dir(fTarget), 0777); err != nil {
					t.Fatal(err)
				}

				if err := ioutil.WriteFile(fTarget, []byte{}, 0777); err != nil {
					t.Fatal(err)
				}

				if err := os.MkdirAll(filepath.Dir(fLink), 0777); err != nil {
					t.Fatal(err)
				}

				if err := os.Symlink(fTarget, fLink); err != nil {
					t.Fatal(err)
				}
			}

			t.Run("none are whitelisted", func(t *testing.T) {
				tc := testcase{
					files: files,
					expectedLayeredMap: &LayeredMap{
						hasher:      hasher,
						cacheHasher: cacheHasher,
						layers:      []map[string]string{},
						whiteouts: []map[string]string{
							map[string]string{},
						},
					},
				}

				layer := make(map[string]string)
				inputFiles := []string{}

				for _, f := range files {
					link := filepath.Join(dir, "link", f)
					layer[link] = link
					inputFiles = append(inputFiles, link)

					target := filepath.Join(dir, "target", f)
					layer[target] = target

				}

				tc.expectedLayeredMap.layers = append(tc.expectedLayeredMap.layers, layer)

				lm := &LayeredMap{
					hasher:      hasher,
					cacheHasher: cacheHasher,
				}

				snap := &Snapshotter{
					l: lm,
				}

				tarPath, err := snap.SnapshotFiles(inputFiles)

				validateResults(t, tc, snap, tarPath, err)
			})

			t.Run("some are whitelisted", func(t *testing.T) {
				tc := testcase{
					expectedLayeredMap: &LayeredMap{
						hasher:      hasher,
						cacheHasher: cacheHasher,
						layers:      []map[string]string{},
						whiteouts: []map[string]string{
							map[string]string{},
						},
					},
				}

				whitelist := []util.WhitelistEntry{
					{
						Path: filepath.Join(dir, "link", "baz"),
					},
					{
						Path: filepath.Join(dir, "target", "foo"),
					},
				}

				layer := make(map[string]string)
				inputFiles := []string{}

				for _, f := range files {
					link := filepath.Join(dir, "link", f)
					inputFiles = append(inputFiles, link)

					if util.CheckWhitelistEntries(whitelist, link) {
						t.Logf("skipping %s", link)
						continue
					}

					layer[link] = link

					target := filepath.Join(dir, "target", f)

					if util.CheckWhitelistEntries(whitelist, target) {
						t.Logf("skipping %s", target)
						continue
					}

					layer[target] = target
				}

				link := filepath.Join(dir, "link", "zoom/")

				target := filepath.Join(dir, "target", "zaam/")
				if err := os.MkdirAll(target, 0777); err != nil {
					t.Fatal(err)
				}

				if err := ioutil.WriteFile(filepath.Join(target, "meow.txt"), []byte{}, 0777); err != nil {
					t.Fatal(err)
				}

				if err := os.Symlink(target, link); err != nil {
					t.Fatal(err)
				}

				file := filepath.Join(link, "meow.txt")
				inputFiles = append(inputFiles, file)

				layer[link] = link

				targetFile := filepath.Join(target, "meow.txt")
				layer[targetFile] = targetFile

				tc.expectedLayeredMap.layers = append(tc.expectedLayeredMap.layers, layer)

				lm := &LayeredMap{
					hasher:      hasher,
					cacheHasher: cacheHasher,
				}

				snap := &Snapshotter{
					l:         lm,
					whitelist: whitelist,
				}

				tarPath, err := snap.SnapshotFiles(inputFiles)

				validateResults(t, tc, snap, tarPath, err)
			})
		})
	})

	testCases := []testcase{
		{
			desc:  "empty set of files",
			files: []string{},
			expectedLayeredMap: &LayeredMap{
				layers: []map[string]string{
					map[string]string{},
				},
				whiteouts: []map[string]string{
					map[string]string{},
				},
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.desc, func(t *testing.T) {
			lm := &LayeredMap{}
			snap := &Snapshotter{
				l: lm,
			}

			tarPath, err := snap.SnapshotFiles(tc.files)

			validateResults(t, tc, snap, tarPath, err)
		})
	}
}

func layeredMapsMatch(a *LayeredMap, b *LayeredMap) bool {
	if a == nil && b == nil {
		return true
	}

	if a == nil && b != nil {
		return false
	}

	if a != nil && b == nil {
		return false
	}

	if a.layers == nil && b.layers != nil {
		return false
	}

	if a.layers != nil && b.layers == nil {
		return false
	}

	if a.whiteouts == nil && b.whiteouts != nil {
		return false
	}

	if a.whiteouts != nil && b.whiteouts == nil {
		return false
	}

	if len(a.layers) != len(b.layers) {
		return false
	}

	if len(a.whiteouts) != len(b.whiteouts) {
		return false
	}

	for i, layer := range a.layers {
		tcLayer := b.layers[i]

		if len(layer) != len(tcLayer) {
			return false
		}

		for key := range layer {
			if _, ok := tcLayer[key]; !ok {
				return false
			}
		}

		for key, file := range layer {
			if !reflect.DeepEqual(file, tcLayer[key]) {
				return false
			}
		}
	}

	for i, whiteout := range a.whiteouts {
		tcWhiteout := b.whiteouts[i]

		if len(whiteout) != len(tcWhiteout) {
			return false
		}

		for key := range whiteout {
			if _, ok := tcWhiteout[key]; !ok {
				return false
			}
		}

		for key, file := range whiteout {
			if !reflect.DeepEqual(file, tcWhiteout[key]) {
				return false
			}
		}
	}

	return true
}

func TestSnapshotFSFileChange(t *testing.T) {
	testDir, snapshotter, cleanup, err := setUpTest()
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
	testDir, snapshotter, cleanup, err := setUpTest()
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
	testDir, snapshotter, cleanup, err := setUpTest()
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
	testDir, snapshotter, cleanup, err := setUpTest()
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
	_, snapshotter, cleanup, err := setUpTest()
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

func TestFileWithLinks(t *testing.T) {

	link := "baz/link"
	tcs := []struct {
		name           string
		path           string
		linkFileTarget string
		expected       []string
		shouldErr      bool
	}{
		{
			name:           "given path is a symlink that points to a valid target",
			path:           link,
			linkFileTarget: "file",
			expected:       []string{link, "baz/file"},
		},
		{
			name:           "given path is a symlink points to non existing path",
			path:           link,
			linkFileTarget: "does-not-exists",
			expected:       []string{link},
		},
		{
			name:           "given path is a regular file",
			path:           "kaniko/file",
			linkFileTarget: "file",
			expected:       []string{"kaniko/file"},
		},
	}

	for _, tt := range tcs {
		t.Run(tt.name, func(t *testing.T) {
			testDir, cleanup, err := setUpTestDir()
			if err != nil {
				t.Fatal(err)
			}
			defer cleanup()
			if err := setupSymlink(testDir, link, tt.linkFileTarget); err != nil {
				t.Fatalf("could not set up symlink due to %s", err)
			}
			actual, err := filesWithLinks(filepath.Join(testDir, tt.path))
			if err != nil {
				t.Fatalf("unexpected error %s", err)
			}
			sortAndCompareFilepaths(t, testDir, tt.expected, actual)
		})
	}
}

func TestSnasphotPreservesFileOrder(t *testing.T) {
	newFiles := map[string]string{
		"foo":     "newbaz1",
		"bar/bat": "baz",
		"bar/qux": "quuz",
		"qux":     "quuz",
		"corge":   "grault",
		"garply":  "waldo",
		"fred":    "plugh",
		"xyzzy":   "thud",
	}

	newFileNames := []string{}

	for fileName := range newFiles {
		newFileNames = append(newFileNames, fileName)
	}

	filesInTars := [][]string{}

	for i := 0; i <= 2; i++ {
		testDir, snapshotter, cleanup, err := setUpTest()
		testDirWithoutLeadingSlash := strings.TrimLeft(testDir, "/")
		defer cleanup()

		if err != nil {
			t.Fatal(err)
		}
		// Make some changes to the filesystem
		if err := testutil.SetupFiles(testDir, newFiles); err != nil {
			t.Fatalf("Error setting up fs: %s", err)
		}

		filesToSnapshot := []string{}
		for _, file := range newFileNames {
			filesToSnapshot = append(filesToSnapshot, filepath.Join(testDir, file))
		}

		// Take a snapshot
		tarPath, err := snapshotter.TakeSnapshot(filesToSnapshot)

		if err != nil {
			t.Fatalf("Error taking snapshot of fs: %s", err)
		}

		f, err := os.Open(tarPath)
		if err != nil {
			t.Fatal(err)
		}
		tr := tar.NewReader(f)
		filesInTars = append(filesInTars, []string{})
		for {
			hdr, err := tr.Next()
			if err == io.EOF {
				break
			}
			if err != nil {
				t.Fatal(err)
			}
			filesInTars[i] = append(filesInTars[i], strings.TrimPrefix(hdr.Name, testDirWithoutLeadingSlash))
		}
	}

	// Check contents of all snapshots, make sure files appear in consistent order
	for i := 1; i < len(filesInTars); i++ {
		testutil.CheckErrorAndDeepEqual(t, false, nil, filesInTars[0], filesInTars[i])
	}
}

func TestSnapshotOmitsUnameGname(t *testing.T) {
	_, snapshotter, cleanup, err := setUpTest()

	defer cleanup()
	if err != nil {
		t.Fatal(err)
	}

	tarPath, err := snapshotter.TakeSnapshotFS()
	if err != nil {
		t.Fatal(err)
	}

	f, err := os.Open(tarPath)
	if err != nil {
		t.Fatal(err)
	}
	tr := tar.NewReader(f)
	for {
		hdr, err := tr.Next()
		if err == io.EOF {
			break
		}
		if err != nil {
			t.Fatal(err)
		}
		if hdr.Uname != "" || hdr.Gname != "" {
			t.Fatalf("Expected Uname/Gname for %s to be empty: Uname = '%s', Gname = '%s'", hdr.Name, hdr.Uname, hdr.Gname)
		}
	}

}

func setupSymlink(dir string, link string, target string) error {
	return os.Symlink(target, filepath.Join(dir, link))
}

func sortAndCompareFilepaths(t *testing.T, testDir string, expected []string, actual []string) {
	expectedFullPaths := make([]string, len(expected))
	for i, file := range expected {
		expectedFullPaths[i] = filepath.Join(testDir, file)
	}
	sort.Strings(expectedFullPaths)
	sort.Strings(actual)
	testutil.CheckDeepEqual(t, expectedFullPaths, actual)
}

func setUpTestDir() (string, func(), error) {
	testDir, err := ioutil.TempDir("", "")
	if err != nil {
		return "", nil, errors.Wrap(err, "setting up temp dir")
	}
	files := map[string]string{
		"foo":         "baz1",
		"bar/bat":     "baz2",
		"kaniko/file": "file",
		"baz/file":    "testfile",
	}
	// Set up initial files
	if err := testutil.SetupFiles(testDir, files); err != nil {
		return "", nil, errors.Wrap(err, "setting up file system")
	}

	cleanup := func() {
		os.RemoveAll(testDir)
	}

	return testDir, cleanup, nil
}

func setUpTest() (string, *Snapshotter, func(), error) {
	testDir, dirCleanUp, err := setUpTestDir()
	if err != nil {
		return "", nil, nil, err
	}
	snapshotPath, err := ioutil.TempDir("", "")
	if err != nil {
		return "", nil, nil, errors.Wrap(err, "setting up temp dir")
	}

	snapshotPathPrefix = snapshotPath

	// Take the initial snapshot
	l := NewLayeredMap(util.Hasher(), util.CacheHasher())
	snapshotter := NewSnapshotter(l, testDir)
	if err := snapshotter.Init(); err != nil {
		return "", nil, nil, errors.Wrap(err, "initializing snapshotter")
	}

	cleanup := func() {
		os.RemoveAll(snapshotPath)
		dirCleanUp()
	}

	return testDir, snapshotter, cleanup, nil
}
