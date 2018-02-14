/*
Copyright 2018 Google, Inc. All rights reserved.

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
package util

import (
	"archive/tar"
	"fmt"
	"github.com/containers/image/docker"
	"github.com/containers/image/pkg/compression"
	"github.com/containers/image/signature"
	"github.com/containers/image/types"
)

var dir = "/"
var whiteouts map[string]bool

func getFileSystemFromReference(ref types.ImageReference, imgSrc types.ImageSource, path string) error {
	img, err := ref.NewImage(nil)
	if err != nil {
		return err
	}
	defer img.Close()
	whiteouts = make(map[string]bool)
	for _, b := range img.LayerInfos() {
		bi, _, err := imgSrc.GetBlob(b)
		if err != nil {
			return err
		}
		defer bi.Close()
		f, reader, err := compression.DetectCompression(bi)
		if err != nil {
			return err
		}
		// Decompress if necessary.
		if f != nil {
			reader, err = f(reader)
			if err != nil {
				return err
			}
		}
		tr := tar.NewReader(reader)
		err = unpackTar(tr, path)
		if err != nil {
			return err
		}
	}
	return nil
}

func getPolicyContext() (*signature.PolicyContext, error) {
	policy, err := signature.DefaultPolicy(nil)
	if err != nil {
		fmt.Println("Error retrieving policy")
		return nil, err
	}

	policyContext, err := signature.NewPolicyContext(policy)
	if err != nil {
		fmt.Println("Error retrieving policy context")
		return nil, err
	}
	return policyContext, nil
}

// GetFileSystemFromImage pulls an image and unpacks it to a file system at root
func GetFileSystemFromImage(img string) error {
	ref, err := docker.ParseReference("//" + img)
	if err != nil {
		return err
	}
	imgSrc, err := ref.NewImageSource(nil)
	if err != nil {
		return err
	}
	return getFileSystemFromReference(ref, imgSrc, dir)
}
