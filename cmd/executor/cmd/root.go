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
	"errors"
	"os"
	"path/filepath"
	"strings"

	"github.com/GoogleContainerTools/kaniko/pkg/buildcontext"
	"github.com/GoogleContainerTools/kaniko/pkg/constants"
	"github.com/GoogleContainerTools/kaniko/pkg/executor"
	"github.com/GoogleContainerTools/kaniko/pkg/util"
	"github.com/genuinetools/amicontained/container"
	"github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
)

var (
	dockerfilePath              string
	destinations                multiArg
	srcContext                  string
	snapshotMode                string
	bucket                      string
	dockerInsecureSkipTLSVerify bool
	logLevel                    string
	force                       bool
	buildArgs                   multiArg
	tarPath                     string
	singleSnapshot              bool
	reproducible                bool
	target                      string
)

func init() {
	RootCmd.PersistentFlags().StringVarP(&dockerfilePath, "dockerfile", "f", "Dockerfile", "Path to the dockerfile to be built.")
	RootCmd.PersistentFlags().StringVarP(&srcContext, "context", "c", "/workspace/", "Path to the dockerfile build context.")
	RootCmd.PersistentFlags().StringVarP(&bucket, "bucket", "b", "", "Name of the GCS bucket from which to access build context as tarball.")
	RootCmd.PersistentFlags().VarP(&destinations, "destination", "d", "Registry the final image should be pushed to. Set it repeatedly for multiple destinations.")
	RootCmd.MarkPersistentFlagRequired("destination")
	RootCmd.PersistentFlags().StringVarP(&snapshotMode, "snapshotMode", "", "full", "Set this flag to change the file attributes inspected during snapshotting")
	RootCmd.PersistentFlags().VarP(&buildArgs, "build-arg", "", "This flag allows you to pass in ARG values at build time. Set it repeatedly for multiple values.")
	RootCmd.PersistentFlags().BoolVarP(&dockerInsecureSkipTLSVerify, "insecure-skip-tls-verify", "", false, "Push to insecure registry ignoring TLS verify")
	RootCmd.PersistentFlags().StringVarP(&logLevel, "verbosity", "v", constants.DefaultLogLevel, "Log level (debug, info, warn, error, fatal, panic")
	RootCmd.PersistentFlags().BoolVarP(&force, "force", "", false, "Force building outside of a container")
	RootCmd.PersistentFlags().StringVarP(&tarPath, "tarPath", "", "", "Path to save the image in as a tarball instead of pushing")
	RootCmd.PersistentFlags().BoolVarP(&singleSnapshot, "single-snapshot", "", false, "Set this flag to take a single snapshot at the end of the build.")
	RootCmd.PersistentFlags().BoolVarP(&reproducible, "reproducible", "", false, "Strip timestamps out of the image to make it reproducible")
	RootCmd.PersistentFlags().StringVarP(&target, "target", "", "", " Set the target build stage to build")
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
		if !checkContained() {
			if !force {
				logrus.Error("kaniko should only be run inside of a container, run with the --force flag if you are sure you want to continue.")
				os.Exit(1)
			}
			logrus.Warn("kaniko is being run outside of a container. This can have dangerous effects on your system")
		}
		if err := os.Chdir("/"); err != nil {
			logrus.Error(err)
			os.Exit(1)
		}
		image, err := executor.DoBuild(executor.KanikoBuildArgs{
			DockerfilePath: absouteDockerfilePath(),
			SrcContext:     srcContext,
			SnapshotMode:   snapshotMode,
			Args:           buildArgs,
			SingleSnapshot: singleSnapshot,
			Reproducible:   reproducible,
			Target:         target,
		})
		if err != nil {
			logrus.Error(err)
			os.Exit(1)
		}

		if err := executor.DoPush(image, destinations, tarPath, dockerInsecureSkipTLSVerify); err != nil {
			logrus.Error(err)
			os.Exit(1)
		}

	},
}

func checkContained() bool {
	_, err := container.DetectRuntime()
	return err == nil
}

func checkDockerfilePath() error {
	if util.FilepathExists(dockerfilePath) {
		if _, err := filepath.Abs(dockerfilePath); err != nil {
			return err
		}
		return nil
	}
	// Otherwise, check if the path relative to the build context exists
	if util.FilepathExists(filepath.Join(srcContext, dockerfilePath)) {
		return nil
	}
	return errors.New("please provide a valid path to a Dockerfile within the build context")
}

func absouteDockerfilePath() string {
	if util.FilepathExists(dockerfilePath) {
		// Ignore error since we already checked it in checkDockerfilePath()
		abs, _ := filepath.Abs(dockerfilePath)
		return abs
	}
	// Otherwise, return path relative to build context
	return filepath.Join(srcContext, dockerfilePath)
}

// resolveSourceContext unpacks the source context if it is a tar in a bucket
// it resets srcContext to be the path to the unpacked build context within the image
func resolveSourceContext() error {
	if srcContext == "" && bucket == "" {
		return errors.New("please specify a path to the build context with the --context flag or a bucket with the --bucket flag")
	}
	if srcContext != "" && !strings.Contains(srcContext, "://") {
		return nil
	}
	if bucket != "" {
		if !strings.Contains(bucket, "://") {
			srcContext = constants.GCSBuildContextPrefix + bucket
		} else {
			srcContext = bucket
		}
	}
	// if no prefix use Google Cloud Storage as default for backwards compability
	contextExecutor, err := buildcontext.GetBuildContext(srcContext)
	if err != nil {
		return err
	}
	logrus.Debugf("Getting source context from %s", srcContext)
	srcContext, err = contextExecutor.UnpackTarFromBuildContext()
	if err != nil {
		return err
	}
	logrus.Debugf("Build context located at %s", srcContext)
	return nil
}
