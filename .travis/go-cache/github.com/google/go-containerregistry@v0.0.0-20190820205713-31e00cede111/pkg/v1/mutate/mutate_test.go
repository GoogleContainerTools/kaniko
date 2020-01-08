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
	"archive/tar"
	"bytes"
	"errors"
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
	"time"

	"github.com/google/go-cmp/cmp"
	v1 "github.com/google/go-containerregistry/pkg/v1"
	"github.com/google/go-containerregistry/pkg/v1/stream"
	"github.com/google/go-containerregistry/pkg/v1/tarball"
	"github.com/google/go-containerregistry/pkg/v1/types"
)

func TestExtractWhiteout(t *testing.T) {
	img, err := tarball.ImageFromPath("testdata/whiteout_image.tar", nil)
	if err != nil {
		t.Errorf("Error loading image: %v", err)
	}
	tarPath, _ := filepath.Abs("img.tar")
	defer os.Remove(tarPath)
	tr := tar.NewReader(Extract(img))
	for {
		header, err := tr.Next()
		if err == io.EOF {
			break
		}
		name := header.Name
		for _, part := range filepath.SplitList(name) {
			if part == "foo" {
				t.Errorf("whiteout file found in tar: %v", name)
			}
		}
	}
}

func TestExtractOverwrittenFile(t *testing.T) {
	img, err := tarball.ImageFromPath("testdata/overwritten_file.tar", nil)
	if err != nil {
		t.Fatalf("Error loading image: %v", err)
	}
	tr := tar.NewReader(Extract(img))
	for {
		header, err := tr.Next()
		if err == io.EOF {
			break
		}
		name := header.Name
		if strings.Contains(name, "foo.txt") {
			var buf bytes.Buffer
			buf.ReadFrom(tr)
			if strings.Contains(buf.String(), "foo") {
				t.Errorf("Contents of file were not correctly overwritten")
			}
		}
	}
}

// TestExtractError tests that if there are any errors encountered
func TestExtractError(t *testing.T) {
	rc := Extract(invalidImage{})
	if _, err := io.Copy(ioutil.Discard, rc); err == nil {
		t.Errorf("rc.Read; got nil error")
	} else if !strings.Contains(err.Error(), errInvalidImage.Error()) {
		t.Errorf("rc.Read; got %v, want %v", err, errInvalidImage)
	}
}

// TestExtractPartialRead tests that the reader can be partially read (e.g.,
// tar headers) and closed without error.
func TestExtractPartialRead(t *testing.T) {
	rc := Extract(invalidImage{})
	if _, err := io.Copy(ioutil.Discard, io.LimitReader(rc, 1)); err != nil {
		t.Errorf("Could not read one byte from reader")
	}
	if err := rc.Close(); err != nil {
		t.Errorf("rc.Close: %v", err)
	}
}

// invalidImage is an image which returns an error when Layers() is called.
type invalidImage struct {
	v1.Image
}

var errInvalidImage = errors.New("invalid image")

func (invalidImage) Layers() ([]v1.Layer, error) {
	return nil, errInvalidImage
}

func TestWhiteoutDir(t *testing.T) {
	fsMap := map[string]bool{
		"baz":      true,
		"red/blue": true,
	}
	var tests = []struct {
		path     string
		whiteout bool
	}{
		{"usr/bin", false},
		{"baz/foo.txt", true},
		{"baz/bar/foo.txt", true},
		{"red/green", false},
		{"red/yellow.txt", false},
	}

	for _, tt := range tests {
		whiteout := inWhiteoutDir(fsMap, tt.path)
		if whiteout != tt.whiteout {
			t.Errorf("Whiteout %s: expected %v, but got %v", tt.path, tt.whiteout, whiteout)
		}
	}
}

func TestNoopCondition(t *testing.T) {
	source := sourceImage(t)

	result, err := AppendLayers(source, []v1.Layer{}...)
	if err != nil {
		t.Fatalf("Unexpected error creating a writable image: %v", err)
	}

	if !manifestsAreEqual(t, source, result) {
		t.Error("manifests are not the same")
	}

	if !configFilesAreEqual(t, source, result) {
		t.Fatal("config files are not the same")
	}
}

