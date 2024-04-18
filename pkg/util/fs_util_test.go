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

package util

import (
	"archive/tar"
	"bytes"
	"fmt"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"reflect"
	"sort"
	"strings"
	"testing"
	"time"

	"github.com/GoogleContainerTools/kaniko/pkg/config"
	"github.com/GoogleContainerTools/kaniko/pkg/constants"
	"github.com/GoogleContainerTools/kaniko/pkg/mocks/go-containerregistry/mockv1"
	"github.com/GoogleContainerTools/kaniko/testutil"
	"github.com/golang/mock/gomock"
	v1 "github.com/google/go-containerregistry/pkg/v1"
	"github.com/google/go-containerregistry/pkg/v1/types"
)

func Test_DetectFilesystemSkiplist(t *testing.T) {
	testDir := t.TempDir()
	fileContents := `
	228 122 0:90 / / rw,relatime - aufs none rw,si=f8e2406af90782bc,dio,dirperm1
	229 228 0:98 / /proc rw,nosuid,nodev,noexec,relatime - proc proc rw
	230 228 0:99 / /dev rw,nosuid - tmpfs tmpfs rw,size=65536k,mode=755
	231 230 0:100 / /dev/pts rw,nosuid,noexec,relatime - devpts devpts rw,gid=5,mode=620,ptmxmode=666
	232 228 0:101 / /sys ro,nosuid,nodev,noexec,relatime - sysfs sysfs ro`

	path := filepath.Join(testDir, "mountinfo")
	if err := os.MkdirAll(filepath.Dir(path), 0o750); err != nil {
		t.Fatalf("Error creating tempdir: %s", err)
	}
	if err := os.WriteFile(path, []byte(fileContents), 0o644); err != nil {
		t.Fatalf("Error writing file contents to %s: %s", path, err)
	}

	err := DetectFilesystemIgnoreList(path)
	expectedSkiplist := []IgnoreListEntry{
		{"/kaniko", false},
		{"/proc", false},
		{"/dev", false},
		{"/dev/pts", false},
		{"/sys", false},
		{"/etc/mtab", false},
		{"/tmp/apt-key-gpghome", true},
	}
	actualSkiplist := ignorelist
	sort.Slice(actualSkiplist, func(i, j int) bool {
		return actualSkiplist[i].Path < actualSkiplist[j].Path
	})
	sort.Slice(expectedSkiplist, func(i, j int) bool {
		return expectedSkiplist[i].Path < expectedSkiplist[j].Path
	})
	testutil.CheckErrorAndDeepEqual(t, false, err, expectedSkiplist, actualSkiplist)
}

func Test_AddToIgnoreList(t *testing.T) {
	t.Cleanup(func() {
		ignorelist = append([]IgnoreListEntry{}, defaultIgnoreList...)
	})

	AddToIgnoreList(IgnoreListEntry{
		Path:            "/tmp",
		PrefixMatchOnly: false,
	})

	if !CheckIgnoreList("/tmp") {
		t.Errorf("CheckIgnoreList() = %v, want %v", false, true)
	}
}

var tests = []struct {
	files         map[string]string
	directory     string
	expectedFiles []string
}{
	{
		files: map[string]string{
			"/workspace/foo/a": "baz1",
			"/workspace/foo/b": "baz2",
			"/kaniko/file":     "file",
		},
		directory: "/workspace/foo/",
		expectedFiles: []string{
			"workspace/foo/a",
			"workspace/foo/b",
			"workspace/foo",
		},
	},
	{
		files: map[string]string{
			"/workspace/foo/a": "baz1",
		},
		directory: "/workspace/foo/a",
		expectedFiles: []string{
			"workspace/foo/a",
		},
	},
	{
		files: map[string]string{
			"/workspace/foo/a": "baz1",
			"/workspace/foo/b": "baz2",
			"/workspace/baz":   "hey",
			"/kaniko/file":     "file",
		},
		directory: "/workspace",
		expectedFiles: []string{
			"workspace/foo/a",
			"workspace/foo/b",
			"workspace/baz",
			"workspace",
			"workspace/foo",
		},
	},
	{
		files: map[string]string{
			"/workspace/foo/a": "baz1",
			"/workspace/foo/b": "baz2",
		},
		directory: "",
		expectedFiles: []string{
			"workspace/foo/a",
			"workspace/foo/b",
			"workspace",
			"workspace/foo",
			".",
		},
	},
}

func Test_RelativeFiles(t *testing.T) {
	for _, test := range tests {
		testDir := t.TempDir()
		if err := testutil.SetupFiles(testDir, test.files); err != nil {
			t.Fatalf("err setting up files: %v", err)
		}
		actualFiles, err := RelativeFiles(test.directory, testDir)
		sort.Strings(actualFiles)
		sort.Strings(test.expectedFiles)
		testutil.CheckErrorAndDeepEqual(t, false, err, test.expectedFiles, actualFiles)
	}
}

