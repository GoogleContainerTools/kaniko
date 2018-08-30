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

package options

// KanikoOptions are options that are set by command line arguments
type KanikoOptions struct {
	DockerfilePath string
	Destinations   multiArg
	SrcContext     string
	SnapshotMode   string
	Bucket         string
	InsecurePush   bool
	SkipTlsVerify  bool
	BuildArgs      multiArg
	TarPath        string
	SingleSnapshot bool
	Reproducible   bool
	Target         string
	NoPush         bool
}
