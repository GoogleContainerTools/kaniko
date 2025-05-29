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
	"os"
	"path/filepath"

	kConfig "github.com/GoogleContainerTools/kaniko/pkg/config"
	"github.com/GoogleContainerTools/kaniko/pkg/dockerfile"
	"github.com/pkg/errors"

	"github.com/GoogleContainerTools/kaniko/pkg/util"
	v1 "github.com/google/go-containerregistry/pkg/v1"
	"github.com/moby/buildkit/frontend/dockerfile/instructions"
	"github.com/sirupsen/logrus"
)

type WorkdirCommand struct {
	BaseCommand
	cmd           *instructions.WorkdirCommand
	snapshotFiles []string
	shdCache      bool
}

func ToAbsPath(path string, workdir string) string {
	if filepath.IsAbs(path) {
		return path
	} else {
		if workdir != "" {
			return filepath.Join(workdir, path)
		} else {
			return filepath.Join("/", path)
		}
	}
}

// For testing
var mkdirAllWithPermissions = util.MkdirAllWithPermissions

func (w *WorkdirCommand) ExecuteCommand(config *v1.Config, buildArgs *dockerfile.BuildArgs) error {
	logrus.Info("Cmd: workdir")
	workdirPath := w.cmd.Path
	replacementEnvs := buildArgs.ReplacementEnvs(config.Env)
	resolvedWorkingDir, err := util.ResolveEnvironmentReplacement(workdirPath, replacementEnvs, true)
	if err != nil {
		return err
	}
	config.WorkingDir = ToAbsPath(resolvedWorkingDir, config.WorkingDir)
	logrus.Infof("Changed working directory to %s", config.WorkingDir)

	// Only create and snapshot the dir if it didn't exist already
	w.snapshotFiles = []string{}
	if _, err := os.Stat(config.WorkingDir); os.IsNotExist(err) {
		uid, gid := int64(-1), int64(-1)

		if config.User != "" {
			logrus.Debugf("Fetching uid and gid for USER '%s'", config.User)
			uid, gid, err = util.GetUserGroup(config.User, replacementEnvs)
			if err != nil {
				return errors.Wrapf(err, "identifying uid and gid for user %s", config.User)
			}
		}

		logrus.Infof("Creating directory %s with uid %d and gid %d", config.WorkingDir, uid, gid)
		w.snapshotFiles = append(w.snapshotFiles, config.WorkingDir)
		if err := mkdirAllWithPermissions(config.WorkingDir, 0755, uid, gid); err != nil {
			return errors.Wrapf(err, "creating workdir %s", config.WorkingDir)
		}
	}
	return nil
}

// FilesToSnapshot returns the workingdir, which should have been created if it didn't already exist
func (w *WorkdirCommand) FilesToSnapshot() []string {
	return w.snapshotFiles
}

// String returns some information about the command for the image config history
func (w *WorkdirCommand) String() string {
	return w.cmd.String()
}

// CacheCommand returns true since this command should be cached
func (w *WorkdirCommand) CacheCommand(img v1.Image) DockerCommand {

	return &CachingWorkdirCommand{
		img:       img,
		cmd:       w.cmd,
		extractFn: util.ExtractFile,
	}
}

func (w *WorkdirCommand) MetadataOnly() bool {
	return false
}

func (r *WorkdirCommand) RequiresUnpackedFS() bool {
	return true
}

func (w *WorkdirCommand) ShouldCacheOutput() bool {
	return w.shdCache
}

type CachingWorkdirCommand struct {
	BaseCommand
	caching
	img            v1.Image
	extractedFiles []string
	cmd            *instructions.WorkdirCommand
	extractFn      util.ExtractFunction
}

func (wr *CachingWorkdirCommand) ExecuteCommand(config *v1.Config, buildArgs *dockerfile.BuildArgs) error {
	var err error
	logrus.Info("Cmd: workdir")
	workdirPath := wr.cmd.Path
	replacementEnvs := buildArgs.ReplacementEnvs(config.Env)
	resolvedWorkingDir, err := util.ResolveEnvironmentReplacement(workdirPath, replacementEnvs, true)
	if err != nil {
		return err
	}
	config.WorkingDir = ToAbsPath(resolvedWorkingDir, config.WorkingDir)
	logrus.Infof("Changed working directory to %s", config.WorkingDir)

	logrus.Infof("Found cached layer, extracting to filesystem")

	if wr.img == nil {
		return errors.New(fmt.Sprintf("command image is nil %v", wr.String()))
	}

	layers, err := wr.img.Layers()
	if err != nil {
		return errors.Wrap(err, "retrieving image layers")
	}

	if len(layers) > 1 {
		return errors.New(fmt.Sprintf("expected %d layers but got %d", 1, len(layers)))
	} else if len(layers) == 0 {
		// an empty image in cache indicates that no directory was created by WORKDIR
		return nil
	}

	wr.layer = layers[0]

	wr.extractedFiles, err = util.GetFSFromLayers(
		kConfig.RootDir,
		layers,
		util.ExtractFunc(wr.extractFn),
		util.IncludeWhiteout(),
	)
	if err != nil {
		return errors.Wrap(err, "extracting fs from image")
	}

	return nil
}

// FilesToSnapshot returns the workingdir, which should have been created if it didn't already exist
func (wr *CachingWorkdirCommand) FilesToSnapshot() []string {
	f := wr.extractedFiles
	logrus.Debugf("%d files extracted by caching run command", len(f))
	logrus.Tracef("Extracted files: %s", f)

	return f
}

// String returns some information about the command for the image config history
func (wr *CachingWorkdirCommand) String() string {
	if wr.cmd == nil {
		return "nil command"
	}
	return wr.cmd.String()
}

func (wr *CachingWorkdirCommand) MetadataOnly() bool {
	return false
}
