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

package integration

import (
	"fmt"
	"io/ioutil"
	"os"
	"os/exec"
	"path"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
	"time"

	"github.com/GoogleContainerTools/kaniko/pkg/timing"
)

const (
	// ExecutorImage is the name of the kaniko executor image
	ExecutorImage = "executor-image"
	//WarmerImage is the name of the kaniko cache warmer image
	WarmerImage = "warmer-image"

	dockerPrefix     = "docker-"
	kanikoPrefix     = "kaniko-"
	buildContextPath = "/workspace"
	cacheDir         = "/workspace/cache"
	baseImageToCache = "gcr.io/google-appengine/debian9@sha256:1d6a9a6d106bd795098f60f4abb7083626354fa6735e81743c7f8cfca11259f0"
)

// Arguments to build Dockerfiles with, used for both docker and kaniko builds
var argsMap = map[string][]string{
	"Dockerfile_test_run":     {"file=/file"},
	"Dockerfile_test_workdir": {"workdir=/arg/workdir"},
	"Dockerfile_test_add":     {"file=context/foo"},
	"Dockerfile_test_onbuild": {"file=/tmp/onbuild"},
	"Dockerfile_test_scratch": {
		"image=scratch",
		"hello=hello-value",
		"file=context/foo",
		"file3=context/b*",
	},
	"Dockerfile_test_multistage": {"file=/foo2"},
}

// Arguments to build Dockerfiles with when building with docker
var additionalDockerFlagsMap = map[string][]string{
	"Dockerfile_test_target": {"--target=second"},
}

// Arguments to build Dockerfiles with when building with kaniko
var additionalKanikoFlagsMap = map[string][]string{
	"Dockerfile_test_add":     {"--single-snapshot"},
	"Dockerfile_test_scratch": {"--single-snapshot"},
	"Dockerfile_test_target":  {"--target=second"},
}

var bucketContextTests = []string{"Dockerfile_test_copy_bucket"}
var reproducibleTests = []string{"Dockerfile_test_reproducible"}

// GetDockerImage constructs the name of the docker image that would be built with
// dockerfile if it was tagged with imageRepo.
func GetDockerImage(imageRepo, dockerfile string) string {
	return strings.ToLower(imageRepo + dockerPrefix + dockerfile)
}

// GetKanikoImage constructs the name of the kaniko image that would be built with
// dockerfile if it was tagged with imageRepo.
func GetKanikoImage(imageRepo, dockerfile string) string {
	return strings.ToLower(imageRepo + kanikoPrefix + dockerfile)
}

// GetVersionedKanikoImage versions constructs the name of the kaniko image that would be built
// with the dockerfile and versions it for cache testing
func GetVersionedKanikoImage(imageRepo, dockerfile string, version int) string {
	return strings.ToLower(imageRepo + kanikoPrefix + dockerfile + strconv.Itoa(version))
}

// FindDockerFiles will look for test docker files in the directory dockerfilesPath.
// These files must start with `Dockerfile_test`. If the file is one we are intentionally
// skipping, it will not be included in the returned list.
func FindDockerFiles(dockerfilesPath string) ([]string, error) {
	allDockerfiles, err := filepath.Glob(path.Join(dockerfilesPath, "Dockerfile_test*"))
	if err != nil {
		return []string{}, fmt.Errorf("Failed to find docker files at %s: %s", dockerfilesPath, err)
	}

	var dockerfiles []string
	for _, dockerfile := range allDockerfiles {
		// Remove the leading directory from the path
		dockerfile = dockerfile[len("dockerfiles/"):]
		dockerfiles = append(dockerfiles, dockerfile)

	}
	return dockerfiles, err
}

// DockerFileBuilder knows how to build docker files using both Kaniko and Docker and
// keeps track of which files have been built.
type DockerFileBuilder struct {
	// Holds all available docker files and whether or not they've been built
	FilesBuilt           map[string]bool
	DockerfilesToIgnore  map[string]struct{}
	TestCacheDockerfiles map[string]struct{}
}

