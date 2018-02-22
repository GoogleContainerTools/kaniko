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
	"path/filepath"
)

var tests = []struct {
	description    string
	dockerfilePath string
	context        string
	repo           string
}{
	{
		description:    "test extract filesystem",
		dockerfilePath: "dockerfiles/Dockerfile",
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
var executorCommand = "/work-dir/executor"
var dockerImage = "gcr.io/cloud-builders/docker"
var testRepo = "gcr.io/kbuild-test/"
var dockerPrefix = "docker-"
var kbuildPrefix = "kbuild-"

func main() {

	// First, copy container-diff in
	containerDiffStep := step{
		Name: "gcr.io/cloud-builders/gsutil",
		Args: []string{"cp", "gs://container-diff/latest/container-diff-linux-amd64", "."},
	}
	containerDiffPermissions := step{
		Name: "ubuntu",
		Args: []string{"chmod", "+x", "container-diff-linux-amd64"},
	}
	// Build executor image
	buildExecutorImage := step{
		Name: dockerImage,
		Args: []string{"build", "-t", executorImage, "."},
	}

	y := testyaml{
		Steps: []step{containerDiffStep, containerDiffPermissions, buildExecutorImage},
	}
	for _, test := range tests {
		// First, build the image with docker
		var dockerfilePath = filepath.Join("/workspace/integration_tests", test.dockerfilePath)
		dockerBuild := step{
			Name: dockerImage,
			Args: []string{"build", "-t", testRepo + dockerPrefix + test.repo, "-f", dockerfilePath, test.context},
		}

		// Then, buld the image with kbuild and commit it
		var commitID = "test"
		kbuild := step{
			Name: dockerImage,
			Args: []string{"run", "-v", dockerfilePath + ":/dockerfile/Dockerfile", "--name", commitID, executorImage, executorCommand},
		}

		commit := step{
			Name: dockerImage,
			Args: []string{"commit", commitID, testRepo + kbuildPrefix + test.repo},
		}
		y.Steps = append(y.Steps, dockerBuild, kbuild, commit)
	}

	integrationTests := step{
		Name: "gcr.io/cloud-builders/go:debian",
		Args: []string{"test", "integration_tests/integration_test.go"},
		Env:  []string{"GOPATH=/", "PATH=/builder/bin:/go/bin:/usr/local/go/bin:/usr/local/sbin:/usr/local/bin:/usr/sbin:/usr/bin:/sbin:/bin:/workspace"},
	}
	y.Steps = append(y.Steps, integrationTests)

	d, _ := yaml.Marshal(&y)
	fmt.Println(string(d))
}
