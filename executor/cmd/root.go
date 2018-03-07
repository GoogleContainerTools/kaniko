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

package cmd

import (
	"github.com/GoogleCloudPlatform/k8s-container-builder/pkg/commands"
	"github.com/GoogleCloudPlatform/k8s-container-builder/pkg/constants"
	"github.com/GoogleCloudPlatform/k8s-container-builder/pkg/dockerfile"
	"github.com/GoogleCloudPlatform/k8s-container-builder/pkg/image"
	"github.com/GoogleCloudPlatform/k8s-container-builder/pkg/snapshot"
	"github.com/GoogleCloudPlatform/k8s-container-builder/pkg/util"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
	"io/ioutil"
	"os"
)

var (
	dockerfilePath string
	destination    string
	srcContext     string
	logLevel       string
)

func init() {
	RootCmd.PersistentFlags().StringVarP(&dockerfilePath, "dockerfile", "f", "/workspace/Dockerfile", "Path to the dockerfile to be built.")
	RootCmd.PersistentFlags().StringVarP(&srcContext, "context", "c", "", "Path to the dockerfile build context.")
	RootCmd.PersistentFlags().StringVarP(&destination, "destination", "d", "", "Registry the final image should be pushed to (ex: gcr.io/test/example:latest)")
	RootCmd.PersistentFlags().StringVarP(&logLevel, "verbosity", "v", constants.DefaultLogLevel, "Log level (debug, info, warn, error, fatal, panic")
}

var RootCmd = &cobra.Command{
	Use: "executor",
	PersistentPreRunE: func(cmd *cobra.Command, args []string) error {
		return util.SetLogLevel(logLevel)
	},
	Run: func(cmd *cobra.Command, args []string) {
		if err := execute(); err != nil {
			logrus.Error(err)
			os.Exit(1)
		}
	},
}

func execute() error {
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
	logrus.Infof("Unpacking filesystem of %s...", baseImage)
	if err := util.ExtractFileSystemFromImage(baseImage); err != nil {
		return err
	}

	l := snapshot.NewLayeredMap(util.Hasher())
	snapshotter := snapshot.NewSnapshotter(l, constants.RootDir)

	// Take initial snapshot
	if err := snapshotter.Init(); err != nil {
		return err
	}

	// Initialize source image
	if err := image.InitializeSourceImage(baseImage); err != nil {
		return err
	}

	// Execute commands here
	if err := image.SetEnvVariables(); err != nil {
		return err
	}

	// Currently only supports single stage builds
	for _, stage := range stages {
		for _, cmd := range stage.Commands {
			dockerCommand := commands.GetCommand(cmd)
			if dockerCommand == nil {
				return errors.Errorf("Invalid or unsupported docker command: %v", cmd)
			}
			if err := dockerCommand.ExecuteCommand(); err != nil {
				return err
			}
			// Now, we get the files to snapshot from this command
			// If this is nil, snapshot the entire filesystem
			// Else take a snapshot of the specific files
			snapshotFiles := dockerCommand.FilesToSnapshot()
			if snapshotFiles == nil {
				logrus.Info("Taking snapshot of full filesystem...")
				contents, filesAdded, err := snapshotter.TakeSnapshot()
				if err != nil {
					return err
				}
				if !filesAdded {
					logrus.Info("No files were changed, appending empty layer to config.")
					image.AppendConfigHistory(dockerCommand.Author(), true)
					continue
				}
				// Append the layer to the image
				if err := image.AppendLayer(contents, dockerCommand.Author()); err != nil {
					return err
				}
			} else {
				logrus.Infof("Taking snapshot of files %v...", snapshotFiles)
				contents, err := snapshotter.TakeSnapshotOfFiles(snapshotFiles)
				if err != nil {
					return err
				}
				if contents == nil {
					logrus.Info("No files were changed, appending empty layer to config.")
					image.AppendConfigHistory(dockerCommand.Author(), true)
					continue
				}
				// Append the layer to the image
				if err := image.AppendLayer(contents, dockerCommand.Author()); err != nil {
					return err
				}
			}
		}
	}
	// Push the image
	return image.PushImage(destination)
}
