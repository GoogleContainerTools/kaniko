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
	"os"
	"testing"

	"github.com/GoogleContainerTools/kaniko/pkg/dockerfile"

	"github.com/GoogleContainerTools/kaniko/testutil"
	v1 "github.com/google/go-containerregistry/pkg/v1"
	"github.com/moby/buildkit/frontend/dockerfile/instructions"
)

// Each test here changes the same WorkingDir field in the config
// So, some of the tests build off of each other
// This is needed to make sure WorkingDir handles paths correctly
// For example, if WORKDIR specifies a non-absolute path, it should be appended to the current WORKDIR
var workdirTests = []struct {
	description   string
	path          string
	mockWhitelist func(string) (bool, error)
	expectedPath  string
	snapshotFiles []string
	shdErr        bool
}{
	{
		path:          "/a",
		mockWhitelist: func(_ string) (bool, error) { return false, nil },
		expectedPath:  "/a",
		snapshotFiles: []string{"/a"},
	},
	{
		path:          "b",
		mockWhitelist: func(_ string) (bool, error) { return false, nil },
		expectedPath:  "/a/b",
		snapshotFiles: []string{"/a/b"},
	},
	{
		path:          "c",
		mockWhitelist: func(_ string) (bool, error) { return false, nil },
		expectedPath:  "/a/b/c",
		snapshotFiles: []string{"/a/b/c"},
	},
	{
		path:          "/d",
		mockWhitelist: func(_ string) (bool, error) { return false, nil },
		expectedPath:  "/d",
		snapshotFiles: []string{"/d"},
	},
	{
		path:          "$path",
		mockWhitelist: func(_ string) (bool, error) { return false, nil },
		expectedPath:  "/d/usr",
		snapshotFiles: []string{"/d/usr"},
	},
	{
		path:          "$home",
		mockWhitelist: func(_ string) (bool, error) { return false, nil },
		expectedPath:  "/root",
		snapshotFiles: []string{},
	},
	{
		path:          "/foo/$path/$home",
		mockWhitelist: func(_ string) (bool, error) { return false, nil },
		expectedPath:  "/foo/usr/root",
		snapshotFiles: []string{"/foo/usr/root"},
	},
	{
		path:          "/tmp",
		mockWhitelist: func(_ string) (bool, error) { return false, nil },
		expectedPath:  "/tmp",
		snapshotFiles: []string{},
	},
	{
		description:   "workdir updates whitelist",
		path:          "/workdir",
		mockWhitelist: func(_ string) (bool, error) { return true, nil },
		expectedPath:  "/workdir",
		snapshotFiles: []string{"/workdir"},
	},
	{
		description:   "error when updating whitelist",
		path:          "/tmp",
		mockWhitelist: func(_ string) (bool, error) { return false, fmt.Errorf("error") },
		expectedPath:  "/workdir",
		shdErr:        true,
	},
}

// For testing
func mockDir(p string, fi os.FileMode) error {
	return nil
}

func mockStat(p string) (os.FileInfo, error) {
	if p == "/workdir" || p == "/tmp" || p == "/root" {
		return nil, nil
	}
	return nil, os.ErrNotExist
}

func TestWorkdirCommand(t *testing.T) {

	// Mock out mkdir for testing.
	oldMkdir := mkdir
	mkdir = mockDir
	defer func() {
		mkdir = oldMkdir
	}()

	// Mock out stat for testing.
	oldStat := stat
	stat = mockStat
	defer func() {
		stat = oldStat
	}()

	cfg := &v1.Config{
		WorkingDir: "/",
		Env: []string{
			"path=usr/",
			"home=/root",
		},
	}

	for _, test := range workdirTests {
		t.Run(test.description, func(t *testing.T) {
			// Mock updateWhitelist
			originalUpdate := updateWhitelist
			defer func() { updateWhitelist = originalUpdate }()
			updateWhitelist = test.mockWhitelist

			cmd := WorkdirCommand{
				cmd: &instructions.WorkdirCommand{
					Path: test.path,
				},
				snapshotFiles: nil,
			}
			buildArgs := dockerfile.NewBuildArgs([]string{})
			err := cmd.ExecuteCommand(cfg, buildArgs)
			testutil.CheckError(t, test.shdErr, err)
			testutil.CheckErrorAndDeepEqual(t, test.shdErr, err, test.expectedPath, cfg.WorkingDir)
			testutil.CheckErrorAndDeepEqual(t, test.shdErr, err, test.snapshotFiles, cmd.snapshotFiles)
		})
	}
}
