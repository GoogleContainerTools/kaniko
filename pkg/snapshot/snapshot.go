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
	"io"
	"io/ioutil"
	"os"
	"path/filepath"

	"github.com/GoogleContainerTools/kaniko/pkg/constants"
	"github.com/GoogleContainerTools/kaniko/pkg/util"
	"github.com/sirupsen/logrus"
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
	buf := bytes.NewBuffer([]byte{})
	var filesAdded bool
	var err error
	if files == nil {
		filesAdded, err = s.snapShotFS(buf)
	} else {
		filesAdded, err = s.snapshotFiles(buf, files)
	}
	if err != nil {
		return nil, err
	}
	contents := buf.Bytes()
	if !filesAdded {
		return nil, nil
	}
	return contents, err
}

// snapshotFiles takes a snapshot of specific files
// Used for ADD/COPY commands, when we know which files have changed
func (s *Snapshotter) snapshotFiles(f io.Writer, files []string) (bool, error) {
	s.hardlinks = map[uint64]string{}
	s.l.Snapshot()
	if len(files) == 0 {
		logrus.Info("No files changed in this command, skipping snapshotting.")
		return false, nil
	}
	logrus.Infof("Taking snapshot of files %v...", files)
	snapshottedFiles := make(map[string]bool)
	for _, file := range files {
		parentDirs := util.ParentDirectories(file)
		files = append(parentDirs, files...)
	}
	filesAdded := false
	w := tar.NewWriter(f)
	defer w.Close()

	// Now create the tar.
	for _, file := range files {
		file = filepath.Clean(file)
		if val, ok := snapshottedFiles[file]; ok && val {
			continue
		}
		whitelisted, err := util.CheckWhitelist(file)
		if err != nil {
			return false, err
		}
		if whitelisted && !isBuildFile(file) {
			logrus.Infof("Not adding %s to layer, as it's whitelisted", file)
			continue
		}
		snapshottedFiles[file] = true
		info, err := os.Lstat(file)
		if err != nil {
			return false, err
		}
		// Only add to the tar if we add it to the layeredmap.
		addFile, err := s.l.MaybeAdd(file)
		if err != nil {
			return false, err
		}
		if addFile {
			filesAdded = true
			if err := util.AddToTar(file, info, s.hardlinks, w); err != nil {
				return false, err
			}
		}
	}
	return filesAdded, nil
}

func isBuildFile(file string) bool {
	for _, buildFile := range constants.KanikoBuildFiles {
		if file == buildFile {
			return true
		}
	}
	return false
}

func (s *Snapshotter) snapShotFS(f io.Writer) (bool, error) {
	logrus.Info("Taking snapshot of full filesystem...")
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
			addWhiteout, err := s.l.MaybeAddWhiteout(path)
			if err != nil {
				return false, nil
			}
			if addWhiteout {
				logrus.Infof("Adding whiteout for %s", path)
				filesAdded = true
				if err := util.Whiteout(path, w); err != nil {
					return false, err
				}
			}
		}
	}

	// Now create the tar.
	for path, info := range memFs {
		whitelisted, err := util.CheckWhitelist(path)
		if err != nil {
			return false, err
		}
		if whitelisted {
			logrus.Debugf("Not adding %s to layer, as it's whitelisted", path)
			continue
		}

		// Only add to the tar if we add it to the layeredmap.
		maybeAdd, err := s.l.MaybeAdd(path)
		if err != nil {
			return false, err
		}
		if maybeAdd {
			logrus.Debugf("Adding %s to layer, because it was changed.", path)
			filesAdded = true
			if err := util.AddToTar(path, info, s.hardlinks, w); err != nil {
				return false, err
			}
		}
	}

	return filesAdded, nil
}
