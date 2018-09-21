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
	"os"
	"os/exec"
	"path"
	"path/filepath"
	"runtime"
	"strings"
)

const (
	// ExecutorImage is the name of the kaniko executor image
	ExecutorImage = "executor-image"

	dockerPrefix     = "docker-"
	kanikoPrefix     = "kaniko-"
	buildContextPath = "/workspace"
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

// FindDockerFiles will look for test docker files in the directory dockerfilesPath.
// These files must start with `Dockerfile_test`. If the file is one we are intentionally
// skipping, it will not be included in the returned list.
func FindDockerFiles(dockerfilesPath string) ([]string, error) {
	// TODO: remove test_user_run from this when https://github.com/GoogleContainerTools/container-diff/issues/237 is fixed
	testsToIgnore := map[string]bool{"Dockerfile_test_user_run": false}
	allDockerfiles, err := filepath.Glob(path.Join(dockerfilesPath, "Dockerfile_test*"))
	if err != nil {
		return []string{}, fmt.Errorf("Failed to find docker files at %s: %s", dockerfilesPath, err)
	}

	var dockerfiles []string
	for _, dockerfile := range allDockerfiles {
		// Remove the leading directory from the path
		dockerfile = dockerfile[len("dockerfiles/"):]
		if !testsToIgnore[dockerfile] {
			dockerfiles = append(dockerfiles, dockerfile)
		}
	}
	return dockerfiles, err
}

// DockerFileBuilder knows how to build docker files using both Kaniko and Docker and
// keeps track of which files have been built.
type DockerFileBuilder struct {
	// Holds all available docker files and whether or not they've been built
	FilesBuilt map[string]bool
}

// NewDockerFileBuilder will create a DockerFileBuilder initialized with dockerfiles, which
// it will assume are all as yet unbuilt.
func NewDockerFileBuilder(dockerfiles []string) *DockerFileBuilder {
	d := DockerFileBuilder{FilesBuilt: map[string]bool{}}
	for _, f := range dockerfiles {
		d.FilesBuilt[f] = false
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
	_, err := RunCommandWithoutTest(dockerCmd)
	if err != nil {
		return fmt.Errorf("Failed to build image %s with docker command \"%s\": %s", dockerImage, dockerCmd.Args, err)
	}

	contextFlag := "-c"
	contextPath := buildContextPath
	for _, d := range bucketContextTests {
		if d == dockerfile {
			contextFlag = "-b"
			contextPath = gcsBucket
			break
		}
	}

	reproducibleFlag := ""
	for _, d := range reproducibleTests {
		if d == dockerfile {
			reproducibleFlag = "--reproducible"
			break
		}
	}

	// build kaniko image
	additionalFlags = append(buildArgs, additionalKanikoFlagsMap[dockerfile]...)
	kanikoImage := GetKanikoImage(imageRepo, dockerfile)
	kanikoCmd := exec.Command("docker",
		append([]string{"run",
			"-v", os.Getenv("HOME") + "/.config/gcloud:/root/.config/gcloud",
			"-v", cwd + ":/workspace",
			ExecutorImage,
			"-f", path.Join(buildContextPath, dockerfilesPath, dockerfile),
			"-d", kanikoImage, reproducibleFlag,
			contextFlag, contextPath},
			additionalFlags...)...,
	)

	_, err = RunCommandWithoutTest(kanikoCmd)
	if err != nil {
		return fmt.Errorf("Failed to build image %s with kaniko command \"%s\": %s", dockerImage, kanikoCmd.Args, err)
	}

	d.FilesBuilt[dockerfile] = true
	return nil
}
