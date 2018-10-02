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

package executor

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"path/filepath"
	"strconv"
	"time"

	"github.com/google/go-containerregistry/pkg/name"
	"github.com/google/go-containerregistry/pkg/v1"
	"github.com/google/go-containerregistry/pkg/v1/mutate"
	"github.com/google/go-containerregistry/pkg/v1/tarball"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"

	"github.com/GoogleContainerTools/kaniko/pkg/cache"
	"github.com/GoogleContainerTools/kaniko/pkg/commands"
	"github.com/GoogleContainerTools/kaniko/pkg/config"
	"github.com/GoogleContainerTools/kaniko/pkg/constants"
	"github.com/GoogleContainerTools/kaniko/pkg/dockerfile"
	"github.com/GoogleContainerTools/kaniko/pkg/snapshot"
	"github.com/GoogleContainerTools/kaniko/pkg/util"
)

// stageBuilder contains all fields necessary to build one stage of a Dockerfile
type stageBuilder struct {
	stage           config.KanikoStage
	image           v1.Image
	cf              *v1.ConfigFile
	snapshotter     *snapshot.Snapshotter
	baseImageDigest string
}

// newStageBuilder returns a new type stageBuilder which contains all the information required to build the stage
func newStageBuilder(opts *config.KanikoOptions, stage config.KanikoStage) (*stageBuilder, error) {
	sourceImage, err := util.RetrieveSourceImage(stage, opts.BuildArgs)
	if err != nil {
		return nil, err
	}
	imageConfig, err := util.RetrieveConfigFile(sourceImage)
	if err != nil {
		return nil, err
	}
	if err := resolveOnBuild(&stage, &imageConfig.Config); err != nil {
		return nil, err
	}
	hasher, err := getHasher(opts.SnapshotMode)
	if err != nil {
		return nil, err
	}
	l := snapshot.NewLayeredMap(hasher, util.CacheHasher())
	snapshotter := snapshot.NewSnapshotter(l, constants.RootDir)

	digest, err := sourceImage.Digest()
	if err != nil {
		return nil, err
	}
	return &stageBuilder{
		stage:           stage,
		image:           sourceImage,
		cf:              imageConfig,
		snapshotter:     snapshotter,
		baseImageDigest: digest.String(),
	}, nil
}

// key will return a string representation of the build at the cmd
func (s *stageBuilder) key(cmd string) (string, error) {
	fsKey, err := s.snapshotter.Key()
	if err != nil {
		return "", err
	}
	c := bytes.NewBuffer([]byte{})
	enc := json.NewEncoder(c)
	enc.Encode(s.cf)
	cf, err := util.SHA256(c)
	if err != nil {
		return "", err
	}
	logrus.Debugf("%s\n%s\n%s\n%s\n", s.baseImageDigest, fsKey, cf, cmd)
	return util.SHA256(bytes.NewReader([]byte(s.baseImageDigest + fsKey + cf + cmd)))
}

// extractCachedLayer will extract the cached layer and append it to the config file
func (s *stageBuilder) extractCachedLayer(layer v1.Image, createdBy string) error {
	logrus.Infof("Found cached layer, extracting to filesystem")
	extractedFiles, err := util.GetFSFromImage(constants.RootDir, layer)
	if err != nil {
		return errors.Wrap(err, "extracting fs from image")
	}
	if _, err := s.snapshotter.TakeSnapshot(extractedFiles); err != nil {
		return err
	}
	logrus.Infof("Appending cached layer to base image")
	l, err := layer.Layers()
	if err != nil {
		return errors.Wrap(err, "getting cached layer from image")
	}
	s.image, err = mutate.Append(s.image,
		mutate.Addendum{
			Layer: l[0],
			History: v1.History{
				Author:    constants.Author,
				CreatedBy: createdBy,
			},
		},
	)
	return err
}

func (s *stageBuilder) build(opts *config.KanikoOptions) error {
	// Unpack file system to root
	if _, err := util.GetFSFromImage(constants.RootDir, s.image); err != nil {
		return err
	}
	// Take initial snapshot
	if err := s.snapshotter.Init(); err != nil {
		return err
	}
	var volumes []string
	args := dockerfile.NewBuildArgs(opts.BuildArgs)
	for index, cmd := range s.stage.Commands {
		finalCmd := index == len(s.stage.Commands)-1
		command, err := commands.GetCommand(cmd, opts.SrcContext)
		if err != nil {
			return err
		}
		if command == nil {
			continue
		}
		logrus.Info(command.String())
		cacheKey, err := s.key(command.String())
		if err != nil {
			return errors.Wrap(err, "getting key")
		}
		if command.CacheCommand() && opts.Cache {
			image, err := cache.RetrieveLayer(opts, cacheKey)
			if err == nil {
				if err := s.extractCachedLayer(image, command.String()); err != nil {
					return errors.Wrap(err, "extracting cached layer")
				}
				continue
			}
			logrus.Info("No cached layer found, executing command...")
		}
		if err := command.ExecuteCommand(&s.cf.Config, args); err != nil {
			return err
		}
		files := command.FilesToSnapshot()
		if cmd.Name() == constants.VolumeCmdName {
			volumes = append(volumes, files...)
			continue
		}
		var contents []byte

		// If this is an intermediate stage, we only snapshot for the last command and we
		// want to snapshot the entire filesystem since we aren't tracking what was changed
		// by previous commands.
		if !s.stage.Final {
			if finalCmd {
				contents, err = s.snapshotter.TakeSnapshotFS()
			}
		} else {
			// If we are in single snapshot mode, we only take a snapshot once, after all
			// commands have completed.
			if opts.SingleSnapshot {
				if finalCmd {
					contents, err = s.snapshotter.TakeSnapshotFS()
				}
			} else {
				// Otherwise, in the final stage we take a snapshot at each command. If we know
				// the files that were changed, we'll snapshot those explicitly, otherwise we'll
				// check if anything in the filesystem changed.
				if files != nil {
					if len(files) > 0 {
						files = append(files, volumes...)
						volumes = []string{}
					}
					contents, err = s.snapshotter.TakeSnapshot(files)
				} else {
					contents, err = s.snapshotter.TakeSnapshotFS()
					volumes = []string{}
				}
			}
		}
		if err != nil {
			return fmt.Errorf("Error taking snapshot of files for command %s: %s", command, err)
		}

		if contents == nil {
			logrus.Info("No files were changed, appending empty layer to config. No layer added to image.")
			continue
		}
		// Append the layer to the image
		opener := func() (io.ReadCloser, error) {
			return ioutil.NopCloser(bytes.NewReader(contents)), nil
		}
		layer, err := tarball.LayerFromOpener(opener)
		if err != nil {
			return err
		}
		// Push layer to cache now along with new config file
		if command.CacheCommand() && opts.Cache {
			if err := pushLayerToCache(opts, cacheKey, layer, command.String()); err != nil {
				return err
			}
		}
		s.image, err = mutate.Append(s.image,
			mutate.Addendum{
				Layer: layer,
				History: v1.History{
					Author:    constants.Author,
					CreatedBy: command.String(),
				},
			},
		)
		if err != nil {
			return err
		}
	}
	return nil
}

