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
	"archive/tar"
	"bufio"
	"bytes"
	"io"
	"io/ioutil"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"syscall"
	"time"

	"github.com/GoogleContainerTools/kaniko/pkg/constants"
	"github.com/docker/docker/builder/dockerignore"
	"github.com/docker/docker/pkg/fileutils"
	"github.com/google/go-containerregistry/pkg/v1"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
)

type WhitelistEntry struct {
	Path            string
	PrefixMatchOnly bool
}

var whitelist = []WhitelistEntry{
	{
		Path:            "/kaniko",
		PrefixMatchOnly: false,
	},
	{
		// /var/run is a special case. It's common to mount in /var/run/docker.sock or something similar
		// which leads to a special mount on the /var/run/docker.sock file itself, but the directory to exist
		// in the image with no way to tell if it came from the base image or not.
		Path:            "/var/run",
		PrefixMatchOnly: false,
	},
	{
		// similarly, we whitelist /etc/mtab, since there is no way to know if the file was mounted or came
		// from the base image
		Path:            "/etc/mtab",
		PrefixMatchOnly: false,
	},
}

var excluded []string

// GetFSFromImage extracts the layers of img to root
// It returns a list of all files extracted
func GetFSFromImage(root string, img v1.Image) ([]string, error) {
	if err := DetectFilesystemWhitelist(constants.WhitelistPath); err != nil {
		return nil, err
	}
	logrus.Debugf("Mounted directories: %v", whitelist)
	layers, err := img.Layers()
	if err != nil {
		return nil, err
	}
	extractedFiles := []string{}

	for i, l := range layers {
		logrus.Debugf("Extracting layer %d", i)
		r, err := l.Uncompressed()
		if err != nil {
			return nil, err
		}
		tr := tar.NewReader(r)
		for {
			hdr, err := tr.Next()
			if err == io.EOF {
				break
			}
			if err != nil {
				return nil, err
			}
			path := filepath.Join(root, filepath.Clean(hdr.Name))
			base := filepath.Base(path)
			dir := filepath.Dir(path)
			if strings.HasPrefix(base, ".wh.") {
				logrus.Debugf("Whiting out %s", path)
				name := strings.TrimPrefix(base, ".wh.")
				if err := os.RemoveAll(filepath.Join(dir, name)); err != nil {
					return nil, errors.Wrapf(err, "removing whiteout %s", hdr.Name)
				}
				continue
			}
			if err := extractFile(root, hdr, tr); err != nil {
				return nil, err
			}
			extractedFiles = append(extractedFiles, filepath.Join(root, filepath.Clean(hdr.Name)))
		}
	}
	return extractedFiles, nil
}

// DeleteFilesystem deletes the extracted image file system
func DeleteFilesystem() error {
	logrus.Info("Deleting filesystem...")
	return filepath.Walk(constants.RootDir, func(path string, info os.FileInfo, _ error) error {
		whitelisted, err := CheckWhitelist(path)
		if err != nil {
			return err
		}
		if whitelisted || ChildDirInWhitelist(path, constants.RootDir) {
			logrus.Debugf("Not deleting %s, as it's whitelisted", path)
			return nil
		}
		if path == constants.RootDir {
			return nil
		}
		return os.RemoveAll(path)
	})
}

// ChildDirInWhitelist returns true if there is a child file or directory of the path in the whitelist
func ChildDirInWhitelist(path, directory string) bool {
	for _, d := range constants.KanikoBuildFiles {
		dirPath := filepath.Join(directory, d)
		if HasFilepathPrefix(dirPath, path, false) {
			return true
		}
	}
	for _, d := range whitelist {
		dirPath := filepath.Join(directory, d.Path)
		if HasFilepathPrefix(dirPath, path, d.PrefixMatchOnly) {
			return true
		}
	}
	return false
}

// unTar returns a list of files that have been extracted from the tar archive at r to the path at dest
func unTar(r io.Reader, dest string) ([]string, error) {
	var extractedFiles []string
	tr := tar.NewReader(r)
	for {
		hdr, err := tr.Next()
		if err == io.EOF {
			break
		}
		if err != nil {
			return nil, err
		}
		if err := extractFile(dest, hdr, tr); err != nil {
			return nil, err
		}
		extractedFiles = append(extractedFiles, dest)
	}
	return extractedFiles, nil
}

