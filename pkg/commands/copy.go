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
	"strconv"

	"github.com/moby/buildkit/frontend/dockerfile/instructions"
	"github.com/sirupsen/logrus"

	"github.com/GoogleContainerTools/kaniko/pkg/constants"

	"github.com/GoogleContainerTools/kaniko/pkg/dockerfile"
	"github.com/GoogleContainerTools/kaniko/pkg/util"
	"github.com/google/go-containerregistry/pkg/v1"
)

type CopyCommand struct {
	BaseCommand
	cmd           *instructions.CopyCommand
	buildcontext  string
	snapshotFiles []string
}

func (c *CopyCommand) RequiresUnpackedFS() bool {
    // We want to unpack the root filesystem if the copy command is going
    // to require resolving UIDs or GIDs that might have been defined
    // in the base image
    return c.cmd.Chown != ""
}

func (c *CopyCommand) ExecuteCommand(config *v1.Config, buildArgs *dockerfile.BuildArgs) error {
	// Resolve from
	if c.cmd.From != "" {
		c.buildcontext = filepath.Join(constants.KanikoDir, c.cmd.From)
	}

	replacementEnvs := buildArgs.ReplacementEnvs(config.Env)

	srcs, dest, err := resolveEnvAndWildcards(c.cmd.SourcesAndDest, c.buildcontext, replacementEnvs)
	if err != nil {
		return err
	}

	// Default to not setting uid or gid
	uid, gid := -1, -1
	if c.cmd.Chown != "" {
		// Resolve the chown string to a uid:gid format
		uidStr, gidStr, err := util.GetUIDGidFromUserString(c.cmd.Chown, replacementEnvs)
		if err != nil {
			return err
		}

		// Determine uid if uidstr was set
		if uidStr != "" {
			uid64, err := strconv.ParseUint(uidStr, 10, 32)
			if err != nil {
				return err
			}
			uid = int(uid64)
		}

		// Determine gid if gidstr was set
		if gidStr != "" {
			gid64, err := strconv.ParseUint(gidStr, 10, 32)
			if err != nil {
				return err
			}
			gid = int(gid64)
		}
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
				// we need to add '/' to the end to indicate the destination is a directory
				dest = filepath.Join(cwd, dest) + "/"
			}
			copiedFiles, err := util.CopyDir(fullPath, dest, c.buildcontext, uid, gid)
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
			exclude, err := util.CopyFile(fullPath, destPath, c.buildcontext, uid, gid)
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

func resolveEnvAndWildcards(sd instructions.SourcesAndDest, buildcontext string, envs []string) ([]string, string, error) {
	// First, resolve any environment replacement
	resolvedEnvs, err := util.ResolveEnvironmentReplacementList(sd, envs, true)
	if err != nil {
		return nil, "", err
	}
	dest := resolvedEnvs[len(resolvedEnvs)-1]
	// Resolve wildcards and get a list of resolved sources
	srcs, err := util.ResolveSources(resolvedEnvs, buildcontext)
	return srcs, dest, err
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
	// We don't use the context if we're performing a copy --from.
	if c.cmd.From != "" {
		return nil, nil
	}

	replacementEnvs := buildArgs.ReplacementEnvs(config.Env)
	srcs, _, err := resolveEnvAndWildcards(c.cmd.SourcesAndDest, c.buildcontext, replacementEnvs)
	if err != nil {
		return nil, err
	}

	files := []string{}
	for _, src := range srcs {
		fullPath := filepath.Join(c.buildcontext, src)
		files = append(files, fullPath)
	}
	logrus.Infof("Using files from context: %v", files)
	return files, nil
}

func (c *CopyCommand) MetadataOnly() bool {
	return false
}
