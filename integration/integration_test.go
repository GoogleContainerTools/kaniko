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
	"archive/tar"
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"log"
	"math"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/google/go-containerregistry/pkg/name"
	"github.com/google/go-containerregistry/pkg/v1/daemon"
	"github.com/google/go-containerregistry/pkg/v1/remote"
	"github.com/pkg/errors"
	"google.golang.org/api/option"

	"github.com/GoogleContainerTools/kaniko/pkg/timing"
	"github.com/GoogleContainerTools/kaniko/pkg/util"
	"github.com/GoogleContainerTools/kaniko/pkg/util/bucket"
	"github.com/GoogleContainerTools/kaniko/testutil"
)

var (
	config         *integrationTestConfig
	imageBuilder   *DockerFileBuilder
	allDockerfiles []string
)

const (
	daemonPrefix       = "daemon://"
	integrationPath    = "integration"
	dockerfilesPath    = "dockerfiles"
	emptyContainerDiff = `[
     {
       "Image1": "%s",
       "Image2": "%s",
       "DiffType": "File",
       "Diff": {
	 	"Adds": null,
	 	"Dels": null,
	 	"Mods": null
       }
     },
     {
       "Image1": "%s",
       "Image2": "%s",
       "DiffType": "Metadata",
       "Diff": {
	 	"Adds": [],
	 	"Dels": []
       }
     }
   ]`
)

func getDockerMajorVersion() int {
	out, err := exec.Command("docker", "version", "--format", "{{.Server.Version}}").Output()
	if err != nil {
		log.Fatal("Error getting docker version of server:", err)
	}
	versionArr := strings.Split(string(out), ".")

	ver, err := strconv.Atoi(versionArr[0])
	if err != nil {
		log.Fatal("Error getting docker version of server during parsing version string:", err)
	}
	return ver
}

func launchTests(m *testing.M) (int, error) {
	if config.isGcrRepository() {
		contextFilePath, err := CreateIntegrationTarball()
		if err != nil {
			return 1, errors.Wrap(err, "Failed to create tarball of integration files for build context")
		}

		bucketName, item, err := bucket.GetNameAndFilepathFromURI(config.gcsBucket)
		if err != nil {
			return 1, errors.Wrap(err, "failed to get bucket name from uri")
		}
		contextFile, err := os.Open(contextFilePath)
		if err != nil {
			return 1, fmt.Errorf("failed to read file at path %v: %w", contextFilePath, err)
		}
		err = bucket.Upload(context.Background(), bucketName, item, contextFile, config.gcsClient)
		if err != nil {
			return 1, errors.Wrap(err, "Failed to upload build context")
		}

		if err = os.Remove(contextFilePath); err != nil {
			return 1, errors.Wrap(err, fmt.Sprintf("Failed to remove tarball at %s", contextFilePath))
		}

		deleteFunc := func() {
			bucket.Delete(context.Background(), bucketName, item, config.gcsClient)
		}
		RunOnInterrupt(deleteFunc)
		defer deleteFunc()
	}
	if err := buildRequiredImages(); err != nil {
		return 1, errors.Wrap(err, "Error while building images")
	}

	imageBuilder = NewDockerFileBuilder()

	return m.Run(), nil
}

func TestMain(m *testing.M) {
	var err error
	if !meetsRequirements() {
		fmt.Println("Missing required tools")
		os.Exit(1)
	}

	config = initIntegrationTestConfig()
	if allDockerfiles, err = FindDockerFiles(dockerfilesPath, config.dockerfilesPattern); err != nil {
		fmt.Println("Coudn't create map of dockerfiles", err)
		os.Exit(1)
	}

	exitCode, err := launchTests(m)
	if err != nil {
		fmt.Println(err)
	}
	os.Exit(exitCode)
}

func buildRequiredImages() error {
	setupCommands := []struct {
		name    string
		command []string
	}{{
		name:    "Building kaniko image",
		command: []string{"docker", "build", "-t", ExecutorImage, "-f", "../deploy/Dockerfile", "--target", "kaniko-executor", ".."},
	}, {
		name:    "Building cache warmer image",
		command: []string{"docker", "build", "-t", WarmerImage, "-f", "../deploy/Dockerfile", "--target", "kaniko-warmer", ".."},
	}, {
		name:    "Building onbuild base image",
		command: []string{"docker", "build", "-t", config.onbuildBaseImage, "-f", fmt.Sprintf("%s/Dockerfile_onbuild_base", dockerfilesPath), "."},
	}, {
		name:    "Pushing onbuild base image",
		command: []string{"docker", "push", config.onbuildBaseImage},
	}, {
		name:    "Building hardlink base image",
		command: []string{"docker", "build", "-t", config.hardlinkBaseImage, "-f", fmt.Sprintf("%s/Dockerfile_hardlink_base", dockerfilesPath), "."},
	}, {
		name:    "Pushing hardlink base image",
		command: []string{"docker", "push", config.hardlinkBaseImage},
	}}

	for _, setupCmd := range setupCommands {
		fmt.Println(setupCmd.name)
		cmd := exec.Command(setupCmd.command[0], setupCmd.command[1:]...)
		cmd.Env = append(os.Environ(), "DOCKER_BUILDKIT=1") // Build with buildkit enabled.
		if out, err := RunCommandWithoutTest(cmd); err != nil {
			return errors.Wrap(err, fmt.Sprintf("%s failed: %s", setupCmd.name, string(out)))
		}
	}
	return nil
}

