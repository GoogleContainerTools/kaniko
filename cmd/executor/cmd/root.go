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
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"time"

	"github.com/GoogleContainerTools/kaniko/pkg/buildcontext"
	"github.com/GoogleContainerTools/kaniko/pkg/config"
	"github.com/GoogleContainerTools/kaniko/pkg/constants"
	"github.com/GoogleContainerTools/kaniko/pkg/executor"
	"github.com/GoogleContainerTools/kaniko/pkg/logging"
	"github.com/GoogleContainerTools/kaniko/pkg/timing"
	"github.com/GoogleContainerTools/kaniko/pkg/util"
	"github.com/genuinetools/amicontained/container"
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
}

// RootCmd is the kaniko command that is run
var RootCmd = &cobra.Command{
	Use: "executor",
	PersistentPreRunE: func(cmd *cobra.Command, args []string) error {
		if cmd.Use == "executor" {
			resolveEnvironmentBuildArgs(opts.BuildArgs, os.Getenv)

			if err := logging.Configure(logLevel, logFormat, logTimestamp); err != nil {
				return err
			}

			if !opts.NoPush && len(opts.Destinations) == 0 {
				return errors.New("You must provide --destination, or use --no-push")
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
				return errors.New("You must provide --destination if setting ImageNameDigestFile")
			}
			// Update ignored paths
			util.UpdateInitialIgnoreList(opts.IgnoreVarRun)
		}
		return nil
	},
	Run: func(cmd *cobra.Command, args []string) {
		if !checkContained() {
			if !force {
				exit(errors.New("kaniko should only be run inside of a container, run with the --force flag if you are sure you want to continue"))
			}
			logrus.Warn("kaniko is being run outside of a container. This can have dangerous effects on your system")
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
				logrus.Info("uploading to gcs")
				if err := buildcontext.UploadToBucket(strings.NewReader(s), benchmarkFile); err != nil {
					logrus.Infof("Unable to upload %s due to %v", benchmarkFile, err)
				}
				logrus.Infof("benchmark file written at %s", benchmarkFile)
			} else {
				f, err := os.Create(benchmarkFile)
				if err != nil {
					logrus.Warnf("Unable to create benchmarking file %s: %s", benchmarkFile, err)
					return
				}
				defer f.Close()
				f.WriteString(s)
				logrus.Infof("benchmark file written at %s", benchmarkFile)
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
	RootCmd.PersistentFlags().StringVarP(&opts.SnapshotMode, "snapshotMode", "", "full", "Change the file attributes inspected during snapshotting")
	RootCmd.PersistentFlags().VarP(&opts.BuildArgs, "build-arg", "", "This flag allows you to pass in ARG values at build time. Set it repeatedly for multiple values.")
	RootCmd.PersistentFlags().BoolVarP(&opts.Insecure, "insecure", "", false, "Push to insecure registry using plain HTTP")
	RootCmd.PersistentFlags().BoolVarP(&opts.SkipTLSVerify, "skip-tls-verify", "", false, "Push to insecure registry ignoring TLS verify")
	RootCmd.PersistentFlags().BoolVarP(&opts.InsecurePull, "insecure-pull", "", false, "Pull from insecure registry using plain HTTP")
	RootCmd.PersistentFlags().BoolVarP(&opts.SkipTLSVerifyPull, "skip-tls-verify-pull", "", false, "Pull from insecure registry ignoring TLS verify")
	RootCmd.PersistentFlags().StringVarP(&opts.TarPath, "tarPath", "", "", "Path to save the image in as a tarball instead of pushing")
	RootCmd.PersistentFlags().BoolVarP(&opts.SingleSnapshot, "single-snapshot", "", false, "Take a single snapshot at the end of the build.")
	RootCmd.PersistentFlags().BoolVarP(&opts.Reproducible, "reproducible", "", false, "Strip timestamps out of the image to make it reproducible")
	RootCmd.PersistentFlags().StringVarP(&opts.Target, "target", "", "", "Set the target build stage to build")
	RootCmd.PersistentFlags().BoolVarP(&opts.NoPush, "no-push", "", false, "Do not push the image to the registry")
	RootCmd.PersistentFlags().StringVarP(&opts.CacheRepo, "cache-repo", "", "", "Specify a repository to use as a cache, otherwise one will be inferred from the destination provided")
	RootCmd.PersistentFlags().StringVarP(&opts.CacheDir, "cache-dir", "", "/cache", "Specify a local directory to use as a cache.")
	RootCmd.PersistentFlags().StringVarP(&opts.DigestFile, "digest-file", "", "", "Specify a file to save the digest of the built image to.")
	RootCmd.PersistentFlags().StringVarP(&opts.ImageNameDigestFile, "image-name-with-digest-file", "", "", "Specify a file to save the image name w/ digest of the built image to.")
	RootCmd.PersistentFlags().StringVarP(&opts.OCILayoutPath, "oci-layout-path", "", "", "Path to save the OCI image layout of the built image.")
	RootCmd.PersistentFlags().BoolVarP(&opts.Cache, "cache", "", false, "Use cache when building image")
	RootCmd.PersistentFlags().BoolVarP(&opts.Cleanup, "cleanup", "", false, "Clean the filesystem at the end")
	RootCmd.PersistentFlags().DurationVarP(&opts.CacheTTL, "cache-ttl", "", time.Hour*336, "Cache timeout in hours. Defaults to two weeks.")
	RootCmd.PersistentFlags().VarP(&opts.InsecureRegistries, "insecure-registry", "", "Insecure registry using plain HTTP to push and pull. Set it repeatedly for multiple registries.")
	RootCmd.PersistentFlags().VarP(&opts.SkipTLSVerifyRegistries, "skip-tls-verify-registry", "", "Insecure registry ignoring TLS verify to push and pull. Set it repeatedly for multiple registries.")
	opts.RegistriesCertificates = make(map[string]string)
	RootCmd.PersistentFlags().VarP(&opts.RegistriesCertificates, "registry-certificate", "", "Use the provided certificate for TLS communication with the given registry. Expected format is 'my.registry.url=/path/to/the/server/certificate'.")
	RootCmd.PersistentFlags().StringVarP(&opts.RegistryMirror, "registry-mirror", "", "", "Registry mirror to use has pull-through cache instead of docker.io.")
	RootCmd.PersistentFlags().BoolVarP(&opts.IgnoreVarRun, "whitelist-var-run", "", true, "Ignore /var/run directory when taking image snapshot. Set it to false to preserve /var/run/ in destination image. (Default true).")
	RootCmd.PersistentFlags().VarP(&opts.Labels, "label", "", "Set metadata for an image. Set it repeatedly for multiple labels.")
	RootCmd.PersistentFlags().BoolVarP(&opts.SkipUnusedStages, "skip-unused-stages", "", false, "Build only used stages if defined to true. Otherwise it builds by default all stages, even the unnecessaries ones until it reaches the target stage / end of Dockerfile")
	RootCmd.PersistentFlags().BoolVarP(&opts.RunV2, "use-new-run", "", false, "Use the experimental run implementation for detecting changes without requiring file system snapshots.")
	RootCmd.PersistentFlags().Var(&opts.Git, "git", "Branch to clone if build context is a git repository")
}

// addHiddenFlags marks certain flags as hidden from the executor help text
func addHiddenFlags(cmd *cobra.Command) {
	// This flag is added in a vendored directory, hide so that it doesn't come up via --help
	pflag.CommandLine.MarkHidden("azure-container-registry-config")
	// Hide this flag as we want to encourage people to use the --context flag instead
	cmd.PersistentFlags().MarkHidden("bucket")
}

func checkContained() bool {
	_, err := container.DetectRuntime()
	return err == nil
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
	if _, err := util.CopyFile(opts.DockerfilePath, constants.DockerfilePath, util.FileContext{}, util.DoNotChangeUID, util.DoNotChangeGID); err != nil {
		return errors.Wrap(err, "copying dockerfile")
	}
	opts.DockerfilePath = constants.DockerfilePath
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
	fmt.Println(err)
	os.Exit(1)
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
