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
	"errors"
	"fmt"
	"os"
	"strings"

	kConfig "github.com/GoogleContainerTools/kaniko/pkg/config"
	"github.com/go-git/go-billy/v5/osfs"
	"github.com/go-git/go-git/v5"
	"github.com/go-git/go-git/v5/config"
	"github.com/go-git/go-git/v5/plumbing"
	"github.com/go-git/go-git/v5/plumbing/cache"
	"github.com/go-git/go-git/v5/plumbing/transport"
	"github.com/go-git/go-git/v5/plumbing/transport/http"
	"github.com/go-git/go-git/v5/storage/filesystem"
	"github.com/sirupsen/logrus"
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
	opts    BuildOptions
}

// UnpackTarFromBuildContext will provide the directory where Git Repository is Cloned
func (g *Git) UnpackTarFromBuildContext() (string, error) {
	directory := kConfig.BuildContextDir
	parts := strings.Split(g.context, "#")
	url := getGitPullMethod() + "://" + parts[0]
	options := git.CloneOptions{
		URL:               url,
		Auth:              getGitAuth(),
		Progress:          os.Stdout,
		SingleBranch:      g.opts.GitSingleBranch,
		RecurseSubmodules: getRecurseSubmodules(g.opts.GitRecurseSubmodules),
		InsecureSkipTLS:   g.opts.InsecureSkipTLS,
	}
	var fetchRef string
	var checkoutRef string
	if len(parts) > 1 {
		if plumbing.IsHash(parts[1]) || !strings.HasPrefix(parts[1], "refs/pull/") {
			// Handle any non-branch refs separately. First, clone the repo HEAD, and
			// then fetch and check out the fetchRef.
			fetchRef = parts[1]
			if plumbing.IsHash(parts[1]) {
				checkoutRef = fetchRef
			} else {
				// The ReferenceName still needs to be present in the options passed
				// to the clone operation for non-hash references of private repositories.
				options.ReferenceName = plumbing.ReferenceName(fetchRef)
			}
		} else {
			// Branches will be cloned directly.
			options.ReferenceName = plumbing.ReferenceName(parts[1])
		}
	}

	if branch := g.opts.GitBranch; branch != "" {
		ref, err := getGitReferenceName(directory, url, branch)
		if err != nil {
			return directory, err
		}
		options.ReferenceName = ref
	}

	logrus.Debugf("Getting source from reference %s", options.ReferenceName)
	r, err := git.PlainClone(directory, false, &options)
	if err != nil {
		return directory, err
	}

	if fetchRef != "" {
		err = r.Fetch(&git.FetchOptions{
			RemoteName: "origin",
			Auth:       getGitAuth(),
			RefSpecs:   []config.RefSpec{config.RefSpec(fetchRef + ":" + fetchRef)},
		})
		if err != nil && !errors.Is(err, git.NoErrAlreadyUpToDate) {
			return directory, err
		}
	}

	if len(parts) > 2 {
		checkoutRef = parts[2]
	}
	if checkoutRef != "" {
		// ... retrieving the commit being pointed by HEAD
		_, err := r.Head()
		if err != nil {
			return directory, err
		}

		w, err := r.Worktree()
		if err != nil {
			return directory, err
		}

		// ... checking out to desired commit
		err = w.Checkout(&git.CheckoutOptions{
			Hash: plumbing.NewHash(checkoutRef),
		})
		if err != nil {
			return directory, err
		}
	}
	return directory, nil
}

func getGitReferenceName(directory string, url string, branch string) (plumbing.ReferenceName, error) {
	var remote = git.NewRemote(
		filesystem.NewStorage(
			osfs.New(directory),
			cache.NewObjectLRUDefault(),
		),
		&config.RemoteConfig{
			URLs: []string{url},
		},
	)

	refs, err := remote.List(&git.ListOptions{
		Auth: getGitAuth(),
	})
	if err != nil {
		return plumbing.HEAD, err
	}

	if ref := plumbing.NewBranchReferenceName(branch); gitRefExists(ref, refs) {
		return ref, nil
	}

	if ref := plumbing.NewTagReferenceName(branch); gitRefExists(ref, refs) {
		return ref, nil
	}

	return plumbing.HEAD, fmt.Errorf("invalid branch: %s", branch)
}

func gitRefExists(ref plumbing.ReferenceName, refs []*plumbing.Reference) bool {
	for _, ref2 := range refs {
		if ref.String() == ref2.Name().String() {
			return true
		}
	}
	return false
}

func getRecurseSubmodules(v bool) git.SubmoduleRescursivity {
	if v {
		return git.DefaultSubmoduleRecursionDepth
	}
	return git.NoRecurseSubmodules
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
