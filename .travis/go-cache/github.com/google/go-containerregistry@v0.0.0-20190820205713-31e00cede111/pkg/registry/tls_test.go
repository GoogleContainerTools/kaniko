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

package registry_test

import (
	"testing"

	"github.com/google/go-containerregistry/pkg/name"
	"github.com/google/go-containerregistry/pkg/registry"
	"github.com/google/go-containerregistry/pkg/v1/random"
	"github.com/google/go-containerregistry/pkg/v1/remote"
)

func TestTLS(t *testing.T) {
	s, err := registry.TLS("registry.example.com")
	if err != nil {
		t.Fatal(err)
	}
	defer s.Close()

	i, err := random.Image(1024, 1)
	if err != nil {
		t.Fatalf("Unable to make image: %v", err)
	}
	rd, err := i.Digest()
	if err != nil {
		t.Fatalf("Unable to get image digest: %v", err)
	}

	d, err := name.NewDigest("registry.example.com/foo@" + rd.String())
	if err != nil {
		t.Fatalf("Unable to parse digest: %v", err)
	}
	if err := remote.Write(d, i, remote.WithTransport(s.Client().Transport)); err != nil {
		t.Fatalf("Unable to write image to remote: %s", err)
	}
}
