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

package main

import (
	"fmt"
	"os"

	crane "github.com/google/go-containerregistry/cmd/crane/cmd"
	"github.com/google/go-containerregistry/pkg/gcrane"
	"github.com/google/go-containerregistry/pkg/logs"
)

func init() {
	logs.Warn.SetOutput(os.Stderr)
	logs.Progress.SetOutput(os.Stderr)
}

func main() {
	// Maintain a map of google-specific commands that we "override".
	gcrCmds := make(map[string]bool)
	for _, cmd := range gcrane.Root.Commands() {
		gcrCmds[cmd.Use] = true
	}

	// Use crane for everything else so that this can be a drop-in replacement.
	for _, cmd := range crane.Root.Commands() {
		if _, ok := gcrCmds[cmd.Use]; !ok {
			gcrane.Root.AddCommand(cmd)
		}
	}

	if err := gcrane.Root.Execute(); err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
}
