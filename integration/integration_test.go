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
			"hello=hello-value",
			"file=context/foo",
			"file3=context/b*",
		},
		"Dockerfile_test_multistage": {"file=/foo2"},
	}

	bucketContextTests := []string{"Dockerfile_test_copy_bucket"}

	// TODO: remove test_user_run from this when https://github.com/GoogleContainerTools/container-diff/issues/237 is fixed
	testsToIgnore := []string{"Dockerfile_test_user_run"}

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
					contextPath = kanikoTestBucket
					break
				}
			}

			// build kaniko image
			kanikoImage := strings.ToLower(testRepo + kanikoPrefix + dockerfile)
			kanikoCmd := exec.Command("docker",
				append([]string{"run",
					"-v", os.Getenv("HOME") + "/.config/gcloud:/root/.config/gcloud",
					"-v", cwd + ":/workspace",
					executorImage,
					"-f", path.Join(buildContextPath, dockerfilesPath, dockerfile),
					"-d", kanikoImage,
					contextFlag, contextPath},
					buildArgs...)...,
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
