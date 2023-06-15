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
	v1 "github.com/google/go-containerregistry/pkg/v1"
)

type BaseCommand struct {
}

func (b *BaseCommand) IsArgsEnvsRequiredInCache() bool {
	return false
}

func (b *BaseCommand) CacheCommand(v1.Image) DockerCommand {
	return nil
}

func (b *BaseCommand) FilesToSnapshot() []string {
	return []string{}
}

func (b *BaseCommand) ProvidesFilesToSnapshot() bool {
	return true
}

func (b *BaseCommand) FilesUsedFromContext(_ *v1.Config, _ *dockerfile.BuildArgs) ([]string, error) {
	return []string{}, nil
}

func (b *BaseCommand) MetadataOnly() bool {
	return true
}

func (b *BaseCommand) RequiresUnpackedFS() bool {
	return false
}

func (b *BaseCommand) ShouldCacheOutput() bool {
	return false
}

func (b *BaseCommand) ShouldDetectDeletedFiles() bool {
	return false
}
