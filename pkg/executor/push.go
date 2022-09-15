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
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/GoogleContainerTools/kaniko/pkg/cache"
	"github.com/GoogleContainerTools/kaniko/pkg/config"
	"github.com/GoogleContainerTools/kaniko/pkg/constants"
	"github.com/GoogleContainerTools/kaniko/pkg/creds"
	"github.com/GoogleContainerTools/kaniko/pkg/timing"
	"github.com/GoogleContainerTools/kaniko/pkg/util"
	"github.com/GoogleContainerTools/kaniko/pkg/version"
	"github.com/google/go-containerregistry/pkg/name"
	v1 "github.com/google/go-containerregistry/pkg/v1"
	"github.com/google/go-containerregistry/pkg/v1/empty"
	"github.com/google/go-containerregistry/pkg/v1/layout"
	"github.com/google/go-containerregistry/pkg/v1/mutate"
	"github.com/google/go-containerregistry/pkg/v1/remote"
	"github.com/google/go-containerregistry/pkg/v1/remote/transport"
	"github.com/google/go-containerregistry/pkg/v1/tarball"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
	"github.com/spf13/afero"
)

type withUserAgent struct {
	t http.RoundTripper
}

// for testing
var (
	newRetry = transport.NewRetry
)

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

// for testing
var (
	fs                        = afero.NewOsFs()
	checkRemotePushPermission = remote.CheckPushPermission
)

// CheckPushPermissions checks that the configured credentials can be used to
// push to every specified destination.
func CheckPushPermissions(opts *config.KanikoOptions) error {
	targets := opts.Destinations
	// When no push and no push cache are set, we don't need to check permissions
	if opts.NoPush && opts.NoPushCache {
		targets = []string{}
	} else if opts.NoPush && !opts.NoPushCache {
		// When no push is set, we want to check permissions for the cache repo
		// instead of the destinations
		if isOCILayout(opts.CacheRepo) {
			targets = []string{} // no need to check push permissions if we're just writing to disk
		} else {
			targets = []string{opts.CacheRepo}
		}
	}

	checked := map[string]bool{}
	for _, destination := range targets {
		destRef, err := name.NewTag(destination, name.WeakValidation)
		if err != nil {
			return errors.Wrap(err, "getting tag for destination")
		}
		if checked[destRef.Context().String()] {
			continue
		}

		registryName := destRef.Repository.Registry.Name()
		if opts.Insecure || opts.InsecureRegistries.Contains(registryName) {
			newReg, err := name.NewRegistry(registryName, name.WeakValidation, name.Insecure)
			if err != nil {
				return errors.Wrap(err, "getting new insecure registry")
			}
			destRef.Repository.Registry = newReg
		}
		tr := newRetry(util.MakeTransport(opts.RegistryOptions, registryName))
		if err := checkRemotePushPermission(destRef, creds.GetKeychain(), tr); err != nil {
			return errors.Wrapf(err, "checking push permission for %q", destRef)
		}
		checked[destRef.Context().String()] = true
	}
	return nil
}

func getDigest(image v1.Image) ([]byte, error) {
	digest, err := image.Digest()
	if err != nil {
		return nil, err
	}
	return []byte(digest.String()), nil
}

func writeDigestFile(path string, digestByteArray []byte) error {
	parentDir := filepath.Dir(path)
	if _, err := os.Stat(parentDir); os.IsNotExist(err) {
		if err := os.MkdirAll(parentDir, 0700); err != nil {
			logrus.Debugf("Error creating %s, %s", parentDir, err)
			return err
		}
		logrus.Tracef("Created directory %v", parentDir)
	}
	return ioutil.WriteFile(path, digestByteArray, 0644)
}

