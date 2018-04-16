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
	"bufio"
	pkgutil "github.com/GoogleCloudPlatform/container-diff/pkg/util"
	"github.com/GoogleCloudPlatform/kaniko/pkg/constants"
	"github.com/containers/image/docker"
	"github.com/sirupsen/logrus"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"
)

var whitelist = []string{"/kaniko"}
var volumeWhitelist = []string{}

// ExtractFileSystemFromImage pulls an image and unpacks it to a file system at root
func ExtractFileSystemFromImage(img string) error {
	whitelist, err := fileSystemWhitelist(constants.WhitelistPath)
	if err != nil {
		return err
	}
	logrus.Infof("Whitelisted directories are %s", whitelist)
	if img == constants.NoBaseImage {
		logrus.Info("No base image, nothing to extract")
		return nil
	}
	ref, err := docker.ParseReference("//" + img)
	if err != nil {
		return err
	}
	imgSrc, err := ref.NewImageSource(nil)
	if err != nil {
		return err
	}
	return pkgutil.GetFileSystemFromReference(ref, imgSrc, constants.RootDir, whitelist)
}

// PathInWhitelist returns true if the path is whitelisted
func PathInWhitelist(path, directory string) bool {
	if path == constants.KanikoExecutor || path == constants.KanikoCerts {
		return false
	}
	for _, d := range whitelist {
		dirPath := filepath.Join(directory, d)
		if pkgutil.HasFilepathPrefix(path, dirPath) {
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
