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

package image

import (
	"fmt"
	"os"

	"github.com/GoogleContainerTools/kaniko/pkg/version"
	"github.com/containers/image/types"

	img "github.com/GoogleContainerTools/container-diff/pkg/image"
	"github.com/GoogleContainerTools/kaniko/pkg/constants"
	"github.com/containers/image/copy"
	"github.com/containers/image/docker"
	"github.com/containers/image/signature"
	"github.com/containers/image/transports/alltransports"
	"github.com/sirupsen/logrus"
)

// sourceImage is the image that will be modified by the executor

// NewSourceImage initializes the source image with the base image
func NewSourceImage(srcImg string) (*img.MutableSource, error) {
	if srcImg == constants.NoBaseImage {
		return img.NewMutableSource(nil)
	}
	logrus.Infof("Initializing source image %s", srcImg)
	ref, err := docker.ParseReference("//" + srcImg)
	if err != nil {
		return nil, err
	}
	return img.NewMutableSource(ref)
}

// PushImage pushes the final image
func PushImage(ms *img.MutableSource, destImg string, dockerInsecureSkipTLSVerify bool) error {
	srcRef := &img.ProxyReference{
		ImageReference: nil,
		Src:            ms,
	}
	destRef, err := alltransports.ParseImageName("docker://" + destImg)
	if err != nil {
		return err
	}
	policyContext, err := getPolicyContext()
	if err != nil {
		return err
	}
	logrus.Infof("Pushing image to %s", destImg)

	opts := &copy.Options{
		DestinationCtx: &types.SystemContext{
			DockerRegistryUserAgent: fmt.Sprintf("kaniko/executor-%s", version.Version()),
			DockerInsecureSkipTLSVerify: dockerInsecureSkipTLSVerify,
		},
	}
	return copy.Image(policyContext, destRef, srcRef, opts)
}

// SetEnvVariables sets environment variables as specified in the image
func SetEnvVariables(ms *img.MutableSource) error {
	envVars := ms.Env()
	for key, val := range envVars {
		if err := os.Setenv(key, val); err != nil {
			return err
		}
		logrus.Debugf("Setting environment variable %s=%s", key, val)
	}
	return nil
}

func getPolicyContext() (*signature.PolicyContext, error) {
	policyContext, err := signature.NewPolicyContext(&signature.Policy{
		Default: signature.PolicyRequirements{signature.NewPRInsecureAcceptAnything()},
	})
	if err != nil {
		logrus.Debugf("Error retrieving policy context: %s", err)
		return nil, err
	}
	return policyContext, nil
}
