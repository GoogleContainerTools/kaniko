/*
Copyright 2020 Google LLC

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

package buildcontext

import (
	"os"
	"testing"

	"github.com/GoogleContainerTools/kaniko/testutil"
)

func TestGetGitPullMethod(t *testing.T) {
	tests := []struct {
		testName string
		setEnv   func() (expectedValue string)
	}{
		{
			testName: "noEnv",
			setEnv: func() (expectedValue string) {
				expectedValue = gitPullMethodHTTPS
				return
			},
		},
		{
			testName: "emptyEnv",
			setEnv: func() (expectedValue string) {
				_ = os.Setenv(gitPullMethodEnvKey, "")
				expectedValue = gitPullMethodHTTPS
				return
			},
		},
		{
			testName: "httpEnv",
			setEnv: func() (expectedValue string) {
				err := os.Setenv(gitPullMethodEnvKey, gitPullMethodHTTP)
				if nil != err {
					expectedValue = gitPullMethodHTTPS
				} else {
					expectedValue = gitPullMethodHTTP
				}
				return
			},
		},
		{
			testName: "httpsEnv",
			setEnv: func() (expectedValue string) {
				_ = os.Setenv(gitPullMethodEnvKey, gitPullMethodHTTPS)
				expectedValue = gitPullMethodHTTPS
				return
			},
		},
		{
			testName: "unknownEnv",
			setEnv: func() (expectedValue string) {
				_ = os.Setenv(gitPullMethodEnvKey, "unknown")
				expectedValue = gitPullMethodHTTPS
				return
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.testName, func(t *testing.T) {
			expectedValue := tt.setEnv()
			testutil.CheckDeepEqual(t, expectedValue, getGitPullMethod())
		})
	}
}
