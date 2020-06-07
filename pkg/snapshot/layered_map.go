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
	"os"
	"path/filepath"
	"strings"

	"github.com/GoogleContainerTools/kaniko/pkg/timing"
	"github.com/GoogleContainerTools/kaniko/pkg/util"
	"github.com/sirupsen/logrus"
)

type LayeredMap struct {
	layers         []map[string]string
	whiteouts      []map[string]struct{}
	layerHashCache map[string]string
	hasher         func(string) (string, error)
	// cacheHasher doesn't include mtime in it's hash so that filesystem cache keys are stable
	cacheHasher func(string) (string, error)
}

func NewLayeredMap(h func(string) (string, error), c func(string) (string, error)) *LayeredMap {
	l := LayeredMap{
		hasher:      h,
		cacheHasher: c,
	}
	l.layers = []map[string]string{}
	l.layerHashCache = map[string]string{}
	return &l
}

func (l *LayeredMap) Snapshot() {
	l.whiteouts = append(l.whiteouts, map[string]struct{}{})
	l.layers = append(l.layers, map[string]string{})
}

// Key returns a hash for added files
func (l *LayeredMap) Key() (string, error) {
	c := bytes.NewBuffer([]byte{})
	enc := json.NewEncoder(c)
	enc.Encode(l.layers)
	return util.SHA256(c)
}

// GetFlattenedPathsForWhiteOut returns all paths in the current FS
func (l *LayeredMap) getFlattenedPathsForWhiteOut() map[string]struct{} {
	paths := map[string]struct{}{}
	for _, l := range l.layers {
		for p := range l {
			if strings.HasPrefix(filepath.Base(p), ".wh.") {
				delete(paths, p)
			}
			paths[p] = struct{}{}
		}
	}
	return paths
}

func (l *LayeredMap) Get(s string) (string, bool) {
	for i := len(l.layers) - 1; i >= 0; i-- {
		if v, ok := l.layers[i][s]; ok {
			return v, ok
		}
	}
	return "", false
}

func (l *LayeredMap) GetWhiteout(s string) bool {
	for i := len(l.whiteouts) - 1; i >= 0; i-- {
		if _, ok := l.whiteouts[i][s]; ok {
			return ok
		}
	}
	return false
}

func (l *LayeredMap) MaybeAddWhiteout(s string) bool {
	ok := l.GetWhiteout(s)
	if ok {
		return false
	}
	l.whiteouts[len(l.whiteouts)-1][s] = struct{}{}
	return true
}

// Add will add the specified file s to the layered map.
func (l *LayeredMap) Add(s string) error {
	// Use hash function and add to layers
	newV, err := func(s string) (string, error) {
		if v, ok := l.layerHashCache[s]; ok {
			// clear it cache for next layer.
			delete(l.layerHashCache, s)
			return v, nil
		}
		return l.hasher(s)
	}(s)
	if err != nil {
		return fmt.Errorf("error creating hash for %s: %v", s, err)
	}
	l.layers[len(l.layers)-1][s] = newV
	return nil
}

// CheckFileChange checks whether a given file changed
// from the current layered map by its hashing function.
// Returns true if the file is changed.
func (l *LayeredMap) CheckFileChange(s string) (bool, error) {
	t := timing.Start("Hashing files")
	defer timing.DefaultRun.Stop(t)
	newV, err := l.hasher(s)
	if err != nil {
		// if this file does not exist in the new layer return.
		if os.IsNotExist(err) {
			logrus.Tracef("%s detected as changed but does not exist", s)
			return false, nil
		}
		return false, err
	}
	l.layerHashCache[s] = newV
	oldV, ok := l.Get(s)
	if ok && newV == oldV {
		return false, nil
	}
	return true, nil
}
