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

package main

import (
	"fmt"
	"gopkg.in/yaml.v2"
)

var tests = []struct {
	description    string
	dockerfilePath string
	configPath     string
	context        string
	repo           string
}{
	{
		description:    "test extract filesystem",
		dockerfilePath: "/workspace/integration_tests/dockerfiles/Dockerfile_test_extract_fs",
		configPath:     "/workspace/integration_tests/dockerfiles/config_test_extract_fs.json",
		context:        "integration_tests/dockerfiles/",
		repo:           "extract-filesystem",
	},
}

type step struct {
	Name string
	Args []string
	Env  []string
}

type testyaml struct {
	Steps []step
}

var executorImage = "executor-image"
var executorCommand = "/workspace/executor"
var dockerImage = "gcr.io/cloud-builders/docker"
var ubuntuImage = "ubuntu"
var testRepo = "gcr.io/kbuild-test/"
var dockerPrefix = "docker-"
var kbuildPrefix = "kbuild-"
var daemonPrefix = "daemon://"
var containerDiffOutputFile = "container-diff.json"

func main() {

	// First, copy container-diff in
	containerDiffStep := step{
		Name: "gcr.io/cloud-builders/gsutil",
		Args: []string{"cp", "gs://container-diff/latest/container-diff-linux-amd64", "."},
	}
	containerDiffPermissions := step{
		Name: ubuntuImage,
		Args: []string{"chmod", "+x", "container-diff-linux-amd64"},
	}
	// Build executor image
	buildExecutorImage := step{
		Name: dockerImage,
		Args: []string{"build", "-t", executorImage, "-f", "integration_tests/executor/Dockerfile", "."},
	}

	y := testyaml{
		Steps: []step{containerDiffStep, containerDiffPermissions, buildExecutorImage},
	}
	for _, test := range tests {
		// First, build the image with docker
		dockerBuild := step{
			Name: dockerImage,
			Args: []string{"build", "-t", testRepo + dockerPrefix + test.repo, "-f", test.dockerfilePath, test.context},
		}

		// Then, buld the image with kbuild and commit it
		var commitID = "test"
		kbuild := step{
			Name: dockerImage,
			Args: []string{"run", "-v", test.dockerfilePath + ":/workspace/Dockerfile", "--name", commitID, executorImage, executorCommand},
		}

		commit := step{
			Name: dockerImage,
			Args: []string{"commit", commitID, testRepo + kbuildPrefix + test.repo},
		}

		dockerImage := daemonPrefix + testRepo + dockerPrefix + test.repo
		kbuildImage := daemonPrefix + testRepo + kbuildPrefix + test.repo
		// Run container diff on the images
		args := "container-diff-linux-amd64 diff " + dockerImage + " " + kbuildImage + " --type=file -j > " + containerDiffOutputFile
		containerDiff := step{
			Name: ubuntuImage,
			Args: []string{"sh", "-c", args},
			Env:  []string{"PATH=/workspace:/bin"},
		}

		// Compare output files
		compareOutputs := step{
			Name: ubuntuImage,
			Args: []string{"cmp", test.configPath, containerDiffOutputFile},
		}

		y.Steps = append(y.Steps, dockerBuild, kbuild, commit, containerDiff, compareOutputs)
	}

	d, _ := yaml.Marshal(&y)
	fmt.Println(string(d))
}
