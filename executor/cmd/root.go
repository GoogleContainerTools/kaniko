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
	"fmt"
	"github.com/GoogleCloudPlatform/k8s-container-builder/pkg/commands"
	"github.com/GoogleCloudPlatform/k8s-container-builder/pkg/constants"
	"github.com/GoogleCloudPlatform/k8s-container-builder/pkg/dockerfile"
	"github.com/GoogleCloudPlatform/k8s-container-builder/pkg/image"
	"github.com/GoogleCloudPlatform/k8s-container-builder/pkg/snapshot"
	"github.com/GoogleCloudPlatform/k8s-container-builder/pkg/util"
	"github.com/containers/image/manifest"
	"github.com/docker/docker/builder/dockerfile/instructions"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
	"io/ioutil"
	"os"
	"path/filepath"
)

var (
	dockerfilePath string
	destination    string
	srcContext     string
	snapshotMode   string
	bucket         string
	logLevel       string
)

func init() {
	RootCmd.PersistentFlags().StringVarP(&dockerfilePath, "dockerfile", "f", "Dockerfile", "Path to the dockerfile to be built.")
	RootCmd.PersistentFlags().StringVarP(&srcContext, "context", "c", "", "Path to the dockerfile build context.")
	RootCmd.PersistentFlags().StringVarP(&bucket, "bucket", "b", "", "Name of the GCS bucket from which to access build context as tarball.")
	RootCmd.PersistentFlags().StringVarP(&destination, "destination", "d", "", "Registry the final image should be pushed to (ex: gcr.io/test/example:latest)")
	RootCmd.PersistentFlags().StringVarP(&snapshotMode, "snapshotMode", "", "full", "Set this flag to change the file attributes inspected during snapshotting")
	RootCmd.PersistentFlags().StringVarP(&logLevel, "verbosity", "v", constants.DefaultLogLevel, "Log level (debug, info, warn, error, fatal, panic")
}

var RootCmd = &cobra.Command{
	Use: "executor",
	PersistentPreRunE: func(cmd *cobra.Command, args []string) error {
		if err := util.SetLogLevel(logLevel); err != nil {
			return err
		}
		if err := resolveSourceContext(); err != nil {
			return err
		}
		return checkDockerfilePath()
	},
	Run: func(cmd *cobra.Command, args []string) {
		if err := execute(); err != nil {
			logrus.Error(err)
			os.Exit(1)
		}
	},
}

func checkDockerfilePath() error {
	if util.FilepathExists(dockerfilePath) {
		return nil
	}
	// Otherwise, check if the path relative to the build context exists
	if util.FilepathExists(filepath.Join(srcContext, dockerfilePath)) {
		dockerfilePath = filepath.Join(srcContext, dockerfilePath)
		return nil
	}
	return errors.New("please provide a valid path to a Dockerfile within the build context")
}

// resolveSourceContext unpacks the source context if it is a tar in a GCS bucket
// it resets srcContext to be the path to the unpacked build context within the image
func resolveSourceContext() error {
	if srcContext == "" && bucket == "" {
		return errors.New("please specify a path to the build context with the --context flag or a GCS bucket with the --bucket flag")
	}
	if srcContext != "" && bucket != "" {
		return errors.New("please specify either --bucket or --context as the desired build context")
	}
	if srcContext != "" {
		return nil
	}
	logrus.Infof("Using GCS bucket %s as source context", bucket)
	buildContextPath := constants.BuildContextDir
	if err := util.UnpackTarFromGCSBucket(bucket, buildContextPath); err != nil {
		return err
	}
	logrus.Debugf("Unpacked tar from %s to path %s", bucket, buildContextPath)
	srcContext = buildContextPath
	return nil
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

	hasher, err := getHasher()
	if err != nil {
		return err
	}
	l := snapshot.NewLayeredMap(hasher)
	snapshotter := snapshot.NewSnapshotter(l, constants.RootDir)

	// Take initial snapshot
	if err := snapshotter.Init(); err != nil {
		return err
	}

	// Initialize source image
	sourceImage, err := image.NewSourceImage(baseImage)
	if err != nil {
		return err
	}

	// Set environment variables within the image
	if err := image.SetEnvVariables(sourceImage); err != nil {
		return err
	}

	imageConfig := sourceImage.Config()
	// Currently only supports single stage builds
	for _, stage := range stages {
		if err := resolveOnBuild(&stage, imageConfig); err != nil {
			return err
		}
		for _, cmd := range stage.Commands {
			dockerCommand, err := commands.GetCommand(cmd, srcContext)
			if err != nil {
				return err
			}
			if err := dockerCommand.ExecuteCommand(imageConfig); err != nil {
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
				sourceImage.AppendConfigHistory(constants.Author, true)
				continue
			}
			// Append the layer to the image
			if err := sourceImage.AppendLayer(contents, constants.Author); err != nil {
				return err
			}
		}
	}
	// Push the image
	if err := setDefaultEnv(); err != nil {
		return err
	}
	return image.PushImage(sourceImage, destination)
}

func getHasher() (func(string) (string, error), error) {
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
