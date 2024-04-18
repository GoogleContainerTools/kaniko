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

package cmd

import (
	"fmt"
	"io/fs"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/GoogleContainerTools/kaniko/pkg/buildcontext"
	"github.com/GoogleContainerTools/kaniko/pkg/config"
	"github.com/GoogleContainerTools/kaniko/pkg/constants"
	"github.com/GoogleContainerTools/kaniko/pkg/executor"
	"github.com/GoogleContainerTools/kaniko/pkg/logging"
	"github.com/GoogleContainerTools/kaniko/pkg/timing"
	"github.com/GoogleContainerTools/kaniko/pkg/util"
	"github.com/GoogleContainerTools/kaniko/pkg/util/proc"
	"github.com/containerd/containerd/platforms"
	"github.com/google/go-containerregistry/pkg/name"
	v1 "github.com/google/go-containerregistry/pkg/v1"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
)

var (
	opts         = &config.KanikoOptions{}
	ctxSubPath   string
	force        bool
	logLevel     string
	logFormat    string
	logTimestamp bool
)

func init() {
	RootCmd.PersistentFlags().StringVarP(&logLevel, "verbosity", "v", logging.DefaultLevel, "Log level (trace, debug, info, warn, error, fatal, panic)")
	RootCmd.PersistentFlags().StringVar(&logFormat, "log-format", logging.FormatColor, "Log format (text, color, json)")
	RootCmd.PersistentFlags().BoolVar(&logTimestamp, "log-timestamp", logging.DefaultLogTimestamp, "Timestamp in log output")
	RootCmd.PersistentFlags().BoolVarP(&force, "force", "", false, "Force building outside of a container")

	addKanikoOptionsFlags()
	addHiddenFlags(RootCmd)
	RootCmd.PersistentFlags().BoolVarP(&opts.IgnoreVarRun, "whitelist-var-run", "", true, "Ignore /var/run directory when taking image snapshot. Set it to false to preserve /var/run/ in destination image.")
	RootCmd.PersistentFlags().MarkDeprecated("whitelist-var-run", "Please use ignore-var-run instead.")
}

func validateFlags() {
	checkNoDeprecatedFlags()

	// Allow setting --registry-mirror using an environment variable.
	if val, ok := os.LookupEnv("KANIKO_REGISTRY_MIRROR"); ok {
		opts.RegistryMirrors.Set(val)
	}

	// Allow setting --no-push using an environment variable.
	if val, ok := os.LookupEnv("KANIKO_NO_PUSH"); ok {
		valBoolean, err := strconv.ParseBool(val)
		if err != nil {
			errors.New("invalid value (true/false) for KANIKO_NO_PUSH environment variable")
		}
		opts.NoPush = valBoolean
	}

	// Allow setting --registry-maps using an environment variable.
	if val, ok := os.LookupEnv("KANIKO_REGISTRY_MAP"); ok {
		opts.RegistryMaps.Set(val)
	}

	for _, target := range opts.RegistryMirrors {
		opts.RegistryMaps.Set(fmt.Sprintf("%s=%s", name.DefaultRegistry, target))
	}

	if len(opts.RegistryMaps) > 0 {
		for src, dsts := range opts.RegistryMaps {
			logrus.Debugf("registry-map remaps %s to %s.", src, strings.Join(dsts, ", "))
		}
	}

	// Default the custom platform flag to our current platform, and validate it.
	if opts.CustomPlatform == "" {
		opts.CustomPlatform = platforms.Format(platforms.Normalize(platforms.DefaultSpec()))
	}
	if _, err := v1.ParsePlatform(opts.CustomPlatform); err != nil {
		logrus.Fatalf("Invalid platform %q: %v", opts.CustomPlatform, err)
	}
}

