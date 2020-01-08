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

	"github.com/google/go-containerregistry/pkg/name"
	"github.com/google/go-containerregistry/pkg/v1/google"
	"github.com/spf13/cobra"
)

func init() { Root.AddCommand(NewCmdList()) }

// NewCmdList creates a new cobra.Command for the ls subcommand.
func NewCmdList() *cobra.Command {
	recursive := false
	cmd := &cobra.Command{
		Use:   "ls",
		Short: "List the contents of a repo",
		Args:  cobra.ExactArgs(1),
		Run: func(_ *cobra.Command, args []string) {
			ls(args[0], recursive)
		},
	}

	cmd.Flags().BoolVarP(&recursive, "recursive", "r", false, "Whether to recurse through repos")

	return cmd
}

func ls(root string, recursive bool) {
	repo, err := name.NewRepository(root)
	if err != nil {
		log.Fatalln(err)
	}

	auth, err := google.Keychain.Resolve(repo.Registry)
	if err != nil {
		log.Fatalf("getting auth for %q: %v", root, err)
	}

	if recursive {
		if err := google.Walk(repo, printImages, google.WithAuth(auth)); err != nil {
			log.Fatalln(err)
		}
		return
	}

	tags, err := google.List(repo, google.WithAuth(auth))
	if err != nil {
		log.Fatalln(err)
	}

	if len(tags.Manifests) == 0 && len(tags.Children) == 0 {
		// If we didn't see any GCR extensions, just list the tags like normal.
		for _, tag := range tags.Tags {
			fmt.Printf("%s:%s\n", repo, tag)
		}
		return
	}

	// Since we're not recursing, print the subdirectories too.
	for _, child := range tags.Children {
		fmt.Printf("%s/%s\n", repo, child)
	}

	if err := printImages(repo, tags, err); err != nil {
		log.Fatalln(err)
	}
}

func printImages(repo name.Repository, tags *google.Tags, err error) error {
	if err != nil {
		return err
	}

	for digest, manifest := range tags.Manifests {
		fmt.Printf("%s@%s\n", repo, digest)

		for _, tag := range manifest.Tags {
			fmt.Printf("%s:%s\n", repo, tag)
		}
	}

	return nil
}
