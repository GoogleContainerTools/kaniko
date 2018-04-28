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

	"github.com/docker/docker/builder/dockerfile/instructions"
	"github.com/google/go-containerregistry/v1"
	"github.com/sirupsen/logrus"
)

type EntrypointCommand struct {
	cmd *instructions.EntrypointCommand
}

// ExecuteCommand handles command processing similar to CMD and RUN,
func (e *EntrypointCommand) ExecuteCommand(config *v1.Config) error {
	logrus.Info("cmd: ENTRYPOINT")
	var newCommand []string
	if e.cmd.PrependShell {
		// This is the default shell on Linux
		var shell []string
		if len(config.Shell) > 0 {
			shell = config.Shell
		} else {
			shell = append(shell, "/bin/sh", "-c")
		}

		newCommand = append(shell, strings.Join(e.cmd.CmdLine, " "))
	} else {
		newCommand = e.cmd.CmdLine
	}

	logrus.Infof("Replacing Entrypoint in config with %v", newCommand)
	config.Entrypoint = newCommand
	return nil
}

// FilesToSnapshot returns an empty array since this is a metadata command
func (e *EntrypointCommand) FilesToSnapshot() []string {
	return []string{}
}

// CreatedBy returns some information about the command for the image config history
func (e *EntrypointCommand) CreatedBy() string {
	entrypoint := []string{"ENTRYPOINT"}
	cmdLine := strings.Join(e.cmd.CmdLine, " ")
	if e.cmd.PrependShell {
		// TODO: Support shell command here
		shell := []string{"/bin/sh", "-c"}
		appendedShell := append(entrypoint, shell...)
		return strings.Join(append(appendedShell, cmdLine), " ")
	}
	return strings.Join(append(entrypoint, cmdLine), " ")
}
