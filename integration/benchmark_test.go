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
	"path/filepath"
	"testing"
)

func TestSnapshotBenchmark(t *testing.T) {
	cwd, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	contextDir := filepath.Join(cwd, "benchmark")

	nums := []int{10000, 50000, 100000, 200000, 300000, 500000, 700000, 800000}

	for _, num := range nums {

		t.Run("test_benchmark"+string(num), func(t *testing.T) {
			t.Parallel()
			dockerfile := "Dockerfile_FS_benchmark"
			kanikoImage := GetKanikoImage(config.imageRepo, dockerfile)
			buildArgs := []string{"--build-arg", fmt.Sprintf("NUM=%d", num)}
			if _, err := buildKanikoImage("", dockerfile,
				buildArgs, []string{}, kanikoImage, contextDir, config.gcsBucket, config.serviceAccount); err != nil {
				t.Errorf("could not run benchmark results for num %d", num)
			}
		})
	}
	if err := logBenchmarks("benchmark"); err != nil {
		t.Logf("Failed to create benchmark file: %v", err)
	}
}
