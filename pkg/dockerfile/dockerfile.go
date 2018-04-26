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

package dockerfile

import (
	"bytes"
	"github.com/GoogleContainerTools/kaniko/pkg/commands"
	"github.com/GoogleContainerTools/kaniko/pkg/constants"
	"github.com/GoogleContainerTools/kaniko/pkg/util"
	"github.com/docker/docker/builder/dockerfile/instructions"
	"github.com/docker/docker/builder/dockerfile/parser"
	"github.com/google/go-containerregistry/authn"
	"github.com/google/go-containerregistry/name"
	"github.com/google/go-containerregistry/v1"
	"github.com/google/go-containerregistry/v1/empty"
	"github.com/google/go-containerregistry/v1/remote"
	"net/http"
	"path/filepath"
	"strconv"
	"strings"
)

// Parse parses the contents of a Dockerfile and returns a list of commands
func Parse(b []byte) ([]instructions.Stage, error) {
	p, err := parser.Parse(bytes.NewReader(b))
	if err != nil {
		return nil, err
	}
	stages, _, err := instructions.Parse(p.AST)
	if err != nil {
		return nil, err
	}
	return stages, err
}

// ResolveStages resolves any calls to previous stages with names to indices
// Ex. --from=second_stage should be --from=1 for easier processing later on
func ResolveStages(stages []instructions.Stage) {
	nameToIndex := make(map[string]string)
	for i, stage := range stages {
		index := strconv.Itoa(i)
		nameToIndex[stage.Name] = index
		nameToIndex[index] = index
		for _, cmd := range stage.Commands {
			switch c := cmd.(type) {
			case *instructions.CopyCommand:
				if c.From != "" {
					c.From = nameToIndex[c.From]
				}
			}
		}
	}
}

// ParseCommands parses an array of commands into an array of instructions.Command; used for onbuild
func ParseCommands(cmdArray []string) ([]instructions.Command, error) {
	var cmds []instructions.Command
	cmdString := strings.Join(cmdArray, "\n")
	ast, err := parser.Parse(strings.NewReader(cmdString))
	if err != nil {
		return nil, err
	}
	for _, child := range ast.AST.Children {
		cmd, err := instructions.ParseCommand(child)
		if err != nil {
			return nil, err
		}
		cmds = append(cmds, cmd)
	}
	return cmds, nil
}

// Dependencies returns a list of files in this stage that will be needed in later stages
func Dependencies(index int, stages []instructions.Stage) ([]string, error) {
	var dependencies []string
	for stageIndex, stage := range stages {
		if stageIndex <= index {
			continue
		}
		var sourceImage v1.Image
		if stage.BaseName == constants.NoBaseImage {
			sourceImage = empty.Image
		} else {
			// Initialize source image
			ref, err := name.ParseReference(stage.BaseName, name.WeakValidation)
			if err != nil {
				return nil, err

			}
			auth, err := authn.DefaultKeychain.Resolve(ref.Context().Registry)
			if err != nil {
				return nil, err
			}
			sourceImage, err = remote.Image(ref, auth, http.DefaultTransport)
			if err != nil {
				return nil, err
			}
		}
		imageConfig, err := sourceImage.ConfigFile()
		if err != nil {
			return nil, err
		}
		for _, cmd := range stage.Commands {
			switch c := cmd.(type) {
			case *instructions.EnvCommand:
				envCommand := commands.NewEnvCommand(c)
				if err := envCommand.ExecuteCommand(&imageConfig.Config); err != nil {
					return nil, err
				}
			case *instructions.CopyCommand:
				if c.From != strconv.Itoa(index) {
					continue
				}
				// First, resolve any environment replacement
				resolvedEnvs, err := util.ResolveEnvironmentReplacementList(c.SourcesAndDest, imageConfig.Config.Env, true)
				if err != nil {
					return nil, err
				}
				// Resolve wildcards and get a list of resolved sources
				srcs, err := util.ResolveSources(resolvedEnvs, constants.RootDir)
				if err != nil {
					return nil, err
				}
				for index, src := range srcs {
					if !filepath.IsAbs(src) {
						srcs[index] = filepath.Join(constants.RootDir, src)
					}
				}
				dependencies = append(dependencies, srcs...)
			}
		}
	}
	return dependencies, nil
}
