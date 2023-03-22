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
	"io"
	"net/http"
	"os"
	"path/filepath"

	kConfig "github.com/GoogleContainerTools/kaniko/pkg/config"
	"github.com/GoogleContainerTools/kaniko/pkg/constants"
	"github.com/GoogleContainerTools/kaniko/pkg/util"
	"github.com/sirupsen/logrus"
)

// HTTPSTar struct for https tar.gz files processing
type HTTPSTar struct {
	context string
}

// UnpackTarFromBuildContext downloads context file from https server
func (h *HTTPSTar) UnpackTarFromBuildContext() (directory string, err error) {

	logrus.Info("Retrieving https tar file")

	// Create directory and target file for downloading the context file
	directory = kConfig.BuildContextDir
	tarPath := filepath.Join(directory, constants.ContextTar)
	file, err := util.CreateTargetTarfile(tarPath)
	if err != nil {
		return
	}

	// Download tar file from remote https server
	// and save it into the target tar file
	resp, err := http.Get(h.context) //nolint:noctx
	if err != nil {
		return
	}
	defer func() {
		if closeErr := resp.Body.Close(); err == nil && closeErr != nil {
			err = closeErr
		}
	}()

	if resp.StatusCode != http.StatusOK {
		return directory, fmt.Errorf("HTTPSTar bad status from server: %s", resp.Status)
	}

	if _, err = io.Copy(file, resp.Body); err != nil {
		return tarPath, err
	}

	logrus.Info("Retrieved https tar file")

	if err = util.UnpackCompressedTar(tarPath, directory); err != nil {
		return
	}

	logrus.Info("Extracted https tar file")

	// Remove the tar so it doesn't interfere with subsequent commands
	return directory, os.Remove(tarPath)
}
