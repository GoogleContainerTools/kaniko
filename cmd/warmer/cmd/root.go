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
	"time"

	"github.com/GoogleContainerTools/kaniko/pkg/cache"
	"github.com/GoogleContainerTools/kaniko/pkg/config"
	"github.com/GoogleContainerTools/kaniko/pkg/logging"
	"github.com/pkg/errors"
	"github.com/spf13/cobra"
)

var (
	opts         = &config.WarmerOptions{}
	logLevel     string
	logFormat    string
	logTimestamp bool
)

func init() {
	RootCmd.PersistentFlags().StringVarP(&logLevel, "verbosity", "v", logging.DefaultLevel, "Log level (trace, debug, info, warn, error, fatal, panic)")
	RootCmd.PersistentFlags().StringVar(&logFormat, "log-format", logging.FormatColor, "Log format (text, color, json)")
	RootCmd.PersistentFlags().BoolVar(&logTimestamp, "log-timestamp", logging.DefaultLogTimestamp, "Timestamp in log output")

	addKanikoOptionsFlags()
	addHiddenFlags()
}

var RootCmd = &cobra.Command{
	Use: "cache warmer",
	PersistentPreRunE: func(cmd *cobra.Command, args []string) error {
		if err := logging.Configure(logLevel, logFormat, logTimestamp); err != nil {
			return err
		}

		if len(opts.Images) == 0 {
			return errors.New("You must select at least one image to cache")
		}
		return nil
	},
	Run: func(cmd *cobra.Command, args []string) {
		if _, err := os.Stat(opts.CacheDir); os.IsNotExist(err) {
			err = os.MkdirAll(opts.CacheDir, 0755)
			if err != nil {
				exit(errors.Wrap(err, "Failed to create cache directory"))
			}
		}
		if err := cache.WarmCache(opts); err != nil {
			exit(errors.Wrap(err, "Failed warming cache"))
		}
	},
}

// addKanikoOptionsFlags configures opts
func addKanikoOptionsFlags() {
	RootCmd.PersistentFlags().VarP(&opts.Images, "image", "i", "Image to cache. Set it repeatedly for multiple images.")
	RootCmd.PersistentFlags().StringVarP(&opts.CacheDir, "cache-dir", "c", "/cache", "Directory of the cache.")
	RootCmd.PersistentFlags().BoolVarP(&opts.Force, "force", "f", false, "Force cache overwriting.")
	RootCmd.PersistentFlags().DurationVarP(&opts.CacheTTL, "cache-ttl", "", time.Hour*336, "Cache timeout in hours. Defaults to two weeks.")
}

// addHiddenFlags marks certain flags as hidden from the executor help text
func addHiddenFlags() {
	RootCmd.PersistentFlags().MarkHidden("azure-container-registry-config")
}

func exit(err error) {
	fmt.Println(err)
	os.Exit(1)
}
