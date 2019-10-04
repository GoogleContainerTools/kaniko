// Copyright 2018 Google LLC All Rights Reserved.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//    http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package daemon

import (
	"context"
	"io"
	"io/ioutil"

	"github.com/docker/docker/api/types"
	"github.com/docker/docker/client"
	"github.com/google/go-containerregistry/pkg/name"
	v1 "github.com/google/go-containerregistry/pkg/v1"
	"github.com/google/go-containerregistry/pkg/v1/tarball"
	"github.com/pkg/errors"
)

// ImageLoader is an interface for testing.
type ImageLoader interface {
	ImageLoad(context.Context, io.Reader, bool) (types.ImageLoadResponse, error)
	ImageTag(context.Context, string, string) error
}

// GetImageLoader is a variable so we can override in tests.
var GetImageLoader = func() (ImageLoader, error) {
	cli, err := client.NewEnvClient()
	if err != nil {
		return nil, err
	}
	cli.NegotiateAPIVersion(context.Background())
	return cli, nil
}

// Tag adds a tag to an already existent image.
func Tag(src, dest name.Tag) error {
	cli, err := GetImageLoader()
	if err != nil {
		return err
	}

	return cli.ImageTag(context.Background(), src.String(), dest.String())
}

// Write saves the image into the daemon as the given tag.
func Write(tag name.Tag, img v1.Image) (string, error) {
	cli, err := GetImageLoader()
	if err != nil {
		return "", err
	}

	pr, pw := io.Pipe()
	go func() {
		pw.CloseWithError(tarball.Write(tag, img, pw))
	}()

	// write the image in docker save format first, then load it
	resp, err := cli.ImageLoad(context.Background(), pr, false)
	if err != nil {
		return "", errors.Wrapf(err, "error loading image")
	}
	defer resp.Body.Close()
	b, readErr := ioutil.ReadAll(resp.Body)
	response := string(b)
	if readErr != nil {
		return response, errors.Wrapf(err, "error reading load response body")
	}
	return response, nil
}
