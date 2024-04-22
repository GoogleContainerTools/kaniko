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
	"io/fs"
	"path/filepath"

	v1 "github.com/google/go-containerregistry/pkg/v1"
	"github.com/moby/buildkit/frontend/dockerfile/instructions"
	"github.com/pkg/errors"

	"github.com/GoogleContainerTools/kaniko/pkg/dockerfile"

	"github.com/GoogleContainerTools/kaniko/pkg/util"
	"github.com/sirupsen/logrus"
)

type AddCommand struct {
	BaseCommand
	cmd           *instructions.AddCommand
	fileContext   util.FileContext
	snapshotFiles []string
}

// ExecuteCommand executes the ADD command
// Special stuff about ADD:
//  1. If <src> is a remote file URL:
//     - destination will have permissions of 0600
//     - If remote file has HTTP Last-Modified header, we set the mtime of the file to that timestamp
//     - If dest doesn't end with a slash, the filepath is inferred to be <dest>/<filename>
//  2. If <src> is a local tar archive:
//     - it is unpacked at the dest, as 'tar -x' would
func (a *AddCommand) ExecuteCommand(config *v1.Config, buildArgs *dockerfile.BuildArgs) error {
	replacementEnvs := buildArgs.ReplacementEnvs(config.Env)

	chmod, useDefaultChmod, err := util.GetChmod(a.cmd.Chmod, replacementEnvs)
	if err != nil {
		return errors.Wrap(err, "getting permissions from chmod")
	}
	if useDefaultChmod {
		chmod = fs.FileMode(0o600)
	}

	uid, gid, err := util.GetUserGroup(a.cmd.Chown, replacementEnvs)
	if err != nil {
		return errors.Wrap(err, "getting user group from chown")
	}

	srcs, dest, err := util.ResolveEnvAndWildcards(a.cmd.SourcesAndDest, a.fileContext, replacementEnvs)
	if err != nil {
		return err
	}

	var unresolvedSrcs []string
	// If any of the sources are local tar archives:
	// 	1. Unpack them to the specified destination
	// If any of the sources is a remote file URL:
	//	1. Download and copy it to the specified dest
	// Else, add to the list of unresolved sources
	for _, src := range srcs {
		fullPath := filepath.Join(a.fileContext.Root, src)
		if util.IsSrcRemoteFileURL(src) {
			urlDest, err := util.URLDestinationFilepath(src, dest, config.WorkingDir, replacementEnvs)
			if err != nil {
				return err
			}
			logrus.Infof("Adding remote URL %s to %s", src, urlDest)
			if err := util.DownloadFileToDest(src, urlDest, uid, gid, chmod); err != nil {
				return errors.Wrap(err, "downloading remote source file")
			}
			a.snapshotFiles = append(a.snapshotFiles, urlDest)
		} else if util.IsFileLocalTarArchive(fullPath) {
			tarDest, err := util.DestinationFilepath("", dest, config.WorkingDir)
			if err != nil {
				return errors.Wrap(err, "determining dest for tar")
			}
			logrus.Infof("Unpacking local tar archive %s to %s", src, tarDest)
			extractedFiles, err := util.UnpackLocalTarArchive(fullPath, tarDest)
			if err != nil {
				return errors.Wrap(err, "unpacking local tar")
			}
			logrus.Debugf("Added %v from local tar archive %s", extractedFiles, src)
			a.snapshotFiles = append(a.snapshotFiles, extractedFiles...)
		} else {
			unresolvedSrcs = append(unresolvedSrcs, src)
		}
	}
	// With the remaining "normal" sources, create and execute a standard copy command
	if len(unresolvedSrcs) == 0 {
		return nil
	}

	copyCmd := CopyCommand{
		cmd: &instructions.CopyCommand{
			SourcesAndDest: instructions.SourcesAndDest{SourcePaths: unresolvedSrcs, DestPath: dest},
			Chown:          a.cmd.Chown,
			Chmod:          a.cmd.Chmod,
		},
		fileContext: a.fileContext,
	}

	if err := copyCmd.ExecuteCommand(config, buildArgs); err != nil {
		return errors.Wrap(err, "executing copy command")
	}
	a.snapshotFiles = append(a.snapshotFiles, copyCmd.snapshotFiles...)
	return nil
}

// FilesToSnapshot should return an empty array if still nil; no files were changed
func (a *AddCommand) FilesToSnapshot() []string {
	return a.snapshotFiles
}

// String returns some information about the command for the image config
func (a *AddCommand) String() string {
	return a.cmd.String()
}

func (a *AddCommand) FilesUsedFromContext(config *v1.Config, buildArgs *dockerfile.BuildArgs) ([]string, error) {
	replacementEnvs := buildArgs.ReplacementEnvs(config.Env)

	srcs, _, err := util.ResolveEnvAndWildcards(a.cmd.SourcesAndDest, a.fileContext, replacementEnvs)
	if err != nil {
		return nil, err
	}

	files := []string{}
	for _, src := range srcs {
		if util.IsSrcRemoteFileURL(src) {
			continue
		}
		if util.IsFileLocalTarArchive(src) {
			continue
		}
		fullPath := filepath.Join(a.fileContext.Root, src)
		files = append(files, fullPath)
	}

	logrus.Infof("Using files from context: %v", files)
	return files, nil
}

func (a *AddCommand) MetadataOnly() bool {
	return false
}

func (a *AddCommand) RequiresUnpackedFS() bool {
	return true
}