func Test_ParentDirectories(t *testing.T) {
	tests := []struct {
		name     string
		path     string
		rootDir  string
		expected []string
	}{
		{
			name:    "regular path",
			path:    "/path/to/dir",
			rootDir: "/",
			expected: []string{
				"/",
				"/path",
				"/path/to",
			},
		},
		{
			name:    "current directory",
			path:    ".",
			rootDir: "/",
			expected: []string{
				"/",
			},
		},
		{
			name:    "non / root directory",
			path:    "/tmp/kaniko/test/another/dir",
			rootDir: "/tmp/kaniko/",
			expected: []string{
				"/tmp/kaniko",
				"/tmp/kaniko/test",
				"/tmp/kaniko/test/another",
			},
		},
		{
			name:    "non / root director same path",
			path:    "/tmp/123",
			rootDir: "/tmp/123",
			expected: []string{
				"/tmp/123",
			},
		},
		{
			name:    "non / root directory path",
			path:    "/tmp/120162240/kaniko",
			rootDir: "/tmp/120162240",
			expected: []string{
				"/tmp/120162240",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			original := config.RootDir
			defer func() { config.RootDir = original }()
			config.RootDir = tt.rootDir
			actual := ParentDirectories(tt.path)

			testutil.CheckErrorAndDeepEqual(t, false, nil, tt.expected, actual)
		})
	}
}

func Test_ParentDirectoriesWithoutLeadingSlash(t *testing.T) {
	tests := []struct {
		name     string
		path     string
		expected []string
	}{
		{
			name: "regular path",
			path: "/path/to/dir",
			expected: []string{
				"/",
				"path",
				"path/to",
			},
		},
		{
			name: "current directory",
			path: ".",
			expected: []string{
				"/",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			actual := ParentDirectoriesWithoutLeadingSlash(tt.path)
			testutil.CheckErrorAndDeepEqual(t, false, nil, tt.expected, actual)
		})
	}
}

