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
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/GoogleContainerTools/kaniko/pkg/util"
	"github.com/pkg/errors"
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

// Key returns the human readable composite key as a string.
func (s *CompositeCache) Key() string {
	return strings.Join(s.keys, "-")
}

// Hash returns the composite key in a string SHA256 format.
func (s *CompositeCache) Hash() (string, error) {
	return util.SHA256(strings.NewReader(s.Key()))
}

func (s *CompositeCache) AddPath(p string, context util.FileContext) error {
	sha := sha256.New()
	fi, err := os.Lstat(p)
	if err != nil {
		return errors.Wrap(err, "could not add path")
	}

	if fi.Mode().IsDir() {
		empty, k, err := hashDir(p, context)
		if err != nil {
			return err
		}

		// Only add the hash of this directory to the key
		// if there is any ignored content.
		if !empty || !context.ExcludesFile(p) {
			s.keys = append(s.keys, k)
		}
		return nil
	}

	if context.ExcludesFile(p) {
		return nil
	}
	fh, err := util.CacheHasher()(p)
	if err != nil {
		return err
	}
	if _, err := sha.Write([]byte(fh)); err != nil {
		return err
	}

	s.keys = append(s.keys, fmt.Sprintf("%x", sha.Sum(nil)))
	return nil
}

// HashDir returns a hash of the directory.
func hashDir(p string, context util.FileContext) (bool, string, error) {
	sha := sha256.New()
	empty := true
	if err := filepath.Walk(p, func(path string, fi os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		exclude := context.ExcludesFile(path)
		if exclude {
			return nil
		}

		fileHash, err := util.CacheHasher()(path)
		if err != nil {
			return err
		}
		if _, err := sha.Write([]byte(fileHash)); err != nil {
			return err
		}
		empty = false
		return nil
	}); err != nil {
		return false, "", err
	}

	return empty, fmt.Sprintf("%x", sha.Sum(nil)), nil
}
