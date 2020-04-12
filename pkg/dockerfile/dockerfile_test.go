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
	"reflect"
	"testing"

	"github.com/GoogleContainerTools/kaniko/pkg/config"
	"github.com/GoogleContainerTools/kaniko/testutil"
	v1 "github.com/google/go-containerregistry/pkg/v1"
	"github.com/moby/buildkit/frontend/dockerfile/instructions"
)

func Test_ParseStages_ArgValueWithQuotes(t *testing.T) {
	dockerfile := `
	ARG IMAGE="ubuntu:16.04"
	ARG FOO=bar
	FROM ${IMAGE}
	RUN echo hi > /hi
	
	FROM scratch AS second
	COPY --from=0 /hi /hi2
	
	FROM scratch
	COPY --from=second /hi2 /hi3
	`
	tmpfile, err := ioutil.TempFile("", "Dockerfile.test")
	if err != nil {
		t.Fatal(err)
	}

	defer os.Remove(tmpfile.Name())

	if _, err := tmpfile.Write([]byte(dockerfile)); err != nil {
		t.Fatal(err)
	}
	if err := tmpfile.Close(); err != nil {
		t.Fatal(err)
	}

	stages, metaArgs, err := ParseStages(&config.KanikoOptions{DockerfilePath: tmpfile.Name()})
	if err != nil {
		t.Fatal(err)
	}

	if len(stages) == 0 {
		t.Fatal("length of stages expected to be greater than zero, but was zero")
	}

	if len(metaArgs) != 2 {
		t.Fatalf("length of stage meta args expected to be 2, but was %d", len(metaArgs))
	}

	for i, expectedVal := range []string{"ubuntu:16.04", "bar"} {
		if metaArgs[i].ValueString() != expectedVal {
			t.Fatalf("expected metaArg %d val to be %s but was %s", i, expectedVal, metaArgs[i].ValueString())
		}
	}
}

func Test_stripEnclosingQuotes(t *testing.T) {
	type testCase struct {
		name     string
		inArgs   []instructions.ArgCommand
		expected []string
		success  bool
	}

	newArgCommand := func(key, val string) instructions.ArgCommand {
		return instructions.ArgCommand{
			KeyValuePairOptional: instructions.KeyValuePairOptional{Key: key, Value: &val},
		}
	}

	cases := []testCase{{
		name:     "value with no enclosing quotes",
		inArgs:   []instructions.ArgCommand{newArgCommand("MEOW", "Purr")},
		expected: []string{"Purr"},
		success:  true,
	}, {
		name:   "value with unmatched leading double quote",
		inArgs: []instructions.ArgCommand{newArgCommand("MEOW", "\"Purr")},
	}, {
		name:   "value with unmatched trailing double quote",
		inArgs: []instructions.ArgCommand{newArgCommand("MEOW", "Purr\"")},
	}, {
		name:     "value with enclosing double quotes",
		inArgs:   []instructions.ArgCommand{newArgCommand("MEOW", "\"mrow\"")},
		expected: []string{"mrow"},
		success:  true,
	}, {
		name:   "value with unmatched leading single quote",
		inArgs: []instructions.ArgCommand{newArgCommand("MEOW", "'Purr")},
	}, {
		name:   "value with unmatched trailing single quote",
		inArgs: []instructions.ArgCommand{newArgCommand("MEOW", "Purr'")},
	}, {
		name:     "value with enclosing single quotes",
		inArgs:   []instructions.ArgCommand{newArgCommand("MEOW", "'mrow'")},
		expected: []string{"mrow"},
		success:  true,
	}, {
		name:     "blank value with enclosing double quotes",
		inArgs:   []instructions.ArgCommand{newArgCommand("MEOW", `""`)},
		expected: []string{""},
		success:  true,
	}, {
		name:     "blank value with enclosing single quotes",
		inArgs:   []instructions.ArgCommand{newArgCommand("MEOW", "''")},
		expected: []string{""},
		success:  true,
	}, {
		name:     "value with escaped, enclosing double quotes",
		inArgs:   []instructions.ArgCommand{newArgCommand("MEOW", `\"Purr\"`)},
		expected: []string{`\"Purr\"`},
		success:  true,
	}, {
		name:     "value with escaped, enclosing single quotes",
		inArgs:   []instructions.ArgCommand{newArgCommand("MEOW", `\'Purr\'`)},
		expected: []string{`\'Purr\'`},
		success:  true,
	}, {
		name: "multiple values enclosed with single quotes",
		inArgs: []instructions.ArgCommand{
			newArgCommand("MEOW", `'Purr'`),
			newArgCommand("MEW", `'Mrow'`),
		},
		expected: []string{"Purr", "Mrow"},
		success:  true,
	}, {
		name: "multiple values, one blank, one a single int",
		inArgs: []instructions.ArgCommand{
			newArgCommand("MEOW", `""`),
			newArgCommand("MEW", `1`),
		},
		expected: []string{"", "1"},
		success:  true,
	}, {
		name:    "no values",
		success: true,
	}}

	for _, test := range cases {
		t.Run(test.name, func(t *testing.T) {
			inArgs := test.inArgs
			expected := test.expected
			success := test.success

			out, err := stripEnclosingQuotes(inArgs)
			if success && err != nil {
				t.Fatal(err)
			}

			if !success && err == nil {
				t.Fatal("expected an error but none received")
			}

			if len(expected) != len(out) {
				t.Fatalf("Expected %d args but got %d", len(expected), len(out))
			}

			for i := range out {
				if expected[i] != out[i].ValueString() {
					t.Errorf(
						"Expected arg at index %d to equal %v but instead equaled %v",
						i,
						expected[i],
						out[i].ValueString())
				}
			}
		})
	}
}