func Test_CheckIgnoreList(t *testing.T) {
	type args struct {
		path       string
		ignorelist []IgnoreListEntry
	}
	tests := []struct {
		name string
		args args
		want bool
	}{
		{
			name: "file ignored",
			args: args{
				path:       "/foo",
				ignorelist: []IgnoreListEntry{{"/foo", false}},
			},
			want: true,
		},
		{
			name: "directory ignored",
			args: args{
				path:       "/foo/bar",
				ignorelist: []IgnoreListEntry{{"/foo", false}},
			},
			want: true,
		},
		{
			name: "grandparent ignored",
			args: args{
				path:       "/foo/bar/baz",
				ignorelist: []IgnoreListEntry{{"/foo", false}},
			},
			want: true,
		},
		{
			name: "sibling ignored",
			args: args{
				path:       "/foo/bar/baz",
				ignorelist: []IgnoreListEntry{{"/foo/bat", false}},
			},
			want: false,
		},
		{
			name: "prefix match only ",
			args: args{
				path:       "/tmp/apt-key-gpghome.xft/gpg.key",
				ignorelist: []IgnoreListEntry{{"/tmp/apt-key-gpghome.*", true}},
			},
			want: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			original := ignorelist
			defer func() {
				ignorelist = original
			}()
			ignorelist = tt.args.ignorelist
			got := CheckIgnoreList(tt.args.path)
			if got != tt.want {
				t.Errorf("CheckIgnoreList() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestHasFilepathPrefix(t *testing.T) {
	type args struct {
		path            string
		prefix          string
		prefixMatchOnly bool
	}
	tests := []struct {
		name string
		args args
		want bool
	}{
		{
			name: "parent",
			args: args{
				path:            "/foo/bar",
				prefix:          "/foo",
				prefixMatchOnly: false,
			},
			want: true,
		},
		{
			name: "nested parent",
			args: args{
				path:            "/foo/bar/baz",
				prefix:          "/foo/bar",
				prefixMatchOnly: false,
			},
			want: true,
		},
		{
			name: "sibling",
			args: args{
				path:            "/foo/bar",
				prefix:          "/bar",
				prefixMatchOnly: false,
			},
			want: false,
		},
		{
			name: "nested sibling",
			args: args{
				path:            "/foo/bar/baz",
				prefix:          "/foo/bar",
				prefixMatchOnly: false,
			},
			want: true,
		},
		{
			name: "name prefix",
			args: args{
				path:            "/foo2/bar",
				prefix:          "/foo",
				prefixMatchOnly: false,
			},
			want: false,
		},
		{
			name: "prefix match only (volume)",
			args: args{
				path:            "/foo",
				prefix:          "/foo",
				prefixMatchOnly: true,
			},
			want: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := HasFilepathPrefix(tt.args.path, tt.args.prefix, tt.args.prefixMatchOnly); got != tt.want {
				t.Errorf("HasFilepathPrefix() = %v, want %v", got, tt.want)
			}
		})
	}
}

func BenchmarkHasFilepathPrefix(b *testing.B) {
	tests := []struct {
		path            string
		prefix          string
		prefixMatchOnly bool
	}{
		{
			path:            "/foo/bar",
			prefix:          "/foo",
			prefixMatchOnly: true,
		},
		{
			path:            "/foo/bar/baz",
			prefix:          "/foo",
			prefixMatchOnly: true,
		},
		{
			path:            "/foo/bar/baz/foo",
			prefix:          "/foo",
			prefixMatchOnly: true,
		},
		{
			path:            "/foo/bar/baz/foo/foobar",
			prefix:          "/foo",
			prefixMatchOnly: true,
		},
		{
			path:            "/foo/bar",
			prefix:          "/foo/bar",
			prefixMatchOnly: true,
		},
		{
			path:            "/foo/bar/baz",
			prefix:          "/foo/bar",
			prefixMatchOnly: true,
		},
		{
			path:            "/foo/bar/baz/foo",
			prefix:          "/foo/bar",
			prefixMatchOnly: true,
		},
		{
			path:            "/foo/bar/baz/foo/foobar",
			prefix:          "/foo/bar",
			prefixMatchOnly: true,
		},
		{
			path:            "/foo/bar",
			prefix:          "/foo/bar/baz",
			prefixMatchOnly: true,
		},
		{
			path:            "/foo/bar/baz",
			prefix:          "/foo/bar/baz",
			prefixMatchOnly: true,
		},
		{
			path:            "/foo/bar/baz/foo",
			prefix:          "/foo/bar/baz",
			prefixMatchOnly: true,
		},
		{
			path:            "/foo/bar/baz/foo/foobar",
			prefix:          "/foo/bar/baz",
			prefixMatchOnly: true,
		},
	}
	for _, ts := range tests {
		name := fmt.Sprint("PathDepth=", strings.Count(ts.path, "/"), ",PrefixDepth=", strings.Count(ts.prefix, "/"))
		b.Run(name, func(b *testing.B) {
			b.ReportAllocs()
			for i := 0; i < b.N; i++ {
				HasFilepathPrefix(ts.path, ts.prefix, ts.prefixMatchOnly)
			}
		})
	}
}

type checker func(root string, t *testing.T)

func fileExists(p string) checker {
	return func(root string, t *testing.T) {
		_, err := os.Stat(filepath.Join(root, p))
		if err != nil {
			t.Fatalf("File %s does not exist", filepath.Join(root, p))
		}
	}
}

func fileMatches(p string, c []byte) checker {
	return func(root string, t *testing.T) {
		actual, err := os.ReadFile(filepath.Join(root, p))
		if err != nil {
			t.Fatalf("error reading file: %s", p)
		}
		if !reflect.DeepEqual(actual, c) {
			t.Errorf("file contents do not match. %v!=%v", actual, c)
		}
	}
}

func timesMatch(p string, fTime time.Time) checker {
	return func(root string, t *testing.T) {
		fi, err := os.Stat(filepath.Join(root, p))
		if err != nil {
			t.Fatalf("error statting file %s", p)
		}

		if fi.ModTime().UTC() != fTime.UTC() {
			t.Errorf("Expected modtime to equal %v but was %v", fTime, fi.ModTime())
		}
	}
}

func permissionsMatch(p string, perms os.FileMode) checker {
	return func(root string, t *testing.T) {
		fi, err := os.Stat(filepath.Join(root, p))
		if err != nil {
			t.Fatalf("error statting file %s", p)
		}
		if fi.Mode() != perms {
			t.Errorf("Permissions do not match. %s != %s", fi.Mode(), perms)
		}
	}
}

func linkPointsTo(src, dst string) checker {
	return func(root string, t *testing.T) {
		link := filepath.Join(root, src)
		got, err := os.Readlink(link)
		if err != nil {
			t.Fatalf("error reading link %s: %s", link, err)
		}
		if got != dst {
			t.Errorf("link destination does not match: %s != %s", got, dst)
		}
	}
}

func filesAreHardlinks(first, second string) checker {
	return func(root string, t *testing.T) {
		fi1, err := os.Stat(filepath.Join(root, first))
		if err != nil {
			t.Fatalf("error getting file %s", first)
		}
		fi2, err := os.Stat(filepath.Join(root, second))
		if err != nil {
			t.Fatalf("error getting file %s", second)
		}
		stat1 := getSyscallStatT(fi1)
		stat2 := getSyscallStatT(fi2)
		if stat1.Ino != stat2.Ino {
			t.Errorf("%s and %s aren't hardlinks as they dont' have the same inode", first, second)
		}
	}
}

func fileHeader(name string, contents string, mode int64, fTime time.Time) *tar.Header {
	return &tar.Header{
		Name:       name,
		Size:       int64(len(contents)),
		Mode:       mode,
		Typeflag:   tar.TypeReg,
		Uid:        os.Getuid(),
		Gid:        os.Getgid(),
		AccessTime: fTime,
		ModTime:    fTime,
	}
}

func linkHeader(name, linkname string) *tar.Header {
	return &tar.Header{
		Name:     name,
		Size:     0,
		Typeflag: tar.TypeSymlink,
		Linkname: linkname,
	}
}

func hardlinkHeader(name, linkname string) *tar.Header {
	return &tar.Header{
		Name:     name,
		Size:     0,
		Typeflag: tar.TypeLink,
		Linkname: linkname,
	}
}

func dirHeader(name string, mode int64) *tar.Header {
	return &tar.Header{
		Name:     name,
		Size:     0,
		Typeflag: tar.TypeDir,
		Mode:     mode,
		Uid:      os.Getuid(),
		Gid:      os.Getgid(),
	}
}

func createUncompressedTar(fileContents map[string]string, tarFileName, testDir string) error {
	if err := testutil.SetupFiles(testDir, fileContents); err != nil {
		return err
	}
	tarFile, err := os.Create(filepath.Join(testDir, tarFileName))
	if err != nil {
		return err
	}
	t := NewTar(tarFile)
	defer t.Close()
	for file := range fileContents {
		filePath := filepath.Join(testDir, file)
		if err := t.AddFileToTar(filePath); err != nil {
			return err
		}
	}
	return nil
}

func Test_UnTar(t *testing.T) {
	tcs := []struct {
		name             string
		setupTarContents map[string]string
		tarFileName      string
		destination      string
		expectedFileList []string
		errorExpected    bool
	}{
		{
			name: "multfile tar",
			setupTarContents: map[string]string{
				"foo/file1": "hello World",
				"bar/file1": "hello World",
				"bar/file2": "hello World",
				"file1":     "hello World",
			},
			tarFileName:      "test.tar",
			destination:      "/",
			expectedFileList: []string{"foo/file1", "bar/file1", "bar/file2", "file1"},
			errorExpected:    false,
		},
		{
			name:             "empty tar",
			setupTarContents: map[string]string{},
			tarFileName:      "test.tar",
			destination:      "/",
			expectedFileList: nil,
			errorExpected:    false,
		},
	}
	for _, tc := range tcs {
		t.Run(tc.name, func(t *testing.T) {
			testDir := t.TempDir()
			if err := createUncompressedTar(tc.setupTarContents, tc.tarFileName, testDir); err != nil {
				t.Fatal(err)
			}
			file, err := os.Open(filepath.Join(testDir, tc.tarFileName))
			if err != nil {
				t.Fatal(err)
			}
			fileList, err := UnTar(file, tc.destination)
			if err != nil {
				t.Fatal(err)
			}
			// update expectedFileList to take into factor temp directory
			for i, file := range tc.expectedFileList {
				tc.expectedFileList[i] = filepath.Join(testDir, file)
			}
			// sort both slices to ensure objects are in the same order for deep equals
			sort.Strings(tc.expectedFileList)
			sort.Strings(fileList)
			testutil.CheckErrorAndDeepEqual(t, tc.errorExpected, err, tc.expectedFileList, fileList)
		})
	}
}

func TestExtractFile(t *testing.T) {
	type tc struct {
		name     string
		hdrs     []*tar.Header
		tmpdir   string
		contents []byte
		checkers []checker
	}

	defaultTestTime, err := time.Parse(time.RFC3339, "1912-06-23T00:00:00Z")
	if err != nil {
		t.Fatal(err)
	}

	tcs := []tc{
		{
			name:     "normal file",
			contents: []byte("helloworld"),
			hdrs:     []*tar.Header{fileHeader("./bar", "helloworld", 0o644, defaultTestTime)},
			checkers: []checker{
				fileExists("/bar"),
				fileMatches("/bar", []byte("helloworld")),
				permissionsMatch("/bar", 0o644),
				timesMatch("/bar", defaultTestTime),
			},
		},
		{
			name:     "normal file, directory does not exist",
			contents: []byte("helloworld"),
			hdrs:     []*tar.Header{fileHeader("./foo/bar", "helloworld", 0o644, defaultTestTime)},
			checkers: []checker{
				fileExists("/foo/bar"),
				fileMatches("/foo/bar", []byte("helloworld")),
				permissionsMatch("/foo/bar", 0o644),
				permissionsMatch("/foo", 0o755|os.ModeDir),
			},
		},
		{
			name:     "normal file, directory is created after",
			contents: []byte("helloworld"),
			hdrs: []*tar.Header{
				fileHeader("./foo/bar", "helloworld", 0o644, defaultTestTime),
				dirHeader("./foo", 0o722),
			},
			checkers: []checker{
				fileExists("/foo/bar"),
				fileMatches("/foo/bar", []byte("helloworld")),
				permissionsMatch("/foo/bar", 0o644),
				permissionsMatch("/foo", 0o722|os.ModeDir),
			},
		},
		{
			name: "symlink",
			hdrs: []*tar.Header{linkHeader("./bar", "bar/bat")},
			checkers: []checker{
				linkPointsTo("/bar", "bar/bat"),
			},
		},
		{
			name: "symlink relative path",
			hdrs: []*tar.Header{linkHeader("./bar", "./foo/bar/baz")},
			checkers: []checker{
				linkPointsTo("/bar", "./foo/bar/baz"),
			},
		},
		{
			name: "symlink parent does not exist",
			hdrs: []*tar.Header{linkHeader("./foo/bar/baz", "../../bat")},
			checkers: []checker{
				linkPointsTo("/foo/bar/baz", "../../bat"),
			},
		},
		{
			name: "symlink parent does not exist 2",
			hdrs: []*tar.Header{linkHeader("./foo/bar/baz", "../../bat")},
			checkers: []checker{
				linkPointsTo("/foo/bar/baz", "../../bat"),
				permissionsMatch("/foo", 0o755|os.ModeDir),
				permissionsMatch("/foo/bar", 0o755|os.ModeDir),
			},
		},
		{
			name:   "hardlink",
			tmpdir: "/tmp/hardlink",
			hdrs: []*tar.Header{
				fileHeader("/bin/gzip", "gzip-binary", 0o751, defaultTestTime),
				hardlinkHeader("/bin/uncompress", "/bin/gzip"),
			},
			checkers: []checker{
				fileExists("/bin/gzip"),
				filesAreHardlinks("/bin/uncompress", "/bin/gzip"),
			},
		},
		{
			name:     "file with setuid bit",
			contents: []byte("helloworld"),
			hdrs:     []*tar.Header{fileHeader("./bar", "helloworld", 0o4644, defaultTestTime)},
			checkers: []checker{
				fileExists("/bar"),
				fileMatches("/bar", []byte("helloworld")),
				permissionsMatch("/bar", 0o644|os.ModeSetuid),
			},
		},
		{
			name:     "dir with sticky bit",
			contents: []byte("helloworld"),
			hdrs: []*tar.Header{
				dirHeader("./foo", 0o1755),
				fileHeader("./foo/bar", "helloworld", 0o644, defaultTestTime),
			},
			checkers: []checker{
				fileExists("/foo/bar"),
				fileMatches("/foo/bar", []byte("helloworld")),
				permissionsMatch("/foo/bar", 0o644),
				permissionsMatch("/foo", 0o755|os.ModeDir|os.ModeSticky),
			},
		},
	}

	for _, tc := range tcs {
		t.Run(tc.name, func(t *testing.T) {
			tc := tc
			t.Parallel()
			r := ""

			if tc.tmpdir != "" {
				r = tc.tmpdir
			} else {
				r = t.TempDir()
			}
			defer os.RemoveAll(r)

			for _, hdr := range tc.hdrs {
				if err := ExtractFile(r, hdr, filepath.Clean(hdr.Name), bytes.NewReader(tc.contents)); err != nil {
					t.Fatal(err)
				}
			}
			for _, checker := range tc.checkers {
				checker(r, t)
			}
		})
	}
}

func TestCopySymlink(t *testing.T) {
	type tc struct {
		name       string
		linkTarget string
		dest       string
		beforeLink func(r string) error
	}

	tcs := []tc{{
		name:       "absolute symlink",
		linkTarget: "/abs/dest",
	}, {
		name:       "relative symlink",
		linkTarget: "rel",
	}, {
		name:       "symlink copy overwrites existing file",
		linkTarget: "/abs/dest",
		dest:       "overwrite_me",
		beforeLink: func(r string) error {
			return os.WriteFile(filepath.Join(r, "overwrite_me"), nil, 0o644)
		},
	}}

	for _, tc := range tcs {
		t.Run(tc.name, func(t *testing.T) {
			tc := tc
			t.Parallel()
			r := t.TempDir()
			os.MkdirAll(filepath.Join(r, filepath.Dir(tc.linkTarget)), 0o777)
			tc.linkTarget = filepath.Join(r, tc.linkTarget)
			os.WriteFile(tc.linkTarget, nil, 0o644)

			if tc.beforeLink != nil {
				if err := tc.beforeLink(r); err != nil {
					t.Fatal(err)
				}
			}
			link := filepath.Join(r, "link")
			dest := filepath.Join(r, "copy")
			if tc.dest != "" {
				dest = filepath.Join(r, tc.dest)
			}
			if err := os.Symlink(tc.linkTarget, link); err != nil {
				t.Fatal(err)
			}
			if _, err := CopySymlink(link, dest, FileContext{}); err != nil {
				t.Fatal(err)
			}
			if _, err := os.Lstat(dest); err != nil {
				t.Fatalf("error reading link %s: %s", link, err)
			}
		})
	}
}

func Test_childDirInSkiplist(t *testing.T) {
	type args struct {
		path       string
		ignorelist []IgnoreListEntry
	}
	tests := []struct {
		name string
		args args
		want bool
	}{
		{
			name: "not in ignorelist",
			args: args{
				path: "/foo",
			},
			want: false,
		},
		{
			name: "child in ignorelist",
			args: args{
				path: "/foo",
				ignorelist: []IgnoreListEntry{
					{
						Path: "/foo/bar",
					},
				},
			},
			want: true,
		},
	}
	oldIgnoreList := ignorelist
	defer func() {
		ignorelist = oldIgnoreList
	}()

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ignorelist = tt.args.ignorelist
			if got := childDirInIgnoreList(tt.args.path); got != tt.want {
				t.Errorf("childDirInIgnoreList() = %v, want %v", got, tt.want)
			}
		})
	}
}

