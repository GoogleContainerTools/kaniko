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
	"testing"

	"github.com/GoogleContainerTools/kaniko/testutil"
	"github.com/moby/buildkit/frontend/dockerfile/instructions"
)

func strPtr(source string) *string {
	return &source
}

func TestGetAllAllowed(t *testing.T) {
	buildArgs := newBuildArgsFromMap(map[string]*string{
		"ArgNotUsedInDockerfile":              strPtr("fromopt1"),
		"ArgOverriddenByOptions":              strPtr("fromopt2"),
		"ArgNoDefaultInDockerfileFromOptions": strPtr("fromopt3"),
		"HTTP_PROXY":                          strPtr("theproxy"),
		"all_proxy":                           strPtr("theproxy2"),
	})

	buildArgs.AddMetaArgs([]instructions.ArgCommand{
		{
			Args: []instructions.KeyValuePairOptional{
				{
					Key:   "ArgFromMeta",
					Value: strPtr("frommeta1"),
				},
				{
					Key:   "ArgOverriddenByOptions",
					Value: strPtr("frommeta2"),
				},
			},
		},
		{
			Args: []instructions.KeyValuePairOptional{
				{
					Key:   "ArgFromMetaNotUsed",
					Value: strPtr("frommeta3"),
				},
			},
		},
	})

	buildArgs.AddArg("ArgOverriddenByOptions", strPtr("fromdockerfile2"))
	buildArgs.AddArg("ArgWithDefaultInDockerfile", strPtr("fromdockerfile1"))
	buildArgs.AddArg("ArgNoDefaultInDockerfile", nil)
	buildArgs.AddArg("ArgNoDefaultInDockerfileFromOptions", nil)
	buildArgs.AddArg("ArgFromMeta", nil)
	buildArgs.AddArg("ArgFromMetaOverridden", strPtr("fromdockerfile3"))

	all := buildArgs.GetAllAllowed()
	expected := map[string]string{
		"HTTP_PROXY":                          "theproxy",
		"all_proxy":                           "theproxy2",
		"ArgOverriddenByOptions":              "fromopt2",
		"ArgWithDefaultInDockerfile":          "fromdockerfile1",
		"ArgNoDefaultInDockerfileFromOptions": "fromopt3",
		"ArgFromMeta":                         "frommeta1",
		"ArgFromMetaOverridden":               "fromdockerfile3",
	}
	testutil.CheckDeepEqual(t, expected, all)
}

func TestGetAllMeta(t *testing.T) {
	buildArgs := newBuildArgsFromMap(map[string]*string{
		"ArgNotUsedInDockerfile":        strPtr("fromopt1"),
		"ArgOverriddenByOptions":        strPtr("fromopt2"),
		"ArgNoDefaultInMetaFromOptions": strPtr("fromopt3"),
		"HTTP_PROXY":                    strPtr("theproxy"),
	})

	buildArgs.AddMetaArgs([]instructions.ArgCommand{
		{
			Args: []instructions.KeyValuePairOptional{
				{
					Key:   "ArgFromMeta",
					Value: strPtr("frommeta1"),
				},
				{
					Key:   "ArgOverriddenByOptions",
					Value: strPtr("frommeta2"),
				},
			},
		},
		{
			Args: []instructions.KeyValuePairOptional{
				{
					Key:   "ArgNoDefaultInMetaFromOptions",
					Value: nil,
				},
			},
		},
	})

	all := buildArgs.GetAllMeta()
	expected := map[string]string{
		"HTTP_PROXY":                    "theproxy",
		"ArgFromMeta":                   "frommeta1",
		"ArgOverriddenByOptions":        "fromopt2",
		"ArgNoDefaultInMetaFromOptions": "fromopt3",
	}
	testutil.CheckDeepEqual(t, expected, all)
}
