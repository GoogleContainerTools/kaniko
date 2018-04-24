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
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/google/go-containerregistry/v1"

	"github.com/GoogleContainerTools/kaniko/pkg/constants"
	"github.com/sirupsen/logrus"
)

var whitelist = []string{
	"/kaniko",
	// /var/run is a special case. It's common to mount in /var/run/docker.sock or something similar
	// which leads to a special mount on the /var/run/docker.sock file itself, but the directory to exist
	// in the image with no way to tell if it came from the base image or not.
	"/var/run",
}
var volumeWhitelist = []string{}

func GetFSFromImage(img v1.Image) error {
	whitelist, err := fileSystemWhitelist(constants.WhitelistPath)
	if err != nil {
		return err
	}
	logrus.Infof("Mounted directories: %v", whitelist)
	layers, err := img.Layers()
	if err != nil {
		return err
	}

	fs := map[string]struct{}{}
	whiteouts := map[string]struct{}{}

	for i := len(layers) - 1; i >= 0; i-- {
		logrus.Infof("Unpacking layer: %d", i)
		l := layers[i]
		r, err := l.Uncompressed()
		if err != nil {
			return err
		}
		tr := tar.NewReader(r)
		for {
			hdr, err := tr.Next()
			if err == io.EOF {
				break
			}
			if err != nil {
				return err
			}
			path := filepath.Join("/", filepath.Clean(hdr.Name))
			base := filepath.Base(path)
			dir := filepath.Dir(path)
			if strings.HasPrefix(base, ".wh.") {
				logrus.Infof("Whiting out %s", path)
				name := strings.TrimPrefix(base, ".wh.")
				whiteouts[filepath.Join(dir, name)] = struct{}{}
				continue
			}

			if checkWhiteouts(path, whiteouts) {
				logrus.Infof("Not adding %s because it is whited out", path)
				continue
			}
			if _, ok := fs[path]; ok {
				logrus.Infof("Not adding %s because it was added by a prior layer", path)
				continue
			}

			if checkWhitelist(path, whitelist) {
				logrus.Infof("Not adding %s because it is whitelisted", path)
				continue
			}
			fs[path] = struct{}{}

			if err := extractFile("/", hdr, tr); err != nil {
				return err
			}
		}
	}
	return nil
}

func unTar(r io.Reader, dest string) error {
	tr := tar.NewReader(r)
	for {
		hdr, err := tr.Next()
		if err == io.EOF {
			break
		}
		if err != nil {
			return err
		}
		if err := extractFile(dest, hdr, tr); err != nil {
			return err
		}
	}
	return nil
}

func extractFile(dest string, hdr *tar.Header, tr io.Reader) error {
	path := filepath.Join(dest, filepath.Clean(hdr.Name))
	base := filepath.Base(path)
	dir := filepath.Dir(path)
	mode := hdr.FileInfo().Mode()
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

	case tar.TypeLink:
		logrus.Debugf("link from %s to %s", hdr.Linkname, path)
		// The base directory for a link may not exist before it is created.
		dir := filepath.Dir(path)
		if err := os.MkdirAll(dir, 0755); err != nil {
			return err
		}
		if err := os.Symlink(filepath.Clean(filepath.Join("/", hdr.Linkname)), path); err != nil {
			return err
		}
	case tar.TypeSymlink:
		logrus.Debugf("symlink from %s to %s", hdr.Linkname, path)
		// The base directory for a symlink may not exist before it is created.
		dir := filepath.Dir(path)
		if err := os.MkdirAll(dir, 0755); err != nil {
			return err
		}
		if err := os.Symlink(hdr.Linkname, path); err != nil {
			return err
		}
	}
	return nil
}

func PathInWhitelist(path, directory string) bool {
	for _, c := range constants.KanikoBuildFiles {
		if path == c {
			return false
		}
	}
	for _, d := range whitelist {
		dirPath := filepath.Join(directory, d)
		if HasFilepathPrefix(path, dirPath) {
			return true
		}
	}
	return false
}

func checkWhiteouts(path string, whiteouts map[string]struct{}) bool {
	// Don't add the file if it or it's directory are whited out.
	if _, ok := whiteouts[path]; ok {
		return true
	}
	for wd := range whiteouts {
		if HasFilepathPrefix(path, wd) {
			logrus.Infof("Not adding %s because it's directory is whited out", path)
			return true
		}
	}
	return false
}