// RootCmd is the kaniko command that is run
var RootCmd = &cobra.Command{
	Use: "executor",
	PersistentPreRunE: func(cmd *cobra.Command, args []string) error {
		if cmd.Use == "executor" {

			if err := logging.Configure(logLevel, logFormat, logTimestamp); err != nil {
				return err
			}

			validateFlags()

			// Command line flag takes precedence over the KANIKO_DIR environment variable.
			dir := config.KanikoDir
			if opts.KanikoDir != constants.DefaultKanikoPath {
				dir = opts.KanikoDir
			}

			if err := checkKanikoDir(dir); err != nil {
				return err
			}

			resolveEnvironmentBuildArgs(opts.BuildArgs, os.Getenv)

			if !opts.NoPush && len(opts.Destinations) == 0 {
				return errors.New("you must provide --destination, or use --no-push")
			}
			if err := cacheFlagsValid(); err != nil {
				return errors.Wrap(err, "cache flags invalid")
			}
			if err := resolveSourceContext(); err != nil {
				return errors.Wrap(err, "error resolving source context")
			}
			if err := resolveDockerfilePath(); err != nil {
				return errors.Wrap(err, "error resolving dockerfile path")
			}
			if len(opts.Destinations) == 0 && opts.ImageNameDigestFile != "" {
				return errors.New("you must provide --destination if setting ImageNameDigestFile")
			}
			if len(opts.Destinations) == 0 && opts.ImageNameTagDigestFile != "" {
				return errors.New("you must provide --destination if setting ImageNameTagDigestFile")
			}
			// Update ignored paths
			if opts.IgnoreVarRun {
				// /var/run is a special case. It's common to mount in /var/run/docker.sock
				// or something similar which leads to a special mount on the /var/run/docker.sock
				// file itself, but the directory to exist in the image with no way to tell if it came
				// from the base image or not.
				logrus.Trace("Adding /var/run to default ignore list")
				util.AddToDefaultIgnoreList(util.IgnoreListEntry{
					Path:            "/var/run",
					PrefixMatchOnly: false,
				})
			}
			for _, p := range opts.IgnorePaths {
				util.AddToDefaultIgnoreList(util.IgnoreListEntry{
					Path:            p,
					PrefixMatchOnly: false,
				})
			}
		}
		return nil
	},
	Run: func(cmd *cobra.Command, args []string) {
		if !checkContained() {
			if !force {
				exit(errors.New("kaniko should only be run inside of a container, run with the --force flag if you are sure you want to continue"))
			}
			logrus.Warn("Kaniko is being run outside of a container. This can have dangerous effects on your system")
		}
		if !opts.NoPush || opts.CacheRepo != "" {
			if err := executor.CheckPushPermissions(opts); err != nil {
				exit(errors.Wrap(err, "error checking push permissions -- make sure you entered the correct tag name, and that you are authenticated correctly, and try again"))
			}
		}
		if err := resolveRelativePaths(); err != nil {
			exit(errors.Wrap(err, "error resolving relative paths to absolute paths"))
		}
		if err := os.Chdir("/"); err != nil {
			exit(errors.Wrap(err, "error changing to root dir"))
		}
		image, err := executor.DoBuild(opts)
		if err != nil {
			exit(errors.Wrap(err, "error building image"))
		}
		if err := executor.DoPush(image, opts); err != nil {
			exit(errors.Wrap(err, "error pushing image"))
		}

		benchmarkFile := os.Getenv("BENCHMARK_FILE")
		// false is a keyword for integration tests to turn off benchmarking
		if benchmarkFile != "" && benchmarkFile != "false" {
			s, err := timing.JSON()
			if err != nil {
				logrus.Warnf("Unable to write benchmark file: %s", err)
				return
			}
			if strings.HasPrefix(benchmarkFile, "gs://") {
				logrus.Info("Uploading to gcs")
				if err := buildcontext.UploadToBucket(strings.NewReader(s), benchmarkFile); err != nil {
					logrus.Infof("Unable to upload %s due to %v", benchmarkFile, err)
				}
				logrus.Infof("Benchmark file written at %s", benchmarkFile)
			} else {
				f, err := os.Create(benchmarkFile)
				if err != nil {
					logrus.Warnf("Unable to create benchmarking file %s: %s", benchmarkFile, err)
					return
				}
				defer f.Close()
				f.WriteString(s)
				logrus.Infof("Benchmark file written at %s", benchmarkFile)
			}
		}
	},
}

