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
	"github.com/GoogleContainerTools/kaniko/pkg/constants"
	"github.com/GoogleContainerTools/kaniko/pkg/util"
	"github.com/pkg/errors"
	"os"
	"path/filepath"
)

// Git unifies calls to download and unpack the build context.
type GitHubArchive struct {
	context string
}

// UnpackTarFromBuildContext will provide the directory where Git Repository is Cloned
func (g *GitHubArchive) UnpackTarFromBuildContext() (string, error) {
	uri := "https://" + g.context
	_, err := downloadAndUnpackTar(uri, constants.BuildContextDir)
	return constants.BuildContextDir, err
}

func downloadAndUnpackTar(uri, directory string) ([]string, error) {
	tarPath := filepath.Join(directory, constants.ContextTar)
	if err := util.DownloadFileToDest(uri, tarPath, int64(os.Getuid()),  int64(os.Getgid())); err != nil {
		return nil, errors.Wrap(err, fmt.Sprintf("downloading tar from %s", uri))
	}
	return util.UnpackCompressedTar(tarPath, directory, true)
}
