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
	"archive/tar"
	"bytes"

	"github.com/GoogleContainerTools/kaniko/pkg/util"
	"github.com/sirupsen/logrus"

	"io"
	"io/ioutil"
	"os"
	"path/filepath"
)

// Snapshotter holds the root directory from which to take snapshots, and a list of snapshots taken
type Snapshotter struct {
	l         *LayeredMap
	directory string
	hardlinks map[uint64]string
}

// NewSnapshotter creates a new snapshotter rooted at d
func NewSnapshotter(l *LayeredMap, d string) *Snapshotter {
	return &Snapshotter{l: l, directory: d}
}

// Init initializes a new snapshotter
func (s *Snapshotter) Init() error {
	if _, err := s.snapShotFS(ioutil.Discard); err != nil {
		return err
	}
	return nil
}

// TakeSnapshot takes a snapshot of the filesystem, avoiding directories in the whitelist, and creates
// a tarball of the changed files. Return contents of the tarball, and whether or not any files were changed
func (s *Snapshotter) TakeSnapshot(files []string) ([]byte, error) {
	if files != nil {
		return s.TakeSnapshotOfFiles(files)
	}
	logrus.Info("Taking snapshot of full filesystem...")
	buf := bytes.NewBuffer([]byte{})
	filesAdded, err := s.snapShotFS(buf)
	if err != nil {
		return nil, err
	}
	contents := buf.Bytes()
	if !filesAdded {
		return nil, nil
	}
	return contents, err
}

// TakeSnapshotOfFiles takes a snapshot of specific files
// Used for ADD/COPY commands, when we know which files have changed
func (s *Snapshotter) TakeSnapshotOfFiles(files []string) ([]byte, error) {
	logrus.Infof("Taking snapshot of files %v...", files)
	s.l.Snapshot()
	if len(files) == 0 {
		logrus.Info("No files changed in this command, skipping snapshotting.")
		return nil, nil
	}
	buf := bytes.NewBuffer([]byte{})
	w := tar.NewWriter(buf)
	defer w.Close()
	filesAdded := false
	for _, file := range files {
		info, err := os.Lstat(file)
		if err != nil {
			return nil, err
		}
		if util.PathInWhitelist(file, s.directory) {
			logrus.Debugf("Not adding %s to layer, as it is whitelisted", file)
			continue
		}
		// Only add to the tar if we add it to the layeredmap.
		maybeAdd, err := s.l.MaybeAdd(file)
		if err != nil {
			return nil, err
		}
		if maybeAdd {
			filesAdded = true
			util.AddToTar(file, info, s.hardlinks, w)
		}
	}
	if !filesAdded {
		return nil, nil
	}
	return ioutil.ReadAll(buf)
}

func (s *Snapshotter) snapShotFS(f io.Writer) (bool, error) {
	s.hardlinks = map[uint64]string{}
	s.l.Snapshot()
	existingPaths := s.l.GetFlattenedPathsForWhiteOut()
	filesAdded := false
	w := tar.NewWriter(f)
	defer w.Close()

	// Save the fs state in a map to iterate over later.
	memFs := map[string]os.FileInfo{}
	filepath.Walk(s.directory, func(path string, info os.FileInfo, err error) error {
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
			logrus.Infof("Adding whiteout for %s", path)
			filesAdded = true
			if err := util.Whiteout(path, w); err != nil {
				return false, err
			}
		}
	}

	// Now create the tar.
	for path, info := range memFs {
		if util.PathInWhitelist(path, s.directory) {
			logrus.Debugf("Not adding %s to layer, as it's whitelisted", path)
			continue
		}

		// Only add to the tar if we add it to the layeredmap.
		maybeAdd, err := s.l.MaybeAdd(path)
		if err != nil {
			return false, err
		}
		if maybeAdd {
			filesAdded = true
			if err := util.AddToTar(path, info, s.hardlinks, w); err != nil {
				return false, err
			}
		}
	}

	return filesAdded, nil
}