// addKanikoOptionsFlags configures opts
func addKanikoOptionsFlags() {
	RootCmd.PersistentFlags().StringVarP(&opts.DockerfilePath, "dockerfile", "f", "Dockerfile", "Path to the dockerfile to be built.")
	RootCmd.PersistentFlags().StringVarP(&opts.SrcContext, "context", "c", "/workspace/", "Path to the dockerfile build context.")
	RootCmd.PersistentFlags().StringVarP(&ctxSubPath, "context-sub-path", "", "", "Sub path within the given context.")
	RootCmd.PersistentFlags().StringVarP(&opts.Bucket, "bucket", "b", "", "Name of the GCS bucket from which to access build context as tarball.")
	RootCmd.PersistentFlags().VarP(&opts.Destinations, "destination", "d", "Registry the final image should be pushed to. Set it repeatedly for multiple destinations.")
	RootCmd.PersistentFlags().StringVarP(&opts.SnapshotMode, "snapshot-mode", "", "full", "Change the file attributes inspected during snapshotting")
	RootCmd.PersistentFlags().StringVarP(&opts.CustomPlatform, "custom-platform", "", "", "Specify the build platform if different from the current host")
	RootCmd.PersistentFlags().VarP(&opts.BuildArgs, "build-arg", "", "This flag allows you to pass in ARG values at build time. Set it repeatedly for multiple values.")
	RootCmd.PersistentFlags().BoolVarP(&opts.Insecure, "insecure", "", false, "Push to insecure registry using plain HTTP")
	RootCmd.PersistentFlags().BoolVarP(&opts.SkipTLSVerify, "skip-tls-verify", "", false, "Push to insecure registry ignoring TLS verify")
	RootCmd.PersistentFlags().BoolVarP(&opts.InsecurePull, "insecure-pull", "", false, "Pull from insecure registry using plain HTTP")
	RootCmd.PersistentFlags().BoolVarP(&opts.SkipTLSVerifyPull, "skip-tls-verify-pull", "", false, "Pull from insecure registry ignoring TLS verify")
	RootCmd.PersistentFlags().IntVar(&opts.PushRetry, "push-retry", 0, "Number of retries for the push operation")
	RootCmd.PersistentFlags().BoolVar(&opts.PushIgnoreImmutableTagErrors, "push-ignore-immutable-tag-errors", false, "If true, known tag immutability errors are ignored and the push finishes with success.")
	RootCmd.PersistentFlags().IntVar(&opts.ImageFSExtractRetry, "image-fs-extract-retry", 0, "Number of retries for image FS extraction")
	RootCmd.PersistentFlags().IntVar(&opts.ImageDownloadRetry, "image-download-retry", 0, "Number of retries for downloading the remote image")
	RootCmd.PersistentFlags().StringVarP(&opts.KanikoDir, "kaniko-dir", "", constants.DefaultKanikoPath, "Path to the kaniko directory, this takes precedence over the KANIKO_DIR environment variable.")
	RootCmd.PersistentFlags().StringVarP(&opts.TarPath, "tar-path", "", "", "Path to save the image in as a tarball instead of pushing")
	RootCmd.PersistentFlags().BoolVarP(&opts.SingleSnapshot, "single-snapshot", "", false, "Take a single snapshot at the end of the build.")
	RootCmd.PersistentFlags().BoolVarP(&opts.Reproducible, "reproducible", "", false, "Strip timestamps out of the image to make it reproducible")
	RootCmd.PersistentFlags().StringVarP(&opts.Target, "target", "", "", "Set the target build stage to build")
	RootCmd.PersistentFlags().BoolVarP(&opts.NoPush, "no-push", "", false, "Do not push the image to the registry")
	RootCmd.PersistentFlags().BoolVarP(&opts.NoPushCache, "no-push-cache", "", false, "Do not push the cache layers to the registry")
	RootCmd.PersistentFlags().StringVarP(&opts.CacheRepo, "cache-repo", "", "", "Specify a repository to use as a cache, otherwise one will be inferred from the destination provided; when prefixed with 'oci:' the repository will be written in OCI image layout format at the path provided")
	RootCmd.PersistentFlags().StringVarP(&opts.CacheDir, "cache-dir", "", "/cache", "Specify a local directory to use as a cache.")
	RootCmd.PersistentFlags().StringVarP(&opts.DigestFile, "digest-file", "", "", "Specify a file to save the digest of the built image to.")
	RootCmd.PersistentFlags().StringVarP(&opts.ImageNameDigestFile, "image-name-with-digest-file", "", "", "Specify a file to save the image name w/ digest of the built image to.")
	RootCmd.PersistentFlags().StringVarP(&opts.ImageNameTagDigestFile, "image-name-tag-with-digest-file", "", "", "Specify a file to save the image name w/ image tag w/ digest of the built image to.")
	RootCmd.PersistentFlags().StringVarP(&opts.OCILayoutPath, "oci-layout-path", "", "", "Path to save the OCI image layout of the built image.")
	RootCmd.PersistentFlags().VarP(&opts.Compression, "compression", "", "Compression algorithm (gzip, zstd)")
	RootCmd.PersistentFlags().IntVarP(&opts.CompressionLevel, "compression-level", "", -1, "Compression level")
	RootCmd.PersistentFlags().BoolVarP(&opts.Cache, "cache", "", false, "Use cache when building image")
	RootCmd.PersistentFlags().BoolVarP(&opts.CompressedCaching, "compressed-caching", "", true, "Compress the cached layers. Decreases build time, but increases memory usage.")
	RootCmd.PersistentFlags().BoolVarP(&opts.Cleanup, "cleanup", "", false, "Clean the filesystem at the end")
	RootCmd.PersistentFlags().DurationVarP(&opts.CacheTTL, "cache-ttl", "", time.Hour*336, "Cache timeout, requires value and unit of duration -> ex: 6h. Defaults to two weeks.")
	RootCmd.PersistentFlags().VarP(&opts.InsecureRegistries, "insecure-registry", "", "Insecure registry using plain HTTP to push and pull. Set it repeatedly for multiple registries.")
	RootCmd.PersistentFlags().VarP(&opts.SkipTLSVerifyRegistries, "skip-tls-verify-registry", "", "Insecure registry ignoring TLS verify to push and pull. Set it repeatedly for multiple registries.")
	opts.RegistriesCertificates = make(map[string]string)
	RootCmd.PersistentFlags().VarP(&opts.RegistriesCertificates, "registry-certificate", "", "Use the provided certificate for TLS communication with the given registry. Expected format is 'my.registry.url=/path/to/the/server/certificate'.")
	opts.RegistriesClientCertificates = make(map[string]string)
	RootCmd.PersistentFlags().VarP(&opts.RegistriesClientCertificates, "registry-client-cert", "", "Use the provided client certificate for mutual TLS (mTLS) communication with the given registry. Expected format is 'my.registry.url=/path/to/client/cert,/path/to/client/key'.")
	opts.RegistryMaps = make(map[string][]string)
	RootCmd.PersistentFlags().VarP(&opts.RegistryMaps, "registry-map", "", "Registry map of mirror to use as pull-through cache instead. Expected format is 'orignal.registry=new.registry;other-original.registry=other-remap.registry'")
	RootCmd.PersistentFlags().VarP(&opts.RegistryMirrors, "registry-mirror", "", "Registry mirror to use as pull-through cache instead of docker.io. Set it repeatedly for multiple mirrors.")
	RootCmd.PersistentFlags().BoolVarP(&opts.SkipDefaultRegistryFallback, "skip-default-registry-fallback", "", false, "If an image is not found on any mirrors (defined with registry-mirror) do not fallback to the default registry. If registry-mirror is not defined, this flag is ignored.")
	RootCmd.PersistentFlags().BoolVarP(&opts.IgnoreVarRun, "ignore-var-run", "", true, "Ignore /var/run directory when taking image snapshot. Set it to false to preserve /var/run/ in destination image.")
	RootCmd.PersistentFlags().VarP(&opts.Labels, "label", "", "Set metadata for an image. Set it repeatedly for multiple labels.")
	RootCmd.PersistentFlags().BoolVarP(&opts.SkipUnusedStages, "skip-unused-stages", "", false, "Build only used stages if defined to true. Otherwise it builds by default all stages, even the unnecessaries ones until it reaches the target stage / end of Dockerfile")
	RootCmd.PersistentFlags().BoolVarP(&opts.RunV2, "use-new-run", "", false, "Use the experimental run implementation for detecting changes without requiring file system snapshots.")
	RootCmd.PersistentFlags().Var(&opts.Git, "git", "Branch to clone if build context is a git repository")
	RootCmd.PersistentFlags().BoolVarP(&opts.CacheCopyLayers, "cache-copy-layers", "", false, "Caches copy layers")
	RootCmd.PersistentFlags().BoolVarP(&opts.CacheRunLayers, "cache-run-layers", "", true, "Caches run layers")
	RootCmd.PersistentFlags().VarP(&opts.IgnorePaths, "ignore-path", "", "Ignore these paths when taking a snapshot. Set it repeatedly for multiple paths.")
	RootCmd.PersistentFlags().BoolVarP(&opts.ForceBuildMetadata, "force-build-metadata", "", false, "Force add metadata layers to build image")
	RootCmd.PersistentFlags().BoolVarP(&opts.SkipPushPermissionCheck, "skip-push-permission-check", "", false, "Skip check of the push permission")

	// Deprecated flags.
	RootCmd.PersistentFlags().StringVarP(&opts.SnapshotModeDeprecated, "snapshotMode", "", "", "This flag is deprecated. Please use '--snapshot-mode'.")
	RootCmd.PersistentFlags().StringVarP(&opts.CustomPlatformDeprecated, "customPlatform", "", "", "This flag is deprecated. Please use '--custom-platform'.")
	RootCmd.PersistentFlags().StringVarP(&opts.TarPath, "tarPath", "", "", "This flag is deprecated. Please use '--tar-path'.")
}

