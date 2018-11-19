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
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"syscall"

	"github.com/GoogleContainerTools/kaniko/pkg/constants"

	"github.com/GoogleContainerTools/kaniko/pkg/util"
	"github.com/sirupsen/logrus"
)

// For testing
var snapshotPathPrefix = constants.KanikoDir

// Snapshotter holds the root directory from which to take snapshots, and a list of snapshots taken
type Snapshotter struct {
	l         *LayeredMap
	directory string
}

// NewSnapshotter creates a new snapshotter rooted at d
func NewSnapshotter(l *LayeredMap, d string) *Snapshotter {
	return &Snapshotter{l: l, directory: d}
}

// Init initializes a new snapshotter
func (s *Snapshotter) Init() error {
	_, err := s.TakeSnapshotFS()
	return err
}

// Key returns a string based on the current state of the file system
func (s *Snapshotter) Key() (string, error) {
	return s.l.Key()
}

// TakeSnapshot takes a snapshot of the specified files, avoiding directories in the whitelist, and creates
// a tarball of the changed files. Return contents of the tarball, and whether or not any files were changed
func (s *Snapshotter) TakeSnapshot(files []string) (string, error) {
	f, err := ioutil.TempFile(snapshotPathPrefix, "")
	if err != nil {
		return "", err
	}
	defer f.Close()

	s.l.Snapshot()
	if len(files) == 0 {
		logrus.Info("No files changed in this command, skipping snapshotting.")
		return "", nil
	}
	logrus.Info("Taking snapshot of files...")
	logrus.Debugf("Taking snapshot of files %v", files)
	snapshottedFiles := make(map[string]bool)

	t := util.NewTar(f)
	defer t.Close()

	// First add to the tar any parent directories that haven't been added
	parentDirs := []string{}
	for _, file := range files {
		parents := util.ParentDirectories(file)
		parentDirs = append(parentDirs, parents...)
	}
	for _, file := range parentDirs {
		file = filepath.Clean(file)
		if val, ok := snapshottedFiles[file]; ok && val {
			continue
		}
		snapshottedFiles[file] = true

		fileAdded, err := s.l.MaybeAdd(file)
		if err != nil {
			return "", fmt.Errorf("Unable to add parent dir %s to layered map: %s", file, err)
		}

		if fileAdded {
			err = t.AddFileToTar(file)
			if err != nil {
				return "", fmt.Errorf("Error adding parent dir %s to tar: %s", file, err)
			}
		}
	}
	// Next add the files themselves to the tar
	for _, file := range files {
		file = filepath.Clean(file)
		if val, ok := snapshottedFiles[file]; ok && val {
			continue
		}
		snapshottedFiles[file] = true

		if err := s.l.Add(file); err != nil {
			return "", fmt.Errorf("Unable to add file %s to layered map: %s", file, err)
		}
		if err := t.AddFileToTar(file); err != nil {
			return "", fmt.Errorf("Error adding file %s to tar: %s", file, err)
		}
	}
	return f.Name(), nil
}

// TakeSnapshotFS takes a snapshot of the filesystem, avoiding directories in the whitelist, and creates
// a tarball of the changed files.
func (s *Snapshotter) TakeSnapshotFS() (string, error) {
	logrus.Info("Taking snapshot of full filesystem...")

	f, err := ioutil.TempFile(snapshotPathPrefix, "")
	if err != nil {
		return "", err
	}
	defer f.Close()

	// Some of the operations that follow (e.g. hashing) depend on the file system being synced,
	// for example the hashing function that determines if files are equal uses the mtime of the files,
	// which can lag if sync is not called. Unfortunately there can still be lag if too much data needs
	// to be flushed or the disk does its own caching/buffering.
	syscall.Sync()

	s.l.Snapshot()
	existingPaths := s.l.GetFlattenedPathsForWhiteOut()
	t := util.NewTar(f)
	defer t.Close()

	// Save the fs state in a map to iterate over later.
	memFs := map[string]os.FileInfo{}
	filepath.Walk(s.directory, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if util.IsInWhitelist(path) {
			if util.IsDestDir(path) {
				logrus.Infof("Skipping paths under %s, as it is a whitelisted directory", path)
				return filepath.SkipDir
			}
			return nil
		}

		memFs[path] = info
		return nil
	})

	// First handle whiteouts
	for p := range memFs {
		delete(existingPaths, p)
	}
	for path := range existingPaths {
		// Only add the whiteout if the directory for the file still exists.
		dir := filepath.Dir(path)
		if _, ok := memFs[dir]; ok {
			if s.l.MaybeAddWhiteout(path) {
				logrus.Infof("Adding whiteout for %s", path)
				if err := t.Whiteout(path); err != nil {
					return "", err
				}
			}
		}
	}

	// Now create the tar.
	for path := range memFs {
		whitelisted, err := util.CheckWhitelist(path)
		if err != nil {
			return "", err
		}
		if whitelisted {
			logrus.Debugf("Not adding %s to layer, as it's whitelisted", path)
			continue
		}
		// Only add to the tar if we add it to the layeredmap.
		maybeAdd, err := s.l.MaybeAdd(path)
		if err != nil {
			return "", err
		}
		if maybeAdd {
			logrus.Debugf("Adding %s to layer, because it was changed.", path)
			if err := t.AddFileToTar(path); err != nil {
				return "", err
			}
		}
	}

	return f.Name(), nil
}
