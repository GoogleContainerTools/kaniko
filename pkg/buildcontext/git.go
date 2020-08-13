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

package buildcontext

import (
	"os"
	"strings"

	"github.com/GoogleContainerTools/kaniko/pkg/constants"
	"github.com/go-git/go-git/v5"
	"github.com/go-git/go-git/v5/plumbing"
	"github.com/go-git/go-git/v5/plumbing/transport"
	"github.com/go-git/go-git/v5/plumbing/transport/http"
)

const (
	gitPullMethodEnvKey = "GIT_PULL_METHOD"
	gitPullMethodHTTPS  = "https"
	gitPullMethodHTTP   = "http"

	gitAuthUsernameEnvKey = "GIT_USERNAME"
	gitAuthPasswordEnvKey = "GIT_PASSWORD"
	gitAuthTokenEnvKey    = "GIT_TOKEN"
)

var (
	supportedGitPullMethods = map[string]bool{gitPullMethodHTTPS: true, gitPullMethodHTTP: true}
)

// Git unifies calls to download and unpack the build context.
type Git struct {
	context string
}

// UnpackTarFromBuildContext will provide the directory where Git Repository is Cloned
func (g *Git) UnpackTarFromBuildContext() (string, error) {
	directory := constants.BuildContextDir
	parts := strings.Split(g.context, "#")
	options := git.CloneOptions{
		URL:               getGitPullMethod() + "://" + parts[0],
		Auth:              getGitAuth(),
		Progress:          os.Stdout,
		RecurseSubmodules: git.DefaultSubmoduleRecursionDepth,
	}
	if len(parts) > 1 {
		options.ReferenceName = plumbing.ReferenceName(parts[1])
	}
	_, err := git.PlainClone(directory, false, &options)
	return directory, err
}

func getGitAuth() transport.AuthMethod {
	username := os.Getenv(gitAuthUsernameEnvKey)
	password := os.Getenv(gitAuthPasswordEnvKey)
	token := os.Getenv(gitAuthTokenEnvKey)
	if token != "" {
		username = token
		password = ""
	}
	if username != "" || password != "" {
		return &http.BasicAuth{
			Username: username,
			Password: password,
		}
	}
	return nil
}

func getGitPullMethod() string {
	gitPullMethod := os.Getenv(gitPullMethodEnvKey)
	if ok := supportedGitPullMethods[gitPullMethod]; !ok {
		gitPullMethod = gitPullMethodHTTPS
	}
	return gitPullMethod
}
