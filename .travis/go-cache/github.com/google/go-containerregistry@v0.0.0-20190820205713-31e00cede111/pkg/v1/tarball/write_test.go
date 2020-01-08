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

package tarball

import (
	"io/ioutil"
	"os"
	"testing"

	"github.com/google/go-cmp/cmp"

	"github.com/google/go-containerregistry/pkg/name"
	v1 "github.com/google/go-containerregistry/pkg/v1"
	"github.com/google/go-containerregistry/pkg/v1/random"
)

func TestWrite(t *testing.T) {
	// Make a tempfile for tarball writes.
	fp, err := ioutil.TempFile("", "")
	if err != nil {
		t.Fatalf("Error creating temp file.")
	}
	t.Log(fp.Name())
	defer fp.Close()
	defer os.Remove(fp.Name())

	// Make a random image
	randImage, err := random.Image(256, 8)
	if err != nil {
		t.Fatalf("Error creating random image.")
	}
	tag, err := name.NewTag("gcr.io/foo/bar:latest", name.StrictValidation)
	if err != nil {
		t.Fatalf("Error creating test tag.")
	}
	if err := WriteToFile(fp.Name(), tag, randImage); err != nil {
		t.Fatalf("Unexpected error writing tarball: %v", err)
	}

	// Make sure the image is valid and can be loaded.
	// Load it both by nil and by its name.
	for _, it := range []*name.Tag{nil, &tag} {
		tarImage, err := ImageFromPath(fp.Name(), it)
		if err != nil {
			t.Fatalf("Unexpected error reading tarball: %v", err)
		}

		tarManifest, err := tarImage.Manifest()
		if err != nil {
			t.Fatalf("Unexpected error reading tarball: %v", err)
		}
		randManifest, err := randImage.Manifest()
		if err != nil {
			t.Fatalf("Unexpected error reading tarball: %v", err)
		}

		if diff := cmp.Diff(randManifest, tarManifest); diff != "" {
			t.Errorf("Manifests not equal. (-rand +tar) %s", diff)
		}

		assertImageLayersMatchManifestLayers(t, tarImage)
		assertLayersAreIdentical(t, randImage, tarImage)
	}

	// Try loading a different tag, it should error.
	fakeTag, err := name.NewTag("gcr.io/notthistag:latest", name.StrictValidation)
	if err != nil {
		t.Fatalf("Error generating tag: %v", err)
	}
	if _, err := ImageFromPath(fp.Name(), &fakeTag); err == nil {
		t.Errorf("Expected error loading tag %v from image", fakeTag)
	}
}

func TestMultiWriteSameImage(t *testing.T) {
	// Make a tempfile for tarball writes.
	fp, err := ioutil.TempFile("", "")
	if err != nil {
		t.Fatalf("Error creating temp file.")
	}
	t.Log(fp.Name())
	defer fp.Close()
	defer os.Remove(fp.Name())

	// Make a random image
	randImage, err := random.Image(256, 8)
	if err != nil {
		t.Fatalf("Error creating random image.")
	}

	// Make two tags that point to the random image above.
	tag1, err := name.NewTag("gcr.io/foo/bar:latest", name.StrictValidation)
	if err != nil {
		t.Fatalf("Error creating test tag1.")
	}
	tag2, err := name.NewTag("gcr.io/baz/bat:latest", name.StrictValidation)
	if err != nil {
		t.Fatalf("Error creating test tag2.")
	}
	dig3, err := name.NewDigest("gcr.io/baz/baz@sha256:e3b0c44298fc1c149afbf4c8996fb92427ae41e4649b934ca495991b7852b855", name.StrictValidation)
	if err != nil {
		t.Fatalf("Error creating test dig3.")
	}
	refToImage := make(map[name.Reference]v1.Image)
	refToImage[tag1] = randImage
	refToImage[tag2] = randImage
	refToImage[dig3] = randImage

	// Write the images with both tags to the tarball
	if err := MultiRefWriteToFile(fp.Name(), refToImage); err != nil {
		t.Fatalf("Unexpected error writing tarball: %v", err)
	}
	for ref := range refToImage {
		tag, ok := ref.(name.Tag)
		if !ok {
			continue
		}

		tarImage, err := ImageFromPath(fp.Name(), &tag)
		if err != nil {
			t.Fatalf("Unexpected error reading tarball: %v", err)
		}

		tarManifest, err := tarImage.Manifest()
		if err != nil {
			t.Fatalf("Unexpected error reading tarball: %v", err)
		}
		randManifest, err := randImage.Manifest()
		if err != nil {
			t.Fatalf("Unexpected error reading tarball: %v", err)
		}

		if diff := cmp.Diff(randManifest, tarManifest); diff != "" {
			t.Errorf("Manifests not equal. (-rand +tar) %s", diff)
		}

		assertImageLayersMatchManifestLayers(t, tarImage)
		assertLayersAreIdentical(t, randImage, tarImage)
	}
}

