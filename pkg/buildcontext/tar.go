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
	"io/ioutil"
	"os"
	"path/filepath"

	"github.com/GoogleContainerTools/kaniko/pkg/constants"
	"github.com/GoogleContainerTools/kaniko/pkg/util"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
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
	if t.context == "stdin" {
		fi, _ := os.Stdin.Stat()
		if (fi.Mode() & os.ModeCharDevice) != 0 {
			return "", fmt.Errorf("no data found.. don't forget to add the '--interactive, -i' flag")
		}
		logrus.Infof("To simulate EOF and exit, press 'Ctrl+D'")
		// if launched through docker in interactive mode and without piped data
		// process will be stuck here until EOF is sent
		data, err := util.GetInputFrom(os.Stdin)
		if err != nil {
			return "", errors.Wrap(err, "fail to get standard input")
		}
		t.context = filepath.Join(directory, constants.ContextTar)
		if err := ioutil.WriteFile(t.context, data, 0644); err != nil {
			return "", errors.Wrap(err, "fail to redirect standard input into compressed tar file")
		}
	}

	return directory, util.UnpackCompressedTar(t.context, directory)
}