func TestRun(t *testing.T) {
	for _, dockerfile := range allDockerfiles {
		t.Run("test_"+dockerfile, func(t *testing.T) {
			dockerfile := dockerfile
			t.Parallel()
			if _, ok := imageBuilder.DockerfilesToIgnore[dockerfile]; ok {
				t.SkipNow()
			}
			if _, ok := imageBuilder.TestCacheDockerfiles[dockerfile]; ok {
				t.SkipNow()
			}

			buildImage(t, dockerfile, imageBuilder)

			dockerImage := GetDockerImage(config.imageRepo, dockerfile)
			kanikoImage := GetKanikoImage(config.imageRepo, dockerfile)

			diff := containerDiff(t, daemonPrefix+dockerImage, kanikoImage, "--no-cache")

			expected := fmt.Sprintf(emptyContainerDiff, dockerImage, kanikoImage, dockerImage, kanikoImage)
			checkContainerDiffOutput(t, diff, expected)
		})
	}

	err := logBenchmarks("benchmark")
	if err != nil {
		t.Logf("Failed to create benchmark file: %v", err)
	}
}

func getBranchCommitAndURL() (branch, commit, url string) {
	repo := os.Getenv("GITHUB_REPOSITORY")
	commit = os.Getenv("GITHUB_SHA")
	if _, isPR := os.LookupEnv("GITHUB_HEAD_REF"); isPR {
		branch = "main"
	} else {
		branch = os.Getenv("GITHUB_REF")
		log.Printf("GITHUB_HEAD_REF is unset (not a PR); using GITHUB_REF=%q", branch)
		branch = strings.TrimPrefix(branch, "refs/heads/")
	}
	if repo == "" {
		repo = "GoogleContainerTools/kaniko"
	}
	if branch == "" {
		branch = "main"
	}
	log.Printf("repo=%q / commit=%q / branch=%q", repo, commit, branch)
	url = "github.com/" + repo
	return
}

func getGitRepo(explicit bool) string {
	branch, commit, url := getBranchCommitAndURL()
	if explicit && commit != "" {
		return url + "#" + commit
	}
	return url + "#refs/heads/" + branch
}

func testGitBuildcontextHelper(t *testing.T, repo string) {
	t.Log("testGitBuildcontextHelper repo", repo)
	dockerfile := fmt.Sprintf("%s/%s/Dockerfile_test_run_2", integrationPath, dockerfilesPath)

	// Build with docker
	dockerImage := GetDockerImage(config.imageRepo, "Dockerfile_test_git")
	dockerCmd := exec.Command("docker",
		append([]string{
			"build",
			"-t", dockerImage,
			"-f", dockerfile,
			repo,
		})...)
	out, err := RunCommandWithoutTest(dockerCmd)
	if err != nil {
		t.Errorf("Failed to build image %s with docker command %q: %s %s", dockerImage, dockerCmd.Args, err, string(out))
	}

	// Build with kaniko
	kanikoImage := GetKanikoImage(config.imageRepo, "Dockerfile_test_git")
	dockerRunFlags := []string{"run", "--net=host"}
	dockerRunFlags = addServiceAccountFlags(dockerRunFlags, config.serviceAccount)
	dockerRunFlags = append(dockerRunFlags, ExecutorImage,
		"-f", dockerfile,
		"-d", kanikoImage,
		"-c", fmt.Sprintf("git://%s", repo))

	kanikoCmd := exec.Command("docker", dockerRunFlags...)

	out, err = RunCommandWithoutTest(kanikoCmd)
	if err != nil {
		t.Errorf("Failed to build image %s with kaniko command %q: %v %s", dockerImage, kanikoCmd.Args, err, string(out))
	}

	diff := containerDiff(t, daemonPrefix+dockerImage, kanikoImage, "--no-cache")

	expected := fmt.Sprintf(emptyContainerDiff, dockerImage, kanikoImage, dockerImage, kanikoImage)
	checkContainerDiffOutput(t, diff, expected)
}

// TestGitBuildcontext explicitly names the main branch
// Example:
//
//	git://github.com/myuser/repo#refs/heads/main
func TestGitBuildcontext(t *testing.T) {
	repo := getGitRepo(false)
	testGitBuildcontextHelper(t, repo)
}

// TestGitBuildcontextNoRef builds without any commit / branch reference
// Example:
//
//	git://github.com/myuser/repo
func TestGitBuildcontextNoRef(t *testing.T) {
	t.Skip("Docker's behavior is to assume a 'master' branch, which the Kaniko repo doesn't have")
	_, _, url := getBranchCommitAndURL()
	testGitBuildcontextHelper(t, url)
}

// TestGitBuildcontextExplicitCommit uses an explicit commit hash instead of named reference
// Example:
//
//	git://github.com/myuser/repo#b873088c4a7b60bb7e216289c58da945d0d771b6
func TestGitBuildcontextExplicitCommit(t *testing.T) {
	repo := getGitRepo(true)
	testGitBuildcontextHelper(t, repo)
}