func TestMultiWriteDifferentImages(t *testing.T) {
	// Make a tempfile for tarball writes.
	fp, err := ioutil.TempFile("", "")
	if err != nil {
		t.Fatalf("Error creating temp file.")
	}
	t.Log(fp.Name())
	defer fp.Close()
	defer os.Remove(fp.Name())

	// Make a random image
	randImage1, err := random.Image(256, 8)
	if err != nil {
		t.Fatalf("Error creating random image 1.")
	}

	// Make another random image
	randImage2, err := random.Image(256, 8)
	if err != nil {
		t.Fatalf("Error creating random image 2.")
	}

	// Make another random image
	randImage3, err := random.Image(256, 8)
	if err != nil {
		t.Fatalf("Error creating random image 3.")
	}

	// Create two tags, one pointing to each image created.
	tag1, err := name.NewTag("gcr.io/foo/bar:latest", name.StrictValidation)
	if err != nil {
		t.Fatalf("Error creating test tag1.")
	}
	tag2, err := name.NewTag("gcr.io/baz/bat:latest", name.StrictValidation)
	if err != nil {
		t.Fatalf("Error creating test tag2.")
	}
	dig3, err := name.NewDigest("gcr.io/baz/baz@sha256:e3b0c44298fc1c149afbf4c8996fb92427ae41e4649b934ca495991b7852b855", name.StrictValidation)
	if err != nil {
		t.Fatalf("Error creating test dig3.")
	}
	refToImage := make(map[name.Reference]v1.Image)
	refToImage[tag1] = randImage1
	refToImage[tag2] = randImage2
	refToImage[dig3] = randImage3

	// Write both images to the tarball.
	if err := MultiRefWriteToFile(fp.Name(), refToImage); err != nil {
		t.Fatalf("Unexpected error writing tarball: %v", err)
	}
	for ref, img := range refToImage {
		tag, ok := ref.(name.Tag)
		if !ok {
			continue
		}

		tarImage, err := ImageFromPath(fp.Name(), &tag)
		if err != nil {
			t.Fatalf("Unexpected error reading tarball: %v", err)
		}

		tarManifest, err := tarImage.Manifest()
		if err != nil {
			t.Fatalf("Unexpected error reading tarball: %v", err)
		}
		randManifest, err := img.Manifest()
		if err != nil {
			t.Fatalf("Unexpected error reading tarball: %v", err)
		}

		if diff := cmp.Diff(randManifest, tarManifest); diff != "" {
			t.Errorf("Manifests not equal. (-rand +tar) %s", diff)
		}

		assertImageLayersMatchManifestLayers(t, tarImage)
		assertLayersAreIdentical(t, img, tarImage)
	}
}

func assertImageLayersMatchManifestLayers(t *testing.T, i v1.Image) {
	t.Helper()

	layers, err := i.Layers()
	if err != nil {
		t.Fatalf("error getting layers: %v", err)
	}

	digestsFromImage := make([]v1.Hash, len(layers))

	for i, layer := range layers {
		digest, err := layer.Digest()
		if err != nil {
			t.Fatalf("error getting digests: %v", err)
		}
		digestsFromImage[i] = digest
	}

	m, err := i.Manifest()
	if err != nil {
		t.Fatalf("error getting layers to compare: %v", err)
	}

	digestsFromManifest := make([]v1.Hash, 0, len(m.Layers))
	for _, layer := range m.Layers {
		digestsFromManifest = append(digestsFromManifest, layer.Digest)
	}

	if diff := cmp.Diff(digestsFromImage, digestsFromManifest); diff != "" {
		t.Fatalf("image.Layers() are not in the same order as "+
			"the image.Manifest().Layers (-image +manifest) %s", diff)
	}
}

func assertLayersAreIdentical(t *testing.T, a, b v1.Image) {
	t.Helper()

	aLayers, err := a.Layers()
	if err != nil {
		t.Fatalf("error getting layers to compare: %v", err)
	}

	bLayers, err := b.Layers()
	if err != nil {
		t.Fatalf("error getting layers to compare: %v", err)
	}

	if diff := cmp.Diff(getDigests(t, aLayers), getDigests(t, bLayers)); diff != "" {
		t.Fatalf("layers digests are not identical (-rand +tar) %s", diff)
	}

	if diff := cmp.Diff(getDiffIDs(t, aLayers), getDiffIDs(t, bLayers)); diff != "" {
		t.Fatalf("layers digests are not identical (-rand +tar) %s", diff)
	}
}

func getDigests(t *testing.T, layers []v1.Layer) []v1.Hash {
	t.Helper()

	digests := make([]v1.Hash, 0, len(layers))
	for _, layer := range layers {
		digest, err := layer.Digest()
		if err != nil {
			t.Fatalf("error getting digests: %s", err)
		}
		digests = append(digests, digest)
	}

	return digests
}

func getDiffIDs(t *testing.T, layers []v1.Layer) []v1.Hash {
	t.Helper()

	diffIDs := make([]v1.Hash, 0, len(layers))
	for _, layer := range layers {
		diffID, err := layer.DiffID()
		if err != nil {
			t.Fatalf("error getting diffID: %s", err)
		}
		diffIDs = append(diffIDs, diffID)
	}

	return diffIDs
}
