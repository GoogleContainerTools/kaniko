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

package buildcontext

import (
	"bytes"
	"compress/gzip"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"sync"
	"testing"

	"github.com/GoogleContainerTools/kaniko/pkg/util"
	"github.com/GoogleContainerTools/kaniko/testutil"
)

func TestBuildWithLocalTar(t *testing.T) {
	_, ex, _, _ := runtime.Caller(0)
	cwd := filepath.Dir(ex)

	testDir := "test_dir"
	testDirLongPath := filepath.Join(cwd, testDir)
	dirUnpack := filepath.Join(testDirLongPath, "dir_where_to_unpack")

	if err := os.MkdirAll(dirUnpack, 0750); err != nil {
		t.Errorf("Failed to create dir_where_to_extract: %v", err)
	}

	validDockerfile := "Dockerfile_valid"
	invalidDockerfile := "Dockerfile_invalid"
	nonExistingDockerfile := "Dockerfile_non_existing"

	files := map[string]string{
		validDockerfile:   "FROM debian:10.13\nRUN echo \"valid\"",
		invalidDockerfile: "FROM debian:10.13\nRUN echo \"invalid\"",
	}

	if err := testutil.SetupFiles(testDir, files); err != nil {
		t.Errorf("Failed to setup files %v on %s: %v", files, testDir, err)
	}

	if err := os.Chdir(testDir); err != nil {
		t.Fatalf("Failed to Chdir on %s: %v", testDir, err)
	}

	validTarPath := fmt.Sprintf("%s.tar.gz", validDockerfile)
	invalidTarPath := fmt.Sprintf("%s.tar.gz", invalidDockerfile)
	nonExistingTarPath := fmt.Sprintf("%s.tar.gz", nonExistingDockerfile)

	var wg sync.WaitGroup
	wg.Add(1)
	// Create Tar Gz File with dockerfile inside
	go func(wg *sync.WaitGroup) {
		defer wg.Done()
		validTarFile, err := os.Create(validTarPath)
		if err != nil {
			t.Errorf("Failed to create %s: %v", validTarPath, err)
		}
		defer validTarFile.Close()

		invalidTarFile, err := os.Create(invalidTarPath)
		if err != nil {
			t.Errorf("Failed to create %s: %v", invalidTarPath, err)
		}
		defer invalidTarFile.Close()

		gw := gzip.NewWriter(validTarFile)
		defer gw.Close()

		tw := util.NewTar(gw)
		defer tw.Close()

		if err := tw.AddFileToTar(validDockerfile); err != nil {
			t.Errorf("Failed to add %s to %s: %v", validDockerfile, validTarPath, err)
		}
	}(&wg)

	// Waiting for the Tar Gz file creation to be done before moving on
	wg.Wait()

	tests := []struct {
		dockerfile       string
		srcContext       string
		unpackShouldErr  bool
		srcShaShouldErr  bool
		destShaShouldErr bool
	}{
		{
			dockerfile:       validDockerfile,
			srcContext:       filepath.Join(testDir, validTarPath),
			unpackShouldErr:  false,
			srcShaShouldErr:  false,
			destShaShouldErr: false,
		},
		{
			dockerfile:       invalidDockerfile,
			srcContext:       filepath.Join(testDir, invalidTarPath),
			unpackShouldErr:  true,
			srcShaShouldErr:  false,
			destShaShouldErr: true,
		},
		{
			dockerfile:       nonExistingDockerfile,
			srcContext:       filepath.Join(testDir, nonExistingTarPath),
			unpackShouldErr:  true,
			srcShaShouldErr:  true,
			destShaShouldErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.dockerfile, func(t *testing.T) {
			err := util.UnpackCompressedTar(filepath.Join(cwd, tt.srcContext), dirUnpack)
			testutil.CheckError(t, tt.unpackShouldErr, err)
			srcSHA, err := getSHAFromFilePath(tt.dockerfile)
			testutil.CheckError(t, tt.srcShaShouldErr, err)
			destSHA, err := getSHAFromFilePath(filepath.Join(dirUnpack, tt.dockerfile))
			testutil.CheckError(t, tt.destShaShouldErr, err)
			if err == nil {
				testutil.CheckDeepEqual(t, srcSHA, destSHA)
			}
		})
	}

	if err := os.RemoveAll(testDirLongPath); err != nil {
		t.Errorf("Failed to remove %s: %v", testDirLongPath, err)
	}
}

func getSHAFromFilePath(f string) (string, error) {
	data, err := os.ReadFile(f)
	if err != nil {
		return "", err
	}
	sha, err := util.SHA256(bytes.NewReader(data))
	if err != nil {
		return "", err
	}

	return sha, nil
}
