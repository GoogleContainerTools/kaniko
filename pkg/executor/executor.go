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
	"io/ioutil"
	"net/http"
	"os"

	"github.com/google/go-containerregistry/v1/empty"

	"github.com/google/go-containerregistry/v1/tarball"

	"github.com/google/go-containerregistry/authn"
	"github.com/google/go-containerregistry/name"
	"github.com/google/go-containerregistry/v1"
	"github.com/google/go-containerregistry/v1/mutate"
	"github.com/google/go-containerregistry/v1/remote"

	"github.com/GoogleContainerTools/kaniko/pkg/commands"
	"github.com/GoogleContainerTools/kaniko/pkg/constants"
	"github.com/GoogleContainerTools/kaniko/pkg/dockerfile"
	"github.com/GoogleContainerTools/kaniko/pkg/image"
	"github.com/GoogleContainerTools/kaniko/pkg/snapshot"
	"github.com/GoogleContainerTools/kaniko/pkg/util"
	"github.com/docker/docker/builder/dockerfile/instructions"
	"github.com/sirupsen/logrus"
)

func DoBuild(dockerfilePath, srcContext, destination, snapshotMode string, dockerInsecureSkipTLSVerify bool) error {
	// Parse dockerfile and unpack base image to root
	d, err := ioutil.ReadFile(dockerfilePath)
	if err != nil {
		return err
	}

	stages, err := dockerfile.Parse(d)
	if err != nil {
		return err
	}
	baseImage := stages[0].BaseName

	// Unpack file system to root
	var sourceImage v1.Image
	var ref name.Reference
	logrus.Infof("Unpacking filesystem of %s...", baseImage)
	if baseImage == constants.NoBaseImage {
		logrus.Info("No base image, nothing to extract")
		sourceImage = empty.Image
	} else {
		// Initialize source image
		ref, err = name.ParseReference(baseImage, name.WeakValidation)
		if err != nil {
			return err
		}
		auth, err := authn.DefaultKeychain.Resolve(ref.Context().Registry)
		if err != nil {
			return err
		}
		sourceImage, err = remote.Image(ref, auth, http.DefaultTransport)
		if err != nil {
			return err
		}
	}
	if err := util.GetFSFromImage(sourceImage); err != nil {
		return err
	}

	hasher, err := getHasher(snapshotMode)
	if err != nil {
		return err
	}
	l := snapshot.NewLayeredMap(hasher)
	snapshotter := snapshot.NewSnapshotter(l, constants.RootDir)

	// Take initial snapshot
	if err := snapshotter.Init(); err != nil {
		return err
	}

	destRef, err := name.ParseReference(destination, name.WeakValidation)
	if err != nil {
		return err
	}
	// Set environment variables within the image
	if err := image.SetEnvVariables(sourceImage); err != nil {
		return err
	}

	imageConfig, err := sourceImage.ConfigFile()
	if err != nil {
		return err
	}
	// Currently only supports single stage builds
	for _, stage := range stages {
		if err := resolveOnBuild(&stage, &imageConfig.Config); err != nil {
			return err
		}
		for _, cmd := range stage.Commands {
			dockerCommand, err := commands.GetCommand(cmd, srcContext)
			if err != nil {
				return err
			}
			if dockerCommand == nil {
				continue
			}
			if err := dockerCommand.ExecuteCommand(&imageConfig.Config); err != nil {
				return err
			}
			// Now, we get the files to snapshot from this command and take the snapshot
			snapshotFiles := dockerCommand.FilesToSnapshot()
			contents, err := snapshotter.TakeSnapshot(snapshotFiles)
			if err != nil {
				return err
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
				return err
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
				return err
			}
		}
	}
	// Push the image
	if err := setDefaultEnv(); err != nil {
		return err
	}

	wo := remote.WriteOptions{}
	if ref != nil {
		wo.MountPaths = []name.Repository{ref.Context()}
	}
	pushAuth, err := authn.DefaultKeychain.Resolve(destRef.Context().Registry)
	if err != nil {
		return err
	}
	sourceImage, err = mutate.Config(sourceImage, imageConfig.Config)
	if err != nil {
		return err
	}

	return remote.Write(destRef, sourceImage, pushAuth, http.DefaultTransport, wo)
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
