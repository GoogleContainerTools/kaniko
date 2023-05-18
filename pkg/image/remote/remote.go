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
	manifestCache = make(map[string]v1.Image)
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

	if ref.Context().RegistryStr() == name.DefaultRegistry {
		ref, err := normalizeReference(ref, image)
		if err != nil {
			return nil, err
		}

		for _, registryMirror := range opts.RegistryMirrors {
			var newReg name.Registry
			if opts.InsecurePull || opts.InsecureRegistries.Contains(registryMirror) {
				newReg, err = name.NewRegistry(registryMirror, name.WeakValidation, name.Insecure)
			} else {
				newReg, err = name.NewRegistry(registryMirror, name.StrictValidation)
			}
			if err != nil {
				return nil, err
			}
			ref := setNewRegistry(ref, newReg)

			logrus.Infof("Retrieving image %s from registry mirror %s", ref, registryMirror)
			remoteImage, err := remote.Image(ref, remoteOptions(registryMirror, opts, customPlatform)...)
			if err != nil {
				logrus.Warnf("Failed to retrieve image %s from registry mirror %s: %s. Will try with the next mirror, or fallback to the default registry.", ref, registryMirror, err)
				continue
			}

			manifestCache[image] = remoteImage

			return remoteImage, nil
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

	remoteImage, err := remote.Image(ref, remoteOptions(registryName, opts, customPlatform)...)

	if remoteImage != nil {
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
