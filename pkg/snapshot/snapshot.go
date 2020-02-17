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
	"sort"
	"strings"
	"syscall"

	"github.com/GoogleContainerTools/kaniko/pkg/timing"

	"github.com/karrick/godirwalk"

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
	whitelist []util.WhitelistEntry
}

// NewSnapshotter creates a new snapshotter rooted at d
func NewSnapshotter(l *LayeredMap, d string) *Snapshotter {
	return &Snapshotter{l: l, directory: d, whitelist: util.Whitelist()}
}

// Init initializes a new snapshotter
func (s *Snapshotter) Init() error {
	_, _, err := s.scanFullFilesystem()
	return err
}

// Key returns a string based on the current state of the file system
func (s *Snapshotter) Key() (string, error) {
	return s.l.Key()
}

func (s *Snapshotter) SnapshotFiles(files []string) (filesToAdd []string, err error) {
	if len(files) == 0 {
		logrus.Info("Nothing to snapshot")
		return
	}

	logrus.Info("Resolving files")
	//logrus.Debugf("Taking snapshot of files %v", files)

	fileSet := make(map[string]bool)

	for _, f := range files {
		// If the given path is part of the whitelist ignore it
		if util.CheckWhitelistEntries(s.whitelist, f) {
			logrus.Debugf("path %s is whitelisted, ignoring it", f)
			continue
		}

		link, e := util.ResolveSymlinkAncestor(f)
		if err != nil {
			return nil, e
		}

		if f != link {
			logrus.Tracef("updated link %s to %s", f, link)
		}

		//logrus.Tracef("add %s to file set", link)
		fileSet[link] = true

		// If the path is a symlink we need to also consider the target of that
		// link
		evaled, err := filepath.EvalSymlinks(f)
		if err != nil {
			if !os.IsNotExist(err) {
				logrus.Errorf("couldn't eval %s with link %s", f, link)
				return nil, err
			}

			logrus.Warnf("path %s, does not exist", f)
		}

		// If the given path is a symlink and the target is part of the whitelist
		// ignore the target
		if util.CheckWhitelistEntries(s.whitelist, evaled) {
			logrus.Debugf("path %s is whitelisted, ignoring it", evaled)
			continue
		}

		//logrus.Tracef("add %s to file set", evaled)
		fileSet[evaled] = true
	}

	filesToAdd = make([]string, 0, len(fileSet))

	for file := range fileSet {
		filesToAdd = append(filesToAdd, file)
	}

	// Also add parent directories to keep the permission of them correctly.
	filesToAdd = filesWithParentDirs(filesToAdd)

	return
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

	filesToAdd, err := s.SnapshotFiles(files)
	if err != nil {
		return "", nil
	}

	logrus.Info("Taking snapshot of files...")
	logrus.Debugf("Taking snapshot of files %v", files)

	//// Also add parent directories to keep the permission of them correctly.
	//filesToAdd := filesWithParentDirs(files)

	sort.Strings(filesToAdd)

	// Add files to the layered map
	for _, file := range filesToAdd {
		if err := s.l.Add(file); err != nil {
			return "", fmt.Errorf("unable to add file %s to layered map: %s", file, err)
		}
	}

	t := util.NewTar(f)
	defer t.Close()
	if err := writeToTar(t, filesToAdd, nil); err != nil {
		return "", err
	}
	return f.Name(), nil
}

// TakeSnapshotFS takes a snapshot of the filesystem, avoiding directories in the whitelist, and creates
// a tarball of the changed files.
func (s *Snapshotter) TakeSnapshotFS() (string, error) {
	f, err := ioutil.TempFile(snapshotPathPrefix, "")
	if err != nil {
		return "", err
	}
	defer f.Close()
	t := util.NewTar(f)
	defer t.Close()

	filesToAdd, filesToWhiteOut, err := s.scanFullFilesystem()
	if err != nil {
		return "", err
	}

	if err := writeToTar(t, filesToAdd, filesToWhiteOut); err != nil {
		return "", err
	}
	return f.Name(), nil
}

