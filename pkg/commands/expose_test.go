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
	"testing"

	"github.com/GoogleContainerTools/kaniko/pkg/dockerfile"
	v1 "github.com/google/go-containerregistry/pkg/v1"

	"github.com/GoogleContainerTools/kaniko/testutil"
	"github.com/moby/buildkit/frontend/dockerfile/instructions"
)

func TestUpdateExposedPorts(t *testing.T) {
	cfg := &v1.Config{
		ExposedPorts: map[string]struct{}{
			"8080/tcp": {},
		},
		Env: []string{
			"port=udp",
			"num=8085",
		},
	}

	ports := []string{
		"8080",
		"8081/tcp",
		"8082",
		"8083/udp",
		"8084/$port",
		"$num",
		"$num/$port",
	}

	exposeCmd := &ExposeCommand{
		cmd: &instructions.ExposeCommand{
			Ports: ports,
		},
	}

	expectedPorts := map[string]struct{}{
		"8080/tcp": {},
		"8081/tcp": {},
		"8082/tcp": {},
		"8083/udp": {},
		"8084/udp": {},
		"8085/tcp": {},
		"8085/udp": {},
	}
	buildArgs := dockerfile.NewBuildArgs([]string{})
	err := exposeCmd.ExecuteCommand(cfg, buildArgs)
	testutil.CheckErrorAndDeepEqual(t, false, err, expectedPorts, cfg.ExposedPorts)
}

func TestInvalidProtocol(t *testing.T) {
	cfg := &v1.Config{
		ExposedPorts: map[string]struct{}{},
	}

	ports := []string{
		"80/garbage",
	}

	exposeCmd := &ExposeCommand{
		cmd: &instructions.ExposeCommand{
			Ports: ports,
		},
	}
	buildArgs := dockerfile.NewBuildArgs([]string{})
	err := exposeCmd.ExecuteCommand(cfg, buildArgs)
	testutil.CheckErrorAndDeepEqual(t, true, err, nil, nil)
}
