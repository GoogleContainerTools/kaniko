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
	"github.com/GoogleCloudPlatform/k8s-container-builder/testutil"
	"testing"
)

var testEnvReplacement = []struct {
	path         string
	command      string
	envs         []string
	expectedPath string
}{
	{
		path:    "/simple/path",
		command: "WORKDIR /simple/path",
		envs: []string{
			"simple=/path/",
		},
		expectedPath: "/simple/path",
	},
	{
		path:    "${a}/b",
		command: "WORKDIR ${a}/b",
		envs: []string{
			"a=/path/",
			"b=/path2/",
		},
		expectedPath: "/path/b",
	},
	{
		path:    "/$a/b",
		command: "COPY ${a}/b /c/",
		envs: []string{
			"a=/path/",
			"b=/path2/",
		},
		expectedPath: "/path/b",
	},
	{
		path:    "\\$foo",
		command: "COPY \\$foo /quux",
		envs: []string{
			"foo=/path/",
		},
		expectedPath: "$foo",
	},
	{
		path:    "8080/$protocol",
		command: "EXPOSE 8080/$protocol",
		envs: []string{
			"protocol=udp",
		},
		expectedPath: "8080/udp",
	},
}

func Test_EnvReplacement(t *testing.T) {
	for _, test := range testEnvReplacement {
		actualPath, err := ResolveEnvironmentReplacement(test.command, test.path, test.envs)
		testutil.CheckErrorAndDeepEqual(t, false, err, test.expectedPath, actualPath)
	}
}
