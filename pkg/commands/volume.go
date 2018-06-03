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
	"os"
	"strings"

	"github.com/GoogleContainerTools/kaniko/pkg/dockerfile"

	"github.com/GoogleContainerTools/kaniko/pkg/util"
	"github.com/docker/docker/builder/dockerfile/instructions"
	"github.com/google/go-containerregistry/pkg/v1"
	"github.com/sirupsen/logrus"
)

type VolumeCommand struct {
	cmd           *instructions.VolumeCommand
	snapshotFiles []string
}

func (v *VolumeCommand) ExecuteCommand(config *v1.Config, buildArgs *dockerfile.BuildArgs) error {
	logrus.Info("cmd: VOLUME")
	volumes := v.cmd.Volumes
	replacementEnvs := buildArgs.ReplacementEnvs(config.Env)
	resolvedVolumes, err := util.ResolveEnvironmentReplacementList(volumes, replacementEnvs, true)
	if err != nil {
		return err
	}
	existingVolumes := config.Volumes
	if existingVolumes == nil {
		existingVolumes = map[string]struct{}{}
	}
	for _, volume := range resolvedVolumes {
		var x struct{}
		existingVolumes[volume] = x
		err := util.AddPathToVolumeWhitelist(volume)
		if err != nil {
			return err
		}

		logrus.Infof("Creating directory %s", volume)
		if err := os.MkdirAll(volume, 0755); err != nil {
			return err
		}

		//Check if directory already exists?
		v.snapshotFiles = append(v.snapshotFiles, volume)
	}
	config.Volumes = existingVolumes

	return nil
}

func (v *VolumeCommand) FilesToSnapshot() []string {
	return v.snapshotFiles
}

func (v *VolumeCommand) CreatedBy() string {
	return strings.Join(append([]string{v.cmd.Name()}, v.cmd.Volumes...), " ")
}
