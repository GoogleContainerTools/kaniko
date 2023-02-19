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

var entrypointTests = []struct {
	prependShell bool
	cmdLine      []string
	expectedCmd  []string
}{
	{
		prependShell: true,
		cmdLine:      []string{"echo", "cmd1"},
		expectedCmd:  []string{"/bin/sh", "-c", "echo cmd1"},
	},
	{
		prependShell: false,
		cmdLine:      []string{"echo", "cmd2"},
		expectedCmd:  []string{"echo", "cmd2"},
	},
}

func TestEntrypointExecuteCmd(t *testing.T) {

	cfg := &v1.Config{
		Cmd: nil,
	}

	for _, test := range entrypointTests {
		cmd := EntrypointCommand{
			cmd: &instructions.EntrypointCommand{
				ShellDependantCmdLine: instructions.ShellDependantCmdLine{
					PrependShell: test.prependShell,
					CmdLine:      test.cmdLine,
				},
			},
		}
		err := cmd.ExecuteCommand(cfg, nil)
		testutil.CheckErrorAndDeepEqual(t, false, err, test.expectedCmd, cfg.Entrypoint)
	}
}
