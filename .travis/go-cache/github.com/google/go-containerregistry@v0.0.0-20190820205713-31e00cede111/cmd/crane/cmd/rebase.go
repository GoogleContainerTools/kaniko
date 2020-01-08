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
	"fmt"
	"log"

	"github.com/google/go-containerregistry/pkg/crane"
	"github.com/google/go-containerregistry/pkg/v1/mutate"
	"github.com/spf13/cobra"
)

func init() { Root.AddCommand(NewCmdRebase()) }

// NewCmdRebase creates a new cobra.Command for the rebase subcommand.
func NewCmdRebase() *cobra.Command {
	var orig, oldBase, newBase, rebased string
	rebaseCmd := &cobra.Command{
		Use:   "rebase",
		Short: "Rebase an image onto a new base image",
		Args:  cobra.NoArgs,
		Run: func(*cobra.Command, []string) {
			origImg, err := crane.Pull(orig)
			if err != nil {
				log.Fatalf("pulling %s: %v", orig, err)
			}

			oldBaseImg, err := crane.Pull(oldBase)
			if err != nil {
				log.Fatalf("pulling %s: %v", oldBase, err)
			}

			newBaseImg, err := crane.Pull(newBase)
			if err != nil {
				log.Fatalf("pulling %s: %v", newBase, err)
			}

			img, err := mutate.Rebase(origImg, oldBaseImg, newBaseImg)
			if err != nil {
				log.Fatalf("rebasing: %v", err)
			}

			if err := crane.Push(img, rebased); err != nil {
				log.Fatalf("pushing %s: %v", rebased, err)
			}

			digest, err := img.Digest()
			if err != nil {
				log.Fatalf("digesting rebased: %v", err)
			}
			fmt.Println(digest.String())
		},
	}
	rebaseCmd.Flags().StringVarP(&orig, "original", "", "", "Original image to rebase")
	rebaseCmd.Flags().StringVarP(&oldBase, "old_base", "", "", "Old base image to remove")
	rebaseCmd.Flags().StringVarP(&newBase, "new_base", "", "", "New base image to insert")
	rebaseCmd.Flags().StringVarP(&rebased, "rebased", "", "", "Tag to apply to rebased image")

	rebaseCmd.MarkFlagRequired("original")
	rebaseCmd.MarkFlagRequired("old_base")
	rebaseCmd.MarkFlagRequired("new_base")
	rebaseCmd.MarkFlagRequired("rebased")
	return rebaseCmd
}
