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

package main

import (
	"log"
	"os"

	"github.com/google/go-containerregistry/pkg/authn/k8schain"
	"github.com/google/go-containerregistry/pkg/name"
	"github.com/google/go-containerregistry/pkg/v1/remote"
)

func main() {
	if len(os.Args) != 2 {
		log.Fatalf("expected usage: <command> <arg>, got: %v", os.Args)
	}

	kc, err := k8schain.NewInCluster(k8schain.Options{})
	if err != nil {
		log.Fatalf("k8schain.New() = %v", err)
	}

	ref, err := name.NewDigest(os.Args[1])
	if err != nil {
		log.Fatalf("NewDigest() = %v", err)
	}

	img, err := remote.Image(ref, remote.WithAuthFromKeychain(kc))
	if err != nil {
		log.Fatalf("remote.Image() = %v", err)
	}

	digest, err := img.Digest()
	if err != nil {
		log.Fatalf("Digest() = %v", err)
	}
	log.Printf("got digest: %v", digest)
}
