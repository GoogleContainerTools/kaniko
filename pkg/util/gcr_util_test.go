/*
Copyright 2021 Google LLC

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

package util

import (
	"os"
	"path/filepath"
	"testing"
)

const (
	DefaultKanikoDockerConfigJSON = "/kaniko/.docker/config.json"
	DockerConfigEnvKey            = "DOCKER_CONFIG"
)

func TestDockerConfLocationWithInvalidFileLocation(t *testing.T) {
	originalDockerConfig := os.Getenv(DockerConfigEnvKey)
	if err := os.Unsetenv(DockerConfigEnvKey); err != nil {
		t.Fatalf("Failed to unset DOCKER_CONFIG: %v", err)
	}
	tmpDir := t.TempDir()
	random := "fdgdsfrdfgdf-fdfsf-24dsgfd" //replace with a really random string
	file := filepath.Join(tmpDir, random)  // an random file name, shouldn't exist
	if err := os.Setenv(DockerConfigEnvKey, file); err != nil {
		t.Fatalf("Failed to unset DOCKER_CONFIG: %v", err)
	}
	unset := DockerConfLocation()
	// as file doesn't point to a real file,  DockerConfLocation() should return the default kaniko path for config.json
	unsetExpected := DefaultKanikoDockerConfigJSON
	if unset != unsetExpected {
		t.Errorf("Unexpected default Docker configuration file location: expected:'%s' got:'%s'", unsetExpected, unset)
	}
	restoreOriginalDockerConfigEnv(t, originalDockerConfig)
}

func TestDockerConfLocation(t *testing.T) {
	originalDockerConfig := os.Getenv(DockerConfigEnvKey)

	if err := os.Unsetenv(DockerConfigEnvKey); err != nil {
		t.Fatalf("Failed to unset DOCKER_CONFIG: %v", err)
	}
	unset := DockerConfLocation()
	unsetExpected := DefaultKanikoDockerConfigJSON // will fail on Windows
	if unset != unsetExpected {
		t.Errorf("Unexpected default Docker configuration file location: expected:'%s' got:'%s'", unsetExpected, unset)
	}
	tmpDir := t.TempDir()

	dir := filepath.Join(tmpDir, "/kaniko/.docker")
	os.MkdirAll(dir, os.ModePerm)
	if err := os.Setenv(DockerConfigEnvKey, dir); err != nil {
		t.Fatalf("Failed to set DOCKER_CONFIG: %v", err)
	}
	kanikoDefault := DockerConfLocation()
	kanikoDefaultExpected := filepath.Join(tmpDir, DefaultKanikoDockerConfigJSON) // will fail on Windows
	if kanikoDefault != kanikoDefaultExpected {
		t.Errorf("Unexpected kaniko default Docker conf file location: expected:'%s' got:'%s'", kanikoDefaultExpected, kanikoDefault)
	}

	differentPath := t.TempDir()
	if err := os.Setenv(DockerConfigEnvKey, differentPath); err != nil {
		t.Fatalf("Failed to set DOCKER_CONFIG: %v", err)
	}
	set := DockerConfLocation()
	setExpected := filepath.Join(differentPath, "config.json") // will fail on Windows ?
	if set != setExpected {
		t.Errorf("Unexpected DOCKER_CONF-based file location: expected:'%s' got:'%s'", setExpected, set)
	}
	restoreOriginalDockerConfigEnv(t, originalDockerConfig)
}

func restoreOriginalDockerConfigEnv(t *testing.T, originalDockerConfig string) {
	if originalDockerConfig != "" {
		if err := os.Setenv(DockerConfigEnvKey, originalDockerConfig); err != nil {
			t.Fatalf("Failed to set DOCKER_CONFIG back to original value '%s': %v", originalDockerConfig, err)
		}
	} else {
		if err := os.Unsetenv(DockerConfigEnvKey); err != nil {
			t.Fatalf("Failed to unset DOCKER_CONFIG after testing: %v", err)
		}
	}
}

func TestDockerConfLocationWithFileLocation(t *testing.T) {
	originalDockerConfig := os.Getenv(DockerConfigEnvKey)
	if err := os.Unsetenv(DockerConfigEnvKey); err != nil {
		t.Fatalf("Failed to unset DOCKER_CONFIG: %v", err)
	}
	file, err := os.CreateTemp("", "docker.conf")
	if err != nil {
		t.Fatalf("could not create temp file: %s", err)
	}
	defer os.Remove(file.Name())
	if err := os.Setenv(DockerConfigEnvKey, file.Name()); err != nil {
		t.Fatalf("Failed to unset DOCKER_CONFIG: %v", err)
	}
	unset := DockerConfLocation()
	unsetExpected := file.Name()
	if unset != unsetExpected {
		t.Errorf("Unexpected default Docker configuration file location: expected:'%s' got:'%s'", unsetExpected, unset)
	}
	restoreOriginalDockerConfigEnv(t, originalDockerConfig)
}
