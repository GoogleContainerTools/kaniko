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
	"io/ioutil"
	"log"
	"os"
	"os/exec"
	"time"
)

// CreateIntegrationTarball will take the contents of the integration directory and write
// them to a tarball in a temmporary dir. It will return a path to the tarball.
func CreateIntegrationTarball() (string, error) {
	log.Println("Creating tarball of integration test files to use as build context")
	dir, err := os.Getwd()
	if err != nil {
		return "", fmt.Errorf("Failed find path to integration dir: %w", err)
	}
	tempDir, err := ioutil.TempDir("", "")
	if err != nil {
		return "", fmt.Errorf("Failed to create temporary directory to hold tarball: %w", err)
	}
	contextFile := fmt.Sprintf("%s/context_%d.tar.gz", tempDir, time.Now().UnixNano())
	cmd := exec.Command("tar", "-C", dir, "-zcvf", contextFile, ".")
	_, err = RunCommandWithoutTest(cmd)
	if err != nil {
		return "", fmt.Errorf("Failed to create build context tarball from integration dir: %w", err)
	}
	return contextFile, err
}

// UploadFileToBucket will upload the at filePath to gcsBucket. It will return the path
// of the file in gcsBucket.
func UploadFileToBucket(gcsBucket string, filePath string, gcsPath string) (string, error) {
	dst := fmt.Sprintf("%s/%s", gcsBucket, gcsPath)
	log.Printf("Uploading file at %s to GCS bucket at %s\n", filePath, dst)

	cmd := exec.Command("gsutil", "cp", filePath, dst)
	out, err := RunCommandWithoutTest(cmd)
	if err != nil {
		log.Printf("Error uploading file %s to GCS at %s: %s", filePath, dst, err)
		log.Println(string(out))
		return "", fmt.Errorf("Failed to copy tarball to GCS bucket %s: %w", gcsBucket, err)
	}

	return dst, nil
}

// DeleteFromBucket will remove the content at path. path should be the full path
// to a file in GCS.
func DeleteFromBucket(path string) error {
	cmd := exec.Command("gsutil", "rm", path)
	_, err := RunCommandWithoutTest(cmd)
	if err != nil {
		return fmt.Errorf("Failed to delete file %s from GCS: %w", path, err)
	}
	return err
}
