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
	img "github.com/GoogleCloudPlatform/container-diff/pkg/image"
	"github.com/GoogleCloudPlatform/k8s-container-builder/pkg/constants"
	"github.com/containers/image/copy"
	"github.com/containers/image/docker"
	"github.com/containers/image/signature"
	"github.com/containers/image/transports/alltransports"
	"github.com/sirupsen/logrus"
)

// sourceImage is the image that will be modified by the executor
var sourceImage img.MutableSource

// InitializeSourceImage initializes the source image with the base image
func InitializeSourceImage(srcImg string) error {
	ref, err := docker.ParseReference("//" + srcImg)
	if err != nil {
		return err
	}
	ms, err := img.NewMutableSource(ref)
	if err != nil {
		return err
	}
	sourceImage = *ms
	return nil
}

// AppendLayer appends a layer onto the base image
func AppendLayer(contents []byte, author string) error {
	return sourceImage.AppendLayer(contents, author)
}

// PushImage pushes the final image
func PushImage(destImg string) error {
	srcRef := &img.ProxyReference{
		ImageReference: nil,
		Src:            &sourceImage,
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
	return copy.Image(policyContext, destRef, srcRef, nil)
}

func getPolicyContext() (*signature.PolicyContext, error) {
	policy, err := signature.NewPolicyFromFile(constants.PolicyJSONPath)
	if err != nil {
		logrus.Debugf("Error retrieving policy: %s", err)
		return nil, err
	}
	policyContext, err := signature.NewPolicyContext(policy)
	if err != nil {
		logrus.Debugf("Error retrieving policy context: %s", err)
		return nil, err
	}
	return policyContext, nil
}
