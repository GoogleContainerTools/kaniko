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

package snapshot

import (
	"bytes"
	"encoding/json"
	"fmt"

	"github.com/GoogleContainerTools/kaniko/pkg/timing"
	"github.com/GoogleContainerTools/kaniko/pkg/util"
)

type LayeredMap struct {
	adds    []map[string]string   // All layers with added files.
	deletes []map[string]struct{} // All layers with deleted files.

	currentImage        map[string]string // All files and hashes in the current image (up to the last layer).
	isCurrentImageValid bool              // If the currentImage is not out-of-date.

	layerHashCache map[string]string
	hasher         func(string) (string, error)
}

// NewLayeredMap creates a new layered map which keeps track of adds and deletes.
func NewLayeredMap(h func(string) (string, error)) *LayeredMap {
	l := LayeredMap{
		hasher: h,
	}

	l.currentImage = map[string]string{}
	l.layerHashCache = map[string]string{}
	return &l
}

// Snapshot creates a new layer.
func (l *LayeredMap) Snapshot() {

	// Save current state of image
	l.updateCurrentImage()

	l.adds = append(l.adds, map[string]string{})
	l.deletes = append(l.deletes, map[string]struct{}{})
	l.layerHashCache = map[string]string{} // Erase the hash cache for this new layer.
}

// Key returns a hash for added and delted files.
func (l *LayeredMap) Key() (string, error) {

	var adds map[string]string
	var deletes map[string]struct{}

	if len(l.adds) != 0 {
		adds = l.adds[len(l.adds)-1]
		deletes = l.deletes[len(l.deletes)-1]
	}

	c := bytes.NewBuffer([]byte{})
	enc := json.NewEncoder(c)
	err := enc.Encode(adds)
	if err != nil {
		return "", err
	}
	err = enc.Encode(deletes)
	if err != nil {
		return "", err
	}
	return util.SHA256(c)
}

// getCurrentImage returns the current image by merging the latest
// adds and deletes on to the current image (if its not yet valid.)
func (l *LayeredMap) getCurrentImage() map[string]string {
	if l.isCurrentImageValid || len(l.adds) == 0 {
		// No layers yet or current image is valid.
		return l.currentImage
	}

	current := map[string]string{}

	// Copy current image paths/hashes.
	for p, h := range l.currentImage {
		current[p] = h
	}

	// Add the last layer on top.
	addedFiles := l.adds[len(l.adds)-1]
	deletedFiles := l.deletes[len(l.deletes)-1]

	for add, hash := range addedFiles {
		current[add] = hash
	}

	for del := range deletedFiles {
		delete(current, del)
	}

	return current
}

// updateCurrentImage update the internal current image by merging the
// top adds and deletes onto the current image.
func (l *LayeredMap) updateCurrentImage() {
	if l.isCurrentImageValid {
		return
	}

	l.currentImage = l.getCurrentImage()
	l.isCurrentImageValid = true
}

// get returns the current hash in the current image `l.currentImage`.
func (l *LayeredMap) get(s string) (string, bool) {
	h, ok := l.currentImage[s]
	return h, ok
}

// GetCurrentPaths returns all existing paths in the actual current image
// cached by FlattenLayers.
func (l *LayeredMap) GetCurrentPaths() map[string]struct{} {
	current := l.getCurrentImage()

	paths := map[string]struct{}{}
	for f := range current {
		paths[f] = struct{}{}
	}
	return paths
}

// AddDelete will delete the specific files in the current layer.
func (l *LayeredMap) AddDelete(s string) error {
	l.isCurrentImageValid = false

	l.deletes[len(l.deletes)-1][s] = struct{}{}
	return nil
}

// Add will add the specified file s to the current layer.
func (l *LayeredMap) Add(s string) error {
	l.isCurrentImageValid = false

	// Use hash function and add to layers
	newV, err := func(s string) (string, error) {
		if v, ok := l.layerHashCache[s]; ok {
			return v, nil
		}
		return l.hasher(s)
	}(s)

	if err != nil {
		return fmt.Errorf("Error creating hash for %s: %w", s, err)
	}

	l.adds[len(l.adds)-1][s] = newV
	return nil
}

// CheckFileChange checks whether a given file (needs to exist) changed
// from the current layered map by its hashing function.
// If the file does not exist, an error is returned.
// Returns true if the file is changed.
func (l *LayeredMap) CheckFileChange(s string) (bool, error) {
	t := timing.Start("Hashing files")
	defer timing.DefaultRun.Stop(t)

	newV, err := l.hasher(s)
	if err != nil {
		return false, err
	}

	// Save hash to not recompute it when
	// adding the file.
	l.layerHashCache[s] = newV

	oldV, ok := l.get(s)
	if ok && newV == oldV {
		// File hash did not change => Unchanged.
		return false, nil
	}

	// File does not exist in current image,
	// or it did change => Changed.
	return true, nil
}
