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

package cache

import (
	"fmt"
	"os"
	"path"
	"time"

	"github.com/GoogleContainerTools/kaniko/pkg/config"
	"github.com/google/go-containerregistry/pkg/authn"
	"github.com/google/go-containerregistry/pkg/authn/k8schain"
	"github.com/google/go-containerregistry/pkg/name"
	"github.com/google/go-containerregistry/pkg/v1"
	"github.com/google/go-containerregistry/pkg/v1/remote"
	"github.com/google/go-containerregistry/pkg/v1/tarball"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
)

// LayerCache is the layer cache
type LayerCache interface {
	RetrieveLayer(string) (v1.Image, error)
}

// RegistryCache is the registry cache
type RegistryCache struct {
	Opts *config.KanikoOptions
}

// RetrieveLayer retrieves a layer from the cache given the cache key ck.
func (rc *RegistryCache) RetrieveLayer(ck string) (v1.Image, error) {
	cache, err := Destination(rc.Opts, ck)
	if err != nil {
		return nil, errors.Wrap(err, "getting cache destination")
	}
	logrus.Infof("Checking for cached layer %s...", cache)

	cacheRef, err := name.NewTag(cache, name.WeakValidation)
	if err != nil {
		return nil, errors.Wrap(err, fmt.Sprintf("getting reference for %s", cache))
	}
	k8sc, err := k8schain.NewNoClient()
	if err != nil {
		return nil, err
	}
	kc := authn.NewMultiKeychain(authn.DefaultKeychain, k8sc)
	img, err := remote.Image(cacheRef, remote.WithAuthFromKeychain(kc))
	if err != nil {
		return nil, err
	}

	cf, err := img.ConfigFile()
	if err != nil {
		return nil, errors.Wrap(err, fmt.Sprintf("retrieving config file for %s", cache))
	}

	expiry := cf.Created.Add(rc.Opts.CacheTTL)
	// Layer is stale, rebuild it.
	if expiry.Before(time.Now()) {
		logrus.Infof("Cache entry expired: %s", cache)
		return nil, errors.New(fmt.Sprintf("Cache entry expired: %s", cache))
	}

	// Force the manifest to be populated
	if _, err := img.RawManifest(); err != nil {
		return nil, err
	}
	return img, nil
}

// Destination returns the repo where the layer should be stored
// If no cache is specified, one is inferred from the destination provided
func Destination(opts *config.KanikoOptions, cacheKey string) (string, error) {
	cache := opts.CacheRepo
	if cache == "" {
		destination := opts.Destinations[0]
		destRef, err := name.NewTag(destination, name.WeakValidation)
		if err != nil {
			return "", errors.Wrap(err, "getting tag for destination")
		}
		return fmt.Sprintf("%s/cache:%s", destRef.Context(), cacheKey), nil
	}
	return fmt.Sprintf("%s:%s", cache, cacheKey), nil
}

// LocalSource retieves a source image from a local cache given cacheKey
func LocalSource(opts *config.KanikoOptions, cacheKey string) (v1.Image, error) {
	cache := opts.CacheDir
	if cache == "" {
		return nil, nil
	}

	path := path.Join(cache, cacheKey)

	fi, err := os.Stat(path)
	if err != nil {
		return nil, errors.Wrap(err, "geting file info")
	}

	// A stale cache is a bad cache
	expiry := fi.ModTime().Add(opts.CacheTTL)
	if expiry.Before(time.Now()) {
		logrus.Debugf("Cached image is too old: %v", fi.ModTime())
		return nil, nil
	}

	imgTar, err := tarball.ImageFromPath(path, nil)
	if err != nil {
		return nil, errors.Wrap(err, "getting image from path")
	}

	logrus.Infof("Found %s in local cache", cacheKey)
	return imgTar, nil
}