func TestGitBuildcontextSubPath(t *testing.T) {
	repo := getGitRepo(false)
	dockerfile := "Dockerfile_test_run_2"

	// Build with docker
	dockerImage := GetDockerImage(config.imageRepo, "Dockerfile_test_git")
	dockerCmd := exec.Command("docker",
		append([]string{
			"build",
			"-t", dockerImage,
			"-f", filepath.Join(integrationPath, dockerfilesPath, dockerfile),
			repo,
		})...)
	out, err := RunCommandWithoutTest(dockerCmd)
	if err != nil {
		t.Errorf("Failed to build image %s with docker command %q: %s %s", dockerImage, dockerCmd.Args, err, string(out))
	}

	// Build with kaniko
	kanikoImage := GetKanikoImage(config.imageRepo, "Dockerfile_test_git")
	dockerRunFlags := []string{"run", "--net=host"}
	dockerRunFlags = addServiceAccountFlags(dockerRunFlags, config.serviceAccount)
	dockerRunFlags = append(
		dockerRunFlags,
		ExecutorImage,
		"-f", dockerfile,
		"-d", kanikoImage,
		"-c", fmt.Sprintf("git://%s", repo),
		"--context-sub-path", filepath.Join(integrationPath, dockerfilesPath),
	)

	kanikoCmd := exec.Command("docker", dockerRunFlags...)

	out, err = RunCommandWithoutTest(kanikoCmd)
	if err != nil {
		t.Errorf("Failed to build image %s with kaniko command %q: %v %s", dockerImage, kanikoCmd.Args, err, string(out))
	}

	diff := containerDiff(t, daemonPrefix+dockerImage, kanikoImage, "--no-cache")

	expected := fmt.Sprintf(emptyContainerDiff, dockerImage, kanikoImage, dockerImage, kanikoImage)
	checkContainerDiffOutput(t, diff, expected)
}

func TestBuildViaRegistryMirrors(t *testing.T) {
	repo := getGitRepo(false)
	dockerfile := fmt.Sprintf("%s/%s/Dockerfile_registry_mirror", integrationPath, dockerfilesPath)

	// Build with docker
	dockerImage := GetDockerImage(config.imageRepo, "Dockerfile_registry_mirror")
	dockerCmd := exec.Command("docker",
		append([]string{
			"build",
			"-t", dockerImage,
			"-f", dockerfile,
			repo,
		})...)
	out, err := RunCommandWithoutTest(dockerCmd)
	if err != nil {
		t.Errorf("Failed to build image %s with docker command %q: %s %s", dockerImage, dockerCmd.Args, err, string(out))
	}

	// Build with kaniko
	kanikoImage := GetKanikoImage(config.imageRepo, "Dockerfile_registry_mirror")
	dockerRunFlags := []string{"run", "--net=host"}
	dockerRunFlags = addServiceAccountFlags(dockerRunFlags, config.serviceAccount)
	dockerRunFlags = append(dockerRunFlags, ExecutorImage,
		"-f", dockerfile,
		"-d", kanikoImage,
		"--registry-mirror", "doesnotexist.example.com",
		"--registry-mirror", "us-mirror.gcr.io",
		"-c", fmt.Sprintf("git://%s", repo))

	kanikoCmd := exec.Command("docker", dockerRunFlags...)

	out, err = RunCommandWithoutTest(kanikoCmd)
	if err != nil {
		t.Errorf("Failed to build image %s with kaniko command %q: %v %s", dockerImage, kanikoCmd.Args, err, string(out))
	}

	diff := containerDiff(t, daemonPrefix+dockerImage, kanikoImage, "--no-cache")

	expected := fmt.Sprintf(emptyContainerDiff, dockerImage, kanikoImage, dockerImage, kanikoImage)
	checkContainerDiffOutput(t, diff, expected)
}

func TestBuildViaRegistryMap(t *testing.T) {
	repo := getGitRepo(false)
	dockerfile := fmt.Sprintf("%s/%s/Dockerfile_registry_mirror", integrationPath, dockerfilesPath)

	// Build with docker
	dockerImage := GetDockerImage(config.imageRepo, "Dockerfile_registry_mirror")
	dockerCmd := exec.Command("docker",
		append([]string{
			"build",
			"-t", dockerImage,
			"-f", dockerfile,
			repo,
		})...)
	out, err := RunCommandWithoutTest(dockerCmd)
	if err != nil {
		t.Errorf("Failed to build image %s with docker command %q: %s %s", dockerImage, dockerCmd.Args, err, string(out))
	}

	// Build with kaniko
	kanikoImage := GetKanikoImage(config.imageRepo, "Dockerfile_registry_mirror")
	dockerRunFlags := []string{"run", "--net=host"}
	dockerRunFlags = addServiceAccountFlags(dockerRunFlags, config.serviceAccount)
	dockerRunFlags = append(dockerRunFlags, ExecutorImage,
		"-f", dockerfile,
		"-d", kanikoImage,
		"--registry-map", "index.docker.io=doesnotexist.example.com",
		"--registry-map", "index.docker.io=us-mirror.gcr.io",
		"-c", fmt.Sprintf("git://%s", repo))

	kanikoCmd := exec.Command("docker", dockerRunFlags...)

	out, err = RunCommandWithoutTest(kanikoCmd)
	if err != nil {
		t.Errorf("Failed to build image %s with kaniko command %q: %v %s", dockerImage, kanikoCmd.Args, err, string(out))
	}

	diff := containerDiff(t, daemonPrefix+dockerImage, kanikoImage, "--no-cache")

	expected := fmt.Sprintf(emptyContainerDiff, dockerImage, kanikoImage, dockerImage, kanikoImage)
	checkContainerDiffOutput(t, diff, expected)
}

