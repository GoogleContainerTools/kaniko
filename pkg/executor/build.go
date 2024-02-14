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
	"runtime"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/google/go-containerregistry/pkg/name"
	v1 "github.com/google/go-containerregistry/pkg/v1"
	"github.com/google/go-containerregistry/pkg/v1/empty"
	"github.com/google/go-containerregistry/pkg/v1/mutate"
	"github.com/google/go-containerregistry/pkg/v1/tarball"
	"github.com/google/go-containerregistry/pkg/v1/types"
	"github.com/moby/buildkit/frontend/dockerfile/instructions"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
	"golang.org/x/sync/errgroup"

	"github.com/GoogleContainerTools/kaniko/pkg/cache"
	"github.com/GoogleContainerTools/kaniko/pkg/commands"
	"github.com/GoogleContainerTools/kaniko/pkg/config"
	"github.com/GoogleContainerTools/kaniko/pkg/constants"
	"github.com/GoogleContainerTools/kaniko/pkg/dockerfile"
	image_util "github.com/GoogleContainerTools/kaniko/pkg/image"
	"github.com/GoogleContainerTools/kaniko/pkg/image/remote"
	"github.com/GoogleContainerTools/kaniko/pkg/snapshot"
	"github.com/GoogleContainerTools/kaniko/pkg/timing"
	"github.com/GoogleContainerTools/kaniko/pkg/util"
	"github.com/google/go-containerregistry/pkg/v1/partial"
)

// This is the size of an empty tar in Go
const emptyTarSize = 1024

// for testing
var (
	initializeConfig = initConfig
	getFSFromImage   = util.GetFSFromImage
)

type cachePusher func(*config.KanikoOptions, string, string, string) error
type snapShotter interface {
	Init() error
	TakeSnapshotFS() (string, error)
	TakeSnapshot([]string, bool, bool) (string, error)
}

// stageBuilder contains all fields necessary to build one stage of a Dockerfile
type stageBuilder struct {
	stage            config.KanikoStage
	image            v1.Image
	cf               *v1.ConfigFile
	baseImageDigest  string
	finalCacheKey    string
	opts             *config.KanikoOptions
	fileContext      util.FileContext
	cmds             []commands.DockerCommand
	args             *dockerfile.BuildArgs
	crossStageDeps   map[int][]string
	digestToCacheKey map[string]string
	stageIdxToDigest map[string]string
	snapshotter      snapShotter
	layerCache       cache.LayerCache
	pushLayerToCache cachePusher
}

// newStageBuilder returns a new type stageBuilder which contains all the information required to build the stage
func newStageBuilder(args *dockerfile.BuildArgs, opts *config.KanikoOptions, stage config.KanikoStage, crossStageDeps map[int][]string, dcm map[string]string, sid map[string]string, stageNameToIdx map[string]string, fileContext util.FileContext) (*stageBuilder, error) {
	sourceImage, err := image_util.RetrieveSourceImage(stage, opts)
	if err != nil {
		return nil, err
	}

	imageConfig, err := initializeConfig(sourceImage, opts)
	if err != nil {
		return nil, err
	}

	if err := resolveOnBuild(&stage, &imageConfig.Config, stageNameToIdx); err != nil {
		return nil, err
	}

	err = util.InitIgnoreList()
	if err != nil {
		return nil, errors.Wrap(err, "failed to initialize ignore list")
	}

	hasher, err := getHasher(opts.SnapshotMode)
	if err != nil {
		return nil, err
	}
	l := snapshot.NewLayeredMap(hasher)
	snapshotter := snapshot.NewSnapshotter(l, config.RootDir)

	digest, err := sourceImage.Digest()
	if err != nil {
		return nil, err
	}
	s := &stageBuilder{
		stage:            stage,
		image:            sourceImage,
		cf:               imageConfig,
		snapshotter:      snapshotter,
		baseImageDigest:  digest.String(),
		opts:             opts,
		fileContext:      fileContext,
		crossStageDeps:   crossStageDeps,
		digestToCacheKey: dcm,
		stageIdxToDigest: sid,
		layerCache:       newLayerCache(opts),
		pushLayerToCache: pushLayerToCache,
	}

	for _, cmd := range s.stage.Commands {
		command, err := commands.GetCommand(cmd, fileContext, opts.RunV2, opts.CacheCopyLayers, opts.CacheRunLayers)
		if err != nil {
			return nil, err
		}
		if command == nil {
			continue
		}
		s.cmds = append(s.cmds, command)
	}

	if args != nil {
		s.args = args.Clone()
	} else {
		s.args = dockerfile.NewBuildArgs(s.opts.BuildArgs)
	}
	s.args.AddMetaArgs(s.stage.MetaArgs)
	return s, nil
}