func extractFile(dest string, hdr *tar.Header, tr io.Reader) error {
	path := filepath.Join(dest, filepath.Clean(hdr.Name))
	base := filepath.Base(path)
	dir := filepath.Dir(path)
	mode := hdr.FileInfo().Mode()
	uid := hdr.Uid
	gid := hdr.Gid

	whitelisted, err := CheckWhitelist(path)
	if err != nil {
		return err
	}
	if whitelisted && !checkWhitelistRoot(dest) {
		logrus.Debugf("Not adding %s because it is whitelisted", path)
		return nil
	}
	switch hdr.Typeflag {
	case tar.TypeReg:
		logrus.Debugf("creating file %s", path)
		// It's possible a file is in the tar before it's directory.
		if _, err := os.Stat(dir); os.IsNotExist(err) {
			logrus.Debugf("base %s for file %s does not exist. Creating.", base, path)
			if err := os.MkdirAll(dir, 0755); err != nil {
				return err
			}
		}
		// Check if something already exists at path (symlinks etc.)
		// If so, delete it
		if FilepathExists(path) {
			if err := os.RemoveAll(path); err != nil {
				return errors.Wrapf(err, "error removing %s to make way for new file.", path)
			}
		}
		currFile, err := os.Create(path)
		if err != nil {
			return err
		}
		// manually set permissions on file, since the default umask (022) will interfere
		if err = os.Chmod(path, mode); err != nil {
			return err
		}
		if _, err = io.Copy(currFile, tr); err != nil {
			return err
		}
		if err = currFile.Chown(uid, gid); err != nil {
			return err
		}
		currFile.Close()
	case tar.TypeDir:
		logrus.Debugf("creating dir %s", path)
		if err := os.MkdirAll(path, mode); err != nil {
			return err
		}
		// In some cases, MkdirAll doesn't change the permissions, so run Chmod
		if err := os.Chmod(path, mode); err != nil {
			return err
		}
		if err := os.Chown(path, uid, gid); err != nil {
			return err
		}

	case tar.TypeLink:
		logrus.Debugf("link from %s to %s", hdr.Linkname, path)
		whitelisted, err := CheckWhitelist(hdr.Linkname)
		if err != nil {
			return err
		}
		if whitelisted {
			logrus.Debugf("skipping symlink from %s to %s because %s is whitelisted", hdr.Linkname, path, hdr.Linkname)
			return nil
		}
		// The base directory for a link may not exist before it is created.
		if err := os.MkdirAll(dir, 0755); err != nil {
			return err
		}
		// Check if something already exists at path
		// If so, delete it
		if FilepathExists(path) {
			if err := os.RemoveAll(path); err != nil {
				return errors.Wrapf(err, "error removing %s to make way for new link", hdr.Name)
			}
		}
		link := filepath.Clean(filepath.Join(dest, hdr.Linkname))
		if err := os.Link(link, path); err != nil {
			return err
		}

	case tar.TypeSymlink:
		logrus.Debugf("symlink from %s to %s", hdr.Linkname, path)
		// The base directory for a symlink may not exist before it is created.
		if err := os.MkdirAll(dir, 0755); err != nil {
			return err
		}
		// Check if something already exists at path
		// If so, delete it
		if FilepathExists(path) {
			if err := os.RemoveAll(path); err != nil {
				return errors.Wrapf(err, "error removing %s to make way for new symlink", hdr.Name)
			}
		}
		if err := os.Symlink(hdr.Linkname, path); err != nil {
			return err
		}
	}
	return nil
}

func IsInWhitelist(path string) bool {
	for _, wl := range whitelist {
		if !wl.PrefixMatchOnly && path == wl.Path {
			return true
		}
	}
	return false
}

func CheckWhitelist(path string) (bool, error) {
	abs, err := filepath.Abs(path)
	if err != nil {
		logrus.Infof("unable to get absolute path for %s", path)
		return false, err
	}
	for _, wl := range whitelist {
		if HasFilepathPrefix(abs, wl.Path, wl.PrefixMatchOnly) {
			return true, nil
		}
	}
	return false, nil
}

func checkWhitelistRoot(root string) bool {
	if root == constants.RootDir {
		return false
	}
	for _, wl := range whitelist {
		if HasFilepathPrefix(root, wl.Path, wl.PrefixMatchOnly) {
			return true
		}
	}
	return false
}

// Get whitelist from roots of mounted files
// Each line of /proc/self/mountinfo is in the form:
// 36 35 98:0 /mnt1 /mnt2 rw,noatime master:1 - ext3 /dev/root rw,errors=continue
// (1)(2)(3)   (4)   (5)      (6)      (7)   (8) (9)   (10)         (11)
// Where (5) is the mount point relative to the process's root
// From: https://www.kernel.org/doc/Documentation/filesystems/proc.txt
func DetectFilesystemWhitelist(path string) error {
	f, err := os.Open(path)
	if err != nil {
		return err
	}
	defer f.Close()
	reader := bufio.NewReader(f)
	for {
		line, err := reader.ReadString('\n')
		logrus.Debugf("Read the following line from %s: %s", path, line)
		if err != nil && err != io.EOF {
			return err
		}
		lineArr := strings.Split(line, " ")
		if len(lineArr) < 5 {
			if err == io.EOF {
				logrus.Debugf("Reached end of file %s", path)
				break
			}
			continue
		}
		if lineArr[4] != constants.RootDir {
			logrus.Debugf("Appending %s from line: %s", lineArr[4], line)
			whitelist = append(whitelist, WhitelistEntry{
				Path:            lineArr[4],
				PrefixMatchOnly: false,
			})
		}
		if err == io.EOF {
			logrus.Debugf("Reached end of file %s", path)
			break
		}
	}
	return nil
}