func TestBuildSkipFallback(t *testing.T) {
	repo := getGitRepo(false)
	dockerfile := fmt.Sprintf("%s/%s/Dockerfile_registry_mirror", integrationPath, dockerfilesPath)

	// Build with kaniko
	kanikoImage := GetKanikoImage(config.imageRepo, "Dockerfile_registry_mirror")
	dockerRunFlags := []string{"run", "--net=host"}
	dockerRunFlags = addServiceAccountFlags(dockerRunFlags, config.serviceAccount)
	dockerRunFlags = append(dockerRunFlags, ExecutorImage,
		"-f", dockerfile,
		"-d", kanikoImage,
		"--registry-mirror", "doesnotexist.example.com",
		"--skip-default-registry-fallback",
		"-c", fmt.Sprintf("git://%s", repo))

	kanikoCmd := exec.Command("docker", dockerRunFlags...)

	_, err := RunCommandWithoutTest(kanikoCmd)
	if err == nil {
		t.Errorf("Build should fail after using skip-default-registry-fallback and registry-mirror fail to pull")
	}
}

// TestKanikoDir tests that a build that sets --kaniko-dir produces the same output as the equivalent docker build.
func TestKanikoDir(t *testing.T) {
	repo := getGitRepo(false)
	dockerfile := fmt.Sprintf("%s/%s/Dockerfile_registry_mirror", integrationPath, dockerfilesPath)

	// Build with docker
	dockerImage := GetDockerImage(config.imageRepo, "Dockerfile_registry_mirror")
	dockerCmd := exec.Command("docker",
		append([]string{
			"build",
			"-t", dockerImage,
			"-f", dockerfile,
			repo,
		})...)
	out, err := RunCommandWithoutTest(dockerCmd)
	if err != nil {
		t.Errorf("Failed to build image %s with docker command %q: %s %s", dockerImage, dockerCmd.Args, err, string(out))
	}

	// Build with kaniko
	kanikoImage := GetKanikoImage(config.imageRepo, "Dockerfile_registry_mirror")
	dockerRunFlags := []string{"run", "--net=host"}
	dockerRunFlags = addServiceAccountFlags(dockerRunFlags, config.serviceAccount)
	dockerRunFlags = append(dockerRunFlags, ExecutorImage,
		"-f", dockerfile,
		"-d", kanikoImage,
		"--kaniko-dir", "/not-kaniko",
		"-c", fmt.Sprintf("git://%s", repo))

	kanikoCmd := exec.Command("docker", dockerRunFlags...)

	out, err = RunCommandWithoutTest(kanikoCmd)
	if err != nil {
		t.Errorf("Failed to build image %s with kaniko command %q: %v %s", dockerImage, kanikoCmd.Args, err, string(out))
	}

	diff := containerDiff(t, daemonPrefix+dockerImage, kanikoImage, "--no-cache")

	expected := fmt.Sprintf(emptyContainerDiff, dockerImage, kanikoImage, dockerImage, kanikoImage)
	checkContainerDiffOutput(t, diff, expected)
}

func TestBuildWithLabels(t *testing.T) {
	repo := getGitRepo(false)
	dockerfile := fmt.Sprintf("%s/%s/Dockerfile_test_label", integrationPath, dockerfilesPath)

	testLabel := "mylabel=myvalue"

	// Build with docker
	dockerImage := GetDockerImage(config.imageRepo, "Dockerfile_test_label:mylabel")
	dockerCmd := exec.Command("docker",
		append([]string{
			"build",
			"-t", dockerImage,
			"-f", dockerfile,
			"--label", testLabel,
			repo,
		})...)
	out, err := RunCommandWithoutTest(dockerCmd)
	if err != nil {
		t.Errorf("Failed to build image %s with docker command %q: %s %s", dockerImage, dockerCmd.Args, err, string(out))
	}

	// Build with kaniko
	kanikoImage := GetKanikoImage(config.imageRepo, "Dockerfile_test_label:mylabel")
	dockerRunFlags := []string{"run", "--net=host"}
	dockerRunFlags = addServiceAccountFlags(dockerRunFlags, config.serviceAccount)
	dockerRunFlags = append(dockerRunFlags, ExecutorImage,
		"-f", dockerfile,
		"-d", kanikoImage,
		"--label", testLabel,
		"-c", fmt.Sprintf("git://%s", repo),
	)

	kanikoCmd := exec.Command("docker", dockerRunFlags...)

	out, err = RunCommandWithoutTest(kanikoCmd)
	if err != nil {
		t.Errorf("Failed to build image %s with kaniko command %q: %v %s", dockerImage, kanikoCmd.Args, err, string(out))
	}

	diff := containerDiff(t, daemonPrefix+dockerImage, kanikoImage, "--no-cache")

	expected := fmt.Sprintf(emptyContainerDiff, dockerImage, kanikoImage, dockerImage, kanikoImage)
	checkContainerDiffOutput(t, diff, expected)
}

