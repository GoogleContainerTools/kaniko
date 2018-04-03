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
	"github.com/GoogleCloudPlatform/k8s-container-builder/testutil"
	"github.com/containers/image/manifest"
	"github.com/docker/docker/builder/dockerfile/instructions"
	"testing"
)

func TestUpdateVolume(t *testing.T) {
	cfg := &manifest.Schema2Config{
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
		&instructions.VolumeCommand{
			Volumes: volumes,
		},
	}

	expectedVolumes := map[string]struct{}{
		"/tmp":     {},
		"/var/lib": {},
		"/etc":     {},
	}

	err := volumeCmd.ExecuteCommand(cfg)
	testutil.CheckErrorAndDeepEqual(t, false, err, expectedVolumes, cfg.Volumes)
}
