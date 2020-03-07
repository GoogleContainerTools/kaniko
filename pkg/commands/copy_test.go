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
	"archive/tar"
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/GoogleContainerTools/kaniko/pkg/dockerfile"
	"github.com/GoogleContainerTools/kaniko/testutil"
	v1 "github.com/google/go-containerregistry/pkg/v1"
	"github.com/moby/buildkit/frontend/dockerfile/instructions"
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

	for _, test := range copyTests {
		t.Run(test.name, func(t *testing.T) {
			dirList := []string{}

			cmd := CopyCommand{
				cmd: &instructions.CopyCommand{
					SourcesAndDest: test.sourcesAndDest,
				},
				buildcontext: tempDir,
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

func Test_CachingCopyCommand_ExecuteCommand(t *testing.T) {
	tempDir := setupTestTemp()

	tarContent, err := prepareTarFixture([]string{"foo.txt"})
	if err != nil {
		t.Errorf("couldn't prepare tar fixture %v", err)
	}

	config := &v1.Config{}
	buildArgs := &dockerfile.BuildArgs{}

	type testCase struct {
		desctiption    string
		expectLayer    bool
		expectErr      bool
		count          *int
		expectedCount  int
		command        *CachingCopyCommand
		extractedFiles []string
		contextFiles   []string
	}
	testCases := []testCase{
		func() testCase {
			err = ioutil.WriteFile(filepath.Join(tempDir, "foo.txt"), []byte("meow"), 0644)
			if err != nil {
				t.Errorf("couldn't write tempfile %v", err)
				t.FailNow()
			}

			c := &CachingCopyCommand{
				img: fakeImage{
					ImageLayers: []v1.Layer{
						fakeLayer{TarContent: tarContent},
					},
				},
				buildcontext: tempDir,
				cmd: &instructions.CopyCommand{
					SourcesAndDest: []string{
						"foo.txt", "foo.txt",
					},
				},
			}
			count := 0
			tc := testCase{
				desctiption:    "with valid image and valid layer",
				count:          &count,
				expectedCount:  1,
				expectLayer:    true,
				extractedFiles: []string{"/foo.txt"},
				contextFiles:   []string{"foo.txt"},
			}
			c.extractFn = func(_ string, _ *tar.Header, _ io.Reader) error {
				*tc.count++
				return nil
			}
			tc.command = c
			return tc
		}(),
		func() testCase {
			c := &CachingCopyCommand{}
			tc := testCase{
				desctiption: "with no image",
				expectErr:   true,
			}
			c.extractFn = func(_ string, _ *tar.Header, _ io.Reader) error {
				return nil
			}
			tc.command = c
			return tc
		}(),
		func() testCase {
			c := &CachingCopyCommand{
				img: fakeImage{},
			}
			c.extractFn = func(_ string, _ *tar.Header, _ io.Reader) error {
				return nil
			}
			return testCase{
				desctiption: "with image containing no layers",
				expectErr:   true,
				command:     c,
			}
		}(),
		func() testCase {
			c := &CachingCopyCommand{
				img: fakeImage{
					ImageLayers: []v1.Layer{
						fakeLayer{},
					},
				},
			}
			c.extractFn = func(_ string, _ *tar.Header, _ io.Reader) error {
				return nil
			}
			tc := testCase{
				desctiption: "with image one layer which has no tar content",
				expectErr:   false, // this one probably should fail but doesn't because of how ExecuteCommand and util.GetFSFromLayers are implemented - cvgw- 2019-11-25
				expectLayer: true,
			}
			tc.command = c
			return tc
		}(),
	}

	for _, tc := range testCases {
		t.Run(tc.desctiption, func(t *testing.T) {
			c := tc.command
			err := c.ExecuteCommand(config, buildArgs)
			if !tc.expectErr && err != nil {
				t.Errorf("Expected err to be nil but was %v", err)
			} else if tc.expectErr && err == nil {
				t.Error("Expected err but was nil")
			}

			if tc.count != nil {
				if *tc.count != tc.expectedCount {
					t.Errorf("Expected extractFn to be called %v times but was called %v times", tc.expectedCount, *tc.count)
				}
				for _, file := range tc.extractedFiles {
					match := false
					cFiles := c.FilesToSnapshot()
					for _, cFile := range cFiles {
						if file == cFile {
							match = true
							break
						}
					}
					if !match {
						t.Errorf("Expected extracted files to include %v but did not %v", file, cFiles)
					}
				}

				cmdFiles, err := c.FilesUsedFromContext(
					config, buildArgs,
				)
				if err != nil {
					t.Errorf("failed to get files used from context from command %v", err)
				}

				if len(cmdFiles) != len(tc.contextFiles) {
					t.Errorf("expected files used from context to equal %v but was %v", tc.contextFiles, cmdFiles)
				}
			}

			if c.layer == nil && tc.expectLayer {
				t.Error("expected the command to have a layer set but instead was nil")
			} else if c.layer != nil && !tc.expectLayer {
				t.Error("expected the command to have no layer set but instead found a layer")
			}

			if c.readSuccess != tc.expectLayer {
				t.Errorf("expected read success to be %v but was %v", tc.expectLayer, c.readSuccess)
			}
		})
	}
}

func TestGetUserGroup(t *testing.T) {
	tests := []struct {
		description string
		chown       string
		env         []string
		mock        func(string, bool) (uint32, uint32, error)
		expectedU   int64
		expectedG   int64
		shdErr      bool
	}{
		{
			description: "non empty chown",
			chown:       "some:some",
			env:         []string{},
			mock:        func(string, bool) (uint32, uint32, error) { return 100, 1000, nil },
			expectedU:   100,
			expectedG:   1000,
		},
		{
			description: "non empty chown with env replacement",
			chown:       "some:$foo",
			env:         []string{"foo=key"},
			mock: func(c string, t bool) (uint32, uint32, error) {
				if c == "some:key" {
					return 10, 100, nil
				}
				return 0, 0, fmt.Errorf("did not resolve environment variable")
			},
			expectedU: 10,
			expectedG: 100,
		},
		{
			description: "empty chown string",
			mock: func(c string, t bool) (uint32, uint32, error) {
				return 0, 0, fmt.Errorf("should not be called")
			},
			expectedU: -1,
			expectedG: -1,
		},
	}
	for _, tc := range tests {
		t.Run(tc.description, func(t *testing.T) {
			original := getUIDAndGID
			defer func() { getUIDAndGID = original }()
			getUIDAndGID = tc.mock
			uid, gid, err := getUserGroup(tc.chown, tc.env)
			testutil.CheckErrorAndDeepEqual(t, tc.shdErr, err, uid, tc.expectedU)
			testutil.CheckErrorAndDeepEqual(t, tc.shdErr, err, gid, tc.expectedG)
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
		os.Symlink(targetPath, filepath.Join(dir, "sym.link"))

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
			buildcontext: testDir,
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
			buildcontext: testDir,
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
			buildcontext: testDir,
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
			buildcontext: testDir,
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
			buildcontext: testDir,
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
		// sym.link should be present as a Regular file with contents of target "woof"
		testutil.CheckDeepEqual(t, 1, len(files))
		testutil.CheckDeepEqual(t, files[0].Name(), "sym.link")
		testutil.CheckDeepEqual(t, false, files[0].Mode()&os.ModeSymlink != 0)
		c, err := ioutil.ReadFile(filepath.Join(testDir, "dest", "sym.link"))
		testutil.CheckDeepEqual(t, "woof", string(c))
	})

	t.Run("copy src symlink dir to a dir", func(t *testing.T) {
		testDir, srcDir := setupDirs(t)
		defer os.RemoveAll(testDir)
		expected, err := ioutil.ReadDir(filepath.Join(testDir, srcDir))

		another := filepath.Join(testDir, "another")
		os.Symlink(filepath.Join(testDir, srcDir), another)

		cmd := CopyCommand{
			cmd: &instructions.CopyCommand{
				SourcesAndDest: []string{"another", "dest"},
			},
			buildcontext: testDir,
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
			buildcontext: testDir,
		}

		cfg := &v1.Config{
			Cmd:        nil,
			Env:        []string{},
			WorkingDir: testDir,
		}

		err = cmd.ExecuteCommand(cfg, dockerfile.NewBuildArgs([]string{}))
		testutil.CheckNoError(t, err)
		// Check if "dest" dir exists contests of srcDir and an extra zSym.link created
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

		another := filepath.Join(testDir, "another")
		os.Symlink(filepath.Join(testDir, srcDir), another)

		cmd := CopyCommand{
			cmd: &instructions.CopyCommand{
				SourcesAndDest: []string{"another", "dest"},
			},
			buildcontext: testDir,
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
	t.Run("copy file to a dest which is a symlink", func(t *testing.T) {
		testDir, srcDir := setupDirs(t)
		//defer os.RemoveAll(testDir)
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
			buildcontext: testDir,
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
}
