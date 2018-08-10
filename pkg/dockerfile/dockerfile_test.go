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
	"io/ioutil"
	"os"
	"path/filepath"
	"strconv"
	"testing"

	"github.com/GoogleContainerTools/kaniko/testutil"
	"github.com/moby/buildkit/frontend/dockerfile/instructions"
)

func Test_ResolveStages(t *testing.T) {
	dockerfile := `
	FROM scratch
	RUN echo hi > /hi
	
	FROM gcr.io/distroless/base AS second
	COPY --from=0 /hi /hi2
	
	FROM another/image
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

func Test_SaveStage(t *testing.T) {
	tempDir, err := ioutil.TempDir("", "")
	if err != nil {
		t.Fatalf("couldn't create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)
	files := map[string]string{
		"Dockerfile": `
		FROM scratch
		RUN echo hi > /hi
		
		FROM scratch AS second
		COPY --from=0 /hi /hi2
	
		FROM second
		RUN xxx
		
		FROM scratch
		COPY --from=second /hi2 /hi3
		`,
	}
	if err := testutil.SetupFiles(tempDir, files); err != nil {
		t.Fatalf("couldn't create dockerfile: %v", err)
	}
	stages, err := Stages(filepath.Join(tempDir, "Dockerfile"), "")
	if err != nil {
		t.Fatalf("couldn't retrieve stages from Dockerfile: %v", err)
	}
	tests := []struct {
		name     string
		index    int
		expected bool
	}{
		{
			name:     "reference stage in later copy command",
			index:    0,
			expected: true,
		},
		{
			name:     "reference stage in later from command",
			index:    1,
			expected: true,
		},
		{
			name:     "don't reference stage later",
			index:    2,
			expected: false,
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			actual := SaveStage(test.index, stages)
			testutil.CheckErrorAndDeepEqual(t, false, nil, test.expected, actual)
		})
	}
}
