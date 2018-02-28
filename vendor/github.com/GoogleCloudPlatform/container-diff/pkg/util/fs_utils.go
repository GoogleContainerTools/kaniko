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
	"bytes"
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"

	"github.com/sirupsen/logrus"
)

// Directory stores a representation of a file directory.
type Directory struct {
	Root    string
	Content []string
}

type DirectoryEntry struct {
	Name string
	Size int64
}

func GetSize(path string) int64 {
	stat, err := os.Stat(path)
	if err != nil {
		logrus.Errorf("Could not obtain size for %s: %s", path, err)
		return -1
	}
	if stat.IsDir() {
		size, err := getDirectorySize(path)
		if err != nil {
			logrus.Errorf("Could not obtain directory size for %s: %s", path, err)
		}
		return size
	}
	return stat.Size()
}

//GetFileContents returns the contents of a file at the specified path
func GetFileContents(path string) (*string, error) {
	if _, err := os.Stat(path); os.IsNotExist(err) {
		return nil, err
	}

	contents, err := ioutil.ReadFile(path)
	if err != nil {
		return nil, err
	}

	strContents := string(contents)
	//If file is empty, return nil
	if strContents == "" {
		return nil, nil
	}
	return &strContents, nil
}

func getDirectorySize(path string) (int64, error) {
	var size int64
	err := filepath.Walk(path, func(_ string, info os.FileInfo, err error) error {
		if !info.IsDir() {
			size += info.Size()
		}
		return err
	})
	return size, err
}

// GetDirectoryContents converts the directory starting at the provided path into a Directory struct.
func GetDirectory(path string, deep bool) (Directory, error) {
	var directory Directory
	directory.Root = path
	var err error
	if deep {
		walkFn := func(currPath string, info os.FileInfo, err error) error {
			newContent := strings.TrimPrefix(currPath, directory.Root)
			if newContent != "" {
				directory.Content = append(directory.Content, newContent)
			}
			return nil
		}

		err = filepath.Walk(path, walkFn)
	} else {
		contents, err := ioutil.ReadDir(path)
		if err != nil {
			return directory, err
		}

		for _, file := range contents {
			fileName := "/" + file.Name()
			directory.Content = append(directory.Content, fileName)
		}
	}
	return directory, err
}

func GetDirectoryEntries(d Directory) []DirectoryEntry {
	return CreateDirectoryEntries(d.Root, d.Content)
}

func CreateDirectoryEntries(root string, entryNames []string) (entries []DirectoryEntry) {
	for _, name := range entryNames {
		entryPath := filepath.Join(root, name)
		size := GetSize(entryPath)

		entry := DirectoryEntry{
			Name: name,
			Size: size,
		}
		entries = append(entries, entry)
	}
	return entries
}

func CheckSameFile(f1name, f2name string) (bool, error) {
	// Check first if files differ in size and immediately return
	f1stat, err := os.Stat(f1name)
	if err != nil {
		return false, err
	}
	f2stat, err := os.Stat(f2name)
	if err != nil {
		return false, err
	}

	if f1stat.Size() != f2stat.Size() {
		return false, nil
	}

	// Next, check file contents
	f1, err := ioutil.ReadFile(f1name)
	if err != nil {
		return false, err
	}
	f2, err := ioutil.ReadFile(f2name)
	if err != nil {
		return false, err
	}

	if !bytes.Equal(f1, f2) {
		return false, nil
	}
	return true, nil
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