func Test_correctDockerignoreFileIsUsed(t *testing.T) {
	type args struct {
		dockerfilepath string
		buildcontext   string
		excluded       []string
		included       []string
	}
	tests := []struct {
		name string
		args args
	}{
		{
			name: "relative dockerfile used",
			args: args{
				dockerfilepath: "../../integration/dockerfiles/Dockerfile_dockerignore_relative",
				buildcontext:   "../../integration/",
				excluded:       []string{"ignore_relative/bar"},
				included:       []string{"ignore_relative/foo", "ignore/bar"},
			},
		},
		{
			name: "context dockerfile is used",
			args: args{
				dockerfilepath: "../../integration/dockerfiles/Dockerfile_test_dockerignore",
				buildcontext:   "../../integration/",
				excluded:       []string{"ignore/bar"},
				included:       []string{"ignore/foo", "ignore_relative/bar"},
			},
		},
	}
	for _, tt := range tests {
		fileContext, err := NewFileContextFromDockerfile(tt.args.dockerfilepath, tt.args.buildcontext)
		if err != nil {
			t.Fatal(err)
		}
		for _, excl := range tt.args.excluded {
			t.Run(tt.name+" to exclude "+excl, func(t *testing.T) {
				if !fileContext.ExcludesFile(excl) {
					t.Errorf("'%v' not excluded", excl)
				}
			})
		}
		for _, incl := range tt.args.included {
			t.Run(tt.name+" to include "+incl, func(t *testing.T) {
				if fileContext.ExcludesFile(incl) {
					t.Errorf("'%v' not included", incl)
				}
			})
		}
	}
}