// NewDockerFileBuilder will create a DockerFileBuilder initialized with dockerfiles, which
// it will assume are all as yet unbuilt.
func NewDockerFileBuilder(dockerfiles []string) *DockerFileBuilder {
	d := DockerFileBuilder{FilesBuilt: map[string]bool{}}
	for _, f := range dockerfiles {
		d.FilesBuilt[f] = false
	}
	d.DockerfilesToIgnore = map[string]struct{}{
		// TODO: remove test_user_run from this when https://github.com/GoogleContainerTools/container-diff/issues/237 is fixed
		"Dockerfile_test_user_run": {},
	}
	d.TestCacheDockerfiles = map[string]struct{}{
		"Dockerfile_test_cache":         {},
		"Dockerfile_test_cache_install": {},
		"Dockerfile_test_cache_perm":    {},
	}
	return &d
}

// BuildImage will build dockerfile (located at dockerfilesPath) using both kaniko and docker.
// The resulting image will be tagged with imageRepo. If the dockerfile will be built with
// context (i.e. it is in `buildContextTests`) the context will be pulled from gcsBucket.
func (d *DockerFileBuilder) BuildImage(imageRepo, gcsBucket, dockerfilesPath, dockerfile string) error {
	_, ex, _, _ := runtime.Caller(0)
	cwd := filepath.Dir(ex)

	fmt.Printf("Building images for Dockerfile %s\n", dockerfile)

	var buildArgs []string
	buildArgFlag := "--build-arg"
	for _, arg := range argsMap[dockerfile] {
		buildArgs = append(buildArgs, buildArgFlag)
		buildArgs = append(buildArgs, arg)
	}
	// build docker image
	additionalFlags := append(buildArgs, additionalDockerFlagsMap[dockerfile]...)
	dockerImage := strings.ToLower(imageRepo + dockerPrefix + dockerfile)
	dockerCmd := exec.Command("docker",
		append([]string{"build",
			"-t", dockerImage,
			"-f", path.Join(dockerfilesPath, dockerfile),
			"."},
			additionalFlags...)...,
	)

	timer := timing.Start(dockerfile + "_docker")
	out, err := RunCommandWithoutTest(dockerCmd)
	timing.DefaultRun.Stop(timer)
	if err != nil {
		return fmt.Errorf("Failed to build image %s with docker command \"%s\": %s %s", dockerImage, dockerCmd.Args, err, string(out))
	}

	contextFlag := "-c"
	contextPath := buildContextPath
	for _, d := range bucketContextTests {
		if d == dockerfile {
			contextFlag = "-b"
			contextPath = gcsBucket
		}
	}

	reproducibleFlag := ""
	for _, d := range reproducibleTests {
		if d == dockerfile {
			reproducibleFlag = "--reproducible"
			break
		}
	}

	benchmarkEnv := "BENCHMARK_FILE=false"
	benchmarkDir, err := ioutil.TempDir("", "")
	if err != nil {
		return err
	}
	if b, err := strconv.ParseBool(os.Getenv("BENCHMARK")); err == nil && b {
		benchmarkEnv = "BENCHMARK_FILE=/kaniko/benchmarks/" + dockerfile
		benchmarkFile := path.Join(benchmarkDir, dockerfile)
		fileName := fmt.Sprintf("run_%s_%s", time.Now().Format("2006-01-02-15:04"), dockerfile)
		dst := path.Join("benchmarks", fileName)
		defer UploadFileToBucket(gcsBucket, benchmarkFile, dst)
	}

	// build kaniko image
	additionalFlags = append(buildArgs, additionalKanikoFlagsMap[dockerfile]...)
	kanikoImage := GetKanikoImage(imageRepo, dockerfile)
	kanikoCmd := exec.Command("docker",
		append([]string{"run",
			"-v", os.Getenv("HOME") + "/.config/gcloud:/root/.config/gcloud",
			"-v", benchmarkDir + ":/kaniko/benchmarks",
			"-v", cwd + ":/workspace",
			"-e", benchmarkEnv,
			ExecutorImage,
			"-f", path.Join(buildContextPath, dockerfilesPath, dockerfile),
			"-d", kanikoImage, reproducibleFlag,
			contextFlag, contextPath},
			additionalFlags...)...,
	)

	timer = timing.Start(dockerfile + "_kaniko")
	out, err = RunCommandWithoutTest(kanikoCmd)
	timing.DefaultRun.Stop(timer)
	if err != nil {
		return fmt.Errorf("Failed to build image %s with kaniko command \"%s\": %s %s", dockerImage, kanikoCmd.Args, err, string(out))
	}

	d.FilesBuilt[dockerfile] = true
	return nil
}

