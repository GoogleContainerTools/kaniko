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
	"io/ioutil"
	"os"
	"path/filepath"
	"reflect"
	"sort"
	"strings"
	"testing"

	"github.com/GoogleContainerTools/kaniko/testutil"
)

func Test_DetectFilesystemWhitelist(t *testing.T) {
	testDir, err := ioutil.TempDir("", "")
	if err != nil {
		t.Fatalf("Error creating tempdir: %s", err)
	}
	fileContents := `
	228 122 0:90 / / rw,relatime - aufs none rw,si=f8e2406af90782bc,dio,dirperm1
	229 228 0:98 / /proc rw,nosuid,nodev,noexec,relatime - proc proc rw
	230 228 0:99 / /dev rw,nosuid - tmpfs tmpfs rw,size=65536k,mode=755
	231 230 0:100 / /dev/pts rw,nosuid,noexec,relatime - devpts devpts rw,gid=5,mode=620,ptmxmode=666
	232 228 0:101 / /sys ro,nosuid,nodev,noexec,relatime - sysfs sysfs ro`

	path := filepath.Join(testDir, "mountinfo")
	if err := os.MkdirAll(filepath.Dir(path), 0750); err != nil {
		t.Fatalf("Error creating tempdir: %s", err)
	}
	if err := ioutil.WriteFile(path, []byte(fileContents), 0644); err != nil {
		t.Fatalf("Error writing file contents to %s: %s", path, err)
	}

	err = DetectFilesystemWhitelist(path)
	expectedWhitelist := []WhitelistEntry{
		{"/kaniko", false},
		{"/proc", false},
		{"/dev", false},
		{"/dev/pts", false},
		{"/sys", false},
		{"/var/run", false},
		{"/etc/mtab", false},
	}
	actualWhitelist := whitelist
	sort.Slice(actualWhitelist, func(i, j int) bool {
		return actualWhitelist[i].Path < actualWhitelist[j].Path
	})
	sort.Slice(expectedWhitelist, func(i, j int) bool {
		return expectedWhitelist[i].Path < expectedWhitelist[j].Path
	})
	testutil.CheckErrorAndDeepEqual(t, false, err, expectedWhitelist, actualWhitelist)
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
		testDir, err := ioutil.TempDir("", "")
		if err != nil {
			t.Fatalf("err setting up temp dir: %v", err)
		}
		defer os.RemoveAll(testDir)
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
		expected []string
	}{
		{
			name: "regular path",
			path: "/path/to/dir",
			expected: []string{
				"/",
				"/path",
				"/path/to",
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

func Test_CheckWhitelist(t *testing.T) {
	type args struct {
		path      string
		whitelist []WhitelistEntry
	}
	tests := []struct {
		name string
		args args
		want bool
	}{
		{
			name: "file whitelisted",
			args: args{
				path:      "/foo",
				whitelist: []WhitelistEntry{{"/foo", false}},
			},
			want: true,
		},
		{
			name: "directory whitelisted",
			args: args{
				path:      "/foo/bar",
				whitelist: []WhitelistEntry{{"/foo", false}},
			},
			want: true,
		},
		{
			name: "grandparent whitelisted",
			args: args{
				path:      "/foo/bar/baz",
				whitelist: []WhitelistEntry{{"/foo", false}},
			},
			want: true,
		},
		{
			name: "sibling whitelisted",
			args: args{
				path:      "/foo/bar/baz",
				whitelist: []WhitelistEntry{{"/foo/bat", false}},
			},
			want: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			original := whitelist
			defer func() {
				whitelist = original
			}()
			whitelist = tt.args.whitelist
			got := CheckWhitelist(tt.args.path)
			if got != tt.want {
				t.Errorf("CheckWhitelist() = %v, want %v", got, tt.want)
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
		actual, err := ioutil.ReadFile(filepath.Join(root, p))
		if err != nil {
			t.Fatalf("error reading file: %s", p)
		}
		if !reflect.DeepEqual(actual, c) {
			t.Errorf("file contents do not match. %v!=%v", actual, c)
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

func fileHeader(name string, contents string, mode int64) *tar.Header {
	return &tar.Header{
		Name:     name,
		Size:     int64(len(contents)),
		Mode:     mode,
		Typeflag: tar.TypeReg,
		Uid:      os.Getuid(),
		Gid:      os.Getgid(),
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

func TestExtractFile(t *testing.T) {
	type tc struct {
		name     string
		hdrs     []*tar.Header
		tmpdir   string
		contents []byte
		checkers []checker
	}

	tcs := []tc{
		{
			name:     "normal file",
			contents: []byte("helloworld"),
			hdrs:     []*tar.Header{fileHeader("./bar", "helloworld", 0644)},
			checkers: []checker{
				fileExists("/bar"),
				fileMatches("/bar", []byte("helloworld")),
				permissionsMatch("/bar", 0644),
			},
		},
		{
			name:     "normal file, directory does not exist",
			contents: []byte("helloworld"),
			hdrs:     []*tar.Header{fileHeader("./foo/bar", "helloworld", 0644)},
			checkers: []checker{
				fileExists("/foo/bar"),
				fileMatches("/foo/bar", []byte("helloworld")),
				permissionsMatch("/foo/bar", 0644),
				permissionsMatch("/foo", 0755|os.ModeDir),
			},
		},
		{
			name:     "normal file, directory is created after",
			contents: []byte("helloworld"),
			hdrs: []*tar.Header{
				fileHeader("./foo/bar", "helloworld", 0644),
				dirHeader("./foo", 0722),
			},
			checkers: []checker{
				fileExists("/foo/bar"),
				fileMatches("/foo/bar", []byte("helloworld")),
				permissionsMatch("/foo/bar", 0644),
				permissionsMatch("/foo", 0722|os.ModeDir),
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
				permissionsMatch("/foo", 0755|os.ModeDir),
				permissionsMatch("/foo/bar", 0755|os.ModeDir),
			},
		},
		{
			name:   "hardlink",
			tmpdir: "/tmp/hardlink",
			hdrs: []*tar.Header{
				fileHeader("/bin/gzip", "gzip-binary", 0751),
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
			hdrs:     []*tar.Header{fileHeader("./bar", "helloworld", 04644)},
			checkers: []checker{
				fileExists("/bar"),
				fileMatches("/bar", []byte("helloworld")),
				permissionsMatch("/bar", 0644|os.ModeSetuid),
			},
		},
		{
			name:     "dir with sticky bit",
			contents: []byte("helloworld"),
			hdrs: []*tar.Header{
				dirHeader("./foo", 01755),
				fileHeader("./foo/bar", "helloworld", 0644),
			},
			checkers: []checker{
				fileExists("/foo/bar"),
				fileMatches("/foo/bar", []byte("helloworld")),
				permissionsMatch("/foo/bar", 0644),
				permissionsMatch("/foo", 0755|os.ModeDir|os.ModeSticky),
			},
		},
	}

	for _, tc := range tcs {
		t.Run(tc.name, func(t *testing.T) {
			tc := tc
			t.Parallel()
			r := ""
			var err error

			if tc.tmpdir != "" {
				r = tc.tmpdir
			} else {
				r, err = ioutil.TempDir("", "")
				if err != nil {
					t.Fatal(err)
				}
			}
			defer os.RemoveAll(r)

			for _, hdr := range tc.hdrs {
				if err := extractFile(r, hdr, bytes.NewReader(tc.contents)); err != nil {
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
			return ioutil.WriteFile(filepath.Join(r, "overwrite_me"), nil, 0644)
		},
	}}

	for _, tc := range tcs {
		t.Run(tc.name, func(t *testing.T) {
			tc := tc
			t.Parallel()
			r, err := ioutil.TempDir("", "")
			if err != nil {
				t.Fatal(err)
			}
			defer os.RemoveAll(r)

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
			if _, err := CopySymlink(link, dest, ""); err != nil {
				t.Fatal(err)
			}
			got, err := os.Readlink(dest)
			if err != nil {
				t.Fatalf("error reading link %s: %s", link, err)
			}
			if got != tc.linkTarget {
				t.Errorf("link target does not match: %s != %s", got, tc.linkTarget)
			}
		})
	}
}

func Test_childDirInWhitelist(t *testing.T) {
	type args struct {
		path      string
		whitelist []WhitelistEntry
	}
	tests := []struct {
		name string
		args args
		want bool
	}{
		{
			name: "not in whitelist",
			args: args{
				path: "/foo",
			},
			want: false,
		},
		{
			name: "child in whitelist",
			args: args{
				path: "/foo",
				whitelist: []WhitelistEntry{
					{
						Path: "/foo/bar",
					},
				},
			},
			want: true,
		},
	}
	oldWhitelist := whitelist
	defer func() {
		whitelist = oldWhitelist
	}()

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			whitelist = tt.args.whitelist
			if got := childDirInWhitelist(tt.args.path); got != tt.want {
				t.Errorf("childDirInWhitelist() = %v, want %v", got, tt.want)
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
		if err := GetExcludedFiles(tt.args.dockerfilepath, tt.args.buildcontext); err != nil {
			t.Fatal(err)
		}
		for _, excl := range tt.args.excluded {
			t.Run(tt.name+" to exclude "+excl, func(t *testing.T) {
				if !excludeFile(excl, tt.args.buildcontext) {
					t.Errorf("'%v' not excluded", excl)
				}
			})
		}
		for _, incl := range tt.args.included {
			t.Run(tt.name+" to include "+incl, func(t *testing.T) {
				if excludeFile(incl, tt.args.buildcontext) {
					t.Errorf("'%v' not included", incl)
				}
			})
		}
	}
}
