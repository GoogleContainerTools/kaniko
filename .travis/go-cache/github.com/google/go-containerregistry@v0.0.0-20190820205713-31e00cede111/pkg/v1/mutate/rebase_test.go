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

package mutate

import (
	"testing"
	"time"

	v1 "github.com/google/go-containerregistry/pkg/v1"
	"github.com/google/go-containerregistry/pkg/v1/random"
)

func layerDigests(t *testing.T, img v1.Image) []string {
	var layerDigests []string
	layers, err := img.Layers()
	if err != nil {
		t.Fatalf("oldBase.Layers: %v", err)
	}
	for i, l := range layers {
		dig, err := l.Digest()
		if err != nil {
			t.Fatalf("layer.Digest %d: %v", i, err)
		}
		t.Log(dig)
		layerDigests = append(layerDigests, dig.String())
	}
	return layerDigests
}

// TestRebase tests that layer digests are expected when performing a rebase on
// random.Image layers.
func TestRebase(t *testing.T) {
	// Create a random old base image of 5 layers and get those layers' digests.
	oldBase, err := random.Image(100, 5)
	if err != nil {
		t.Fatalf("random.Image (oldBase): %v", err)
	}
	t.Log("Old base:")
	_ = layerDigests(t, oldBase)

	// Construct an image with 1 layer on top of oldBase.
	top, err := random.Image(100, 1)
	if err != nil {
		t.Fatalf("random.Image (top): %v", err)
	}
	topLayers, err := top.Layers()
	if err != nil {
		t.Fatalf("top.Layers: %v", err)
	}
	topHistory := v1.History{
		Author:    "me",
		Created:   v1.Time{time.Now()},
		CreatedBy: "test",
		Comment:   "this is a test",
	}
	orig, err := Append(oldBase, Addendum{
		Layer:   topLayers[0],
		History: topHistory,
	})
	if err != nil {
		t.Fatalf("Append: %v", err)
	}

	t.Log("Original:")
	origLayerDigests := layerDigests(t, orig)

	// Create a random new base image of 3 layers.
	newBase, err := random.Image(100, 3)
	if err != nil {
		t.Fatalf("random.Image (newBase): %v", err)
	}
	t.Log("New base:")
	newBaseLayerDigests := layerDigests(t, newBase)

	// Rebase original image onto new base.
	rebased, err := Rebase(orig, oldBase, newBase)
	if err != nil {
		t.Fatalf("Rebase: %v", err)
	}

	var rebasedLayerDigests []string
	rebasedBaseLayers, err := rebased.Layers()
	if err != nil {
		t.Fatalf("rebased.Layers: %v", err)
	}
	t.Log("Rebased image layer digests:")
	for i, l := range rebasedBaseLayers {
		dig, err := l.Digest()
		if err != nil {
			t.Fatalf("layer.Digest (rebased base layer %d): %v", i, err)
		}
		t.Log(dig)
		rebasedLayerDigests = append(rebasedLayerDigests, dig.String())
	}

	// Compare rebased layers.
	wantLayerDigests := append(newBaseLayerDigests, origLayerDigests[len(origLayerDigests)-1])
	if len(rebasedLayerDigests) != len(wantLayerDigests) {
		t.Fatalf("Rebased image contained %d layers, want %d", len(rebasedLayerDigests), len(wantLayerDigests))
	}
	for i, rl := range rebasedLayerDigests {
		if got, want := rl, wantLayerDigests[i]; got != want {
			t.Errorf("Layer %d mismatch, got %q, want %q", i, got, want)
		}
	}
}