func TestAppendWithHistory(t *testing.T) {
	source := sourceImage(t)

	addendum := Addendum{
		Layer: mockLayer{},
		History: v1.History{
			Author: "dave",
		},
	}

	result, err := Append(source, addendum)
	if err != nil {
		t.Fatalf("failed to append: %v", err)
	}

	layers := getLayers(t, result)

	if diff := cmp.Diff(layers[1], mockLayer{}); diff != "" {
		t.Fatalf("correct layer was not appended (-got, +want) %v", diff)
	}

	if configSizesAreEqual(t, source, result) {
		t.Fatal("adding a layer MUST change the config file size")
	}

	cf := getConfigFile(t, result)

	if diff := cmp.Diff(cf.History[1], addendum.History); diff != "" {
		t.Fatalf("the appended history is not the same (-got, +want) %s", diff)
	}
}

func TestAppendLayers(t *testing.T) {
	source := sourceImage(t)
	result, err := AppendLayers(source, mockLayer{})
	if err != nil {
		t.Fatalf("failed to append a layer: %v", err)
	}

	if manifestsAreEqual(t, source, result) {
		t.Fatal("appending a layer did not mutate the manifest")
	}

	if configFilesAreEqual(t, source, result) {
		t.Fatal("appending a layer did not mutate the config file")
	}

	if configSizesAreEqual(t, source, result) {
		t.Fatal("adding a layer MUST change the config file size")
	}

	layers := getLayers(t, result)

	if got, want := len(layers), 2; got != want {
		t.Fatalf("Layers did not return the appended layer "+
			"- got size %d; expected 2", len(layers))
	}

	if diff := cmp.Diff(layers[1], mockLayer{}); diff != "" {
		t.Fatalf("correct layer was not appended (-got, +want) %v", diff)
	}

	assertLayerOrderMatchesConfig(t, result)
	assertLayerOrderMatchesManifest(t, result)
	assertQueryingForLayerSucceeds(t, result, layers[1])
}

func TestMutateConfig(t *testing.T) {
	source := sourceImage(t)
	cfg, err := source.ConfigFile()
	if err != nil {
		t.Fatalf("error getting source config file")
	}

	newEnv := []string{"foo=bar"}
	cfg.Config.Env = newEnv
	result, err := Config(source, cfg.Config)
	if err != nil {
		t.Fatalf("failed to mutate a config: %v", err)
	}

	if manifestsAreEqual(t, source, result) {
		t.Error("mutating the config MUST mutate the manifest")
	}

	if configFilesAreEqual(t, source, result) {
		t.Error("mutating the config did not mutate the config file")
	}

	if configSizesAreEqual(t, source, result) {
		t.Error("adding an environment variable MUST change the config file size")
	}

	if configDigestsAreEqual(t, source, result) {
		t.Errorf("mutating the config MUST mutate the config digest")
	}

	assertAccurateManifestConfigDigest(t, result)

	if !reflect.DeepEqual(cfg.Config.Env, newEnv) {
		t.Errorf("incorrect environment set %v!=%v", cfg.Config.Env, newEnv)
	}
}

func TestMutateCreatedAt(t *testing.T) {
	source := sourceImage(t)
	want := time.Now().Add(-2 * time.Minute)
	result, err := CreatedAt(source, v1.Time{want})
	if err != nil {
		t.Fatalf("CreatedAt: %v", err)
	}

	if configDigestsAreEqual(t, source, result) {
		t.Errorf("mutating the created time MUST mutate the config digest")
	}

	got := getConfigFile(t, result).Created.Time
	if got != want {
		t.Errorf("mutating the created time MUST mutate the time from %v to %v", got, want)
	}
}

func TestMutateTime(t *testing.T) {
	source := sourceImage(t)
	want := time.Time{}
	result, err := Time(source, want)
	if err != nil {
		t.Fatalf("failed to mutate a config: %v", err)
	}

	if configDigestsAreEqual(t, source, result) {
		t.Fatal("mutating the created time MUST mutate the config digest")
	}

	got := getConfigFile(t, result).Created.Time
	if got != want {
		t.Fatalf("mutating the created time MUST mutate the time from %v to %v", got, want)
	}
}

func TestLayerTime(t *testing.T) {
	source := sourceImage(t)
	layers := getLayers(t, source)
	expectedTime := time.Date(1970, time.January, 1, 0, 0, 1, 0, time.UTC)
	fmt.Printf("expectedTime %v", expectedTime)

	for _, layer := range layers {
		result, err := layerTime(layer, expectedTime)
		if err != nil {
			t.Fatalf("setting layer time failed: %v", err)
		}

		assertMTime(t, result, expectedTime)
	}
}

