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

	"github.com/GoogleContainerTools/kaniko/pkg/constants"
	"github.com/GoogleContainerTools/kaniko/pkg/util"
	"github.com/pkg/errors"
)

// Tar unifies calls to download and unpack the build context.
type Tar struct {
	context string
}

// UnpackTarFromBuildContext unpack the compressed tar file
func (t *Tar) UnpackTarFromBuildContext() (string, error) {
	directory := constants.BuildContextDir
	if err := os.MkdirAll(directory, 0750); err != nil {
		return "", errors.Wrap(err, "unpacking tar from build context")
	}

	return directory, util.UnpackCompressedTar(t.context, directory)
}
