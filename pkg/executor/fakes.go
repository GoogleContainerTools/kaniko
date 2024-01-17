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

// for use in tests
package executor

import (
	"bytes"
	"errors"
	"io"

	"github.com/GoogleContainerTools/kaniko/pkg/commands"
	"github.com/GoogleContainerTools/kaniko/pkg/dockerfile"
	v1 "github.com/google/go-containerregistry/pkg/v1"
	"github.com/google/go-containerregistry/pkg/v1/types"
)

type fakeSnapShotter struct {
	file        string
	tarPath     string
	initialized bool
}

func (f *fakeSnapShotter) Init() error {
	f.initialized = true
	return nil
}
func (f *fakeSnapShotter) TakeSnapshotFS() (string, error) {
	return f.tarPath, nil
}
func (f *fakeSnapShotter) TakeSnapshot(_ []string, _, _ bool) (string, error) {
	return f.tarPath, nil
}

type MockDockerCommand struct {
	command             string
	contextFiles        []string
	cacheCommand        commands.DockerCommand
	argToCompositeCache bool
}

func (m MockDockerCommand) ExecuteCommand(c *v1.Config, args *dockerfile.BuildArgs) error { return nil }
func (m MockDockerCommand) String() string {
	return m.command
}
func (m MockDockerCommand) FilesToSnapshot() []string {
	return []string{"meow-snapshot-no-cache"}
}
func (m MockDockerCommand) ProvidesFilesToSnapshot() bool {
	return true
}
func (m MockDockerCommand) CacheCommand(image v1.Image) commands.DockerCommand {
	return m.cacheCommand
}
func (m MockDockerCommand) FilesUsedFromContext(c *v1.Config, args *dockerfile.BuildArgs) ([]string, error) {
	return m.contextFiles, nil
}
func (m MockDockerCommand) MetadataOnly() bool {
	return false
}
func (m MockDockerCommand) RequiresUnpackedFS() bool {
	return false
}
func (m MockDockerCommand) ShouldCacheOutput() bool {
	return true
}
func (m MockDockerCommand) ShouldDetectDeletedFiles() bool {
	return false
}
func (m MockDockerCommand) IsArgsEnvsRequiredInCache() bool {
	return m.argToCompositeCache
}

type MockCachedDockerCommand struct {
	contextFiles        []string
	argToCompositeCache bool
}

func (m MockCachedDockerCommand) ExecuteCommand(c *v1.Config, args *dockerfile.BuildArgs) error {
	return nil
}
func (m MockCachedDockerCommand) String() string {
	return "meow"
}
func (m MockCachedDockerCommand) FilesToSnapshot() []string {
	return []string{"meow-snapshot"}
}
func (m MockCachedDockerCommand) ProvidesFilesToSnapshot() bool {
	return true
}
func (m MockCachedDockerCommand) CacheCommand(image v1.Image) commands.DockerCommand {
	return nil
}
func (m MockCachedDockerCommand) ShouldDetectDeletedFiles() bool {
	return false
}
func (m MockCachedDockerCommand) FilesUsedFromContext(c *v1.Config, args *dockerfile.BuildArgs) ([]string, error) {
	return m.contextFiles, nil
}
func (m MockCachedDockerCommand) MetadataOnly() bool {
	return false
}
func (m MockCachedDockerCommand) RequiresUnpackedFS() bool {
	return false
}
func (m MockCachedDockerCommand) ShouldCacheOutput() bool {
	return false
}
func (m MockCachedDockerCommand) IsArgsEnvsRequiredInCache() bool {
	return m.argToCompositeCache
}

type fakeLayerCache struct {
	retrieve     bool
	receivedKeys []string
	img          v1.Image
	keySequence  []string
}

func (f *fakeLayerCache) RetrieveLayer(key string) (v1.Image, error) {
	f.receivedKeys = append(f.receivedKeys, key)
	if len(f.keySequence) > 0 {
		if f.keySequence[0] == key {
			f.keySequence = f.keySequence[1:]
			return f.img, nil
		}
		return f.img, errors.New("could not find layer")
	}

	if !f.retrieve {
		return nil, errors.New("could not find layer")
	}
	return f.img, nil
}

type fakeLayer struct {
	TarContent []byte
	mediaType  types.MediaType
}

func (f fakeLayer) Digest() (v1.Hash, error) {
	return v1.Hash{}, nil
}
func (f fakeLayer) DiffID() (v1.Hash, error) {
	return v1.Hash{}, nil
}
func (f fakeLayer) Compressed() (io.ReadCloser, error) {
	return nil, nil
}
func (f fakeLayer) Uncompressed() (io.ReadCloser, error) {
	return io.NopCloser(bytes.NewReader(f.TarContent)), nil
}
func (f fakeLayer) Size() (int64, error) {
	return 0, nil
}
func (f fakeLayer) MediaType() (types.MediaType, error) {
	return f.mediaType, nil
}

type fakeImage struct {
	ImageLayers []v1.Layer
}

func (f fakeImage) Layers() ([]v1.Layer, error) {
	return f.ImageLayers, nil
}
func (f fakeImage) MediaType() (types.MediaType, error) {
	return "", nil
}
func (f fakeImage) Size() (int64, error) {
	return 0, nil
}
func (f fakeImage) ConfigName() (v1.Hash, error) {
	return v1.Hash{}, nil
}
func (f fakeImage) ConfigFile() (*v1.ConfigFile, error) {
	return &v1.ConfigFile{}, nil
}
func (f fakeImage) RawConfigFile() ([]byte, error) {
	return []byte{}, nil
}
func (f fakeImage) Digest() (v1.Hash, error) {
	return v1.Hash{}, nil
}
func (f fakeImage) Manifest() (*v1.Manifest, error) {
	return &v1.Manifest{}, nil
}
func (f fakeImage) RawManifest() ([]byte, error) {
	return []byte{}, nil
}
func (f fakeImage) LayerByDigest(v1.Hash) (v1.Layer, error) {
	return fakeLayer{}, nil
}
func (f fakeImage) LayerByDiffID(v1.Hash) (v1.Layer, error) {
	return fakeLayer{}, nil
}

type ociFakeImage struct {
	*fakeImage
}

func (f ociFakeImage) MediaType() (types.MediaType, error) {
	return types.OCIManifestSchema1, nil
}

type dockerFakeImage struct {
	*fakeImage
}

func (f dockerFakeImage) MediaType() (types.MediaType, error) {
	return types.DockerManifestSchema2, nil
}