// DoBuild executes building the Dockerfile
func DoBuild(opts *config.KanikoOptions) (v1.Image, error) {
	// Parse dockerfile and unpack base image to root
	stages, err := dockerfile.Stages(opts)
	if err != nil {
		return nil, err
	}
	for index, stage := range stages {
		sb, err := newStageBuilder(opts, stage)
		if err != nil {
			return nil, errors.Wrap(err, fmt.Sprintf("getting stage builder for stage %d", index))
		}
		if err := sb.build(opts); err != nil {
			return nil, errors.Wrap(err, "error building stage")
		}
		reviewConfig(stage, &sb.cf.Config)
		sourceImage, err := mutate.Config(sb.image, sb.cf.Config)
		if err != nil {
			return nil, err
		}
		if stage.Final {
			sourceImage, err = mutate.CreatedAt(sourceImage, v1.Time{Time: time.Now()})
			if err != nil {
				return nil, err
			}
			if opts.Reproducible {
				sourceImage, err = mutate.Canonical(sourceImage)
				if err != nil {
					return nil, err
				}
			}
			if opts.Cleanup {
				if err = util.DeleteFilesystem(); err != nil {
					return nil, err
				}
			}
			return sourceImage, nil
		}
		if stage.SaveStage {
			if err := saveStageAsTarball(index, sourceImage); err != nil {
				return nil, err
			}
			if err := extractImageToDependecyDir(index, sourceImage); err != nil {
				return nil, err
			}
		}
		// Delete the filesystem
		if err := util.DeleteFilesystem(); err != nil {
			return nil, err
		}
	}
	return nil, err
}

func extractImageToDependecyDir(index int, image v1.Image) error {
	dependencyDir := filepath.Join(constants.KanikoDir, strconv.Itoa(index))
	if err := os.MkdirAll(dependencyDir, 0755); err != nil {
		return err
	}
	logrus.Infof("trying to extract to %s", dependencyDir)
	_, err := util.GetFSFromImage(dependencyDir, image)
	return err
}

func saveStageAsTarball(stageIndex int, image v1.Image) error {
	destRef, err := name.NewTag("temp/tag", name.WeakValidation)
	if err != nil {
		return err
	}
	if err := os.MkdirAll(constants.KanikoIntermediateStagesDir, 0750); err != nil {
		return err
	}
	tarPath := filepath.Join(constants.KanikoIntermediateStagesDir, strconv.Itoa(stageIndex))
	logrus.Infof("Storing source image from stage %d at path %s", stageIndex, tarPath)
	return tarball.WriteToFile(tarPath, destRef, image)
}

func getHasher(snapshotMode string) (func(string) (string, error), error) {
	if snapshotMode == constants.SnapshotModeTime {
		logrus.Info("Only file modification time will be considered when snapshotting")
		return util.MtimeHasher(), nil
	}
	if snapshotMode == constants.SnapshotModeFull {
		return util.Hasher(), nil
	}
	return nil, fmt.Errorf("%s is not a valid snapshot mode", snapshotMode)
}

func resolveOnBuild(stage *config.KanikoStage, config *v1.Config) error {
	if config.OnBuild == nil {
		return nil
	}
	// Otherwise, parse into commands
	cmds, err := dockerfile.ParseCommands(config.OnBuild)
	if err != nil {
		return err
	}
	// Append to the beginning of the commands in the stage
	stage.Commands = append(cmds, stage.Commands...)
	logrus.Infof("Executing %v build triggers", len(cmds))

	// Blank out the Onbuild command list for this image
	config.OnBuild = nil
	return nil
}

// reviewConfig makes sure the value of CMD is correct after building the stage
// If ENTRYPOINT was set in this stage but CMD wasn't, then CMD should be cleared out
// See Issue #346 for more info
func reviewConfig(stage config.KanikoStage, config *v1.Config) {
	entrypoint := false
	cmd := false

	for _, c := range stage.Commands {
		if c.Name() == constants.Cmd {
			cmd = true
		}
		if c.Name() == constants.Entrypoint {
			entrypoint = true
		}
	}
	if entrypoint && !cmd {
		config.Cmd = nil
	}
}
