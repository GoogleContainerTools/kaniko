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
package commands

import (
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"
	"syscall"
	"testing"

	"github.com/GoogleContainerTools/kaniko/pkg/dockerfile"
	"github.com/GoogleContainerTools/kaniko/pkg/util"
	"github.com/GoogleContainerTools/kaniko/testutil"
	v1 "github.com/google/go-containerregistry/pkg/v1"
	"github.com/moby/buildkit/frontend/dockerfile/instructions"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
)

var copyTests = []struct {
	name           string
	sourcesAndDest []string
	expectedDest   []string
}{
	{
		name:           "copy foo into tempCopyExecuteTest/",
		sourcesAndDest: []string{"foo", "tempCopyExecuteTest/"},
		expectedDest:   []string{"foo"},
	},
	{
		name:           "copy foo into tempCopyExecuteTest",
		sourcesAndDest: []string{"foo", "tempCopyExecuteTest"},
		expectedDest:   []string{"tempCopyExecuteTest"},
	},
}

func setupTestTemp() string {
	tempDir, err := ioutil.TempDir("", "")
	if err != nil {
		logrus.Fatalf("error creating temp dir %s", err)
	}
	logrus.Debugf("Tempdir: %s", tempDir)

	srcPath, err := filepath.Abs("../../integration/context")
	if err != nil {
		logrus.Fatalf("error getting abs path %s", srcPath)
	}
	cperr := filepath.Walk(srcPath,
		func(path string, info os.FileInfo, err error) error {
			if err != nil {
				return err
			}
			if path != srcPath {
				if err != nil {
					return err
				}
				tempPath := strings.TrimPrefix(path, srcPath)
				fileInfo, err := os.Stat(path)
				if err != nil {
					return err
				}
				if fileInfo.IsDir() {
					os.MkdirAll(tempDir+"/"+tempPath, 0777)
				} else {
					out, err := os.Create(tempDir + "/" + tempPath)
					if err != nil {
						return err
					}
					defer out.Close()

					in, err := os.Open(path)
					if err != nil {
						return err
					}
					defer in.Close()

					_, err = io.Copy(out, in)
					if err != nil {
						return err
					}
				}
			}
			return nil
		})
	if cperr != nil {
		logrus.Fatalf("error populating temp dir %s", cperr)
	}

	return tempDir
}
func TestCopyExecuteCmd(t *testing.T) {
	tempDir := setupTestTemp()
	defer os.RemoveAll(tempDir)

	cfg := &v1.Config{
		Cmd:        nil,
		Env:        []string{},
		WorkingDir: tempDir,
	}
	fileContext := util.FileContext{Root: tempDir}

	for _, test := range copyTests {
		t.Run(test.name, func(t *testing.T) {
			dirList := []string{}

			cmd := CopyCommand{
				cmd: &instructions.CopyCommand{
					SourcesAndDest: test.sourcesAndDest,
				},
				fileContext: fileContext,
			}

			buildArgs := copySetUpBuildArgs()
			dest := cfg.WorkingDir + "/" + test.sourcesAndDest[len(test.sourcesAndDest)-1]

			err := cmd.ExecuteCommand(cfg, buildArgs)
			if err != nil {
				t.Error()
			}

			fi, err := os.Open(dest)
			if err != nil {
				t.Error()
			}
			defer fi.Close()
			fstat, err := fi.Stat()
			if err != nil {
				t.Error()
			}
			if fstat.IsDir() {
				files, err := ioutil.ReadDir(dest)
				if err != nil {
					t.Error()
				}
				for _, file := range files {
					logrus.Debugf("file: %v", file.Name())
					dirList = append(dirList, file.Name())
				}
			} else {
				dirList = append(dirList, filepath.Base(dest))
			}

			testutil.CheckErrorAndDeepEqual(t, false, err, test.expectedDest, dirList)
			os.RemoveAll(dest)
		})
	}
}

