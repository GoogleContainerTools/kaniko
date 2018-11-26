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

var filesToIgnore = []string{"ignore/fo*", "!ignore/foobar", "ignore/Dockerfile_test_ignore"}

const (
	ignoreDir                = "ignore"
	ignoreDockerfile         = "Dockerfile_test_ignore"
	ignoreDockerfileContents = `FROM scratch
	COPY . .`
)

// Set up a test dir to ignore with the structure:
// ignore
//  -- Dockerfile_test_ignore
//  -- foo
//  -- foobar

func setupIgnoreTestDir() error {
	if err := os.MkdirAll(ignoreDir, 0750); err != nil {
		return err
	}
	// Create and write contents to dockerfile
	path := filepath.Join(ignoreDir, ignoreDockerfile)
	f, err := os.Create(path)
	if err != nil {
		return err
	}
	defer f.Close()
	if _, err := f.Write([]byte(ignoreDockerfileContents)); err != nil {
		return err
	}

	additionalFiles := []string{"ignore/foo", "ignore/foobar"}
	for _, add := range additionalFiles {
		a, err := os.Create(add)
		if err != nil {
			return err
		}
		defer a.Close()
	}
	return generateDockerIgnore()
}

// generate the .dockerignore file
func generateDockerIgnore() error {
	f, err := os.Create(".dockerignore")
	if err != nil {
		return err
	}
	defer f.Close()
	contents := strings.Join(filesToIgnore, "\n")
	if _, err := f.Write([]byte(contents)); err != nil {
		return err
	}
	return nil
}

func generateDockerignoreImages(imageRepo string) error {

	dockerfilePath := filepath.Join(ignoreDir, ignoreDockerfile)

	dockerImage := strings.ToLower(imageRepo + dockerPrefix + ignoreDockerfile)
	dockerCmd := exec.Command("docker", "build",
		"-t", dockerImage,
		"-f", path.Join(dockerfilePath),
		".")
	_, err := RunCommandWithoutTest(dockerCmd)
	if err != nil {
		return fmt.Errorf("Failed to build image %s with docker command \"%s\": %s", dockerImage, dockerCmd.Args, err)
	}

	_, ex, _, _ := runtime.Caller(0)
	cwd := filepath.Dir(ex)
	kanikoImage := GetKanikoImage(imageRepo, ignoreDockerfile)
	kanikoCmd := exec.Command("docker",
		"run",
		"-v", os.Getenv("HOME")+"/.config/gcloud:/root/.config/gcloud",
		"-v", cwd+":/workspace",
		ExecutorImage,
		"-f", path.Join(buildContextPath, dockerfilePath),
		"-d", kanikoImage,
		"-c", buildContextPath)

	_, err = RunCommandWithoutTest(kanikoCmd)
	return err
}
