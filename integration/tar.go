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
	"compress/gzip"
	"fmt"
	"log"
	"os"
	"time"

	"github.com/GoogleContainerTools/kaniko/pkg/util"
)

// CreateIntegrationTarball will take the contents of the integration directory and write
// them to a tarball in a temmporary dir. It will return the path to the tarball.
func CreateIntegrationTarball() (string, error) {
	log.Println("Creating tarball of integration test files to use as build context")
	dir, err := os.Getwd()
	if err != nil {
		return "nil", fmt.Errorf("Failed find path to integration dir: %w", err)
	}
	tempDir, err := os.MkdirTemp("", "")
	if err != nil {
		return "", fmt.Errorf("Failed to create temporary directory to hold tarball: %w", err)
	}
	contextFilePath := fmt.Sprintf("%s/context_%d.tar.gz", tempDir, time.Now().UnixNano())

	file, err := os.OpenFile(contextFilePath, os.O_RDWR|os.O_CREATE|os.O_TRUNC, 0644)
	if err != nil {
		return "", err
	}

	gzipWriter := gzip.NewWriter(file)
	defer gzipWriter.Close()

	err = util.CreateTarballOfDirectory(dir, file)
	if err != nil {
		return "", fmt.Errorf("creating tarball of integration dir: %w", err)
	}

	return contextFilePath, nil
}
