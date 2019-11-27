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
	"io/ioutil"
	"os"
	"path/filepath"
	"reflect"
	"sort"
	"testing"

	"github.com/GoogleContainerTools/kaniko/pkg/commands"
	"github.com/GoogleContainerTools/kaniko/pkg/config"
	"github.com/GoogleContainerTools/kaniko/pkg/dockerfile"
	"github.com/GoogleContainerTools/kaniko/testutil"
	"github.com/google/go-cmp/cmp"
	v1 "github.com/google/go-containerregistry/pkg/v1"
	"github.com/google/go-containerregistry/pkg/v1/empty"
	"github.com/google/go-containerregistry/pkg/v1/mutate"
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

type MockCommand struct {
	name string
}

func (m *MockCommand) Name() string {
	return m.name
}

func Test_stageBuilder_shouldTakeSnapshot(t *testing.T) {
	commands := []instructions.Command{
		&MockCommand{name: "command1"},
		&MockCommand{name: "command2"},
		&MockCommand{name: "command3"},
	}

	stage := instructions.Stage{
		Commands: commands,
	}

	type fields struct {
		stage config.KanikoStage
		opts  *config.KanikoOptions
	}
	type args struct {
		index int
		files []string
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
					Stage: stage,
				},
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
					Stage: stage,
				},
			},
			args: args{
				index: len(commands) - 1,
			},
			want: true,
		},
		{
			name: "not final stage not last command",
			fields: fields{
				stage: config.KanikoStage{
					Final: false,
					Stage: stage,
				},
			},
			args: args{
				index: 0,
			},
			want: true,
		},
		{
			name: "caching enabled intermediate container",
			fields: fields{
				stage: config.KanikoStage{
					Final: false,
					Stage: stage,
				},
				opts: &config.KanikoOptions{Cache: true},
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
			}
			if got := s.shouldTakeSnapshot(tt.args.index, tt.args.files); got != tt.want {
				t.Errorf("stageBuilder.shouldTakeSnapshot() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestCalculateDependencies(t *testing.T) {
	type args struct {
		dockerfile string
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
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			f, _ := ioutil.TempFile("", "")
			ioutil.WriteFile(f.Name(), []byte(tt.args.dockerfile), 0755)
			opts := &config.KanikoOptions{
				DockerfilePath: f.Name(),
			}

			got, err := CalculateDependencies(opts)
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
			if err != nil {
				t.Errorf("error creating tmpdir: %s", err)
			}
			defer os.RemoveAll(tmpDir)

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

			args := []string{}
			for _, arg := range tt.args {
				args = append(args, filepath.Join(tmpDir, arg))
			}
			got, err := filesToSave(args)
			if err != nil {
				t.Errorf("got err: %s", err)
			}
			want := []string{}
			for _, w := range tt.want {
				want = append(want, filepath.Join(tmpDir, w))
			}
			sort.Strings(want)
			sort.Strings(got)
			if !reflect.DeepEqual(got, want) {
				t.Errorf("filesToSave() = %v, want %v", got, want)
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
		actual, _ := initializeConfig(img)
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
			sb := &stageBuilder{opts: tc.opts, cf: cf, snapshotter: snap, layerCache: lc}
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

func Test_stageBuilder_build(t *testing.T) {
	type testcase struct {
		description       string
		opts              *config.KanikoOptions
		layerCache        *fakeLayerCache
		expectedCacheKeys []string
		pushedCacheKeys   []string
		commands          []commands.DockerCommand
		fileName          string
		rootDir           string
		image             v1.Image
		config            *v1.ConfigFile
	}
	// The two copy command test cases use the same filesystem content and should generate the same cache keys. If they don't then something is wrong. They share this variable to help ensure that.
	copyCommandCacheKey := "7263908b66952551d89fd895ffb067e2e30f474be9f38a8f1792af2b6df7c6e3"
	tempDirAndFile := func() (string, string) {
		dir, err := ioutil.TempDir("", "foo")
		if err != nil {
			t.Errorf("could not create temp dir %v", err)
		}
		filename := "bar.txt"
		filepath := filepath.Join(dir, filename)
		err = ioutil.WriteFile(filepath, []byte(`meow`), 0777)
		if err != nil {
			t.Errorf("could not create temp file %v", err)
		}
		buf := bytes.NewBuffer([]byte{})
		writer := tar.NewWriter(buf)
		defer writer.Close()

		return dir, filename
	}
	testCases := []testcase{
		{
			description:       "fake command cache enabled but key not in cache",
			opts:              &config.KanikoOptions{Cache: true},
			expectedCacheKeys: []string{"2cd95a0195a42f2873273b7e8c970e3a87970bd0e5d330b3c7d068e7419e5017"},
			pushedCacheKeys:   []string{"2cd95a0195a42f2873273b7e8c970e3a87970bd0e5d330b3c7d068e7419e5017"},
		},
		{
			description: "fake command cache enabled and key in cache",
			opts:        &config.KanikoOptions{Cache: true},
			layerCache: &fakeLayerCache{
				retrieve: true,
			},
			expectedCacheKeys: []string{"2cd95a0195a42f2873273b7e8c970e3a87970bd0e5d330b3c7d068e7419e5017"},
			pushedCacheKeys:   []string{"2cd95a0195a42f2873273b7e8c970e3a87970bd0e5d330b3c7d068e7419e5017"},
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
			dir, filename := tempDirAndFile()
			filepath := filepath.Join(dir, filename)

			buf := bytes.NewBuffer([]byte{})
			writer := tar.NewWriter(buf)
			defer writer.Close()

			info, err := os.Stat(filepath)
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

			content, err := ioutil.ReadFile(filepath)
			if err != nil {
				t.Errorf("could not read tempfile %v", err)
			}

			if _, err := writer.Write(content); err != nil {
				t.Errorf("could not write file contents to tar")
			}
			tarContent := buf.Bytes()
			return testcase{
				description: "copy command cache enabled and key in cache",
				opts:        &config.KanikoOptions{Cache: true},
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
				commands: func() []commands.DockerCommand {
					cmd, err := commands.GetCommand(
						&instructions.CopyCommand{
							SourcesAndDest: []string{
								filename, "foo.txt",
							},
						},
						dir,
					)
					if err != nil {
						panic(err)
					}
					return []commands.DockerCommand{
						cmd,
					}
				}(),
				fileName: filename,
			}
		}(),
		func() testcase {
			dir, filename := tempDirAndFile()

			tarContent := []byte{}
			destDir, err := ioutil.TempDir("", "baz")
			if err != nil {
				t.Errorf("could not create temp dir %v", err)
			}
			return testcase{
				description: "copy command cache enabled and key is not in cache",
				opts:        &config.KanikoOptions{Cache: true},
				config:      &v1.ConfigFile{Config: v1.Config{WorkingDir: destDir}},
				layerCache: &fakeLayerCache{
					retrieve: false,
				},
				image: fakeImage{
					ImageLayers: []v1.Layer{
						fakeLayer{
							TarContent: tarContent,
						},
					},
				},
				rootDir:           dir,
				expectedCacheKeys: []string{copyCommandCacheKey},
				pushedCacheKeys:   []string{copyCommandCacheKey},
				commands: func() []commands.DockerCommand {
					cmd, err := commands.GetCommand(
						&instructions.CopyCommand{
							SourcesAndDest: []string{
								filename, "foo.txt",
							},
						},
						dir,
					)
					if err != nil {
						panic(err)
					}
					return []commands.DockerCommand{
						cmd,
					}
				}(),
				fileName: filename,
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
				args:        &dockerfile.BuildArgs{}, //required or code will panic
				image:       tc.image,
				opts:        tc.opts,
				cf:          cf,
				snapshotter: snap,
				layerCache:  lc,
				pushCache: func(_ *config.KanikoOptions, cacheKey, _, _ string) error {
					keys = append(keys, cacheKey)
					return nil
				},
			}
			sb.cmds = tc.commands
			tmp := commands.RootDir
			if tc.rootDir != "" {
				commands.RootDir = tc.rootDir
			}
			err := sb.build()
			if err != nil {
				t.Errorf("Expected error to be nil but was %v", err)
			}

			if len(tc.expectedCacheKeys) != len(lc.receivedKeys) {
				t.Errorf("expected to receive %v keys but was %v", len(tc.expectedCacheKeys), len(lc.receivedKeys))
			}
			for _, key := range tc.expectedCacheKeys {
				match := false
				for _, receivedKey := range lc.receivedKeys {
					if key == receivedKey {
						match = true
						break
					}
				}
				if !match {
					t.Errorf("expected received keys to include %v but did not %v", key, lc.receivedKeys)
				}
			}
			if len(tc.pushedCacheKeys) != len(keys) {
				t.Errorf("expected to push %v keys but was %v", len(tc.pushedCacheKeys), len(keys))
			}
			for _, key := range tc.pushedCacheKeys {
				match := false
				for _, pushedKey := range keys {
					if key == pushedKey {
						match = true
						break
					}
				}
				if !match {
					t.Errorf("expected pushed keys to include %v but did not %v", key, keys)
				}
			}
			commands.RootDir = tmp

		})
	}
}
