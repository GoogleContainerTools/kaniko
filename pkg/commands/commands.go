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
	"github.com/docker/docker/builder/dockerfile/instructions"
	"github.com/sirupsen/logrus"
)

type DockerCommand interface {
	ExecuteCommand() error
	// The config file has an "author" field, should return information about the command
	Author() string
	// A list of files to snapshot, empty for metadata commands or nil if we don't know
	FilesToSnapshot() []string
}

func GetCommand(cmd instructions.Command) DockerCommand {
	switch c := cmd.(type) {
	case *instructions.RunCommand:
		return &RunCommand{cmd: c}
	}
	logrus.Errorf("%s is not a supported command.", cmd.Name())
	return nil
}
