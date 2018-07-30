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

package util

import (
	"path/filepath"
	"strconv"

	"github.com/docker/docker/builder/dockerfile/instructions"
	"github.com/google/go-containerregistry/pkg/authn"
	"github.com/google/go-containerregistry/pkg/authn/k8schain"
	"github.com/google/go-containerregistry/pkg/name"
	"github.com/google/go-containerregistry/pkg/v1"
	"github.com/google/go-containerregistry/pkg/v1/empty"
	"github.com/google/go-containerregistry/pkg/v1/remote"
	"github.com/google/go-containerregistry/pkg/v1/tarball"
	"github.com/sirupsen/logrus"

	"github.com/GoogleContainerTools/kaniko/pkg/constants"
)

var (
	// For testing
	retrieveRemoteImage = remoteImage
	retrieveTarImage    = tarballImage
)

// RetrieveSourceImage returns the base image of the stage at index
func RetrieveSourceImage(index int, buildArgs []string, stages []instructions.Stage) (v1.Image, error) {
	currentStage := stages[index]
	currentBaseName, err := ResolveEnvironmentReplacement(currentStage.BaseName, buildArgs, false)
	if err != nil {
		return nil, err
	}
	// First, check if the base image is a scratch image
	if currentBaseName == constants.NoBaseImage {
		logrus.Info("No base image, nothing to extract")
		return empty.Image, nil
	}
	// Next, check if the base image of the current stage is built from a previous stage
	// If so, retrieve the image from the stored tarball
	for i, stage := range stages {
		if i > index {
			continue
		}
		if stage.Name == currentBaseName {
			return retrieveTarImage(i)
		}
	}
	// Otherwise, initialize image as usual
	return retrieveRemoteImage(currentBaseName)
}

func tarballImage(index int) (v1.Image, error) {
	tarPath := filepath.Join(constants.KanikoIntermediateStagesDir, strconv.Itoa(index))
	logrus.Infof("Base image from previous stage %d found, using saved tar at path %s", index, tarPath)
	return tarball.ImageFromPath(tarPath, nil)
}

func remoteImage(image string) (v1.Image, error) {
	logrus.Infof("Downloading base image %s", image)
	ref, err := name.ParseReference(image, name.WeakValidation)
	if err != nil {
		return nil, err
	}
	k8sc, err := k8schain.NewNoClient()
	if err != nil {
		return nil, err
	}
	kc := authn.NewMultiKeychain(authn.DefaultKeychain, k8sc)
	return remote.Image(ref, remote.WithAuthFromKeychain(kc))

}
