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
	"fmt"
	"os"
	"testing"

	"github.com/GoogleContainerTools/kaniko/pkg/config"
	"github.com/GoogleContainerTools/kaniko/testutil"
)

func TestGitClone(t *testing.T) {
	testGit := Git{
		context: "github.com/GoogleContainerTools/kaniko.git#refs/heads/master",
		options: &config.ContextOptions{
			InsecureGit: false,
		},
	}
	gopath := os.Getenv("GOPATH")
	gitPath := fmt.Sprintf("%s/tests/dir/git", gopath)
	os.Setenv("GITPATH", gitPath)
	_, err := testGit.UnpackTarFromBuildContext()
	testutil.CheckError(t, false, err)
	os.RemoveAll(gitPath)
}
