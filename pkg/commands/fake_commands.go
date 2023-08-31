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

// used for testing in the commands package
package commands

import (
	"bytes"
	"io"

	v1 "github.com/google/go-containerregistry/pkg/v1"
	"github.com/google/go-containerregistry/pkg/v1/types"
)

type fakeLayer struct {
	TarContent []byte
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
	return "", nil
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
