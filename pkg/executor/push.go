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
	"io/ioutil"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/GoogleContainerTools/kaniko/pkg/cache"
	"github.com/GoogleContainerTools/kaniko/pkg/config"
	"github.com/GoogleContainerTools/kaniko/pkg/constants"
	"github.com/GoogleContainerTools/kaniko/pkg/creds"
	"github.com/GoogleContainerTools/kaniko/pkg/timing"
	"github.com/GoogleContainerTools/kaniko/pkg/version"
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

const (
	UpstreamClientUaKey = "UPSTREAM_CLIENT_TYPE"
)

func (w *withUserAgent) RoundTrip(r *http.Request) (*http.Response, error) {
	ua := []string{fmt.Sprintf("kaniko/%s", version.Version())}
	if upstream := os.Getenv(UpstreamClientUaKey); upstream != "" {
		ua = append(ua, upstream)
	}
	r.Header.Set("User-Agent", strings.Join(ua, ","))
	return w.t.RoundTrip(r)
}

// CheckPushPermissionos checks that the configured credentials can be used to
// push to every specified destination.
func CheckPushPermissions(opts *config.KanikoOptions) error {
	if opts.NoPush {
		return nil
	}

	checked := map[string]bool{}
	for _, destination := range opts.Destinations {
		destRef, err := name.NewTag(destination, name.WeakValidation)
		if err != nil {
			return errors.Wrap(err, "getting tag for destination")
		}
		if checked[destRef.Context().RepositoryStr()] {
			continue
		}

		registryName := destRef.Repository.Registry.Name()
		tr := makeTransport(opts, registryName)
		if err := remote.CheckPushPermission(destRef, creds.GetKeychain(), tr); err != nil {
			return errors.Wrapf(err, "checking push permission for %q", destRef)
		}
		checked[destRef.Context().RepositoryStr()] = true
	}
	return nil
}

// DoPush is responsible for pushing image to the destinations specified in opts
func DoPush(image v1.Image, opts *config.KanikoOptions) error {
	t := timing.Start("Total Push Time")

	if opts.DigestFile != "" {
		digest, err := image.Digest()
		if err != nil {
			return errors.Wrap(err, "error fetching digest")
		}
		digestByteArray := []byte(digest.String())
		err = ioutil.WriteFile(opts.DigestFile, digestByteArray, 0644)
		if err != nil {
			return errors.Wrap(err, "writing digest to file failed")
		}
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

	if opts.NoPush {
		logrus.Info("Skipping push to container registry due to --no-push flag")
		return nil
	}

	// continue pushing unless an error occurs
	for _, destRef := range destRefs {
		registryName := destRef.Repository.Registry.Name()
		if opts.Insecure || opts.InsecureRegistries.Contains(registryName) {
			newReg, err := name.NewRegistry(registryName, name.WeakValidation, name.Insecure)
			if err != nil {
				return errors.Wrap(err, "getting new insecure registry")
			}
			destRef.Repository.Registry = newReg
		}

		pushAuth, err := creds.GetKeychain().Resolve(destRef.Context().Registry)
		if err != nil {
			return errors.Wrap(err, "resolving pushAuth")
		}

		tr := makeTransport(opts, registryName)
		rt := &withUserAgent{t: tr}

		if err := remote.Write(destRef, image, remote.WithAuth(pushAuth), remote.WithTransport(rt)); err != nil {
			return errors.Wrap(err, fmt.Sprintf("failed to push to destination %s", destRef))
		}
	}
	timing.DefaultRun.Stop(t)
	return nil
}

func makeTransport(opts *config.KanikoOptions, registryName string) http.RoundTripper {
	// Create a transport to set our user-agent.
	tr := http.DefaultTransport
	if opts.SkipTLSVerify || opts.SkipTLSVerifyRegistries.Contains(registryName) {
		tr.(*http.Transport).TLSClientConfig = &tls.Config{
			InsecureSkipVerify: true,
		}
	}
	return tr
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
	cacheOpts.TarPath = ""   // tarPath doesn't make sense for Docker layers
	cacheOpts.NoPush = false // we want to push cached layers
	cacheOpts.Destinations = []string{cache}
	cacheOpts.InsecureRegistries = opts.InsecureRegistries
	cacheOpts.SkipTLSVerifyRegistries = opts.SkipTLSVerifyRegistries
	return DoPush(empty, &cacheOpts)
}
