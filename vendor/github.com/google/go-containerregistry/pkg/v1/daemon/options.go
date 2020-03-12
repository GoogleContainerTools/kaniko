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

	"github.com/docker/docker/api/types"
)

// WithBufferedOpener buffers the image.
func WithBufferedOpener() ImageOption {
	return func(i *imageOpener) error {
		return i.setBuffered(true)
	}
}

// WithUnbufferedOpener streams the image to avoid buffering.
func WithUnbufferedOpener() ImageOption {
	return func(i *imageOpener) error {
		return i.setBuffered(false)
	}
}

func (i *imageOpener) setBuffered(buffer bool) error {
	i.buffered = buffer
	return nil
}

// WithClient is a functional option to allow injecting a docker client.
//
// By default, github.com/docker/docker/client.FromEnv is used.
func WithClient(client Client) ImageOption {
	return func(i *imageOpener) error {
		i.client = client
		return nil
	}
}

// Client represents the subset of a docker client that the daemon
// package uses.
type Client interface {
	NegotiateAPIVersion(ctx context.Context)
	ImageSave(context.Context, []string) (io.ReadCloser, error)
	ImageLoad(context.Context, io.Reader, bool) (types.ImageLoadResponse, error)
	ImageTag(context.Context, string, string) error
}
