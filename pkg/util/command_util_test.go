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

package util

import (
	"reflect"
	"sort"
	"testing"

	"github.com/GoogleContainerTools/kaniko/testutil"
)

var testURL = "https://github.com/GoogleContainerTools/runtimes-common/blob/master/LICENSE"

var testEnvReplacement = []struct {
	path         string
	envs         []string
	isFilepath   bool
	expectedPath string
}{
	{
		path: "/simple/path",
		envs: []string{
			"simple=/path/",
		},
		isFilepath:   true,
		expectedPath: "/simple/path",
	},
	{
		path: "/simple/path/",
		envs: []string{
			"simple=/path/",
		},
		isFilepath:   true,
		expectedPath: "/simple/path/",
	},
	{
		path: "${a}/b",
		envs: []string{
			"a=/path/",
			"b=/path2/",
		},
		isFilepath:   true,
		expectedPath: "/path/b",
	},
	{
		path: "/$a/b",
		envs: []string{
			"a=/path/",
			"b=/path2/",
		},
		isFilepath:   true,
		expectedPath: "/path/b",
	},
	{
		path: "/$a/b/",
		envs: []string{
			"a=/path/",
			"b=/path2/",
		},
		isFilepath:   true,
		expectedPath: "/path/b/",
	},
	{
		path: "\\$foo",
		envs: []string{
			"foo=/path/",
		},
		isFilepath:   true,
		expectedPath: "$foo",
	},
	{
		path: "8080/$protocol",
		envs: []string{
			"protocol=udp",
		},
		expectedPath: "8080/udp",
	},
	{
		path: "8080/$protocol",
		envs: []string{
			"protocol=udp",
		},
		expectedPath: "8080/udp",
	},
	{
		path: "$url",
		envs: []string{
			"url=http://example.com",
		},
		isFilepath:   true,
		expectedPath: "http://example.com",
	},
	{
		path: "$url",
		envs: []string{
			"url=http://example.com",
		},
		isFilepath:   false,
		expectedPath: "http://example.com",
	},
}

func Test_EnvReplacement(t *testing.T) {
	for _, test := range testEnvReplacement {
		actualPath, err := ResolveEnvironmentReplacement(test.path, test.envs, test.isFilepath)
		testutil.CheckErrorAndDeepEqual(t, false, err, test.expectedPath, actualPath)

	}
}

var buildContextPath = "../../integration/"

var destinationFilepathTests = []struct {
	src              string
	dest             string
	cwd              string
	expectedFilepath string
}{
	{
		src:              "context/foo",
		dest:             "/foo",
		cwd:              "/",
		expectedFilepath: "/foo",
	},
	{
		src:              "context/foo",
		dest:             "/foodir/",
		cwd:              "/",
		expectedFilepath: "/foodir/foo",
	},
	{
		src:              "context/foo",
		cwd:              "/",
		dest:             "foo",
		expectedFilepath: "/foo",
	},
	{
		src:              "context/bar/",
		cwd:              "/",
		dest:             "pkg/",
		expectedFilepath: "/pkg/bar",
	},
	{
		src:              "context/bar/",
		cwd:              "/newdir",
		dest:             "pkg/",
		expectedFilepath: "/newdir/pkg/bar",
	},
	{
		src:              "./context/empty",
		cwd:              "/",
		dest:             "/empty",
		expectedFilepath: "/empty",
	},
	{
		src:              "./context/empty",
		cwd:              "/dir",
		dest:             "/empty",
		expectedFilepath: "/empty",
	},
	{
		src:              "./",
		cwd:              "/",
		dest:             "/dir",
		expectedFilepath: "/dir",
	},
	{
		src:              "context/foo",
		cwd:              "/test",
		dest:             ".",
		expectedFilepath: "/test/foo",
	},
}