func initConfig(img partial.WithConfigFile, opts *config.KanikoOptions) (*v1.ConfigFile, error) {
	imageConfig, err := img.ConfigFile()
	if err != nil {
		return nil, err
	}

	if imageConfig.Config.Env == nil {
		imageConfig.Config.Env = constants.ScratchEnvVars
	}

	if opts == nil {
		return imageConfig, nil
	}

	if l := len(opts.Labels); l > 0 {
		if imageConfig.Config.Labels == nil {
			imageConfig.Config.Labels = make(map[string]string)
		}
		for _, label := range opts.Labels {
			parts := strings.SplitN(label, "=", 2)
			if len(parts) != 2 {
				return nil, fmt.Errorf("labels must be of the form key=value, got %s", label)
			}

			imageConfig.Config.Labels[parts[0]] = parts[1]
		}
	}

	return imageConfig, nil
}

func newLayerCache(opts *config.KanikoOptions) cache.LayerCache {
	if isOCILayout(opts.CacheRepo) {
		return &cache.LayoutCache{
			Opts: opts,
		}
	}
	return &cache.RegistryCache{
		Opts: opts,
	}
}

func isOCILayout(path string) bool {
	return strings.HasPrefix(path, "oci:")
}

func (s *stageBuilder) populateCompositeKey(command commands.DockerCommand, files []string, compositeKey CompositeCache, args *dockerfile.BuildArgs, env []string) (CompositeCache, error) {
	// First replace all the environment variables or args in the command
	replacementEnvs := args.ReplacementEnvs(env)
	// The sort order of `replacementEnvs` is basically undefined, sort it
	// so we can ensure a stable cache key.
	sort.Strings(replacementEnvs)
	// Use the special argument "|#" at the start of the args array. This will
	// avoid conflicts with any RUN command since commands can not
	// start with | (vertical bar). The "#" (number of build envs) is there to
	// help ensure proper cache matches.

	if command.IsArgsEnvsRequiredInCache() {
		if len(replacementEnvs) > 0 {
			compositeKey.AddKey(fmt.Sprintf("|%d", len(replacementEnvs)))
			compositeKey.AddKey(replacementEnvs...)
		}
	}

	// Add the next command to the cache key.
	compositeKey.AddKey(command.String())

	for _, f := range files {
		if err := compositeKey.AddPath(f, s.fileContext); err != nil {
			return compositeKey, err
		}
	}
	return compositeKey, nil
}

