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
	"io/ioutil"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/GoogleContainerTools/kaniko/pkg/dockerfile"
	"github.com/GoogleContainerTools/kaniko/pkg/util"
	v1 "github.com/google/go-containerregistry/pkg/v1"
	"github.com/moby/buildkit/frontend/dockerfile/instructions"
)

type RunMarkerCommand struct {
	BaseCommand
	cmd   *instructions.RunCommand
	Files []string
}

func (r *RunMarkerCommand) ExecuteCommand(config *v1.Config, buildArgs *dockerfile.BuildArgs) error {
	// run command `touch filemarker`
	markerFile, err := ioutil.TempFile("", "marker")
	defer func() {
		os.Remove(markerFile.Name())
	}()

	if err := runCommandInExec(config, buildArgs, r.cmd); err != nil {
		return err
	}

	// run command find to find all new files generated
	find := exec.Command("find", "/", "-newer", markerFile.Name())
	out, err := find.Output()
	if err != nil {
		r.Files = []string{}
		return nil
	}

	r.Files = []string{}
	s := strings.Split(string(out), "\n")
	for _, path := range s {
		path = filepath.Clean(path)
		if util.IsDestDir(path) || util.CheckIgnoreList(path) {
			continue
		}
		r.Files = append(r.Files, path)
	}
	return nil
}

// String returns some information about the command for the image config
func (r *RunMarkerCommand) String() string {
	return r.cmd.String()
}

func (r *RunMarkerCommand) FilesToSnapshot() []string {
	return nil
}

func (r *RunMarkerCommand) ProvidesFilesToSnapshot() bool {
	return false
}

// CacheCommand returns true since this command should be cached
func (r *RunMarkerCommand) CacheCommand(img v1.Image) DockerCommand {

	return &CachingRunCommand{
		img:       img,
		cmd:       r.cmd,
		extractFn: util.ExtractFile,
	}
}

func (r *RunMarkerCommand) MetadataOnly() bool {
	return false
}

func (r *RunMarkerCommand) RequiresUnpackedFS() bool {
	return true
}

func (r *RunMarkerCommand) ShouldCacheOutput() bool {
	return true
}

func (b *BaseCommand) ShouldDetectDelete() bool {
	return true
}
