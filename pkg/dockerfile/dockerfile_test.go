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
	"strconv"
	"testing"

	"github.com/GoogleContainerTools/kaniko/testutil"
	"github.com/moby/buildkit/frontend/dockerfile/instructions"
)

func Test_resolveStages(t *testing.T) {
	dockerfile := `
	FROM scratch
	RUN echo hi > /hi
	
	FROM scratch AS second
	COPY --from=0 /hi /hi2
	
	FROM scratch
	COPY --from=second /hi2 /hi3
	`
	stages, _, err := Parse([]byte(dockerfile))
	if err != nil {
		t.Fatal(err)
	}
	resolveStages(stages)
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

func Test_targetStage(t *testing.T) {
	dockerfile := `
	FROM scratch
	RUN echo hi > /hi
	
	FROM scratch AS second
	COPY --from=0 /hi /hi2
	
	FROM scratch
	COPY --from=second /hi2 /hi3
	`
	stages, _, err := Parse([]byte(dockerfile))
	if err != nil {
		t.Fatal(err)
	}
	tests := []struct {
		name        string
		target      string
		targetIndex int
		shouldErr   bool
	}{
		{
			name:        "test valid target",
			target:      "second",
			targetIndex: 1,
			shouldErr:   false,
		},
		{
			name:        "test no target",
			target:      "",
			targetIndex: 2,
			shouldErr:   false,
		},
		{
			name:        "test invalid target",
			target:      "invalid",
			targetIndex: -1,
			shouldErr:   true,
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			target, err := targetStage(stages, test.target)
			testutil.CheckError(t, test.shouldErr, err)
			if !test.shouldErr {
				if target != test.targetIndex {
					t.Errorf("got incorrect target, expected %d got %d", test.targetIndex, target)
				}
			}
		})
	}
}

func Test_SaveStage(t *testing.T) {
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
		{
			name:     "reference current stage in next stage",
			index:    4,
			expected: true,
		},
		{
			name:     "from prebuilt stage, and reference current stage in next stage",
			index:    5,
			expected: true,
		},
		{
			name:     "final stage",
			index:    6,
			expected: false,
		},
	}
	stages, _, err := Parse([]byte(testutil.Dockerfile))
	if err != nil {
		t.Fatalf("couldn't retrieve stages from Dockerfile: %v", err)
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			actual := saveStage(test.index, stages)
			testutil.CheckErrorAndDeepEqual(t, false, nil, test.expected, actual)
		})
	}
}

func Test_baseImageIndex(t *testing.T) {
	tests := []struct {
		name         string
		currentStage int
		expected     int
	}{
		{
			name:         "stage that is built off of a previous stage",
			currentStage: 2,
			expected:     1,
		},
		{
			name:         "another stage that is built off of a previous stage",
			currentStage: 5,
			expected:     4,
		},
		{
			name:         "stage that isn't built off of a previous stage",
			currentStage: 4,
			expected:     -1,
		},
	}

	stages, _, err := Parse([]byte(testutil.Dockerfile))
	if err != nil {
		t.Fatalf("couldn't retrieve stages from Dockerfile: %v", err)
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			actual := baseImageIndex(test.currentStage, stages)
			if actual != test.expected {
				t.Fatalf("unexpected result, expected %d got %d", test.expected, actual)
			}
		})
	}
}
