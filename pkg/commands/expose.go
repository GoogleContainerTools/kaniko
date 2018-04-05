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
	"github.com/GoogleCloudPlatform/k8s-container-builder/pkg/util"
	"github.com/containers/image/manifest"
	"github.com/docker/docker/builder/dockerfile/instructions"
	"github.com/sirupsen/logrus"
	"strings"
)

type ExposeCommand struct {
	cmd *instructions.ExposeCommand
}

func (r *ExposeCommand) ExecuteCommand(config *manifest.Schema2Config) error {
	logrus.Info("cmd: EXPOSE")
	// Grab the currently exposed ports
	existingPorts := config.ExposedPorts
	// Add any new ones in
	for _, p := range r.cmd.Ports {
		// Resolve any environment variables
		p, err := util.ResolveEnvironmentReplacement(p, config.Env, false)
		if err != nil {
			return err
		}
		// Add the default protocol if one isn't specified
		if !strings.Contains(p, "/") {
			p = p + "/tcp"
		}
		protocol := strings.Split(p, "/")[1]
		if !validProtocol(protocol) {
			return fmt.Errorf("Invalid protocol: %s", protocol)
		}
		logrus.Infof("Adding exposed port: %s", p)
		var x struct{}
		existingPorts[manifest.Schema2Port(p)] = x
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

func (r *ExposeCommand) FilesToSnapshot() []string {
	return []string{}
}

func (r *ExposeCommand) CreatedBy() string {
	s := []string{r.cmd.Name()}
	return strings.Join(append(s, r.cmd.Ports...), " ")
}
