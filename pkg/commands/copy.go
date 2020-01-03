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
	"github.com/GoogleContainerTools/kaniko/pkg/constants"
	"github.com/moby/buildkit/frontend/dockerfile/instructions"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
	"os"
	"path/filepath"
	"strings"

	"github.com/GoogleContainerTools/kaniko/pkg/dockerfile"
	"github.com/GoogleContainerTools/kaniko/pkg/util"
	v1 "github.com/google/go-containerregistry/pkg/v1"
)

type CopyCommand struct {
	BaseCommand
	cmd           *instructions.CopyCommand
	buildcontext  string
	snapshotFiles []string
}

func (c *CopyCommand) ExecuteCommand(config *v1.Config, buildArgs *dockerfile.BuildArgs) error {
	// Resolve from
	if c.cmd.From != "" {
		c.buildcontext = filepath.Join(constants.KanikoDir, c.cmd.From)
	}

	replacementEnvs := buildArgs.ReplacementEnvs(config.Env)

	srcs, dest, err := util.ResolveEnvAndWildcards(c.cmd.SourcesAndDest, c.buildcontext, replacementEnvs)
	if err != nil {
		return err
	}

	// For each source, iterate through and copy it over
	for _, src := range srcs {
		fullPath := filepath.Join(c.buildcontext, src)
		fi, err := os.Lstat(fullPath)
		if err != nil {
			return err
		}
		cwd := config.WorkingDir
		if cwd == "" {
			cwd = constants.RootDir
		}

		destPath, err := util.DestinationFilepath(src, dest, cwd)
		if err != nil {
			return err
		}

		// If the destination dir is a symlink we need to resolve the path and use
		// that instead of the symlink path
		destPath, err = resolveIfSymlink(destPath)
		if err != nil {
			return err
		}

		if fi.IsDir() {
			copiedFiles, err := util.CopyDir(fullPath, destPath, c.buildcontext)
			if err != nil {
				return err
			}
			c.snapshotFiles = append(c.snapshotFiles, copiedFiles...)
		} else if fi.Mode()&os.ModeSymlink != 0 {
			// If file is a symlink, we want to create the same relative symlink
			exclude, err := util.CopySymlink(fullPath, destPath, c.buildcontext)
			if err != nil {
				return err
			}
			if exclude {
				continue
			}
			c.snapshotFiles = append(c.snapshotFiles, destPath)
		} else {
			// ... Else, we want to copy over a file
			exclude, err := util.CopyFile(fullPath, destPath, c.buildcontext)
			if err != nil {
				return err
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
	return copyCmdFilesUsedFromContext(config, buildArgs, c.cmd, c.buildcontext)
}

func (c *CopyCommand) MetadataOnly() bool {
	return false
}

func (c *CopyCommand) RequiresUnpackedFS() bool {
	return true
}

func (c *CopyCommand) ShouldCacheOutput() bool {
	return true
}

// CacheCommand returns true since this command should be cached
func (c *CopyCommand) CacheCommand(img v1.Image) DockerCommand {

	return &CachingCopyCommand{
		img:          img,
		cmd:          c.cmd,
		buildcontext: c.buildcontext,
		extractFn:    util.ExtractFile,
	}
}

func (c *CopyCommand) From() string {
	return c.cmd.From
}

type CachingCopyCommand struct {
	BaseCommand
	img            v1.Image
	extractedFiles []string
	cmd            *instructions.CopyCommand
	buildcontext   string
	extractFn      util.ExtractFunction
}

func (cr *CachingCopyCommand) ExecuteCommand(config *v1.Config, buildArgs *dockerfile.BuildArgs) error {
	logrus.Infof("Found cached layer, extracting to filesystem")
	var err error

	if cr.img == nil {
		return errors.New(fmt.Sprintf("cached command image is nil %v", cr.String()))
	}
	cr.extractedFiles, err = util.GetFSFromImage(RootDir, cr.img, cr.extractFn)

	logrus.Infof("extractedFiles: %s", cr.extractedFiles)
	if err != nil {
		return errors.Wrap(err, "extracting fs from image")
	}

	return nil
}

func (cr *CachingCopyCommand) FilesUsedFromContext(config *v1.Config, buildArgs *dockerfile.BuildArgs) ([]string, error) {
	return copyCmdFilesUsedFromContext(config, buildArgs, cr.cmd, cr.buildcontext)
}

func (cr *CachingCopyCommand) FilesToSnapshot() []string {
	return cr.extractedFiles
}

func (cr *CachingCopyCommand) String() string {
	if cr.cmd == nil {
		return "nil command"
	}
	return cr.cmd.String()
}

func (cr *CachingCopyCommand) From() string {
	return cr.cmd.From
}

func resolveIfSymlink(destPath string) (string, error) {
	if !filepath.IsAbs(destPath) {
		return "", errors.New("dest path must be abs")
	}

	pathPrefix := "/"
	pathTokens := strings.Split(destPath, string(os.PathSeparator))
	for i, pathToken := range pathTokens {
		currentPath := filepath.Join(pathPrefix, pathToken)
		info, err := os.Lstat(currentPath)
		if err != nil {
			if os.IsNotExist(err) {
				pathPrefix = filepath.Join(append([]string{currentPath}, pathTokens[i+1:]...)...)
				break
			} else {
				return "", errors.Wrap(err, "failed to lstat")
			}
		}

		switch mode := info.Mode(); {
		case mode&os.ModeSymlink != 0:
			linkPath, err := os.Readlink(currentPath)
			if err != nil {
				return "", errors.Wrap(err, "failed to read symlink")
			}
			if filepath.IsAbs(linkPath) {
				pathPrefix = linkPath
			} else {
				pathPrefix = filepath.Join(pathPrefix, linkPath)
			}
		default:
			pathPrefix = filepath.Join(pathPrefix, pathToken)
		}
	}
	if destPath != pathPrefix {
		logrus.Tracef("Updating destination path from %v to %v due to symlink", destPath, pathPrefix)
	}
	return filepath.Clean(pathPrefix), nil
}

func copyCmdFilesUsedFromContext(
	config *v1.Config, buildArgs *dockerfile.BuildArgs, cmd *instructions.CopyCommand,
	buildcontext string,
) ([]string, error) {
	// We don't use the context if we're performing a copy --from.
	if cmd.From != "" {
		return nil, nil
	}

	replacementEnvs := buildArgs.ReplacementEnvs(config.Env)

	srcs, _, err := util.ResolveEnvAndWildcards(
		cmd.SourcesAndDest, buildcontext, replacementEnvs,
	)
	if err != nil {
		return nil, err
	}

	files := []string{}
	for _, src := range srcs {
		fullPath := filepath.Join(buildcontext, src)
		files = append(files, fullPath)
	}

	logrus.Debugf("Using files from context: %v", files)

	return files, nil
}