// RelativeFiles returns a list of all files at the filepath relative to root
func RelativeFiles(fp string, root string) ([]string, error) {
	var files []string
	fullPath := filepath.Join(root, fp)
	logrus.Debugf("Getting files and contents at root %s", fullPath)
	err := filepath.Walk(fullPath, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		whitelisted, err := CheckWhitelist(path)
		if err != nil {
			return err
		}
		if whitelisted && !HasFilepathPrefix(path, root, false) {
			return nil
		}
		if err != nil {
			return err
		}
		relPath, err := filepath.Rel(root, path)
		if err != nil {
			return err
		}
		files = append(files, relPath)
		return nil
	})
	return files, err
}

// ParentDirectories returns a list of paths to all parent directories
// Ex. /some/temp/dir -> [/, /some, /some/temp, /some/temp/dir]
func ParentDirectories(path string) []string {
	path = filepath.Clean(path)
	dirs := strings.Split(path, "/")
	dirPath := constants.RootDir
	paths := []string{constants.RootDir}
	for index, dir := range dirs {
		if dir == "" || index == (len(dirs)-1) {
			continue
		}
		dirPath = filepath.Join(dirPath, dir)
		paths = append(paths, dirPath)
	}
	return paths
}

// FilepathExists returns true if the path exists
func FilepathExists(path string) bool {
	_, err := os.Lstat(path)
	return !os.IsNotExist(err)
}

// CreateFile creates a file at path and copies over contents from the reader
func CreateFile(path string, reader io.Reader, perm os.FileMode, uid uint32, gid uint32) error {
	// Create directory path if it doesn't exist
	baseDir := filepath.Dir(path)
	if _, err := os.Lstat(baseDir); os.IsNotExist(err) {
		logrus.Debugf("baseDir %s for file %s does not exist. Creating.", baseDir, path)
		if err := os.MkdirAll(baseDir, 0755); err != nil {
			return err
		}
	}
	dest, err := os.Create(path)
	if err != nil {
		return err
	}
	defer dest.Close()
	if _, err := io.Copy(dest, reader); err != nil {
		return err
	}
	if err := dest.Chmod(perm); err != nil {
		return err
	}
	return dest.Chown(int(uid), int(gid))
}

// AddVolumePathToWhitelist adds the given path to the whitelist with
// PrefixMatchOnly set to true. Snapshotting will ignore paths prefixed
// with the volume, but the volume itself will not be ignored.
func AddVolumePathToWhitelist(path string) error {
	logrus.Infof("adding volume %s to whitelist", path)
	whitelist = append(whitelist, WhitelistEntry{
		Path:            path,
		PrefixMatchOnly: true,
	})
	return nil
}

// DownloadFileToDest downloads the file at rawurl to the given dest for the ADD command
// From add command docs:
// 	1. If <src> is a remote file URL:
// 		- destination will have permissions of 0600
// 		- If remote file has HTTP Last-Modified header, we set the mtime of the file to that timestamp
func DownloadFileToDest(rawurl, dest string) error {
	resp, err := http.Get(rawurl)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	// TODO: set uid and gid according to current user
	if err := CreateFile(dest, resp.Body, 0600, 0, 0); err != nil {
		return err
	}
	mTime := time.Time{}
	lastMod := resp.Header.Get("Last-Modified")
	if lastMod != "" {
		if parsedMTime, err := http.ParseTime(lastMod); err == nil {
			mTime = parsedMTime
		}
	}
	return os.Chtimes(dest, mTime, mTime)
}

// If the given uid/gid are negative, returns the uid/gid from the
// file info.
// Otherwise returns the given uid/gid
func determineChownUIDGid(fileInfo os.FileInfo, chownUID, chownGid int) (uint32, uint32) {
	uid := uint32(chownUID)
	if chownUID < 0 {
		uid = fileInfo.Sys().(*syscall.Stat_t).Uid
	}

	gid := uint32(chownGid)
	if chownGid < 0 {
		gid = fileInfo.Sys().(*syscall.Stat_t).Gid
	}
	return uid, gid
}

