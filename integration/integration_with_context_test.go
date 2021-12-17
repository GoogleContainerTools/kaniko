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
	"os"
	"path/filepath"
	"testing"
)

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
				t, config, "", name, testDir,
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
