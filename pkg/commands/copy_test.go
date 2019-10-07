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
	"io/ioutil"
	"os"
	"path/filepath"
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

func TestCopyExecuteCmd(t *testing.T) {
	cfg := &v1.Config{
		Cmd:        nil,
		Env:        []string{},
		WorkingDir: "../../integration/context/",
	}

	for _, test := range copyTests {
		t.Run(test.name, func(t *testing.T) {
			dirList := []string{}

			logrus.Infof("Running test: %v", test.name)

			cmd := CopyCommand{
				cmd: &instructions.CopyCommand{
					SourcesAndDest: test.sourcesAndDest,
				},
				buildcontext: "../../integration/context/",
			}

			buildArgs := copySetUpBuildArgs()
			dest := cfg.WorkingDir + test.sourcesAndDest[len(test.sourcesAndDest)-1]

			os.RemoveAll(dest)

			err := cmd.ExecuteCommand(cfg, buildArgs)

			fi, ferr := os.Open(dest)
			if ferr != nil {
				t.Error()
			}
			defer fi.Close()
			fstat, ferr := fi.Stat()
			if ferr != nil {
				t.Error()
			}
			if fstat.IsDir() {
				files, ferr := ioutil.ReadDir(dest)
				if ferr != nil {
					t.Error()
				}
				for _, file := range files {
					logrus.Infof("file: %v", file.Name())
					dirList = append(dirList, file.Name())
				}
			} else {
				dirList = append(dirList, filepath.Base(dest))
			}
			//dir, err := os.Getwd()
			//			logrus.Infof("CWD: %v", dir)
			//			logrus.Infof("test.SourcesAndDest: %v", test.SourcesAndDest)
			logrus.Infof("dest: %v", dest)
			logrus.Infof("test.expectedDest: %v", test.expectedDest)
			logrus.Infof("dirList: %v", dirList)

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
