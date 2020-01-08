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

	"github.com/google/go-containerregistry/pkg/authn/k8schain"
	"github.com/google/go-containerregistry/pkg/name"
	"github.com/google/go-containerregistry/pkg/v1/remote"
)

func main() {
	ref, err := name.NewTag("gcr.io/build-crd-testing/secret-sauce:latest")
	if err != nil {
		log.Fatalf("NewTag() = %v", err)
	}

	kc, err := k8schain.NewInCluster(k8schain.Options{
		Namespace:          "serviceaccount-namespace",
		ServiceAccountName: "serviceaccount",
		// This is the name of the imagePullSecrets attached to this service account.
		// ImagePullSecrets: []string{
		// 	"serviceaccount-secret",
		// },
	})
	if err != nil {
		log.Fatalf("k8schain.New() = %v", err)
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
