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
	"github.com/google/go-containerregistry/pkg/name"
	"github.com/google/go-containerregistry/pkg/v1"
	"github.com/google/go-containerregistry/pkg/v1/daemon"
	"github.com/google/go-containerregistry/pkg/v1/tarball"
	"github.com/google/go-containerregistry/pkg/v1/validate"
	"github.com/spf13/cobra"
)

func init() { Root.AddCommand(NewCmdValidate()) }

// NewCmdValidate creates a new cobra.Command for the validate subcommand.
func NewCmdValidate() *cobra.Command {
	var tarballPath, remoteRef, daemonRef string
	validateCmd := &cobra.Command{
		Use:   "validate",
		Short: "Validate that an image is well-formed",
		Args:  cobra.ExactArgs(0),
		Run: func(_ *cobra.Command, args []string) {
			for flag, maker := range map[string]func(string) (v1.Image, error){
				tarballPath: makeTarball,
				remoteRef:   crane.Pull,
				daemonRef:   makeDaemon,
			} {
				img, err := maker(flag)
				if err != nil {
					log.Fatalf("failed to read image %s: %v", flag, err)
				}

				if err := validate.Image(img); err != nil {
					fmt.Printf("FAIL: %s: %v\n", flag, err)
				} else {
					fmt.Printf("PASS: %s\n", flag)
				}
			}
		},
	}
	validateCmd.Flags().StringVar(&tarballPath, "tarball", "", "Path to tarball to validate")
	validateCmd.Flags().StringVar(&remoteRef, "remote", "", "Name of remote image to validate")
	validateCmd.Flags().StringVar(&daemonRef, "daemon", "", "Name of image in daemon to validate")

	return validateCmd
}

func makeTarball(path string) (v1.Image, error) {
	return tarball.ImageFromPath(path, nil)
}

func makeDaemon(daemonRef string) (v1.Image, error) {
	ref, err := name.ParseReference(daemonRef)
	if err != nil {
		return nil, fmt.Errorf("parsing daemon ref %q: %v", daemonRef, err)
	}

	return daemon.Image(ref)
}
