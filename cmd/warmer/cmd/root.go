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
	"os"
	"path/filepath"
	"strings"

	"github.com/GoogleContainerTools/kaniko/pkg/buildcontext"
	"github.com/GoogleContainerTools/kaniko/pkg/config"
	"github.com/GoogleContainerTools/kaniko/pkg/constants"
	"github.com/GoogleContainerTools/kaniko/pkg/executor"
	"github.com/GoogleContainerTools/kaniko/pkg/util"
	"github.com/genuinetools/amicontained/container"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
)

var (
	opts     = &config.KanikoOptions{}
	logLevel string
	force    bool
)

func init() {
	RootCmd.PersistentFlags().StringVarP(&logLevel, "verbosity", "v", constants.DefaultLogLevel, "Log level (debug, info, warn, error, fatal, panic")
	addKanikoOptionsFlags(RootCmd)
	addHiddenFlags(RootCmd)
}

var RootCmd = &cobra.Command{
	Use: "cache warmer",
	PersistentPreRunE: func(cmd *cobra.Command, args []string) error {
		if err := util.ConfigureLogging(logLevel); err != nil {
			return err
		}
		if len(opts.Images) == 0 {
			return errors.New("You must select at least one image to cache")
		}
		return nil
	},
	Run: func(cmd *cobra.Command, args []string) {
		if !checkContained() {
			if !force {
				exit(errors.New("kaniko should only be run inside of a container, run with the --force flag if you are sure you want to continue"))
			}
			logrus.Warn("kaniko is being run outside of a container. This can have dangerous effects on your system")
		}
		if err := os.Chdir("/"); err != nil {
			exit(errors.Wrap(err, "error changing to root dir"))
		}
		image, err := executor.DoBuild(opts)
		if err != nil {
			exit(errors.Wrap(err, "error building image"))
		}
		if err := executor.DoPush(image, opts); err != nil {
			exit(errors.Wrap(err, "error pushing image"))
		}
	},
}

// addKanikoOptionsFlags configures opts
func addKanikoOptionsFlags(cmd *cobra.Command) {
	RootCmd.PersistentFlags().VarP(&opts.Images, "image", "i", "Image to cache. Set it repeatedly for multiple images.")
	RootCmd.PersistentFlags().StringVarP(&opts.CacheDir, "cache-dir", "c", "/cache", "Directory of the cache.")
}

// addHiddenFlags marks certain flags as hidden from the executor help text
func addHiddenFlags(cmd *cobra.Command) {
}

func exit(err error) {
	fmt.Println(err)
	os.Exit(1)
}
