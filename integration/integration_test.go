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
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"math"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/google/go-containerregistry/pkg/name"
	"github.com/google/go-containerregistry/pkg/v1/daemon"
	"golang.org/x/sync/errgroup"

	"github.com/GoogleContainerTools/kaniko/pkg/timing"
	"github.com/GoogleContainerTools/kaniko/pkg/util"
	"github.com/GoogleContainerTools/kaniko/testutil"
)

var config = initGCPConfig()
var imageBuilder *DockerFileBuilder

type gcpConfig struct {
	gcsBucket         string
	imageRepo         string
	onbuildBaseImage  string
	hardlinkBaseImage string
}

type imageDetails struct {
	name      string
	numLayers int
	digest    string
}

func (i imageDetails) String() string {
	return fmt.Sprintf("Image: [%s] Digest: [%s] Number of Layers: [%d]", i.name, i.digest, i.numLayers)
}

func initGCPConfig() *gcpConfig {
	var c gcpConfig
	flag.StringVar(&c.gcsBucket, "bucket", "gs://kaniko-test-bucket", "The gcs bucket argument to uploaded the tar-ed contents of the `integration` dir to.")
	flag.StringVar(&c.imageRepo, "repo", "gcr.io/kaniko-test", "The (docker) image repo to build and push images to during the test. `gcloud` must be authenticated with this repo.")
	flag.Parse()

	if c.gcsBucket == "" || c.imageRepo == "" {
		log.Fatalf("You must provide a gcs bucket (\"%s\" was provided) and a docker repo (\"%s\" was provided)", c.gcsBucket, c.imageRepo)
	}
	if !strings.HasSuffix(c.imageRepo, "/") {
		c.imageRepo = c.imageRepo + "/"
	}
	c.onbuildBaseImage = c.imageRepo + "onbuild-base:latest"
	c.hardlinkBaseImage = c.imageRepo + "hardlink-base:latest"
	return &c
}

