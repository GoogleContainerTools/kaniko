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

package remote

import (
	"fmt"
	"strings"

	"github.com/GoogleContainerTools/kaniko/pkg/config"
	"github.com/GoogleContainerTools/kaniko/pkg/creds"
	"github.com/GoogleContainerTools/kaniko/pkg/util"

	"github.com/google/go-containerregistry/pkg/name"
	v1 "github.com/google/go-containerregistry/pkg/v1"
	"github.com/google/go-containerregistry/pkg/v1/remote"

	"github.com/sirupsen/logrus"
)

var (
	manifestCache   = make(map[string]v1.Image)
	remoteImageFunc = remote.Image
)

// RetrieveRemoteImage retrieves the manifest for the specified image from the specified registry
func RetrieveRemoteImage(image string, opts config.RegistryOptions, customPlatform string) (v1.Image, error) {
	logrus.Infof("Retrieving image manifest %s", image)

	cachedRemoteImage := manifestCache[image]
	if cachedRemoteImage != nil {
		logrus.Infof("Returning cached image manifest")
		return cachedRemoteImage, nil
	}

	ref, err := name.ParseReference(image, name.WeakValidation)
	if err != nil {
		return nil, err
	}

	if newRegURLs, found := opts.RegistryMaps[ref.Context().RegistryStr()]; found {
		ref, err := normalizeReference(ref, image)
		if err != nil {
			return nil, err
		}

		for _, regToMapTo := range newRegURLs {
			var newReg name.Registry
			if opts.InsecurePull || opts.InsecureRegistries.Contains(regToMapTo) {
				newReg, err = name.NewRegistry(regToMapTo, name.WeakValidation, name.Insecure)
			} else {
				newReg, err = name.NewRegistry(regToMapTo, name.StrictValidation)
			}
			if err != nil {
				return nil, err
			}
			ref := setNewRegistry(ref, newReg)
			logrus.Infof("Retrieving image %s from mapped registry %s", ref, regToMapTo)
			retryFunc := func() (v1.Image, error) {
				return remoteImageFunc(ref, remoteOptions(regToMapTo, opts, customPlatform)...)
			}

			var remoteImage v1.Image
			var err error
			if remoteImage, err = util.RetryWithResult(retryFunc, opts.ImageDownloadRetry, 1000); err != nil {
				logrus.Warnf("Failed to retrieve image %s from remapped registry %s: %s. Will try with the next registry, or fallback to the original registry.", ref, regToMapTo, err)
				continue
			}

			manifestCache[image] = remoteImage

			return remoteImage, nil
		}

		if len(newRegURLs) > 0 && opts.SkipDefaultRegistryFallback {
			return nil, fmt.Errorf("image not found on any configured mapped registries for %s", ref)
		}
	}

	registryName := ref.Context().RegistryStr()
	if opts.InsecurePull || opts.InsecureRegistries.Contains(registryName) {
		newReg, err := name.NewRegistry(registryName, name.WeakValidation, name.Insecure)
		if err != nil {
			return nil, err
		}
		ref = setNewRegistry(ref, newReg)
	}

	logrus.Infof("Retrieving image %s from registry %s", ref, registryName)

	retryFunc := func() (v1.Image, error) {
		return remoteImageFunc(ref, remoteOptions(registryName, opts, customPlatform)...)
	}

	var remoteImage v1.Image
	if remoteImage, err = util.RetryWithResult(retryFunc, opts.ImageDownloadRetry, 1000); remoteImage != nil {
		manifestCache[image] = remoteImage
	}

	return remoteImage, err
}

// normalizeReference adds the library/ prefix to images without it.
//
// It is mostly useful when using a registry mirror that is not able to perform
// this fix automatically.
func normalizeReference(ref name.Reference, image string) (name.Reference, error) {
	if !strings.ContainsRune(image, '/') {
		return name.ParseReference("library/"+image, name.WeakValidation)
	}

	return ref, nil
}

func setNewRegistry(ref name.Reference, newReg name.Registry) name.Reference {
	switch r := ref.(type) {
	case name.Tag:
		r.Repository.Registry = newReg
		return r
	case name.Digest:
		r.Repository.Registry = newReg
		return r
	default:
		return ref
	}
}

func remoteOptions(registryName string, opts config.RegistryOptions, customPlatform string) []remote.Option {
	tr, err := util.MakeTransport(opts, registryName)

	// The MakeTransport function will only return errors if there was a problem
	// with registry certificates (Verification or mTLS)
	if err != nil {
		logrus.Fatalf("Unable to setup transport for registry %q: %v", customPlatform, err)
	}

	// The platform value has previously been validated.
	platform, err := v1.ParsePlatform(customPlatform)
	if err != nil {
		logrus.Fatalf("Invalid platform %q: %v", customPlatform, err)
	}

	return []remote.Option{remote.WithTransport(tr), remote.WithAuthFromKeychain(creds.GetKeychain()), remote.WithPlatform(*platform)}
}
