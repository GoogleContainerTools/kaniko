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
	"runtime"
	"strings"

	"github.com/GoogleContainerTools/kaniko/pkg/config"
	d "github.com/docker/docker/builder/dockerfile"
	"github.com/moby/buildkit/frontend/dockerfile/instructions"
)

type BuildArgs struct {
	d.BuildArgs
}

func NewBuildArgs(args []string) *BuildArgs {
	argsFromOptions := make(map[string]*string)
	for _, a := range args {
		s := strings.SplitN(a, "=", 2)
		if len(s) == 1 {
			argsFromOptions[s[0]] = nil
		} else {
			argsFromOptions[s[0]] = &s[1]
		}
	}
	return &BuildArgs{
		BuildArgs: *d.NewBuildArgs(argsFromOptions),
	}
}

func (b *BuildArgs) Clone() *BuildArgs {
	clone := b.BuildArgs.Clone()
	return &BuildArgs{
		BuildArgs: *clone,
	}
}

// ReplacementEnvs returns a list of filtered environment variables
func (b *BuildArgs) ReplacementEnvs(envs []string) []string {
	// Ensure that we operate on a new array and do not modify the underlying array
	resultEnv := make([]string, len(envs))
	copy(resultEnv, envs)
	filtered := b.FilterAllowed(envs)
	// Disable makezero linter, since the previous make is paired with a same sized copy
	return append(resultEnv, filtered...) //nolint:makezero
}

// AddMetaArgs adds the supplied args map to b's allowedMetaArgs
func (b *BuildArgs) AddMetaArgs(metaArgs []instructions.ArgCommand) {
	for _, marg := range metaArgs {
		for _, arg := range marg.Args {
			v := arg.Value
			b.AddMetaArg(arg.Key, v)
		}
	}
}

// AddPreDefinedBuildArgs adds pre-defined build args. Such as TARGETOS, TARGETARCH, BUILDPLATFORM, TARGETPLATFORM
func (b *BuildArgs) AddPreDefinedBuildArgs(opts *config.KanikoOptions) {
	buildPlatform := runtime.GOOS + "/" + runtime.GOARCH
	buildOs := runtime.GOOS
	buildArch := runtime.GOARCH

	targetPlatform := ""
	targetOs := ""
	targetArch := ""
	targetVariant := ""

	if opts.CustomPlatform == "" {
		targetPlatform = buildPlatform
		targetOs = buildOs
		targetArch = buildArch
	} else {
		targetPlatform = opts.CustomPlatform
		platformParts := strings.Split(opts.CustomPlatform, "/")
		if len(platformParts) > 0 {
			targetOs = platformParts[0]
		}
		if len(platformParts) > 1 {
			targetArch = platformParts[1]
		}
		if len(platformParts) > 2 {
			targetVariant = platformParts[2]
		}
	}

	b.AddArg("BUILDPLATFORM", &buildPlatform)
	b.AddArg("BUILDOS", &buildOs)
	b.AddArg("BUILDARCH", &buildArch)

	b.AddArg("TARGETPLATFORM", &targetPlatform)
	b.AddArg("TARGETOS", &targetOs)
	b.AddArg("TARGETARCH", &targetArch)
	b.AddArg("TARGETVARIANT", &targetVariant)
}
