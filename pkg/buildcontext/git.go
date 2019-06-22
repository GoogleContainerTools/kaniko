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
	"strings"

	"github.com/GoogleContainerTools/kaniko/pkg/constants"
	git "gopkg.in/src-d/go-git.v4"
	"gopkg.in/src-d/go-git.v4/plumbing"
)

// Git unifies calls to download and unpack the build context.
type Git struct {
	context string
}

// UnpackTarFromBuildContext will provide the directory where Git Repository is Cloned
func (g *Git) UnpackTarFromBuildContext() (string, error) {
	directory := constants.BuildContextDir
	parts := strings.Split(g.context, "#")
	url := "https://" + parts[0]
	branch := "master"
	if len(parts) > 1 {
		branch = parts[1]
	}
	fmt.Println("will clone branch", branch)
	_, err := git.PlainClone(directory, false, &git.CloneOptions{
		URL:           url,
		Progress:      os.Stdout,
		ReferenceName: plumbing.ReferenceName(branch),
		SingleBranch:  true,
	})
	return directory, err
}
