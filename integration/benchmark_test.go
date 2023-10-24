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
	"encoding/json"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"sync"
	"testing"
	"time"
)

type result struct {
	totalBuildTime float64
	resolvingFiles float64
	walkingFiles   float64
	hashingFiles   float64
}

func TestSnapshotBenchmark(t *testing.T) {
	if b, err := strconv.ParseBool(os.Getenv("BENCHMARK")); err != nil || !b {
		t.SkipNow()
	}
	cwd, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	contextDir := filepath.Join(cwd, "benchmark_fs")

	nums := []int{10000, 50000, 100000, 200000, 300000, 500000, 700000}

	var timeMap sync.Map
	var wg sync.WaitGroup
	for _, num := range nums {
		t.Run(fmt.Sprintf("test_benchmark_%d", num), func(t *testing.T) {
			wg.Add(1)
			var err error
			go func(num int, err *error) {
				dockerfile := "Dockerfile"
				kanikoImage := fmt.Sprintf("%s_%d", GetKanikoImage(config.imageRepo, dockerfile), num)
				buildArgs := []string{"--build-arg", fmt.Sprintf("NUM=%d", num)}
				var benchmarkDir string
				benchmarkDir, *err = buildKanikoImage(t.Logf, "", dockerfile,
					buildArgs, []string{}, kanikoImage, contextDir, config.gcsBucket, config.gcsClient,
					config.serviceAccount, false)
				if *err != nil {
					return
				}
				r := newResult(t, filepath.Join(benchmarkDir, dockerfile))
				timeMap.Store(num, r)
				wg.Done()
				defer os.Remove(benchmarkDir)
			}(num, &err)
			if err != nil {
				t.Errorf("could not run benchmark results for num %d due to %s", num, err)
			}
		})
	}
	wg.Wait()

	t.Log("Number of Files,Total Build Time,Walking Filesystem, Resolving Files")
	timeMap.Range(func(key interface{}, value interface{}) bool {
		d, _ := key.(int)
		v, _ := value.(result)
		t.Logf("%d,%f,%f,%f", d, v.totalBuildTime, v.walkingFiles, v.resolvingFiles)
		return true
	})

}

func newResult(t *testing.T, f string) result {
	var current map[string]time.Duration
	jsonFile, err := os.Open(f)
	defer jsonFile.Close()
	if err != nil {
		t.Errorf("could not read benchmark file %s", f)
	}
	byteValue, _ := io.ReadAll(jsonFile)
	if err := json.Unmarshal(byteValue, &current); err != nil {
		t.Errorf("could not unmarshal benchmark file")
	}
	r := result{}
	if c, ok := current["Resolving Paths"]; ok {
		r.resolvingFiles = c.Seconds()
	}
	if c, ok := current["Walking filesystem"]; ok {
		r.walkingFiles = c.Seconds()
	}
	if c, ok := current["Total Build Time"]; ok {
		r.totalBuildTime = c.Seconds()
	}
	if c, ok := current["Hashing files"]; ok {
		r.hashingFiles = c.Seconds()
	}
	t.Log(r)
	return r
}

func TestSnapshotBenchmarkGcloud(t *testing.T) {
	if b, err := strconv.ParseBool(os.Getenv("BENCHMARK")); err != nil || !b {
		t.SkipNow()
	}
	cwd, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	contextDir := filepath.Join(cwd, "benchmark_fs")

	nums := []int{10000, 50000, 100000, 200000, 300000, 500000, 700000}

	var wg sync.WaitGroup
	t.Log("Number of Files,Total Build Time,Walking Filesystem, Resolving Files")
	for _, num := range nums {
		t.Run(fmt.Sprintf("test_benchmark_%d", num), func(t *testing.T) {
			wg.Add(1)
			go func(num int) {
				dir, err := runInGcloud(contextDir, num)
				if err != nil {
					t.Errorf("error when running in gcloud %v", err)
					return
				}
				r := newResult(t, filepath.Join(dir, "results"))
				t.Log(fmt.Sprintf("%d,%f,%f,%f, %f", num, r.totalBuildTime, r.walkingFiles, r.resolvingFiles, r.hashingFiles))
				wg.Done()
				defer os.Remove(dir)
				defer os.Chdir(cwd)
			}(num)
		})
	}
	wg.Wait()
}

func runInGcloud(dir string, num int) (string, error) {
	os.Chdir(dir)
	cmd := exec.Command("gcloud", "builds",
		"submit", "--config=cloudbuild.yaml",
		fmt.Sprintf("--substitutions=_COUNT=%d", num))
	_, err := RunCommandWithoutTest(cmd)
	if err != nil {
		return "", err
	}

	// grab gcs and to temp dir and return
	tmpDir, err := os.MkdirTemp("", fmt.Sprintf("%d", num))
	if err != nil {
		return "", err
	}
	src := fmt.Sprintf("%s/gcb/benchmark_file_%d", config.gcsBucket, num)
	dest := filepath.Join(tmpDir, "results")
	copyCommand := exec.Command("gsutil", "cp", src, dest)
	_, err = RunCommandWithoutTest(copyCommand)
	if err != nil {
		return "", fmt.Errorf("failed to download file to GCS bucket %s: %w", src, err)
	}
	return tmpDir, nil
}
