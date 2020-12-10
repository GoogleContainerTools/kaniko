package buildcontext

import (
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"

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
	directory = constants.BuildContextDir
	tarPath := filepath.Join(directory, constants.ContextTar)
	file, err := util.CreateTargetTarfile(tarPath)
	if err != nil {
		return
	}

	// Download tar file from remote https server
	// and save it into the target tar file
	resp, err := http.Get(h.context)
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
