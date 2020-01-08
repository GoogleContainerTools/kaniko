// Copyright 2018 Google LLC All Rights Reserved.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//    http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package cmd

import (
	"log"

	"github.com/google/go-containerregistry/pkg/crane"
	"github.com/google/go-containerregistry/pkg/v1/cache"
	"github.com/spf13/cobra"
)

func init() { Root.AddCommand(NewCmdPull()) }

// NewCmdPull creates a new cobra.Command for the pull subcommand.
func NewCmdPull() *cobra.Command {
	var cachePath string
	pull := &cobra.Command{
		Use:   "pull",
		Short: "Pull a remote image by reference and store its contents in a tarball",
		Args:  cobra.ExactArgs(2),
		Run: func(_ *cobra.Command, args []string) {
			src, path := args[0], args[1]
			img, err := crane.Pull(src)
			if err != nil {
				log.Fatal(err)
			}
			if cachePath != "" {
				img = cache.Image(img, cache.NewFilesystemCache(cachePath))
			}
			if err := crane.Save(img, src, path); err != nil {
				log.Fatalf("saving tarball %s: %v", path, err)
			}
		},
	}
	pull.Flags().StringVarP(&cachePath, "cache_path", "c", "", "Path to cache image layers")
	return pull
}