func populateVolumeCache() error {
	_, ex, _, _ := runtime.Caller(0)
	cwd := filepath.Dir(ex)
	warmerCmd := exec.Command("docker",
		append([]string{"run",
			"-v", os.Getenv("HOME") + "/.config/gcloud:/root/.config/gcloud",
			"-v", cwd + ":/workspace",
			WarmerImage,
			"-c", cacheDir,
			"-i", baseImageToCache},
		)...,
	)

	if _, err := RunCommandWithoutTest(warmerCmd); err != nil {
		return fmt.Errorf("Failed to warm kaniko cache: %s", err)
	}

	return nil
}

// buildCachedImages builds the images for testing caching via kaniko where version is the nth time this image has been built
func (d *DockerFileBuilder) buildCachedImages(imageRepo, cacheRepo, dockerfilesPath string, version int) error {
	_, ex, _, _ := runtime.Caller(0)
	cwd := filepath.Dir(ex)

	cacheFlag := "--cache=true"

	for dockerfile := range d.TestCacheDockerfiles {
		benchmarkEnv := "BENCHMARK_FILE=false"
		if b, err := strconv.ParseBool(os.Getenv("BENCHMARK")); err == nil && b {
			os.Mkdir("benchmarks", 0755)
			benchmarkEnv = "BENCHMARK_FILE=/workspace/benchmarks/" + dockerfile
		}
		kanikoImage := GetVersionedKanikoImage(imageRepo, dockerfile, version)
		kanikoCmd := exec.Command("docker",
			append([]string{"run",
				"-v", os.Getenv("HOME") + "/.config/gcloud:/root/.config/gcloud",
				"-v", cwd + ":/workspace",
				"-e", benchmarkEnv,
				ExecutorImage,
				"-f", path.Join(buildContextPath, dockerfilesPath, dockerfile),
				"-d", kanikoImage,
				"-c", buildContextPath,
				cacheFlag,
				"--cache-repo", cacheRepo,
				"--cache-dir", cacheDir})...,
		)

		timer := timing.Start(dockerfile + "_kaniko_cached_" + strconv.Itoa(version))
		_, err := RunCommandWithoutTest(kanikoCmd)
		timing.DefaultRun.Stop(timer)
		if err != nil {
			return fmt.Errorf("Failed to build cached image %s with kaniko command \"%s\": %s", kanikoImage, kanikoCmd.Args, err)
		}
	}
	return nil
}

// buildRelativePathsImage builds the images for testing passing relatives paths to Kaniko
func (d *DockerFileBuilder) buildRelativePathsImage(imageRepo, dockerfile string) error {
	_, ex, _, _ := runtime.Caller(0)
	cwd := filepath.Dir(ex)

	buildContextPath := "./relative-subdirectory"
	kanikoImage := GetKanikoImage(imageRepo, dockerfile)

	kanikoCmd := exec.Command("docker",
		append([]string{"run",
			"-v", os.Getenv("HOME") + "/.config/gcloud:/root/.config/gcloud",
			"-v", cwd + ":/workspace",
			ExecutorImage,
			"-f", dockerfile,
			"-d", kanikoImage,
			"--digest-file", "./digest",
			"-c", buildContextPath,
		})...,
	)

	timer := timing.Start(dockerfile + "_kaniko_relative_paths")
	_, err := RunCommandWithoutTest(kanikoCmd)
	timing.DefaultRun.Stop(timer)
	if err != nil {
		return fmt.Errorf("Failed to build relative path image %s with kaniko command \"%s\": %s", kanikoImage, kanikoCmd.Args, err)
	}

	return nil
}
