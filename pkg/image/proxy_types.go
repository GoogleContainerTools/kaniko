/*
Copyright 2017 Google, Inc. All rights reserved.
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

package image

import (
	"context"
	"github.com/containers/image/docker/reference"
	"io"

	"github.com/containers/image/types"
	digest "github.com/opencontainers/go-digest"
)

// ProxySource is a type that implements types.ImageSource by proxying all calls to an underlying implementation.
type ProxySource struct {
	Ref types.ImageReference
	src types.ImageSource
	img types.Image
}

func NewProxySource(ref types.ImageReference) (*ProxySource, error) {
	src, err := ref.NewImageSource(nil)
	if err != nil {
		return nil, err
	}
	img, err := ref.NewImage(nil)
	if err != nil {
		return nil, err
	}

	return &ProxySource{
		Ref: ref,
		img: img,
		src: src,
	}, nil
}

func (p ProxySource) Reference() types.ImageReference {
	return p.Ref
}

func (p ProxySource) Close() error {
	return nil
}

func (p ProxySource) GetTargetManifest(digest digest.Digest) ([]byte, string, error) {
	return p.GetTargetManifest(digest)
}

// GetSignatures returns the image's signatures.  It may use a remote (= slow) service.
func (p ProxySource) GetSignatures(ctx context.Context, d *digest.Digest) ([][]byte, error) {
	return p.src.GetSignatures(ctx, d)
}

func (p ProxySource) LayerInfosForCopy() []types.BlobInfo {
	return nil
}

func (p ProxySource) GetBlob(b types.BlobInfo) (io.ReadCloser, int64, error) {
	return p.src.GetBlob(b)
}

func (p ProxySource) GetManifest(d *digest.Digest) ([]byte, string, error) {
	return p.src.GetManifest(d)
}

// ProxyReference implements types.Reference by proxying calls to an underlying implementation.
type ProxyReference struct {
	ref types.ImageReference
	src types.ImageSource
}

func NewProxyReference(ref types.ImageReference, src types.ImageSource) (*ProxyReference, error) {
	return &ProxyReference{
		ref: ref,
		src: src,
	}, nil
}

func (p ProxyReference) Transport() types.ImageTransport {
	return p.ref.Transport()
}

func (p ProxyReference) StringWithinTransport() string {
	return p.ref.StringWithinTransport()
}

func (p ProxyReference) DockerReference() reference.Named {
	return p.ref.DockerReference()
}

func (p ProxyReference) PolicyConfigurationIdentity() string {
	return p.ref.PolicyConfigurationIdentity()
}

func (p ProxyReference) PolicyConfigurationNamespaces() []string {
	return p.ref.PolicyConfigurationNamespaces()
}

func (p ProxyReference) NewImage(ctx *types.SystemContext) (types.ImageCloser, error) {
	return p.ref.NewImage(ctx)
}

func (p ProxyReference) NewImageSource(ctx *types.SystemContext) (types.ImageSource, error) {
	return p.src, nil
}

func (p ProxyReference) NewImageDestination(ctx *types.SystemContext) (types.ImageDestination, error) {
	return p.ref.NewImageDestination(ctx)
}

func (p ProxyReference) DeleteImage(ctx *types.SystemContext) error {
	return nil
}