// addHiddenFlags marks certain flags as hidden from the executor help text
func addHiddenFlags(cmd *cobra.Command) {
	// This flag is added in a vendored directory, hide so that it doesn't come up via --help
	pflag.CommandLine.MarkHidden("azure-container-registry-config")
	// Hide this flag as we want to encourage people to use the --context flag instead
	cmd.PersistentFlags().MarkHidden("bucket")
}

// checkKanikoDir will check whether the executor is operating in the default '/kaniko' directory,
// conducting the relevant operations if it is not
func checkKanikoDir(dir string) error {
	if dir != constants.DefaultKanikoPath {

		// The destination directory may be across a different partition, so we cannot simply rename/move the directory in this case.
		if _, err := util.CopyDir(constants.DefaultKanikoPath, dir, util.FileContext{}, util.DoNotChangeUID, util.DoNotChangeGID, fs.FileMode(0o600), true); err != nil {
			return err
		}

		if err := os.RemoveAll(constants.DefaultKanikoPath); err != nil {
			return err
		}
		// After remove DefaultKankoPath, the DOKCER_CONFIG env will point to a non-exist dir, so we should update DOCKER_CONFIG env to new dir
		if err := os.Setenv("DOCKER_CONFIG", filepath.Join(dir, "/.docker")); err != nil {
			return err
		}
	}
	return nil
}

