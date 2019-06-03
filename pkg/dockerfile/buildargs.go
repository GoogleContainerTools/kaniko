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
	"strings"

	d "github.com/docker/docker/builder/dockerfile"
	"github.com/moby/buildkit/frontend/dockerfile/instructions"
	"github.com/moby/buildkit/frontend/dockerfile/parser"
	"github.com/moby/buildkit/frontend/dockerfile/shell"
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
	filtered := b.FilterAllowed(envs)
	return append(envs, filtered...)
}

// AddMetaArgs adds the supplied args map to b's allowedMetaArgs
func (b *BuildArgs) AddMetaArgs(metaArgs []instructions.ArgCommand) {
	for _, arg := range metaArgs {
		v := arg.Value
		b.AddMetaArg(arg.Key, v)
	}
}

func (b *BuildArgs) processMetaArgs(metaArgs []instructions.ArgCommand) error {
	shlex := shell.NewLex(parser.DefaultEscapeToken)
	for _, meta := range metaArgs {
		var envs []string
		for k, v := range b.GetAllAllowed() {
			envs = append(envs, k+"="+v)
		}
		err := meta.Expand(func(word string) (string, error) {
			return shlex.ProcessWord(word, envs)
		})
		if err != nil {
			return err
		}
		b.AddArg(meta.Key, meta.Value)
		b.AddMetaArg(meta.Key, meta.Value)
	}
	return nil
}
