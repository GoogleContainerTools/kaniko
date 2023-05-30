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
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"sync"
	"testing"

	"github.com/GoogleContainerTools/kaniko/pkg/util"
	"github.com/GoogleContainerTools/kaniko/testutil"
)

func TestBuildWithStdin(t *testing.T) {
	_, ex, _, _ := runtime.Caller(0)
	cwd := filepath.Dir(ex)

	testDir := "test_dir"
	testDirLongPath := filepath.Join(cwd, testDir)

	if err := os.MkdirAll(testDirLongPath, 0750); err != nil {
		t.Errorf("Failed to create dir_where_to_extract: %v", err)
	}

	dockerfile := "Dockerfile_test_stdin"

	files := map[string]string{
		dockerfile: "FROM debian:10.13\nRUN echo \"hey\"",
	}

	if err := testutil.SetupFiles(testDir, files); err != nil {
		t.Errorf("Failed to setup files %v on %s: %v", files, testDir, err)
	}

	if err := os.Chdir(testDir); err != nil {
		t.Fatalf("Failed to Chdir on %s: %v", testDir, err)
	}

	tarPath := fmt.Sprintf("%s.tar.gz", dockerfile)

	var wg sync.WaitGroup
	wg.Add(1)
	// Create Tar Gz File with dockerfile inside
	go func(wg *sync.WaitGroup) {
		defer wg.Done()
		tarFile, err := os.Create(tarPath)
		if err != nil {
			t.Errorf("Failed to create %s: %v", tarPath, err)
		}
		defer tarFile.Close()

		gw := gzip.NewWriter(tarFile)
		defer gw.Close()

		tw := util.NewTar(gw)
		defer tw.Close()

		if err := tw.AddFileToTar(dockerfile); err != nil {
			t.Errorf("Failed to add %s to %s: %v", dockerfile, tarPath, err)
		}
	}(&wg)

	// Waiting for the Tar Gz file creation to be done before moving on
	wg.Wait()

	// Build with docker

	dockerImage := GetDockerImage(config.imageRepo, dockerfile)
	dockerCmd := exec.Command("docker",
		append([]string{"build",
			"-t", dockerImage,
			"-f", dockerfile,
			"."})...)

	_, err := RunCommandWithoutTest(dockerCmd)
	if err != nil {
		t.Fatalf("can't run %s: %v", dockerCmd.String(), err)
	}

	// Build with kaniko using Stdin
	kanikoImageStdin := GetKanikoImage(config.imageRepo, dockerfile)
	tarCmd := exec.Command("tar", "-cf", "-", dockerfile)
	gzCmd := exec.Command("gzip", "-9")

	dockerRunFlags := []string{"run", "--interactive", "--net=host", "-v", cwd + ":/workspace"}
	dockerRunFlags = addServiceAccountFlags(dockerRunFlags, config.serviceAccount)
	dockerRunFlags = append(dockerRunFlags,
		ExecutorImage,
		"-f", dockerfile,
		"-c", "tar://stdin",
		"-d", kanikoImageStdin)

	kanikoCmdStdin := exec.Command("docker", dockerRunFlags...)

	gzCmd.Stdin, err = tarCmd.StdoutPipe()
	if err != nil {
		t.Fatalf("can't set gzCmd stdin: %v", err)
	}
	kanikoCmdStdin.Stdin, err = gzCmd.StdoutPipe()
	if err != nil {
		t.Fatalf("can't set kanikoCmd stdin: %v", err)
	}

	if err := kanikoCmdStdin.Start(); err != nil {
		t.Fatalf("can't start %s: %v", kanikoCmdStdin.String(), err)
	}

	if err := gzCmd.Start(); err != nil {
		t.Fatalf("can't start %s: %v", gzCmd.String(), err)
	}

	if err := tarCmd.Run(); err != nil {
		t.Fatalf("can't start %s: %v", tarCmd.String(), err)
	}

	if err := gzCmd.Wait(); err != nil {
		t.Fatalf("can't wait %s: %v", gzCmd.String(), err)
	}

	if err := kanikoCmdStdin.Wait(); err != nil {
		t.Fatalf("can't wait %s: %v", kanikoCmdStdin.String(), err)
	}

	diff := containerDiff(t, daemonPrefix+dockerImage, kanikoImageStdin, "--no-cache")

	expected := fmt.Sprintf(emptyContainerDiff, dockerImage, kanikoImageStdin, dockerImage, kanikoImageStdin)
	checkContainerDiffOutput(t, diff, expected)

	if err := os.RemoveAll(testDirLongPath); err != nil {
		t.Errorf("Failed to remove %s: %v", testDirLongPath, err)
	}
}