func Test_CopyFile_skips_self(t *testing.T) {
	t.Parallel()
	tempDir := t.TempDir()

	tempFile := filepath.Join(tempDir, "foo")
	expected := "bar"

	if err := os.WriteFile(
		tempFile,
		[]byte(expected),
		0o755,
	); err != nil {
		t.Fatal(err)
	}

	ignored, err := CopyFile(tempFile, tempFile, FileContext{}, DoNotChangeUID, DoNotChangeGID, fs.FileMode(0o600), true)
	if err != nil {
		t.Fatal(err)
	}

	if ignored {
		t.Fatal("expected file to NOT be ignored")
	}

	// Ensure file has expected contents
	actualData, err := os.ReadFile(tempFile)
	if err != nil {
		t.Fatal(err)
	}

	if actual := string(actualData); actual != expected {
		t.Fatalf("expected file contents to be %q, but got %q", expected, actual)
	}
}

func fakeExtract(_ string, _ *tar.Header, _ string, _ io.Reader) error {
	return nil
}

func Test_GetFSFromLayers_with_whiteouts_include_whiteout_enabled(t *testing.T) {
	resetMountInfoFile := provideEmptyMountinfoFile()
	defer resetMountInfoFile()

	ctrl := gomock.NewController(t)

	root := t.TempDir()
	// Write a whiteout path
	d1 := []byte("Hello World\n")
	if err := os.WriteFile(filepath.Join(root, "foobar"), d1, 0o644); err != nil {
		t.Fatal(err)
	}

	opts := []FSOpt{
		// I'd rather use the real func (util.ExtractFile)
		// but you have to be root to chown
		ExtractFunc(fakeExtract),
		IncludeWhiteout(),
	}

	expectErr := false

	f := func(expectedFiles []string, tw *tar.Writer) {
		for _, f := range expectedFiles {
			f := strings.TrimPrefix(strings.TrimPrefix(f, root), "/")

			hdr := &tar.Header{
				Name: f,
				Mode: 0o644,
				Size: int64(len("Hello World\n")),
			}

			if err := tw.WriteHeader(hdr); err != nil {
				t.Fatal(err)
			}

			if _, err := tw.Write([]byte("Hello World\n")); err != nil {
				t.Fatal(err)
			}
		}

		if err := tw.Close(); err != nil {
			t.Fatal(err)
		}
	}

	expectedFiles := []string{
		filepath.Join(root, "foobar"),
	}

	buf := new(bytes.Buffer)
	tw := tar.NewWriter(buf)

	f(expectedFiles, tw)

	mockLayer := mockv1.NewMockLayer(ctrl)
	mockLayer.EXPECT().MediaType().Return(types.OCILayer, nil)

	rc := io.NopCloser(buf)
	mockLayer.EXPECT().Uncompressed().Return(rc, nil)

	secondLayerFiles := []string{
		filepath.Join(root, ".wh.foobar"),
	}

	buf = new(bytes.Buffer)
	tw = tar.NewWriter(buf)

	f(secondLayerFiles, tw)

	mockLayer2 := mockv1.NewMockLayer(ctrl)
	mockLayer2.EXPECT().MediaType().Return(types.OCILayer, nil)

	rc = io.NopCloser(buf)
	mockLayer2.EXPECT().Uncompressed().Return(rc, nil)

	layers := []v1.Layer{
		mockLayer,
		mockLayer2,
	}

	expectedFiles = append(expectedFiles, secondLayerFiles...)

	actualFiles, err := GetFSFromLayers(root, layers, opts...)

	assertGetFSFromLayers(
		t,
		actualFiles,
		expectedFiles,
		err,
		expectErr,
	)
	// Make sure whiteout files are removed form the root.
	_, err = os.Lstat(filepath.Join(root, "foobar"))
	if err == nil || !os.IsNotExist(err) {
		t.Errorf("expected whiteout foobar file to be deleted. However found it.")
	}
}

