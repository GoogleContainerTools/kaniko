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

package executor

import (
	"crypto/sha256"
	"os"
	"path/filepath"
	"strings"

	"github.com/GoogleContainerTools/kaniko/pkg/util"
)

// NewCompositeCache returns an initialized composite cache object.
func NewCompositeCache(initial ...string) *CompositeCache {
	c := CompositeCache{
		keys: initial,
	}
	return &c
}

// CompositeCache is a type that generates a cache key from a series of keys.
type CompositeCache struct {
	keys []string
}

// AddKey adds the specified key to the sequence.
func (s *CompositeCache) AddKey(k ...string) {
	s.keys = append(s.keys, k...)
}

// AddDir adds the contents of a directory to the composite key.
func (s *CompositeCache) AddDir(p string) error {
	sha := sha256.New()
	if err := filepath.Walk(p, func(path string, fi os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		fileHash, err := util.CacheHasher()(path)
		if err != nil {
			return err
		}
		if _, err := sha.Write([]byte(fileHash)); err != nil {
			return err
		}
		return nil
	}); err != nil {
		return err
	}

	s.AddKey(string(sha.Sum(nil)))
	return nil
}

// Key returns the human readable composite key as a string.
func (s *CompositeCache) Key() string {
	return strings.Join(s.keys, "-")
}

// Hash returns the composite key in a string SHA256 format.
func (s *CompositeCache) Hash() (string, error) {
	return util.SHA256(strings.NewReader(s.Key()))
}
