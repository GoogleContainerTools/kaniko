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

package crane

import (
	"fmt"
	"log"

	"github.com/google/go-containerregistry/pkg/authn"
	"github.com/google/go-containerregistry/pkg/logs"
	"github.com/google/go-containerregistry/pkg/name"
	"github.com/google/go-containerregistry/pkg/v1/remote"
	"github.com/google/go-containerregistry/pkg/v1/types"
)

// Copy copies a remote image or index from src to dst.
func Copy(src, dst string) error {
	srcRef, err := name.ParseReference(src)
	if err != nil {
		return fmt.Errorf("parsing reference %q: %v", src, err)
	}
	logs.Progress.Printf("Pulling %v", srcRef)

	desc, err := remote.Get(srcRef, remote.WithAuthFromKeychain(authn.DefaultKeychain))
	if err != nil {
		return fmt.Errorf("parsing reference %q: %v", dst, err)
	}

	dstRef, err := name.ParseReference(dst)
	if err != nil {
		return fmt.Errorf("getting creds for %q: %v", srcRef, err)
	}

	logs.Progress.Printf("Pushing %v", dstRef)

	switch desc.MediaType {
	case types.OCIImageIndex, types.DockerManifestList:
		// Handle indexes separately.
		if err := copyIndex(desc, dstRef); err != nil {
			log.Fatalf("failed to copy index: %v", err)
		}
	default:
		// Assume anything else is an image, since some registries don't set mediaTypes properly.
		if err := copyImage(desc, dstRef); err != nil {
			log.Fatalf("failed to copy image: %v", err)
		}
	}

	return nil
}

func copyImage(desc *remote.Descriptor, dstRef name.Reference) error {
	img, err := desc.Image()
	if err != nil {
		return err
	}
	return remote.Write(dstRef, img, remote.WithAuthFromKeychain(authn.DefaultKeychain))
}

func copyIndex(desc *remote.Descriptor, dstRef name.Reference) error {
	idx, err := desc.ImageIndex()
	if err != nil {
		return err
	}
	return remote.WriteIndex(dstRef, idx, remote.WithAuthFromKeychain(authn.DefaultKeychain))
}
