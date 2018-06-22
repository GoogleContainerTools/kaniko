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

	"github.com/google/go-containerregistry/pkg/v1/tarball"

	"github.com/docker/docker/client"
	"github.com/google/go-containerregistry/pkg/name"
	"github.com/google/go-containerregistry/pkg/v1"
)

// image accesses an image from a docker daemon
type image struct {
	v1.Image
}

type ReadOptions struct {
	Buffer bool
}

var _ v1.Image = (*image)(nil)

// API interface for testing.
type ImageSaver interface {
	ImageSave(context.Context, []string) (io.ReadCloser, error)
}

// This is a variable so we can override in tests.
var getImageSaver = func() (ImageSaver, error) {
	return client.NewEnvClient()
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

// Image exposes an image reference from within the Docker daemon.
func Image(ref name.Reference, ro *ReadOptions) (v1.Image, error) {
	var opener tarball.Opener
	var err error
	if ro.Buffer {
		opener, err = bufferedOpener(ref)
	} else {
		opener, err = unbufferedOpener(ref)
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
