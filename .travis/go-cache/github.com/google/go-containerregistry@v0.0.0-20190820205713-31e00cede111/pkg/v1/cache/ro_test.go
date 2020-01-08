package cache

import (
	"testing"

	v1 "github.com/google/go-containerregistry/pkg/v1"
	"github.com/google/go-containerregistry/pkg/v1/random"
)

func TestReadOnly(t *testing.T) {
	m := &memcache{map[v1.Hash]v1.Layer{}}
	ro := ReadOnly(m)

	// Populate the cache.
	img, err := random.Image(10, 1)
	if err != nil {
		t.Fatalf("random.Image: %v", err)
	}
	img = Image(img, m)
	ls, err := img.Layers()
	if err != nil {
		t.Fatalf("Layers: %v", err)
	}
	if got, want := len(ls), 1; got != want {
		t.Fatalf("Layers returned %d layers, want %d", got, want)
	}
	h, err := ls[0].Digest()
	if err != nil {
		t.Fatalf("layer.Digest: %v", err)
	}
	m.m[h] = ls[0]

	// Layer can be read from original cache and RO cache.
	if _, err := m.Get(h); err != nil {
		t.Fatalf("m.Get: %v", err)
	}
	if _, err := ro.Get(h); err != nil {
		t.Fatalf("ro.Get: %v", err)
	}
	ln := len(m.m)

	// RO Put is a no-op.
	if _, err := ro.Put(ls[0]); err != nil {
		t.Fatalf("ro.Put: %v", err)
	}
	if got, want := len(m.m), ln; got != want {
		t.Errorf("After Put, got %v entries, want %v", got, want)
	}

	// RO Delete is a no-op.
	if err := ro.Delete(h); err != nil {
		t.Fatalf("ro.Delete: %v", err)
	}
	if got, want := len(m.m), ln; got != want {
		t.Errorf("After Delete, got %v entries, want %v", got, want)
	}

	// Deleting from the underlying RW cache updates RO view.
	if err := m.Delete(h); err != nil {
		t.Fatalf("m.Delete: %v", err)
	}
	if got, want := len(m.m), 0; got != want {
		t.Errorf("After RW Delete, got %v entries, want %v", got, want)
	}
}
