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
	"github.com/GoogleContainerTools/kaniko/pkg/dockerfile"
	"github.com/docker/docker/builder/dockerfile/instructions"
	"github.com/google/go-containerregistry/v1"
	"github.com/sirupsen/logrus"
	"strings"
)

type ShellCommand struct {
	cmd *instructions.ShellCommand
}

// ExecuteCommand handles command processing similar to CMD and RUN,
func (s *ShellCommand) ExecuteCommand(config *v1.Config, buildArgs *dockerfile.BuildArgs) error {
	logrus.Info("cmd: SHELL")
	var newShell []string

	newShell = s.cmd.Shell

	logrus.Infof("Replacing Shell in config with %v", newShell)
	config.Shell = newShell
	return nil
}

// FilesToSnapshot returns an empty array since this is a metadata command
func (s *ShellCommand) FilesToSnapshot() []string {
	return []string{}
}

// CreatedBy returns some information about the command for the image config history
func (s *ShellCommand) CreatedBy() string {
	entrypoint := []string{"SHELL"}
	cmdLine := strings.Join(s.cmd.Shell, " ")

	return strings.Join(append(entrypoint, cmdLine), " ")
}