func Test_DestinationFilepath(t *testing.T) {
	for _, test := range destinationFilepathTests {
		actualFilepath, err := DestinationFilepath(test.src, test.dest, test.cwd)
		testutil.CheckErrorAndDeepEqual(t, false, err, test.expectedFilepath, actualFilepath)
	}
}

var urlDestFilepathTests = []struct {
	url          string
	cwd          string
	dest         string
	expectedDest string
	envs         []string
}{
	{
		url:          "https://something/something",
		cwd:          "/test",
		dest:         ".",
		expectedDest: "/test/something",
	},
	{
		url:          "https://something/something",
		cwd:          "/cwd",
		dest:         "/test",
		expectedDest: "/test",
	},
	{
		url:          "https://something/something",
		cwd:          "/test",
		dest:         "/dest/",
		expectedDest: "/dest/something",
	},
	{
		url:          "https://something/$foo.tar.gz",
		cwd:          "/test",
		dest:         "/foo/",
		expectedDest: "/foo/bar.tar.gz",
		envs:         []string{"foo=bar"},
	},
}

func Test_UrlDestFilepath(t *testing.T) {
	for _, test := range urlDestFilepathTests {
		actualDest, err := URLDestinationFilepath(test.url, test.dest, test.cwd, test.envs)
		testutil.CheckErrorAndDeepEqual(t, false, err, test.expectedDest, actualDest)
	}
}

var matchSourcesTests = []struct {
	srcs          []string
	files         []string
	expectedFiles []string
}{
	{
		srcs: []string{
			"pkg/*",
			"/root/dir?",
			testURL,
		},
		files: []string{
			"pkg/a",
			"pkg/b",
			"/pkg/d",
			"pkg/b/d/",
			"dir/",
			"root/dir1",
		},
		expectedFiles: []string{
			"/root/dir1",
			"pkg/a",
			"pkg/b",
			testURL,
		},
	},
}

func Test_MatchSources(t *testing.T) {
	for _, test := range matchSourcesTests {
		actualFiles, err := matchSources(test.srcs, test.files)
		sort.Strings(actualFiles)
		sort.Strings(test.expectedFiles)
		testutil.CheckErrorAndDeepEqual(t, false, err, test.expectedFiles, actualFiles)
	}
}

var isSrcValidTests = []struct {
	name            string
	srcsAndDest     []string
	resolvedSources []string
	shouldErr       bool
}{
	{
		name: "dest isn't directory",
		srcsAndDest: []string{
			"context/foo",
			"context/bar",
			"dest",
		},
		resolvedSources: []string{
			"context/foo",
			"context/bar",
		},
		shouldErr: true,
	},
	{
		name: "dest is directory",
		srcsAndDest: []string{
			"context/foo",
			"context/bar",
			"dest/",
		},
		resolvedSources: []string{
			"context/foo",
			"context/bar",
		},
		shouldErr: false,
	},
	{
		name: "copy file to file",
		srcsAndDest: []string{
			"context/bar/bam",
			"dest",
		},
		resolvedSources: []string{
			"context/bar/bam",
		},
		shouldErr: false,
	},
	{
		name: "copy files with wildcards to dir",
		srcsAndDest: []string{
			"context/foo",
			"context/b*",
			"dest/",
		},
		resolvedSources: []string{
			"context/foo",
			"context/bar",
		},
		shouldErr: false,
	},
	{
		name: "copy multilple files with wildcards to file",
		srcsAndDest: []string{
			"context/foo",
			"context/b*",
			"dest",
		},
		resolvedSources: []string{
			"context/foo",
			"context/bar",
		},
		shouldErr: true,
	},
	{
		name: "copy two files to file, one of which doesn't exist",
		srcsAndDest: []string{
			"context/foo",
			"context/doesntexist*",
			"dest",
		},
		resolvedSources: []string{
			"context/foo",
		},
		shouldErr: false,
	},
	{
		name: "copy dir to dest not specified as dir",
		srcsAndDest: []string{
			"context/",
			"dest",
		},
		resolvedSources: []string{
			"context/",
		},
		shouldErr: false,
	},
	{
		name: "copy url to file",
		srcsAndDest: []string{
			testURL,
			"dest",
		},
		resolvedSources: []string{
			testURL,
		},
		shouldErr: false,
	},
	{
		name: "copy two srcs, one excluded, to file",
		srcsAndDest: []string{
			"ignore/foo",
			"ignore/bar",
			"dest",
		},
		resolvedSources: []string{
			"ignore/foo",
			"ignore/bar",
		},
		shouldErr: false,
	},
	{
		name: "copy two srcs, both excluded, to file",
		srcsAndDest: []string{
			"ignore/baz",
			"ignore/bar",
			"dest",
		},
		resolvedSources: []string{
			"ignore/baz",
			"ignore/bar",
		},
		shouldErr: true,
	},
}

