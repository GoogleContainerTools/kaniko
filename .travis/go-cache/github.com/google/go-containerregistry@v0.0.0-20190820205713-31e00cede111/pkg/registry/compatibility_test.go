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
	"bytes"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/google/go-containerregistry/pkg/name"
	"github.com/google/go-containerregistry/pkg/registry"
	"github.com/google/go-containerregistry/pkg/v1/random"
	"github.com/google/go-containerregistry/pkg/v1/remote"
	"github.com/google/go-containerregistry/pkg/v1/tarball"
)

func TestPushAndPullContainer(t *testing.T) {
	s := httptest.NewServer(registry.New())
	defer s.Close()

	r := strings.TrimPrefix(s.URL, "http://") + "/foo:latest"
	d, err := name.NewTag(r)
	if err != nil {
		t.Fatalf("Unable to create tag: %v", err)
	}

	i, err := random.Image(1024, 1)
	if err != nil {
		t.Fatalf("Unable to make random image: %v", err)
	}

	if err := remote.Write(d, i); err != nil {
		t.Fatalf("Error writing image : %v", err)
	}

	ref, err := name.ParseReference(r)
	if err != nil {
		t.Fatalf("Error parsing tag  %v", err)
	}

	ri, err := remote.Image(ref)
	if err != nil {
		t.Fatalf("Error reading image %v", err)
	}

	b := &bytes.Buffer{}
	if err := tarball.Write(ref, ri, b); err != nil {
		t.Fatalf("Error writing image to tarball %v", err)
	}
}