func copySetUpBuildArgs() *dockerfile.BuildArgs {
	buildArgs := dockerfile.NewBuildArgs([]string{
		"buildArg1=foo",
		"buildArg2=foo2",
	})
	buildArgs.AddArg("buildArg1", nil)
	d := "default"
	buildArgs.AddArg("buildArg2", &d)
	return buildArgs
}

func Test_resolveIfSymlink(t *testing.T) {
	type testCase struct {
		destPath     string
		expectedPath string
		err          error
	}

	tmpDir, err := ioutil.TempDir("", "copy-test")
	if err != nil {
		t.Error(err)
	}

	baseDir, err := ioutil.TempDir(tmpDir, "not-linked")
	if err != nil {
		t.Error(err)
	}

	path, err := ioutil.TempFile(baseDir, "foo.txt")
	if err != nil {
		t.Error(err)
	}

	thepath, err := filepath.Abs(filepath.Dir(path.Name()))
	if err != nil {
		t.Error(err)
	}
	cases := []testCase{
		{destPath: thepath, expectedPath: thepath, err: nil},
		{destPath: "/", expectedPath: "/", err: nil},
	}
	baseDir = tmpDir
	symLink := filepath.Join(baseDir, "symlink")
	if err := os.Symlink(filepath.Base(thepath), symLink); err != nil {
		t.Error(err)
	}
	cases = append(cases,
		testCase{filepath.Join(symLink, "foo.txt"), filepath.Join(thepath, "foo.txt"), nil},
		testCase{filepath.Join(symLink, "inner", "foo.txt"), filepath.Join(thepath, "inner", "foo.txt"), nil},
	)

	absSymlink := filepath.Join(tmpDir, "abs-symlink")
	if err := os.Symlink(thepath, absSymlink); err != nil {
		t.Error(err)
	}
	cases = append(cases,
		testCase{filepath.Join(absSymlink, "foo.txt"), filepath.Join(thepath, "foo.txt"), nil},
		testCase{filepath.Join(absSymlink, "inner", "foo.txt"), filepath.Join(thepath, "inner", "foo.txt"), nil},
	)

	for i, c := range cases {
		t.Run(fmt.Sprintf("%d", i), func(t *testing.T) {
			res, e := resolveIfSymlink(c.destPath)
			if e != c.err {
				t.Errorf("%s: expected %v but got %v", c.destPath, c.err, e)
			}

			if res != c.expectedPath {
				t.Errorf("%s: expected %v but got %v", c.destPath, c.expectedPath, res)
			}
		})
	}
}

