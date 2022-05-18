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
	"fmt"
	"strings"

	"github.com/GoogleContainerTools/kaniko/pkg/dockerfile"
	v1 "github.com/google/go-containerregistry/pkg/v1"

	"github.com/GoogleContainerTools/kaniko/pkg/util"
	"github.com/moby/buildkit/frontend/dockerfile/instructions"
	"github.com/sirupsen/logrus"
)

type ExposeCommand struct {
	BaseCommand
	cmd *instructions.ExposeCommand
}

func (r *ExposeCommand) ExecuteCommand(config *v1.Config, buildArgs *dockerfile.BuildArgs) error {
	logrus.Info("Cmd: EXPOSE")
	// Grab the currently exposed ports
	existingPorts := config.ExposedPorts
	if existingPorts == nil {
		existingPorts = make(map[string]struct{})
	}
	replacementEnvs := buildArgs.ReplacementEnvs(config.Env)
	// Add any new ones in
	for _, p := range r.cmd.Ports {
		// Resolve any environment variables
		p, err := util.ResolveEnvironmentReplacement(p, replacementEnvs, false)
		if err != nil {
			return err
		}
		// Add the default protocol if one isn't specified
		if !strings.Contains(p, "/") {
			p = p + "/tcp"
		}
		protocol := strings.Split(p, "/")[1]
		if !validProtocol(protocol) {
			return fmt.Errorf("invalid protocol: %s", protocol)
		}
		logrus.Infof("Adding exposed port: %s", p)
		existingPorts[p] = struct{}{}
	}
	config.ExposedPorts = existingPorts
	return nil
}

func validProtocol(protocol string) bool {
	validProtocols := [2]string{"tcp", "udp"}
	for _, p := range validProtocols {
		if protocol == p {
			return true
		}
	}
	return false
}

func (r *ExposeCommand) String() string {
	return r.cmd.String()
}
