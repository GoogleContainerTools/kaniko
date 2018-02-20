/*
Copyright 2018 Google, Inc. All rights reserved.
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
	"github.com/spf13/cobra"
)

var (
	dockerfile string
	name       string
	srcContext string
	logLevel   string
	kubeconfig string
)

var RootCmd = &cobra.Command{
	Use:   "kbuild",
	Short: "kbuild is a CLI tool for building container images with full Dockerfile support without the need for Docker",
	Long: `kbuild is a CLI tool for building container images with full Dockerfile support. It doesn't require Docker,
			and builds the images in a Kubernetes cluster before pushing the final image to a registry.`,
	Run: func(cmd *cobra.Command, args []string) {
	},
}

func init() {
	RootCmd.PersistentFlags().StringVarP(&dockerfile, "dockerfile", "d", "Dockerfile", "Path to the dockerfile to be built.")
	RootCmd.PersistentFlags().StringVarP(&srcContext, "context", "c", "", "Path to the dockerfile context.")
	RootCmd.PersistentFlags().StringVarP(&name, "name", "n", "", "Name of the registry location the final image should be pushed to (ex: gcr.io/test/example:latest)")
	RootCmd.PersistentFlags().StringVarP(&logLevel, "verbosity", "v", "info", "Log level (debug, info, warn, error, fatal, panic")
}
