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