func Test_IsSrcsValid(t *testing.T) {
	for _, test := range isSrcValidTests {
		t.Run(test.name, func(t *testing.T) {
			if err := GetExcludedFiles("", buildContextPath); err != nil {
				t.Fatalf("error getting excluded files: %v", err)
			}
			err := IsSrcsValid(test.srcsAndDest, test.resolvedSources, buildContextPath)
			testutil.CheckError(t, test.shouldErr, err)
		})
	}
}

var testResolveSources = []struct {
	srcsAndDest  []string
	expectedList []string
}{
	{
		srcsAndDest: []string{
			"context/foo",
			"context/b*",
			testURL,
		},
		expectedList: []string{
			"context/foo",
			"context/bar",
			testURL,
		},
	},
}

func Test_ResolveSources(t *testing.T) {
	for _, test := range testResolveSources {
		actualList, err := ResolveSources(test.srcsAndDest, buildContextPath)
		testutil.CheckErrorAndDeepEqual(t, false, err, test.expectedList, actualList)
	}
}

var testRemoteUrls = []struct {
	name  string
	url   string
	valid bool
}{
	{
		name:  "Valid URL",
		url:   "https://google.com",
		valid: true,
	},
	{
		name:  "Invalid URL",
		url:   "not/real/",
		valid: false,
	},
	{
		name:  "URL which fails on GET",
		url:   "https://thereisnowaythiswilleverbearealurlrightrightrightcatsarethebest.com/something/not/real",
		valid: false,
	},
}

func Test_RemoteUrls(t *testing.T) {
	for _, test := range testRemoteUrls {
		t.Run(test.name, func(t *testing.T) {
			valid := IsSrcRemoteFileURL(test.url)
			testutil.CheckErrorAndDeepEqual(t, false, nil, test.valid, valid)
		})
	}

}

func TestResolveEnvironmentReplacementList(t *testing.T) {
	type args struct {
		values     []string
		envs       []string
		isFilepath bool
	}
	tests := []struct {
		name    string
		args    args
		want    []string
		wantErr bool
	}{
		{
			name: "url",
			args: args{
				values: []string{
					"https://google.com/$foo", "$bar", "$url",
				},
				envs: []string{
					"foo=baz",
					"bar=bat",
					"url=https://google.com",
				},
			},
			want: []string{"https://google.com/baz", "bat", "https://google.com"},
		},
		{
			name: "mixed",
			args: args{
				values: []string{
					"$foo", "$bar$baz", "baz",
				},
				envs: []string{
					"foo=FOO",
					"bar=BAR",
					"baz=BAZ",
				},
			},
			want: []string{"FOO", "BARBAZ", "baz"},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := ResolveEnvironmentReplacementList(tt.args.values, tt.args.envs, tt.args.isFilepath)
			if (err != nil) != tt.wantErr {
				t.Errorf("ResolveEnvironmentReplacementList() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("ResolveEnvironmentReplacementList() = %v, want %v", got, tt.want)
			}
		})
	}
}
