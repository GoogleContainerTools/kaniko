package cache

import (
	"io"
	"io/ioutil"
	"os"
	"testing"

	v1 "github.com/google/go-containerregistry/pkg/v1"
	"github.com/google/go-containerregistry/pkg/v1/random"
)

func TestFilesystemCache(t *testing.T) {
	dir, err := ioutil.TempDir("", "ggcr-cache")
	if err != nil {
		t.Fatalf("TempDir: %v", err)
	}
	defer os.RemoveAll(dir)

	numLayers := 5
	img, err := random.Image(10, int64(numLayers))
	if err != nil {
		t.Fatalf("random.Image: %v", err)
	}
	c := NewFilesystemCache(dir)
	img = Image(img, c)

	// Read all the (compressed) layers to populate the cache.
	ls, err := img.Layers()
	if err != nil {
		t.Fatalf("Layers: %v", err)
	}
	for i, l := range ls {
		rc, err := l.Compressed()
		if err != nil {
			t.Fatalf("layer[%d].Compressed: %v", i, err)
		}
		if _, err := io.Copy(ioutil.Discard, rc); err != nil {
			t.Fatalf("Error reading contents: %v", err)
		}
		rc.Close()
	}

	// Check that layers exist in the fs cache.
	files, err := ioutil.ReadDir(dir)
	if err != nil {
		t.Fatalf("ReadDir: %v", err)
	}
	if got, want := len(files), numLayers; got != want {
		t.Errorf("Got %d cached files, want %d", got, want)
	}
	for _, fi := range files {
		if fi.Size() == 0 {
			t.Errorf("Cached file %q is empty", fi.Name())
		}
	}

	// Read all (uncompressed) layers, those populate the cache too.
	for i, l := range ls {
		rc, err := l.Uncompressed()
		if err != nil {
			t.Fatalf("layer[%d].Compressed: %v", i, err)
		}
		if _, err := io.Copy(ioutil.Discard, rc); err != nil {
			t.Fatalf("Error reading contents: %v", err)
		}
		rc.Close()
	}

	// Check that double the layers are present now, both compressed and
	// uncompressed.
	files, err = ioutil.ReadDir(dir)
	if err != nil {
		t.Fatalf("ReadDir: %v", err)
	}
	if got, want := len(files), numLayers*2; got != want {
		t.Errorf("Got %d cached files, want %d", got, want)
	}
	for _, fi := range files {
		if fi.Size() == 0 {
			t.Errorf("Cached file %q is empty", fi.Name())
		}
	}

	// Delete a cached layer, see it disappear.
	l := ls[0]
	h, err := l.Digest()
	if err != nil {
		t.Fatalf("layer.Digest: %v", err)
	}
	if err := c.Delete(h); err != nil {
		t.Errorf("cache.Delete: %v", err)
	}
	files, err = ioutil.ReadDir(dir)
	if err != nil {
		t.Fatalf("ReadDir: %v", err)
	}
	if got, want := len(files), numLayers*2-1; got != want {
		t.Errorf("Got %d cached files, want %d", got, want)
	}

	// Read the image again, see the layer reappear.
	for i, l := range ls {
		rc, err := l.Compressed()
		if err != nil {
			t.Fatalf("layer[%d].Compressed: %v", i, err)
		}
		if _, err := io.Copy(ioutil.Discard, rc); err != nil {
			t.Fatalf("Error reading contents: %v", err)
		}
		rc.Close()
	}

	// Check that layers exist in the fs cache.
	files, err = ioutil.ReadDir(dir)
	if err != nil {
		t.Fatalf("ReadDir: %v", err)
	}
	if got, want := len(files), numLayers*2; got != want {
		t.Errorf("Got %d cached files, want %d", got, want)
	}
	for _, fi := range files {
		if fi.Size() == 0 {
			t.Errorf("Cached file %q is empty", fi.Name())
		}
	}
}

func TestErrNotFound(t *testing.T) {
	dir, err := ioutil.TempDir("", "ggcr-cache")
	if err != nil {
		t.Fatalf("TempDir: %v", err)
	}
	os.RemoveAll(dir) // Remove the tempdir.

	c := NewFilesystemCache(dir)
	h := v1.Hash{Algorithm: "fake", Hex: "not-found"}
	if _, err := c.Get(h); err != ErrNotFound {
		t.Errorf("Get(%q): %v", h, err)
	}
	if err := c.Delete(h); err != ErrNotFound {
		t.Errorf("Delete(%q): %v", h, err)
	}
}
