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
	"github.com/GoogleCloudPlatform/k8s-container-builder/pkg/constants"
	"github.com/GoogleCloudPlatform/k8s-container-builder/pkg/dockerfile"
	"github.com/GoogleCloudPlatform/k8s-container-builder/pkg/image"
	"github.com/GoogleCloudPlatform/k8s-container-builder/pkg/snapshot"
	"github.com/GoogleCloudPlatform/k8s-container-builder/pkg/util"
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
	sourceImage, err := image.NewSourceImage(baseImage)
	if err != nil {
		return err
	}

	// Execute commands here

	// Push the image
	return image.PushImage(sourceImage, destination)
}