func TestBuildWithHTTPError(t *testing.T) {
	repo := getGitRepo(false)
	dockerfile := fmt.Sprintf("%s/%s/Dockerfile_test_add_404", integrationPath, dockerfilesPath)

	// Build with docker
	dockerImage := GetDockerImage(config.imageRepo, "Dockerfile_test_add_404")
	dockerCmd := exec.Command("docker",
		append([]string{
			"build",
			"-t", dockerImage,
			"-f", dockerfile,
			repo,
		})...)
	out, err := RunCommandWithoutTest(dockerCmd)
	if err == nil {
		t.Errorf("an error was expected, got %s", string(out))
	}

	// Build with kaniko
	kanikoImage := GetKanikoImage(config.imageRepo, "Dockerfile_test_add_404")
	dockerRunFlags := []string{"run", "--net=host"}
	dockerRunFlags = addServiceAccountFlags(dockerRunFlags, config.serviceAccount)
	dockerRunFlags = append(dockerRunFlags, ExecutorImage,
		"-f", dockerfile,
		"-d", kanikoImage,
		"-c", fmt.Sprintf("git://%s", repo),
	)

	kanikoCmd := exec.Command("docker", dockerRunFlags...)

	out, err = RunCommandWithoutTest(kanikoCmd)
	if err == nil {
		t.Errorf("an error was expected, got %s", string(out))
	}
}

func TestLayers(t *testing.T) {
	offset := map[string]int{
		"Dockerfile_test_add":     12,
		"Dockerfile_test_scratch": 3,
	}

	if os.Getenv("CI") == "true" {
		// TODO: tejaldesai fix this!
		// This files build locally with difference 0, on CI docker
		// produces a different amount of layers (?).
		offset["Dockerfile_test_copy_same_file_many_times"] = 47
		offset["Dockerfile_test_meta_arg"] = 1
		offset["Dockerfile_test_copyadd_chmod"] = 6
	}

	for _, dockerfile := range allDockerfiles {
		t.Run("test_layer_"+dockerfile, func(t *testing.T) {
			dockerfileTest := dockerfile

			t.Parallel()
			if _, ok := imageBuilder.DockerfilesToIgnore[dockerfileTest]; ok {
				t.SkipNow()
			}

			buildImage(t, dockerfileTest, imageBuilder)

			// Pull the kaniko image
			dockerImage := GetDockerImage(config.imageRepo, dockerfileTest)
			kanikoImage := GetKanikoImage(config.imageRepo, dockerfileTest)
			pullCmd := exec.Command("docker", "pull", kanikoImage)
			RunCommand(pullCmd, t)
			checkLayers(t, dockerImage, kanikoImage, offset[dockerfileTest])
		})
	}

	err := logBenchmarks("benchmark_layers")
	if err != nil {
		t.Logf("Failed to create benchmark file: %v", err)
	}
}

func TestReplaceFolderWithFileOrLink(t *testing.T) {
	dockerfiles := []string{"TestReplaceFolderWithFile", "TestReplaceFolderWithLink"}
	for _, dockerfile := range dockerfiles {
		t.Run(dockerfile, func(t *testing.T) {
			buildImage(t, dockerfile, imageBuilder)
			kanikoImage := GetKanikoImage(config.imageRepo, dockerfile)

			kanikoFiles, err := getLastLayerFiles(kanikoImage)
			if err != nil {
				t.Fatal(err)
			}
			fmt.Println(kanikoFiles)

			for _, file := range kanikoFiles {
				if strings.HasPrefix(file, "a/.wh.") {
					t.Errorf("Last layer should not add whiteout files to deleted directory but found %s", file)
				}
			}
		})
	}
}

func buildImage(t *testing.T, dockerfile string, imageBuilder *DockerFileBuilder) {
	t.Logf("Building image '%v'...", dockerfile)

	if err := imageBuilder.BuildImage(t, config, dockerfilesPath, dockerfile); err != nil {
		t.Errorf("Error building image: %s", err)
		t.FailNow()
	}
	return
}

// Build each image with kaniko twice, and then make sure they're exactly the same
func TestCache(t *testing.T) {
	populateVolumeCache()

	// Build dockerfiles with registry cache
	for dockerfile := range imageBuilder.TestCacheDockerfiles {
		t.Run("test_cache_"+dockerfile, func(t *testing.T) {
			dockerfile := dockerfile
			cache := filepath.Join(config.imageRepo, "cache", fmt.Sprintf("%v", time.Now().UnixNano()))
			t.Parallel()
			verifyBuildWith(t, cache, dockerfile)
		})
	}

	// Build dockerfiles with layout cache
	for dockerfile := range imageBuilder.TestOCICacheDockerfiles {
		t.Run("test_oci_cache_"+dockerfile, func(t *testing.T) {
			dockerfile := dockerfile
			cache := filepath.Join("oci:", cacheDir, "cached", fmt.Sprintf("%v", time.Now().UnixNano()))
			t.Parallel()
			verifyBuildWith(t, cache, dockerfile)
		})
	}

	if err := logBenchmarks("benchmark_cache"); err != nil {
		t.Logf("Failed to create benchmark file: %v", err)
	}
}

// Attempt to warm an image two times : first time should populate the cache, second time should find the image in the cache.
func TestWarmerTwice(t *testing.T) {
	_, ex, _, _ := runtime.Caller(0)
	cwd := filepath.Dir(ex) + "/tmpCache"

	// Start a sleeping warmer container
	dockerRunFlags := []string{"run", "--net=host"}
	dockerRunFlags = addServiceAccountFlags(dockerRunFlags, config.serviceAccount)
	dockerRunFlags = append(dockerRunFlags,
		"--memory=16m",
		"-v", cwd+":/cache",
		WarmerImage,
		"--cache-dir=/cache",
		"-i", "debian:trixie-slim")

	warmCmd := exec.Command("docker", dockerRunFlags...)
	out, err := RunCommandWithoutTest(warmCmd)
	if err != nil {
		t.Fatalf("Unable to perform first warming: %s", err)
	}
	t.Logf("First warm output: %s", out)

	warmCmd = exec.Command("docker", dockerRunFlags...)
	out, err = RunCommandWithoutTest(warmCmd)
	if err != nil {
		t.Fatalf("Unable to perform second warming: %s", err)
	}
	t.Logf("Second warm output: %s", out)
}

