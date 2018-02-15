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

package commands

import (
	"fmt"
	pkgutil "github.com/GoogleCloudPlatform/container-diff/pkg/util"
	"github.com/GoogleCloudPlatform/k8s-container-builder/contexts/dest"
	"github.com/GoogleCloudPlatform/k8s-container-builder/pkg/util"
	"github.com/docker/docker/builder/dockerfile/instructions"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
	"path/filepath"
	"strings"
)

type CopyCommand struct {
	cmd     *instructions.CopyCommand
	context dest.Context
}

func (c CopyCommand) ExecuteCommand() error {
	srcs := c.cmd.SourcesAndDest[:len(c.cmd.SourcesAndDest)-1]
	dest := c.cmd.SourcesAndDest[len(c.cmd.SourcesAndDest)-1]

	fmt.Println("cmd: copy", srcs)
	fmt.Println("dest: ", dest)

	if containsWildcards(srcs) {
		return c.executeWithWildcards()
	}
	// If there are multiple sources, the destination must be a directory
	if len(srcs) > 1 {
		if !isDir(dest) {
			return errors.Errorf("When specifying multiple sources in a COPY command, destination must be a directory and end in '/'")
		}
	}
	// Go through each src, and copy over the files into dest
	for _, src := range srcs {
		src = filepath.Clean(src)
		files, err := c.context.GetFilesFromSource(src)
		if err != nil {
			return err
		}
		for file, contents := range files {
			if !isDir(dest) {
				logrus.Infof("Copying from %s to %s", file, dest)
				return util.CreateFile(dest, contents)
			}
			relPath, err := filepath.Rel(src, file)
			if err != nil {
				return err
			}
			destPath := filepath.Join(dest, relPath)
			logrus.Infof("Copying from %s to %s", file, dest)
			err = util.CreateFile(destPath, contents)
			if err != nil {
				return err
			}
		}
	}
	return nil
}

func (c CopyCommand) executeWithWildcards() error {
	srcs := c.cmd.SourcesAndDest[:len(c.cmd.SourcesAndDest)-1]
	dest := c.cmd.SourcesAndDest[len(c.cmd.SourcesAndDest)-1]

	// Get all files from the source, since each needs to be matched against the sources
	files, err := c.context.GetFilesFromSource("")
	if err != nil {
		return err
	}
	matchedFiles, err := getMatchedFiles(srcs, files)
	if err != nil {
		return err
	}
	// If destination is a directory, copy all the matched files
	// for each source into it
	if isDir(dest) {
		for src, srcFiles := range matchedFiles {
			for _, file := range srcFiles {
				// Calculate relative path between src and the file
				relPath, err := filepath.Rel(src, file)
				if err != nil {
					return err
				}
				// Join destination and relative path to create final path for the file
				destPath := filepath.Join(dest, relPath)
				logrus.Infof("Copying %s into file %s", file, destPath)
				err = util.CreateFile(destPath, files[file])
				if err != nil {
					return err
				}
			}
		}
	} else {
		// If dest is not a directory, make sure only 1 file was matched
		totalFiles := 0
		for _, srcFiles := range matchedFiles {
			totalFiles += len(srcFiles)
		}
		if totalFiles > 1 {
			return errors.Errorf("When specifying multiple sources in a COPY command, destination must be a directory and end in '/'")
		}
		for _, srcFiles := range matchedFiles {
			for _, file := range srcFiles {
				logrus.Infof("Copying %s into file %s", file, dest)
				return util.CreateFile(dest, files[file])
			}
		}
	}
	return nil
}

func isDir(path string) bool {
	return strings.HasSuffix(path, "/")
}

func containsWildcards(paths []string) bool {
	for _, path := range paths {
		for i := 0; i < len(path); i++ {
			ch := path[i]
			// These are the wildcards that correspond to filepath.Match
			if ch == '*' || ch == '?' || ch == '[' {
				return true
			}
		}
	}
	return false
}

func getMatchedFiles(srcs []string, files map[string][]byte) (map[string][]string, error) {
	f := make(map[string][]string)
	for _, src := range srcs {
		src = filepath.Clean(src)
		matchedFiles := []string{}
		for file := range files {
			matched, err := filepath.Match(src, file)
			keep := matched || pkgutil.HasFilepathPrefix(file, src)
			if err != nil {
				return nil, err
			}
			if !keep {
				continue
			}
			matchedFiles = append(matchedFiles, file)
		}
		f[src] = matchedFiles
	}
	return f, nil
}
