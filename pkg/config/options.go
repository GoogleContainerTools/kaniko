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
	OCILayoutPath           string
	Destinations            multiArg
	BuildArgs               multiArg
	Insecure                bool
	SkipTLSVerify           bool
	InsecurePull            bool
	SkipTLSVerifyPull       bool
	SingleSnapshot          bool
	Reproducible            bool
	NoPush                  bool
	Cache                   bool
	Cleanup                 bool
	InsecureRegistries      multiArg
	SkipTLSVerifyRegistries multiArg
}

// WarmerOptions are options that are set by command line arguments to the cache warmer.
type WarmerOptions struct {
	CacheOptions
	Images multiArg
	Force  bool
}
