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
	"strings"

	"github.com/GoogleContainerTools/kaniko/pkg/dockerfile"
	v1 "github.com/google/go-containerregistry/pkg/v1"

	"github.com/moby/buildkit/frontend/dockerfile/instructions"
)

type CmdCommand struct {
	BaseCommand
	cmd *instructions.CmdCommand
}

// ExecuteCommand executes the CMD command
// Argument handling is the same as RUN.
func (c *CmdCommand) ExecuteCommand(config *v1.Config, buildArgs *dockerfile.BuildArgs) error {
	var newCommand []string
	if c.cmd.PrependShell {
		// This is the default shell on Linux
		var shell []string
		if len(config.Shell) > 0 {
			shell = config.Shell
		} else {
			shell = append(shell, "/bin/sh", "-c")
		}

		newCommand = append(shell, strings.Join(c.cmd.CmdLine, " "))
	} else {
		newCommand = c.cmd.CmdLine
	}

	config.Cmd = newCommand
	// ArgsEscaped is only used in windows environments
	config.ArgsEscaped = false
	return nil
}

// String returns some information about the command for the image config history
func (c *CmdCommand) String() string {
	return c.cmd.String()
}
