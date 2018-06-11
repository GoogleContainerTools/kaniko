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

	"github.com/GoogleContainerTools/kaniko/pkg/util"
	"github.com/docker/docker/builder/dockerfile/instructions"
	"github.com/google/go-containerregistry/pkg/v1"
	"github.com/sirupsen/logrus"
)

type EnvCommand struct {
	cmd *instructions.EnvCommand
}

func (e *EnvCommand) ExecuteCommand(config *v1.Config, buildArgs *dockerfile.BuildArgs) error {
	logrus.Info("cmd: ENV")
	newEnvs := e.cmd.Env
	replacementEnvs := buildArgs.ReplacementEnvs(config.Env)
	return util.UpdateConfigEnv(newEnvs, config, replacementEnvs)
}

// We know that no files have changed, so return an empty array
func (e *EnvCommand) FilesToSnapshot() []string {
	return []string{}
}

// CreatedBy returns some information about the command for the image config history
func (e *EnvCommand) CreatedBy() string {
	envArray := []string{e.cmd.Name()}
	for _, pair := range e.cmd.Env {
		envArray = append(envArray, pair.Key+"="+pair.Value)
	}
	return strings.Join(envArray, " ")
}
