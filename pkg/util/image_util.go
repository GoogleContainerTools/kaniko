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
	"crypto/tls"
	"fmt"
	"net/http"
	"path/filepath"
	"strconv"

	"github.com/GoogleContainerTools/kaniko/pkg/timing"

	"github.com/GoogleContainerTools/kaniko/pkg/creds"

	"github.com/google/go-containerregistry/pkg/name"
	v1 "github.com/google/go-containerregistry/pkg/v1"
	"github.com/google/go-containerregistry/pkg/v1/empty"
	"github.com/google/go-containerregistry/pkg/v1/remote"
	"github.com/google/go-containerregistry/pkg/v1/tarball"
	"github.com/sirupsen/logrus"

	"github.com/GoogleContainerTools/kaniko/pkg/cache"
	"github.com/GoogleContainerTools/kaniko/pkg/config"
	"github.com/GoogleContainerTools/kaniko/pkg/constants"
)

var (
	// RetrieveRemoteImage downloads an image from a remote location
	RetrieveRemoteImage = remoteImage
	retrieveTarImage    = tarballImage
)

// RetrieveSourceImage returns the base image of the stage at index
func RetrieveSourceImage(stage config.KanikoStage, opts *config.KanikoOptions) (v1.Image, error) {
	t := timing.Start("Retrieving Source Image")
	defer timing.DefaultRun.Stop(t)
	buildArgs := opts.BuildArgs
	var metaArgsString []string
	for _, arg := range stage.MetaArgs {
		metaArgsString = append(metaArgsString, fmt.Sprintf("%s=%s", arg.Key, arg.ValueString()))
	}
	buildArgs = append(buildArgs, metaArgsString...)
	currentBaseName, err := ResolveEnvironmentReplacement(stage.BaseName, buildArgs, false)
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
	if stage.BaseImageStoredLocally {
		return retrieveTarImage(stage.BaseImageIndex)
	}

	// Finally, check if local caching is enabled
	// If so, look in the local cache before trying the remote registry
	if opts.CacheDir != "" {
		cachedImage, err := cachedImage(opts, currentBaseName)
		if cachedImage != nil {
			return cachedImage, nil
		}

		if err != nil {
			logrus.Infof("Error while retrieving image from cache: %v", err)
		}
	}

	// Otherwise, initialize image as usual
	return RetrieveRemoteImage(currentBaseName, opts)
}

func tarballImage(index int) (v1.Image, error) {
	tarPath := filepath.Join(constants.KanikoIntermediateStagesDir, strconv.Itoa(index))
	logrus.Infof("Base image from previous stage %d found, using saved tar at path %s", index, tarPath)
	return tarball.ImageFromPath(tarPath, nil)
}

func remoteImage(image string, opts *config.KanikoOptions) (v1.Image, error) {
	logrus.Infof("Downloading base image %s", image)
	ref, err := name.ParseReference(image, name.WeakValidation)
	if err != nil {
		return nil, err
	}

	registryName := ref.Context().RegistryStr()
	if opts.InsecurePull || opts.InsecureRegistries.Contains(registryName) {
		newReg, err := name.NewRegistry(registryName, name.WeakValidation, name.Insecure)
		if err != nil {
			return nil, err
		}
		if tag, ok := ref.(name.Tag); ok {
			tag.Repository.Registry = newReg
			ref = tag
		}
		if digest, ok := ref.(name.Digest); ok {
			digest.Repository.Registry = newReg
			ref = digest
		}
	}

	tr := http.DefaultTransport.(*http.Transport)
	if opts.SkipTLSVerifyPull || opts.SkipTLSVerifyRegistries.Contains(registryName) {
		tr.TLSClientConfig = &tls.Config{
			InsecureSkipVerify: true,
		}
	}

	return remote.Image(ref, remote.WithTransport(tr), remote.WithAuthFromKeychain(creds.GetKeychain()))
}

func cachedImage(opts *config.KanikoOptions, image string) (v1.Image, error) {
	ref, err := name.ParseReference(image, name.WeakValidation)
	if err != nil {
		return nil, err
	}

	var cacheKey string
	if d, ok := ref.(name.Digest); ok {
		cacheKey = d.DigestStr()
	} else {
		img, err := remoteImage(image, opts)
		if err != nil {
			return nil, err
		}

		d, err := img.Digest()
		if err != nil {
			return nil, err
		}

		cacheKey = d.String()
	}

	return cache.LocalSource(&opts.CacheOptions, cacheKey)
}
