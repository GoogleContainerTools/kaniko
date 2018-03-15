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

package util

import (
	"github.com/docker/docker/builder/dockerfile/instructions"
	"github.com/pkg/errors"
	"os"
	"path/filepath"
	"strings"
)

// ContainsWildcards returns true if any entry in paths contains wildcards
func ContainsWildcards(paths []string) bool {
	for _, path := range paths {
		for i := 0; i < len(path); i++ {
			ch := path[i]
			// These are the wildcards that correspond to filepath.Match
			if ch == '*' || ch == '?' || ch == '[' {
				return true
			}
		}
	}
	return false
}

// ResolveSources resolves the given sources if the sources contains wildcard
// It returns a map of [src]:[files rooted at src]
func ResolveSources(srcsAndDest instructions.SourcesAndDest, root, cwd string) (map[string][]string, error) {
	srcs := srcsAndDest[:len(srcsAndDest)-1]
	// If sources contain wildcards, we first need to resolve them to actual paths
	wildcard := ContainsWildcards(srcs)
	if wildcard {
		files, err := Files("", root)
		if err != nil {
			return nil, err
		}
		srcs, err = matchSources(srcs, files, cwd)
		if err != nil {
			return nil, err
		}
	}
	// Now, get a map of [src]:[files rooted at src]
	srcMap, err := SourcesToFilesMap(srcs, root)
	if err != nil {
		return nil, err
	}
	// Check to make sure the sources are valid
	return srcMap, IsSrcsValid(srcsAndDest, srcMap)
}

// matchSources returns a map of [src]:[matching filepaths], used to resolve wildcards
func matchSources(srcs, files []string, cwd string) ([]string, error) {
	var matchedSources []string
	for _, src := range srcs {
		src = filepath.Clean(src)
		for _, file := range files {
			matched, err := filepath.Match(src, file)
			if err != nil {
				return nil, err
			}
			// Check cwd
			matchedRoot, err := filepath.Match(filepath.Join(cwd, src), file)
			if err != nil {
				return nil, err
			}
			if !(matched || matchedRoot) {
				continue
			}
			matchedSources = append(matchedSources, file)
		}
	}
	return matchedSources, nil
}

func IsDestDir(path string) bool {
	return strings.HasSuffix(path, "/")
}

// RelativeFilepath returns the relative filepath
// If source is a file:
//	If dest is a dir, copy it to /cwd/dest/relpath
// 	If dest is a file, copy directly to /cwd/dest

// If source is a dir:
//	Assume dest is also a dir, and copy to /cwd/dest/relpath
func RelativeFilepath(filename, srcName, dest, cwd, buildcontext string) (string, error) {
	fi, err := os.Stat(filepath.Join(buildcontext, filename))
	if err != nil {
		return "", err
	}
	src, err := os.Stat(filepath.Join(buildcontext, srcName))
	if err != nil {
		return "", err
	}
	if src.IsDir() || IsDestDir(dest) {
		relPath, err := filepath.Rel(srcName, filename)
		if err != nil {
			return "", err
		}
		if relPath == "." && !fi.IsDir() {
			relPath = filepath.Base(filename)
		}
		destPath := filepath.Join(cwd, dest, relPath)
		return destPath, nil
	}
	return filepath.Join(cwd, dest), nil
}

// SourcesToFilesMap returns a map of [src]:[files rooted at source]
func SourcesToFilesMap(srcs []string, root string) (map[string][]string, error) {
	srcMap := make(map[string][]string)
	for _, src := range srcs {
		src = filepath.Clean(src)
		files, err := Files(src, root)
		if err != nil {
			return nil, err
		}
		srcMap[src] = files
	}
	return srcMap, nil
}

// IsSrcsValid returns an error if the sources provided are invalid, or nil otherwise
func IsSrcsValid(srcsAndDest instructions.SourcesAndDest, srcMap map[string][]string) error {
	srcs := srcsAndDest[:len(srcsAndDest)-1]
	dest := srcsAndDest[len(srcsAndDest)-1]
	// If destination is a directory, return nil
	if IsDestDir(dest) {
		return nil
	}
	// If no wildcards and multiple sources, return error
	if !ContainsWildcards(srcs) {
		if len(srcs) > 1 {
			return errors.New("when specifying multiple sources in a COPY command, destination must be a directory and end in '/'")
		}
		return nil
	}
	// If there are wildcards, and the destination is a file, there must be exactly one file to copy over
	totalFiles := 0
	for _, files := range srcMap {
		totalFiles += len(files)
	}
	if totalFiles == 0 {
		return errors.New("copy failed: no source files specified")
	}
	if totalFiles > 1 {
		return errors.New("when specifying multiple sources in a COPY command, destination must be a directory and end in '/'")
	}
	return nil
}
