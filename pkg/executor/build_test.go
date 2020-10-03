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
	"io/ioutil"
	"os"
	"path/filepath"
	"reflect"
	"sort"
	"strconv"
	"testing"

	"github.com/GoogleContainerTools/kaniko/pkg/commands"
	"github.com/GoogleContainerTools/kaniko/pkg/config"
	"github.com/GoogleContainerTools/kaniko/pkg/dockerfile"
	"github.com/GoogleContainerTools/kaniko/pkg/util"
	"github.com/GoogleContainerTools/kaniko/testutil"
	"github.com/google/go-cmp/cmp"
	v1 "github.com/google/go-containerregistry/pkg/v1"
	"github.com/google/go-containerregistry/pkg/v1/empty"
	"github.com/google/go-containerregistry/pkg/v1/mutate"
	"github.com/google/go-containerregistry/pkg/v1/partial"
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

			f, _ := ioutil.TempFile("", "")
			ioutil.WriteFile(f.Name(), []byte(tt.args.dockerfile), 0755)
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
			tmpDir, err := ioutil.TempDir("", "")
			original := config.RootDir
			config.RootDir = tmpDir
			if err != nil {
				t.Errorf("error creating tmpdir: %s", err)
			}
			defer func() {
				config.RootDir = original
				os.RemoveAll(tmpDir)
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
			snap := fakeSnapShotter{}
			lc := &fakeLayerCache{retrieve: tc.retrieve}
			sb := &stageBuilder{opts: tc.opts, cf: cf, snapshotter: snap, layerCache: lc,
				args: dockerfile.NewBuildArgs([]string{})}
			ck := CompositeCache{}
			file, err := ioutil.TempFile("", "foo")
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
	testCases := []struct {
		description string
		cmd1        stageContext
		cmd2        stageContext
		shdEqual    bool
	}{
		{
			description: "cache key for same command, different buildargs, args not used in command",
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
			shdEqual: true,
		},
		{
			description: "cache key for same command with same build args",
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
			description: "cache key for same command with same env",
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
			shdEqual: true,
		},
		{
			description: "cache key for same command with a build arg values",
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
			description: "cache key for same command with different env values",
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
			description: "cache key for different command same context",
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
	}
	for _, tc := range testCases {
		t.Run(tc.description, func(t *testing.T) {
			sb := &stageBuilder{fileContext: util.FileContext{Root: "workspace"}}
			ck := CompositeCache{}

			ck1, err := sb.populateCompositeKey(tc.cmd1.command, []string{}, ck, tc.cmd1.args, tc.cmd1.env)
			if err != nil {
				t.Errorf("Expected error to be nil but was %v", err)
			}
			ck2, err := sb.populateCompositeKey(tc.cmd2.command, []string{}, ck, tc.cmd2.args, tc.cmd2.env)
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
		description       string
		opts              *config.KanikoOptions
		args              map[string]string
		layerCache        *fakeLayerCache
		expectedCacheKeys []string
		pushedCacheKeys   []string
		commands          []commands.DockerCommand
		fileName          string
		rootDir           string
		image             v1.Image
		config            *v1.ConfigFile
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

			destDir, err := ioutil.TempDir("", "baz")
			if err != nil {
				t.Errorf("could not create temp dir %v", err)
			}
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

			destDir, err := ioutil.TempDir("", "baz")
			if err != nil {
				t.Errorf("could not create temp dir %v", err)
			}
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
			tarContent := generateTar(t, filename)

			destDir, err := ioutil.TempDir("", "baz")
			if err != nil {
				t.Errorf("could not create temp dir %v", err)
			}

			ch := NewCompositeCache("", fmt.Sprintf("RUN foobar"))

			hash1, err := ch.Hash()
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
RUN foobar
COPY %s bar.txt
`, filename)
			f, _ := ioutil.TempFile("", "")
			ioutil.WriteFile(f.Name(), []byte(dockerFile), 0755)
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
				description: "cached run command followed by copy command results in consistent read and write hashes",
				opts:        &config.KanikoOptions{Cache: true},
				rootDir:     dir,
				config:      &v1.ConfigFile{Config: v1.Config{WorkingDir: destDir}},
				layerCache: &fakeLayerCache{
					keySequence: []string{hash1},
					img:         image,
				},
				image: image,
				// hash1 is the read cachekey for the first layer
				expectedCacheKeys: []string{hash1},
				pushedCacheKeys:   []string{},
				commands:          getCommands(util.FileContext{Root: dir}, cmds),
			}
		}(),
		func() testcase {
			dir, filenames := tempDirAndFile(t)
			filename := filenames[0]
			tarContent := generateTar(t, filename)

			destDir, err := ioutil.TempDir("", "baz")
			if err != nil {
				t.Errorf("could not create temp dir %v", err)
			}

			filePath := filepath.Join(dir, filename)

			ch := NewCompositeCache("", fmt.Sprintf("COPY %s bar.txt", filename))
			ch.AddPath(filePath, util.FileContext{})

			// copy hash
			_, err = ch.Hash()
			if err != nil {
				t.Errorf("couldn't create hash %v", err)
			}

			ch.AddKey(fmt.Sprintf("RUN foobar"))

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
			f, _ := ioutil.TempFile("", "")
			ioutil.WriteFile(f.Name(), []byte(dockerFile), 0755)
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
				opts:        &config.KanikoOptions{Cache: true},
				rootDir:     dir,
				config:      &v1.ConfigFile{Config: v1.Config{WorkingDir: destDir}},
				layerCache: &fakeLayerCache{
					keySequence: []string{runHash},
					img:         image,
				},
				image:             image,
				expectedCacheKeys: []string{runHash},
				pushedCacheKeys:   []string{},
				commands:          getCommands(util.FileContext{Root: dir}, cmds),
			}
		}(),
		func() testcase {
			dir, _ := tempDirAndFile(t)
			ch := NewCompositeCache("")
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
			ch.AddKey("RUN value")
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
			ch2.AddKey("RUN anotherValue")
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
	}
	for _, tc := range testCases {
		t.Run(tc.description, func(t *testing.T) {
			var fileName string
			if tc.commands == nil {
				file, err := ioutil.TempFile("", "foo")
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

			snap := fakeSnapShotter{file: fileName}
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
			err := sb.build()
			if err != nil {
				t.Errorf("Expected error to be nil but was %v", err)
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

func getCommands(fileContext util.FileContext, cmds []instructions.Command) []commands.DockerCommand {
	outCommands := make([]commands.DockerCommand, 0)
	for _, c := range cmds {
		cmd, err := commands.GetCommand(
			c,
			fileContext,
			false,
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

	dir, err := ioutil.TempDir("", "foo")
	if err != nil {
		t.Errorf("could not create temp dir %v", err)
	}
	for _, filename := range filenames {
		filepath := filepath.Join(dir, filename)
		err = ioutil.WriteFile(filepath, []byte(`meow`), 0777)
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

		content, err := ioutil.ReadFile(filePath)
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