func (s *stageBuilder) optimize(compositeKey CompositeCache, cfg v1.Config) error {
	if !s.opts.Cache {
		return nil
	}
	var buildArgs = s.args.Clone()
	// Restore build args back to their original values
	defer func() {
		s.args = buildArgs
	}()

	stopCache := false
	// Possibly replace commands with their cached implementations.
	// We walk through all the commands, running any commands that only operate on metadata.
	// We throw the metadata away after, but we need it to properly track command dependencies
	// for things like COPY ${FOO} or RUN commands that use environment variables.
	for i, command := range s.cmds {
		if command == nil {
			continue
		}
		files, err := command.FilesUsedFromContext(&cfg, s.args)
		if err != nil {
			return errors.Wrap(err, "failed to get files used from context")
		}

		compositeKey, err = s.populateCompositeKey(command, files, compositeKey, s.args, cfg.Env)
		if err != nil {
			return err
		}

		logrus.Debugf("Optimize: composite key for command %v %v", command.String(), compositeKey)
		ck, err := compositeKey.Hash()
		if err != nil {
			return errors.Wrap(err, "failed to hash composite key")
		}

		logrus.Debugf("Optimize: cache key for command %v %v", command.String(), ck)
		s.finalCacheKey = ck

		if command.ShouldCacheOutput() && !stopCache {
			img, err := s.layerCache.RetrieveLayer(ck)

			if err != nil {
				logrus.Debugf("Failed to retrieve layer: %s", err)
				logrus.Infof("No cached layer found for cmd %s", command.String())
				logrus.Debugf("Key missing was: %s", compositeKey.Key())
				stopCache = true
				continue
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
	var compositeKey *CompositeCache
	if cacheKey, ok := s.digestToCacheKey[s.baseImageDigest]; ok {
		compositeKey = NewCompositeCache(cacheKey)
	} else {
		compositeKey = NewCompositeCache(s.baseImageDigest)
	}

	// Apply optimizations to the instructions.
	if err := s.optimize(*compositeKey, s.cf.Config); err != nil {
		return errors.Wrap(err, "failed to optimize instructions")
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
	if s.stage.Index == 0 && s.opts.InitialFSUnpacked {
		shouldUnpack = false
	}

	if shouldUnpack {
		t := timing.Start("FS Unpacking")

		retryFunc := func() error {
			_, err := getFSFromImage(config.RootDir, s.image, util.ExtractFile)
			return err
		}

		if err := util.Retry(retryFunc, s.opts.ImageFSExtractRetry, 1000); err != nil {
			return errors.Wrap(err, "failed to get filesystem from image")
		}

		timing.DefaultRun.Stop(t)
	} else {
		logrus.Info("Skipping unpacking as no commands require it.")
	}

	initSnapshotTaken := false
	if s.opts.SingleSnapshot {
		if err := s.initSnapshotWithTimings(); err != nil {
			return err
		}
		initSnapshotTaken = true
	}

	cacheGroup := errgroup.Group{}
	for index, command := range s.cmds {
		if command == nil {
			continue
		}

		t := timing.Start("Command: " + command.String())

		// If the command uses files from the context, add them.
		files, err := command.FilesUsedFromContext(&s.cf.Config, s.args)
		if err != nil {
			return errors.Wrap(err, "failed to get files used from context")
		}

		if s.opts.Cache {
			*compositeKey, err = s.populateCompositeKey(command, files, *compositeKey, s.args, s.cf.Config.Env)
			if err != nil && s.opts.Cache {
				return err
			}
		}

		logrus.Info(command.String())

		isCacheCommand := func() bool {
			switch command.(type) {
			case commands.Cached:
				return true
			default:
				return false
			}
		}()
		if !initSnapshotTaken && !isCacheCommand && !command.ProvidesFilesToSnapshot() {
			// Take initial snapshot if command does not expect to return
			// a list of files.
			if err := s.initSnapshotWithTimings(); err != nil {
				return err
			}
			initSnapshotTaken = true
		}

		if err := command.ExecuteCommand(&s.cf.Config, s.args); err != nil {
			return errors.Wrap(err, "failed to execute command")
		}
		files = command.FilesToSnapshot()
		timing.DefaultRun.Stop(t)

		if !s.shouldTakeSnapshot(index, command.MetadataOnly()) && !s.opts.ForceBuildMetadata {
			logrus.Debugf("Build: skipping snapshot for [%v]", command.String())
			continue
		}
		if isCacheCommand {
			v := command.(commands.Cached)
			layer := v.Layer()
			if err := s.saveLayerToImage(layer, command.String()); err != nil {
				return errors.Wrap(err, "failed to save layer")
			}
		} else {
			tarPath, err := s.takeSnapshot(files, command.ShouldDetectDeletedFiles())
			if err != nil {
				return errors.Wrap(err, "failed to take snapshot")
			}

			if s.opts.Cache {
				logrus.Debugf("Build: composite key for command %v %v", command.String(), compositeKey)
				ck, err := compositeKey.Hash()
				if err != nil {
					return errors.Wrap(err, "failed to hash composite key")
				}

				logrus.Debugf("Build: cache key for command %v %v", command.String(), ck)

				// Push layer to cache (in parallel) now along with new config file
				if command.ShouldCacheOutput() && !s.opts.NoPushCache {
					cacheGroup.Go(func() error {
						return s.pushLayerToCache(s.opts, ck, tarPath, command.String())
					})
				}
			}
			if err := s.saveSnapshotToImage(command.String(), tarPath); err != nil {
				return errors.Wrap(err, "failed to save snapshot to image")
			}
		}
	}

	if err := cacheGroup.Wait(); err != nil {
		logrus.Warnf("Error uploading layer to cache: %s", err)
	}

	return nil
}

func (s *stageBuilder) takeSnapshot(files []string, shdDelete bool) (string, error) {
	var snapshot string
	var err error

	t := timing.Start("Snapshotting FS")
	if files == nil || s.opts.SingleSnapshot {
		snapshot, err = s.snapshotter.TakeSnapshotFS()
	} else {
		// Volumes are very weird. They get snapshotted in the next command.
		files = append(files, util.Volumes()...)
		snapshot, err = s.snapshotter.TakeSnapshot(files, shdDelete, s.opts.ForceBuildMetadata)
	}
	timing.DefaultRun.Stop(t)
	return snapshot, err
}

func (s *stageBuilder) shouldTakeSnapshot(index int, isMetadatCmd bool) bool {
	isLastCommand := index == len(s.cmds)-1

	// We only snapshot the very end with single snapshot mode on.
	if s.opts.SingleSnapshot {
		return isLastCommand
	}

	// Always take snapshots if we're using the cache.
	if s.opts.Cache {
		return true
	}

	// if command is a metadata command, do not snapshot.
	return !isMetadatCmd
}

func (s *stageBuilder) saveSnapshotToImage(createdBy string, tarPath string) error {
	layer, err := s.saveSnapshotToLayer(tarPath)
	if err != nil {
		return err
	}

	if layer == nil {
		return nil
	}

	return s.saveLayerToImage(layer, createdBy)
}

func (s *stageBuilder) saveSnapshotToLayer(tarPath string) (v1.Layer, error) {
	if tarPath == "" {
		return nil, nil
	}
	fi, err := os.Stat(tarPath)
	if err != nil {
		return nil, errors.Wrap(err, "tar file path does not exist")
	}
	if fi.Size() <= emptyTarSize && !s.opts.ForceBuildMetadata {
		logrus.Info("No files were changed, appending empty layer to config. No layer added to image.")
		return nil, nil
	}

	layerOpts := s.getLayerOptionFromOpts()
	imageMediaType, err := s.image.MediaType()
	if err != nil {
		return nil, err
	}
	// Only appending MediaType for OCI images as the default is docker
	if extractMediaTypeVendor(imageMediaType) == types.OCIVendorPrefix {
		if s.opts.Compression == config.ZStd {
			layerOpts = append(layerOpts, tarball.WithCompression("zstd"), tarball.WithMediaType(types.OCILayerZStd))
		} else {
			layerOpts = append(layerOpts, tarball.WithMediaType(types.OCILayer))
		}
	}

	layer, err := tarball.LayerFromFile(tarPath, layerOpts...)
	if err != nil {
		return nil, err
	}

	return layer, nil
}

func (s *stageBuilder) getLayerOptionFromOpts() []tarball.LayerOption {
	var layerOpts []tarball.LayerOption

	if s.opts.CompressedCaching {
		layerOpts = append(layerOpts, tarball.WithCompressedCaching)
	}

	if s.opts.CompressionLevel > 0 {
		layerOpts = append(layerOpts, tarball.WithCompressionLevel(s.opts.CompressionLevel))
	}
	return layerOpts
}

func extractMediaTypeVendor(mt types.MediaType) string {
	if strings.Contains(string(mt), types.OCIVendorPrefix) {
		return types.OCIVendorPrefix
	}
	return types.DockerVendorPrefix
}

// https://github.com/opencontainers/image-spec/blob/main/media-types.md#compatibility-matrix
func convertMediaType(mt types.MediaType) types.MediaType {
	switch mt {
	case types.DockerManifestSchema1, types.DockerManifestSchema2:
		return types.OCIManifestSchema1
	case types.DockerManifestList:
		return types.OCIImageIndex
	case types.DockerLayer:
		return types.OCILayer
	case types.DockerConfigJSON:
		return types.OCIConfigJSON
	case types.DockerForeignLayer:
		return types.OCIUncompressedRestrictedLayer
	case types.DockerUncompressedLayer:
		return types.OCIUncompressedLayer
	case types.OCIImageIndex:
		return types.DockerManifestList
	case types.OCIManifestSchema1:
		return types.DockerManifestSchema2
	case types.OCIConfigJSON:
		return types.DockerConfigJSON
	case types.OCILayer, types.OCILayerZStd:
		return types.DockerLayer
	case types.OCIRestrictedLayer:
		return types.DockerForeignLayer
	case types.OCIUncompressedLayer:
		return types.DockerUncompressedLayer
	case types.OCIContentDescriptor, types.OCIUncompressedRestrictedLayer, types.DockerManifestSchema1Signed, types.DockerPluginConfig:
		return ""
	default:
		return ""
	}
}

func (s *stageBuilder) convertLayerMediaType(layer v1.Layer) (v1.Layer, error) {
	layerMediaType, err := layer.MediaType()
	if err != nil {
		return nil, err
	}
	imageMediaType, err := s.image.MediaType()
	if err != nil {
		return nil, err
	}
	if extractMediaTypeVendor(layerMediaType) != extractMediaTypeVendor(imageMediaType) {
		layerOpts := s.getLayerOptionFromOpts()
		targetMediaType := convertMediaType(layerMediaType)

		if extractMediaTypeVendor(imageMediaType) == types.OCIVendorPrefix {
			if s.opts.Compression == config.ZStd {
				targetMediaType = types.OCILayerZStd
				layerOpts = append(layerOpts, tarball.WithCompression("zstd"))
			}
		}

		layerOpts = append(layerOpts, tarball.WithMediaType(targetMediaType))

		if targetMediaType != "" {
			return tarball.LayerFromOpener(layer.Uncompressed, layerOpts...)
		}
		return nil, fmt.Errorf(
			"layer with media type %v cannot be converted to a media type that matches %v",
			layerMediaType,
			imageMediaType,
		)
	}
	return layer, nil
}

func (s *stageBuilder) saveLayerToImage(layer v1.Layer, createdBy string) error {
	var err error
	layer, err = s.convertLayerMediaType(layer)
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

func CalculateDependencies(stages []config.KanikoStage, opts *config.KanikoOptions, stageNameToIdx map[string]string) (map[int][]string, error) {
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
			image, err = image_util.RetrieveSourceImage(s, opts)
			if err != nil {
				return nil, err
			}
		}
		cfg, err := initializeConfig(image, opts)
		if err != nil {
			return nil, err
		}

		cmds, err := dockerfile.GetOnBuildInstructions(&cfg.Config, stageNameToIdx)
		cmds = append(cmds, s.Commands...)

		for _, c := range cmds {
			switch cmd := c.(type) {
			case *instructions.CopyCommand:
				if cmd.From != "" {
					i, err := strconv.Atoi(cmd.From)
					if err != nil {
						continue
					}
					resolved, err := util.ResolveEnvironmentReplacementList(cmd.SourcesAndDest.SourcePaths, ba.ReplacementEnvs(cfg.Config.Env), true)
					if err != nil {
						return nil, err
					}
					depGraph[i] = append(depGraph[i], resolved...)
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
				for _, arg := range cmd.Args {
					k, v, err := commands.ParseArg(arg.Key, arg.Value, cfg.Config.Env, ba)
					if err != nil {
						return nil, err
					}
					ba.AddArg(k, v)
				}
			}
		}
		images = append(images, image)
	}
	return depGraph, nil
}

// DoBuild executes building the Dockerfile
func DoBuild(opts *config.KanikoOptions) (v1.Image, error) {
	t := timing.Start("Total Build Time")
	digestToCacheKey := make(map[string]string)
	stageIdxToDigest := make(map[string]string)

	stages, metaArgs, err := dockerfile.ParseStages(opts)
	if err != nil {
		return nil, err
	}

	kanikoStages, err := dockerfile.MakeKanikoStages(opts, stages, metaArgs)
	if err != nil {
		return nil, err
	}
	stageNameToIdx := ResolveCrossStageInstructions(kanikoStages)

	fileContext, err := util.NewFileContextFromDockerfile(opts.DockerfilePath, opts.SrcContext)
	if err != nil {
		return nil, err
	}

	// Some stages may refer to other random images, not previous stages
	if err := fetchExtraStages(kanikoStages, opts); err != nil {
		return nil, err
	}
	crossStageDependencies, err := CalculateDependencies(kanikoStages, opts, stageNameToIdx)
	if err != nil {
		return nil, err
	}
	logrus.Infof("Built cross stage deps: %v", crossStageDependencies)

	var args *dockerfile.BuildArgs

	for index, stage := range kanikoStages {
		sb, err := newStageBuilder(
			args, opts, stage,
			crossStageDependencies,
			digestToCacheKey,
			stageIdxToDigest,
			stageNameToIdx,
			fileContext)

		logrus.Infof("Building stage '%v' [idx: '%v', base-idx: '%v']",
			stage.BaseName, stage.Index, stage.BaseImageIndex)

		if err != nil {
			return nil, err
		}
		args = sb.args
		if err := sb.build(); err != nil {
			return nil, errors.Wrap(err, "error building stage")
		}

		reviewConfig(stage, &sb.cf.Config)

		sourceImage, err := mutate.Config(sb.image, sb.cf.Config)
		if err != nil {
			return nil, err
		}

		configFile, err := sourceImage.ConfigFile()
		if err != nil {
			return nil, err
		}
		if opts.CustomPlatform == "" {
			configFile.OS = runtime.GOOS
			configFile.Architecture = runtime.GOARCH
		} else {
			configFile.OS = strings.Split(opts.CustomPlatform, "/")[0]
			configFile.Architecture = strings.Split(opts.CustomPlatform, "/")[1]
		}
		sourceImage, err = mutate.ConfigFile(sourceImage, configFile)
		if err != nil {
			return nil, err
		}

		d, err := sourceImage.Digest()
		if err != nil {
			return nil, err
		}
		stageIdxToDigest[fmt.Sprintf("%d", sb.stage.Index)] = d.String()
		logrus.Debugf("Mapping stage idx %v to digest %v", sb.stage.Index, d.String())

		digestToCacheKey[d.String()] = sb.finalCacheKey
		logrus.Debugf("Mapping digest %v to cachekey %v", d.String(), sb.finalCacheKey)

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
		dstDir := filepath.Join(config.KanikoDir, strconv.Itoa(index))
		if err := os.MkdirAll(dstDir, 0644); err != nil {
			return nil, errors.Wrap(err,
				fmt.Sprintf("to create workspace for stage %s",
					stageIdxToDigest[strconv.Itoa(index)],
				))
		}
		for _, p := range filesToSave {
			logrus.Infof("Saving file %s for later use", p)
			if err := util.CopyFileOrSymlink(p, dstDir, config.RootDir); err != nil {
				return nil, errors.Wrap(err, "could not save file")
			}
		}

		// Delete the filesystem
		if err := util.DeleteFilesystem(); err != nil {
			return nil, errors.Wrap(err, fmt.Sprintf("deleting file system after stage %d", index))
		}
	}

	return nil, err
}

// filesToSave returns all the files matching the given pattern in deps.
// If a file is a symlink, it also returns the target file.
func filesToSave(deps []string) ([]string, error) {
	srcFiles := []string{}
	for _, src := range deps {
		srcs, err := filepath.Glob(filepath.Join(config.RootDir, src))
		if err != nil {
			return nil, err
		}
		for _, f := range srcs {
			if link, err := util.EvalSymLink(f); err == nil {
				link, err = filepath.Rel(config.RootDir, link)
				if err != nil {
					return nil, errors.Wrap(err, fmt.Sprintf("could not find relative path to %s", config.RootDir))
				}
				srcFiles = append(srcFiles, link)
			}
			f, err = filepath.Rel(config.RootDir, f)
			if err != nil {
				return nil, errors.Wrap(err, fmt.Sprintf("could not find relative path to %s", config.RootDir))
			}
			srcFiles = append(srcFiles, f)
		}
	}
	// remove duplicates
	deduped := deduplicatePaths(srcFiles)

	return deduped, nil
}

// deduplicatePaths returns a deduplicated slice of shortest paths
// For example {"usr/lib", "usr/lib/ssl"} will return only {"usr/lib"}
func deduplicatePaths(paths []string) []string {
	type node struct {
		children map[string]*node
		value    bool
	}

	root := &node{children: make(map[string]*node)}

	// Create a tree marking all present paths
	for _, f := range paths {
		parts := strings.Split(f, "/")
		current := root
		for i := 0; i < len(parts)-1; i++ {
			part := parts[i]
			if _, ok := current.children[part]; !ok {
				current.children[part] = &node{children: make(map[string]*node)}
			}
			current = current.children[part]
		}
		current.children[parts[len(parts)-1]] = &node{children: make(map[string]*node), value: true}
	}

	// Collect all paths
	deduped := []string{}
	var traverse func(*node, string)
	traverse = func(n *node, path string) {
		if n.value {
			deduped = append(deduped, strings.TrimPrefix(path, "/"))
			return
		}
		for k, v := range n.children {
			traverse(v, path+"/"+k)
		}
	}

	traverse(root, "")

	return deduped
}

func fetchExtraStages(stages []config.KanikoStage, opts *config.KanikoOptions) error {
	t := timing.Start("Fetching Extra Stages")
	defer timing.DefaultRun.Stop(t)

	var names []string

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
			if fromPreviousStage(c, names) {
				continue
			}

			// This must be an image name, fetch it.
			logrus.Debugf("Found extra base image stage %s", c.From)
			sourceImage, err := remote.RetrieveRemoteImage(c.From, opts.RegistryOptions, opts.CustomPlatform)
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

func fromPreviousStage(copyCommand *instructions.CopyCommand, previousStageNames []string) bool {
	for _, previousStageName := range previousStageNames {
		if previousStageName == copyCommand.From {
			return true
		}
	}
	return false
}

func extractImageToDependencyDir(name string, image v1.Image) error {
	t := timing.Start("Extracting Image to Dependency Dir")
	defer timing.DefaultRun.Stop(t)
	dependencyDir := filepath.Join(config.KanikoDir, name)
	if err := os.MkdirAll(dependencyDir, 0755); err != nil {
		return err
	}
	logrus.Debugf("Trying to extract to %s", dependencyDir)
	_, err := util.GetFSFromImage(dependencyDir, image, util.ExtractFile)
	return err
}

func saveStageAsTarball(path string, image v1.Image) error {
	t := timing.Start("Saving stage as tarball")
	defer timing.DefaultRun.Stop(t)
	destRef, err := name.NewTag("temp/tag", name.WeakValidation)
	if err != nil {
		return err
	}
	tarPath := filepath.Join(config.KanikoIntermediateStagesDir, path)
	logrus.Infof("Storing source image from stage %s at path %s", path, tarPath)
	if err := os.MkdirAll(filepath.Dir(tarPath), 0750); err != nil {
		return err
	}
	return tarball.WriteToFile(tarPath, destRef, image)
}

func getHasher(snapshotMode string) (func(string) (string, error), error) {
	switch snapshotMode {
	case constants.SnapshotModeTime:
		logrus.Info("Only file modification time will be considered when snapshotting")
		return util.MtimeHasher(), nil
	case constants.SnapshotModeFull:
		return util.Hasher(), nil
	case constants.SnapshotModeRedo:
		return util.RedoHasher(), nil
	default:
		return nil, fmt.Errorf("%s is not a valid snapshot mode", snapshotMode)
	}
}

func resolveOnBuild(stage *config.KanikoStage, config *v1.Config, stageNameToIdx map[string]string) error {
	cmds, err := dockerfile.GetOnBuildInstructions(config, stageNameToIdx)
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

// iterates over a list of KanikoStage and resolves instructions referring to earlier stages
// returns a mapping of stage name to stage id, f.e - ["first": "0", "second": "1", "target": "2"]
func ResolveCrossStageInstructions(stages []config.KanikoStage) map[string]string {
	nameToIndex := make(map[string]string)
	for i, stage := range stages {
		index := strconv.Itoa(i)
		if stage.Name != "" {
			nameToIndex[stage.Name] = index
		}
		dockerfile.ResolveCrossStageCommands(stage.Commands, nameToIndex)
	}

	logrus.Debugf("Built stage name to index map: %v", nameToIndex)
	return nameToIndex
}

func (s stageBuilder) initSnapshotWithTimings() error {
	t := timing.Start("Initial FS snapshot")
	if err := s.snapshotter.Init(); err != nil {
		return err
	}
	timing.DefaultRun.Stop(t)
	return nil
}