func verifyBuildWith(t *testing.T, cache, dockerfile string) {
	args := []string{}
	if strings.HasPrefix(dockerfile, "Dockerfile_test_cache_copy") {
		args = append(args, "--cache-copy-layers=true")
	}

	// Build the initial image which will cache layers
	if err := imageBuilder.buildCachedImage(config, cache, dockerfilesPath, dockerfile, 0, args); err != nil {
		t.Fatalf("error building cached image for the first time: %v", err)
	}
	// Build the second image which should pull from the cache
	if err := imageBuilder.buildCachedImage(config, cache, dockerfilesPath, dockerfile, 1, args); err != nil {
		t.Fatalf("error building cached image for the second time: %v", err)
	}
	// Make sure both images are the same
	kanikoVersion0 := GetVersionedKanikoImage(config.imageRepo, dockerfile, 0)
	kanikoVersion1 := GetVersionedKanikoImage(config.imageRepo, dockerfile, 1)

	diff := containerDiff(t, kanikoVersion0, kanikoVersion1)

	expected := fmt.Sprintf(emptyContainerDiff, kanikoVersion0, kanikoVersion1, kanikoVersion0, kanikoVersion1)
	checkContainerDiffOutput(t, diff, expected)
}

func TestRelativePaths(t *testing.T) {
	dockerfile := "Dockerfile_relative_copy"

	t.Run("test_relative_"+dockerfile, func(t *testing.T) {
		t.Parallel()

		dockerfile = filepath.Join("./dockerfiles", dockerfile)

		contextPath := "./context"

		err := imageBuilder.buildRelativePathsImage(
			config.imageRepo,
			dockerfile,
			config.serviceAccount,
			contextPath,
		)
		if err != nil {
			t.Fatal(err)
		}

		dockerImage := GetDockerImage(config.imageRepo, "test_relative_"+dockerfile)
		kanikoImage := GetKanikoImage(config.imageRepo, "test_relative_"+dockerfile)

		diff := containerDiff(t, daemonPrefix+dockerImage, kanikoImage, "--no-cache")

		expected := fmt.Sprintf(emptyContainerDiff, dockerImage, kanikoImage, dockerImage, kanikoImage)
		checkContainerDiffOutput(t, diff, expected)
	})
}

func TestExitCodePropagation(t *testing.T) {
	currentDir, err := os.Getwd()
	if err != nil {
		t.Fatal("Could not get working dir")
	}

	context := fmt.Sprintf("%s/testdata/exit-code-propagation", currentDir)
	dockerfile := fmt.Sprintf("%s/Dockerfile_exit_code_propagation", context)

	t.Run("test error code propagation", func(t *testing.T) {
		// building the image with docker should fail with exit code 42
		dockerImage := GetDockerImage(config.imageRepo, "Dockerfile_exit_code_propagation")
		dockerFlags := []string{
			"build",
			"-t", dockerImage,
			"-f", dockerfile,
		}
		dockerCmd := exec.Command("docker", append(dockerFlags, context)...)

		out, kanikoErr := RunCommandWithoutTest(dockerCmd)
		if kanikoErr == nil {
			t.Fatalf("docker build did not produce an error:\n%s", out)
		}
		var dockerCmdExitErr *exec.ExitError
		var dockerExitCode int

		if errors.As(kanikoErr, &dockerCmdExitErr) {
			dockerExitCode = dockerCmdExitErr.ExitCode()
			testutil.CheckDeepEqual(t, 42, dockerExitCode)
			if t.Failed() {
				t.Fatalf("Output was:\n%s", out)
			}
		} else {
			t.Fatalf("did not produce the expected error:\n%s", out)
		}

		// try to build the same image with kaniko the error code should match with the one from the plain docker build
		contextVolume := fmt.Sprintf("%s:/workspace", context)

		dockerFlags = []string{
			"run",
			"-v", contextVolume,
		}
		dockerFlags = addServiceAccountFlags(dockerFlags, "")
		dockerFlags = append(dockerFlags, ExecutorImage,
			"-c", "dir:///workspace/",
			"-f", "./Dockerfile_exit_code_propagation",
			"--no-push",
			"--force", // TODO: detection of whether kaniko is being run inside a container might be broken?
		)

		dockerCmdWithKaniko := exec.Command("docker", dockerFlags...)

		out, kanikoErr = RunCommandWithoutTest(dockerCmdWithKaniko)
		if kanikoErr == nil {
			t.Fatalf("the kaniko build did not produce the expected error:\n%s", out)
		}

		var kanikoExitErr *exec.ExitError
		if errors.As(kanikoErr, &kanikoExitErr) {
			testutil.CheckDeepEqual(t, dockerExitCode, kanikoExitErr.ExitCode())
			if t.Failed() {
				t.Fatalf("Output was:\n%s", out)
			}
		} else {
			t.Fatalf("did not produce the expected error:\n%s", out)
		}
	})
}