// DoPush is responsible for pushing image to the destinations specified in opts
func DoPush(image v1.Image, opts *config.KanikoOptions) error {
	t := timing.Start("Total Push Time")
	var digestByteArray []byte
	var builder strings.Builder
	if opts.DigestFile != "" || opts.ImageNameDigestFile != "" || opts.ImageNameTagDigestFile != "" {
		var err error
		digestByteArray, err = getDigest(image)
		if err != nil {
			return errors.Wrap(err, "error fetching digest")
		}
	}

	if opts.DigestFile != "" {
		err := writeDigestFile(opts.DigestFile, digestByteArray)
		if err != nil {
			return errors.Wrap(err, "writing digest to file failed")
		}
	}

	if opts.OCILayoutPath != "" {
		path, err := layout.Write(opts.OCILayoutPath, empty.Index)
		if err != nil {
			return errors.Wrap(err, "writing empty layout")
		}
		if err := path.AppendImage(image); err != nil {
			return errors.Wrap(err, "appending image")
		}
	}

	destRefs := []name.Tag{}
	for _, destination := range opts.Destinations {
		destRef, err := name.NewTag(destination, name.WeakValidation)
		if err != nil {
			return errors.Wrap(err, "getting tag for destination")
		}
		if opts.ImageNameDigestFile != "" || opts.ImageNameTagDigestFile != "" {
			tag := ""
			if opts.ImageNameTagDigestFile != "" && destRef.TagStr() != "" {
				tag = ":" + destRef.TagStr()
			}
			imageName := []byte(destRef.Repository.Name() + tag + "@")
			builder.Write(append(imageName, digestByteArray...))
			builder.WriteString("\n")
		}
		destRefs = append(destRefs, destRef)
	}

	if opts.ImageNameDigestFile != "" {
		err := writeDigestFile(opts.ImageNameDigestFile, []byte(builder.String()))
		if err != nil {
			return errors.Wrap(err, "writing image name with digest to file failed")
		}
	}

	if opts.ImageNameTagDigestFile != "" {
		err := writeDigestFile(opts.ImageNameTagDigestFile, []byte(builder.String()))
		if err != nil {
			return errors.Wrap(err, "writing image name with image tag and digest to file failed")
		}
	}

	if opts.TarPath != "" {
		tagToImage := map[name.Tag]v1.Image{}

		if len(destRefs) == 0 {
			return errors.New("must provide at least one destination when tarPath is specified")
		}

		for _, destRef := range destRefs {
			tagToImage[destRef] = image
		}
		err := tarball.MultiWriteToFile(opts.TarPath, tagToImage)
		if err != nil {
			return errors.Wrap(err, "writing tarball to file failed")
		}
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

		tr := newRetry(util.MakeTransport(opts.RegistryOptions, registryName))
		rt := &withUserAgent{t: tr}

		logrus.Infof("Pushing image to %s", destRef.String())

		retryFunc := func() error {
			dig, err := image.Digest()
			if err != nil {
				return err
			}
			if err := remote.Write(destRef, image, remote.WithAuth(pushAuth), remote.WithTransport(rt)); err != nil {
				return err
			}
			logrus.Infof("Pushed %s", destRef.Context().Digest(dig.String()))
			return nil
		}

		if err := util.Retry(retryFunc, opts.PushRetry, 1000); err != nil {
			return errors.Wrap(err, fmt.Sprintf("failed to push to destination %s", destRef))
		}
	}
	timing.DefaultRun.Stop(t)
	return writeImageOutputs(image, destRefs)
}

func writeImageOutputs(image v1.Image, destRefs []name.Tag) error {
	dir := os.Getenv("BUILDER_OUTPUT")
	if dir == "" {
		return nil
	}
	f, err := fs.Create(filepath.Join(dir, "images"))
	if err != nil {
		return err
	}
	defer f.Close()

	d, err := image.Digest()
	if err != nil {
		return err
	}

	type imageOutput struct {
		Name   string `json:"name"`
		Digest string `json:"digest"`
	}
	for _, r := range destRefs {
		if err := json.NewEncoder(f).Encode(imageOutput{
			Name:   r.String(),
			Digest: d.String(),
		}); err != nil {
			return err
		}
	}
	return nil
}

// pushLayerToCache pushes layer (tagged with cacheKey) to opts.CacheRepo
// if opts.CacheRepo doesn't exist, infer the cache from the given destination
func pushLayerToCache(opts *config.KanikoOptions, cacheKey string, tarPath string, createdBy string) error {
	var layer v1.Layer
	var err error
	if opts.CompressedCaching == true {
		layer, err = tarball.LayerFromFile(tarPath, tarball.WithCompressedCaching)
	} else {
		layer, err = tarball.LayerFromFile(tarPath)
	}
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
	if isOCILayout(cache) {
		cacheOpts.OCILayoutPath = strings.TrimPrefix(cache, "oci:")
		cacheOpts.NoPush = true
	}
	return DoPush(empty, &cacheOpts)
}
