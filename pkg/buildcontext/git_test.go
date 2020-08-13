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
	"github.com/go-git/go-git/v5/plumbing/transport"
	"github.com/go-git/go-git/v5/plumbing/transport/http"
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

func TestGetGitAuth(t *testing.T) {
	tests := []struct {
		testName string
		setEnv   func() (expectedValue transport.AuthMethod)
	}{
		{
			testName: "noEnv",
			setEnv: func() (expectedValue transport.AuthMethod) {
				expectedValue = nil
				return
			},
		},
		{
			testName: "emptyUsernameEnv",
			setEnv: func() (expectedValue transport.AuthMethod) {
				_ = os.Setenv(gitAuthUsernameEnvKey, "")
				expectedValue = nil
				return
			},
		},
		{
			testName: "emptyPasswordEnv",
			setEnv: func() (expectedValue transport.AuthMethod) {
				_ = os.Setenv(gitAuthPasswordEnvKey, "")
				expectedValue = nil
				return
			},
		},
		{
			testName: "emptyEnv",
			setEnv: func() (expectedValue transport.AuthMethod) {
				_ = os.Setenv(gitAuthUsernameEnvKey, "")
				_ = os.Setenv(gitAuthPasswordEnvKey, "")
				expectedValue = nil
				return
			},
		},
		{
			testName: "withUsername",
			setEnv: func() (expectedValue transport.AuthMethod) {
				username := "foo"
				_ = os.Setenv(gitAuthUsernameEnvKey, username)
				expectedValue = &http.BasicAuth{Username: username}
				return
			},
		},
		{
			testName: "withPassword",
			setEnv: func() (expectedValue transport.AuthMethod) {
				pass := "super-secret-password-1234"
				_ = os.Setenv(gitAuthPasswordEnvKey, pass)
				expectedValue = &http.BasicAuth{Password: pass}
				return
			},
		},
		{
			testName: "withUsernamePassword",
			setEnv: func() (expectedValue transport.AuthMethod) {
				username := "foo"
				pass := "super-secret-password-1234"
				_ = os.Setenv(gitAuthUsernameEnvKey, username)
				_ = os.Setenv(gitAuthPasswordEnvKey, pass)
				expectedValue = &http.BasicAuth{Username: username, Password: pass}
				return
			},
		},
		{
			testName: "withToken",
			setEnv: func() (expectedValue transport.AuthMethod) {
				token := "some-other-token"
				_ = os.Setenv(gitAuthTokenEnvKey, token)
				expectedValue = &http.BasicAuth{Username: token}
				return
			},
		},
		{
			testName: "withTokenUsernamePassword",
			setEnv: func() (expectedValue transport.AuthMethod) {
				username := "foo-user"
				token := "some-token-45678"
				pass := "some-password-12345"
				_ = os.Setenv(gitAuthUsernameEnvKey, username)
				_ = os.Setenv(gitAuthPasswordEnvKey, pass)
				_ = os.Setenv(gitAuthTokenEnvKey, token)
				expectedValue = &http.BasicAuth{Username: token}
				return
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.testName, func(t *testing.T) {
			// Make sure to unset environment vars to get a clean test each time
			defer clearTestAuthEnv()

			expectedValue := tt.setEnv()
			testutil.CheckDeepEqual(t, expectedValue, getGitAuth())
		})
	}

}

func clearTestAuthEnv() {
	_ = os.Unsetenv(gitAuthUsernameEnvKey)
	_ = os.Unsetenv(gitAuthPasswordEnvKey)
}
