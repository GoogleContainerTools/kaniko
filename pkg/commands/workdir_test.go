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
	path          string
	expectedPath  string
	snapshotFiles []string
}{
	{
		path:          "a",
		expectedPath:  "/a",
		snapshotFiles: []string{"/a"},
	},
	{
		path:          "/a",
		expectedPath:  "/a",
		snapshotFiles: []string{"/a"},
	},
	{
		path:          "b",
		expectedPath:  "/a/b",
		snapshotFiles: []string{"/a/b"},
	},
	{
		path:          "c",
		expectedPath:  "/a/b/c",
		snapshotFiles: []string{"/a/b/c"},
	},
	{
		path:          "/d",
		expectedPath:  "/d",
		snapshotFiles: []string{"/d"},
	},
	{
		path:          "$path",
		expectedPath:  "/d/usr",
		snapshotFiles: []string{"/d/usr"},
	},
	{
		path:          "$home",
		expectedPath:  "/root",
		snapshotFiles: []string{},
	},
	{
		path:          "/foo/$path/$home",
		expectedPath:  "/foo/usr/root",
		snapshotFiles: []string{"/foo/usr/root"},
	},
	{
		path:          "/tmp",
		expectedPath:  "/tmp",
		snapshotFiles: []string{},
	},
}

// For testing
func mockDir(path string, mode os.FileMode, uid, gid int64) error {
	return nil
}
func TestWorkdirCommand(t *testing.T) {

	// Mock out mkdir for testing.
	oldMkdir := mkdirAllWithPermissions
	mkdirAllWithPermissions = mockDir

	defer func() {
		mkdirAllWithPermissions = oldMkdir
	}()

	cfg := &v1.Config{
		WorkingDir: "",
		Env: []string{
			"path=usr/",
			"home=/root",
		},
	}

	for _, test := range workdirTests {
		cmd := WorkdirCommand{
			cmd: &instructions.WorkdirCommand{
				Path: test.path,
			},
			snapshotFiles: nil,
		}
		buildArgs := dockerfile.NewBuildArgs([]string{})
		cmd.ExecuteCommand(cfg, buildArgs)
		testutil.CheckErrorAndDeepEqual(t, false, nil, test.expectedPath, cfg.WorkingDir)
		testutil.CheckErrorAndDeepEqual(t, false, nil, test.snapshotFiles, cmd.snapshotFiles)
	}
}
