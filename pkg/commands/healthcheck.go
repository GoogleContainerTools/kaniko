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
	"github.com/docker/docker/api/types/container"
	v1 "github.com/google/go-containerregistry/pkg/v1"
	"github.com/moby/buildkit/frontend/dockerfile/instructions"
)

func convertDockerHealthConfigToContainerRegistryFormat(dockerHealthcheck container.HealthConfig) v1.HealthConfig {
	return v1.HealthConfig{
		Test:        dockerHealthcheck.Test,
		Interval:    dockerHealthcheck.Interval,
		Timeout:     dockerHealthcheck.Timeout,
		StartPeriod: dockerHealthcheck.StartPeriod,
		Retries:     dockerHealthcheck.Retries,
	}
}

type HealthCheckCommand struct {
	BaseCommand
	cmd *instructions.HealthCheckCommand
}

// ExecuteCommand handles command processing similar to CMD and RUN,
func (h *HealthCheckCommand) ExecuteCommand(config *v1.Config, buildArgs *dockerfile.BuildArgs) error {
	check := convertDockerHealthConfigToContainerRegistryFormat(*h.cmd.Health)
	config.Healthcheck = &check

	return nil
}

// String returns some information about the command for the image config history
func (h *HealthCheckCommand) String() string {
	return h.cmd.String()
}
