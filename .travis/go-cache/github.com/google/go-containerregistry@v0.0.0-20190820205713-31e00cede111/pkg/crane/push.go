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
	"log"

	"github.com/google/go-containerregistry/pkg/authn"
	"github.com/google/go-containerregistry/pkg/name"
	v1 "github.com/google/go-containerregistry/pkg/v1"
	"github.com/google/go-containerregistry/pkg/v1/remote"
	"github.com/google/go-containerregistry/pkg/v1/tarball"
)

// Load reads the tarball at path as a v1.Image.
func Load(path string) (v1.Image, error) {
	// TODO: Allow tag?
	return tarball.ImageFromPath(path, nil)
}

// Push pushes the v1.Image img to a registry as dst.
func Push(img v1.Image, dst string) error {
	tag, err := name.NewTag(dst)
	if err != nil {
		log.Fatalf("parsing tag %q: %v", dst, err)
	}

	return remote.Write(tag, img, remote.WithAuthFromKeychain(authn.DefaultKeychain))
}