const (
	daemonPrefix       = "daemon://"
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

func meetsRequirements() bool {
	requiredTools := []string{"container-diff", "gsutil"}
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

func TestMain(m *testing.M) {
	if !meetsRequirements() {
		fmt.Println("Missing required tools")
		os.Exit(1)
	}
	contextFile, err := CreateIntegrationTarball()
	if err != nil {
		fmt.Println("Failed to create tarball of integration files for build context", err)
		os.Exit(1)
	}

	fileInBucket, err := UploadFileToBucket(config.gcsBucket, contextFile, contextFile)
	if err != nil {
		fmt.Println("Failed to upload build context", err)
		os.Exit(1)
	}

	err = os.Remove(contextFile)
	if err != nil {
		fmt.Printf("Failed to remove tarball at %s: %s\n", contextFile, err)
		os.Exit(1)
	}

	RunOnInterrupt(func() { DeleteFromBucket(fileInBucket) })
	defer DeleteFromBucket(fileInBucket)

	setupCommands := []struct {
		name    string
		command []string
	}{
		{
			name:    "Building kaniko image",
			command: []string{"docker", "build", "-t", ExecutorImage, "-f", "../deploy/Dockerfile", ".."},
		},
		{
			name:    "Building cache warmer image",
			command: []string{"docker", "build", "-t", WarmerImage, "-f", "../deploy/Dockerfile_warmer", ".."},
		},
		{
			name:    "Building onbuild base image",
			command: []string{"docker", "build", "-t", config.onbuildBaseImage, "-f", "dockerfiles/Dockerfile_onbuild_base", "."},
		},
		{
			name:    "Pushing onbuild base image",
			command: []string{"docker", "push", config.onbuildBaseImage},
		},
		{
			name:    "Building hardlink base image",
			command: []string{"docker", "build", "-t", config.hardlinkBaseImage, "-f", "dockerfiles/Dockerfile_hardlink_base", "."},
		},
		{
			name:    "Pushing hardlink base image",
			command: []string{"docker", "push", config.hardlinkBaseImage},
		},
	}

	for _, setupCmd := range setupCommands {
		fmt.Println(setupCmd.name)
		cmd := exec.Command(setupCmd.command[0], setupCmd.command[1:]...)
		if out, err := RunCommandWithoutTest(cmd); err != nil {
			fmt.Printf("%s failed: %s", setupCmd.name, err)
			fmt.Println(string(out))
			os.Exit(1)
		}
	}

	dockerfiles, err := FindDockerFiles(dockerfilesPath)
	if err != nil {
		fmt.Printf("Coudn't create map of dockerfiles: %s", err)
		os.Exit(1)
	}
	imageBuilder = NewDockerFileBuilder(dockerfiles)

	g := errgroup.Group{}
	for dockerfile := range imageBuilder.FilesBuilt {
		df := dockerfile
		g.Go(func() error {
			return imageBuilder.BuildImage(config.imageRepo, config.gcsBucket, dockerfilesPath, df)
		})
	}
	if err := g.Wait(); err != nil {
		fmt.Printf("Error building images: %s", err)
		os.Exit(1)
	}
	os.Exit(m.Run())
}
func TestRun(t *testing.T) {
	for dockerfile := range imageBuilder.FilesBuilt {
		t.Run("test_"+dockerfile, func(t *testing.T) {
			dockerfile := dockerfile
			t.Parallel()
			if _, ok := imageBuilder.DockerfilesToIgnore[dockerfile]; ok {
				t.SkipNow()
			}
			if _, ok := imageBuilder.TestCacheDockerfiles[dockerfile]; ok {
				t.SkipNow()
			}
			dockerImage := GetDockerImage(config.imageRepo, dockerfile)
			kanikoImage := GetKanikoImage(config.imageRepo, dockerfile)

			// container-diff
			daemonDockerImage := daemonPrefix + dockerImage
			containerdiffCmd := exec.Command("container-diff", "diff", "--no-cache",
				daemonDockerImage, kanikoImage,
				"-q", "--type=file", "--type=metadata", "--json")
			diff := RunCommand(containerdiffCmd, t)
			t.Logf("diff = %s", string(diff))

			expected := fmt.Sprintf(emptyContainerDiff, dockerImage, kanikoImage, dockerImage, kanikoImage)
			checkContainerDiffOutput(t, diff, expected)

		})
	}

	err := logBenchmarks("benchmark")
	if err != nil {
		t.Logf("Failed to create benchmark file: %v", err)
	}
}

func TestGitBuildcontext(t *testing.T) {
	repo := "github.com/GoogleContainerTools/kaniko"
	dockerfile := "integration/dockerfiles/Dockerfile_test_run_2"

	// Build with docker
	dockerImage := GetDockerImage(config.imageRepo, "Dockerfile_test_git")
	dockerCmd := exec.Command("docker",
		append([]string{"build",
			"-t", dockerImage,
			"-f", dockerfile,
			repo})...)
	out, err := RunCommandWithoutTest(dockerCmd)
	if err != nil {
		t.Errorf("Failed to build image %s with docker command \"%s\": %s %s", dockerImage, dockerCmd.Args, err, string(out))
	}

	// Build with kaniko
	kanikoImage := GetKanikoImage(config.imageRepo, "Dockerfile_test_git")
	kanikoCmd := exec.Command("docker",
		append([]string{"run",
			"-v", os.Getenv("HOME") + "/.config/gcloud:/root/.config/gcloud",
			ExecutorImage,
			"-f", dockerfile,
			"-d", kanikoImage,
			"-c", fmt.Sprintf("git://%s", repo)})...)

	out, err = RunCommandWithoutTest(kanikoCmd)
	if err != nil {
		t.Errorf("Failed to build image %s with kaniko command \"%s\": %v %s", dockerImage, kanikoCmd.Args, err, string(out))
	}

	// container-diff
	daemonDockerImage := daemonPrefix + dockerImage
	containerdiffCmd := exec.Command("container-diff", "diff", "--no-cache",
		daemonDockerImage, kanikoImage,
		"-q", "--type=file", "--type=metadata", "--json")
	diff := RunCommand(containerdiffCmd, t)
	t.Logf("diff = %s", string(diff))

	expected := fmt.Sprintf(emptyContainerDiff, dockerImage, kanikoImage, dockerImage, kanikoImage)
	checkContainerDiffOutput(t, diff, expected)
}

func TestGitBuildContextWithBranch(t *testing.T) {
	repo := "github.com/GoogleContainerTools/kaniko#refs/tags/v0.10.0"
	dockerfile := "integration/dockerfiles/Dockerfile_test_run_2"

	// Build with docker
	dockerImage := GetDockerImage(config.imageRepo, "Dockerfile_test_git")
	dockerCmd := exec.Command("docker",
		append([]string{"build",
			"-t", dockerImage,
			"-f", dockerfile,
			repo})...)
	out, err := RunCommandWithoutTest(dockerCmd)
	if err != nil {
		t.Errorf("Failed to build image %s with docker command \"%s\": %s %s", dockerImage, dockerCmd.Args, err, string(out))
	}

	// Build with kaniko
	kanikoImage := GetKanikoImage(config.imageRepo, "Dockerfile_test_git")
	kanikoCmd := exec.Command("docker",
		append([]string{"run",
			"-v", os.Getenv("HOME") + "/.config/gcloud:/root/.config/gcloud",
			ExecutorImage,
			"-f", dockerfile,
			"-d", kanikoImage,
			"-c", fmt.Sprintf("git://%s", repo)})...)

	out, err = RunCommandWithoutTest(kanikoCmd)
	if err != nil {
		t.Errorf("Failed to build image %s with kaniko command \"%s\": %v %s", dockerImage, kanikoCmd.Args, err, string(out))
	}

	// container-diff
	daemonDockerImage := daemonPrefix + dockerImage
	containerdiffCmd := exec.Command("container-diff", "diff", "--no-cache",
		daemonDockerImage, kanikoImage,
		"-q", "--type=file", "--type=metadata", "--json")
	diff := RunCommand(containerdiffCmd, t)
	t.Logf("diff = %s", string(diff))

	expected := fmt.Sprintf(emptyContainerDiff, dockerImage, kanikoImage, dockerImage, kanikoImage)
	checkContainerDiffOutput(t, diff, expected)
}

func TestLayers(t *testing.T) {
	offset := map[string]int{
		"Dockerfile_test_add":     12,
		"Dockerfile_test_scratch": 3,
	}
	for dockerfile := range imageBuilder.FilesBuilt {
		t.Run("test_layer_"+dockerfile, func(t *testing.T) {
			dockerfile := dockerfile
			t.Parallel()
			if _, ok := imageBuilder.DockerfilesToIgnore[dockerfile]; ok {
				t.SkipNow()
			}
			// Pull the kaniko image
			dockerImage := GetDockerImage(config.imageRepo, dockerfile)
			kanikoImage := GetKanikoImage(config.imageRepo, dockerfile)
			pullCmd := exec.Command("docker", "pull", kanikoImage)
			RunCommand(pullCmd, t)
			checkLayers(t, dockerImage, kanikoImage, offset[dockerfile])
		})
	}

	err := logBenchmarks("benchmark_layers")
	if err != nil {
		t.Logf("Failed to create benchmark file: %v", err)
	}
}

// Build each image with kaniko twice, and then make sure they're exactly the same
func TestCache(t *testing.T) {
	populateVolumeCache()
	for dockerfile := range imageBuilder.TestCacheDockerfiles {
		t.Run("test_cache_"+dockerfile, func(t *testing.T) {
			dockerfile := dockerfile
			t.Parallel()
			cache := filepath.Join(config.imageRepo, "cache", fmt.Sprintf("%v", time.Now().UnixNano()))
			// Build the initial image which will cache layers
			if err := imageBuilder.buildCachedImages(config.imageRepo, cache, dockerfilesPath, 0); err != nil {
				t.Fatalf("error building cached image for the first time: %v", err)
			}
			// Build the second image which should pull from the cache
			if err := imageBuilder.buildCachedImages(config.imageRepo, cache, dockerfilesPath, 1); err != nil {
				t.Fatalf("error building cached image for the first time: %v", err)
			}
			// Make sure both images are the same
			kanikoVersion0 := GetVersionedKanikoImage(config.imageRepo, dockerfile, 0)
			kanikoVersion1 := GetVersionedKanikoImage(config.imageRepo, dockerfile, 1)

			// container-diff
			containerdiffCmd := exec.Command("container-diff", "diff",
				kanikoVersion0, kanikoVersion1,
				"-q", "--type=file", "--type=metadata", "--json")

			diff := RunCommand(containerdiffCmd, t)
			t.Logf("diff = %s", diff)

			expected := fmt.Sprintf(emptyContainerDiff, kanikoVersion0, kanikoVersion1, kanikoVersion0, kanikoVersion1)
			checkContainerDiffOutput(t, diff, expected)
		})
	}

	if err := logBenchmarks("benchmark_cache"); err != nil {
		t.Logf("Failed to create benchmark file: %v", err)
	}
}

func TestRelativePaths(t *testing.T) {

	dockerfile := "Dockerfile_test_copy"

	t.Run("test_relative_"+dockerfile, func(t *testing.T) {
		t.Parallel()
		imageBuilder.buildRelativePathsImage(config.imageRepo, dockerfile)

		dockerImage := GetDockerImage(config.imageRepo, dockerfile)
		kanikoImage := GetKanikoImage(config.imageRepo, dockerfile)

		// container-diff
		daemonDockerImage := daemonPrefix + dockerImage
		containerdiffCmd := exec.Command("container-diff", "diff", "--no-cache",
			daemonDockerImage, kanikoImage,
			"-q", "--type=file", "--type=metadata", "--json")
		diff := RunCommand(containerdiffCmd, t)
		t.Logf("diff = %s", string(diff))

		expected := fmt.Sprintf(emptyContainerDiff, dockerImage, kanikoImage, dockerImage, kanikoImage)
		checkContainerDiffOutput(t, diff, expected)
	})
}

type fileDiff struct {
	Name string
	Size int
}

type diffOutput struct {
	Image1   string
	Image2   string
	DiffType string
	Diff     struct {
		Adds []fileDiff
		Dels []fileDiff
	}
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

	// Some differences (whitelisted paths, etc.) are known and expected.
	diffInt[0].Diff.Adds = filterDiff(diffInt[0].Diff.Adds)
	diffInt[0].Diff.Dels = filterDiff(diffInt[0].Diff.Dels)

	testutil.CheckErrorAndDeepEqual(t, false, nil, expectedInt, diffInt)
}

func filterDiff(f []fileDiff) []fileDiff {
	var newDiffs []fileDiff
	for _, diff := range f {
		isWhitelisted := false
		for _, p := range allowedDiffPaths {
			if util.HasFilepathPrefix(diff.Name, p, false) {
				isWhitelisted = true
				break
			}
		}
		if !isWhitelisted {
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
		return nil, fmt.Errorf("Couldn't parse referance to image %s: %s", image, err)
	}
	imgRef, err := daemon.Image(ref)
	if err != nil {
		return nil, fmt.Errorf("Couldn't get reference to image %s from daemon: %s", image, err)
	}
	layers, err := imgRef.Layers()
	if err != nil {
		return nil, fmt.Errorf("Error getting layers for image %s: %s", image, err)
	}
	digest, err := imgRef.Digest()
	if err != nil {
		return nil, fmt.Errorf("Error getting digest for image %s: %s", image, err)
	}
	return &imageDetails{
		name:      image,
		numLayers: len(layers),
		digest:    digest.Hex,
	}, nil
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
