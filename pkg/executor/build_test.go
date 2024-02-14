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

package executor

import (
	"archive/tar"
	"bytes"
	"fmt"
	"os"
	"path/filepath"
	"reflect"
	"sort"
	"strconv"
	"testing"

	"github.com/GoogleContainerTools/kaniko/pkg/cache"
	"github.com/GoogleContainerTools/kaniko/pkg/commands"
	"github.com/GoogleContainerTools/kaniko/pkg/config"
	"github.com/GoogleContainerTools/kaniko/pkg/dockerfile"
	"github.com/GoogleContainerTools/kaniko/pkg/util"
	"github.com/GoogleContainerTools/kaniko/testutil"
	"github.com/containerd/containerd/platforms"
	"github.com/google/go-cmp/cmp"
	v1 "github.com/google/go-containerregistry/pkg/v1"
	"github.com/google/go-containerregistry/pkg/v1/empty"
	"github.com/google/go-containerregistry/pkg/v1/mutate"
	"github.com/google/go-containerregistry/pkg/v1/partial"
	"github.com/google/go-containerregistry/pkg/v1/types"
	"github.com/moby/buildkit/frontend/dockerfile/instructions"
)

func Test_reviewConfig(t *testing.T) {
	tests := []struct {
		name               string
		dockerfile         string
		originalCmd        []string
		originalEntrypoint []string
		expectedCmd        []string
	}{
		{
			name: "entrypoint and cmd declared",
			dockerfile: `
			FROM scratch
			CMD ["mycmd"]
			ENTRYPOINT ["myentrypoint"]`,
			originalEntrypoint: []string{"myentrypoint"},
			originalCmd:        []string{"mycmd"},
			expectedCmd:        []string{"mycmd"},
		},
		{
			name: "only entrypoint declared",
			dockerfile: `
			FROM scratch
			ENTRYPOINT ["myentrypoint"]`,
			originalEntrypoint: []string{"myentrypoint"},
			originalCmd:        []string{"mycmd"},
			expectedCmd:        nil,
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			config := &v1.Config{
				Cmd:        test.originalCmd,
				Entrypoint: test.originalEntrypoint,
			}
			reviewConfig(stage(t, test.dockerfile), config)
			testutil.CheckErrorAndDeepEqual(t, false, nil, test.expectedCmd, config.Cmd)
		})
	}
}

func stage(t *testing.T, d string) config.KanikoStage {
	stages, _, err := dockerfile.Parse([]byte(d))
	if err != nil {
		t.Fatalf("error parsing dockerfile: %v", err)
	}
	return config.KanikoStage{
		Stage: stages[0],
	}
}

