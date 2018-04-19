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
	"github.com/docker/docker/builder/dockerfile/parser"
	"github.com/docker/docker/builder/dockerfile/shell"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strings"
)

// ResolveEnvironmentReplacementList resolves a list of values by calling resolveEnvironmentReplacement
func ResolveEnvironmentReplacementList(values, envs []string, isFilepath bool) ([]string, error) {
	var resolvedValues []string
	for _, value := range values {
		if IsSrcRemoteFileURL(value) {
			resolvedValues = append(resolvedValues, value)
			continue
		}
		resolved, err := ResolveEnvironmentReplacement(value, envs, isFilepath)
		logrus.Debugf("Resolved %s to %s", value, resolved)
		if err != nil {
			return nil, err
		}
		resolvedValues = append(resolvedValues, resolved)
	}
	return resolvedValues, nil
}

// ResolveEnvironmentReplacement resolves replacing env variables in some text from envs
// It takes in a string representation of the command, the value to be resolved, and a list of envs (config.Env)
// Ex: fp = $foo/newdir, envs = [foo=/foodir], then this should return /foodir/newdir
// The dockerfile/shell package handles processing env values
// It handles escape characters and supports expansion from the config.Env array
// Shlex handles some of the following use cases (these and more are tested in integration tests)
// ""a'b'c"" -> "a'b'c"
// "Rex\ The\ Dog \" -> "Rex The Dog"
// "a\"b" -> "a"b"
func ResolveEnvironmentReplacement(value string, envs []string, isFilepath bool) (string, error) {
	shlex := shell.NewLex(parser.DefaultEscapeToken)
	fp, err := shlex.ProcessWord(value, envs)
	if !isFilepath {
		return fp, err
	}
	if err != nil {
		return "", err
	}
	fp = filepath.Clean(fp)
	if IsDestDir(value) {
		fp = fp + "/"
	}
	return fp, nil
}

// ContainsWildcards returns true if any entry in paths contains wildcards
func ContainsWildcards(paths []string) bool {
	for _, path := range paths {
		if strings.ContainsAny(path, "*?[") {
			return true
		}
	}
	return false
}

// ResolveSources resolves the given sources if the sources contains wildcards
// It returns a map of [src]:[files rooted at src]
func ResolveSources(srcsAndDest instructions.SourcesAndDest, root string) ([]string, error) {
	srcs := srcsAndDest[:len(srcsAndDest)-1]
	// If sources contain wildcards, we first need to resolve them to actual paths
	if ContainsWildcards(srcs) {
		logrus.Debugf("Resolving srcs %v...", srcs)
		files, err := RelativeFiles("", root)
		if err != nil {
			return nil, err
		}
		srcs, err = matchSources(srcs, files)
		if err != nil {
			return nil, err
		}
		logrus.Debugf("Resolved sources to %v", srcs)
	}
	// Check to make sure the sources are valid
	return srcs, IsSrcsValid(srcsAndDest, srcs, root)
}

// matchSources returns a list of sources that match wildcards
func matchSources(srcs, files []string) ([]string, error) {
	var matchedSources []string
	for _, src := range srcs {
		if IsSrcRemoteFileURL(src) {
			matchedSources = append(matchedSources, src)
			continue
		}
		src = filepath.Clean(src)
		for _, file := range files {
			matched, err := filepath.Match(src, file)
			if err != nil {
				return nil, err
			}
			if matched || src == file {
				matchedSources = append(matchedSources, file)
			}
		}
	}
	return matchedSources, nil
}

func IsDestDir(path string) bool {
	return strings.HasSuffix(path, "/") || path == "."
}

// DestinationFilepath returns the destination filepath from the build context to the image filesystem
// If source is a file:
//	If dest is a dir, copy it to /dest/relpath
// 	If dest is a file, copy directly to dest
// If source is a dir:
//	Assume dest is also a dir, and copy to dest/relpath
// If dest is not an absolute filepath, add /cwd to the beginning
func DestinationFilepath(filename, srcName, dest, cwd, buildcontext string) (string, error) {
	fi, err := os.Lstat(filepath.Join(buildcontext, filename))
	if err != nil {
		return "", err
	}
	src, err := os.Lstat(filepath.Join(buildcontext, srcName))
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
		destPath := filepath.Join(dest, relPath)
		if filepath.IsAbs(dest) {
			return destPath, nil
		}
		return filepath.Join(cwd, destPath), nil
	}
	if filepath.IsAbs(dest) {
		return dest, nil
	}
	return filepath.Join(cwd, dest), nil
}

// URLDestinationFilepath gives the destination a file from a remote URL should be saved to
func URLDestinationFilepath(rawurl, dest, cwd string) string {
	if !IsDestDir(dest) {
		if !filepath.IsAbs(dest) {
			return filepath.Join(cwd, dest)
		}
		return dest
	}
	urlBase := filepath.Base(rawurl)
	destPath := filepath.Join(dest, urlBase)

	if !filepath.IsAbs(dest) {
		destPath = filepath.Join(cwd, destPath)
	}
	return destPath
}

// SourcesToFilesMap returns a map of [src]:[files rooted at source]
func SourcesToFilesMap(srcs []string, root string) (map[string][]string, error) {
	srcMap := make(map[string][]string)
	for _, src := range srcs {
		if IsSrcRemoteFileURL(src) {
			srcMap[src] = []string{src}
			continue
		}
		src = filepath.Clean(src)
		files, err := RelativeFiles(src, root)
		if err != nil {
			return nil, err
		}
		srcMap[src] = files
	}
	return srcMap, nil
}

// IsSrcsValid returns an error if the sources provided are invalid, or nil otherwise
func IsSrcsValid(srcsAndDest instructions.SourcesAndDest, resolvedSrcs []string, root string) error {
	srcs := srcsAndDest[:len(srcsAndDest)-1]
	dest := srcsAndDest[len(srcsAndDest)-1]

	// Now, get a map of [src]:[files rooted at src]
	srcMap, err := SourcesToFilesMap(resolvedSrcs, root)
	if err != nil {
		return err
	}

	totalFiles := 0
	for _, files := range srcMap {
		totalFiles += len(files)
	}
	if totalFiles == 0 {
		return errors.New("copy failed: no source files specified")
	}

	if !ContainsWildcards(srcs) {
		// If multiple sources and destination isn't a directory, return an error
		if len(srcs) > 1 && !IsDestDir(dest) {
			return errors.New("when specifying multiple sources in a COPY command, destination must be a directory and end in '/'")
		}
		return nil
	}

	// If there are wildcards, and the destination is a file, there must be exactly one file to copy over,
	// Otherwise, return an error
	if !IsDestDir(dest) && totalFiles > 1 {
		return errors.New("when specifying multiple sources in a COPY command, destination must be a directory and end in '/'")
	}
	return nil
}

func IsSrcRemoteFileURL(rawurl string) bool {
	_, err := url.ParseRequestURI(rawurl)
	if err != nil {
		return false
	}
	_, err = http.Get(rawurl)
	if err != nil {
		return false
	}
	return true
}
