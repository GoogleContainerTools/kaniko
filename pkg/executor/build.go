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
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"time"

	"github.com/otiai10/copy"

	"github.com/google/go-containerregistry/pkg/v1/partial"

	"github.com/moby/buildkit/frontend/dockerfile/instructions"

	"golang.org/x/sync/errgroup"

	"github.com/google/go-containerregistry/pkg/name"
	v1 "github.com/google/go-containerregistry/pkg/v1"
	"github.com/google/go-containerregistry/pkg/v1/empty"
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
	"github.com/GoogleContainerTools/kaniko/pkg/timing"
	"github.com/GoogleContainerTools/kaniko/pkg/util"
)

// This is the size of an empty tar in Go
const emptyTarSize = 1024

// stageBuilder contains all fields necessary to build one stage of a Dockerfile
type stageBuilder struct {
	stage           config.KanikoStage
	image           v1.Image
	cf              *v1.ConfigFile
	snapshotter     *snapshot.Snapshotter
	baseImageDigest string
	opts            *config.KanikoOptions
	cmds            []commands.DockerCommand
	args            *dockerfile.BuildArgs
	crossStageDeps  map[int][]string
}

// newStageBuilder returns a new type stageBuilder which contains all the information required to build the stage
func newStageBuilder(opts *config.KanikoOptions, stage config.KanikoStage, crossStageDeps map[int][]string) (*stageBuilder, error) {
	sourceImage, err := util.RetrieveSourceImage(stage, opts)
	if err != nil {
		return nil, err
	}

	imageConfig, err := initializeConfig(sourceImage)
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
	s := &stageBuilder{
		stage:           stage,
		image:           sourceImage,
		cf:              imageConfig,
		snapshotter:     snapshotter,
		baseImageDigest: digest.String(),
		opts:            opts,
		crossStageDeps:  crossStageDeps,
	}

	for _, cmd := range s.stage.Commands {
		command, err := commands.GetCommand(cmd, opts.SrcContext)
		if err != nil {
			return nil, err
		}
		if command == nil {
			continue
		}
		s.cmds = append(s.cmds, command)
	}

	s.args = dockerfile.NewBuildArgs(s.opts.BuildArgs)
	s.args.AddMetaArgs(s.stage.MetaArgs)
	return s, nil
}

func initializeConfig(img partial.WithConfigFile) (*v1.ConfigFile, error) {
	imageConfig, err := img.ConfigFile()
	if err != nil {
		return nil, err
	}

	if imageConfig.Config.Env == nil {
		imageConfig.Config.Env = constants.ScratchEnvVars
	}
	return imageConfig, nil
}

func (s *stageBuilder) optimize(compositeKey CompositeCache, cfg v1.Config) error {
	if !s.opts.Cache {
		return nil
	}

	layerCache := &cache.RegistryCache{
		Opts: s.opts,
	}

	// Possibly replace commands with their cached implementations.
	// We walk through all the commands, running any commands that only operate on metadata.
	// We throw the metadata away after, but we need it to properly track command dependencies
	// for things like COPY ${FOO} or RUN commands that use environment variables.
	for i, command := range s.cmds {
		if command == nil {
			continue
		}
		compositeKey.AddKey(command.String())
		// If the command uses files from the context, add them.
		files, err := command.FilesUsedFromContext(&cfg, s.args)
		if err != nil {
			return err
		}
		for _, f := range files {
			if err := compositeKey.AddPath(f); err != nil {
				return err
			}
		}

		ck, err := compositeKey.Hash()
		if err != nil {
			return err
		}
		if command.ShouldCacheOutput() {
			img, err := layerCache.RetrieveLayer(ck)
			if err != nil {
				logrus.Debugf("Failed to retrieve layer: %s", err)
				logrus.Infof("No cached layer found for cmd %s", command.String())
				logrus.Debugf("Key missing was: %s", compositeKey.Key())
				break
			}

			if cacheCmd := command.CacheCommand(img); cacheCmd != nil {
				logrus.Infof("Using caching version of cmd: %s", command.String())
				s.cmds[i] = cacheCmd
			}
		}

		// Mutate the config for any commands that require it.
		if command.MetadataOnly() {
			if err := command.ExecuteCommand(&cfg, s.args); err != nil {
				return err
			}
		}
	}
	return nil
}

