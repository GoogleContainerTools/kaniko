package godirwalk

import (
	"io/ioutil"
	"os"
	"path/filepath"
	"testing"
)

func teardown(tb testing.TB, testDataRoot string) {
	if err := os.RemoveAll(testDataRoot); err != nil {
		tb.Error(err)
	}
}

func setup(tb testing.TB) string {
	testDataRoot, err := ioutil.TempDir(os.TempDir(), "godirwalk-")

	// Create files, creating directories along the way, then create symbolic links

	files := []string{
		"dir1/dir1a/file1a1",
		"dir1/dir1a/skip",
		"dir1/dir1a/z1a2",
		"dir1/file1b",
		"dir2/file2a",
		"dir2/skip/file2b1",
		"dir2/z2c/file2c1",
		"dir3/aaa.txt",
		"dir3/zzz/aaa.txt",
		"dir4/aaa.txt",
		"dir4/zzz/aaa.txt",
		"dir5/a1.txt",
		"dir5/a2/a2a/a2a1.txt",
		"dir5/a2/a2b.txt",
		"dir6/bravo.txt",
		"dir6/code/123.txt",
		"file3",
	}

	for _, pathname := range files {
		pathname = filepath.Join(testDataRoot, filepath.FromSlash(pathname))
		if err := os.MkdirAll(filepath.Dir(pathname), os.ModePerm); err != nil {
			tb.Fatalf("cannot create directory for test scaffolding: %s\n", err)
		}
		if err = ioutil.WriteFile(pathname, []byte("some test data\n"), os.ModePerm); err != nil {
			tb.Fatalf("cannot create file for test scaffolding: %s\n", err)
		}
	}

	symlinks := map[string]string{
		"dir3/skip":                "zzz",
		"dir4/symlinkToDirectory":  "zzz",
		"dir4/symlinkToFile":       "aaa.txt",
		"symlinks/dir-symlink":     "../symlinks",
		"symlinks/file-symlink":    "../file3",
		"symlinks/invalid-symlink": "/non/existing/file",
	}

	for pathname, referent := range symlinks {
		pathname = filepath.Join(testDataRoot, filepath.FromSlash(pathname))
		if err := os.MkdirAll(filepath.Dir(pathname), os.ModePerm); err != nil {
			tb.Fatalf("cannot create directory for test scaffolding: %s\n", err)
		}
		referent = filepath.FromSlash(referent)
		if err := os.Symlink(referent, pathname); err != nil {
			tb.Fatalf("cannot create symbolic link for test scaffolding: %s\n", err)
		}
	}

	extraDirs := []string{
		"dir6/abc",
		"dir6/def",
	}

	for _, pathname := range extraDirs {
		pathname = filepath.Join(testDataRoot, filepath.FromSlash(pathname))
		if err := os.MkdirAll(pathname, os.ModePerm); err != nil {
			tb.Fatalf("cannot create directory for test scaffolding: %s\n", err)
		}
	}

	if err := os.MkdirAll(filepath.Join(testDataRoot, filepath.FromSlash("dir6/noaccess")), 0); err != nil {
		tb.Fatalf("cannot create directory for test scaffolding: %s\n", err)
	}

	return testDataRoot
}
