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
	"os"
	"path/filepath"
	"strings"

	kConfig "github.com/GoogleContainerTools/kaniko/pkg/config"
	"github.com/moby/buildkit/frontend/dockerfile/instructions"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"

	"github.com/GoogleContainerTools/kaniko/pkg/dockerfile"
	"github.com/GoogleContainerTools/kaniko/pkg/util"
	v1 "github.com/google/go-containerregistry/pkg/v1"
)

// for testing
var (
	getUserGroup = util.GetUserGroup
)

type CopyCommand struct {
	BaseCommand
	cmd           *instructions.CopyCommand
	fileContext   util.FileContext
	snapshotFiles []string
}

func (c *CopyCommand) ExecuteCommand(config *v1.Config, buildArgs *dockerfile.BuildArgs) error {
	// Resolve from
	if c.cmd.From != "" {
		c.fileContext = util.FileContext{Root: filepath.Join(kConfig.KanikoDir, c.cmd.From)}
	}

	replacementEnvs := buildArgs.ReplacementEnvs(config.Env)
	uid, gid, err := getUserGroup(c.cmd.Chown, replacementEnvs)
	if err != nil {
		return errors.Wrap(err, "getting user group from chown")
	}

	srcs, dest, err := util.ResolveEnvAndWildcards(c.cmd.SourcesAndDest, c.fileContext, replacementEnvs)
	if err != nil {
		return errors.Wrap(err, "resolving src")
	}

	// For each source, iterate through and copy it over
	for _, src := range srcs {
		fullPath := filepath.Join(c.fileContext.Root, src)

		fi, err := os.Lstat(fullPath)
		if err != nil {
			return errors.Wrap(err, "could not copy source")
		}
		if fi.IsDir() && !strings.HasSuffix(fullPath, string(os.PathSeparator)) {
			fullPath += "/"
		}
		cwd := config.WorkingDir
		if cwd == "" {
			cwd = kConfig.RootDir
		}

		destPath, err := util.DestinationFilepath(fullPath, dest, cwd)
		if err != nil {
			return errors.Wrap(err, "find destination path")
		}

		// If the destination dir is a symlink we need to resolve the path and use
		// that instead of the symlink path
		destPath, err = resolveIfSymlink(destPath)
		if err != nil {
			return errors.Wrap(err, "resolving dest symlink")
		}

		if fi.IsDir() {
			copiedFiles, err := util.CopyDir(fullPath, destPath, c.fileContext, uid, gid)
			if err != nil {
				return errors.Wrap(err, "copying dir")
			}
			c.snapshotFiles = append(c.snapshotFiles, copiedFiles...)
		} else if util.IsSymlink(fi) {
			// If file is a symlink, we want to copy the target file to destPath
			exclude, err := util.CopySymlink(fullPath, destPath, c.fileContext)
			if err != nil {
				return errors.Wrap(err, "copying symlink")
			}
			if exclude {
				continue
			}
			c.snapshotFiles = append(c.snapshotFiles, destPath)
		} else {
			// ... Else, we want to copy over a file
			exclude, err := util.CopyFile(fullPath, destPath, c.fileContext, uid, gid)
			if err != nil {
				return errors.Wrap(err, "copying file")
			}
			if exclude {
				continue
			}
			c.snapshotFiles = append(c.snapshotFiles, destPath)
		}
	}
	return nil
}

// FilesToSnapshot should return an empty array if still nil; no files were changed
func (c *CopyCommand) FilesToSnapshot() []string {
	return c.snapshotFiles
}

// String returns some information about the command for the image config
func (c *CopyCommand) String() string {
	return c.cmd.String()
}

func (c *CopyCommand) FilesUsedFromContext(config *v1.Config, buildArgs *dockerfile.BuildArgs) ([]string, error) {
	return copyCmdFilesUsedFromContext(config, buildArgs, c.cmd, c.fileContext)
}

func (c *CopyCommand) MetadataOnly() bool {
	return false
}

func (c *CopyCommand) RequiresUnpackedFS() bool {
	return true
}

func (c *CopyCommand) From() string {
	return c.cmd.From
}

func resolveIfSymlink(destPath string) (string, error) {
	if !filepath.IsAbs(destPath) {
		return "", errors.New("dest path must be abs")
	}

	var nonexistentPaths []string

	newPath := destPath
	for newPath != "/" {
		_, err := os.Lstat(newPath)
		if err != nil {
			if os.IsNotExist(err) {
				dir, file := filepath.Split(newPath)
				newPath = filepath.Clean(dir)
				nonexistentPaths = append(nonexistentPaths, file)
				continue
			} else {
				return "", errors.Wrap(err, "failed to lstat")
			}
		}

		newPath, err = filepath.EvalSymlinks(newPath)
		if err != nil {
			return "", errors.Wrap(err, "failed to eval symlinks")
		}
		break
	}

	for i := len(nonexistentPaths) - 1; i >= 0; i-- {
		newPath = filepath.Join(newPath, nonexistentPaths[i])
	}

	if destPath != newPath {
		logrus.Tracef("Updating destination path from %v to %v due to symlink", destPath, newPath)
	}

	return filepath.Clean(newPath), nil
}

func copyCmdFilesUsedFromContext(
	config *v1.Config, buildArgs *dockerfile.BuildArgs, cmd *instructions.CopyCommand,
	fileContext util.FileContext,
) ([]string, error) {
	// We don't use the context if we're performing a copy --from.
	if cmd.From != "" {
		return nil, nil
	}

	replacementEnvs := buildArgs.ReplacementEnvs(config.Env)

	srcs, _, err := util.ResolveEnvAndWildcards(
		cmd.SourcesAndDest, fileContext, replacementEnvs,
	)
	if err != nil {
		return nil, err
	}

	files := []string{}
	for _, src := range srcs {
		fullPath := filepath.Join(fileContext.Root, src)
		files = append(files, fullPath)
	}

	logrus.Debugf("Using files from context: %v", files)

	return files, nil
}
