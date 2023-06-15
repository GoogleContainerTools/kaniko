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

package commands

import (
	"github.com/GoogleContainerTools/kaniko/pkg/dockerfile"
	"github.com/GoogleContainerTools/kaniko/pkg/util"
	v1 "github.com/google/go-containerregistry/pkg/v1"
	"github.com/moby/buildkit/frontend/dockerfile/instructions"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
)

type CurrentCacheKey func() (string, error)

type DockerCommand interface {
	// ExecuteCommand is responsible for:
	// 	1. Making required changes to the filesystem (ex. copying files for ADD/COPY or setting ENV variables)
	//  2. Updating metadata fields in the config
	// It should not change the config history.
	ExecuteCommand(*v1.Config, *dockerfile.BuildArgs) error
	// Returns a string representation of the command
	String() string
	// A list of files to snapshot, empty for metadata commands or nil if we don't know
	FilesToSnapshot() []string

	// ProvidesFileToSnapshot is true for all metadata commands and commands which know
	// list of files changed. False for Run command.
	ProvidesFilesToSnapshot() bool

	// Return a cache-aware implementation of this command, if it exists.
	CacheCommand(v1.Image) DockerCommand

	// Return true if this command depends on the build context.
	FilesUsedFromContext(*v1.Config, *dockerfile.BuildArgs) ([]string, error)

	MetadataOnly() bool

	RequiresUnpackedFS() bool

	// Whether the output layer of this command should be cached in and
	// retrieved from the layer cache.
	ShouldCacheOutput() bool

	// ShouldDetectDeletedFiles returns true if the command could delete files.
	ShouldDetectDeletedFiles() bool

	// True if need add ARGs and EVNs to composite cache string with resolved command
	// need only for RUN instruction
	IsArgsEnvsRequiredInCache() bool
}

func GetCommand(cmd instructions.Command, fileContext util.FileContext, useNewRun bool, cacheCopy bool, cacheRun bool) (DockerCommand, error) {
	switch c := cmd.(type) {
	case *instructions.RunCommand:
		if useNewRun {
			return &RunMarkerCommand{cmd: c, shdCache: cacheRun}, nil
		}
		return &RunCommand{cmd: c, shdCache: cacheRun}, nil
	case *instructions.CopyCommand:
		return &CopyCommand{cmd: c, fileContext: fileContext, shdCache: cacheCopy}, nil
	case *instructions.ExposeCommand:
		return &ExposeCommand{cmd: c}, nil
	case *instructions.EnvCommand:
		return &EnvCommand{cmd: c}, nil
	case *instructions.WorkdirCommand:
		return &WorkdirCommand{cmd: c}, nil
	case *instructions.AddCommand:
		return &AddCommand{cmd: c, fileContext: fileContext}, nil
	case *instructions.CmdCommand:
		return &CmdCommand{cmd: c}, nil
	case *instructions.EntrypointCommand:
		return &EntrypointCommand{cmd: c}, nil
	case *instructions.LabelCommand:
		return &LabelCommand{cmd: c}, nil
	case *instructions.UserCommand:
		return &UserCommand{cmd: c}, nil
	case *instructions.OnbuildCommand:
		return &OnBuildCommand{cmd: c}, nil
	case *instructions.VolumeCommand:
		return &VolumeCommand{cmd: c}, nil
	case *instructions.StopSignalCommand:
		return &StopSignalCommand{cmd: c}, nil
	case *instructions.ArgCommand:
		return &ArgCommand{cmd: c}, nil
	case *instructions.ShellCommand:
		return &ShellCommand{cmd: c}, nil
	case *instructions.HealthCheckCommand:
		return &HealthCheckCommand{cmd: c}, nil
	case *instructions.MaintainerCommand:
		logrus.Warnf("%s is deprecated, skipping", cmd.Name())
		return nil, nil
	}
	return nil, errors.Errorf("%s is not a supported command", cmd.Name())
}
