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
	"path/filepath"
	"strings"
	"time"

	"github.com/GoogleContainerTools/kaniko/pkg/config"
	"github.com/GoogleContainerTools/kaniko/pkg/creds"
	"github.com/GoogleContainerTools/kaniko/pkg/util"
	"github.com/google/go-containerregistry/pkg/name"
	v1 "github.com/google/go-containerregistry/pkg/v1"
	"github.com/google/go-containerregistry/pkg/v1/layout"
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

	registryName := cacheRef.Repository.Registry.Name()
	if rc.Opts.Insecure || rc.Opts.InsecureRegistries.Contains(registryName) {
		newReg, err := name.NewRegistry(registryName, name.WeakValidation, name.Insecure)
		if err != nil {
			return nil, err
		}
		cacheRef.Repository.Registry = newReg
	}

	tr, err := util.MakeTransport(rc.Opts.RegistryOptions, registryName)
	if err != nil {
		return nil, errors.Wrapf(err, "making transport for registry %q", registryName)
	}

	img, err := remote.Image(cacheRef, remote.WithTransport(tr), remote.WithAuthFromKeychain(creds.GetKeychain()))
	if err != nil {
		return nil, err
	}

	if err = verifyImage(img, rc.Opts.CacheTTL, cache); err != nil {
		return nil, err
	}
	return img, nil
}

func verifyImage(img v1.Image, cacheTTL time.Duration, cache string) error {
	cf, err := img.ConfigFile()
	if err != nil {
		return errors.Wrap(err, fmt.Sprintf("retrieving config file for %s", cache))
	}

	expiry := cf.Created.Add(cacheTTL)
	// Layer is stale, rebuild it.
	if expiry.Before(time.Now()) {
		logrus.Infof("Cache entry expired: %s", cache)
		return fmt.Errorf("Cache entry expired: %s", cache)
	}

	// Force the manifest to be populated
	if _, err := img.RawManifest(); err != nil {
		return err
	}
	return nil
}

// LayoutCache is the OCI image layout cache
type LayoutCache struct {
	Opts *config.KanikoOptions
}

func (lc *LayoutCache) RetrieveLayer(ck string) (v1.Image, error) {
	cache, err := Destination(lc.Opts, ck)
	if err != nil {
		return nil, errors.Wrap(err, "getting cache destination")
	}
	logrus.Infof("Checking for cached layer %s...", cache)

	var img v1.Image
	if img, err = locateImage(strings.TrimPrefix(cache, "oci:")); err != nil {
		return nil, errors.Wrap(err, "locating cache image")
	}

	if err = verifyImage(img, lc.Opts.CacheTTL, cache); err != nil {
		return nil, err
	}
	return img, nil
}

func locateImage(path string) (v1.Image, error) {
	var img v1.Image
	layoutPath, err := layout.FromPath(path)
	if err != nil {
		return nil, errors.Wrap(err, fmt.Sprintf("constructing layout path from %s", path))
	}
	index, err := layoutPath.ImageIndex()
	if err != nil {
		return nil, errors.Wrap(err, fmt.Sprintf("retrieving index file for %s", layoutPath))
	}
	manifest, err := index.IndexManifest()
	if err != nil {
		return nil, errors.Wrap(err, fmt.Sprintf("retrieving manifest file for %s", layoutPath))
	}
	for _, m := range manifest.Manifests {
		// assume there is only one image
		img, err = layoutPath.Image(m.Digest)
		if err != nil {
			return nil, errors.Wrap(err, fmt.Sprintf("initializing image with digest %s", m.Digest.String()))
		}
	}
	if img == nil {
		return nil, fmt.Errorf("path contains no images")
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

// LocalSource retrieves a source image from a local cache given cacheKey
func LocalSource(opts *config.CacheOptions, cacheKey string) (v1.Image, error) {
	cache := opts.CacheDir
	if cache == "" {
		return nil, nil
	}

	path := path.Join(cache, cacheKey)

	fi, err := os.Stat(path)
	if err != nil {
		msg := fmt.Sprintf("No file found for cache key %v %v", cacheKey, err)
		logrus.Debug(msg)
		return nil, NotFoundErr{msg: msg}
	}

	// A stale cache is a bad cache
	expiry := fi.ModTime().Add(opts.CacheTTL)
	if expiry.Before(time.Now()) {
		msg := fmt.Sprintf("Cached image is too old: %v", fi.ModTime())
		logrus.Debug(msg)
		return nil, ExpiredErr{msg: msg}
	}

	logrus.Infof("Found %s in local cache", cacheKey)
	return cachedImageFromPath(path)
}

// cachedImage represents a v1.Tarball that is cached locally in a CAS.
// Computing the digest for a v1.Tarball is very expensive. If the tarball
// is named with the digest we can store this and return it directly rather
// than recompute it.
type cachedImage struct {
	digest string
	v1.Image
	mfst *v1.Manifest
}

func (c *cachedImage) Digest() (v1.Hash, error) {
	return v1.NewHash(c.digest)
}

func (c *cachedImage) Manifest() (*v1.Manifest, error) {
	if c.mfst == nil {
		return c.Image.Manifest()
	}
	return c.mfst, nil
}

func mfstFromPath(p string) (*v1.Manifest, error) {
	f, err := os.Open(p)
	if err != nil {
		return nil, err
	}
	defer f.Close()
	return v1.ParseManifest(f)
}

func cachedImageFromPath(p string) (v1.Image, error) {
	imgTar, err := tarball.ImageFromPath(p, nil)
	if err != nil {
		return nil, errors.Wrap(err, "getting image from path")
	}

	// Manifests may be present next to the tar, named with a ".json" suffix
	mfstPath := p + ".json"

	var mfst *v1.Manifest
	if _, err := os.Stat(mfstPath); err != nil {
		logrus.Debugf("Manifest does not exist at file: %s", mfstPath)
	} else {
		mfst, err = mfstFromPath(mfstPath)
		if err != nil {
			logrus.Debugf("Error parsing manifest from file: %s", mfstPath)
		} else {
			logrus.Infof("Found manifest at %s", mfstPath)
		}
	}

	return &cachedImage{
		digest: filepath.Base(p),
		Image:  imgTar,
		mfst:   mfst,
	}, nil
}
