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
	ARG HELLO="Hello"
	ARG WORLD="World"
	ARG NESTED="$HELLO $WORLD"
	FROM ${IMAGE}
	RUN echo hi > /hi

	FROM scratch AS second
	COPY --from=0 /hi /hi2

	FROM scratch
	COPY --from=second /hi2 /hi3
	`
	tmpfile, err := os.CreateTemp("", "Dockerfile.test")
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

	if len(metaArgs) != 5 {
		t.Fatalf("length of stage meta args expected to be 5, but was %d", len(metaArgs))
	}

	for i, expectedVal := range []string{"ubuntu:16.04", "bar", "Hello", "World", "Hello World"} {
		if metaArgs[i].Args[0].ValueString() != expectedVal {
			t.Fatalf("expected metaArg %d val to be %s but was %s", i, expectedVal, metaArgs[i].Args[0].ValueString())
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
			Args: []instructions.KeyValuePairOptional{{Key: key, Value: &val}},
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
				if expected[i] != out[i].Args[0].ValueString() {
					t.Errorf(
						"Expected arg at index %d to equal %v but instead equaled %v",
						i,
						expected[i],
						out[i].Args[0].ValueString())
				}
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
					SourcesAndDest: instructions.SourcesAndDest{SourcePaths: []string{"a.txt"}, DestPath: "b.txt"},
					From:           "0",
				},
				&instructions.CopyCommand{
					SourcesAndDest: instructions.SourcesAndDest{SourcePaths: []string{"/app"}, DestPath: "/app"},
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

	FROM scratch AS UPPER_CASE
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
			name:        "test valid upper case target",
			target:      "UPPER_CASE",
			targetIndex: 2,
			shouldErr:   false,
		},
		{
			name:        "test no target",
			target:      "",
			targetIndex: 3,
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

func Test_ResolveStagesArgs(t *testing.T) {
	dockerfile := `
	ARG IMAGE="ubuntu:16.04"
	ARG LAST_STAGE_VARIANT
	FROM ${IMAGE} as base
	RUN echo hi > /hi
	FROM base AS base-dev
	RUN echo dev >> /hi
	FROM base AS base-prod
	RUN echo prod >> /hi
	FROM base-${LAST_STAGE_VARIANT}
	RUN cat /hi
	`

	buildArgLastVariants := []string{"dev", "prod"}
	buildArgImages := []string{"alpine:3.11", ""}
	var expectedImage string

	for _, buildArgLastVariant := range buildArgLastVariants {
		for _, buildArgImage := range buildArgImages {
			if buildArgImage != "" {
				expectedImage = buildArgImage
			} else {
				expectedImage = "ubuntu:16.04"
			}
			buildArgs := []string{fmt.Sprintf("IMAGE=%s", buildArgImage), fmt.Sprintf("LAST_STAGE_VARIANT=%s", buildArgLastVariant)}

			stages, metaArgs, err := Parse([]byte(dockerfile))
			if err != nil {
				t.Fatal(err)
			}
			stagesLen := len(stages)
			args := unifyArgs(metaArgs, buildArgs)
			if err := resolveStagesArgs(stages, args); err != nil {
				t.Fatalf("fail to resolves args %v: %v", buildArgs, err)
			}
			tests := []struct {
				name               string
				actualSourceCode   string
				actualBaseName     string
				expectedSourceCode string
				expectedBaseName   string
			}{
				{
					name:               "Test_BuildArg_From_First_Stage",
					actualSourceCode:   stages[0].SourceCode,
					actualBaseName:     stages[0].BaseName,
					expectedSourceCode: "FROM ${IMAGE} as base",
					expectedBaseName:   expectedImage,
				},
				{
					name:               "Test_BuildArg_From_Last_Stage",
					actualSourceCode:   stages[stagesLen-1].SourceCode,
					actualBaseName:     stages[stagesLen-1].BaseName,
					expectedSourceCode: "FROM base-${LAST_STAGE_VARIANT}",
					expectedBaseName:   fmt.Sprintf("base-%s", buildArgLastVariant),
				},
			}
			for _, test := range tests {
				t.Run(test.name, func(t *testing.T) {
					testutil.CheckDeepEqual(t, test.expectedSourceCode, test.actualSourceCode)
					testutil.CheckDeepEqual(t, test.expectedBaseName, test.actualBaseName)
				})
			}
		}
	}
}

func Test_SkipingUnusedStages(t *testing.T) {
	tests := []struct {
		description                   string
		dockerfile                    string
		targets                       []string
		expectedSourceCodes           map[string][]string
		expectedTargetIndexBeforeSkip map[string]int
		expectedTargetIndexAfterSkip  map[string]int
	}{
		{
			description: "dockerfile_without_copyFrom",
			dockerfile: `
			FROM alpine:3.11 AS base-dev
			RUN echo dev > /hi
			FROM alpine:3.11 AS base-prod
			RUN echo prod > /hi
			FROM base-dev as final-stage
			RUN cat /hi
			`,
			targets: []string{"base-dev", "base-prod", ""},
			expectedSourceCodes: map[string][]string{
				"base-dev":  {"FROM alpine:3.11 AS base-dev"},
				"base-prod": {"FROM alpine:3.11 AS base-prod"},
				"":          {"FROM alpine:3.11 AS base-dev", "FROM base-dev as final-stage"},
			},
			expectedTargetIndexBeforeSkip: map[string]int{
				"base-dev":  0,
				"base-prod": 1,
				"":          2,
			},
			expectedTargetIndexAfterSkip: map[string]int{
				"base-dev":  0,
				"base-prod": 0,
				"":          1,
			},
		},
		{
			description: "dockerfile_with_copyFrom",
			dockerfile: `
			FROM alpine:3.11 AS base-dev
			RUN echo dev > /hi
			FROM alpine:3.11 AS base-prod
			RUN echo prod > /hi
			FROM alpine:3.11
			COPY --from=base-prod /hi /finalhi
			RUN cat /finalhi
			`,
			targets: []string{"base-dev", "base-prod", ""},
			expectedSourceCodes: map[string][]string{
				"base-dev":  {"FROM alpine:3.11 AS base-dev"},
				"base-prod": {"FROM alpine:3.11 AS base-prod"},
				"":          {"FROM alpine:3.11 AS base-prod", "FROM alpine:3.11"},
			},
			expectedTargetIndexBeforeSkip: map[string]int{
				"base-dev":  0,
				"base-prod": 1,
				"":          2,
			},
			expectedTargetIndexAfterSkip: map[string]int{
				"base-dev":  0,
				"base-prod": 0,
				"":          1,
			},
		},
		{
			description: "dockerfile_with_two_copyFrom",
			dockerfile: `
			FROM alpine:3.11 AS base-dev
			RUN echo dev > /hi
			FROM alpine:3.11 AS base-prod
			RUN echo prod > /hi
			FROM alpine:3.11
			COPY --from=base-dev /hi /finalhidev
			COPY --from=base-prod /hi /finalhiprod
			RUN cat /finalhidev
			RUN cat /finalhiprod
			`,
			targets: []string{"base-dev", "base-prod", ""},
			expectedSourceCodes: map[string][]string{
				"base-dev":  {"FROM alpine:3.11 AS base-dev"},
				"base-prod": {"FROM alpine:3.11 AS base-prod"},
				"":          {"FROM alpine:3.11 AS base-dev", "FROM alpine:3.11 AS base-prod", "FROM alpine:3.11"},
			},
			expectedTargetIndexBeforeSkip: map[string]int{
				"base-dev":  0,
				"base-prod": 1,
				"":          2,
			},
			expectedTargetIndexAfterSkip: map[string]int{
				"base-dev":  0,
				"base-prod": 0,
				"":          2,
			},
		},
		{
			description: "dockerfile_with_two_copyFrom_and_arg",
			dockerfile: `
			FROM debian:10.13 as base
			COPY . .
			FROM scratch as second
			ENV foopath context/foo
			COPY --from=0 $foopath context/b* /foo/
			FROM second as third
			COPY --from=base /context/foo /new/foo
			FROM base as fourth
			# Make sure that we snapshot intermediate images correctly
			RUN date > /date
			ENV foo bar
			# This base image contains symlinks with relative paths to ignored directories
			# We need to test they're extracted correctly
			FROM fedora@sha256:c4cc32b09c6ae3f1353e7e33a8dda93dc41676b923d6d89afa996b421cc5aa48
			FROM fourth
			ARG file=/foo2
			COPY --from=second /foo ${file}
			COPY --from=debian:10.13 /etc/os-release /new
			`,
			targets: []string{"base", ""},
			expectedSourceCodes: map[string][]string{
				"base":   {"FROM debian:10.13 as base"},
				"second": {"FROM debian:10.13 as base", "FROM scratch as second"},
				"":       {"FROM debian:10.13 as base", "FROM scratch as second", "FROM base as fourth", "FROM fourth"},
			},
			expectedTargetIndexBeforeSkip: map[string]int{
				"base":   0,
				"second": 1,
				"":       5,
			},
			expectedTargetIndexAfterSkip: map[string]int{
				"base":   0,
				"second": 1,
				"":       3,
			},
		},
		{
			description: "dockerfile_without_final_dependencies",
			dockerfile: `
			FROM alpine:3.11
			FROM debian:10.13 as base
			RUN echo foo > /foo
			FROM debian:10.13 as fizz
			RUN echo fizz >> /fizz
			COPY --from=base /foo /fizz
			FROM alpine:3.11 as buzz
			RUN echo buzz > /buzz
			FROM alpine:3.11 as final
			RUN echo bar > /bar
			`,
			targets: []string{"final", "buzz", "fizz", ""},
			expectedSourceCodes: map[string][]string{
				"final": {"FROM alpine:3.11 as final"},
				"buzz":  {"FROM alpine:3.11 as buzz"},
				"fizz":  {"FROM debian:10.13 as base", "FROM debian:10.13 as fizz"},
				"":      {"FROM alpine:3.11", "FROM debian:10.13 as base", "FROM debian:10.13 as fizz", "FROM alpine:3.11 as buzz", "FROM alpine:3.11 as final"},
			},
			expectedTargetIndexBeforeSkip: map[string]int{
				"final": 4,
				"buzz":  3,
				"fizz":  2,
				"":      4,
			},
			expectedTargetIndexAfterSkip: map[string]int{
				"final": 0,
				"buzz":  0,
				"fizz":  1,
				"":      4,
			},
		},
	}

	for _, test := range tests {
		stages, _, err := Parse([]byte(test.dockerfile))
		testutil.CheckError(t, false, err)
		actualSourceCodes := make(map[string][]string)
		for _, target := range test.targets {
			targetIndex, err := targetStage(stages, target)
			testutil.CheckError(t, false, err)
			targetIndexBeforeSkip := targetIndex
			onlyUsedStages := skipUnusedStages(stages, &targetIndex, target)
			for _, s := range onlyUsedStages {
				actualSourceCodes[target] = append(actualSourceCodes[target], s.SourceCode)
			}
			t.Run(test.description, func(t *testing.T) {
				testutil.CheckDeepEqual(t, test.expectedSourceCodes[target], actualSourceCodes[target])
				testutil.CheckDeepEqual(t, test.expectedTargetIndexBeforeSkip[target], targetIndexBeforeSkip)
				testutil.CheckDeepEqual(t, test.expectedTargetIndexAfterSkip[target], targetIndex)
			})
		}
	}
}
