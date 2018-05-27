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
	"github.com/google/go-containerregistry/v1"
	"github.com/sirupsen/logrus"
)

type HealthCheckCommand struct {
	cmd *instructions.HealthCheckCommand
}

// ExecuteCommand handles command processing similar to CMD and RUN,
func (h *HealthCheckCommand) ExecuteCommand(config *v1.Config, buildArgs *dockerfile.BuildArgs) error {
	logrus.Info("cmd: HEALTHCHECK")

	check := v1.HealthConfig(*h.cmd.Health)
	config.Healthcheck = &check

	return nil
}

// FilesToSnapshot returns an empty array since this is a metadata command
func (h *HealthCheckCommand) FilesToSnapshot() []string {
	return []string{}
}

// CreatedBy returns some information about the command for the image config history
func (h *HealthCheckCommand) CreatedBy() string {
	entrypoint := []string{"HEALTHCHECK"}

	return strings.Join(append(entrypoint, strings.Join(h.cmd.Health.Test, " ")), " ")
}
