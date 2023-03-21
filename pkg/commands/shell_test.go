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
	"testing"

	"github.com/GoogleContainerTools/kaniko/testutil"
	v1 "github.com/google/go-containerregistry/pkg/v1"
	"github.com/moby/buildkit/frontend/dockerfile/instructions"
)

var shellTests = []struct {
	cmdLine       []string
	expectedShell []string
}{
	{
		cmdLine:       []string{"/bin/bash", "-c"},
		expectedShell: []string{"/bin/bash", "-c"},
	},
	{
		cmdLine:       []string{"/bin/bash"},
		expectedShell: []string{"/bin/bash"},
	},
}

func TestShellExecuteCmd(t *testing.T) {

	cfg := &v1.Config{
		Shell: nil,
	}

	for _, test := range shellTests {
		cmd := ShellCommand{
			cmd: &instructions.ShellCommand{
				Shell: test.cmdLine,
			},
		}
		err := cmd.ExecuteCommand(cfg, nil)
		testutil.CheckErrorAndDeepEqual(t, false, err, test.expectedShell, cfg.Shell)
	}
}
