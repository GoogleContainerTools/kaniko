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
	"github.com/GoogleCloudPlatform/k8s-container-builder/pkg/util"
	"github.com/containers/image/manifest"
	"github.com/docker/docker/builder/dockerfile/instructions"
	"github.com/sirupsen/logrus"
	"path/filepath"
	"strings"
)

type AddCommand struct {
	cmd           *instructions.AddCommand
	buildcontext  string
	snapshotFiles []string
}

// ExecuteCommand executes the ADD command
// Special stuff about ADD:
// 	1. If <src> is a remote file URL:
// 		- destination will have permissions of 0600
// 		- If remote file has HTTP Last-Modified header, we set the mtime of the file to that timestamp
// 		- If dest doesn't end with a slash, the filepath is inferred to be <dest>/<filename>
// 	2. If <src> is a local tar archive:
// 		-If <src> is a local tar archive, it is unpacked at the dest, as 'tar -x' would
func (a *AddCommand) ExecuteCommand(config *manifest.Schema2Config) error {
	srcs := a.cmd.SourcesAndDest[:len(a.cmd.SourcesAndDest)-1]
	dest := a.cmd.SourcesAndDest[len(a.cmd.SourcesAndDest)-1]

	logrus.Infof("cmd: Add %s", srcs)
	logrus.Infof("dest: %s", dest)

	// First, resolve any environment replacement
	resolvedEnvs, err := util.ResolveEnvironmentReplacementList(a.cmd.SourcesAndDest, config.Env, true)
	if err != nil {
		return err
	}
	dest = resolvedEnvs[len(resolvedEnvs)-1]
	// Get a map of [src]:[files rooted at src]
	srcMap, err := util.ResolveSources(resolvedEnvs, a.buildcontext)
	if err != nil {
		return err
	}
	// If any of the sources are local tar archives:
	// 	1. Unpack them to the specified destination
	// 	2. Remove it as a source that needs to be copied over
	// If any of the sources is a remote file URL:
	//	1. Download and copy it to the specifed dest
	//  2. Remove it as a source that needs to be copied
	for src, files := range srcMap {
		for _, file := range files {
			// If file is a local tar archive, then we unpack it to dest
			filePath := filepath.Join(a.buildcontext, file)
			isFilenameSource, err := isFilenameSource(srcMap, file)
			if err != nil {
				return err
			}
			if util.IsSrcRemoteFileURL(file) {
				urlDest := util.URLDestinationFilepath(file, dest, config.WorkingDir)
				logrus.Infof("Adding remote URL %s to %s", file, urlDest)
				if err := util.DownloadFileToDest(file, urlDest); err != nil {
					return err
				}
				a.snapshotFiles = append(a.snapshotFiles, urlDest)
				delete(srcMap, src)
			} else if isFilenameSource && util.IsFileLocalTarArchive(filePath) {
				logrus.Infof("Unpacking local tar archive %s to %s", file, dest)
				if err := util.UnpackLocalTarArchive(filePath, dest); err != nil {
					return err
				}
				// Add the unpacked files to the snapshotter
				filesAdded, err := util.Files(dest)
				if err != nil {
					return err
				}
				logrus.Debugf("Added %v from local tar archive %s", filesAdded, file)
				a.snapshotFiles = append(a.snapshotFiles, filesAdded...)
				delete(srcMap, src)
			}
		}
	}
	// With the remaining "normal" sources, create and execute a standard copy command
	if len(srcMap) == 0 {
		return nil
	}
	var regularSrcs []string
	for src := range srcMap {
		regularSrcs = append(regularSrcs, src)
	}
	copyCmd := CopyCommand{
		cmd: &instructions.CopyCommand{
			SourcesAndDest: append(regularSrcs, dest),
		},
		buildcontext: a.buildcontext,
	}
	if err := copyCmd.ExecuteCommand(config); err != nil {
		return err
	}
	a.snapshotFiles = append(a.snapshotFiles, copyCmd.snapshotFiles...)
	return nil
}

func isFilenameSource(srcMap map[string][]string, fileName string) (bool, error) {
	for src := range srcMap {
		matched, err := filepath.Match(src, fileName)
		if err != nil {
			return false, err
		}
		if matched || (src == fileName) {
			return true, nil
		}
	}
	return false, nil
}

// FilesToSnapshot should return an empty array if still nil; no files were changed
func (a *AddCommand) FilesToSnapshot() []string {
	return a.snapshotFiles
}

// CreatedBy returns some information about the command for the image config
func (a *AddCommand) CreatedBy() string {
	return strings.Join(a.cmd.SourcesAndDest, " ")
}
