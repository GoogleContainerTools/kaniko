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

var fileTests = []struct {
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
	{
		description:    "test run",
		dockerfilePath: "/workspace/integration_tests/dockerfiles/Dockerfile_test_run",
		configPath:     "/workspace/integration_tests/dockerfiles/config_test_run.json",
		context:        "integration_tests/dockerfiles/",
		repo:           "test-run",
	},
	{
		description:    "test run no files changed",
		dockerfilePath: "/workspace/integration_tests/dockerfiles/Dockerfile_test_run_2",
		configPath:     "/workspace/integration_tests/dockerfiles/config_test_run_2.json",
		context:        "integration_tests/dockerfiles/",
		repo:           "test-run-2",
	},
	{
		description:    "test copy",
		dockerfilePath: "/workspace/integration_tests/dockerfiles/Dockerfile_test_copy",
		configPath:     "/workspace/integration_tests/dockerfiles/config_test_copy.json",
		context:        "/workspace/integration_tests/",
		repo:           "test-copy",
	},
	{
		description:    "test workdir",
		dockerfilePath: "/workspace/integration_tests/dockerfiles/Dockerfile_test_workdir",
		configPath:     "/workspace/integration_tests/dockerfiles/config_test_workdir.json",
		context:        "/workspace/integration_tests/",
		repo:           "test-workdir",
	},
	{
		description:    "test volume",
		dockerfilePath: "/workspace/integration_tests/dockerfiles/Dockerfile_test_volume",
		configPath:     "/workspace/integration_tests/dockerfiles/config_test_volume.json",
		context:        "/workspace/integration_tests/",
		repo:           "test-volume",
	},
}

var structureTests = []struct {
	description           string
	dockerfilePath        string
	structureTestYamlPath string
	dockerBuildContext    string
	repo                  string
}{
	{
		description:           "test env",
		dockerfilePath:        "/workspace/integration_tests/dockerfiles/Dockerfile_test_env",
		repo:                  "test-env",
		dockerBuildContext:    "/workspace/integration_tests/dockerfiles/",
		structureTestYamlPath: "/workspace/integration_tests/dockerfiles/test_env.yaml",
	},
	{
		description:           "test metadata",
		dockerfilePath:        "/workspace/integration_tests/dockerfiles/Dockerfile_test_metadata",
		repo:                  "test-metadata",
		dockerBuildContext:    "/workspace/integration_tests/dockerfiles/",
		structureTestYamlPath: "/workspace/integration_tests/dockerfiles/test_metadata.yaml",
	},
	{
		description:           "test user command",
		dockerfilePath:        "/workspace/integration_tests/dockerfiles/Dockerfile_test_user_run",
		repo:                  "test-user",
		dockerBuildContext:    "/workspace/integration_tests/dockerfiles/",
		structureTestYamlPath: "/workspace/integration_tests/dockerfiles/test_user.yaml",
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
var executorCommand = "/kbuild/executor"
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
	structureTestsStep := step{
		Name: "gcr.io/cloud-builders/gsutil",
		Args: []string{"cp", "gs://container-structure-test/latest/container-structure-test", "."},
	}
	structureTestPermissions := step{
		Name: ubuntuImage,
		Args: []string{"chmod", "+x", "container-structure-test"},
	}
	// Build executor image
	buildExecutorImage := step{
		Name: dockerImage,
		Args: []string{"build", "-t", executorImage, "-f", "deploy/Dockerfile", "."},
	}
	y := testyaml{
		Steps: []step{containerDiffStep, containerDiffPermissions, structureTestsStep, structureTestPermissions, buildExecutorImage},
	}
	for _, test := range fileTests {
		// First, build the image with docker
		dockerImageTag := testRepo + dockerPrefix + test.repo
		dockerBuild := step{
			Name: dockerImage,
			Args: []string{"build", "-t", dockerImageTag, "-f", test.dockerfilePath, test.context},
		}

		// Then, buld the image with kbuild
		kbuildImage := testRepo + kbuildPrefix + test.repo
		kbuild := step{
			Name: executorImage,
			Args: []string{executorCommand, "--destination", kbuildImage, "--dockerfile", test.dockerfilePath, "--context", test.context},
		}

		// Pull the kbuild image
		pullKbuildImage := step{
			Name: dockerImage,
			Args: []string{"pull", kbuildImage},
		}

		daemonDockerImage := daemonPrefix + dockerImageTag
		daemonKbuildImage := daemonPrefix + kbuildImage
		// Run container diff on the images
		args := "container-diff-linux-amd64 diff " + daemonDockerImage + " " + daemonKbuildImage + " --type=file -j >" + containerDiffOutputFile
		containerDiff := step{
			Name: ubuntuImage,
			Args: []string{"sh", "-c", args},
			Env:  []string{"PATH=/workspace:/bin"},
		}

		catContainerDiffOutput := step{
			Name: ubuntuImage,
			Args: []string{"cat", containerDiffOutputFile},
		}
		compareOutputs := step{
			Name: ubuntuImage,
			Args: []string{"cmp", test.configPath, containerDiffOutputFile},
		}

		y.Steps = append(y.Steps, dockerBuild, kbuild, pullKbuildImage, containerDiff, catContainerDiffOutput, compareOutputs)
	}

	for _, test := range structureTests {

		// First, build the image with docker
		dockerImageTag := testRepo + dockerPrefix + test.repo
		dockerBuild := step{
			Name: dockerImage,
			Args: []string{"build", "-t", dockerImageTag, "-f", test.dockerfilePath, test.dockerBuildContext},
		}

		// Build the image with kbuild
		kbuildImage := testRepo + kbuildPrefix + test.repo
		kbuild := step{
			Name: executorImage,
			Args: []string{executorCommand, "--destination", kbuildImage, "--dockerfile", test.dockerfilePath},
		}
		// Pull the kbuild image
		pullKbuildImage := step{
			Name: dockerImage,
			Args: []string{"pull", kbuildImage},
		}
		// Run structure tests on the kbuild and docker image
		args := "container-structure-test -image " + kbuildImage + " " + test.structureTestYamlPath
		structureTest := step{
			Name: ubuntuImage,
			Args: []string{"sh", "-c", args},
			Env:  []string{"PATH=/workspace:/bin"},
		}
		args = "container-structure-test -image " + dockerImageTag + " " + test.structureTestYamlPath
		dockerStructureTest := step{
			Name: ubuntuImage,
			Args: []string{"sh", "-c", args},
			Env:  []string{"PATH=/workspace:/bin"},
		}

		y.Steps = append(y.Steps, dockerBuild, kbuild, pullKbuildImage, structureTest, dockerStructureTest)
	}

	d, _ := yaml.Marshal(&y)
	fmt.Println(string(d))
}
