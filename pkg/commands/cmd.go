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

	"github.com/docker/docker/builder/dockerfile/instructions"
	"github.com/google/go-containerregistry/pkg/v1"
	"github.com/sirupsen/logrus"
)

type CmdCommand struct {
	cmd *instructions.CmdCommand
}

// ExecuteCommand executes the CMD command
// Argument handling is the same as RUN.
func (c *CmdCommand) ExecuteCommand(config *v1.Config, buildArgs *dockerfile.BuildArgs) error {
	logrus.Info("cmd: CMD")
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

	logrus.Infof("Replacing CMD in config with %v", newCommand)
	config.Cmd = newCommand
	config.ArgsEscaped = true
	return nil
}

// FilesToSnapshot returns an empty array since this is a metadata command
func (c *CmdCommand) FilesToSnapshot() []string {
	return []string{}
}

// CreatedBy returns some information about the command for the image config history
func (c *CmdCommand) CreatedBy() string {
	cmd := []string{"CMD"}
	cmdLine := strings.Join(c.cmd.CmdLine, " ")
	if c.cmd.PrependShell {
		// TODO: Support shell command here
		shell := []string{"/bin/sh", "-c"}
		appendedShell := append(cmd, shell...)
		return strings.Join(append(appendedShell, cmdLine), " ")
	}
	return strings.Join(append(cmd, cmdLine), " ")
}
