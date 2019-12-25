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

	"github.com/GoogleContainerTools/kaniko/pkg/config"

	"github.com/GoogleContainerTools/kaniko/pkg/constants"
	git "gopkg.in/src-d/go-git.v4"
	"gopkg.in/src-d/go-git.v4/plumbing"
)

// Git unifies calls to download and unpack the build context.
type Git struct {
	context string
	options *config.ContextOptions
}

// UnpackTarFromBuildContext will provide the directory where Git Repository is Cloned
func (g *Git) UnpackTarFromBuildContext() (string, error) {
	directory := os.Getenv("GITPATH")
	if directory == "" {
		directory = constants.BuildContextDir
	}
	parts := strings.Split(g.context, "#")
	url := parts[0]
	if g.options.InsecureGit {
		url = "http://" + url
	} else {
		url = "https://" + url
	}
	options := git.CloneOptions{
		URL:      url,
		Progress: os.Stdout,
	}
	if len(parts) > 1 {
		options.ReferenceName = plumbing.ReferenceName(parts[1])
	}
	_, err := git.PlainClone(directory, false, &options)
	return directory, err
}
