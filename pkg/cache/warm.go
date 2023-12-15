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
	"io"
	"net/http"
	"os"
	"path"
	"regexp"

	"github.com/GoogleContainerTools/kaniko/pkg/config"
	"github.com/GoogleContainerTools/kaniko/pkg/dockerfile"
	"github.com/GoogleContainerTools/kaniko/pkg/image/remote"
	"github.com/GoogleContainerTools/kaniko/pkg/util"
	"github.com/google/go-containerregistry/pkg/name"
	v1 "github.com/google/go-containerregistry/pkg/v1"
	"github.com/google/go-containerregistry/pkg/v1/tarball"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
)

// WarmCache populates the cache
func WarmCache(opts *config.WarmerOptions) error {
	var dockerfileImages []string
	cacheDir := opts.CacheDir
	images := opts.Images

	// if opts.image is empty,we need to parse dockerfilepath to get images list
	if opts.DockerfilePath != "" {
		var err error
		if dockerfileImages, err = ParseDockerfile(opts); err != nil {
			return errors.Wrap(err, "failed to parse Dockerfile")
		}
	}

	// TODO: Implement deduplication logic later.
	images = append(images, dockerfileImages...)

	logrus.Debugf("%s\n", cacheDir)
	logrus.Debugf("%s\n", images)

	errs := 0
	for _, img := range images {
		err := warmToFile(cacheDir, img, opts)
		if err != nil {
			logrus.Warnf("Error while trying to warm image: %v %v", img, err)
			errs++
		}
	}

	if len(images) == errs {
		return errors.New("Failed to warm any of the given images")
	}

	return nil
}

// Download image in temporary files then move files to final destination
func warmToFile(cacheDir, img string, opts *config.WarmerOptions) error {
	f, err := os.CreateTemp(cacheDir, "warmingImage.*")
	if err != nil {
		return err
	}
	// defer called in reverse order
	defer os.Remove(f.Name())
	defer f.Close()

	mtfsFile, err := os.CreateTemp(cacheDir, "warmingManifest.*")
	if err != nil {
		return err
	}
	defer os.Remove(mtfsFile.Name())
	defer mtfsFile.Close()

	cw := &Warmer{
		Remote:         remote.RetrieveRemoteImage,
		Local:          LocalSource,
		TarWriter:      f,
		ManifestWriter: mtfsFile,
	}

	digest, err := cw.Warm(img, opts)
	if err != nil {
		if IsAlreadyCached(err) {
			logrus.Infof("Image already in cache: %v", img)
			return nil
		}
		logrus.Warnf("Error while trying to warm image: %v %v", img, err)
		return err
	}

	finalCachePath := path.Join(cacheDir, digest.String())
	finalMfstPath := finalCachePath + ".json"

	err = os.Rename(f.Name(), finalCachePath)
	if err != nil {
		return err
	}

	err = os.Rename(mtfsFile.Name(), finalMfstPath)
	if err != nil {
		return errors.Wrap(err, "Failed to rename manifest file")
	}

	logrus.Debugf("Wrote %s to cache", img)
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

func ParseDockerfile(opts *config.WarmerOptions) ([]string, error) {
	var err error
	var d []uint8
	var baseNames []string
	match, _ := regexp.MatchString("^https?://", opts.DockerfilePath)
	if match {
		response, e := http.Get(opts.DockerfilePath) //nolint:noctx
		if e != nil {
			return nil, e
		}
		d, err = io.ReadAll(response.Body)
	} else {
		d, err = os.ReadFile(opts.DockerfilePath)
	}

	if err != nil {
		return nil, errors.Wrap(err, fmt.Sprintf("reading dockerfile at path %s", opts.DockerfilePath))
	}

	stages, _, err := dockerfile.Parse(d)
	if err != nil {
		return nil, errors.Wrap(err, "parsing dockerfile")
	}

	for i, s := range stages {
		resolvedBaseName, err := util.ResolveEnvironmentReplacement(s.BaseName, opts.BuildArgs, false)
		if err != nil {
			return nil, errors.Wrap(err, fmt.Sprintf("resolving base name %s", s.BaseName))
		}
		if s.BaseName != resolvedBaseName {
			stages[i].BaseName = resolvedBaseName
		}
		baseNames = append(baseNames, resolvedBaseName)
	}
	return baseNames, nil

}
