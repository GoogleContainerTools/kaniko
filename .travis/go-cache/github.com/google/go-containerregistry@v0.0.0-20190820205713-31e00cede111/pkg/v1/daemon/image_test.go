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
	"os"
	"reflect"
	"testing"

	"github.com/google/go-containerregistry/pkg/name"

	"github.com/google/go-containerregistry/pkg/v1/tarball"
)

var imagePath = "../tarball/testdata/test_image_1.tar"

type MockImageSaver struct {
	path string
}

func (m *MockImageSaver) ImageSave(_ context.Context, _ []string) (io.ReadCloser, error) {
	return os.Open(m.path)
}

func init() {
	getImageSaver = func() (ImageSaver, error) {
		return &MockImageSaver{path: imagePath}, nil
	}
}

func TestImage(t *testing.T) {
	testImage, err := tarball.ImageFromPath(imagePath, nil)
	if err != nil {
		t.Fatalf("error loading test image: %s", err)
	}

	tag, err := name.NewTag("unused", name.WeakValidation)
	if err != nil {
		t.Fatalf("error creating test name: %s", err)
	}

	runTest := func(buffered bool) {
		var bufferedOption ImageOption
		if buffered {
			bufferedOption = WithBufferedOpener()
		} else {
			bufferedOption = WithUnbufferedOpener()
		}
		daemonImage, err := Image(tag, bufferedOption)
		if err != nil {
			t.Errorf("Error loading daemon image: %s", err)
		}

		dmfst, err := daemonImage.Manifest()
		if err != nil {
			t.Errorf("Error getting daemon manifest: %s", err)
		}
		tmfst, err := testImage.Manifest()
		if err != nil {
			t.Errorf("Error getting test manifest: %s", err)
		}
		if !reflect.DeepEqual(dmfst, tmfst) {
			t.Errorf("%v != %v", testImage, daemonImage)
		}
	}

	runTest(false)
	runTest(true)

}
