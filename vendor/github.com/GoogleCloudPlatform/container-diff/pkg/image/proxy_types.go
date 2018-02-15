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
	"github.com/containers/image/types"
)

// ProxySource is a type that implements types.ImageSource by proxying all calls to an underlying implementation.
type ProxySource struct {
	Ref types.ImageReference
	types.ImageSource
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
		Ref:         ref,
		img:         img,
		ImageSource: src,
	}, nil
}

func (p *ProxySource) Reference() types.ImageReference {
	return p.Ref
}

func (p *ProxySource) LayerInfosForCopy() []types.BlobInfo {
	return nil
}

// ProxyReference implements types.Reference by proxying calls to an underlying implementation.
type ProxyReference struct {
	types.ImageReference
	Src types.ImageSource
}

func (p *ProxyReference) NewImageSource(ctx *types.SystemContext) (types.ImageSource, error) {
	return p.Src, nil
}