type fileDiff struct {
	Name string `json:"Name"`
	Size int    `json:"Size"`
}

type fileDiffResult struct {
	Adds []fileDiff `json:"Adds"`
	Dels []fileDiff `json:"Dels"`
}

type metaDiffResult struct {
	Adds []string `json:"Adds"`
	Dels []string `json:"Dels"`
}

type diffOutput struct {
	Image1   string
	Image2   string
	DiffType string
	Diff     interface{}
}

func (diff *diffOutput) UnmarshalJSON(data []byte) error {
	type Alias diffOutput
	aux := &struct{ *Alias }{Alias: (*Alias)(diff)}
	var rawJSON json.RawMessage
	aux.Diff = &rawJSON
	err := json.Unmarshal(data, &aux)
	if err != nil {
		return err
	}
	switch diff.DiffType {
	case "File":
		var dst fileDiffResult
		err = json.Unmarshal(rawJSON, &dst)
		diff.Diff = &dst
	case "Metadata":
		var dst metaDiffResult
		err = json.Unmarshal(rawJSON, &dst)
		diff.Diff = &dst
	}
	if err != nil {
		return err
	}
	return err
}

var allowedDiffPaths = []string{"/sys"}

func checkContainerDiffOutput(t *testing.T, diff []byte, expected string) {
	// Let's compare the json objects themselves instead of strings to avoid
	// issues with spaces and indents
	t.Helper()

	diffInt := []diffOutput{}
	expectedInt := []diffOutput{}

	err := json.Unmarshal(diff, &diffInt)
	if err != nil {
		t.Error(err)
	}

	err = json.Unmarshal([]byte(expected), &expectedInt)
	if err != nil {
		t.Error(err)
	}

	// Some differences (ignored paths, etc.) are known and expected.
	fdr := diffInt[0].Diff.(*fileDiffResult)
	fdr.Adds = filterFileDiff(fdr.Adds)
	fdr.Dels = filterFileDiff(fdr.Dels)
	// Remove some of the meta diffs that shouldn't be checked
	mdr := diffInt[1].Diff.(*metaDiffResult)
	mdr.Adds = filterMetaDiff(mdr.Adds)
	mdr.Dels = filterMetaDiff(mdr.Dels)

	testutil.CheckErrorAndDeepEqual(t, false, nil, expectedInt, diffInt)
}

func filterMetaDiff(metaDiff []string) []string {
	// TODO remove this once we agree testing shouldn't run on docker 18.xx
	// currently docker 18.xx will build an image with Metadata set
	// ArgsEscaped: true, however Docker 19.xx will build an image and have
	// ArgsEscaped: false
	if config.dockerMajorVersion == 19 {
		return metaDiff
	}
	newDiffs := []string{}
	for _, meta := range metaDiff {
		if !strings.HasPrefix(meta, "ArgsEscaped") {
			newDiffs = append(newDiffs, meta)
		}
	}
	return newDiffs
}

func filterFileDiff(f []fileDiff) []fileDiff {
	var newDiffs []fileDiff
	for _, diff := range f {
		isIgnored := false
		for _, p := range allowedDiffPaths {
			if util.HasFilepathPrefix(diff.Name, p, false) {
				isIgnored = true
				break
			}
		}
		if !isIgnored {
			newDiffs = append(newDiffs, diff)
		}
	}
	return newDiffs
}

func checkLayers(t *testing.T, image1, image2 string, offset int) {
	t.Helper()
	img1, err := getImageDetails(image1)
	if err != nil {
		t.Fatalf("Couldn't get details from image reference for (%s): %s", image1, err)
	}

	img2, err := getImageDetails(image2)
	if err != nil {
		t.Fatalf("Couldn't get details from image reference for (%s): %s", image2, err)
	}

	actualOffset := int(math.Abs(float64(img1.numLayers - img2.numLayers)))
	if actualOffset != offset {
		t.Fatalf("Difference in number of layers in each image is %d but should be %d. Image 1: %s, Image 2: %s", actualOffset, offset, img1, img2)
	}
}

func getImageDetails(image string) (*imageDetails, error) {
	ref, err := name.ParseReference(image, name.WeakValidation)
	if err != nil {
		return nil, fmt.Errorf("Couldn't parse referance to image %s: %w", image, err)
	}
	imgRef, err := daemon.Image(ref)
	if err != nil {
		return nil, fmt.Errorf("Couldn't get reference to image %s from daemon: %w", image, err)
	}
	layers, err := imgRef.Layers()
	if err != nil {
		return nil, fmt.Errorf("Error getting layers for image %s: %w", image, err)
	}
	digest, err := imgRef.Digest()
	if err != nil {
		return nil, fmt.Errorf("Error getting digest for image %s: %w", image, err)
	}
	return &imageDetails{
		name:      image,
		numLayers: len(layers),
		digest:    digest.Hex,
	}, nil
}

func getLastLayerFiles(image string) ([]string, error) {
	ref, err := name.ParseReference(image, name.WeakValidation)
	if err != nil {
		return nil, fmt.Errorf("Couldn't parse referance to image %s: %w", image, err)
	}

	imgRef, err := remote.Image(ref)
	if err != nil {
		return nil, fmt.Errorf("Couldn't get reference to image %s from daemon: %w", image, err)
	}
	layers, err := imgRef.Layers()
	if err != nil {
		return nil, fmt.Errorf("Error getting layers for image %s: %w", image, err)
	}
	readCloser, err := layers[len(layers)-1].Uncompressed()
	if err != nil {
		return nil, err
	}

	tr := tar.NewReader(readCloser)
	var files []string
	for {
		hdr, err := tr.Next()
		if errors.Is(err, io.EOF) {
			break
		}
		if err != nil {
			return nil, err
		}
		files = append(files, hdr.Name)
	}
	return files, nil
}