func provideEmptyMountinfoFile() func() {
	// Provide empty mountinfo file to prevent /tmp from ending up in ignore list on
	// distributions with /tmp mountpoint. Otherwise, tests expecting operations in /tmp
	// can fail.
	config.MountInfoPath = "/dev/null"
	return func() {
		config.MountInfoPath = constants.MountInfoPath
	}
}

func Test_GetFSFromLayers_with_whiteouts_include_whiteout_disabled(t *testing.T) {
	resetMountInfoFile := provideEmptyMountinfoFile()
	defer resetMountInfoFile()

	ctrl := gomock.NewController(t)

	root := t.TempDir()
	// Write a whiteout path
	d1 := []byte("Hello World\n")
	if err := os.WriteFile(filepath.Join(root, "foobar"), d1, 0o644); err != nil {
		t.Fatal(err)
	}

	opts := []FSOpt{
		// I'd rather use the real func (util.ExtractFile)
		// but you have to be root to chown
		ExtractFunc(fakeExtract),
	}

	expectErr := false

	f := func(expectedFiles []string, tw *tar.Writer) {
		for _, f := range expectedFiles {
			f := strings.TrimPrefix(strings.TrimPrefix(f, root), "/")

			hdr := &tar.Header{
				Name: f,
				Mode: 0o644,
				Size: int64(len("Hello world\n")),
			}

			if err := tw.WriteHeader(hdr); err != nil {
				t.Fatal(err)
			}

			if _, err := tw.Write([]byte("Hello world\n")); err != nil {
				t.Fatal(err)
			}
		}

		if err := tw.Close(); err != nil {
			t.Fatal(err)
		}
	}

	expectedFiles := []string{
		filepath.Join(root, "foobar"),
	}

	buf := new(bytes.Buffer)
	tw := tar.NewWriter(buf)

	f(expectedFiles, tw)

	mockLayer := mockv1.NewMockLayer(ctrl)
	mockLayer.EXPECT().MediaType().Return(types.OCILayer, nil)
	layerFiles := []string{
		filepath.Join(root, "foobar"),
	}
	buf = new(bytes.Buffer)
	tw = tar.NewWriter(buf)

	f(layerFiles, tw)

	rc := io.NopCloser(buf)
	mockLayer.EXPECT().Uncompressed().Return(rc, nil)

	secondLayerFiles := []string{
		filepath.Join(root, ".wh.foobar"),
	}

	buf = new(bytes.Buffer)
	tw = tar.NewWriter(buf)

	f(secondLayerFiles, tw)

	mockLayer2 := mockv1.NewMockLayer(ctrl)
	mockLayer2.EXPECT().MediaType().Return(types.OCILayer, nil)

	rc = io.NopCloser(buf)
	mockLayer2.EXPECT().Uncompressed().Return(rc, nil)

	layers := []v1.Layer{
		mockLayer,
		mockLayer2,
	}

	actualFiles, err := GetFSFromLayers(root, layers, opts...)

	assertGetFSFromLayers(
		t,
		actualFiles,
		expectedFiles,
		err,
		expectErr,
	)
	// Make sure whiteout files are removed form the root.
	_, err = os.Lstat(filepath.Join(root, "foobar"))
	if err == nil || !os.IsNotExist(err) {
		t.Errorf("expected whiteout foobar file to be deleted. However found it.")
	}
}

