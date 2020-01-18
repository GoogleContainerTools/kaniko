/*
Copyright 2019 Google LLC

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

package cache

import (
	"bytes"
	"testing"

	"github.com/GoogleContainerTools/kaniko/pkg/config"
	"github.com/GoogleContainerTools/kaniko/pkg/fakes"
	"github.com/google/go-containerregistry/pkg/name"
	v1 "github.com/google/go-containerregistry/pkg/v1"
	"github.com/google/go-containerregistry/pkg/v1/remote"
)

const (
	image = "foo:latest"
)

func Test_Warmer_Warm_not_in_cache(t *testing.T) {
	tarBuf := new(bytes.Buffer)
	manifestBuf := new(bytes.Buffer)

	cw := &Warmer{
		Remote: func(_ name.Reference, _ ...remote.Option) (v1.Image, error) {
			return fakes.FakeImage{}, nil
		},
		Local: func(_ *config.CacheOptions, _ string) (v1.Image, error) {
			return nil, NotFoundErr{}
		},
		TarWriter:      tarBuf,
		ManifestWriter: manifestBuf,
	}

	opts := &config.WarmerOptions{}

	_, err := cw.Warm(image, opts)
	if err != nil {
		t.Errorf("expected error to be nil but was %v", err)
		t.FailNow()
	}

	if len(tarBuf.Bytes()) == 0 {
		t.Error("expected image to be written but buffer was empty")
	}
}

func Test_Warmer_Warm_in_cache_not_expired(t *testing.T) {
	tarBuf := new(bytes.Buffer)
	manifestBuf := new(bytes.Buffer)

	cw := &Warmer{
		Remote: func(_ name.Reference, _ ...remote.Option) (v1.Image, error) {
			return fakes.FakeImage{}, nil
		},
		Local: func(_ *config.CacheOptions, _ string) (v1.Image, error) {
			return fakes.FakeImage{}, nil
		},
		TarWriter:      tarBuf,
		ManifestWriter: manifestBuf,
	}

	opts := &config.WarmerOptions{}

	_, err := cw.Warm(image, opts)
	if !IsAlreadyCached(err) {
		t.Errorf("expected error to be already cached err but was %v", err)
		t.FailNow()
	}

	if len(tarBuf.Bytes()) != 0 {
		t.Errorf("expected nothing to be written")
	}
}

func Test_Warmer_Warm_in_cache_expired(t *testing.T) {
	tarBuf := new(bytes.Buffer)
	manifestBuf := new(bytes.Buffer)

	cw := &Warmer{
		Remote: func(_ name.Reference, _ ...remote.Option) (v1.Image, error) {
			return fakes.FakeImage{}, nil
		},
		Local: func(_ *config.CacheOptions, _ string) (v1.Image, error) {
			return fakes.FakeImage{}, ExpiredErr{}
		},
		TarWriter:      tarBuf,
		ManifestWriter: manifestBuf,
	}

	opts := &config.WarmerOptions{}

	_, err := cw.Warm(image, opts)
	if !IsAlreadyCached(err) {
		t.Errorf("expected error to be already cached err but was %v", err)
		t.FailNow()
	}

	if len(tarBuf.Bytes()) != 0 {
		t.Errorf("expected nothing to be written")
	}
}
