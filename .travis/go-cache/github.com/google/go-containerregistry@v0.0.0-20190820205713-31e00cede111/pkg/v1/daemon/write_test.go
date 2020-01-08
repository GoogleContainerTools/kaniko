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
	"io/ioutil"
	"strings"
	"testing"

	"github.com/docker/docker/api/types"

	"github.com/google/go-containerregistry/pkg/name"
	"github.com/google/go-containerregistry/pkg/v1/tarball"
)

type MockImageLoader struct{}

func (m *MockImageLoader) ImageLoad(context.Context, io.Reader, bool) (types.ImageLoadResponse, error) {
	return types.ImageLoadResponse{
		Body: ioutil.NopCloser(strings.NewReader("Loaded")),
	}, nil
}

func (m *MockImageLoader) ImageTag(ctx context.Context, source, target string) error {
	return nil
}

func init() {
	GetImageLoader = func() (ImageLoader, error) {
		return &MockImageLoader{}, nil
	}
}

func TestWriteImage(t *testing.T) {
	image, err := tarball.ImageFromPath("../tarball/testdata/test_image_1.tar", nil)
	if err != nil {
		t.Errorf("Error loading image: %v", err.Error())
	}
	tag, err := name.NewTag("test_image_2:latest", name.WeakValidation)
	if err != nil {
		t.Errorf(err.Error())
	}
	response, err := Write(tag, image)
	if err != nil {
		t.Errorf("Error writing image tar: %s", err.Error())
	}
	if !strings.Contains(response, "Loaded") {
		t.Errorf("Error loading image. Response: %s", response)
	}
}
