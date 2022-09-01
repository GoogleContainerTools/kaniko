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

package config

import (
	"errors"
	"fmt"
	"strconv"
	"strings"
	"time"
)

// CacheOptions are base image cache options that are set by command line arguments
type CacheOptions struct {
	CacheDir string
	CacheTTL time.Duration
}

// RegistryOptions are all the options related to the registries, set by command line arguments.
type RegistryOptions struct {
	RegistryMirrors         multiArg
	InsecureRegistries      multiArg
	SkipTLSVerifyRegistries multiArg
	RegistriesCertificates  keyValueArg
	Insecure                bool
	SkipTLSVerify           bool
	InsecurePull            bool
	SkipTLSVerifyPull       bool
	PushRetry               int
}

// KanikoOptions are options that are set by command line arguments
type KanikoOptions struct {
	RegistryOptions
	CacheOptions
	Destinations             multiArg
	BuildArgs                multiArg
	Labels                   multiArg
	Git                      KanikoGitOptions
	IgnorePaths              multiArg
	DockerfilePath           string
	SrcContext               string
	SnapshotMode             string
	SnapshotModeDeprecated   string
	CustomPlatform           string
	CustomPlatformDeprecated string
	Bucket                   string
	TarPath                  string
	TarPathDeprecated        string
	KanikoDir                string
	Target                   string
	CacheRepo                string
	DigestFile               string
	ImageNameDigestFile      string
	ImageNameTagDigestFile   string
	OCILayoutPath            string
	ImageFSExtractRetry      int
	SingleSnapshot           bool
	Reproducible             bool
	NoPush                   bool
	NoPushCache              bool
	Cache                    bool
	Cleanup                  bool
	CompressedCaching        bool
	IgnoreVarRun             bool
	SkipUnusedStages         bool
	RunV2                    bool
	CacheCopyLayers          bool
	CacheRunLayers           bool
	ForceBuildMetadata       bool
	InitialFSUnpacked        bool
}

type KanikoGitOptions struct {
	Branch            string
	SingleBranch      bool
	RecurseSubmodules bool
}

var ErrInvalidGitFlag = errors.New("invalid git flag, must be in the key=value format")

func (k *KanikoGitOptions) Type() string {
	return "gitoptions"
}

func (k *KanikoGitOptions) String() string {
	return fmt.Sprintf("branch=%s,single-branch=%t,recurse-submodules=%t", k.Branch, k.SingleBranch, k.RecurseSubmodules)
}

func (k *KanikoGitOptions) Set(s string) error {
	var parts = strings.SplitN(s, "=", 2)
	if len(parts) != 2 {
		return fmt.Errorf("%w: %s", ErrInvalidGitFlag, s)
	}
	switch parts[0] {
	case "branch":
		k.Branch = parts[1]
	case "single-branch":
		v, err := strconv.ParseBool(parts[1])
		if err != nil {
			return err
		}
		k.SingleBranch = v
	case "recurse-submodules":
		v, err := strconv.ParseBool(parts[1])
		if err != nil {
			return err
		}
		k.RecurseSubmodules = v
	}
	return nil
}

// WarmerOptions are options that are set by command line arguments to the cache warmer.
type WarmerOptions struct {
	CacheOptions
	RegistryOptions
	CustomPlatform string
	Images         multiArg
	Force          bool
}
