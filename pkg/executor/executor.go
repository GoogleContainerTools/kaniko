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
	"bytes"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strconv"

	"github.com/GoogleContainerTools/kaniko/pkg/snapshot"

	"github.com/google/go-containerregistry/pkg/authn"
	"github.com/google/go-containerregistry/pkg/name"
	"github.com/google/go-containerregistry/pkg/v1"
	"github.com/google/go-containerregistry/pkg/v1/empty"
	"github.com/google/go-containerregistry/pkg/v1/mutate"
	"github.com/google/go-containerregistry/pkg/v1/remote"
	"github.com/google/go-containerregistry/pkg/v1/tarball"

	"io/ioutil"

	"github.com/GoogleContainerTools/kaniko/pkg/commands"
	"github.com/GoogleContainerTools/kaniko/pkg/constants"
	"github.com/GoogleContainerTools/kaniko/pkg/dockerfile"
	"github.com/GoogleContainerTools/kaniko/pkg/util"
	"github.com/docker/docker/builder/dockerfile/instructions"
	"github.com/sirupsen/logrus"
)

func DoBuild(dockerfilePath, srcContext, snapshotMode string, args []string, reproducible bool) (name.Reference, v1.Image, error) {
	// Parse dockerfile and unpack base image to root
	d, err := ioutil.ReadFile(dockerfilePath)
	if err != nil {
		return nil, nil, err
	}

	stages, err := dockerfile.Parse(d)
	if err != nil {
		return nil, nil, err
	}
	dockerfile.ResolveStages(stages)

	hasher, err := getHasher(snapshotMode)
	if err != nil {
		return nil, nil, err
	}
	for index, stage := range stages {
		baseImage, err := util.ResolveEnvironmentReplacement(stage.BaseName, args, false)
		if err != nil {
			return nil, nil, err
		}
		finalStage := index == len(stages)-1
		// Unpack file system to root
		logrus.Infof("Unpacking filesystem of %s...", baseImage)
		var sourceImage v1.Image
		var ref name.Reference
		if baseImage == constants.NoBaseImage {
			logrus.Info("No base image, nothing to extract")
			sourceImage = empty.Image
		} else {
			// Initialize source image
			ref, err = name.ParseReference(baseImage, name.WeakValidation)
			if err != nil {
				return nil, nil, err
			}
			auth, err := authn.DefaultKeychain.Resolve(ref.Context().Registry)
			if err != nil {
				return nil, nil, err
			}
			sourceImage, err = remote.Image(ref, auth, http.DefaultTransport)
			if err != nil {
				return nil, nil, err
			}
		}
		if err := util.GetFSFromImage(sourceImage); err != nil {
			return nil, nil, err
		}
		l := snapshot.NewLayeredMap(hasher)
		snapshotter := snapshot.NewSnapshotter(l, constants.RootDir)
		// Take initial snapshot
		if err := snapshotter.Init(); err != nil {
			return nil, nil, err
		}
		imageConfig, err := sourceImage.ConfigFile()
		if baseImage == constants.NoBaseImage {
			imageConfig.Config.Env = constants.ScratchEnvVars
		}
		if err != nil {
			return nil, nil, err
		}
		if err := resolveOnBuild(&stage, &imageConfig.Config); err != nil {
			return nil, nil, err
		}
		buildArgs := dockerfile.NewBuildArgs(args)
		for _, cmd := range stage.Commands {
			dockerCommand, err := commands.GetCommand(cmd, srcContext)
			if err != nil {
				return nil, nil, err
			}
			if dockerCommand == nil {
				continue
			}
			if err := dockerCommand.ExecuteCommand(&imageConfig.Config, buildArgs); err != nil {
				return nil, nil, err
			}
			if !finalStage {
				continue
			}
			// Now, we get the files to snapshot from this command and take the snapshot
			snapshotFiles := dockerCommand.FilesToSnapshot()
			contents, err := snapshotter.TakeSnapshot(snapshotFiles)
			if err != nil {
				return nil, nil, err
			}
			util.MoveVolumeWhitelistToWhitelist()
			if contents == nil {
				logrus.Info("No files were changed, appending empty layer to config.")
				continue
			}
			// Append the layer to the image
			opener := func() (io.ReadCloser, error) {
				return ioutil.NopCloser(bytes.NewReader(contents)), nil
			}
			layer, err := tarball.LayerFromOpener(opener)
			if err != nil {
				return nil, nil, err
			}
			sourceImage, err = mutate.Append(sourceImage,
				mutate.Addendum{
					Layer: layer,
					History: v1.History{
						Author:    constants.Author,
						CreatedBy: dockerCommand.CreatedBy(),
					},
				},
			)
			if err != nil {
				return nil, nil, err
			}
		}
		if finalStage {
			sourceImage, err = mutate.Config(sourceImage, imageConfig.Config)
			if err != nil {
				return nil, nil, err
			}

			if reproducible {
				sourceImage, err = mutate.Canonical(sourceImage)
			}

			if err != nil {
				return nil, nil, err
			}

			return ref, sourceImage, nil
		}
		if err := saveStageDependencies(index, stages, buildArgs.Clone()); err != nil {
			return nil, nil, err
		}
		// Delete the filesystem
		if err := util.DeleteFilesystem(); err != nil {
			return nil, nil, err
		}
	}
	return nil, nil, err
}

func DoPush(ref name.Reference, image v1.Image, destinations []string, tarPath string) error {
	// continue pushing unless an error occurs
	for _, destination := range destinations {
		// Push the image
		destRef, err := name.NewTag(destination, name.WeakValidation)
		if err != nil {
			return err
		}

		if tarPath != "" {
			return tarball.WriteToFile(tarPath, destRef, image, nil)
		}

		pushAuth, err := authn.DefaultKeychain.Resolve(destRef.Context().Registry)
		if err != nil {
			return err
		}

		wo := remote.WriteOptions{}
		err = remote.Write(destRef, image, pushAuth, http.DefaultTransport, wo)
		if err != nil {
			logrus.Error(fmt.Errorf("Failed to push to destination %s", destination))
			return err
		}
	}
	return nil
}
func saveStageDependencies(index int, stages []instructions.Stage, buildArgs *dockerfile.BuildArgs) error {
	// First, get the files in this stage later stages will need
	dependencies, err := dockerfile.Dependencies(index, stages, buildArgs)
	logrus.Infof("saving dependencies %s", dependencies)
	if err != nil {
		return err
	}
	// Then, create the directory they will exist in
	i := strconv.Itoa(index)
	dependencyDir := filepath.Join(constants.KanikoDir, i)
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

func resolveOnBuild(stage *instructions.Stage, config *v1.Config) error {
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

	// Blank out the Onbuild command list for this image
	config.OnBuild = nil
	return nil
}
