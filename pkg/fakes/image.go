/*
Copyright 2019 Google LLC

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

package fakes

import (
	v1 "github.com/google/go-containerregistry/pkg/v1"
	"github.com/google/go-containerregistry/pkg/v1/types"
)

type FakeImage struct {
	Hash v1.Hash
}

func (f FakeImage) Layers() ([]v1.Layer, error) {
	return nil, nil
}
func (f FakeImage) MediaType() (types.MediaType, error) {
	return "", nil
}
func (f FakeImage) Size() (int64, error) {
	return 0, nil
}
func (f FakeImage) ConfigName() (v1.Hash, error) {
	return v1.Hash{}, nil
}
func (f FakeImage) ConfigFile() (*v1.ConfigFile, error) {
	return &v1.ConfigFile{}, nil
}
func (f FakeImage) RawConfigFile() ([]byte, error) {
	return []byte{}, nil
}
func (f FakeImage) Digest() (v1.Hash, error) {
	return f.Hash, nil
}
func (f FakeImage) Manifest() (*v1.Manifest, error) {
	return &v1.Manifest{}, nil
}
func (f FakeImage) RawManifest() ([]byte, error) {
	return []byte{}, nil
}
func (f FakeImage) LayerByDigest(v1.Hash) (v1.Layer, error) {
	return nil, nil
}
func (f FakeImage) LayerByDiffID(v1.Hash) (v1.Layer, error) {
	return nil, nil
}
