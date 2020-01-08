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

package gcrane

import (
	"fmt"
	"log"

	"github.com/google/go-containerregistry/pkg/authn"
	"github.com/google/go-containerregistry/pkg/name"
	"github.com/google/go-containerregistry/pkg/v1/google"
	"github.com/spf13/cobra"
)

func init() { Root.AddCommand(NewCmdGc()) }

// NewCmdGc creates a new cobra.Command for the gc subcommand.
func NewCmdGc() *cobra.Command {
	recursive := false
	cmd := &cobra.Command{
		Use:   "gc",
		Short: "List images that are not tagged",
		Args:  cobra.ExactArgs(1),
		Run: func(_ *cobra.Command, args []string) {
			gc(args[0], recursive)
		},
	}

	cmd.Flags().BoolVarP(&recursive, "recursive", "r", false, "Whether to recurse through repos")

	return cmd
}

func gc(root string, recursive bool) {
	repo, err := name.NewRepository(root)
	if err != nil {
		log.Fatalln(err)
	}

	auth := google.WithAuthFromKeychain(authn.DefaultKeychain)

	if recursive {
		if err := google.Walk(repo, printUntaggedImages, auth); err != nil {
			log.Fatalln(err)
		}
		return
	}

	tags, err := google.List(repo, auth)
	if err := printUntaggedImages(repo, tags, err); err != nil {
		log.Fatalln(err)
	}
}

func printUntaggedImages(repo name.Repository, tags *google.Tags, err error) error {
	if err != nil {
		return err
	}

	for digest, manifest := range tags.Manifests {
		if len(manifest.Tags) == 0 {
			fmt.Printf("%s@%s\n", repo, digest)
		}
	}

	return nil
}