// CopyDir copies the file or directory at src to dest
// will chown the file or directory to the uid/gid if non-negative
// It returns a list of files it copied over
func CopyDir(src, dest, buildcontext string, chownUID, chownGid int) ([]string, error) {
	files, err := RelativeFiles("", src)
	if err != nil {
		return nil, err
	}
	var copiedFiles []string
	for _, file := range files {
		fullPath := filepath.Join(src, file)
		fi, err := os.Lstat(fullPath)
		if err != nil {
			return nil, err
		}

		uid, gid := determineChownUIDGid(fi, chownUID, chownGid)

		if excludeFile(fullPath, buildcontext) {
			logrus.Debugf("%s found in .dockerignore, ignoring", src)
			continue
		}
		destPath := filepath.Join(dest, file)
		if fi.IsDir() {
			logrus.Debugf("Creating directory %s", destPath)

			if err := os.MkdirAll(destPath, fi.Mode()); err != nil {
				return nil, err
			}
			if err := os.Chown(destPath, int(uid), int(gid)); err != nil {
				return nil, err
			}
		} else if fi.Mode()&os.ModeSymlink != 0 {
			// If file is a symlink, we want to create the same relative symlink
			if _, err := CopySymlink(fullPath, destPath, buildcontext); err != nil {
				return nil, err
			}
		} else {
			// ... Else, we want to copy over a file
			if _, err := CopyFile(fullPath, destPath, buildcontext, chownUID, chownGid); err != nil {
				return nil, err
			}
		}
		copiedFiles = append(copiedFiles, destPath)
	}
	return copiedFiles, nil
}

// CopySymlink copies the symlink at src to dest
// NOTE: Docker does not allow for copying symlinks and will copy as a regular
// file. Trying to stat/chown symlinks here like we do for CopyFile would result
// in trying to stat/chown the underlying file (which therefore needs to exist, but won't in unit tests)
// it also doesn't totally make sense to chown the actual link itself, so instead
// we just don't allow COPY --chown for symlinks (since docker doesn't either)
func CopySymlink(src, dest, buildcontext string) (bool, error) {
	if excludeFile(src, buildcontext) {
		logrus.Debugf("%s found in .dockerignore, ignoring", src)
		return true, nil
	}
	link, err := os.Readlink(src)
	if err != nil {
		return false, err
	}
	if FilepathExists(dest) {
		if err := os.RemoveAll(dest); err != nil {
			return false, err
		}
	}
	return false, os.Symlink(link, dest)
}

// CopyFile copies the file at src to dest
// If uid/gid are non-negative, files will be created with that owner/group
// otherwise uid/gid will be preserved
func CopyFile(src, dest, buildcontext string, uidSigned, gidSigned int) (bool, error) {
	if excludeFile(src, buildcontext) {
		logrus.Debugf("%s found in .dockerignore, ignoring", src)
		return true, nil
	}
	fi, err := os.Stat(src)
	if err != nil {
		return false, err
	}
	logrus.Debugf("Copying file %s to %s", src, dest)
	srcFile, err := os.Open(src)
	if err != nil {
		return false, err
	}
	defer srcFile.Close()

	uid, gid := determineChownUIDGid(fi, uidSigned, gidSigned)
	return false, CreateFile(dest, srcFile, fi.Mode(), uid, gid)
}

// GetExcludedFiles gets a list of files to exclude from the .dockerignore
func GetExcludedFiles(buildcontext string) error {
	path := filepath.Join(buildcontext, ".dockerignore")
	if !FilepathExists(path) {
		return nil
	}
	contents, err := ioutil.ReadFile(path)
	if err != nil {
		return errors.Wrap(err, "parsing .dockerignore")
	}
	reader := bytes.NewBuffer(contents)
	excluded, err = dockerignore.ReadAll(reader)
	return err
}

// excludeFile returns true if the .dockerignore specified this file should be ignored
func excludeFile(path, buildcontext string) bool {
	if HasFilepathPrefix(path, buildcontext, false) {
		var err error
		path, err = filepath.Rel(buildcontext, path)
		if err != nil {
			logrus.Errorf("unable to get relative path, including %s in build: %v", path, err)
			return false
		}
	}
	match, err := fileutils.Matches(path, excluded)
	if err != nil {
		logrus.Errorf("error matching, including %s in build: %v", path, err)
		return false
	}
	return match
}

// HasFilepathPrefix checks  if the given file path begins with prefix
func HasFilepathPrefix(path, prefix string, prefixMatchOnly bool) bool {
	path = filepath.Clean(path)
	prefix = filepath.Clean(prefix)
	pathArray := strings.Split(path, "/")
	prefixArray := strings.Split(prefix, "/")

	if len(pathArray) < len(prefixArray) {
		return false
	}
	if prefixMatchOnly && len(pathArray) == len(prefixArray) {
		return false
	}
	for index := range prefixArray {
		if prefixArray[index] == pathArray[index] {
			continue
		}
		return false
	}
	return true
}
