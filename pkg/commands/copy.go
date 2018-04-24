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
	"github.com/GoogleContainerTools/kaniko/pkg/constants"
	"github.com/GoogleContainerTools/kaniko/pkg/util"
	"github.com/containers/image/manifest"
	"github.com/docker/docker/builder/dockerfile/instructions"
	"github.com/sirupsen/logrus"
	"os"
	"path/filepath"
	"strings"
)

type CopyCommand struct {
	cmd           *instructions.CopyCommand
	buildcontext  string
	snapshotFiles []string
}

func (c *CopyCommand) ExecuteCommand(config *manifest.Schema2Config) error {
	srcs := c.cmd.SourcesAndDest[:len(c.cmd.SourcesAndDest)-1]
	dest := c.cmd.SourcesAndDest[len(c.cmd.SourcesAndDest)-1]

	logrus.Infof("cmd: copy %s", srcs)
	logrus.Infof("dest: %s", dest)

	// Resolve from
	if c.cmd.From != "" {
		c.buildcontext = filepath.Join(constants.BuildContextDir, c.cmd.From)
	}

	// First, resolve any environment replacement
	resolvedEnvs, err := util.ResolveEnvironmentReplacementList(c.cmd.SourcesAndDest, config.Env, true)
	if err != nil {
		return err
	}
	dest = resolvedEnvs[len(resolvedEnvs)-1]
	// Resolve wildcards and get a list of resolved sources
	srcs, err = util.ResolveSources(resolvedEnvs, c.buildcontext)
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
		if fi.IsDir() {
			if !filepath.IsAbs(dest) {
				dest = filepath.Join(cwd, dest)
			}
			if err := util.CopyDir(fullPath, dest); err != nil {
				return err
			}
			copiedFiles, err := util.Files(dest)
			if err != nil {
				return err
			}
			c.snapshotFiles = append(c.snapshotFiles, copiedFiles...)
		} else if fi.Mode()&os.ModeSymlink != 0 {
			// If file is a symlink, we want to create the same relative symlink
			if err := util.CopySymlink(fullPath, destPath); err != nil {
				return err
			}
			c.snapshotFiles = append(c.snapshotFiles, destPath)
		} else {
			// ... Else, we want to copy over a file
			if err := util.CopyFile(fullPath, destPath); err != nil {
				return err
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

// CreatedBy returns some information about the command for the image config
func (c *CopyCommand) CreatedBy() string {
	return strings.Join(c.cmd.SourcesAndDest, " ")
}
