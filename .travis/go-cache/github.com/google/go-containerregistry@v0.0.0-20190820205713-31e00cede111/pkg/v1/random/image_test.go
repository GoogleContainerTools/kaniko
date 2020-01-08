// Copyright 2018 Google LLC All Rights Reserved.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//    http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package random

import (
	"archive/tar"
	"io"
	"io/ioutil"
	"testing"

	"github.com/google/go-containerregistry/pkg/v1/types"
)

func TestManifestAndConfig(t *testing.T) {
	want := int64(12)
	img, err := Image(1024, want)
	if err != nil {
		t.Fatalf("Error loading image: %v", err)
	}
	manifest, err := img.Manifest()
	if err != nil {
		t.Fatalf("Error loading manifest: %v", err)
	}
	if got := int64(len(manifest.Layers)); got != want {
		t.Fatalf("num layers; got %v, want %v", got, want)
	}

	config, err := img.ConfigFile()
	if err != nil {
		t.Fatalf("Error loading config file: %v", err)
	}
	if got := int64(len(config.RootFS.DiffIDs)); got != want {
		t.Fatalf("num diff ids; got %v, want %v", got, want)
	}
}

func TestTarLayer(t *testing.T) {
	img, err := Image(1024, 5)
	if err != nil {
		t.Fatalf("Image: %v", err)
	}
	layers, err := img.Layers()
	if err != nil {
		t.Fatalf("Layers: %v", err)
	}
	if len(layers) != 5 {
		t.Errorf("Got %d layers, want 5", len(layers))
	}
	for i, l := range layers {
		mediaType, err := l.MediaType()
		if err != nil {
			t.Fatalf("MediaType: %v", err)
		}
		if got, want := mediaType, types.DockerLayer; got != want {
			t.Fatalf("MediaType(); got %q, want %q", got, want)
		}

		rc, err := l.Uncompressed()
		if err != nil {
			t.Errorf("Uncompressed(%d): %v", i, err)
		}
		defer rc.Close()
		tr := tar.NewReader(rc)
		if _, err := tr.Next(); err != nil {
			t.Errorf("tar.Next: %v", err)
		}

		if n, err := io.Copy(ioutil.Discard, tr); err != nil {
			t.Errorf("Reading tar layer: %v", err)
		} else if n != 1024 {
			t.Errorf("Layer %d was %d bytes, want 1024", i, n)
		}

		if _, err := tr.Next(); err != io.EOF {
			t.Errorf("Layer contained more files; got %v, want EOF", err)
		}
	}
}