func (s *stageBuilder) build() error {
	// Set the initial cache key to be the base image digest, the build args and the SrcContext.
	compositeKey := NewCompositeCache(s.baseImageDigest)
	compositeKey.AddKey(s.opts.BuildArgs...)

	// Apply optimizations to the instructions.
	if err := s.optimize(*compositeKey, s.cf.Config); err != nil {
		return err
	}

	// Unpack file system to root if we need to.
	shouldUnpack := false
	for _, cmd := range s.cmds {
		if cmd.RequiresUnpackedFS() {
			logrus.Infof("Unpacking rootfs as cmd %s requires it.", cmd.String())
			shouldUnpack = true
			break
		}
	}
	if len(s.crossStageDeps[s.stage.Index]) > 0 {
		shouldUnpack = true
	}

	if shouldUnpack {
		t := timing.Start("FS Unpacking")
		if _, err := util.GetFSFromImage(constants.RootDir, s.image); err != nil {
			return err
		}
		timing.DefaultRun.Stop(t)
	} else {
		logrus.Info("Skipping unpacking as no commands require it.")
	}
	if err := util.DetectFilesystemWhitelist(constants.WhitelistPath); err != nil {
		return err
	}
	// Take initial snapshot
	t := timing.Start("Initial FS snapshot")
	if err := s.snapshotter.Init(); err != nil {
		return err
	}
	timing.DefaultRun.Stop(t)

	cacheGroup := errgroup.Group{}
	for index, command := range s.cmds {
		if command == nil {
			continue
		}

		// Add the next command to the cache key.
		compositeKey.AddKey(command.String())
		t := timing.Start("Command: " + command.String())
		// If the command uses files from the context, add them.
		files, err := command.FilesUsedFromContext(&s.cf.Config, s.args)
		if err != nil {
			return err
		}
		for _, f := range files {
			if err := compositeKey.AddPath(f); err != nil {
				return err
			}
		}
		logrus.Info(command.String())

		if err := command.ExecuteCommand(&s.cf.Config, s.args); err != nil {
			return err
		}
		files = command.FilesToSnapshot()
		timing.DefaultRun.Stop(t)

		if !s.shouldTakeSnapshot(index, files) {
			continue
		}

		tarPath, err := s.takeSnapshot(files)
		if err != nil {
			return err
		}

		ck, err := compositeKey.Hash()
		if err != nil {
			return err
		}
		// Push layer to cache (in parallel) now along with new config file
		if s.opts.Cache && command.ShouldCacheOutput() {
			cacheGroup.Go(func() error {
				return pushLayerToCache(s.opts, ck, tarPath, command.String())
			})
		}
		if err := s.saveSnapshotToImage(command.String(), tarPath); err != nil {
			return err
		}
	}
	if err := cacheGroup.Wait(); err != nil {
		logrus.Warnf("error uploading layer to cache: %s", err)
	}
	return nil
}

func (s *stageBuilder) takeSnapshot(files []string) (string, error) {
	var snapshot string
	var err error
	t := timing.Start("Snapshotting FS")
	if files == nil || s.opts.SingleSnapshot {
		snapshot, err = s.snapshotter.TakeSnapshotFS()
	} else {
		// Volumes are very weird. They get snapshotted in the next command.
		files = append(files, util.Volumes()...)
		snapshot, err = s.snapshotter.TakeSnapshot(files)
	}
	timing.DefaultRun.Stop(t)
	return snapshot, err
}

