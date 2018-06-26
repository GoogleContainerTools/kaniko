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
	"fmt"
	"github.com/GoogleContainerTools/kaniko/pkg/constants"
	"github.com/google/go-containerregistry/pkg/name"
	"github.com/google/go-containerregistry/pkg/v1/daemon"
	"math"
	"os"
	"os/exec"
	"path"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	"github.com/GoogleContainerTools/kaniko/testutil"
)

const (
	executorImage      = "executor-image"
	dockerImage        = "gcr.io/cloud-builders/docker"
	ubuntuImage        = "ubuntu"
	testRepo           = "gcr.io/kaniko-test/"
	dockerPrefix       = "docker-"
	kanikoPrefix       = "kaniko-"
	daemonPrefix       = "daemon://"
	kanikoTestBucket   = "kaniko-test-bucket"
	dockerfilesPath    = "dockerfiles"
	onbuildBaseImage   = testRepo + "onbuild-base:latest"
	buildContextPath   = "/workspace"
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

// TODO: remove test_user_run from this when https://github.com/GoogleContainerTools/container-diff/issues/237 is fixed
var testsToIgnore = []string{"Dockerfile_test_user_run"}

func TestMain(m *testing.M) {
	buildKaniko := exec.Command("docker", "build", "-t", executorImage, "-f", "../deploy/Dockerfile", "..")
	err := buildKaniko.Run()
	if err != nil {
		fmt.Print(err)
		fmt.Print("Building kaniko failed.")
		os.Exit(1)
	}

	// Make sure container-diff is on user's PATH
	_, err = exec.LookPath("container-diff")
	if err != nil {
		fmt.Print("Make sure you have container-diff installed and on your PATH")
		os.Exit(1)
	}

	os.Exit(m.Run())
}

func TestRun(t *testing.T) {
	dockerfiles, err := filepath.Glob(path.Join(dockerfilesPath, "Dockerfile_test*"))
	if err != nil {
		t.Error(err)
		t.FailNow()
	}

	// Map for test Dockerfile to expected ARGs
	argsMap := map[string][]string{
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

	// Map for additional flags
	additionalFlagsMap := map[string][]string{
		"Dockerfile_test_add":     {"--single-snapshot"},
		"Dockerfile_test_scratch": {"--single-snapshot"},
	}

	// TODO: remove test_user_run from this when https://github.com/GoogleContainerTools/container-diff/issues/237 is fixed
	testsToIgnore := []string{"Dockerfile_test_user_run"}
	bucketContextTests := []string{"Dockerfile_test_copy_bucket"}
	reproducibleTests := []string{"Dockerfile_test_env"}

	_, ex, _, _ := runtime.Caller(0)
	cwd := filepath.Dir(ex)

	for _, dockerfile := range dockerfiles {
		t.Run("test_"+dockerfile, func(t *testing.T) {
			dockerfile = dockerfile[len("dockerfile/")+1:]
			for _, d := range testsToIgnore {
				if dockerfile == d {
					t.SkipNow()
				}
			}
			t.Logf("%s\n", dockerfile)

			var buildArgs []string
			buildArgFlag := "--build-arg"
			for _, arg := range argsMap[dockerfile] {
				buildArgs = append(buildArgs, buildArgFlag)
				buildArgs = append(buildArgs, arg)
			}
			// build docker image
			dockerImage := strings.ToLower(testRepo + dockerPrefix + dockerfile)
			dockerCmd := exec.Command("docker",
				append([]string{"build",
					"-t", dockerImage,
					"-f", path.Join(dockerfilesPath, dockerfile),
					"."},
					buildArgs...)...,
			)
			RunCommand(dockerCmd, t)

			contextFlag := "-c"
			contextPath := buildContextPath
			for _, d := range bucketContextTests {
				if d == dockerfile {
					contextFlag = "-b"
					contextPath = constants.GCSBuildContextPrefix + kanikoTestBucket
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
			additionalFlags := append(buildArgs, additionalFlagsMap[dockerfile]...)
			kanikoImage := strings.ToLower(testRepo + kanikoPrefix + dockerfile)
			kanikoCmd := exec.Command("docker",
				append([]string{"run",
					"-v", os.Getenv("HOME") + "/.config/gcloud:/root/.config/gcloud",
					"-v", cwd + ":/workspace",
					executorImage,
					"-f", path.Join(buildContextPath, dockerfilesPath, dockerfile),
					"-d", kanikoImage, reproducibleFlag,
					contextFlag, contextPath},
					additionalFlags...)...,
			)

			RunCommand(kanikoCmd, t)

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

			err = json.Unmarshal(diff, &diffInt)
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
	dockerfiles, err := filepath.Glob(path.Join(dockerfilesPath, "Dockerfile_test*"))
	if err != nil {
		t.Error(err)
		t.FailNow()
	}
	offset := map[string]int{
		"Dockerfile_test_add":     9,
		"Dockerfile_test_scratch": 3,
		// the Docker built image combined some of the dirs defined by separate VOLUME commands into one layer
		// which is why this offset exists
		"Dockerfile_test_volume": 1,
	}
	for _, dockerfile := range dockerfiles {
		t.Run("test_layer_"+dockerfile, func(t *testing.T) {
			dockerfile = dockerfile[len("dockerfile/")+1:]
			for _, ignore := range testsToIgnore {
				if dockerfile == ignore {
					t.SkipNow()
				}
			}
			// Pull the kaniko image
			dockerImage := strings.ToLower(testRepo + dockerPrefix + dockerfile)
			kanikoImage := strings.ToLower(testRepo + kanikoPrefix + dockerfile)
			pullCmd := exec.Command("docker", "pull", kanikoImage)
			RunCommand(pullCmd, t)
			if err := checkLayers(dockerImage, kanikoImage, offset[dockerfile]); err != nil {
				t.Error(err)
				t.Fail()
			}
		})
	}
}

func checkLayers(image1, image2 string, offset int) error {
	lenImage1, err := numLayers(image1)
	if err != nil {
		return err
	}
	lenImage2, err := numLayers(image2)
	if err != nil {
		return err
	}
	actualOffset := int(math.Abs(float64(lenImage1 - lenImage2)))
	if actualOffset != offset {
		return fmt.Errorf("incorrect offset between layers of %s and %s: expected %d but got %d", image1, image2, offset, actualOffset)
	}
	return nil
}

func numLayers(image string) (int, error) {
	ref, err := name.ParseReference(image, name.WeakValidation)
	if err != nil {
		return 0, err
	}
	img, err := daemon.Image(ref)
	if err != nil {
		return 0, err
	}
	layers, err := img.Layers()
	return len(layers), err
}

func RunCommand(cmd *exec.Cmd, t *testing.T) []byte {
	output, err := cmd.CombinedOutput()
	if err != nil {
		t.Log(cmd.Args)
		t.Log(string(output))
		t.Error(err)
		t.FailNow()
	}

	return output
}