func checkContained() bool {
	return proc.GetContainerRuntime(0, 0) != proc.RuntimeNotFound
}

// checkNoDeprecatedFlags return an error if deprecated flags are used.
func checkNoDeprecatedFlags() {
	// In version >=2.0.0 make it fail (`Warn` -> `Fatal`)
	if opts.CustomPlatformDeprecated != "" {
		logrus.Warn("Flag --customPlatform is deprecated. Use: --custom-platform")
		opts.CustomPlatform = opts.CustomPlatformDeprecated
	}

	if opts.SnapshotModeDeprecated != "" {
		logrus.Warn("Flag --snapshotMode is deprecated. Use: --snapshot-mode")
		opts.SnapshotMode = opts.SnapshotModeDeprecated
	}

	if opts.TarPathDeprecated != "" {
		logrus.Warn("Flag --tarPath is deprecated. Use: --tar-path")
		opts.TarPath = opts.TarPathDeprecated
	}
}

// cacheFlagsValid makes sure the flags passed in related to caching are valid
func cacheFlagsValid() error {
	if !opts.Cache {
		return nil
	}
	// If --cache=true and --no-push=true, then cache repo must be provided
	// since cache can't be inferred from destination
	if opts.CacheRepo == "" && opts.NoPush {
		return errors.New("if using cache with --no-push, specify cache repo with --cache-repo")
	}
	return nil
}

// resolveDockerfilePath resolves the Dockerfile path to an absolute path
func resolveDockerfilePath() error {
	if isURL(opts.DockerfilePath) {
		return nil
	}
	if util.FilepathExists(opts.DockerfilePath) {
		abs, err := filepath.Abs(opts.DockerfilePath)
		if err != nil {
			return errors.Wrap(err, "getting absolute path for dockerfile")
		}
		opts.DockerfilePath = abs
		return copyDockerfile()
	}
	// Otherwise, check if the path relative to the build context exists
	if util.FilepathExists(filepath.Join(opts.SrcContext, opts.DockerfilePath)) {
		abs, err := filepath.Abs(filepath.Join(opts.SrcContext, opts.DockerfilePath))
		if err != nil {
			return errors.Wrap(err, "getting absolute path for src context/dockerfile path")
		}
		opts.DockerfilePath = abs
		return copyDockerfile()
	}
	return errors.New("please provide a valid path to a Dockerfile within the build context with --dockerfile")
}

