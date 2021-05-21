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
	"errors"
	"fmt"
	"github.com/GoogleContainerTools/kaniko/testutil"
	"io/ioutil"
	"os"
	"os/exec"
	"path/filepath"
	"testing"
)

func TestExitCodePropagation(t *testing.T) {

	currentDir, err := os.Getwd()
	if err != nil {
		t.Fatal("Could not get working dir")
	}

	context := fmt.Sprintf("%s/dockerfiles-with-context/exit-code-propagation", currentDir)
	dockerfile := fmt.Sprintf("%s/Dockerfile_exit_code_propagation", context)

	t.Run("test error code propagation", func(t *testing.T) {
		// building the image with docker should fail with exit code 42
		dockerImage := GetDockerImage(config.imageRepo, "Dockerfile_exit_code_propagation")
		fmt.Printf("Building with docker: %s \n", dockerImage)
		dockerCmd := exec.Command("docker",
			append([]string{"build",
				"-t", dockerImage,
				"-f", dockerfile,
				context})...)
		_, kanikoErr := RunCommandWithoutTest(dockerCmd)
		if kanikoErr == nil {
			t.Fatal("docker build did not produce an error")
		}

		fmt.Printf("Docker Cmd = %s", dockerCmd)

		var dockerCmdExitErr *exec.ExitError
		var dockerExitCode int

		if errors.As(kanikoErr, &dockerCmdExitErr) {
			dockerExitCode = dockerCmdExitErr.ExitCode()
			testutil.CheckDeepEqual(t, 42, dockerExitCode)
		} else {
			t.Fatalf("did not produce the expected error")
		}

		//try to build the same image with kaniko the error code should match with the one from the plain docker build
		contextVolume := fmt.Sprintf("%s:/workspace", context)
		dockerCmdWithKaniko := exec.Command("docker", append([]string{
			"run",
			"-v", contextVolume,
			ExecutorImage,
			"-c", "dir:///workspace/",
			"-f", "./Dockerfile_exit_code_propagation",
			"--no-push",
		})...)
		fmt.Printf("Kaniko cmd = %s", dockerCmdWithKaniko)

		_, kanikoErr = RunCommandWithoutTest(dockerCmdWithKaniko)
		if kanikoErr == nil {
			t.Fatal("the kaniko build did not produce the expected error")
		}

		var kanikoExitErr *exec.ExitError
		if errors.As(kanikoErr, &kanikoExitErr) {
			testutil.CheckDeepEqual(t, dockerExitCode, kanikoExitErr.ExitCode())
		} else {
			t.Fatalf("did not produce the expected error")
		}
	})
}

func TestWithContext(t *testing.T) {
	cwd, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}

	dir := filepath.Join(cwd, "dockerfiles-with-context")

	testDirs, err := ioutil.ReadDir(dir)
	if err != nil {
		t.Fatal(err)
	}

	builder := NewDockerFileBuilder()

	for _, tdInfo := range testDirs {
		name := tdInfo.Name()
		testDir := filepath.Join(dir, name)

		t.Run("test_with_context_"+name, func(t *testing.T) {
			t.Parallel()

			if err := builder.BuildImageWithContext(
				config, "", name, testDir,
			); err != nil {
				t.Fatal(err)
			}

			dockerImage := GetDockerImage(config.imageRepo, name)
			kanikoImage := GetKanikoImage(config.imageRepo, name)

			diff := containerDiff(t, daemonPrefix+dockerImage, kanikoImage, "--no-cache")

			expected := fmt.Sprintf(emptyContainerDiff, dockerImage, kanikoImage, dockerImage, kanikoImage)
			checkContainerDiffOutput(t, diff, expected)

		})
	}

	if err := logBenchmarks("benchmark"); err != nil {
		t.Logf("Failed to create benchmark file: %v", err)
	}
}