func logBenchmarks(benchmark string) error {
	if b, err := strconv.ParseBool(os.Getenv("BENCHMARK")); err == nil && b {
		f, err := os.Create(benchmark)
		if err != nil {
			return err
		}
		f.WriteString(timing.Summary())
		defer f.Close()
	}
	return nil
}

type imageDetails struct {
	name      string
	numLayers int
	digest    string
}

func (i imageDetails) String() string {
	return fmt.Sprintf("Image: [%s] Digest: [%s] Number of Layers: [%d]", i.name, i.digest, i.numLayers)
}

func initIntegrationTestConfig() *integrationTestConfig {
	var c integrationTestConfig

	var gcsEndpoint string
	var disableGcsAuth bool
	flag.StringVar(&c.gcsBucket, "bucket", "gs://kaniko-test-bucket", "The gcs bucket argument to uploaded the tar-ed contents of the `integration` dir to.")
	flag.StringVar(&c.imageRepo, "repo", "gcr.io/kaniko-test", "The (docker) image repo to build and push images to during the test. `gcloud` must be authenticated with this repo or serviceAccount must be set.")
	flag.StringVar(&c.serviceAccount, "serviceAccount", "", "The path to the service account push images to GCR and upload/download files to GCS.")
	flag.StringVar(&gcsEndpoint, "gcs-endpoint", "", "Custom endpoint for GCS. Used for local integration tests")
	flag.BoolVar(&disableGcsAuth, "disable-gcs-auth", false, "Disable GCS Authentication. Used for local integration tests")
	// adds the possibility to run a single dockerfile. This is useful since running all images can exhaust the dockerhub pull limit
	flag.StringVar(&c.dockerfilesPattern, "dockerfiles-pattern", "Dockerfile_test*", "The pattern to match dockerfiles with")
	flag.Parse()

	if len(c.serviceAccount) > 0 {
		absPath, err := filepath.Abs("../" + c.serviceAccount)
		if err != nil {
			log.Fatalf("Error getting absolute path for service account: %s\n", c.serviceAccount)
		}
		if _, err := os.Stat(absPath); os.IsNotExist(err) {
			log.Fatalf("Service account does not exist: %s\n", absPath)
		}
		c.serviceAccount = absPath
		os.Setenv("GOOGLE_APPLICATION_CREDENTIALS", absPath)
	}

	if c.imageRepo == "" {
		log.Fatal("You must provide a image repository")
	}

	if c.isGcrRepository() && c.gcsBucket == "" {
		log.Fatalf("You must provide a gcs bucket when using a Google Container Registry (\"%s\" was provided)", c.imageRepo)
	}
	if !strings.HasSuffix(c.imageRepo, "/") {
		c.imageRepo = c.imageRepo + "/"
	}

	if c.gcsBucket != "" {
		var opts []option.ClientOption
		if gcsEndpoint != "" {
			opts = append(opts, option.WithEndpoint(gcsEndpoint))
		}
		if disableGcsAuth {
			opts = append(opts, option.WithoutAuthentication())
		}

		gcsClient, err := bucket.NewClient(context.Background(), opts...)
		if err != nil {
			log.Fatalf("Could not create a new Google Storage Client: %s", err)
		}
		c.gcsClient = gcsClient
	}

	c.dockerMajorVersion = getDockerMajorVersion()
	c.onbuildBaseImage = c.imageRepo + "onbuild-base:latest"
	c.hardlinkBaseImage = c.imageRepo + "hardlink-base:latest"
	return &c
}

func meetsRequirements() bool {
	requiredTools := []string{"container-diff"}
	hasRequirements := true
	for _, tool := range requiredTools {
		_, err := exec.LookPath(tool)
		if err != nil {
			fmt.Printf("You must have %s installed and on your PATH\n", tool)
			hasRequirements = false
		}
	}
	return hasRequirements
}

// containerDiff compares the container images image1 and image2.
func containerDiff(t *testing.T, image1, image2 string, flags ...string) []byte {
	// workaround for container-diff OCI issue https://github.com/GoogleContainerTools/container-diff/issues/389
	if !strings.HasPrefix(image1, daemonPrefix) {
		dockerPullCmd := exec.Command("docker", "pull", image1)
		out := RunCommand(dockerPullCmd, t)
		t.Logf("docker pull cmd output for image1 = %s", string(out))
		image1 = daemonPrefix + image1
	}

	if !strings.HasPrefix(image2, daemonPrefix) {
		dockerPullCmd := exec.Command("docker", "pull", image2)
		out := RunCommand(dockerPullCmd, t)
		t.Logf("docker pull cmd output for image2 = %s", string(out))
		image2 = daemonPrefix + image2
	}

	flags = append([]string{"diff"}, flags...)
	flags = append(flags, image1, image2,
		"-q", "--type=file", "--type=metadata", "--json")

	containerdiffCmd := exec.Command("container-diff", flags...)
	diff := RunCommand(containerdiffCmd, t)
	t.Logf("diff = %s", string(diff))

	return diff
}