func TestAppendStreamableLayer(t *testing.T) {
	img, err := AppendLayers(
		sourceImage(t),
		stream.NewLayer(ioutil.NopCloser(strings.NewReader(strings.Repeat("a", 100)))),
		stream.NewLayer(ioutil.NopCloser(strings.NewReader(strings.Repeat("b", 100)))),
		stream.NewLayer(ioutil.NopCloser(strings.NewReader(strings.Repeat("c", 100)))),
	)
	if err != nil {
		t.Fatalf("AppendLayers: %v", err)
	}

	// Until the streams are consumed, the image manifest is not yet computed.
	if _, err := img.Manifest(); err != stream.ErrNotComputed {
		t.Errorf("Manifest: got %v, want %v", err, stream.ErrNotComputed)
	}

	// We can still get Layers while some are not yet computed.
	ls, err := img.Layers()
	if err != nil {
		t.Errorf("Layers: %v", err)
	}
	wantDigests := []string{
		"sha256:bfa1c600931132f55789459e2f5a5eb85659ac91bc5a54ce09e3ed14809f8a7f",
		"sha256:77a52b9a141dcc4d3d277d053193765dca725626f50eaf56b903ac2439cf7fd1",
		"sha256:b78472d63f6e3d31059819173b56fcb0d9479a2b13c097d4addd84889f6aff06",
	}
	for i, l := range ls[1:] {
		rc, err := l.Compressed()
		if err != nil {
			t.Errorf("Layer %d Compressed: %v", i, err)
		}

		// Consume the layer's stream and close it to compute the
		// layer's metadata.
		if _, err := io.Copy(ioutil.Discard, rc); err != nil {
			t.Errorf("Reading layer %d: %v", i, err)
		}
		if err := rc.Close(); err != nil {
			t.Errorf("Closing layer %d: %v", i, err)
		}

		// The layer's metadata is now available.
		h, err := l.Digest()
		if err != nil {
			t.Errorf("Digest after consuming layer %d: %v", i, err)
		}
		if h.String() != wantDigests[i] {
			t.Errorf("Layer %d digest got %q, want %q", i, h, wantDigests[i])
		}
	}

	// Now that the streamable layers have been consumed, the image's
	// manifest can be computed.
	if _, err := img.Manifest(); err != nil {
		t.Errorf("Manifest: %v", err)
	}

	h, err := img.Digest()
	if err != nil {
		t.Errorf("Digest: %v", err)
	}
	wantDigest := "sha256:5fa010d1a9ab2cf0125a53087aed97aa327dc845123702cb2a52c844f19cc36b"
	if h.String() != wantDigest {
		t.Errorf("Image digest got %q, want %q", h, wantDigest)
	}

}

func assertMTime(t *testing.T, layer v1.Layer, expectedTime time.Time) {
	l, err := layer.Uncompressed()

	if err != nil {
		t.Fatalf("reading layer failed: %v", err)
	}

	tr := tar.NewReader(l)
	for {
		header, err := tr.Next()
		if err == io.EOF {
			break
		}
		if err != nil {
			t.Fatalf("Error reading layer: %v", err)
		}

		mtime := header.ModTime
		if mtime.Equal(expectedTime) == false {
			t.Errorf("unexpected mod time for layer. expected %v, got %v.", expectedTime, mtime)
		}
	}

}
func assertQueryingForLayerSucceeds(t *testing.T, image v1.Image, layer v1.Layer) {
	t.Helper()

	queryTestCases := []struct {
		name          string
		expectedLayer v1.Layer
		hash          func() (v1.Hash, error)
		query         func(v1.Hash) (v1.Layer, error)
	}{
		{"digest", layer, layer.Digest, image.LayerByDigest},
		{"diff id", layer, layer.DiffID, image.LayerByDiffID},
	}

	for _, tc := range queryTestCases {
		t.Run(fmt.Sprintf("layer by %s", tc.name), func(t *testing.T) {
			hash, err := tc.hash()
			if err != nil {
				t.Fatalf("Unable to fetch %s for layer: %v", tc.name, err)
			}

			gotLayer, err := tc.query(hash)
			if err != nil {
				t.Fatalf("Unable to fetch layer from %s: %v", tc.name, err)
			}

			if gotLayer != tc.expectedLayer {
				t.Fatalf("Querying layer using %s does not return the expected layer %+v %+v", tc.name, gotLayer, tc.expectedLayer)
			}
		})
	}

}

func sourceImage(t *testing.T) v1.Image {
	t.Helper()

	image, err := tarball.ImageFromPath("testdata/source_image.tar", nil)
	if err != nil {
		t.Fatalf("Error loading image: %v", err)
	}
	return image
}

func getManifest(t *testing.T, i v1.Image) *v1.Manifest {
	t.Helper()

	m, err := i.Manifest()
	if err != nil {
		t.Fatalf("Error fetching image manifest: %v", err)
	}

	return m
}

