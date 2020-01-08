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

package gcrane

import (
	"testing"
	"time"

	"github.com/google/go-cmp/cmp"
	"github.com/google/go-containerregistry/pkg/name"
	"github.com/google/go-containerregistry/pkg/v1/google"
	"github.com/google/go-containerregistry/pkg/v1/types"
)

func mustRepo(s string) name.Repository {
	repo, err := name.NewRepository(s, name.WeakValidation)
	if err != nil {
		panic(err)
	}
	return repo
}

func TestRename(t *testing.T) {
	c := copier{
		srcRepo: mustRepo("gcr.io/foo"),
		dstRepo: mustRepo("gcr.io/bar"),
	}

	got, err := c.rename(mustRepo("gcr.io/foo/sub/repo"))
	if err != nil {
		t.Fatalf("unexpected err: %v", err)
	}
	want := mustRepo("gcr.io/bar/sub/repo")

	if want.String() != got.String() {
		t.Errorf("%s != %s", want, got)
	}
}

func TestSubtractStringLists(t *testing.T) {
	cases := []struct {
		minuend    []string
		subtrahend []string
		result     []string
	}{{
		minuend:    []string{"a", "b", "c"},
		subtrahend: []string{"a"},
		result:     []string{"b", "c"},
	}, {
		minuend:    []string{"a", "a", "a"},
		subtrahend: []string{"a", "b"},
		result:     []string{},
	}, {
		minuend:    []string{},
		subtrahend: []string{"a", "b"},
		result:     []string{},
	}, {
		minuend:    []string{"a", "b"},
		subtrahend: []string{},
		result:     []string{"a", "b"},
	}}

	for _, tc := range cases {
		want, got := tc.result, subtractStringLists(tc.minuend, tc.subtrahend)
		if diff := cmp.Diff(want, got); diff != "" {
			t.Errorf("subtracting string lists: %v - %v: (-want +got)\n%s", tc.minuend, tc.subtrahend, diff)
		}
	}
}

func TestDiffImages(t *testing.T) {
	cases := []struct {
		want map[string]google.ManifestInfo
		have map[string]google.ManifestInfo
		need map[string]google.ManifestInfo
	}{{
		// Have everything we need.
		want: map[string]google.ManifestInfo{
			"a": {
				Tags: []string{"b", "c"},
			},
		},
		have: map[string]google.ManifestInfo{
			"a": {
				Tags: []string{"b", "c"},
			},
		},
		need: map[string]google.ManifestInfo{},
	}, {
		// Missing image a.
		want: map[string]google.ManifestInfo{
			"a": {
				Tags: []string{"b", "c", "d"},
			},
		},
		have: map[string]google.ManifestInfo{},
		need: map[string]google.ManifestInfo{
			"a": {
				Tags: []string{"b", "c", "d"},
			},
		},
	}, {
		// Missing tags "b" and "d"
		want: map[string]google.ManifestInfo{
			"a": {
				Tags: []string{"b", "c", "d"},
			},
		},
		have: map[string]google.ManifestInfo{
			"a": {
				Tags: []string{"c"},
			},
		},
		need: map[string]google.ManifestInfo{
			"a": {
				Tags: []string{"b", "d"},
			},
		},
	}, {
		// Make sure all properties get copied over.
		want: map[string]google.ManifestInfo{
			"a": {
				Size:      123,
				MediaType: string(types.DockerManifestSchema2),
				Created:   time.Date(1992, time.January, 7, 6, 40, 00, 5e8, time.UTC),
				Uploaded:  time.Date(2018, time.November, 29, 4, 13, 30, 5e8, time.UTC),
				Tags:      []string{"b", "c", "d"},
			},
		},
		have: map[string]google.ManifestInfo{},
		need: map[string]google.ManifestInfo{
			"a": {
				Size:      123,
				MediaType: string(types.DockerManifestSchema2),
				Created:   time.Date(1992, time.January, 7, 6, 40, 00, 5e8, time.UTC),
				Uploaded:  time.Date(2018, time.November, 29, 4, 13, 30, 5e8, time.UTC),
				Tags:      []string{"b", "c", "d"},
			},
		},
	}}

	for _, tc := range cases {
		want, got := tc.need, diffImages(tc.want, tc.have)
		if diff := cmp.Diff(want, got); diff != "" {
			t.Errorf("diffing images: %v - %v: (-want +got)\n%s", tc.want, tc.have, diff)
		}
	}
}