func TestCopyCommand_ExecuteCommand_Extended(t *testing.T) {
	setupDirs := func(t *testing.T) (string, string) {
		testDir, err := ioutil.TempDir("", "")
		if err != nil {
			t.Fatal(err)
		}

		dir := filepath.Join(testDir, "bar")

		if err := os.MkdirAll(dir, 0777); err != nil {
			t.Fatal(err)
		}
		file := filepath.Join(dir, "bam.txt")

		if err := ioutil.WriteFile(file, []byte("meow"), 0777); err != nil {
			t.Fatal(err)
		}
		targetPath := filepath.Join(dir, "dam.txt")
		if err := ioutil.WriteFile(targetPath, []byte("woof"), 0777); err != nil {
			t.Fatal(err)
		}
		if err := os.Symlink("dam.txt", filepath.Join(dir, "sym.link")); err != nil {
			t.Fatal(err)
		}

		return testDir, filepath.Base(dir)
	}

	t.Run("copy dir to another dir", func(t *testing.T) {
		testDir, srcDir := setupDirs(t)
		defer os.RemoveAll(testDir)
		expected, err := ioutil.ReadDir(filepath.Join(testDir, srcDir))
		if err != nil {
			t.Fatal(err)
		}

		cmd := CopyCommand{
			cmd: &instructions.CopyCommand{
				SourcesAndDest: []string{srcDir, "dest"},
			},
			fileContext: util.FileContext{Root: testDir},
		}

		cfg := &v1.Config{
			Cmd:        nil,
			Env:        []string{},
			WorkingDir: testDir,
		}

		err = cmd.ExecuteCommand(cfg, dockerfile.NewBuildArgs([]string{}))
		if err != nil {
			t.Fatal(err)
		}
		testutil.CheckNoError(t, err)
		// Check if "dest" dir exists with contents of srcDir
		actual, err := ioutil.ReadDir(filepath.Join(testDir, "dest"))
		if err != nil {
			t.Fatal(err)
		}
		for i, f := range actual {
			testutil.CheckDeepEqual(t, expected[i].Name(), f.Name())
			testutil.CheckDeepEqual(t, expected[i].Mode(), f.Mode())
		}
	})

	t.Run("copy file to a dir", func(t *testing.T) {
		testDir, srcDir := setupDirs(t)
		defer os.RemoveAll(testDir)
		cmd := CopyCommand{
			cmd: &instructions.CopyCommand{
				SourcesAndDest: []string{filepath.Join(srcDir, "bam.txt"), "dest/"},
			},
			fileContext: util.FileContext{Root: testDir},
		}

		cfg := &v1.Config{
			Cmd:        nil,
			Env:        []string{},
			WorkingDir: testDir,
		}

		err := cmd.ExecuteCommand(cfg, dockerfile.NewBuildArgs([]string{}))
		testutil.CheckNoError(t, err)
		// Check if "dest" dir exists with file bam.txt
		files, err := ioutil.ReadDir(filepath.Join(testDir, "dest"))
		if err != nil {
			t.Fatal(err)
		}
		testutil.CheckDeepEqual(t, 1, len(files))
		testutil.CheckDeepEqual(t, files[0].Name(), "bam.txt")
	})

	t.Run("copy file to a filepath", func(t *testing.T) {
		testDir, srcDir := setupDirs(t)
		defer os.RemoveAll(testDir)
		cmd := CopyCommand{
			cmd: &instructions.CopyCommand{
				SourcesAndDest: []string{filepath.Join(srcDir, "bam.txt"), "dest"},
			},
			fileContext: util.FileContext{Root: testDir},
		}

		cfg := &v1.Config{
			Cmd:        nil,
			Env:        []string{},
			WorkingDir: testDir,
		}

		err := cmd.ExecuteCommand(cfg, dockerfile.NewBuildArgs([]string{}))
		testutil.CheckNoError(t, err)
		// Check if bam.txt is copied to dest file
		if _, err := os.Lstat(filepath.Join(testDir, "dest")); err != nil {
			t.Fatal(err)
		}
	})
	t.Run("copy file to a dir without trailing /", func(t *testing.T) {
		testDir, srcDir := setupDirs(t)
		defer os.RemoveAll(testDir)

		destDir := filepath.Join(testDir, "dest")
		if err := os.MkdirAll(destDir, 0777); err != nil {
			t.Fatal(err)
		}

		cmd := CopyCommand{
			cmd: &instructions.CopyCommand{
				SourcesAndDest: []string{filepath.Join(srcDir, "bam.txt"), "dest"},
			},
			fileContext: util.FileContext{Root: testDir},
		}

		cfg := &v1.Config{
			Cmd:        nil,
			Env:        []string{},
			WorkingDir: testDir,
		}

		err := cmd.ExecuteCommand(cfg, dockerfile.NewBuildArgs([]string{}))
		testutil.CheckNoError(t, err)
		// Check if "dest" dir exists with file bam.txt
		files, err := ioutil.ReadDir(filepath.Join(testDir, "dest"))
		if err != nil {
			t.Fatal(err)
		}
		testutil.CheckDeepEqual(t, 1, len(files))
		testutil.CheckDeepEqual(t, files[0].Name(), "bam.txt")

	})

	t.Run("copy symlink file to a dir", func(t *testing.T) {
		testDir, srcDir := setupDirs(t)
		defer os.RemoveAll(testDir)

		cmd := CopyCommand{
			cmd: &instructions.CopyCommand{
				SourcesAndDest: []string{filepath.Join(srcDir, "sym.link"), "dest/"},
			},
			fileContext: util.FileContext{Root: testDir},
		}

		cfg := &v1.Config{
			Cmd:        nil,
			Env:        []string{},
			WorkingDir: testDir,
		}

		err := cmd.ExecuteCommand(cfg, dockerfile.NewBuildArgs([]string{}))
		testutil.CheckNoError(t, err)
		// Check if "dest" dir exists with link sym.link
		files, err := ioutil.ReadDir(filepath.Join(testDir, "dest"))
		if err != nil {
			t.Fatal(err)
		}
		// bam.txt and sym.link should be present
		testutil.CheckDeepEqual(t, 1, len(files))
		testutil.CheckDeepEqual(t, files[0].Name(), "sym.link")
		testutil.CheckDeepEqual(t, true, files[0].Mode()&os.ModeSymlink != 0)
		linkName, err := os.Readlink(filepath.Join(testDir, "dest", "sym.link"))
		if err != nil {
			t.Fatal(err)
		}
		testutil.CheckDeepEqual(t, linkName, "dam.txt")
	})

	t.Run("copy deadlink symlink file to a dir", func(t *testing.T) {
		testDir, srcDir := setupDirs(t)
		defer os.RemoveAll(testDir)
		doesNotExists := filepath.Join(testDir, "dead.txt")
		if err := ioutil.WriteFile(doesNotExists, []byte("remove me"), 0777); err != nil {
			t.Fatal(err)
		}
		if err := os.Symlink("../dead.txt", filepath.Join(testDir, srcDir, "dead.link")); err != nil {
			t.Fatal(err)
		}
		if err := os.Remove(doesNotExists); err != nil {
			t.Fatal(err)
		}

		cmd := CopyCommand{
			cmd: &instructions.CopyCommand{
				SourcesAndDest: []string{filepath.Join(srcDir, "dead.link"), "dest/"},
			},
			fileContext: util.FileContext{Root: testDir},
		}

		cfg := &v1.Config{
			Cmd:        nil,
			Env:        []string{},
			WorkingDir: testDir,
		}

		err := cmd.ExecuteCommand(cfg, dockerfile.NewBuildArgs([]string{}))
		testutil.CheckNoError(t, err)
		// Check if "dest" dir exists with link dead.link
		files, err := ioutil.ReadDir(filepath.Join(testDir, "dest"))
		if err != nil {
			t.Fatal(err)
		}
		testutil.CheckDeepEqual(t, 1, len(files))
		testutil.CheckDeepEqual(t, files[0].Name(), "dead.link")
		testutil.CheckDeepEqual(t, true, files[0].Mode()&os.ModeSymlink != 0)
		linkName, err := os.Readlink(filepath.Join(testDir, "dest", "dead.link"))
		if err != nil {
			t.Fatal(err)
		}
		testutil.CheckDeepEqual(t, linkName, "../dead.txt")
	})

	t.Run("copy src symlink dir to a dir", func(t *testing.T) {
		testDir, srcDir := setupDirs(t)
		defer os.RemoveAll(testDir)
		expected, err := ioutil.ReadDir(filepath.Join(testDir, srcDir))
		if err != nil {
			t.Fatal(err)
		}

		another := filepath.Join(testDir, "another")
		os.Symlink(filepath.Join(testDir, srcDir), another)

		cmd := CopyCommand{
			cmd: &instructions.CopyCommand{
				SourcesAndDest: []string{"another", "dest"},
			},
			fileContext: util.FileContext{Root: testDir},
		}

		cfg := &v1.Config{
			Cmd:        nil,
			Env:        []string{},
			WorkingDir: testDir,
		}

		err = cmd.ExecuteCommand(cfg, dockerfile.NewBuildArgs([]string{}))
		testutil.CheckNoError(t, err)
		// Check if "dest" dir exists with contents of srcDir
		actual, err := ioutil.ReadDir(filepath.Join(testDir, "dest"))
		if err != nil {
			t.Fatal(err)
		}
		for i, f := range actual {
			testutil.CheckDeepEqual(t, expected[i].Name(), f.Name())
			testutil.CheckDeepEqual(t, expected[i].Mode(), f.Mode())
		}
	})

	t.Run("copy dir with a symlink to a file outside of current src dir", func(t *testing.T) {
		testDir, srcDir := setupDirs(t)
		defer os.RemoveAll(testDir)
		expected, err := ioutil.ReadDir(filepath.Join(testDir, srcDir))
		if err != nil {
			t.Fatal(err)
		}

		anotherSrc := filepath.Join(testDir, "anotherSrc")
		if err := os.MkdirAll(anotherSrc, 0777); err != nil {
			t.Fatal(err)
		}
		targetPath := filepath.Join(anotherSrc, "target.txt")
		if err := ioutil.WriteFile(targetPath, []byte("woof"), 0777); err != nil {
			t.Fatal(err)
		}
		if err := os.Symlink(targetPath, filepath.Join(testDir, srcDir, "zSym.link")); err != nil {
			t.Fatal(err)
		}

		cmd := CopyCommand{
			cmd: &instructions.CopyCommand{
				SourcesAndDest: []string{srcDir, "dest"},
			},
			fileContext: util.FileContext{Root: testDir},
		}

		cfg := &v1.Config{
			Cmd:        nil,
			Env:        []string{},
			WorkingDir: testDir,
		}

		err = cmd.ExecuteCommand(cfg, dockerfile.NewBuildArgs([]string{}))
		testutil.CheckNoError(t, err)
		// Check if "dest" dir exists contents of srcDir and an extra zSym.link created
		// in this test
		actual, err := ioutil.ReadDir(filepath.Join(testDir, "dest"))
		if err != nil {
			t.Fatal(err)
		}
		testutil.CheckDeepEqual(t, 4, len(actual))
		for i, f := range expected {
			testutil.CheckDeepEqual(t, f.Name(), actual[i].Name())
			testutil.CheckDeepEqual(t, f.Mode(), actual[i].Mode())
		}
		linkName, err := os.Readlink(filepath.Join(testDir, "dest", "zSym.link"))
		if err != nil {
			t.Fatal(err)
		}
		testutil.CheckDeepEqual(t, linkName, targetPath)
	})

	t.Run("copy src symlink dir to a dir", func(t *testing.T) {
		testDir, srcDir := setupDirs(t)
		defer os.RemoveAll(testDir)
		expected, err := ioutil.ReadDir(filepath.Join(testDir, srcDir))
		if err != nil {
			t.Fatal(err)
		}

		another := filepath.Join(testDir, "another")
		os.Symlink(filepath.Join(testDir, srcDir), another)

		cmd := CopyCommand{
			cmd: &instructions.CopyCommand{
				SourcesAndDest: []string{"another", "dest"},
			},
			fileContext: util.FileContext{Root: testDir},
		}

		cfg := &v1.Config{
			Cmd:        nil,
			Env:        []string{},
			WorkingDir: testDir,
		}

		err = cmd.ExecuteCommand(cfg, dockerfile.NewBuildArgs([]string{}))
		testutil.CheckNoError(t, err)
		// Check if "dest" dir exists with bam.txt and "dest" dir is a symlink
		actual, err := ioutil.ReadDir(filepath.Join(testDir, "dest"))
		if err != nil {
			t.Fatal(err)
		}
		for i, f := range actual {
			testutil.CheckDeepEqual(t, expected[i].Name(), f.Name())
			testutil.CheckDeepEqual(t, expected[i].Mode(), f.Mode())
		}
	})

	t.Run("copy src dir to a dest dir which is a symlink", func(t *testing.T) {
		testDir, srcDir := setupDirs(t)
		defer os.RemoveAll(testDir)
		expected, err := ioutil.ReadDir(filepath.Join(testDir, srcDir))
		if err != nil {
			t.Fatal(err)
		}

		dest := filepath.Join(testDir, "dest")
		if err := os.MkdirAll(dest, 0777); err != nil {
			t.Fatal(err)
		}
		linkedDest := filepath.Join(testDir, "linkDest")
		if err := os.Symlink(dest, linkedDest); err != nil {
			t.Fatal(err)
		}

		cmd := CopyCommand{
			cmd: &instructions.CopyCommand{
				SourcesAndDest: []string{srcDir, linkedDest},
			},
			fileContext: util.FileContext{Root: testDir},
		}

		cfg := &v1.Config{
			Cmd:        nil,
			Env:        []string{},
			WorkingDir: testDir,
		}

		err = cmd.ExecuteCommand(cfg, dockerfile.NewBuildArgs([]string{}))
		testutil.CheckNoError(t, err)
		// Check if "linkdest" dir exists with contents of srcDir
		actual, err := ioutil.ReadDir(filepath.Join(testDir, "linkDest"))
		if err != nil {
			t.Fatal(err)
		}
		for i, f := range expected {
			testutil.CheckDeepEqual(t, f.Name(), actual[i].Name())
			testutil.CheckDeepEqual(t, f.Mode(), actual[i].Mode())
		}
		// Check if linkDest -> dest
		linkName, err := os.Readlink(filepath.Join(testDir, "linkDest"))
		if err != nil {
			t.Fatal(err)
		}
		testutil.CheckDeepEqual(t, linkName, dest)
	})

	t.Run("copy src file to a dest dir which is a symlink", func(t *testing.T) {
		testDir, srcDir := setupDirs(t)
		defer os.RemoveAll(testDir)

		dest := filepath.Join(testDir, "dest")
		if err := os.MkdirAll(dest, 0777); err != nil {
			t.Fatal(err)
		}
		linkedDest := filepath.Join(testDir, "linkDest")
		if err := os.Symlink(dest, linkedDest); err != nil {
			t.Fatal(err)
		}

		cmd := CopyCommand{
			cmd: &instructions.CopyCommand{
				SourcesAndDest: []string{fmt.Sprintf("%s/bam.txt", srcDir), linkedDest},
			},
			fileContext: util.FileContext{Root: testDir},
		}

		cfg := &v1.Config{
			Cmd:        nil,
			Env:        []string{},
			WorkingDir: testDir,
		}

		err := cmd.ExecuteCommand(cfg, dockerfile.NewBuildArgs([]string{}))
		testutil.CheckNoError(t, err)
		// Check if "linkDest" link is same.
		actual, err := ioutil.ReadDir(filepath.Join(testDir, "dest"))
		if err != nil {
			t.Fatal(err)
		}
		testutil.CheckDeepEqual(t, "bam.txt", actual[0].Name())
		c, err := ioutil.ReadFile(filepath.Join(testDir, "dest", "bam.txt"))
		if err != nil {
			t.Fatal(err)
		}
		testutil.CheckDeepEqual(t, "meow", string(c))
		// Check if linkDest -> dest
		linkName, err := os.Readlink(filepath.Join(testDir, "linkDest"))
		if err != nil {
			t.Fatal(err)
		}
		testutil.CheckDeepEqual(t, linkName, dest)
	})

	t.Run("copy src file to a dest dir with chown", func(t *testing.T) {
		testDir, srcDir := setupDirs(t)
		defer os.RemoveAll(testDir)

		original := getUserGroup
		defer func() { getUserGroup = original }()

		uid := os.Getuid()
		gid := os.Getgid()

		getUserGroup = func(userStr string, _ []string) (int64, int64, error) {
			return int64(uid), int64(gid), nil
		}

		cmd := CopyCommand{
			cmd: &instructions.CopyCommand{
				SourcesAndDest: []string{fmt.Sprintf("%s/bam.txt", srcDir), testDir},
				Chown:          "alice:group",
			},
			fileContext: util.FileContext{Root: testDir},
		}

		cfg := &v1.Config{
			Cmd:        nil,
			Env:        []string{},
			WorkingDir: testDir,
		}

		err := cmd.ExecuteCommand(cfg, dockerfile.NewBuildArgs([]string{}))
		testutil.CheckNoError(t, err)

		actual, err := ioutil.ReadDir(filepath.Join(testDir))
		if err != nil {
			t.Fatal(err)
		}

		testutil.CheckDeepEqual(t, "bam.txt", actual[0].Name())

		if stat, ok := actual[0].Sys().(*syscall.Stat_t); ok {
			if int(stat.Uid) != uid {
				t.Errorf("uid don't match, got %d, expected %d", stat.Uid, uid)
			}
			if int(stat.Gid) != gid {
				t.Errorf("gid don't match, got %d, expected %d", stat.Gid, gid)
			}
		}
	})

	t.Run("copy src file to a dest dir with chown and random user", func(t *testing.T) {
		testDir, srcDir := setupDirs(t)
		defer os.RemoveAll(testDir)

		original := getUserGroup
		defer func() { getUserGroup = original }()

		getUserGroup = func(userStr string, _ []string) (int64, int64, error) {
			return 12345, 12345, nil
		}

		cmd := CopyCommand{
			cmd: &instructions.CopyCommand{
				SourcesAndDest: []string{fmt.Sprintf("%s/bam.txt", srcDir), testDir},
				Chown:          "missing:missing",
			},
			fileContext: util.FileContext{Root: testDir},
		}

		cfg := &v1.Config{
			Cmd:        nil,
			Env:        []string{},
			WorkingDir: testDir,
		}

		err := cmd.ExecuteCommand(cfg, dockerfile.NewBuildArgs([]string{}))
		if !errors.Is(err, os.ErrPermission) {
			testutil.CheckNoError(t, err)
		}
	})

	t.Run("copy src dir with relative symlinks in a dir", func(t *testing.T) {
		testDir, srcDir := setupDirs(t)
		defer os.RemoveAll(testDir)

		// Make another dir inside bar with a relative symlink
		dir := filepath.Join(testDir, srcDir, "another")
		if err := os.MkdirAll(dir, 0777); err != nil {
			t.Fatal(err)
		}
		os.Symlink("../bam.txt", filepath.Join(dir, "bam_relative.txt"))

		dest := filepath.Join(testDir, "copy")
		cmd := CopyCommand{
			cmd: &instructions.CopyCommand{
				SourcesAndDest: []string{srcDir, dest},
			},
			fileContext: util.FileContext{Root: testDir},
		}

		cfg := &v1.Config{
			Cmd:        nil,
			Env:        []string{},
			WorkingDir: testDir,
		}
		err := cmd.ExecuteCommand(cfg, dockerfile.NewBuildArgs([]string{}))
		testutil.CheckNoError(t, err)
		actual, err := ioutil.ReadDir(filepath.Join(dest, "another"))
		if err != nil {
			t.Fatal(err)
		}
		testutil.CheckDeepEqual(t, "bam_relative.txt", actual[0].Name())
		linkName, err := os.Readlink(filepath.Join(dest, "another", "bam_relative.txt"))
		if err != nil {
			t.Fatal(err)
		}
		testutil.CheckDeepEqual(t, "../bam.txt", linkName)
	})
}