func Test_GetFSFromLayers_ignorelist(t *testing.T) {
	resetMountInfoFile := provideEmptyMountinfoFile()
	defer resetMountInfoFile()

	ctrl := gomock.NewController(t)

	root := t.TempDir()
	// Write a whiteout path
	fileContents := []byte("Hello World\n")
	if err := os.Mkdir(filepath.Join(root, "testdir"), 0o775); err != nil {
		t.Fatal(err)
	}

	opts := []FSOpt{
		// I'd rather use the real func (util.ExtractFile)
		// but you have to be root to chown
		ExtractFunc(fakeExtract),
		IncludeWhiteout(),
	}

	f := func(expectedFiles []string, tw *tar.Writer) {
		for _, f := range expectedFiles {
			f := strings.TrimPrefix(strings.TrimPrefix(f, root), "/")

			hdr := &tar.Header{
				Name: f,
				Mode: 0o644,
				Size: int64(len(string(fileContents))),
			}

			if err := tw.WriteHeader(hdr); err != nil {
				t.Fatal(err)
			}

			if _, err := tw.Write(fileContents); err != nil {
				t.Fatal(err)
			}
		}

		if err := tw.Close(); err != nil {
			t.Fatal(err)
		}
	}

	// first, testdir is not in ignorelist, so it should be deleted
	expectedFiles := []string{
		filepath.Join(root, ".wh.testdir"),
		filepath.Join(root, "testdir", "file"),
		filepath.Join(root, "other-file"),
	}

	buf := new(bytes.Buffer)
	tw := tar.NewWriter(buf)

	f(expectedFiles, tw)

	mockLayer := mockv1.NewMockLayer(ctrl)
	mockLayer.EXPECT().MediaType().Return(types.OCILayer, nil)
	layerFiles := []string{
		filepath.Join(root, ".wh.testdir"),
		filepath.Join(root, "testdir", "file"),
		filepath.Join(root, "other-file"),
	}
	buf = new(bytes.Buffer)
	tw = tar.NewWriter(buf)

	f(layerFiles, tw)

	rc := io.NopCloser(buf)
	mockLayer.EXPECT().Uncompressed().Return(rc, nil)

	layers := []v1.Layer{
		mockLayer,
	}

	actualFiles, err := GetFSFromLayers(root, layers, opts...)
	assertGetFSFromLayers(
		t,
		actualFiles,
		expectedFiles,
		err,
		false,
	)

	// Make sure whiteout files are removed form the root.
	_, err = os.Lstat(filepath.Join(root, "testdir"))
	if err == nil || !os.IsNotExist(err) {
		t.Errorf("expected testdir to be deleted. However found it.")
	}

	// second, testdir is in ignorelist, so it should not be deleted
	original := append([]IgnoreListEntry{}, defaultIgnoreList...)
	defer func() {
		defaultIgnoreList = original
	}()
	defaultIgnoreList = append(defaultIgnoreList, IgnoreListEntry{
		Path: filepath.Join(root, "testdir"),
	})
	if err := os.Mkdir(filepath.Join(root, "testdir"), 0o775); err != nil {
		t.Fatal(err)
	}

	expectedFiles = []string{
		filepath.Join(root, "other-file"),
	}

	buf = new(bytes.Buffer)
	tw = tar.NewWriter(buf)

	f(expectedFiles, tw)

	mockLayer = mockv1.NewMockLayer(ctrl)
	mockLayer.EXPECT().MediaType().Return(types.OCILayer, nil)
	layerFiles = []string{
		filepath.Join(root, ".wh.testdir"),
		filepath.Join(root, "other-file"),
	}
	buf = new(bytes.Buffer)
	tw = tar.NewWriter(buf)

	f(layerFiles, tw)

	rc = io.NopCloser(buf)
	mockLayer.EXPECT().Uncompressed().Return(rc, nil)

	layers = []v1.Layer{
		mockLayer,
	}

	actualFiles, err = GetFSFromLayers(root, layers, opts...)
	assertGetFSFromLayers(
		t,
		actualFiles,
		expectedFiles,
		err,
		false,
	)

	// Make sure testdir still exists.
	_, err = os.Lstat(filepath.Join(root, "testdir"))
	if err != nil {
		t.Errorf("expected testdir to exist, but could not Lstat it: %v", err)
	}
}

