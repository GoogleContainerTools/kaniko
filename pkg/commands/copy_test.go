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
	cases := []testCase{{destPath: thepath, expectedPath: thepath, err: nil}}

	baseDir = tmpDir
	symLink := filepath.Join(baseDir, "symlink")
	if err := os.Symlink(filepath.Base(thepath), symLink); err != nil {
		t.Error(err)
	}
	cases = append(cases, testCase{filepath.Join(symLink, "foo.txt"), filepath.Join(thepath, "foo.txt"), nil})

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
