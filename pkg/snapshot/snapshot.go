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
	"github.com/GoogleCloudPlatform/k8s-container-builder/pkg/util"
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
	snapshots []string
}

// NewSnapshotter creates a new snapshotter rooted at d
func NewSnapshotter(l *LayeredMap, d string) *Snapshotter {
	return &Snapshotter{l: l, directory: d, snapshots: []string{}}
}

// Init initializes a new snapshotter
func (s *Snapshotter) Init() error {
	if _, err := s.snapShotFS(ioutil.Discard); err != nil {
		return err
	}
	return nil
}

// TakeSnapshot takes a snapshot of the filesystem, avoiding directories in the whitelist
// It stores changed files in a tar, and returns the contents of this tar at the end
func (s *Snapshotter) TakeSnapshot() ([]byte, error) {

	buf := bytes.NewBuffer([]byte{})
	added, err := s.snapShotFS(buf)
	if err != nil {
		return nil, err
	}
	if !added {
		logrus.Infof("No files were changed in this command, this layer will not be appended.")
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	// Add buffer contents until buffer is empty
	var contents []byte
	for {
		next := buf.Next(buf.Len())
		if len(next) == 0 {
			break
		}
		contents = append(contents, next...)
	}
	return contents, nil
}

func (s *Snapshotter) snapShotFS(f io.Writer) (bool, error) {
	s.l.Snapshot()
	added := false
	w := tar.NewWriter(f)
	defer w.Close()

	err := filepath.Walk(s.directory, func(path string, info os.FileInfo, err error) error {
		if util.PathInWhitelist(path, s.directory) {
			return nil
		}

		// Only add to the tar if we add it to the layeredmap.
		if s.l.MaybeAdd(path) {
			added = true
			return util.AddToTar(path, info, w)
		}
		return nil
	})
	return added, err
}
