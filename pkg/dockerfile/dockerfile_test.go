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

package dockerfile

import (
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"strconv"
	"testing"

	"github.com/GoogleContainerTools/kaniko/testutil"
	"github.com/docker/docker/builder/dockerfile/instructions"
)

func Test_ResolveStages(t *testing.T) {
	dockerfile := `
	FROM scratch
	RUN echo hi > /hi
	
	FROM scratch AS second
	COPY --from=0 /hi /hi2
	
	FROM scratch
	COPY --from=second /hi2 /hi3
	`
	stages, err := Parse([]byte(dockerfile))
	if err != nil {
		t.Fatal(err)
	}
	ResolveStages(stages)
	for index, stage := range stages {
		if index == 0 {
			continue
		}
		copyCmd := stage.Commands[0].(*instructions.CopyCommand)
		expectedStage := strconv.Itoa(index - 1)
		if copyCmd.From != expectedStage {
			t.Fatalf("unexpected copy command: %s resolved to stage %s, expected %s", copyCmd.String(), copyCmd.From, expectedStage)
		}
	}
}

func Test_ValidateTarget(t *testing.T) {
	dockerfile := `
	FROM scratch
	RUN echo hi > /hi
	
	FROM scratch AS second
	COPY --from=0 /hi /hi2
	
	FROM scratch
	COPY --from=second /hi2 /hi3
	`
	stages, err := Parse([]byte(dockerfile))
	if err != nil {
		t.Fatal(err)
	}
	tests := []struct {
		name      string
		target    string
		shouldErr bool
	}{
		{
			name:      "test valid target",
			target:    "second",
			shouldErr: false,
		},
		{
			name:      "test invalid target",
			target:    "invalid",
			shouldErr: true,
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			actualErr := ValidateTarget(stages, test.target)
			testutil.CheckError(t, test.shouldErr, actualErr)
		})
	}
}

func Test_Dependencies(t *testing.T) {
	testDir, err := ioutil.TempDir("", "")
	if err != nil {
		t.Fatal(err)
	}
	helloPath := filepath.Join(testDir, "hello")
	if err := os.Mkdir(helloPath, 0755); err != nil {
		t.Fatal(err)
	}

	dockerfile := fmt.Sprintf(`
	FROM scratch
	COPY %s %s
	
	FROM scratch AS second
	ENV hienv %s
	COPY a b
	COPY --from=0 /$hienv %s /hi2/
	`, helloPath, helloPath, helloPath, testDir)

	stages, err := Parse([]byte(dockerfile))
	if err != nil {
		t.Fatal(err)
	}

	expectedDependencies := [][]string{
		{
			helloPath,
			testDir,
		},
		{},
	}

	for index := range stages {
		buildArgs := NewBuildArgs([]string{})
		actualDeps, err := Dependencies(index, stages, buildArgs)
		testutil.CheckErrorAndDeepEqual(t, false, err, expectedDependencies[index], actualDeps)
	}
}

func Test_DependenciesWithArg(t *testing.T) {
	testDir, err := ioutil.TempDir("", "")
	if err != nil {
		t.Fatal(err)
	}
	helloPath := filepath.Join(testDir, "hello")
	if err := os.Mkdir(helloPath, 0755); err != nil {
		t.Fatal(err)
	}

	dockerfile := fmt.Sprintf(`
	FROM scratch
	COPY %s %s
	
	FROM scratch AS second
	ARG hienv
	COPY a b
	COPY --from=0 /$hienv %s /hi2/
	`, helloPath, helloPath, testDir)

	stages, err := Parse([]byte(dockerfile))
	if err != nil {
		t.Fatal(err)
	}

	expectedDependencies := [][]string{
		{
			helloPath,
			testDir,
		},
		{},
	}
	buildArgs := NewBuildArgs([]string{fmt.Sprintf("hienv=%s", helloPath)})

	for index := range stages {
		actualDeps, err := Dependencies(index, stages, buildArgs)
		testutil.CheckErrorAndDeepEqual(t, false, err, expectedDependencies[index], actualDeps)
	}
}
