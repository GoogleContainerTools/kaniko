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

package executor

import (
	"crypto/tls"
	"fmt"
	"net/http"
	"time"

	"github.com/GoogleContainerTools/kaniko/pkg/cache"
	"github.com/GoogleContainerTools/kaniko/pkg/config"
	"github.com/GoogleContainerTools/kaniko/pkg/constants"
	"github.com/GoogleContainerTools/kaniko/pkg/version"
	"github.com/google/go-containerregistry/pkg/authn"
	"github.com/google/go-containerregistry/pkg/authn/k8schain"
	"github.com/google/go-containerregistry/pkg/name"
	"github.com/google/go-containerregistry/pkg/v1"
	"github.com/google/go-containerregistry/pkg/v1/empty"
	"github.com/google/go-containerregistry/pkg/v1/mutate"
	"github.com/google/go-containerregistry/pkg/v1/remote"
	"github.com/google/go-containerregistry/pkg/v1/tarball"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
)

type withUserAgent struct {
	t http.RoundTripper
}

func (w *withUserAgent) RoundTrip(r *http.Request) (*http.Response, error) {
	r.Header.Set("User-Agent", fmt.Sprintf("kaniko/%s", version.Version()))
	return w.t.RoundTrip(r)
}

// DoPush is responsible for pushing image to the destinations specified in opts
func DoPush(image v1.Image, opts *config.KanikoOptions) error {
	if opts.NoPush {
		logrus.Info("Skipping push to container registry due to --no-push flag")
		return nil
	}
	destRefs := []name.Tag{}
	for _, destination := range opts.Destinations {
		destRef, err := name.NewTag(destination, name.WeakValidation)
		if err != nil {
			return errors.Wrap(err, "getting tag for destination")
		}
		destRefs = append(destRefs, destRef)
	}

	if opts.TarPath != "" {
		tagToImage := map[name.Tag]v1.Image{}
		for _, destRef := range destRefs {
			tagToImage[destRef] = image
		}
		return tarball.MultiWriteToFile(opts.TarPath, tagToImage)
	}

	// continue pushing unless an error occurs
	for _, destRef := range destRefs {
		if opts.Insecure {
			newReg, err := name.NewInsecureRegistry(destRef.Repository.Registry.Name(), name.WeakValidation)
			if err != nil {
				return errors.Wrap(err, "getting new insecure registry")
			}
			destRef.Repository.Registry = newReg
		}

		k8sc, err := k8schain.NewNoClient()
		if err != nil {
			return errors.Wrap(err, "getting k8schain client")
		}
		kc := authn.NewMultiKeychain(authn.DefaultKeychain, k8sc)
		pushAuth, err := kc.Resolve(destRef.Context().Registry)
		if err != nil {
			return errors.Wrap(err, "resolving pushAuth")
		}

		// Create a transport to set our user-agent.
		tr := http.DefaultTransport
		if opts.SkipTLSVerify {
			tr.(*http.Transport).TLSClientConfig = &tls.Config{
				InsecureSkipVerify: true,
			}
		}
		rt := &withUserAgent{t: tr}

		if err := remote.Write(destRef, image, pushAuth, rt); err != nil {
			return errors.Wrap(err, fmt.Sprintf("failed to push to destination %s", destRef))
		}
	}
	return nil
}

// pushLayerToCache pushes layer (tagged with cacheKey) to opts.Cache
// if opts.Cache doesn't exist, infer the cache from the given destination
func pushLayerToCache(opts *config.KanikoOptions, cacheKey string, tarPath string, createdBy string) error {
	layer, err := tarball.LayerFromFile(tarPath)
	if err != nil {
		return err
	}
	cache, err := cache.Destination(opts, cacheKey)
	if err != nil {
		return errors.Wrap(err, "getting cache destination")
	}
	logrus.Infof("Pushing layer %s to cache now", cache)
	empty := empty.Image
	empty, err = mutate.CreatedAt(empty, v1.Time{Time: time.Now()})
	if err != nil {
		return errors.Wrap(err, "setting empty image created time")
	}

	empty, err = mutate.Append(empty,
		mutate.Addendum{
			Layer: layer,
			History: v1.History{
				Author:    constants.Author,
				CreatedBy: createdBy,
			},
		},
	)
	if err != nil {
		return errors.Wrap(err, "appending layer onto empty image")
	}
	cacheOpts := *opts
	cacheOpts.Destinations = []string{cache}
	return DoPush(empty, &cacheOpts)
}