func (s *stageBuilder) shouldTakeSnapshot(index int, files []string) bool {
	isLastCommand := index == len(s.stage.Commands)-1

	// We only snapshot the very end with single snapshot mode on.
	if s.opts.SingleSnapshot {
		return isLastCommand
	}

	// Always take snapshots if we're using the cache.
	if s.opts.Cache {
		return true
	}

	// nil means snapshot everything.
	if files == nil {
		return true
	}

	// Don't snapshot an empty list.
	if len(files) == 0 {
		return false
	}

	return true
}

func (s *stageBuilder) saveSnapshotToImage(createdBy string, tarPath string) error {
	if tarPath == "" {
		return nil
	}
	fi, err := os.Stat(tarPath)
	if err != nil {
		return err
	}
	if fi.Size() <= emptyTarSize {
		logrus.Info("No files were changed, appending empty layer to config. No layer added to image.")
		return nil
	}

	layer, err := tarball.LayerFromFile(tarPath)
	if err != nil {
		return err
	}
	s.image, err = mutate.Append(s.image,
		mutate.Addendum{
			Layer: layer,
			History: v1.History{
				Author:    constants.Author,
				CreatedBy: createdBy,
			},
		},
	)
	return err

}

func CalculateDependencies(opts *config.KanikoOptions) (map[int][]string, error) {
	stages, err := dockerfile.Stages(opts)
	if err != nil {
		return nil, err
	}
	images := []v1.Image{}
	depGraph := map[int][]string{}
	for _, s := range stages {
		ba := dockerfile.NewBuildArgs(opts.BuildArgs)
		ba.AddMetaArgs(s.MetaArgs)
		var image v1.Image
		var err error
		if s.BaseImageStoredLocally {
			image = images[s.BaseImageIndex]
		} else if s.Name == constants.NoBaseImage {
			image = empty.Image
		} else {
			image, err = util.RetrieveSourceImage(s, opts)
			if err != nil {
				return nil, err
			}
		}
		cfg, err := initializeConfig(image)
		if err != nil {
			return nil, err
		}
		for _, c := range s.Commands {
			switch cmd := c.(type) {
			case *instructions.CopyCommand:
				if cmd.From != "" {
					i, err := strconv.Atoi(cmd.From)
					if err != nil {
						continue
					}
					resolved, err := util.ResolveEnvironmentReplacementList(cmd.SourcesAndDest, ba.ReplacementEnvs(cfg.Config.Env), true)
					if err != nil {
						return nil, err
					}

					depGraph[i] = append(depGraph[i], resolved[0:len(resolved)-1]...)
				}
			case *instructions.EnvCommand:
				if err := util.UpdateConfigEnv(cmd.Env, &cfg.Config, ba.ReplacementEnvs(cfg.Config.Env)); err != nil {
					return nil, err
				}
				image, err = mutate.Config(image, cfg.Config)
				if err != nil {
					return nil, err
				}
			case *instructions.ArgCommand:
				k, v, err := commands.ParseArg(cmd.Key, cmd.Value, cfg.Config.Env, ba)
				if err != nil {
					return nil, err
				}
				ba.AddArg(k, v)
			}
		}
		images = append(images, image)
	}
	return depGraph, nil
}

