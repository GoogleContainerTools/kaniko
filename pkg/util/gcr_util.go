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
	"bytes"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"

	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
	"github.com/spf13/afero"
)

// DockerConfLocation returns the file system location of the Docker
// configuration file under the directory set in the DOCKER_CONFIG environment
// variable.  If that variable is not set, it returns the OS-equivalent of
// "/kaniko/.docker/config.json".
func DockerConfLocation() string {
	configFile := "config.json"
	if dockerConfig := os.Getenv("DOCKER_CONFIG"); dockerConfig != "" {
		file, err := os.Stat(dockerConfig)
		if err == nil {
			if file.IsDir() {
				return filepath.Join(dockerConfig, configFile)
			}
		} else {
			if os.IsNotExist(err) {
				return string(os.PathSeparator) + filepath.Join("kaniko", ".docker", configFile)
			}
		}
		return filepath.Clean(dockerConfig)
	}
	return string(os.PathSeparator) + filepath.Join("kaniko", ".docker", configFile)
}


func ConfigureGCR(flags string) error {
	// Checking for existence of docker.config as it's normally required for
	// authenticated registries and prevent overwriting user provided docker conf
	_, err := afero.NewOsFs().Stat(DockerConfLocation())
	dockerConfNotExists := os.IsNotExist(err)
	if dockerConfNotExists {
		cmd := exec.Command("docker-credential-gcr", "configure-docker", flags)
		var out bytes.Buffer
		cmd.Stderr = &out
		if err := cmd.Run(); err != nil {
			return errors.Wrap(err, fmt.Sprintf("error while configuring docker-credential-gcr helper: %s : %s", cmd.String(), out.String()))
		}
	} else {
		logrus.Warnf("\nSkip running docker-credential-gcr as user provided docker configuration exists at %s", DockerConfLocation())
	}
	return nil
}