func getLayers(t *testing.T, i v1.Image) []v1.Layer {
	t.Helper()

	l, err := i.Layers()
	if err != nil {
		t.Fatalf("Error fetching image layers: %v", err)
	}

	return l
}

func getConfigFile(t *testing.T, i v1.Image) *v1.ConfigFile {
	t.Helper()

	c, err := i.ConfigFile()
	if err != nil {
		t.Fatalf("Error fetching image config file: %v", err)
	}

	return c
}

func getRawConfigFile(t *testing.T, i v1.Image) []byte {
	t.Helper()

	rcfg, err := i.RawConfigFile()
	if err != nil {
		t.Fatalf("Unable to fetch layer raw config file: %v", err)
	}
	return rcfg
}

func computeDigest(t *testing.T, data []byte) v1.Hash {
	d, _, err := v1.SHA256(bytes.NewBuffer(data))
	if err != nil {
		t.Fatalf("Unable to compute config digest: %v", err)
	}
	return d
}

func configFilesAreEqual(t *testing.T, first, second v1.Image) bool {
	t.Helper()

	fc := getConfigFile(t, first)
	sc := getConfigFile(t, second)

	return cmp.Equal(fc, sc)
}

func configDigestsAreEqual(t *testing.T, first, second v1.Image) bool {
	t.Helper()

	fm := getManifest(t, first)
	sm := getManifest(t, second)

	return fm.Config.Digest == sm.Config.Digest
}

func configSizesAreEqual(t *testing.T, first, second v1.Image) bool {
	t.Helper()

	fm := getManifest(t, first)
	sm := getManifest(t, second)

	return fm.Config.Size == sm.Config.Size
}

func manifestsAreEqual(t *testing.T, first, second v1.Image) bool {
	t.Helper()

	fm := getManifest(t, first)
	sm := getManifest(t, second)

	return cmp.Equal(fm, sm)
}

func assertLayerOrderMatchesConfig(t *testing.T, i v1.Image) {
	t.Helper()

	layers := getLayers(t, i)
	cf := getConfigFile(t, i)

	if got, want := len(layers), len(cf.RootFS.DiffIDs); got != want {
		t.Fatalf("Difference in size between the image layers (%d) "+
			"and the config file diff ids (%d)", got, want)
	}

	for i := range layers {
		diffID, err := layers[i].DiffID()
		if err != nil {
			t.Fatalf("Unable to fetch layer diff id: %v", err)
		}

		if got, want := diffID, cf.RootFS.DiffIDs[i]; got != want {
			t.Fatalf("Layer diff id (%v) is not at the expected index (%d) in %+v",
				got, i, cf.RootFS.DiffIDs)
		}
	}
}

func assertLayerOrderMatchesManifest(t *testing.T, i v1.Image) {
	t.Helper()

	layers := getLayers(t, i)
	mf := getManifest(t, i)

	if got, want := len(layers), len(mf.Layers); got != want {
		t.Fatalf("Difference in size between the image layers (%d) "+
			"and the manifest layers (%d)", got, want)
	}

	for i := range layers {
		digest, err := layers[i].Digest()
		if err != nil {
			t.Fatalf("Unable to fetch layer diff id: %v", err)
		}

		if got, want := digest, mf.Layers[i].Digest; got != want {
			t.Fatalf("Layer digest (%v) is not at the expected index (%d) in %+v",
				got, i, mf.Layers)
		}
	}
}

func assertAccurateManifestConfigDigest(t *testing.T, i v1.Image) {
	t.Helper()

	m := getManifest(t, i)
	got := m.Config.Digest

	rcfg := getRawConfigFile(t, i)
	want := computeDigest(t, rcfg)

	if !cmp.Equal(got, want) {
		t.Fatalf("Manifest config digest (%v) does not match digest of config file (%v)", got, want)
	}
}

type mockLayer struct{}

func (m mockLayer) Digest() (v1.Hash, error) {
	return v1.Hash{Algorithm: "fake", Hex: "digest"}, nil
}

func (m mockLayer) DiffID() (v1.Hash, error) {
	return v1.Hash{Algorithm: "fake", Hex: "diff id"}, nil
}

func (m mockLayer) MediaType() (types.MediaType, error) {
	return "some-media-type", nil
}

func (m mockLayer) Size() (int64, error) { return 137438691328, nil }
func (m mockLayer) Compressed() (io.ReadCloser, error) {
	return ioutil.NopCloser(strings.NewReader("compressed times")), nil
}
func (m mockLayer) Uncompressed() (io.ReadCloser, error) {
	return ioutil.NopCloser(strings.NewReader("uncompressed")), nil
}
