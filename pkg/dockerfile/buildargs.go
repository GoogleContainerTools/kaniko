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

	"github.com/moby/buildkit/frontend/dockerfile/instructions"
)

// builtinAllowedBuildArgs is list of built-in allowed build args
// these args are considered transparent and are excluded from the image history.
// Filtering from history is implemented in dispatchers.go
var builtinAllowedBuildArgs = map[string]bool{
	"HTTP_PROXY":  true,
	"http_proxy":  true,
	"HTTPS_PROXY": true,
	"https_proxy": true,
	"FTP_PROXY":   true,
	"ftp_proxy":   true,
	"NO_PROXY":    true,
	"no_proxy":    true,
	"ALL_PROXY":   true,
	"all_proxy":   true,
}

// BuildArgs manages arguments used by the builder
type BuildArgs struct {
	// args that are allowed for expansion/substitution and passing to commands in 'run'.
	allowedBuildArgs map[string]*string
	// args defined before the first `FROM` in a Dockerfile
	allowedMetaArgs map[string]*string
	// args referenced by the Dockerfile
	referencedArgs map[string]struct{}
	// args provided by the user on the command line
	argsFromOptions map[string]*string
}

func NewBuildArgs(args []string) *BuildArgs {
	argsFromOptions := convertKVStringsToMap(args)
	return newBuildArgsFromMap(argsFromOptions)
}

// newBuildArgsFromMap creates a new BuildArgs type
func newBuildArgsFromMap(argsFromOptions map[string]*string) *BuildArgs {
	return &BuildArgs{
		allowedBuildArgs: make(map[string]*string),
		allowedMetaArgs:  make(map[string]*string),
		referencedArgs:   make(map[string]struct{}),
		argsFromOptions:  argsFromOptions,
	}
}

// Clone returns a copy of the BuildArgs type
func (b *BuildArgs) Clone() *BuildArgs {
	result := newBuildArgsFromMap(b.argsFromOptions)
	for k, v := range b.allowedBuildArgs {
		result.allowedBuildArgs[k] = v
	}
	for k, v := range b.allowedMetaArgs {
		result.allowedMetaArgs[k] = v
	}
	for k := range b.referencedArgs {
		result.referencedArgs[k] = struct{}{}
	}
	return result
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
			b.allowedMetaArgs[arg.Key] = v
		}
	}
}

// AddArg adds a new arg that can be used by directives
func (b *BuildArgs) AddArg(key string, value *string) {
	b.allowedBuildArgs[key] = value
	b.referencedArgs[key] = struct{}{}
}

// GetAllAllowed returns a mapping with all the allowed args
func (b *BuildArgs) GetAllAllowed() map[string]string {
	return b.getAllFromMapping(b.allowedBuildArgs)
}

// GetAllMeta returns a mapping with all the meta args
func (b *BuildArgs) GetAllMeta() map[string]string {
	return b.getAllFromMapping(b.allowedMetaArgs)
}

func (b *BuildArgs) getAllFromMapping(source map[string]*string) map[string]string {
	m := make(map[string]string)

	keys := keysFromMaps(source, builtinAllowedBuildArgs)
	for _, key := range keys {
		v, ok := b.getBuildArg(key, source)
		if ok {
			m[key] = v
		}
	}
	return m
}

// FilterAllowed returns all allowed args without the filtered args
func (b *BuildArgs) FilterAllowed(filter []string) []string {
	envs := []string{}
	configEnv := convertKVStringsToMap(filter)

	for key, val := range b.GetAllAllowed() {
		if _, ok := configEnv[key]; !ok {
			envs = append(envs, fmt.Sprintf("%s=%s", key, val))
		}
	}
	return envs
}

func (b *BuildArgs) getBuildArg(key string, mapping map[string]*string) (string, bool) {
	defaultValue, exists := mapping[key]
	// Return override from options if one is defined
	if v, ok := b.argsFromOptions[key]; ok && v != nil {
		return *v, ok
	}

	if defaultValue == nil {
		if v, ok := b.allowedMetaArgs[key]; ok && v != nil {
			return *v, ok
		}
		return "", false
	}
	return *defaultValue, exists
}

func keysFromMaps(source map[string]*string, builtin map[string]bool) []string {
	keys := []string{}
	for key := range source {
		keys = append(keys, key)
	}
	for key := range builtin {
		keys = append(keys, key)
	}
	return keys
}

// convertKVStringsToMap converts ["key=value"] to {"key":"value"}
func convertKVStringsToMap(values []string) map[string]*string {
	result := make(map[string]*string, len(values))
	for _, value := range values {
		kv := strings.SplitN(value, "=", 2)
		if len(kv) == 1 {
			result[kv[0]] = nil
		} else {
			result[kv[0]] = &kv[1]
		}
	}
	return result
}
