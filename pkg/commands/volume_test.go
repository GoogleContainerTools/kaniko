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

	"github.com/kopwei/kaniko/pkg/dockerfile"

	"github.com/kopwei/kaniko/testutil"
	v1 "github.com/google/go-containerregistry/pkg/v1"
	"github.com/moby/buildkit/frontend/dockerfile/instructions"
)

func TestUpdateVolume(t *testing.T) {
	cfg := &v1.Config{
		Env: []string{
			"VOLUME=/etc",
		},
		Volumes: map[string]struct{}{},
	}

	volumes := []string{
		"/tmp",
		"/var/lib",
		"$VOLUME",
	}

	volumeCmd := &VolumeCommand{
		cmd: &instructions.VolumeCommand{
			Volumes: volumes,
		},
	}

	expectedVolumes := map[string]struct{}{
		"/tmp":     {},
		"/var/lib": {},
		"/etc":     {},
	}
	buildArgs := dockerfile.NewBuildArgs([]string{})
	err := volumeCmd.ExecuteCommand(cfg, buildArgs)
	testutil.CheckErrorAndDeepEqual(t, false, err, expectedVolumes, cfg.Volumes)
}
