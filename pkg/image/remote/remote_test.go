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
	"errors"
	"testing"

	"github.com/GoogleContainerTools/kaniko/pkg/config"
	"github.com/google/go-containerregistry/pkg/name"
	v1 "github.com/google/go-containerregistry/pkg/v1"
	"github.com/google/go-containerregistry/pkg/v1/remote"
	"github.com/google/go-containerregistry/pkg/v1/types"
)

const image string = "debian"

// mockImage mocks the v1.Image interface
type mockImage struct {
}

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
	expected := "index.docker.io/library/debian:latest"

	ref, err := name.ParseReference(image)
	if err != nil {
		t.Fatal(err)
	}

	ref2, err := normalizeReference(ref, image, "library")
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

func Test_RetrieveRemoteImage_skipFallback(t *testing.T) {
	registryMirror := "some-registry"

	opts := config.RegistryOptions{
		RegistryMaps:                map[string][]string{name.DefaultRegistry: {registryMirror}},
		SkipDefaultRegistryFallback: false,
	}

	remoteImageFunc = func(ref name.Reference, options ...remote.Option) (v1.Image, error) {
		if ref.Context().Registry.Name() == registryMirror {
			return nil, errors.New("no image found")
		}

		return &mockImage{}, nil
	}

	if _, err := RetrieveRemoteImage(image, opts, ""); err != nil {
		t.Fatal("Expected call to succeed because fallback to default registry")
	}

	opts.SkipDefaultRegistryFallback = true
	//clean cached image
	manifestCache = make(map[string]v1.Image)

	if _, err := RetrieveRemoteImage(image, opts, ""); err == nil {
		t.Fatal("Expected call to fail because fallback to default registry is skipped")
	}
}

func Test_RetryRetrieveRemoteImageSucceeds(t *testing.T) {
	opts := config.RegistryOptions{
		ImageDownloadRetry: 2,
	}
	attempts := 0
	remoteImageFunc = func(ref name.Reference, options ...remote.Option) (v1.Image, error) {
		if attempts < 2 {
			attempts++
			return nil, errors.New("no image found")
		}
		return &mockImage{}, nil
	}

	// Clean cached image
	manifestCache = make(map[string]v1.Image)

	if _, err := RetrieveRemoteImage(image, opts, ""); err != nil {
		t.Fatal("Expected call to succeed because of retry")
	}
}

func Test_NoRetryRetrieveRemoteImageFails(t *testing.T) {
	opts := config.RegistryOptions{
		ImageDownloadRetry: 0,
	}
	attempts := 0
	remoteImageFunc = func(ref name.Reference, options ...remote.Option) (v1.Image, error) {
		if attempts < 1 {
			attempts++
			return nil, errors.New("no image found")
		}
		return &mockImage{}, nil
	}

	// Clean cached image
	manifestCache = make(map[string]v1.Image)

	if _, err := RetrieveRemoteImage(image, opts, ""); err == nil {
		t.Fatal("Expected call to fail because there is no retry")
	}
}

func Test_ExtractPathFromRegistryURL(t *testing.T) {
	tests := []struct {
		name                string
		regFullURL          string
		expectedRegistry    string
		expectedImagePrefix string
	}{
		{
			name:                "Test case 1",
			regFullURL:          "registry.example.com/namespace",
			expectedRegistry:    "registry.example.com",
			expectedImagePrefix: "namespace",
		},
		{
			name:                "Test case 2",
			regFullURL:          "registry.example.com",
			expectedRegistry:    "registry.example.com",
			expectedImagePrefix: "",
		},
		// Add more test cases here
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			path, registry := extractPathFromRegistryURL(tt.regFullURL)
			if registry != tt.expectedRegistry {
				t.Errorf("Expected path: %s, but got: %s", tt.expectedRegistry, registry)
			}
			if path != tt.expectedImagePrefix {
				t.Errorf("Expected repo: %s, but got: %s", tt.expectedImagePrefix, path)
			}
		})
	}
}