func Test_stageBuilder_shouldTakeSnapshot(t *testing.T) {
	cmds := []commands.DockerCommand{
		&MockDockerCommand{command: "command1"},
		&MockDockerCommand{command: "command2"},
		&MockDockerCommand{command: "command3"},
	}

	type fields struct {
		stage config.KanikoStage
		opts  *config.KanikoOptions
		cmds  []commands.DockerCommand
	}
	type args struct {
		index        int
		metadataOnly bool
	}
	tests := []struct {
		name   string
		fields fields
		args   args
		want   bool
	}{
		{
			name: "final stage not last command",
			fields: fields{
				stage: config.KanikoStage{
					Final: true,
				},
				cmds: cmds,
			},
			args: args{
				index: 1,
			},
			want: true,
		},
		{
			name: "not final stage last command",
			fields: fields{
				stage: config.KanikoStage{
					Final: false,
				},
				cmds: cmds,
			},
			args: args{
				index: len(cmds) - 1,
			},
			want: true,
		},
		{
			name: "not final stage not last command",
			fields: fields{
				stage: config.KanikoStage{
					Final: false,
				},
				cmds: cmds,
			},
			args: args{
				index: 0,
			},
			want: true,
		},
		{
			name: "not final stage not last command but empty list of files",
			fields: fields{
				stage: config.KanikoStage{},
			},
			args: args{
				index:        0,
				metadataOnly: true,
			},
			want: false,
		},
		{
			name: "not final stage not last command no files provided",
			fields: fields{
				stage: config.KanikoStage{
					Final: false,
				},
			},
			args: args{
				index:        0,
				metadataOnly: false,
			},
			want: true,
		},
		{
			name: "caching enabled intermediate container",
			fields: fields{
				stage: config.KanikoStage{
					Final: false,
				},
				opts: &config.KanikoOptions{Cache: true},
				cmds: cmds,
			},
			args: args{
				index: 0,
			},
			want: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {

			if tt.fields.opts == nil {
				tt.fields.opts = &config.KanikoOptions{}
			}
			s := &stageBuilder{
				stage: tt.fields.stage,
				opts:  tt.fields.opts,
				cmds:  tt.fields.cmds,
			}
			if got := s.shouldTakeSnapshot(tt.args.index, tt.args.metadataOnly); got != tt.want {
				t.Errorf("stageBuilder.shouldTakeSnapshot() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestCalculateDependencies(t *testing.T) {
	type args struct {
		dockerfile     string
		mockInitConfig func(partial.WithConfigFile, *config.KanikoOptions) (*v1.ConfigFile, error)
	}
	tests := []struct {
		name string
		args args
		want map[int][]string
	}{
		{
			name: "no deps",
			args: args{
				dockerfile: `
FROM debian as stage1
RUN foo
FROM stage1
RUN bar
`,
			},
			want: map[int][]string{},
		},
		{
			name: "args",
			args: args{
				dockerfile: `
ARG myFile=foo
FROM debian as stage1
RUN foo
FROM stage1
ARG myFile
COPY --from=stage1 /tmp/$myFile.txt .
RUN bar
`,
			},
			want: map[int][]string{
				0: {"/tmp/foo.txt"},
			},
		},
		{
			name: "simple deps",
			args: args{
				dockerfile: `
FROM debian as stage1
FROM alpine
COPY --from=stage1 /foo /bar
`,
			},
			want: map[int][]string{
				0: {"/foo"},
			},
		},
		{
			name: "two sets deps",
			args: args{
				dockerfile: `
FROM debian as stage1
FROM ubuntu as stage2
RUN foo
COPY --from=stage1 /foo /bar
FROM alpine
COPY --from=stage2 /bar /bat
`,
			},
			want: map[int][]string{
				0: {"/foo"},
				1: {"/bar"},
			},
		},
		{
			name: "double deps",
			args: args{
				dockerfile: `
FROM debian as stage1
FROM ubuntu as stage2
RUN foo
COPY --from=stage1 /foo /bar
FROM alpine
COPY --from=stage1 /baz /bat
`,
			},
			want: map[int][]string{
				0: {"/foo", "/baz"},
			},
		},
		{
			name: "envs in deps",
			args: args{
				dockerfile: `
FROM debian as stage1
FROM ubuntu as stage2
RUN foo
ENV key1 val1
ENV key2 val2
COPY --from=stage1 /foo/$key1 /foo/$key2 /bar
FROM alpine
COPY --from=stage2 /bar /bat
`,
			},
			want: map[int][]string{
				0: {"/foo/val1", "/foo/val2"},
				1: {"/bar"},
			},
		},
		{
			name: "envs from base image in deps",
			args: args{
				dockerfile: `
FROM debian as stage1
ENV key1 baseval1
FROM stage1 as stage2
RUN foo
ENV key2 val2
COPY --from=stage1 /foo/$key1 /foo/$key2 /bar
FROM alpine
COPY --from=stage2 /bar /bat
`,
			},
			want: map[int][]string{
				0: {"/foo/baseval1", "/foo/val2"},
				1: {"/bar"},
			},
		},
		{
			name: "one image has onbuild config",
			args: args{
				mockInitConfig: func(img partial.WithConfigFile, opts *config.KanikoOptions) (*v1.ConfigFile, error) {
					cfg, err := img.ConfigFile()
					// if image is "alpine" then add ONBUILD to its config
					if cfg != nil && cfg.Architecture != "" {
						cfg.Config.OnBuild = []string{"COPY --from=builder /app /app"}
					}
					return cfg, err
				},
				dockerfile: `
FROM scratch as builder
RUN foo
FROM alpine as second
# This image has an ONBUILD command so it will be executed
COPY --from=builder /foo /bar
FROM scratch as target
COPY --from=second /bar /bat
`,
			},
			want: map[int][]string{
				0: {"/app", "/foo"},
				1: {"/bar"},
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.args.mockInitConfig != nil {
				original := initializeConfig
				defer func() { initializeConfig = original }()
				initializeConfig = tt.args.mockInitConfig
			}

			f, _ := os.CreateTemp("", "")
			os.WriteFile(f.Name(), []byte(tt.args.dockerfile), 0755)
			opts := &config.KanikoOptions{
				DockerfilePath: f.Name(),
				CustomPlatform: platforms.Format(platforms.Normalize(platforms.DefaultSpec())),
			}
			testStages, metaArgs, err := dockerfile.ParseStages(opts)
			if err != nil {
				t.Errorf("Failed to parse test dockerfile to stages: %s", err)
			}

			kanikoStages, err := dockerfile.MakeKanikoStages(opts, testStages, metaArgs)
			if err != nil {
				t.Errorf("Failed to parse stages to Kaniko Stages: %s", err)
			}
			stageNameToIdx := ResolveCrossStageInstructions(kanikoStages)

			got, err := CalculateDependencies(kanikoStages, opts, stageNameToIdx)
			if err != nil {
				t.Errorf("got error: %s,", err)
			}

			if !reflect.DeepEqual(got, tt.want) {
				diff := cmp.Diff(got, tt.want)
				t.Errorf("CalculateDependencies() = %v, want %v, diff %v", got, tt.want, diff)
			}
		})
	}
}

func Test_filesToSave(t *testing.T) {
	tests := []struct {
		name  string
		args  []string
		want  []string
		files []string
	}{
		{
			name:  "simple",
			args:  []string{"foo"},
			files: []string{"foo"},
			want:  []string{"foo"},
		},
		{
			name:  "glob",
			args:  []string{"foo*"},
			files: []string{"foo", "foo2", "fooooo", "bar"},
			want:  []string{"foo", "foo2", "fooooo"},
		},
		{
			name:  "complex glob",
			args:  []string{"foo*", "bar?"},
			files: []string{"foo", "foo2", "fooooo", "bar", "bar1", "bar2", "bar33"},
			want:  []string{"foo", "foo2", "fooooo", "bar1", "bar2"},
		},
		{
			name:  "dir",
			args:  []string{"foo"},
			files: []string{"foo/bar", "foo/baz", "foo/bat/baz"},
			want:  []string{"foo"},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tmpDir := t.TempDir()
			original := config.RootDir
			config.RootDir = tmpDir
			defer func() {
				config.RootDir = original
			}()

			for _, f := range tt.files {
				p := filepath.Join(tmpDir, f)
				dir := filepath.Dir(p)
				if dir != "." {
					if err := os.MkdirAll(dir, 0755); err != nil {
						t.Errorf("error making dir: %s", err)
					}
				}
				fp, err := os.Create(p)
				if err != nil {
					t.Errorf("error making file: %s", err)
				}
				fp.Close()
			}

			got, err := filesToSave(tt.args)
			if err != nil {
				t.Errorf("got err: %s", err)
			}
			sort.Strings(tt.want)
			sort.Strings(got)
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("filesToSave() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestDeduplicatePaths(t *testing.T) {
	tests := []struct {
		name  string
		input []string
		want  []string
	}{
		{
			name:  "no duplicates",
			input: []string{"file1.txt", "file2.txt", "usr/lib"},
			want:  []string{"file1.txt", "file2.txt", "usr/lib"},
		},
		{
			name:  "duplicates",
			input: []string{"file1.txt", "file2.txt", "file2.txt", "usr/lib"},
			want:  []string{"file1.txt", "file2.txt", "usr/lib"},
		},
		{
			name:  "duplicates with paths",
			input: []string{"file1.txt", "file2.txt", "file2.txt", "usr/lib", "usr/lib/ssl"},
			want:  []string{"file1.txt", "file2.txt", "usr/lib"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := deduplicatePaths(tt.input)
			sort.Strings(tt.want)
			sort.Strings(got)
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("TestDeduplicatePaths() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestInitializeConfig(t *testing.T) {
	tests := []struct {
		description string
		cfg         v1.ConfigFile
		expected    v1.Config
	}{
		{
			description: "env is not set in the image",
			cfg: v1.ConfigFile{
				Config: v1.Config{
					Image: "test",
				},
			},
			expected: v1.Config{
				Image: "test",
				Env: []string{
					"PATH=/usr/local/sbin:/usr/local/bin:/usr/sbin:/usr/bin:/sbin:/bin",
				},
			},
		},
		{
			description: "env is set in the image",
			cfg: v1.ConfigFile{
				Config: v1.Config{
					Env: []string{
						"PATH=/usr/local/something",
					},
				},
			},
			expected: v1.Config{
				Env: []string{
					"PATH=/usr/local/something",
				},
			},
		},
		{
			description: "image is empty",
			expected: v1.Config{
				Env: []string{
					"PATH=/usr/local/sbin:/usr/local/bin:/usr/sbin:/usr/bin:/sbin:/bin",
				},
			},
		},
	}
	for _, tt := range tests {
		img, err := mutate.ConfigFile(empty.Image, &tt.cfg)
		if err != nil {
			t.Errorf("error seen when running test %s", err)
			t.Fail()
		}
		actual, _ := initializeConfig(img, nil)
		testutil.CheckDeepEqual(t, tt.expected, actual.Config)
	}
}

func Test_newLayerCache_defaultCache(t *testing.T) {
	t.Run("default layer cache is registry cache", func(t *testing.T) {
		layerCache := newLayerCache(&config.KanikoOptions{CacheRepo: "some-cache-repo"})
		foundCache, ok := layerCache.(*cache.RegistryCache)
		if !ok {
			t.Error("expected layer cache to be a registry cache")
		}
		if foundCache.Opts.CacheRepo != "some-cache-repo" {
			t.Errorf(
				"expected cache repo to be 'some-cache-repo'; got %q", foundCache.Opts.CacheRepo,
			)
		}
	})
}

func Test_newLayerCache_layoutCache(t *testing.T) {
	t.Run("when cache repo has 'oci:' prefix layer cache is layout cache", func(t *testing.T) {
		layerCache := newLayerCache(&config.KanikoOptions{CacheRepo: "oci:/some-cache-repo"})
		foundCache, ok := layerCache.(*cache.LayoutCache)
		if !ok {
			t.Error("expected layer cache to be a layout cache")
		}
		if foundCache.Opts.CacheRepo != "oci:/some-cache-repo" {
			t.Errorf(
				"expected cache repo to be 'oci:/some-cache-repo'; got %q", foundCache.Opts.CacheRepo,
			)
		}
	})
}

func Test_stageBuilder_optimize(t *testing.T) {
	testCases := []struct {
		opts     *config.KanikoOptions
		retrieve bool
		name     string
	}{
		{
			name: "cache enabled and layer not present in cache",
			opts: &config.KanikoOptions{Cache: true},
		},
		{
			name:     "cache enabled and layer present in cache",
			opts:     &config.KanikoOptions{Cache: true},
			retrieve: true,
		},
		{
			name: "cache disabled and layer not present in cache",
			opts: &config.KanikoOptions{Cache: false},
		},
		{
			name:     "cache disabled and layer present in cache",
			opts:     &config.KanikoOptions{Cache: false},
			retrieve: true,
		},
	}
	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			cf := &v1.ConfigFile{}
			snap := &fakeSnapShotter{}
			lc := &fakeLayerCache{retrieve: tc.retrieve}
			sb := &stageBuilder{opts: tc.opts, cf: cf, snapshotter: snap, layerCache: lc,
				args: dockerfile.NewBuildArgs([]string{})}
			ck := CompositeCache{}
			file, err := os.CreateTemp("", "foo")
			if err != nil {
				t.Error(err)
			}
			command := MockDockerCommand{
				contextFiles: []string{file.Name()},
				cacheCommand: MockCachedDockerCommand{},
			}
			sb.cmds = []commands.DockerCommand{command}
			err = sb.optimize(ck, cf.Config)
			if err != nil {
				t.Errorf("Expected error to be nil but was %v", err)
			}

		})
	}
}

type stageContext struct {
	command fmt.Stringer
	args    *dockerfile.BuildArgs
	env     []string
}

func newStageContext(command string, args map[string]string, env []string) stageContext {
	dockerArgs := dockerfile.NewBuildArgs([]string{})
	for k, v := range args {
		dockerArgs.AddArg(k, &v)
	}
	return stageContext{MockDockerCommand{command: command}, dockerArgs, env}
}

func Test_stageBuilder_populateCompositeKey(t *testing.T) {
	type testcase struct {
		description string
		cmd1        stageContext
		cmd2        stageContext
		shdEqual    bool
	}
	testCases := []testcase{
		{
			description: "cache key for same command [RUN] with same build args",
			cmd1: newStageContext(
				"RUN echo $ARG > test",
				map[string]string{"ARG": "foo"},
				[]string{},
			),
			cmd2: newStageContext(
				"RUN echo $ARG > test",
				map[string]string{"ARG": "foo"},
				[]string{},
			),
			shdEqual: true,
		},
		{
			description: "cache key for same command [RUN] with same env and args",
			cmd1: newStageContext(
				"RUN echo $ENV > test",
				map[string]string{"ARG": "foo"},
				[]string{"ENV=same"},
			),
			cmd2: newStageContext(
				"RUN echo $ENV > test",
				map[string]string{"ARG": "foo"},
				[]string{"ENV=same"},
			),
			shdEqual: true,
		},
		{
			description: "cache key for same command [RUN] with same env but different args",
			cmd1: newStageContext(
				"RUN echo $ENV > test",
				map[string]string{"ARG": "foo"},
				[]string{"ENV=same"},
			),
			cmd2: newStageContext(
				"RUN echo $ENV > test",
				map[string]string{"ARG": "bar"},
				[]string{"ENV=same"},
			),
		},
		{
			description: "cache key for same command [RUN], different buildargs, args not used in command",
			cmd1: newStageContext(
				"RUN echo const > test",
				map[string]string{"ARG": "foo"},
				[]string{"ENV=foo1"},
			),
			cmd2: newStageContext(
				"RUN echo const > test",
				map[string]string{"ARG": "bar"},
				[]string{"ENV=bar1"},
			),
		},
		{
			description: "cache key for same command [RUN], different buildargs, args used in script",
			// test.sh
			// #!/bin/sh
			// echo ${ARG}
			cmd1: newStageContext(
				"RUN ./test.sh",
				map[string]string{"ARG": "foo"},
				[]string{"ENV=foo1"},
			),
			cmd2: newStageContext(
				"RUN ./test.sh",
				map[string]string{"ARG": "bar"},
				[]string{"ENV=bar1"},
			),
		},
		{
			description: "cache key for same command [RUN] with a build arg values",
			cmd1: newStageContext(
				"RUN echo $ARG > test",
				map[string]string{"ARG": "foo"},
				[]string{},
			),
			cmd2: newStageContext(
				"RUN echo $ARG > test",
				map[string]string{"ARG": "bar"},
				[]string{},
			),
		},
		{
			description: "cache key for same command [RUN] with different env values",
			cmd1: newStageContext(
				"RUN echo $ENV > test",
				map[string]string{"ARG": "foo"},
				[]string{"ENV=1"},
			),
			cmd2: newStageContext(
				"RUN echo $ENV > test",
				map[string]string{"ARG": "foo"},
				[]string{"ENV=2"},
			),
		},
		{
			description: "cache key for different command [RUN] same context",
			cmd1: newStageContext(
				"RUN echo other > test",
				map[string]string{"ARG": "foo"},
				[]string{"ENV=1"},
			),
			cmd2: newStageContext(
				"RUN echo another > test",
				map[string]string{"ARG": "foo"},
				[]string{"ENV=1"},
			),
		},
		{
			description: "cache key for command [RUN] with same env values [check that variable no interpolate in RUN command]",
			cmd1: newStageContext(
				"RUN echo $ENV > test",
				map[string]string{"ARG": "foo"},
				[]string{"ENV=1"},
			),
			cmd2: newStageContext(
				"RUN echo 1 > test",
				map[string]string{"ARG": "foo"},
				[]string{"ENV=1"},
			),
			shdEqual: false,
		},
		{
			description: "cache key for command [RUN] with different env values [check that variable no interpolate in RUN command]",
			cmd1: newStageContext(
				"RUN echo ${APP_VERSION%.*} ${APP_VERSION%-*} > test",
				map[string]string{"ARG": "foo"},
				[]string{"ENV=1"},
			),
			cmd2: newStageContext(
				"RUN echo ${APP_VERSION%.*} ${APP_VERSION%-*} > test",
				map[string]string{"ARG": "foo"},
				[]string{"ENV=2"},
			),
			shdEqual: false,
		},
		func() testcase {
			dir, files := tempDirAndFile(t)
			file := files[0]
			filePath := filepath.Join(dir, file)
			return testcase{
				description: "cache key for same command [COPY] with same args",
				cmd1: newStageContext(
					fmt.Sprintf("COPY %s /meow", filePath),
					map[string]string{"ARG": "foo"},
					[]string{"ENV=1"},
				),
				cmd2: newStageContext(
					fmt.Sprintf("COPY %s /meow", filePath),
					map[string]string{"ARG": "foo"},
					[]string{"ENV=1"},
				),
				shdEqual: true,
			}
		}(),
		func() testcase {
			dir, files := tempDirAndFile(t)
			file := files[0]
			filePath := filepath.Join(dir, file)
			return testcase{
				description: "cache key for same command [COPY] with different args",
				cmd1: newStageContext(
					fmt.Sprintf("COPY %s /meow", filePath),
					map[string]string{"ARG": "foo"},
					[]string{"ENV=1"},
				),
				cmd2: newStageContext(
					fmt.Sprintf("COPY %s /meow", filePath),
					map[string]string{"ARG": "bar"},
					[]string{"ENV=2"},
				),
				shdEqual: true,
			}
		}(),
		{
			description: "cache key for same command [WORKDIR] with same args",
			cmd1: newStageContext(
				"WORKDIR /",
				map[string]string{"ARG": "foo"},
				[]string{"ENV=1"},
			),
			cmd2: newStageContext(
				"WORKDIR /",
				map[string]string{"ARG": "foo"},
				[]string{"ENV=1"},
			),
			shdEqual: true,
		},
		{
			description: "cache key for same command [WORKDIR] with different args",
			cmd1: newStageContext(
				"WORKDIR /",
				map[string]string{"ARG": "foo"},
				[]string{"ENV=1"},
			),
			cmd2: newStageContext(
				"WORKDIR /",
				map[string]string{"ARG": "bar"},
				[]string{"ENV=2"},
			),
			shdEqual: true,
		},
	}
	for _, tc := range testCases {
		t.Run(tc.description, func(t *testing.T) {
			sb := &stageBuilder{fileContext: util.FileContext{Root: "workspace"}}
			ck := CompositeCache{}

			instructions1, err := dockerfile.ParseCommands([]string{tc.cmd1.command.String()})
			if err != nil {
				t.Fatal(err)
			}

			fc1 := util.FileContext{Root: "workspace"}
			dockerCommand1, err := commands.GetCommand(instructions1[0], fc1, false, true, true)
			if err != nil {
				t.Fatal(err)
			}

			instructions, err := dockerfile.ParseCommands([]string{tc.cmd2.command.String()})
			if err != nil {
				t.Fatal(err)
			}

			fc2 := util.FileContext{Root: "workspace"}
			dockerCommand2, err := commands.GetCommand(instructions[0], fc2, false, true, true)
			if err != nil {
				t.Fatal(err)
			}

			ck1, err := sb.populateCompositeKey(dockerCommand1, []string{}, ck, tc.cmd1.args, tc.cmd1.env)
			if err != nil {
				t.Errorf("Expected error to be nil but was %v", err)
			}
			ck2, err := sb.populateCompositeKey(dockerCommand2, []string{}, ck, tc.cmd2.args, tc.cmd2.env)
			if err != nil {
				t.Errorf("Expected error to be nil but was %v", err)
			}
			key1, key2 := hashCompositeKeys(t, ck1, ck2)
			if b := key1 == key2; b != tc.shdEqual {
				t.Errorf("expected keys to be equal as %t but found %t", tc.shdEqual, !tc.shdEqual)
			}
		})
	}
}

func Test_stageBuilder_build(t *testing.T) {
	type testcase struct {
		description        string
		opts               *config.KanikoOptions
		args               map[string]string
		layerCache         *fakeLayerCache
		expectedCacheKeys  []string
		pushedCacheKeys    []string
		commands           []commands.DockerCommand
		fileName           string
		rootDir            string
		image              v1.Image
		config             *v1.ConfigFile
		stage              config.KanikoStage
		crossStageDeps     map[int][]string
		mockGetFSFromImage func(root string, img v1.Image, extract util.ExtractFunction) ([]string, error)
		shouldInitSnapshot bool
	}

	testCases := []testcase{
		func() testcase {
			dir, files := tempDirAndFile(t)
			file := files[0]
			filePath := filepath.Join(dir, file)
			ch := NewCompositeCache("", "meow")

			ch.AddPath(filePath, util.FileContext{})
			hash, err := ch.Hash()
			if err != nil {
				t.Errorf("couldn't create hash %v", err)
			}
			command := MockDockerCommand{
				command:      "meow",
				contextFiles: []string{filePath},
				cacheCommand: MockCachedDockerCommand{
					contextFiles: []string{filePath},
				},
			}

			destDir := t.TempDir()
			return testcase{
				description:       "fake command cache enabled but key not in cache",
				config:            &v1.ConfigFile{Config: v1.Config{WorkingDir: destDir}},
				opts:              &config.KanikoOptions{Cache: true},
				expectedCacheKeys: []string{hash},
				pushedCacheKeys:   []string{hash},
				commands:          []commands.DockerCommand{command},
				rootDir:           dir,
			}
		}(),
		func() testcase {
			dir, files := tempDirAndFile(t)
			file := files[0]
			filePath := filepath.Join(dir, file)
			ch := NewCompositeCache("", "meow")

			ch.AddPath(filePath, util.FileContext{})
			hash, err := ch.Hash()
			if err != nil {
				t.Errorf("couldn't create hash %v", err)
			}
			command := MockDockerCommand{
				command:      "meow",
				contextFiles: []string{filePath},
				cacheCommand: MockCachedDockerCommand{
					contextFiles: []string{filePath},
				},
			}

			destDir := t.TempDir()
			return testcase{
				description: "fake command cache enabled and key in cache",
				opts:        &config.KanikoOptions{Cache: true},
				config:      &v1.ConfigFile{Config: v1.Config{WorkingDir: destDir}},
				layerCache: &fakeLayerCache{
					retrieve: true,
				},
				expectedCacheKeys: []string{hash},
				pushedCacheKeys:   []string{},
				commands:          []commands.DockerCommand{command},
				rootDir:           dir,
			}
		}(),
		func() testcase {
			dir, files := tempDirAndFile(t)
			file := files[0]
			filePath := filepath.Join(dir, file)
			ch := NewCompositeCache("", "meow")

			ch.AddPath(filePath, util.FileContext{})
			hash, err := ch.Hash()
			if err != nil {
				t.Errorf("couldn't create hash %v", err)
			}
			command := MockDockerCommand{
				command:      "meow",
				contextFiles: []string{filePath},
				cacheCommand: MockCachedDockerCommand{
					contextFiles: []string{filePath},
				},
			}

			destDir := t.TempDir()
			return testcase{
				description: "fake command cache enabled with tar compression disabled and key in cache",
				opts:        &config.KanikoOptions{Cache: true, CompressedCaching: false},
				config:      &v1.ConfigFile{Config: v1.Config{WorkingDir: destDir}},
				layerCache: &fakeLayerCache{
					retrieve: true,
				},
				expectedCacheKeys: []string{hash},
				pushedCacheKeys:   []string{},
				commands:          []commands.DockerCommand{command},
				rootDir:           dir,
			}
		}(),
		{
			description: "use new run",
			opts:        &config.KanikoOptions{RunV2: true},
		},
		{
			description:        "single snapshot",
			opts:               &config.KanikoOptions{SingleSnapshot: true},
			shouldInitSnapshot: true,
		},
		{
			description: "fake command cache disabled and key not in cache",
			opts:        &config.KanikoOptions{Cache: false},
		},
		{
			description: "fake command cache disabled and key in cache",
			opts:        &config.KanikoOptions{Cache: false},
			layerCache: &fakeLayerCache{
				retrieve: true,
			},
		},
		func() testcase {
			dir, filenames := tempDirAndFile(t)
			filename := filenames[0]
			filepath := filepath.Join(dir, filename)

			tarContent := generateTar(t, dir, filename)

			ch := NewCompositeCache("", fmt.Sprintf("COPY %s foo.txt", filename))
			ch.AddPath(filepath, util.FileContext{})

			hash, err := ch.Hash()
			if err != nil {
				t.Errorf("couldn't create hash %v", err)
			}
			copyCommandCacheKey := hash
			dockerFile := fmt.Sprintf(`
		FROM ubuntu:16.04
		COPY %s foo.txt
		`, filename)
			f, _ := os.CreateTemp("", "")
			os.WriteFile(f.Name(), []byte(dockerFile), 0755)
			opts := &config.KanikoOptions{
				DockerfilePath:  f.Name(),
				Cache:           true,
				CacheCopyLayers: true,
			}
			testStages, metaArgs, err := dockerfile.ParseStages(opts)
			if err != nil {
				t.Errorf("Failed to parse test dockerfile to stages: %s", err)
			}

			kanikoStages, err := dockerfile.MakeKanikoStages(opts, testStages, metaArgs)
			if err != nil {
				t.Errorf("Failed to parse stages to Kaniko Stages: %s", err)
			}
			_ = ResolveCrossStageInstructions(kanikoStages)
			stage := kanikoStages[0]

			cmds := stage.Commands

			return testcase{
				description: "copy command cache enabled and key in cache",
				opts:        opts,
				image: fakeImage{
					ImageLayers: []v1.Layer{
						fakeLayer{
							TarContent: tarContent,
						},
					},
				},
				layerCache: &fakeLayerCache{
					retrieve: true,
					img: fakeImage{
						ImageLayers: []v1.Layer{
							fakeLayer{
								TarContent: tarContent,
							},
						},
					},
				},
				rootDir:           dir,
				expectedCacheKeys: []string{copyCommandCacheKey},
				// CachingCopyCommand is not pushed to the cache
				pushedCacheKeys: []string{},
				commands:        getCommands(util.FileContext{Root: dir}, cmds, true, false),
				fileName:        filename,
			}
		}(),
		func() testcase {
			dir, filenames := tempDirAndFile(t)
			filename := filenames[0]
			tarContent := []byte{}
			destDir := t.TempDir()
			filePath := filepath.Join(dir, filename)
			ch := NewCompositeCache("", fmt.Sprintf("COPY %s foo.txt", filename))
			ch.AddPath(filePath, util.FileContext{})

			hash, err := ch.Hash()
			if err != nil {
				t.Errorf("couldn't create hash %v", err)
			}
			dockerFile := fmt.Sprintf(`
FROM ubuntu:16.04
COPY %s foo.txt
`, filename)
			f, _ := os.CreateTemp("", "")
			os.WriteFile(f.Name(), []byte(dockerFile), 0755)
			opts := &config.KanikoOptions{
				DockerfilePath:  f.Name(),
				Cache:           true,
				CacheCopyLayers: true,
			}

			testStages, metaArgs, err := dockerfile.ParseStages(opts)
			if err != nil {
				t.Errorf("Failed to parse test dockerfile to stages: %s", err)
			}

			kanikoStages, err := dockerfile.MakeKanikoStages(opts, testStages, metaArgs)
			if err != nil {
				t.Errorf("Failed to parse stages to Kaniko Stages: %s", err)
			}
			_ = ResolveCrossStageInstructions(kanikoStages)
			stage := kanikoStages[0]

			cmds := stage.Commands
			return testcase{
				description: "copy command cache enabled and key is not in cache",
				opts:        opts,
				config:      &v1.ConfigFile{Config: v1.Config{WorkingDir: destDir}},
				layerCache:  &fakeLayerCache{},
				image: fakeImage{
					ImageLayers: []v1.Layer{
						fakeLayer{
							TarContent: tarContent,
						},
					},
				},
				rootDir:           dir,
				expectedCacheKeys: []string{hash},
				pushedCacheKeys:   []string{hash},
				commands:          getCommands(util.FileContext{Root: dir}, cmds, true, false),
				fileName:          filename,
			}
		}(),
		func() testcase {
			dir, filenames := tempDirAndFile(t)
			filename := filenames[0]
			tarContent := generateTar(t, filename)

			destDir := t.TempDir()
			filePath := filepath.Join(dir, filename)

			ch := NewCompositeCache("", "RUN foobar")

			hash1, err := ch.Hash()
			if err != nil {
				t.Errorf("couldn't create hash %v", err)
			}

			ch.AddKey(fmt.Sprintf("COPY %s bar.txt", filename))
			ch.AddPath(filePath, util.FileContext{})

			hash2, err := ch.Hash()
			if err != nil {
				t.Errorf("couldn't create hash %v", err)
			}
			ch = NewCompositeCache("", fmt.Sprintf("COPY %s foo.txt", filename))
			ch.AddKey(fmt.Sprintf("COPY %s bar.txt", filename))
			ch.AddPath(filePath, util.FileContext{})

			image := fakeImage{
				ImageLayers: []v1.Layer{
					fakeLayer{
						TarContent: tarContent,
					},
				},
			}

			dockerFile := fmt.Sprintf(`
FROM ubuntu:16.04
RUN foobar
COPY %s bar.txt
`, filename)
			f, _ := os.CreateTemp("", "")
			os.WriteFile(f.Name(), []byte(dockerFile), 0755)
			opts := &config.KanikoOptions{
				DockerfilePath: f.Name(),
			}

			testStages, metaArgs, err := dockerfile.ParseStages(opts)
			if err != nil {
				t.Errorf("Failed to parse test dockerfile to stages: %s", err)
			}

			kanikoStages, err := dockerfile.MakeKanikoStages(opts, testStages, metaArgs)
			if err != nil {
				t.Errorf("Failed to parse stages to Kaniko Stages: %s", err)
			}
			_ = ResolveCrossStageInstructions(kanikoStages)
			stage := kanikoStages[0]

			cmds := stage.Commands
			return testcase{
				description: "cached run command followed by uncached copy command results in consistent read and write hashes",
				opts:        &config.KanikoOptions{Cache: true, CacheCopyLayers: true, CacheRunLayers: true},
				rootDir:     dir,
				config:      &v1.ConfigFile{Config: v1.Config{WorkingDir: destDir}},
				layerCache: &fakeLayerCache{
					keySequence: []string{hash1},
					img:         image,
				},
				image: image,
				// hash1 is the read cachekey for the first layer
				expectedCacheKeys: []string{hash1, hash2},
				pushedCacheKeys:   []string{hash2},
				commands:          getCommands(util.FileContext{Root: dir}, cmds, true, true),
			}
		}(),
		func() testcase {
			dir, filenames := tempDirAndFile(t)
			filename := filenames[0]
			tarContent := generateTar(t, filename)

			destDir := t.TempDir()

			filePath := filepath.Join(dir, filename)

			ch := NewCompositeCache("", fmt.Sprintf("COPY %s bar.txt", filename))
			ch.AddPath(filePath, util.FileContext{})

			// copy hash
			_, err := ch.Hash()
			if err != nil {
				t.Errorf("couldn't create hash %v", err)
			}

			ch.AddKey("RUN foobar")

			// run hash
			runHash, err := ch.Hash()
			if err != nil {
				t.Errorf("couldn't create hash %v", err)
			}

			image := fakeImage{
				ImageLayers: []v1.Layer{
					fakeLayer{
						TarContent: tarContent,
					},
				},
			}

			dockerFile := fmt.Sprintf(`
FROM ubuntu:16.04
COPY %s bar.txt
RUN foobar
`, filename)
			f, _ := os.CreateTemp("", "")
			os.WriteFile(f.Name(), []byte(dockerFile), 0755)
			opts := &config.KanikoOptions{
				DockerfilePath: f.Name(),
			}

			testStages, metaArgs, err := dockerfile.ParseStages(opts)
			if err != nil {
				t.Errorf("Failed to parse test dockerfile to stages: %s", err)
			}

			kanikoStages, err := dockerfile.MakeKanikoStages(opts, testStages, metaArgs)
			if err != nil {
				t.Errorf("Failed to parse stages to Kaniko Stages: %s", err)
			}
			_ = ResolveCrossStageInstructions(kanikoStages)
			stage := kanikoStages[0]

			cmds := stage.Commands
			return testcase{
				description: "copy command followed by cached run command results in consistent read and write hashes",
				opts:        &config.KanikoOptions{Cache: true, CacheRunLayers: true},
				rootDir:     dir,
				config:      &v1.ConfigFile{Config: v1.Config{WorkingDir: destDir}},
				layerCache: &fakeLayerCache{
					keySequence: []string{runHash},
					img:         image,
				},
				image:             image,
				expectedCacheKeys: []string{runHash},
				pushedCacheKeys:   []string{},
				commands:          getCommands(util.FileContext{Root: dir}, cmds, false, true),
			}
		}(),
		func() testcase {
			dir, _ := tempDirAndFile(t)
			ch := NewCompositeCache("")
			ch.AddKey("|1")
			ch.AddKey("test=value")
			ch.AddKey("RUN foobar")
			hash, err := ch.Hash()
			if err != nil {
				t.Errorf("couldn't create hash %v", err)
			}

			command := MockDockerCommand{
				command:      "RUN foobar",
				contextFiles: []string{},
				cacheCommand: MockCachedDockerCommand{
					contextFiles: []string{},
				},
				argToCompositeCache: true,
			}

			return testcase{
				description: "cached run command with no build arg value used uses cached layer and does not push anything",
				config:      &v1.ConfigFile{Config: v1.Config{WorkingDir: dir}},
				opts:        &config.KanikoOptions{Cache: true},
				args: map[string]string{
					"test": "value",
				},
				expectedCacheKeys: []string{hash},
				commands:          []commands.DockerCommand{command},
				// layer key needs to be read.
				layerCache: &fakeLayerCache{
					img:         &fakeImage{ImageLayers: []v1.Layer{fakeLayer{}}},
					keySequence: []string{hash},
				},
				rootDir: dir,
			}
		}(),
		func() testcase {
			dir, _ := tempDirAndFile(t)

			ch := NewCompositeCache("")
			ch.AddKey("|1")
			ch.AddKey("arg=value")
			ch.AddKey("RUN $arg")
			hash, err := ch.Hash()
			if err != nil {
				t.Errorf("couldn't create hash %v", err)
			}

			command := MockDockerCommand{
				command:      "RUN $arg",
				contextFiles: []string{},
				cacheCommand: MockCachedDockerCommand{
					contextFiles: []string{},
				},
				argToCompositeCache: true,
			}

			return testcase{
				description: "cached run command with same build arg does not push layer",
				config:      &v1.ConfigFile{Config: v1.Config{WorkingDir: dir}},
				opts:        &config.KanikoOptions{Cache: true},
				args: map[string]string{
					"arg": "value",
				},
				// layer key that exists
				layerCache: &fakeLayerCache{
					img:         &fakeImage{ImageLayers: []v1.Layer{fakeLayer{}}},
					keySequence: []string{hash},
				},
				expectedCacheKeys: []string{hash},
				commands:          []commands.DockerCommand{command},
				rootDir:           dir,
			}
		}(),
		func() testcase {
			dir, _ := tempDirAndFile(t)

			ch1 := NewCompositeCache("")
			ch1.AddKey("RUN value")
			hash1, err := ch1.Hash()
			if err != nil {
				t.Errorf("couldn't create hash %v", err)
			}

			ch2 := NewCompositeCache("")
			ch2.AddKey("|1")
			ch2.AddKey("arg=anotherValue")
			ch2.AddKey("RUN $arg")
			hash2, err := ch2.Hash()
			if err != nil {
				t.Errorf("couldn't create hash %v", err)
			}
			command := MockDockerCommand{
				command:      "RUN $arg",
				contextFiles: []string{},
				cacheCommand: MockCachedDockerCommand{
					contextFiles: []string{},
				},
				argToCompositeCache: true,
			}

			return testcase{
				description: "cached run command with another build arg pushes layer",
				config:      &v1.ConfigFile{Config: v1.Config{WorkingDir: dir}},
				opts:        &config.KanikoOptions{Cache: true},
				args: map[string]string{
					"arg": "anotherValue",
				},
				// layer for arg=value already exists
				layerCache: &fakeLayerCache{
					img:         &fakeImage{ImageLayers: []v1.Layer{fakeLayer{}}},
					keySequence: []string{hash1},
				},
				expectedCacheKeys: []string{hash2},
				pushedCacheKeys:   []string{hash2},
				commands:          []commands.DockerCommand{command},
				rootDir:           dir,
			}
		}(),
		{
			description:    "fs unpacked",
			opts:           &config.KanikoOptions{InitialFSUnpacked: true},
			stage:          config.KanikoStage{Index: 0},
			crossStageDeps: map[int][]string{0: {"some-dep"}},
			mockGetFSFromImage: func(root string, img v1.Image, extract util.ExtractFunction) ([]string, error) {
				return nil, fmt.Errorf("getFSFromImage shouldn't be called if fs is already unpacked")
			},
		},
	}
	for _, tc := range testCases {
		t.Run(tc.description, func(t *testing.T) {
			var fileName string
			if tc.commands == nil {
				file, err := os.CreateTemp("", "foo")
				if err != nil {
					t.Error(err)
				}
				command := MockDockerCommand{
					contextFiles: []string{file.Name()},
					cacheCommand: MockCachedDockerCommand{
						contextFiles: []string{file.Name()},
					},
				}
				tc.commands = []commands.DockerCommand{command}
				fileName = file.Name()
			} else {
				fileName = tc.fileName
			}

			cf := tc.config
			if cf == nil {
				cf = &v1.ConfigFile{
					Config: v1.Config{
						Env: make([]string, 0),
					},
				}
			}

			snap := &fakeSnapShotter{file: fileName}
			lc := tc.layerCache
			if lc == nil {
				lc = &fakeLayerCache{}
			}
			keys := []string{}
			sb := &stageBuilder{
				args:        dockerfile.NewBuildArgs([]string{}), //required or code will panic
				image:       tc.image,
				opts:        tc.opts,
				cf:          cf,
				snapshotter: snap,
				layerCache:  lc,
				pushLayerToCache: func(_ *config.KanikoOptions, cacheKey, _, _ string) error {
					keys = append(keys, cacheKey)
					return nil
				},
			}
			sb.cmds = tc.commands
			for key, value := range tc.args {
				sb.args.AddArg(key, &value)
			}
			tmp := config.RootDir
			if tc.rootDir != "" {
				config.RootDir = tc.rootDir
			}
			sb.stage = tc.stage
			sb.crossStageDeps = tc.crossStageDeps
			if tc.mockGetFSFromImage != nil {
				original := getFSFromImage
				defer func() { getFSFromImage = original }()
				getFSFromImage = tc.mockGetFSFromImage
			}
			err := sb.build()
			if err != nil {
				t.Errorf("Expected error to be nil but was %v", err)
			}
			if tc.shouldInitSnapshot && !snap.initialized {
				t.Errorf("Snapshotter was not initialized but should have been")
			} else if !tc.shouldInitSnapshot && snap.initialized {
				t.Errorf("Snapshotter was initialized but should not have been")
			}
			assertCacheKeys(t, tc.expectedCacheKeys, lc.receivedKeys, "receive")
			assertCacheKeys(t, tc.pushedCacheKeys, keys, "push")

			config.RootDir = tmp

		})
	}
}

func assertCacheKeys(t *testing.T, expectedCacheKeys, actualCacheKeys []string, description string) {
	if len(expectedCacheKeys) != len(actualCacheKeys) {
		t.Errorf("expected to %v %v keys but was %v", description, len(expectedCacheKeys), len(actualCacheKeys))
	}

	sort.Slice(expectedCacheKeys, func(x, y int) bool {
		return expectedCacheKeys[x] > expectedCacheKeys[y]
	})
	sort.Slice(actualCacheKeys, func(x, y int) bool {
		return actualCacheKeys[x] > actualCacheKeys[y]
	})

	if len(expectedCacheKeys) != len(actualCacheKeys) {
		t.Errorf("expected %v to equal %v", actualCacheKeys, expectedCacheKeys)
	}

	for i, key := range expectedCacheKeys {
		if key != actualCacheKeys[i] {
			t.Errorf("expected to %v keys %d to be %v but was %v %v", description, i, key, actualCacheKeys[i], actualCacheKeys)
		}
	}
}

func getCommands(fileContext util.FileContext, cmds []instructions.Command, cacheCopy, cacheRun bool) []commands.DockerCommand {
	outCommands := make([]commands.DockerCommand, 0)
	for _, c := range cmds {
		cmd, err := commands.GetCommand(
			c,
			fileContext,
			false,
			cacheCopy,
			cacheRun,
		)
		if err != nil {
			panic(err)
		}
		outCommands = append(outCommands, cmd)
	}
	return outCommands
}

func tempDirAndFile(t *testing.T) (string, []string) {
	filenames := []string{"bar.txt"}

	dir := t.TempDir()
	for _, filename := range filenames {
		filepath := filepath.Join(dir, filename)
		err := os.WriteFile(filepath, []byte(`meow`), 0777)
		if err != nil {
			t.Errorf("could not create temp file %v", err)
		}
	}

	return dir, filenames
}

func generateTar(t *testing.T, dir string, fileNames ...string) []byte {
	buf := bytes.NewBuffer([]byte{})
	writer := tar.NewWriter(buf)
	defer writer.Close()

	for _, filename := range fileNames {
		filePath := filepath.Join(dir, filename)
		info, err := os.Stat(filePath)
		if err != nil {
			t.Errorf("could not get file info for temp file %v", err)
		}
		hdr, err := tar.FileInfoHeader(info, filename)
		if err != nil {
			t.Errorf("could not get tar header for temp file %v", err)
		}

		if err := writer.WriteHeader(hdr); err != nil {
			t.Errorf("could not write tar header %v", err)
		}

		content, err := os.ReadFile(filePath)
		if err != nil {
			t.Errorf("could not read tempfile %v", err)
		}

		if _, err := writer.Write(content); err != nil {
			t.Errorf("could not write file contents to tar")
		}
	}
	return buf.Bytes()
}

func hashCompositeKeys(t *testing.T, ck1 CompositeCache, ck2 CompositeCache) (string, string) {
	key1, err := ck1.Hash()
	if err != nil {
		t.Errorf("could not hash composite key due to %s", err)
	}
	key2, err := ck2.Hash()
	if err != nil {
		t.Errorf("could not hash composite key due to %s", err)
	}
	return key1, key2
}

func Test_stageBuild_populateCompositeKeyForCopyCommand(t *testing.T) {
	// See https://github.com/GoogleContainerTools/kaniko/issues/589

	for _, tc := range []struct {
		description      string
		command          string
		expectedCacheKey string
	}{
		{
			description: "multi-stage copy command",
			// dont use digest from previoust stage for COPY
			command:          "COPY --from=0 foo.txt bar.txt",
			expectedCacheKey: "COPY --from=0 foo.txt bar.txt",
		},
		{
			description:      "copy command",
			command:          "COPY foo.txt bar.txt",
			expectedCacheKey: "COPY foo.txt bar.txt",
		},
	} {
		t.Run(tc.description, func(t *testing.T) {
			instructions, err := dockerfile.ParseCommands([]string{tc.command})
			if err != nil {
				t.Fatal(err)
			}

			fc := util.FileContext{Root: "workspace"}
			copyCommand, err := commands.GetCommand(instructions[0], fc, false, true, true)
			if err != nil {
				t.Fatal(err)
			}

			for _, useCacheCommand := range []bool{false, true} {
				t.Run(fmt.Sprintf("CacheCommand=%t", useCacheCommand), func(t *testing.T) {
					var cmd commands.DockerCommand = copyCommand
					if useCacheCommand {
						cmd = copyCommand.(*commands.CopyCommand).CacheCommand(nil)
					}

					sb := &stageBuilder{
						fileContext: fc,
						stageIdxToDigest: map[string]string{
							"0": "some-digest",
						},
						digestToCacheKey: map[string]string{
							"some-digest": "some-cache-key",
						},
					}

					ck := CompositeCache{}
					ck, err = sb.populateCompositeKey(
						cmd,
						[]string{},
						ck,
						dockerfile.NewBuildArgs([]string{}),
						[]string{},
					)
					if err != nil {
						t.Fatal(err)
					}

					actualCacheKey := ck.Key()
					if tc.expectedCacheKey != actualCacheKey {
						t.Errorf(
							"Expected cache key to be %s, was %s",
							tc.expectedCacheKey,
							actualCacheKey,
						)
					}

				})
			}
		})
	}
}

func Test_ResolveCrossStageInstructions(t *testing.T) {
	df := `
	FROM scratch
	RUN echo hi > /hi

	FROM scratch AS second
	COPY --from=0 /hi /hi2

	FROM scratch AS tHiRd
	COPY --from=second /hi2 /hi3
	COPY --from=1 /hi2 /hi3

	FROM scratch
	COPY --from=thIrD /hi3 /hi4
	COPY --from=third /hi3 /hi4
	COPY --from=2 /hi3 /hi4
	`
	stages, metaArgs, err := dockerfile.Parse([]byte(df))
	if err != nil {
		t.Fatal(err)
	}
	opts := &config.KanikoOptions{}
	kanikoStages, err := dockerfile.MakeKanikoStages(opts, stages, metaArgs)
	if err != nil {
		t.Fatal(err)
	}
	stageToIdx := ResolveCrossStageInstructions(kanikoStages)
	for index, stage := range stages {
		if index == 0 {
			continue
		}
		expectedStage := strconv.Itoa(index - 1)
		for _, command := range stage.Commands {
			copyCmd := command.(*instructions.CopyCommand)
			if copyCmd.From != expectedStage {
				t.Fatalf("unexpected copy command: %s resolved to stage %s, expected %s", copyCmd.String(), copyCmd.From, expectedStage)
			}
		}

		expectedMap := map[string]string{"second": "1", "third": "2"}
		testutil.CheckDeepEqual(t, expectedMap, stageToIdx)
	}
}

func Test_stageBuilder_saveSnapshotToLayer(t *testing.T) {
	dir, files := tempDirAndFile(t)
	type fields struct {
		stage            config.KanikoStage
		image            v1.Image
		cf               *v1.ConfigFile
		baseImageDigest  string
		finalCacheKey    string
		opts             *config.KanikoOptions
		fileContext      util.FileContext
		cmds             []commands.DockerCommand
		args             *dockerfile.BuildArgs
		crossStageDeps   map[int][]string
		digestToCacheKey map[string]string
		stageIdxToDigest map[string]string
		snapshotter      snapShotter
		layerCache       cache.LayerCache
		pushLayerToCache cachePusher
	}
	type args struct {
		tarPath string
	}
	tests := []struct {
		name              string
		fields            fields
		args              args
		expectedMediaType types.MediaType
		expectedDiff      v1.Hash
		expectedDigest    v1.Hash
		wantErr           bool
	}{
		{
			name: "oci image",
			fields: fields{
				image: ociFakeImage{},
				opts: &config.KanikoOptions{
					ForceBuildMetadata: true,
				},
			},
			args: args{
				tarPath: filepath.Join(dir, files[0]),
			},
			expectedMediaType: types.OCILayer,
			expectedDiff: v1.Hash{
				Algorithm: "sha256",
				Hex:       "404cdd7bc109c432f8cc2443b45bcfe95980f5107215c645236e577929ac3e52",
			},
			expectedDigest: v1.Hash{
				Algorithm: "sha256",
				Hex:       "1dc5887a31ec6b388646be46c5f0b2036f92f4cbba50d12163a8be4074565a91",
			},
		},
		{
			name: "docker image",
			fields: fields{
				image: fakeImage{},
				opts: &config.KanikoOptions{
					ForceBuildMetadata: true,
				},
			},
			args: args{
				tarPath: filepath.Join(dir, files[0]),
			},
			expectedMediaType: types.DockerLayer,
			expectedDiff: v1.Hash{
				Algorithm: "sha256",
				Hex:       "404cdd7bc109c432f8cc2443b45bcfe95980f5107215c645236e577929ac3e52",
			},
			expectedDigest: v1.Hash{
				Algorithm: "sha256",
				Hex:       "1dc5887a31ec6b388646be46c5f0b2036f92f4cbba50d12163a8be4074565a91",
			},
		},
		{
			name: "oci image, zstd compression",
			fields: fields{
				image: ociFakeImage{},
				opts: &config.KanikoOptions{
					ForceBuildMetadata: true,
					Compression:        config.ZStd,
				},
			},
			args: args{
				tarPath: filepath.Join(dir, files[0]),
			},
			expectedMediaType: types.OCILayerZStd,
			expectedDiff: v1.Hash{
				Algorithm: "sha256",
				Hex:       "404cdd7bc109c432f8cc2443b45bcfe95980f5107215c645236e577929ac3e52",
			},
			expectedDigest: v1.Hash{
				Algorithm: "sha256",
				Hex:       "28369c11d9b68c9877781eaf4d8faffb4d0ada1900a1fb83ad452e58a072b45b",
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			s := &stageBuilder{
				stage:            tt.fields.stage,
				image:            tt.fields.image,
				cf:               tt.fields.cf,
				baseImageDigest:  tt.fields.baseImageDigest,
				finalCacheKey:    tt.fields.finalCacheKey,
				opts:             tt.fields.opts,
				fileContext:      tt.fields.fileContext,
				cmds:             tt.fields.cmds,
				args:             tt.fields.args,
				crossStageDeps:   tt.fields.crossStageDeps,
				digestToCacheKey: tt.fields.digestToCacheKey,
				stageIdxToDigest: tt.fields.stageIdxToDigest,
				snapshotter:      tt.fields.snapshotter,
				layerCache:       tt.fields.layerCache,
				pushLayerToCache: tt.fields.pushLayerToCache,
			}
			got, err := s.saveSnapshotToLayer(tt.args.tarPath)
			if (err != nil) != tt.wantErr {
				t.Errorf("stageBuilder.saveSnapshotToLayer() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if mt, _ := got.MediaType(); mt != tt.expectedMediaType {
				t.Errorf("expected mediatype %s, got %s", tt.expectedMediaType, mt)
				return
			}
			if diff, _ := got.DiffID(); diff != tt.expectedDiff {
				t.Errorf("expected diff %s, got %s", tt.expectedDiff, diff)
				return
			}
			if digest, _ := got.Digest(); digest != tt.expectedDigest {
				t.Errorf("expected digest %s, got %s", tt.expectedDigest, digest)
				return
			}
		})
	}
}

func Test_stageBuilder_convertLayerMediaType(t *testing.T) {
	type fields struct {
		stage            config.KanikoStage
		image            v1.Image
		cf               *v1.ConfigFile
		baseImageDigest  string
		finalCacheKey    string
		opts             *config.KanikoOptions
		fileContext      util.FileContext
		cmds             []commands.DockerCommand
		args             *dockerfile.BuildArgs
		crossStageDeps   map[int][]string
		digestToCacheKey map[string]string
		stageIdxToDigest map[string]string
		snapshotter      snapShotter
		layerCache       cache.LayerCache
		pushLayerToCache cachePusher
	}
	type args struct {
		layer v1.Layer
	}
	tests := []struct {
		name              string
		fields            fields
		args              args
		expectedMediaType types.MediaType
		wantErr           bool
	}{
		{
			name: "docker image w/ docker layer",
			fields: fields{
				image: fakeImage{},
			},
			args: args{
				layer: fakeLayer{
					mediaType: types.DockerLayer,
				},
			},
			expectedMediaType: types.DockerLayer,
		},
		{
			name: "oci image w/ oci layer",
			fields: fields{
				image: ociFakeImage{},
			},
			args: args{
				layer: fakeLayer{
					mediaType: types.OCILayer,
				},
			},
			expectedMediaType: types.OCILayer,
		},
		{
			name: "oci image w/ convertable docker layer",
			fields: fields{
				image: ociFakeImage{},
				opts:  &config.KanikoOptions{},
			},
			args: args{
				layer: fakeLayer{
					mediaType: types.DockerLayer,
				},
			},
			expectedMediaType: types.OCILayer,
		},
		{
			name: "oci image w/ convertable docker layer and zstd compression",
			fields: fields{
				image: ociFakeImage{},
				opts: &config.KanikoOptions{
					Compression: config.ZStd,
				},
			},
			args: args{
				layer: fakeLayer{
					mediaType: types.DockerLayer,
				},
			},
			expectedMediaType: types.OCILayerZStd,
		},
		{
			name: "docker image and oci zstd layer",
			fields: fields{
				image: dockerFakeImage{},
				opts:  &config.KanikoOptions{},
			},
			args: args{
				layer: fakeLayer{
					mediaType: types.OCILayerZStd,
				},
			},
			expectedMediaType: types.DockerLayer,
		},
		{
			name: "docker image w/ uncovertable oci image",
			fields: fields{
				image: dockerFakeImage{},
				opts:  &config.KanikoOptions{},
			},
			args: args{
				layer: fakeLayer{
					mediaType: types.OCIUncompressedRestrictedLayer,
				},
			},
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			s := &stageBuilder{
				stage:            tt.fields.stage,
				image:            tt.fields.image,
				cf:               tt.fields.cf,
				baseImageDigest:  tt.fields.baseImageDigest,
				finalCacheKey:    tt.fields.finalCacheKey,
				opts:             tt.fields.opts,
				fileContext:      tt.fields.fileContext,
				cmds:             tt.fields.cmds,
				args:             tt.fields.args,
				crossStageDeps:   tt.fields.crossStageDeps,
				digestToCacheKey: tt.fields.digestToCacheKey,
				stageIdxToDigest: tt.fields.stageIdxToDigest,
				snapshotter:      tt.fields.snapshotter,
				layerCache:       tt.fields.layerCache,
				pushLayerToCache: tt.fields.pushLayerToCache,
			}
			got, err := s.convertLayerMediaType(tt.args.layer)
			if (err != nil) != tt.wantErr {
				t.Errorf("stageBuilder.convertLayerMediaType() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if err == nil {
				mt, _ := got.MediaType()
				if mt != tt.expectedMediaType {
					t.Errorf("stageBuilder.convertLayerMediaType() = %v, want %v", mt, tt.expectedMediaType)
				}
			}
		})
	}
}