// resolveEnvironmentBuildArgs replace build args without value by the same named environment variable
func resolveEnvironmentBuildArgs(arguments []string, resolver func(string) string) {
	for index, argument := range arguments {
		i := strings.Index(argument, "=")
		if i < 0 {
			value := resolver(argument)
			arguments[index] = fmt.Sprintf("%s=%s", argument, value)
		}
	}
}

// copy Dockerfile to /kaniko/Dockerfile so that if it's specified in the .dockerignore
// it won't be copied into the image
func copyDockerfile() error {
	if _, err := util.CopyFile(opts.DockerfilePath, config.DockerfilePath, util.FileContext{}, util.DoNotChangeUID, util.DoNotChangeGID, fs.FileMode(0o600), true); err != nil {
		return errors.Wrap(err, "copying dockerfile")
	}
	dockerignorePath := opts.DockerfilePath + ".dockerignore"
	if util.FilepathExists(dockerignorePath) {
		if _, err := util.CopyFile(dockerignorePath, config.DockerfilePath+".dockerignore", util.FileContext{}, util.DoNotChangeUID, util.DoNotChangeGID, fs.FileMode(0o600), true); err != nil {
			return errors.Wrap(err, "copying Dockerfile.dockerignore")
		}
	}
	opts.DockerfilePath = config.DockerfilePath
	return nil
}

// resolveSourceContext unpacks the source context if it is a tar in a bucket or in kaniko container
// it resets srcContext to be the path to the unpacked build context within the image
func resolveSourceContext() error {
	if opts.SrcContext == "" && opts.Bucket == "" {
		return errors.New("please specify a path to the build context with the --context flag or a bucket with the --bucket flag")
	}
	if opts.SrcContext != "" && !strings.Contains(opts.SrcContext, "://") {
		return nil
	}
	if opts.Bucket != "" {
		if !strings.Contains(opts.Bucket, "://") {
			// if no prefix use Google Cloud Storage as default for backwards compatibility
			opts.SrcContext = constants.GCSBuildContextPrefix + opts.Bucket
		} else {
			opts.SrcContext = opts.Bucket
		}
	}
	contextExecutor, err := buildcontext.GetBuildContext(opts.SrcContext, buildcontext.BuildOptions{
		GitBranch:            opts.Git.Branch,
		GitSingleBranch:      opts.Git.SingleBranch,
		GitRecurseSubmodules: opts.Git.RecurseSubmodules,
		InsecureSkipTLS:      opts.Git.InsecureSkipTLS,
	})
	if err != nil {
		return err
	}
	logrus.Debugf("Getting source context from %s", opts.SrcContext)
	opts.SrcContext, err = contextExecutor.UnpackTarFromBuildContext()
	if err != nil {
		return err
	}
	if ctxSubPath != "" {
		opts.SrcContext = filepath.Join(opts.SrcContext, ctxSubPath)
		if _, err := os.Stat(opts.SrcContext); os.IsNotExist(err) {
			return err
		}
	}
	logrus.Debugf("Build context located at %s", opts.SrcContext)
	return nil
}

func resolveRelativePaths() error {
	optsPaths := []*string{
		&opts.DockerfilePath,
		&opts.SrcContext,
		&opts.CacheDir,
		&opts.TarPath,
		&opts.DigestFile,
		&opts.ImageNameDigestFile,
		&opts.ImageNameTagDigestFile,
	}

	for _, p := range optsPaths {
		if path := *p; shdSkip(path) {
			logrus.Debugf("Skip resolving path %s", path)
			continue
		}

		// Resolve relative path to absolute path
		var err error
		relp := *p // save original relative path
		if *p, err = filepath.Abs(*p); err != nil {
			return errors.Wrapf(err, "Couldn't resolve relative path %s to an absolute path", *p)
		}
		logrus.Debugf("Resolved relative path %s to %s", relp, *p)
	}
	return nil
}

func exit(err error) {
	var execErr *exec.ExitError
	if errors.As(err, &execErr) {
		// if there is an exit code propagate it
		exitWithCode(err, execErr.ExitCode())
	}
	// otherwise exit with catch all 1
	exitWithCode(err, 1)
}

// exits with the given error and exit code
func exitWithCode(err error, exitCode int) {
	fmt.Fprintln(os.Stderr, err)
	os.Exit(exitCode)
}

func isURL(path string) bool {
	if match, _ := regexp.MatchString("^https?://", path); match {
		return true
	}
	return false
}

func shdSkip(path string) bool {
	return path == "" || isURL(path) || filepath.IsAbs(path)
}