func (s *Snapshotter) scanFullFilesystem() ([]string, []string, error) {
	logrus.Info("Taking snapshot of full filesystem...")

	// Some of the operations that follow (e.g. hashing) depend on the file system being synced,
	// for example the hashing function that determines if files are equal uses the mtime of the files,
	// which can lag if sync is not called. Unfortunately there can still be lag if too much data needs
	// to be flushed or the disk does its own caching/buffering.
	syscall.Sync()

	s.l.Snapshot()

	timer := timing.Start("Walking filesystem")
	// Save the fs state in a map to iterate over later.
	memFs := map[string]*godirwalk.Dirent{}
	godirwalk.Walk(s.directory, &godirwalk.Options{
		Callback: func(path string, ent *godirwalk.Dirent) error {
			if util.IsInWhitelist(path) {
				if util.IsDestDir(path) {
					logrus.Tracef("Skipping paths under %s, as it is a whitelisted directory", path)
					return filepath.SkipDir
				}
				return nil
			}
			memFs[path] = ent
			return nil
		},
		Unsorted: true,
	},
	)
	timing.DefaultRun.Stop(timer)

	filesToResolve := make([]string, 0, len(memFs))
	for file := range memFs {
		if strings.HasPrefix(file, "/tmp/dir") {
			logrus.Infof("found %s", file)
		}
		filesToResolve = append(filesToResolve, file)
	}

	//resolvedFiles := filesToResolve
	resolvedFiles, err := s.SnapshotFiles(filesToResolve)
	if err != nil {
		return nil, nil, err
	}

	resolvedMemFs := make(map[string]bool)
	for _, f := range resolvedFiles {
		if strings.HasPrefix(f, "/tmp/dir") {
			logrus.Infof("found again %s", f)
		}
		resolvedMemFs[f] = true
	}

	// First handle whiteouts
	//   Get a list of all the files that existed before this layer
	existingPaths := s.l.getFlattenedPathsForWhiteOut()

	//   Find the delta by removing everything left in this layer.
	for p := range resolvedMemFs {
		delete(existingPaths, p)
	}

	//   The paths left here are the ones that have been deleted in this layer.
	filesToWhiteOut := []string{}
	for path := range existingPaths {
		// Only add the whiteout if the directory for the file still exists.
		dir := filepath.Dir(path)
		if _, ok := resolvedMemFs[dir]; ok {
			if s.l.MaybeAddWhiteout(path) {
				logrus.Debugf("Adding whiteout for %s", path)
				filesToWhiteOut = append(filesToWhiteOut, path)
			}
		}
	}

	filesToAdd := []string{}
	for path := range resolvedMemFs {
		if util.CheckWhitelist(path) {
			logrus.Tracef("Not adding %s to layer, as it's whitelisted", path)
			continue
		}
		// Only add changed files.
		fileChanged, err := s.l.CheckFileChange(path)
		if err != nil {
			return nil, nil, fmt.Errorf("could not check if file has changed %s %s", path, err)
		}
		if fileChanged {
			logrus.Tracef("Adding file %s to layer, because it was changed.", path)
			filesToAdd = append(filesToAdd, path)
		}
	}

	//// Also add parent directories to keep their permissions correctly.
	//filesToAdd = filesWithParentDirs(filesToAdd)

	sort.Strings(filesToAdd)
	// Add files to the layered map
	for _, file := range filesToAdd {
		if err := s.l.Add(file); err != nil {
			return nil, nil, fmt.Errorf("unable to add file %s to layered map: %s", file, err)
		}
	}
	return filesToAdd, filesToWhiteOut, nil
}

func writeToTar(t util.Tar, files, whiteouts []string) error {
	timer := timing.Start("Writing tar file")
	defer timing.DefaultRun.Stop(timer)
	// Now create the tar.
	for _, path := range whiteouts {
		if err := t.Whiteout(path); err != nil {
			return err
		}
	}
	for _, path := range files {
		if err := t.AddFileToTar(path); err != nil {
			return err
		}
	}
	return nil
}

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

// filesWithLinks returns the symlink and the target path if its exists.
func filesWithLinks(path string) ([]string, error) {
	link, err := util.GetSymLink(path)
	if err == util.ErrNotSymLink {
		return []string{path}, nil
	} else if err != nil {
		return nil, err
	}
	// Add symlink if it exists in the FS
	if !filepath.IsAbs(link) {
		link = filepath.Join(filepath.Dir(path), link)
	}
	if _, err := os.Stat(link); err != nil {
		return []string{path}, nil
	}
	return []string{path, link}, nil
}