func checkWhitelist(path string, whitelist []string) bool {
	for _, wl := range whitelist {
		if HasFilepathPrefix(path, wl) {
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
func fileSystemWhitelist(path string) ([]string, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer f.Close()
	reader := bufio.NewReader(f)
	for {
		line, err := reader.ReadString('\n')
		logrus.Debugf("Read the following line from %s: %s", path, line)
		if err != nil && err != io.EOF {
			return nil, err
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
			whitelist = append(whitelist, lineArr[4])
		}
		if err == io.EOF {
			logrus.Debugf("Reached end of file %s", path)
			break
		}
	}
	return whitelist, nil
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
		relPath, err := filepath.Rel(root, path)
		if err != nil {
			return err
		}
		files = append(files, relPath)
		return nil
	})
	return files, err
}

// Files returns a list of all files rooted at root
func Files(root string) ([]string, error) {
	var files []string
	logrus.Debugf("Getting files and contents at root %s", root)
	err := filepath.Walk(root, func(path string, info os.FileInfo, err error) error {
		files = append(files, path)
		return err
	})
	return files, err
}

// FilepathExists returns true if the path exists
func FilepathExists(path string) bool {
	_, err := os.Lstat(path)
	return !os.IsNotExist(err)
}

// CreateFile creates a file at path and copies over contents from the reader
func CreateFile(path string, reader io.Reader, perm os.FileMode) error {
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
	return dest.Chmod(perm)
}

// AddPathToVolumeWhitelist adds the given path to the volume whitelist
// It will get snapshotted when the VOLUME command is run then ignored
// for subsequent commands.
func AddPathToVolumeWhitelist(path string) error {
	logrus.Infof("adding %s to volume whitelist", path)
	volumeWhitelist = append(volumeWhitelist, path)
	return nil
}

// MoveVolumeWhitelistToWhitelist copies over all directories that were volume mounted
// in this step to be whitelisted for all subsequent docker commands.
func MoveVolumeWhitelistToWhitelist() error {
	if len(volumeWhitelist) > 0 {
		whitelist = append(whitelist, volumeWhitelist...)
		volumeWhitelist = []string{}
	}
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
	if err := CreateFile(dest, resp.Body, 0600); err != nil {
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

// CopyDir copies the file or directory at src to dest
func CopyDir(src, dest string) error {
	files, err := RelativeFiles("", src)
	if err != nil {
		return err
	}
	for _, file := range files {
		fullPath := filepath.Join(src, file)
		fi, err := os.Stat(fullPath)
		if err != nil {
			return err
		}
		destPath := filepath.Join(dest, file)
		if fi.IsDir() {
			logrus.Infof("Creating directory %s", destPath)
			if err := os.MkdirAll(destPath, fi.Mode()); err != nil {
				return err
			}
		} else if fi.Mode()&os.ModeSymlink != 0 {
			// If file is a symlink, we want to create the same relative symlink
			if err := CopySymlink(fullPath, destPath); err != nil {
				return err
			}
		} else {
			// ... Else, we want to copy over a file
			if err := CopyFile(fullPath, destPath); err != nil {
				return err
			}
		}
	}
	return nil
}

// CopySymlink copies the symlink at src to dest
func CopySymlink(src, dest string) error {
	link, err := os.Readlink(src)
	if err != nil {
		return err
	}
	linkDst := filepath.Join(dest, link)
	return os.Symlink(linkDst, dest)
}

// CopyFile copies the file at src to dest
func CopyFile(src, dest string) error {
	fi, err := os.Stat(src)
	if err != nil {
		return err
	}
	logrus.Infof("Copying file %s to %s", src, dest)
	srcFile, err := os.Open(src)
	if err != nil {
		return err
	}
	defer srcFile.Close()
	return CreateFile(dest, srcFile, fi.Mode())
}

// HasFilepathPrefix checks if the given file path begins with prefix
func HasFilepathPrefix(path, prefix string) bool {
	path = filepath.Clean(path)
	prefix = filepath.Clean(prefix)
	pathArray := strings.Split(path, "/")
	prefixArray := strings.Split(prefix, "/")

	if len(pathArray) < len(prefixArray) {
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
