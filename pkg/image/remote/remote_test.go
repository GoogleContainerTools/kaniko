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

package remote

import (
	"testing"

	"github.com/GoogleContainerTools/kaniko/pkg/config"
	"github.com/google/go-containerregistry/pkg/name"
	v1 "github.com/google/go-containerregistry/pkg/v1"
	"github.com/google/go-containerregistry/pkg/v1/types"
)

// mockImage mocks the v1.Image interface
type mockImage struct{}

func (m *mockImage) ConfigFile() (*v1.ConfigFile, error) {
	return nil, nil
}

func (m *mockImage) ConfigName() (v1.Hash, error) {
	return v1.Hash{}, nil
}

func (m *mockImage) Descriptor() (*v1.Descriptor, error) {
	return nil, nil
}

func (m *mockImage) Digest() (v1.Hash, error) {
	return v1.Hash{}, nil
}

func (m *mockImage) LayerByDigest(v1.Hash) (v1.Layer, error) {
	return nil, nil
}

func (m *mockImage) LayerByDiffID(v1.Hash) (v1.Layer, error) {
	return nil, nil
}

func (m *mockImage) Layers() ([]v1.Layer, error) {
	return nil, nil
}

func (m *mockImage) Manifest() (*v1.Manifest, error) {
	return nil, nil
}

func (m *mockImage) MediaType() (types.MediaType, error) {
	return "application/vnd.oci.descriptor.v1+json", nil
}

func (m *mockImage) RawManifest() ([]byte, error) {
	return nil, nil
}

func (m *mockImage) RawConfigFile() ([]byte, error) {
	return nil, nil
}

func (m *mockImage) Size() (int64, error) {
	return 0, nil
}

func Test_normalizeReference(t *testing.T) {
	image := "debian"
	expected := "index.docker.io/library/debian:latest"

	ref, err := name.ParseReference(image)
	if err != nil {
		t.Fatal(err)
	}

	ref2, err := normalizeReference(ref, image)
	if err != nil {
		t.Fatal(err)
	}

	if ref2.Name() != ref.Name() || ref2.Name() != expected {
		t.Errorf("%s should have been normalized to %s, got %s", ref2.Name(), expected, ref.Name())
	}
}

func Test_RetrieveRemoteImage_manifestCache(t *testing.T) {
	nonExistingImageName := "this_is_a_non_existing_image_reference"

	if _, err := RetrieveRemoteImage(nonExistingImageName, config.RegistryOptions{}, ""); err == nil {
		t.Fatal("Expected call to fail because there is no manifest for this image.")
	}

	manifestCache[nonExistingImageName] = &mockImage{}

	if image, err := RetrieveRemoteImage(nonExistingImageName, config.RegistryOptions{}, ""); image == nil || err != nil {
		t.Fatal("Expected call to succeed because there is a manifest for this image in the cache.")
	}
}