// DoBuild executes building the Dockerfile
func DoBuild(opts *config.KanikoOptions) (v1.Image, error) {
	t := timing.Start("Total Build Time")
	// Parse dockerfile and unpack base image to root
	stages, err := dockerfile.Stages(opts)
	if err != nil {
		return nil, err
	}
	if err := util.GetExcludedFiles(opts.SrcContext); err != nil {
		return nil, err
	}
	// Some stages may refer to other random images, not previous stages
	if err := fetchExtraStages(stages, opts); err != nil {
		return nil, err
	}

	crossStageDependencies, err := CalculateDependencies(opts)
	if err != nil {
		return nil, err
	}
	logrus.Infof("Built cross stage deps: %v", crossStageDependencies)

	for index, stage := range stages {
		sb, err := newStageBuilder(opts, stage, crossStageDependencies)
		if err != nil {
			return nil, err
		}
		if err := sb.build(); err != nil {
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
			timing.DefaultRun.Stop(t)
			return sourceImage, nil
		}
		if stage.SaveStage {
			if err := saveStageAsTarball(strconv.Itoa(index), sourceImage); err != nil {
				return nil, err
			}
		}

		filesToSave, err := filesToSave(crossStageDependencies[index])
		if err != nil {
			return nil, err
		}
		dstDir := filepath.Join(constants.KanikoDir, strconv.Itoa(index))
		if err := os.MkdirAll(dstDir, 0644); err != nil {
			return nil, err
		}
		for _, p := range filesToSave {
			logrus.Infof("Saving file %s for later use.", p)
			copy.Copy(p, filepath.Join(dstDir, p))
		}

		// Delete the filesystem
		if err := util.DeleteFilesystem(); err != nil {
			return nil, err
		}
	}

	return nil, err
}

func filesToSave(deps []string) ([]string, error) {
	allFiles := []string{}
	for _, src := range deps {
		srcs, err := filepath.Glob(src)
		if err != nil {
			return nil, err
		}
		allFiles = append(allFiles, srcs...)
	}
	return allFiles, nil
}

func fetchExtraStages(stages []config.KanikoStage, opts *config.KanikoOptions) error {
	t := timing.Start("Fetching Extra Stages")
	defer timing.DefaultRun.Stop(t)

	var names = []string{}

	for stageIndex, s := range stages {
		for _, cmd := range s.Commands {
			c, ok := cmd.(*instructions.CopyCommand)
			if !ok || c.From == "" {
				continue
			}

			// FROMs at this point are guaranteed to be either an integer referring to a previous stage,
			// the name of a previous stage, or a name of a remote image.

			// If it is an integer stage index, validate that it is actually a previous index
			if fromIndex, err := strconv.Atoi(c.From); err == nil && stageIndex > fromIndex && fromIndex >= 0 {
				continue
			}
			// Check if the name is the alias of a previous stage
			for _, name := range names {
				if name == c.From {
					continue
				}
			}
			// This must be an image name, fetch it.
			logrus.Debugf("Found extra base image stage %s", c.From)
			sourceImage, err := util.RetrieveRemoteImage(c.From, opts)
			if err != nil {
				return err
			}
			if err := saveStageAsTarball(c.From, sourceImage); err != nil {
				return err
			}
			if err := extractImageToDependencyDir(c.From, sourceImage); err != nil {
				return err
			}
		}
		// Store the name of the current stage in the list with names, if applicable.
		if s.Name != "" {
			names = append(names, s.Name)
		}
	}
	return nil
}
func extractImageToDependencyDir(name string, image v1.Image) error {
	t := timing.Start("Extracting Image to Dependency Dir")
	defer timing.DefaultRun.Stop(t)
	dependencyDir := filepath.Join(constants.KanikoDir, name)
	if err := os.MkdirAll(dependencyDir, 0755); err != nil {
		return err
	}
	logrus.Debugf("trying to extract to %s", dependencyDir)
	_, err := util.GetFSFromImage(dependencyDir, image)
	return err
}

func saveStageAsTarball(path string, image v1.Image) error {
	t := timing.Start("Saving stage as tarball")
	defer timing.DefaultRun.Stop(t)
	destRef, err := name.NewTag("temp/tag", name.WeakValidation)
	if err != nil {
		return err
	}
	tarPath := filepath.Join(constants.KanikoIntermediateStagesDir, path)
	logrus.Infof("Storing source image from stage %s at path %s", path, tarPath)
	if err := os.MkdirAll(filepath.Dir(tarPath), 0750); err != nil {
		return err
	}
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
