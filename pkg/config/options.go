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

// KanikoOptions are options that are set by command line arguments
type KanikoOptions struct {
	CacheOptions
	DockerfilePath          string
	SrcContext              string
	SnapshotMode            string
	Bucket                  string
	TarPath                 string
	Target                  string
	CacheRepo               string
	DigestFile              string
	ImageNameDigestFile     string
	OCILayoutPath           string
	RegistryMirror          string
	Destinations            multiArg
	BuildArgs               multiArg
	InsecureRegistries      multiArg
	Labels                  multiArg
	SkipTLSVerifyRegistries multiArg
	RegistriesCertificates  keyValueArg
	Insecure                bool
	SkipTLSVerify           bool
	InsecurePull            bool
	SkipTLSVerifyPull       bool
	SingleSnapshot          bool
	Reproducible            bool
	NoPush                  bool
	Cache                   bool
	Cleanup                 bool
	IgnoreVarRun            bool
	SkipUnusedStages        bool
	RunV2                   bool
	Git                     KanikoGitOptions
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
	Images multiArg
	Force  bool
}
