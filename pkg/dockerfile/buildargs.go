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
	"fmt"
	"strings"
	"sync"

	"github.com/GoogleContainerTools/kaniko/pkg/config"
	"github.com/containerd/platforms"
	d "github.com/docker/docker/builder/dockerfile"
	containerregistryV1 "github.com/google/go-containerregistry/pkg/v1"
	"github.com/moby/buildkit/frontend/dockerfile/instructions"
	"github.com/sirupsen/logrus"
)

var defaultArgsOnce sync.Once
var defaultArgs []string

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

// DefaultArgs get default args.
// include pre-defined build args: TARGETOS, TARGETARCH, BUILDPLATFORM, TARGETPLATFORM ...
func DefaultArgs(opt config.KanikoOptions) []string {
	defaultArgsOnce.Do(func() {
		// build info
		buildPlatform := platforms.Format(platforms.Normalize(platforms.DefaultSpec()))
		buildPlatformSpec, err := containerregistryV1.ParsePlatform(buildPlatform)
		if err != nil {
			logrus.Fatalf("Parse build platform %q: %v", buildPlatform, err)
		}

		// target info
		var targetPlatform string
		if opt.CustomPlatform != "" {
			targetPlatform = opt.CustomPlatform
		} else {
			targetPlatform = buildPlatform
		}
		targetPlatformSpec, err := containerregistryV1.ParsePlatform(opt.CustomPlatform)
		if err != nil {
			logrus.Fatalf("Invalid platform %q: %v", opt.CustomPlatform, err)
		}

		// pre-defined build args
		defaultArgs = []string{
			fmt.Sprintf("%s=%s", "BUILDPLATFORM", buildPlatform),
			fmt.Sprintf("%s=%s", "BUILDOS", buildPlatformSpec.OS),
			fmt.Sprintf("%s=%s", "BUILDOSVERSION", buildPlatformSpec.OSVersion),
			fmt.Sprintf("%s=%s", "BUILDARCH", buildPlatformSpec.Architecture),
			fmt.Sprintf("%s=%s", "BUILDVARIANT", buildPlatformSpec.Variant),
			fmt.Sprintf("%s=%s", "TARGETPLATFORM", targetPlatform),
			fmt.Sprintf("%s=%s", "TARGETOS", targetPlatformSpec.OS),
			fmt.Sprintf("%s=%s", "TARGETOSVERSION", targetPlatformSpec.OSVersion),
			fmt.Sprintf("%s=%s", "TARGETARCH", targetPlatformSpec.Architecture),
			fmt.Sprintf("%s=%s", "TARGETVARIANT", targetPlatformSpec.Variant),
			fmt.Sprintf("%s=%s", "TARGETSTAGE", opt.Target),
		}
		logrus.Infof("init default args: %s", defaultArgs)
	})
	return defaultArgs
}
