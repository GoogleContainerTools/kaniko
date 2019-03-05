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
	"bytes"
	"context"
	"io"
	"io/ioutil"

	"github.com/docker/docker/client"
	"github.com/google/go-containerregistry/pkg/name"
	v1 "github.com/google/go-containerregistry/pkg/v1"
	"github.com/google/go-containerregistry/pkg/v1/tarball"
)

// image accesses an image from a docker daemon
type image struct {
	v1.Image

	opener tarball.Opener
	ref    name.Reference
}

var _ v1.Image = (*image)(nil)

type imageOpener struct {
	ref      name.Reference
	buffered bool
}

// ImageOption is a functional option for Image.
type ImageOption func(*imageOpener) error

func (i *imageOpener) Open() (v1.Image, error) {
	var opener tarball.Opener
	var err error
	if i.buffered {
		opener, err = bufferedOpener(i.ref)
	} else {
		opener, err = unbufferedOpener(i.ref)
	}
	if err != nil {
		return nil, err
	}

	tb, err := tarball.Image(opener, nil)
	if err != nil {
		return nil, err
	}
	img := &image{
		Image: tb,
	}
	return img, nil
}

// ImageSaver is an interface for testing.
type ImageSaver interface {
	ImageSave(context.Context, []string) (io.ReadCloser, error)
}

// This is a variable so we can override in tests.
var getImageSaver = func() (ImageSaver, error) {
	cli, err := client.NewEnvClient()
	if err != nil {
		return nil, err
	}
	cli.NegotiateAPIVersion(context.Background())
	return cli, nil
}

func saveImage(ref name.Reference) (io.ReadCloser, error) {
	cli, err := getImageSaver()
	if err != nil {
		return nil, err
	}

	return cli.ImageSave(context.Background(), []string{ref.Name()})
}

func bufferedOpener(ref name.Reference) (tarball.Opener, error) {
	// Store the tarball in memory and return a new reader into the bytes each time we need to access something.
	rc, err := saveImage(ref)
	if err != nil {
		return nil, err
	}
	defer rc.Close()

	imageBytes, err := ioutil.ReadAll(rc)
	if err != nil {
		return nil, err
	}
	// The tarball interface takes a function that it can call to return an opened reader-like object.
	// Daemon comes from a set of bytes, so wrap them in a ReadCloser so it looks like an opened file.
	return func() (io.ReadCloser, error) {
		return ioutil.NopCloser(bytes.NewReader(imageBytes)), nil
	}, nil
}

func unbufferedOpener(ref name.Reference) (tarball.Opener, error) {
	// To avoid storing the tarball in memory, do a save every time we need to access something.
	return func() (io.ReadCloser, error) {
		return saveImage(ref)
	}, nil
}

// Image provides access to an image reference from the Docker daemon,
// applying functional options to the underlying imageOpener before
// resolving the reference into a v1.Image.
func Image(ref name.Reference, options ...ImageOption) (v1.Image, error) {
	i := &imageOpener{
		ref:      ref,
		buffered: true, // buffer by default
	}
	for _, option := range options {
		if err := option(i); err != nil {
			return nil, err
		}
	}
	return i.Open()
}
