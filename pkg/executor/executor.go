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

package executor

import (
	"fmt"
	img "github.com/GoogleContainerTools/container-diff/pkg/image"
	"github.com/GoogleContainerTools/kaniko/pkg/commands"
	"github.com/GoogleContainerTools/kaniko/pkg/constants"
	"github.com/GoogleContainerTools/kaniko/pkg/dockerfile"
	"github.com/GoogleContainerTools/kaniko/pkg/image"
	"github.com/GoogleContainerTools/kaniko/pkg/snapshot"
	"github.com/GoogleContainerTools/kaniko/pkg/util"
	"github.com/containers/image/manifest"
	"github.com/docker/docker/builder/dockerfile/instructions"
	"github.com/sirupsen/logrus"
	"io/ioutil"
	"os"
	"path"
	"path/filepath"
	"syscall"
)

func SetupBuild(dockerfilePath, srcContext string) error {
	// Copy over source context
	logrus.Infof("Copying build context from %s to %s", srcContext, filepath.Join(constants.KanikoBuildDir, srcContext))
	if err := util.CopyDir(srcContext, filepath.Join(constants.KanikoBuildDir, srcContext)); err != nil {
		return err
	}
	// Then, extract the image
	dockerfilePath = path.Join(constants.KanikoBuildDir, dockerfilePath)
	d, err := ioutil.ReadFile(dockerfilePath)
	if err != nil {
		return err
	}
	stages, err := dockerfile.Parse(d)
	if err != nil {
		return err
	}
	baseImage := stages[0].BaseName
	logrus.Infof("Unpacking filesystem of %s to %s...", baseImage, constants.KanikoBuildDir)
	if err := util.ExtractFileSystemFromImage(baseImage, constants.KanikoBuildDir); err != nil {
		return err
	}
	return util.CopyWhitelist(constants.KanikoBuildDir)
}

func DoBuild(dockerfilePath, srcContext, snapshotMode string) (*img.MutableSource, error) {
	// Parse dockerfile and unpack base image to root
	d, err := ioutil.ReadFile(dockerfilePath)
	if err != nil {
		return nil, err
	}

	stages, err := dockerfile.Parse(d)
	if err != nil {
		return nil, err
	}
	baseImage := stages[0].BaseName
	// Initialize source image
	sourceImage, err := image.NewSourceImage(baseImage)
	if err != nil {
		return nil, err
	}
	// Change directory and set chroot
	// Set the chroot
	if err := os.Chdir(constants.KanikoBuildDir); err != nil {
		return nil, err
	}
	if err := syscall.Chroot("."); err != nil {
		return nil, err
	}

	hasher, err := getHasher(snapshotMode)
	if err != nil {
		return nil, err
	}
	l := snapshot.NewLayeredMap(hasher)
	snapshotter := snapshot.NewSnapshotter(l, constants.RootDir)

	// Take initial snapshot
	if err := snapshotter.Init(); err != nil {
		return nil, err
	}

	// Set environment variables within the image
	if err := image.SetEnvVariables(sourceImage); err != nil {
		return nil, err
	}

	imageConfig := sourceImage.Config()
	// Currently only supports single stage builds
	for _, stage := range stages {
		if err := resolveOnBuild(&stage, imageConfig); err != nil {
			return nil, err
		}
		for _, cmd := range stage.Commands {
			dockerCommand, err := commands.GetCommand(cmd, srcContext)
			if err != nil {
				return nil, err
			}
			if dockerCommand == nil {
				return nil, err
			}
			if err := dockerCommand.ExecuteCommand(imageConfig); err != nil {
				return nil, err
			}
			// Now, we get the files to snapshot from this command and take the snapshot
			snapshotFiles := dockerCommand.FilesToSnapshot()
			contents, err := snapshotter.TakeSnapshot(snapshotFiles)
			if err != nil {
				return nil, err
			}
			util.MoveVolumeWhitelistToWhitelist()
			if contents == nil {
				logrus.Info("No files were changed, appending empty layer to config.")
				sourceImage.AppendConfigHistory(constants.Author, true)
				continue
			}
			// Append the layer to the image
			if err := sourceImage.AppendLayer(contents, constants.Author); err != nil {
				return nil, err
			}
		}
	}
	return sourceImage, nil
}

func getHasher(snapshotMode string) (func(string) (string, error), error) {
	if snapshotMode == constants.SnapshotModeTime {
		logrus.Info("Only file modification time will be considered when snapshotting")
		return util.MtimeHasher(), nil
	}
	if snapshotMode == constants.SnapshotModeFull {
		return util.Hasher(), nil
	}
	return nil, fmt.Errorf("%s is not a valid snapshot mode", snapshotMode)
}

func resolveOnBuild(stage *instructions.Stage, config *manifest.Schema2Config) error {
	if config.OnBuild == nil {
		return nil
	}
	// Otherwise, parse into commands
	cmds, err := dockerfile.ParseCommands(config.OnBuild)
	if err != nil {
		return err
	}
	// Append to the beginning of the commands in the stage
	stage.Commands = append(cmds, stage.Commands...)
	logrus.Infof("Executing %v build triggers", len(cmds))
	return nil
}

// setDefaultEnv sets default values for HOME and PATH so that
// config.json and docker-credential-gcr can be accessed
func setDefaultEnv() error {
	defaultEnvs := map[string]string{
		"HOME": "/root",
		"PATH": "/usr/local/bin/",
	}
	for key, val := range defaultEnvs {
		if err := os.Setenv(key, val); err != nil {
			return err
		}
	}
	return nil
}
