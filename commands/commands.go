/*
Copyright 2018 Google, Inc. All rights reserved.

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
	"github.com/GoogleCloudPlatform/k8s-container-builder/contexts/dest"
	"github.com/docker/docker/builder/dockerfile/instructions"
)

type DockerCommand interface {
	ExecuteCommand() error
}

func GetCommand(cmd instructions.Command, context dest.Context) DockerCommand {
	switch c := cmd.(type) {
	case *instructions.RunCommand:
		return RunCommand{cmd: c}
	case *instructions.CopyCommand:
		return CopyCommand{cmd: c, context: context}
	}
	return nil
}
