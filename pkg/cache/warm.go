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
	"bytes"
	"io"
	"io/ioutil"
	"os"
	"path"

	"github.com/GoogleContainerTools/kaniko/pkg/config"
	"github.com/GoogleContainerTools/kaniko/pkg/image/remote"
	"github.com/google/go-containerregistry/pkg/name"
	v1 "github.com/google/go-containerregistry/pkg/v1"
	"github.com/google/go-containerregistry/pkg/v1/tarball"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
)

// WarmCache populates the cache
func WarmCache(opts *config.WarmerOptions) error {
	cacheDir := opts.CacheDir
	images := opts.Images
	logrus.Debugf("%s\n", cacheDir)
	logrus.Debugf("%s\n", images)

	errs := 0
	for _, img := range images {
		tarBuf := new(bytes.Buffer)
		manifestBuf := new(bytes.Buffer)

		cw := &Warmer{
			Remote:         remote.RetrieveRemoteImage,
			Local:          LocalSource,
			TarWriter:      tarBuf,
			ManifestWriter: manifestBuf,
		}

		digest, err := cw.Warm(img, opts)
		if err != nil {
			if !IsAlreadyCached(err) {
				logrus.Warnf("Error while trying to warm image: %v %v", img, err)
				errs++
			}

			continue
		}

		cachePath := path.Join(cacheDir, digest.String())

		if err := writeBufsToFile(cachePath, tarBuf, manifestBuf); err != nil {
			logrus.Warnf("Error while writing %v to cache: %v", img, err)
			errs++
			continue
		}

		logrus.Debugf("Wrote %s to cache", img)
	}

	if len(images) == errs {
		return errors.New("Failed to warm any of the given images")
	}

	return nil
}

func writeBufsToFile(cachePath string, tarBuf, manifestBuf *bytes.Buffer) error {
	f, err := os.Create(cachePath)
	if err != nil {
		return err
	}
	defer f.Close()

	if _, err := f.Write(tarBuf.Bytes()); err != nil {
		return errors.Wrap(err, "Failed to save tar to file")
	}

	mfstPath := cachePath + ".json"
	if err := ioutil.WriteFile(mfstPath, manifestBuf.Bytes(), 0666); err != nil {
		return errors.Wrap(err, "Failed to save manifest to file")
	}

	return nil
}

// FetchRemoteImage retrieves a Docker image manifest from a remote source.
// github.com/GoogleContainerTools/kaniko/image/remote.RetrieveRemoteImage can be used as
// this type.
type FetchRemoteImage func(image string, opts config.RegistryOptions, customPlatform string) (v1.Image, error)

// FetchLocalSource retrieves a Docker image manifest from a local source.
// github.com/GoogleContainerTools/kaniko/cache.LocalSource can be used as
// this type.
type FetchLocalSource func(*config.CacheOptions, string) (v1.Image, error)

// Warmer is used to prepopulate the cache with a Docker image
type Warmer struct {
	Remote         FetchRemoteImage
	Local          FetchLocalSource
	TarWriter      io.Writer
	ManifestWriter io.Writer
}

// Warm retrieves a Docker image and populates the supplied buffer with the image content and manifest
// or returns an AlreadyCachedErr if the image is present in the cache.
func (w *Warmer) Warm(image string, opts *config.WarmerOptions) (v1.Hash, error) {
	cacheRef, err := name.ParseReference(image, name.WeakValidation)
	if err != nil {
		return v1.Hash{}, errors.Wrapf(err, "Failed to verify image name: %s", image)
	}

	img, err := w.Remote(image, opts.RegistryOptions, opts.CustomPlatform)
	if err != nil || img == nil {
		return v1.Hash{}, errors.Wrapf(err, "Failed to retrieve image: %s", image)
	}

	digest, err := img.Digest()
	if err != nil {
		return v1.Hash{}, errors.Wrapf(err, "Failed to retrieve digest: %s", image)
	}

	if !opts.Force {
		_, err := w.Local(&opts.CacheOptions, digest.String())
		if err == nil || IsExpired(err) {
			return v1.Hash{}, AlreadyCachedErr{}
		}
	}

	err = tarball.Write(cacheRef, img, w.TarWriter)
	if err != nil {
		return v1.Hash{}, errors.Wrapf(err, "Failed to write %s to tar buffer", image)
	}

	mfst, err := img.RawManifest()
	if err != nil {
		return v1.Hash{}, errors.Wrapf(err, "Failed to retrieve manifest for %s", image)
	}

	if _, err := w.ManifestWriter.Write(mfst); err != nil {
		return v1.Hash{}, errors.Wrapf(err, "Failed to save manifest to buffer for %s", image)
	}

	return digest, nil
}