func Test_GetFSFromLayers(t *testing.T) {
	ctrl := gomock.NewController(t)

	root := t.TempDir()

	opts := []FSOpt{
		// I'd rather use the real func (util.ExtractFile)
		// but you have to be root to chown
		ExtractFunc(fakeExtract),
	}

	expectErr := false
	expectedFiles := []string{
		filepath.Join(root, "foobar"),
	}

	buf := new(bytes.Buffer)
	tw := tar.NewWriter(buf)

	for _, f := range expectedFiles {
		f := strings.TrimPrefix(strings.TrimPrefix(f, root), "/")

		hdr := &tar.Header{
			Name: f,
			Mode: 0o644,
			Size: int64(len("Hello world\n")),
		}

		if err := tw.WriteHeader(hdr); err != nil {
			t.Fatal(err)
		}

		if _, err := tw.Write([]byte("Hello world\n")); err != nil {
			t.Fatal(err)
		}
	}

	if err := tw.Close(); err != nil {
		t.Fatal(err)
	}

	mockLayer := mockv1.NewMockLayer(ctrl)
	mockLayer.EXPECT().MediaType().Return(types.OCILayer, nil)

	rc := io.NopCloser(buf)
	mockLayer.EXPECT().Uncompressed().Return(rc, nil)

	layers := []v1.Layer{
		mockLayer,
	}

	actualFiles, err := GetFSFromLayers(root, layers, opts...)

	assertGetFSFromLayers(
		t,
		actualFiles,
		expectedFiles,
		err,
		expectErr,
	)
}

func assertGetFSFromLayers(
	t *testing.T,
	actualFiles []string,
	expectedFiles []string,
	err error,
	expectErr bool, //nolint:unparam
) {
	t.Helper()
	if !expectErr && err != nil {
		t.Error(err)
		t.FailNow()
	} else if expectErr && err == nil {
		t.Error("expected err to not be nil")
		t.FailNow()
	}

	if len(actualFiles) != len(expectedFiles) {
		t.Errorf("expected %s to equal %s", actualFiles, expectedFiles)
		t.FailNow()
	}

	for i := range expectedFiles {
		if actualFiles[i] != expectedFiles[i] {
			t.Errorf("expected %s to equal %s", actualFiles[i], expectedFiles[i])
		}
	}
}

func TestInitIgnoreList(t *testing.T) {
	mountInfo := `36 35 98:0 /kaniko /test/kaniko rw,noatime master:1 - ext3 /dev/root rw,errors=continue
36 35 98:0 /proc /test/proc rw,noatime master:1 - ext3 /dev/root rw,errors=continue
`
	mFile, err := os.CreateTemp("", "mountinfo")
	if err != nil {
		t.Fatal(err)
	}
	defer mFile.Close()
	if _, err := mFile.WriteString(mountInfo); err != nil {
		t.Fatal(err)
	}
	config.MountInfoPath = mFile.Name()
	defer func() {
		config.MountInfoPath = constants.MountInfoPath
	}()

	expected := []IgnoreListEntry{
		{
			Path:            "/kaniko",
			PrefixMatchOnly: false,
		},
		{
			Path:            "/test/kaniko",
			PrefixMatchOnly: false,
		},
		{
			Path:            "/test/proc",
			PrefixMatchOnly: false,
		},
		{
			Path:            "/etc/mtab",
			PrefixMatchOnly: false,
		},
		{
			Path:            "/tmp/apt-key-gpghome",
			PrefixMatchOnly: true,
		},
	}

	original := append([]IgnoreListEntry{}, ignorelist...)
	defer func() { ignorelist = original }()

	err = InitIgnoreList()
	if err != nil {
		t.Fatal(err)
	}
	sort.Slice(expected, func(i, j int) bool {
		return expected[i].Path < expected[j].Path
	})
	sort.Slice(ignorelist, func(i, j int) bool {
		return ignorelist[i].Path < ignorelist[j].Path
	})
	testutil.CheckDeepEqual(t, expected, ignorelist)
}

func Test_setFileTimes(t *testing.T) {
	testDir := t.TempDir()

	p := filepath.Join(testDir, "foo.txt")

	if err := os.WriteFile(p, []byte("meow"), 0o777); err != nil {
		t.Fatal(err)
	}

	type testcase struct {
		desc  string
		path  string
		aTime time.Time
		mTime time.Time
	}

	testCases := []testcase{
		{
			desc: "zero for mod and access",
			path: p,
		},
		{
			desc:  "zero for mod",
			path:  p,
			aTime: time.Now(),
		},
		{
			desc:  "zero for access",
			path:  p,
			mTime: time.Now(),
		},
		{
			desc:  "both non-zero",
			path:  p,
			mTime: time.Now(),
			aTime: time.Now(),
		},
	}

	for _, tc := range testCases {
		t.Run(tc.desc, func(t *testing.T) {
			err := setFileTimes(tc.path, tc.aTime, tc.mTime)
			if err != nil {
				t.Errorf("expected err to be nil not %s", err)
			}
		})
	}
}
