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
	"github.com/spf13/cobra"
)

func init() { Root.AddCommand(NewCmdAppend()) }

// NewCmdAppend creates a new cobra.Command for the append subcommand.
func NewCmdAppend() *cobra.Command {
	var baseRef, newTag, newLayer, outFile string
	appendCmd := &cobra.Command{
		Use:   "append",
		Short: "Append contents of a tarball to a remote image",
		Args:  cobra.NoArgs,
		Run: func(_ *cobra.Command, args []string) {
			base, err := crane.Pull(baseRef)
			if err != nil {
				log.Fatalf("pulling %s: %v", baseRef, err)
			}

			img, err := crane.Append(base, newLayer)
			if err != nil {
				log.Fatalf("appending %s: %v", newLayer, err)
			}

			if outFile != "" {
				if err := crane.Save(img, newTag, outFile); err != nil {
					log.Fatalf("writing output %q: %v", outFile, err)
				}
			} else {
				if err := crane.Push(img, newTag); err != nil {
					log.Fatalf("pushing image %s: %v", newTag, err)
				}
			}
		},
	}
	appendCmd.Flags().StringVarP(&baseRef, "base", "b", "", "Name of base image to append to")
	appendCmd.Flags().StringVarP(&newTag, "new_tag", "t", "", "Tag to apply to resulting image")
	appendCmd.Flags().StringVarP(&newLayer, "new_layer", "f", "", "Path to tarball to append to image")
	appendCmd.Flags().StringVarP(&outFile, "output", "o", "", "Path to new tarball of resulting image")

	appendCmd.MarkFlagRequired("base")
	appendCmd.MarkFlagRequired("new_tag")
	appendCmd.MarkFlagRequired("new_layer")
	return appendCmd
}