func Test_ResolveCrossStageCommands(t *testing.T) {
	type testCase struct {
		name       string
		cmd        instructions.CopyCommand
		stageToIdx map[string]string
		expFrom    string
	}

	tests := []testCase{
		{
			name:       "resolve copy command",
			cmd:        instructions.CopyCommand{From: "builder"},
			stageToIdx: map[string]string{"builder": "0"},
			expFrom:    "0",
		},
		{
			name:       "resolve upper case FROM",
			cmd:        instructions.CopyCommand{From: "BuIlDeR"},
			stageToIdx: map[string]string{"builder": "0"},
			expFrom:    "0",
		},
		{
			name:       "nothing to resolve",
			cmd:        instructions.CopyCommand{From: "downloader"},
			stageToIdx: map[string]string{"builder": "0"},
			expFrom:    "downloader",
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			cmds := []instructions.Command{&test.cmd}
			ResolveCrossStageCommands(cmds, test.stageToIdx)
			if test.cmd.From != test.expFrom {
				t.Fatalf("Failed to resolve command: expected from %s, resolved to %s", test.expFrom, test.cmd.From)
			}
		})
	}
}

func Test_GetOnBuildInstructions(t *testing.T) {
	type testCase struct {
		name        string
		cfg         *v1.Config
		stageToIdx  map[string]string
		expCommands []instructions.Command
	}

	tests := []testCase{
		{name: "no on-build on config",
			cfg:         &v1.Config{},
			stageToIdx:  map[string]string{"builder": "0"},
			expCommands: nil,
		},
		{name: "onBuild on config, nothing to resolve",
			cfg:         &v1.Config{OnBuild: []string{"WORKDIR /app"}},
			stageToIdx:  map[string]string{"builder": "0", "temp": "1"},
			expCommands: []instructions.Command{&instructions.WorkdirCommand{Path: "/app"}},
		},
		{name: "onBuild on config, resolve multiple stages",
			cfg:        &v1.Config{OnBuild: []string{"COPY --from=builder a.txt b.txt", "COPY --from=temp /app /app"}},
			stageToIdx: map[string]string{"builder": "0", "temp": "1"},
			expCommands: []instructions.Command{
				&instructions.CopyCommand{
					SourcesAndDest: []string{"a.txt b.txt"},
					From:           "0",
				},
				&instructions.CopyCommand{
					SourcesAndDest: []string{"/app /app"},
					From:           "1",
				},
			}},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			cmds, err := GetOnBuildInstructions(test.cfg, test.stageToIdx)
			if err != nil {
				t.Fatalf("Failed to parse config for on-build instructions")
			}
			if len(cmds) != len(test.expCommands) {
				t.Fatalf("Expected %d commands, got %d", len(test.expCommands), len(cmds))
			}

			for i, cmd := range cmds {
				if reflect.TypeOf(cmd) != reflect.TypeOf(test.expCommands[i]) {
					t.Fatalf("Got command %s, expected %s", cmd, test.expCommands[i])
				}
				switch c := cmd.(type) {
				case *instructions.CopyCommand:
					{
						exp := test.expCommands[i].(*instructions.CopyCommand)
						testutil.CheckDeepEqual(t, exp.From, c.From)
					}
				}
			}
		})
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
			expected: false,
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
