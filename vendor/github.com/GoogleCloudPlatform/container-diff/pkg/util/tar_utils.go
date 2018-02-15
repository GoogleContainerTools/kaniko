/*
Copyright 2017 Google, Inc. All rights reserved.

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
	"github.com/sirupsen/logrus"
	"io"
	"os"
	"path/filepath"
	"strings"
)

func unpackTar(tr *tar.Reader, path string, whitelist []string) error {
	for {
		header, err := tr.Next()
		if err == io.EOF {
			// end of tar archive
			break
		}
		if err != nil {
			logrus.Error("Error getting next tar header")
			return err
		}
		if strings.Contains(header.Name, ".wh.") {
			rmPath := filepath.Join(path, header.Name)
			// Remove the .wh file if it was extracted.
			if _, err := os.Stat(rmPath); !os.IsNotExist(err) {
				if err := os.Remove(rmPath); err != nil {
					logrus.Error(err)
				}
			}

			// Remove the whited-out path.
			newName := strings.Replace(rmPath, ".wh.", "", 1)
			if err = os.RemoveAll(newName); err != nil {
				logrus.Error(err)
			}
			continue
		}
		target := filepath.Join(path, header.Name)
		// Make sure the target isn't part of the whitelist
		if checkWhitelist(target, whitelist) {
			continue
		}
		mode := header.FileInfo().Mode()
		switch header.Typeflag {

		// if its a dir and it doesn't exist create it
		case tar.TypeDir:
			if _, err := os.Stat(target); os.IsNotExist(err) {
				if err := os.MkdirAll(target, mode); err != nil {
					return err
				}
				// In some cases, MkdirAll doesn't change the permissions, so run Chmod
				if err := os.Chmod(target, mode); err != nil {
					return err
				}
			}

		// if it's a file create it
		case tar.TypeReg:
			// It's possible for a file to be included before the directory it's in is created.
			baseDir := filepath.Dir(target)
			if _, err := os.Stat(baseDir); os.IsNotExist(err) {
				logrus.Debugf("baseDir %s for file %s does not exist. Creating.", baseDir, target)
				if err := os.MkdirAll(baseDir, 0755); err != nil {
					return err
				}
			}
			// It's possible we end up creating files that can't be overwritten based on their permissions.
			// Explicitly delete an existing file before continuing.
			if _, err := os.Stat(target); !os.IsNotExist(err) {
				logrus.Debugf("Removing %s for overwrite.", target)
				if err := os.Remove(target); err != nil {
					return err
				}
			}

			currFile, err := os.Create(target)
			if err != nil {
				logrus.Errorf("Error creating file %s %s", target, err)
				return err
			}
			// manually set permissions on file, since the default umask (022) will interfere
			if err = os.Chmod(target, mode); err != nil {
				logrus.Errorf("Error updating file permissions on %s", target)
				return err
			}
			_, err = io.Copy(currFile, tr)
			if err != nil {
				return err
			}
			currFile.Close()
		case tar.TypeSymlink:
			// It's possible we end up creating files that can't be overwritten based on their permissions.
			// Explicitly delete an existing file before continuing.
			if _, err := os.Stat(target); !os.IsNotExist(err) {
				logrus.Debugf("Removing %s to create symlink.", target)
				if err := os.RemoveAll(target); err != nil {
					logrus.Debugf("Unable to remove %s: %s", target, err)
				}
			}

			if err = os.Symlink(header.Linkname, target); err != nil {
				logrus.Errorf("Failed to create symlink between %s and %s: %s", header.Linkname, target, err)
			}
		}
	}
	return nil
}

func checkWhitelist(target string, whitelist []string) bool {
	for _, w := range whitelist {
		if HasFilepathPrefix(target, w) {
			logrus.Debugf("Not extracting %s, as it has prefix %s which is whitelisted", target, w)
			return true
		}
	}
	return false
}

// UnTar takes in a path to a tar file and writes the untarred version to the provided target.
// Only untars one level, does not untar nested tars.
func UnTar(r io.Reader, target string, whitelist []string) error {
	if _, ok := os.Stat(target); ok != nil {
		os.MkdirAll(target, 0775)
	}

	tr := tar.NewReader(r)
	if err := unpackTar(tr, target, whitelist); err != nil {
		return err
	}
	return nil
}

func IsTar(path string) bool {
	return filepath.Ext(path) == ".tar" ||
		filepath.Ext(path) == ".tar.gz" ||
		filepath.Ext(path) == ".tgz"
}

func CheckTar(image string) bool {
	if strings.TrimSuffix(image, ".tar") == image {
		return false
	}
	if _, err := os.Stat(image); err != nil {
		logrus.Errorf("%s does not exist", image)
		return false
	}
	return true
}
