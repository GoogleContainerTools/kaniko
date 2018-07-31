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
	"strings"
	"testing"

	"github.com/google/go-containerregistry/pkg/name"
	"github.com/google/go-containerregistry/pkg/v1/daemon"

	"github.com/GoogleContainerTools/kaniko/testutil"
)

var config = initGCPConfig()
var imageBuilder *DockerFileBuilder

type gcpConfig struct {
	gcsBucket        string
	imageRepo        string
	onbuildBaseImage string
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
	flag.StringVar(&c.gcsBucket, "bucket", "", "The gcs bucket argument to uploaded the tar-ed contents of the `integration` dir to.")
	flag.StringVar(&c.imageRepo, "repo", "", "The (docker) image repo to build and push images to during the test. `gcloud` must be authenticated with this repo.")
	flag.Parse()

	if c.gcsBucket == "" || c.imageRepo == "" {
		log.Fatalf("You must provide a gcs bucket (\"%s\" was provided) and a docker repo (\"%s\" was provided)", c.gcsBucket, c.imageRepo)
	}
	if !strings.HasSuffix(c.imageRepo, "/") {
		c.imageRepo = c.imageRepo + "/"
	}
	c.onbuildBaseImage = c.imageRepo + "onbuild-base:latest"
	return &c
}

const (
	ubuntuImage        = "ubuntu"
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

	fileInBucket, err := UploadFileToBucket(config.gcsBucket, contextFile)
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

	fmt.Println("Building kaniko image")
	buildKaniko := exec.Command("docker", "build", "-t", ExecutorImage, "-f", "../deploy/Dockerfile", "..")
	err = buildKaniko.Run()
	if err != nil {
		fmt.Print(err)
		fmt.Print("Building kaniko failed.")
		os.Exit(1)
	}

	dockerfiles, err := FindDockerFiles(dockerfilesPath)
	if err != nil {
		fmt.Printf("Coudn't create map of dockerfiles: %s", err)
		os.Exit(1)
	}
	imageBuilder = NewDockerFileBuilder(dockerfiles)
	os.Exit(m.Run())
}
func TestRun(t *testing.T) {
	for dockerfile, built := range imageBuilder.FilesBuilt {
		t.Run("test_"+dockerfile, func(t *testing.T) {
			if !built {
				err := imageBuilder.BuildImage(config.imageRepo, config.gcsBucket, dockerfilesPath, dockerfile)
				if err != nil {
					t.Fatalf("Failed to build kaniko and docker images for %s: %s", dockerfile, err)
				}
			}
			dockerImage := GetDockerImage(config.imageRepo, dockerfile)
			kanikoImage := GetKanikoImage(config.imageRepo, dockerfile)

			// container-diff
			daemonDockerImage := daemonPrefix + dockerImage
			containerdiffCmd := exec.Command("container-diff", "diff",
				daemonDockerImage, kanikoImage,
				"-q", "--type=file", "--type=metadata", "--json")
			diff := RunCommand(containerdiffCmd, t)
			t.Logf("diff = %s", string(diff))

			expected := fmt.Sprintf(emptyContainerDiff, dockerImage, kanikoImage, dockerImage, kanikoImage)

			// Let's compare the json objects themselves instead of strings to avoid
			// issues with spaces and indents
			var diffInt interface{}
			var expectedInt interface{}

			err := json.Unmarshal(diff, &diffInt)
			if err != nil {
				t.Error(err)
				t.Fail()
			}

			err = json.Unmarshal([]byte(expected), &expectedInt)
			if err != nil {
				t.Error(err)
				t.Fail()
			}

			testutil.CheckErrorAndDeepEqual(t, false, nil, expectedInt, diffInt)
		})
	}
}

func TestLayers(t *testing.T) {
	offset := map[string]int{
		"Dockerfile_test_add":     9,
		"Dockerfile_test_scratch": 3,
		// the Docker built image combined some of the dirs defined by separate VOLUME commands into one layer
		// which is why this offset exists
		"Dockerfile_test_volume": 1,
	}
	for dockerfile, built := range imageBuilder.FilesBuilt {
		t.Run("test_layer_"+dockerfile, func(t *testing.T) {
			if !built {
				err := imageBuilder.BuildImage(config.imageRepo, config.gcsBucket, dockerfilesPath, dockerfile)
				if err != nil {
					t.Fatalf("Failed to build kaniko and docker images for %s: %s", dockerfile, err)
				}
			}
			// Pull the kaniko image
			dockerImage := GetDockerImage(config.imageRepo, dockerfile)
			kanikoImage := GetKanikoImage(config.imageRepo, dockerfile)
			pullCmd := exec.Command("docker", "pull", kanikoImage)
			RunCommand(pullCmd, t)
			if err := checkLayers(t, dockerImage, kanikoImage, offset[dockerfile]); err != nil {
				t.Error(err)
				t.Fail()
			}
		})
	}
}

func checkLayers(t *testing.T, image1, image2 string, offset int) error {
	img1, err := getImageDetails(image1)
	if err != nil {
		return fmt.Errorf("Couldn't get details from image reference for (%s): %s", image1, err)
	}

	img2, err := getImageDetails(image2)
	if err != nil {
		return fmt.Errorf("Couldn't get details from image reference for (%s): %s", image2, err)
	}

	actualOffset := int(math.Abs(float64(img1.numLayers - img2.numLayers)))
	if actualOffset != offset {
		return fmt.Errorf("Difference in number of layers in each image is %d but should be %d. Image 1: %s, Image 2: %s", actualOffset, offset, img1, img2)
	}
	return nil
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
