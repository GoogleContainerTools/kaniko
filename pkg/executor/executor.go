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
	"path/filepath"
	"strconv"
)

// DoBuild executes each stage of the build
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
	dockerfile.ResolveStages(stages)

	hasher, err := getHasher(snapshotMode)
	if err != nil {
		return nil, err
	}

	for index, stage := range stages {
		baseImage := stage.BaseName
		finalStage := index == len(stages)-1
		// Unpack file system to root
		logrus.Infof("Unpacking filesystem of %s...", baseImage)
		if err := util.ExtractFileSystemFromImage(baseImage); err != nil {
			return nil, err
		}

		l := snapshot.NewLayeredMap(hasher)
		snapshotter := snapshot.NewSnapshotter(l, constants.RootDir)

		// Take initial snapshot
		if err := snapshotter.Init(); err != nil {
			return nil, err
		}
		// Initialize source image
		sourceImage, err := image.NewSourceImage(baseImage)
		if err != nil {
			return nil, err
		}
		imageConfig := sourceImage.Config()
		if err := resolveOnBuild(&stage, imageConfig); err != nil {
			return nil, err
		}

		for _, cmd := range stage.Commands {
			dockerCommand, err := commands.GetCommand(cmd, srcContext)
			if err != nil {
				return nil, err
			}
			if dockerCommand == nil {
				continue
			}
			if err := dockerCommand.ExecuteCommand(imageConfig); err != nil {
				return nil, err
			}
			if !finalStage {
				continue
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
		if finalStage {
			return sourceImage, nil
		}
		if err := saveStageDependencies(index, stages); err != nil {
			return nil, err
		}
		// Delete the filesystem
		if err := util.DeleteFilesystem(); err != nil {
			return nil, err
		}
	}
	return nil, nil
}

func saveStageDependencies(index int, stages []instructions.Stage) error {
	// First, get the files in this stage later stages will need
	dependencies, err := dockerfile.Dependencies(index, stages)
	logrus.Infof("saving dependencies %s", dependencies)
	if err != nil {
		return err
	}
	// Then, create the directory they will exist in
	i := strconv.Itoa(index)
	dependencyDir := filepath.Join(constants.BuildContextDir, i)
	if err := os.MkdirAll(dependencyDir, 0755); err != nil {
		return err
	}
	// Now, copy over dependencies to this dir
	for _, d := range dependencies {
		fi, err := os.Lstat(d)
		if err != nil {
			return err
		}
		dest := filepath.Join(dependencyDir, d)
		if fi.IsDir() {
			if err := util.CopyDir(d, dest); err != nil {
				return err
			}
		} else if fi.Mode()&os.ModeSymlink != 0 {
			if err := util.CopySymlink(d, dest); err != nil {
				return err
			}
		} else {
			if err := util.CopyFile(d, dest); err != nil {
				return err
			}
		}
	}
	return nil
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
