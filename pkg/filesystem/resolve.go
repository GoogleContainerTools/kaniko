/*
Copyright 2020 Google LLC

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

package filesystem

import (
	"os"
	"path/filepath"

	"github.com/GoogleContainerTools/kaniko/pkg/config"
	"github.com/GoogleContainerTools/kaniko/pkg/util"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
)

// ResolvePaths takes a slice of file paths and a list of skipped file paths. It resolve each
// file path according to a set of rules and then returns a slice of resolved paths or error.
// File paths are resolved according to the following rules:
// * If path is in ignorelist, skip it.
// * If path is a symlink, resolve it's ancestor link and add it to the output set.
// * If path is a symlink, resolve it's target. If the target is not ignored add it to the
// output set.
// * Add all ancestors of each path to the output set.
func ResolvePaths(paths []string, wl []util.IgnoreListEntry) (pathsToAdd []string, err error) {
	logrus.Tracef("Resolving paths %s", paths)

	fileSet := make(map[string]bool)

	for _, f := range paths {
		// If the given path is part of the ignorelist ignore it
		if util.IsInProvidedIgnoreList(f, wl) {
			logrus.Debugf("Path %s is in list to ignore, ignoring it", f)
			continue
		}

		link, e := resolveSymlinkAncestor(f)
		if e != nil {
			continue
		}

		if f != link {
			logrus.Tracef("Updated link %s to %s", f, link)
		}

		if !fileSet[link] {
			pathsToAdd = append(pathsToAdd, link)
		}
		fileSet[link] = true

		var evaled string

		// If the path is a symlink we need to also consider the target of that
		// link
		evaled, e = filepath.EvalSymlinks(f)
		if e != nil {
			if !os.IsNotExist(e) {
				logrus.Errorf("Couldn't eval %s with link %s", f, link)
				return
			}

			logrus.Tracef("Symlink path %s, target does not exist", f)
			continue
		}
		if f != evaled {
			logrus.Tracef("Resolved symlink %s to %s", f, evaled)
		}

		// If the given path is a symlink and the target is part of the ignorelist
		// ignore the target
		if util.CheckCleanedPathAgainstProvidedIgnoreList(evaled, wl) {
			logrus.Debugf("Path %s is ignored, ignoring it", evaled)
			continue
		}

		if !fileSet[evaled] {
			pathsToAdd = append(pathsToAdd, evaled)
		}
		fileSet[evaled] = true
	}

	// Also add parent directories to keep the permission of them correctly.
	pathsToAdd = filesWithParentDirs(pathsToAdd)
	return
}

// filesWithParentDirs returns every ancestor path for each provided file path.
// I.E. /foo/bar/baz/boom.txt => [/, /foo, /foo/bar, /foo/bar/baz, /foo/bar/baz/boom.txt]
func filesWithParentDirs(files []string) []string {
	filesSet := map[string]bool{}

	for _, file := range files {
		file = filepath.Clean(file)
		filesSet[file] = true

		for _, dir := range util.ParentDirectories(file) {
			dir = filepath.Clean(dir)
			filesSet[dir] = true
		}
	}

	newFiles := []string{}
	for file := range filesSet {
		newFiles = append(newFiles, file)
	}

	return newFiles
}

// resolveSymlinkAncestor returns the ancestor link of the provided symlink path or returns the
// path if it is not a link. The ancestor link is the filenode whose type is a Symlink.
// E.G /baz/boom/bar.txt links to /usr/bin/bar.txt but /baz/boom/bar.txt itself is not a link.
// Instead /bar/boom is actually a link to /usr/bin. In this case resolveSymlinkAncestor would
// return /bar/boom.
func resolveSymlinkAncestor(path string) (string, error) {
	if !filepath.IsAbs(path) {
		return "", errors.New("dest path must be abs")
	}

	last := ""
	newPath := filepath.Clean(path)

loop:
	for newPath != config.RootDir {
		fi, err := os.Lstat(newPath)
		if err != nil {
			return "", errors.Wrap(err, "resolvePaths: failed to lstat")
		}

		if util.IsSymlink(fi) {
			last = filepath.Base(newPath)
			newPath = filepath.Dir(newPath)
		} else {
			// Even if the filenode pointed to by newPath is a regular file,
			// one of its ancestors could be a symlink. We call filepath.EvalSymlinks
			// to test whether there are any links in the path. If the output of
			// EvalSymlinks is different than the input we know one of the nodes in the
			// path is a link.
			target, err := filepath.EvalSymlinks(newPath)
			if err != nil {
				return "", err
			}
			if target != newPath {
				last = filepath.Base(newPath)
				newPath = filepath.Dir(newPath)
			} else {
				break loop
			}
		}
	}
	newPath = filepath.Join(newPath, last)
	return filepath.Clean(newPath), nil
}
