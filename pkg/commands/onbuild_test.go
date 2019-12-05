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

package commands

import (
	"testing"

	"github.com/GoogleContainerTools/kaniko/pkg/dockerfile"
	v1 "github.com/google/go-containerregistry/pkg/v1"

	"github.com/GoogleContainerTools/kaniko/testutil"
	"github.com/moby/buildkit/frontend/dockerfile/instructions"
)

var onbuildTests = []struct {
	expression    string
	onbuildArray  []string
	expectedArray []string
}{
	{
		expression:   "RUN echo \\\"hi\\\" > $dir",
		onbuildArray: nil,
		expectedArray: []string{
			"RUN echo \\\"hi\\\" > $dir",
		},
	},
	{
		expression: "COPY foo foo",
		onbuildArray: []string{
			"RUN echo \"hi\" > /some/dir",
		},
		expectedArray: []string{
			"RUN echo \"hi\" > /some/dir",
			"COPY foo foo",
		},
	},
}

func TestExecuteOnbuild(t *testing.T) {
	for _, test := range onbuildTests {
		cfg := &v1.Config{
			Env: []string{
				"dir=/some/dir",
			},
			OnBuild: test.onbuildArray,
		}

		onbuildCmd := &OnBuildCommand{
			cmd: &instructions.OnbuildCommand{
				Expression: test.expression,
			},
		}
		buildArgs := dockerfile.NewBuildArgs([]string{})
		err := onbuildCmd.ExecuteCommand(cfg, buildArgs)
		testutil.CheckErrorAndDeepEqual(t, false, err, test.expectedArray, cfg.OnBuild)
	}

}
