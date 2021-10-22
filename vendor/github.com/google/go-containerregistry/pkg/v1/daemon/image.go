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
	"sync"

	"github.com/google/go-containerregistry/pkg/name"
	v1 "github.com/google/go-containerregistry/pkg/v1"
	"github.com/google/go-containerregistry/pkg/v1/tarball"
)

type imageOpener struct {
	ref name.Reference
	ctx context.Context

	buffered bool
	client   Client

	once  sync.Once
	bytes []byte
	err   error
}

func (i *imageOpener) saveImage() (io.ReadCloser, error) {
	return i.client.ImageSave(i.ctx, []string{i.ref.Name()})
}

func (i *imageOpener) bufferedOpener() (io.ReadCloser, error) {
	// Store the tarball in memory and return a new reader into the bytes each time we need to access something.
	i.once.Do(func() {
		i.bytes, i.err = func() ([]byte, error) {
			rc, err := i.saveImage()
			if err != nil {
				return nil, err
			}
			defer rc.Close()

			return ioutil.ReadAll(rc)
		}()
	})

	// Wrap the bytes in a ReadCloser so it looks like an opened file.
	return ioutil.NopCloser(bytes.NewReader(i.bytes)), i.err
}

func (i *imageOpener) opener() tarball.Opener {
	if i.buffered {
		return i.bufferedOpener
	}

	// To avoid storing the tarball in memory, do a save every time we need to access something.
	return i.saveImage
}

// Image provides access to an image reference from the Docker daemon,
// applying functional options to the underlying imageOpener before
// resolving the reference into a v1.Image.
func Image(ref name.Reference, options ...Option) (v1.Image, error) {
	o, err := makeOptions(options...)
	if err != nil {
		return nil, err
	}

	i := &imageOpener{
		ref:      ref,
		buffered: o.buffered,
		client:   o.client,
		ctx:      o.ctx,
	}

	return tarball.Image(i.opener(), nil)
}
