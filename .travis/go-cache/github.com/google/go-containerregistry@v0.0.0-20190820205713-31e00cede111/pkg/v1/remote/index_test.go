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

package remote

import (
	"bytes"
	"fmt"
	"net/http"
	"net/http/httptest"
	"net/url"
	"testing"

	"github.com/google/go-cmp/cmp"
	v1 "github.com/google/go-containerregistry/pkg/v1"
	"github.com/google/go-containerregistry/pkg/v1/random"
	"github.com/google/go-containerregistry/pkg/v1/types"
)

func randomIndex(t *testing.T) v1.ImageIndex {
	rnd, err := random.Index(1024, 1, 3)
	if err != nil {
		t.Fatalf("random.Index() = %v", err)
	}
	return rnd
}

func mustIndexManifest(t *testing.T, idx v1.ImageIndex) *v1.IndexManifest {
	m, err := idx.IndexManifest()
	if err != nil {
		t.Fatalf("IndexManifest() = %v", err)
	}
	return m
}

func mustChild(t *testing.T, idx v1.ImageIndex, h v1.Hash) v1.Image {
	img, err := idx.Image(h)
	if err != nil {
		t.Fatalf("Image(%s) = %v", h, err)
	}
	return img
}

func mustMediaType(t *testing.T, man manifest) types.MediaType {
	mt, err := man.MediaType()
	if err != nil {
		t.Fatalf("MediaType() = %v", err)
	}
	return mt
}

func mustHash(t *testing.T, s string) v1.Hash {
	h, err := v1.NewHash(s)
	if err != nil {
		t.Fatalf("NewHash() = %v", err)
	}
	return h
}

func TestIndexRawManifestDigests(t *testing.T) {
	idx := randomIndex(t)
	expectedRepo := "foo/bar"

	cases := []struct {
		name          string
		ref           string
		responseBody  []byte
		contentDigest string
		wantErr       bool
	}{{
		name:          "normal pull, by tag",
		ref:           "latest",
		responseBody:  mustRawManifest(t, idx),
		contentDigest: mustDigest(t, idx).String(),
		wantErr:       false,
	}, {
		name:          "normal pull, by digest",
		ref:           mustDigest(t, idx).String(),
		responseBody:  mustRawManifest(t, idx),
		contentDigest: mustDigest(t, idx).String(),
		wantErr:       false,
	}, {
		name:          "right content-digest, wrong body, by digest",
		ref:           mustDigest(t, idx).String(),
		responseBody:  []byte("not even json"),
		contentDigest: mustDigest(t, idx).String(),
		wantErr:       true,
	}, {
		name:          "right body, wrong content-digest, by tag",
		ref:           "latest",
		responseBody:  mustRawManifest(t, idx),
		contentDigest: bogusDigest,
		wantErr:       false,
	}, {
		// NB: This succeeds! We don't care what the registry thinks.
		name:          "right body, wrong content-digest, by digest",
		ref:           mustDigest(t, idx).String(),
		responseBody:  mustRawManifest(t, idx),
		contentDigest: bogusDigest,
		wantErr:       false,
	}}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			manifestPath := fmt.Sprintf("/v2/%s/manifests/%s", expectedRepo, tc.ref)
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				switch r.URL.Path {
				case manifestPath:
					if r.Method != http.MethodGet {
						t.Errorf("Method; got %v, want %v", r.Method, http.MethodGet)
					}

					w.Header().Set("Docker-Content-Digest", tc.contentDigest)
					w.Write(tc.responseBody)
				default:
					t.Fatalf("Unexpected path: %v", r.URL.Path)
				}
			}))
			defer server.Close()
			u, err := url.Parse(server.URL)
			if err != nil {
				t.Fatalf("url.Parse(%v) = %v", server.URL, err)
			}

			ref, err := newReference(u.Host, expectedRepo, tc.ref)
			if err != nil {
				t.Fatalf("url.Parse(%v, %v, %v) = %v", u.Host, expectedRepo, tc.ref, err)
			}

			rmt := remoteIndex{
				fetcher: fetcher{
					Ref:    ref,
					Client: http.DefaultClient,
				},
			}

			if _, err := rmt.RawManifest(); (err != nil) != tc.wantErr {
				t.Errorf("RawManifest() wrong error: %v, want %v: %v\n", (err != nil), tc.wantErr, err)
			}
		})
	}
}

func TestIndex(t *testing.T) {
	idx := randomIndex(t)
	expectedRepo := "foo/bar"
	manifestPath := fmt.Sprintf("/v2/%s/manifests/latest", expectedRepo)
	childDigest := mustIndexManifest(t, idx).Manifests[0].Digest
	child := mustChild(t, idx, childDigest)
	childPath := fmt.Sprintf("/v2/%s/manifests/%s", expectedRepo, childDigest)
	configPath := fmt.Sprintf("/v2/%s/blobs/%s", expectedRepo, mustConfigName(t, child))
	manifestReqCount := 0
	childReqCount := 0

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/v2/":
			w.WriteHeader(http.StatusOK)
		case manifestPath:
			manifestReqCount++
			if r.Method != http.MethodGet {
				t.Errorf("Method; got %v, want %v", r.Method, http.MethodGet)
			}
			w.Header().Set("Content-Type", string(mustMediaType(t, idx)))
			w.Write(mustRawManifest(t, idx))
		case childPath:
			childReqCount++
			if r.Method != http.MethodGet {
				t.Errorf("Method; got %v, want %v", r.Method, http.MethodGet)
			}
			w.Write(mustRawManifest(t, child))
		case configPath:
			if r.Method != http.MethodGet {
				t.Errorf("Method; got %v, want %v", r.Method, http.MethodGet)
			}
			w.Write(mustRawConfigFile(t, child))
		default:
			t.Fatalf("Unexpected path: %v", r.URL.Path)
		}
	}))
	defer server.Close()
	u, err := url.Parse(server.URL)
	if err != nil {
		t.Fatalf("url.Parse(%v) = %v", server.URL, err)
	}

	tag := mustNewTag(t, fmt.Sprintf("%s/%s:latest", u.Host, expectedRepo))
	rmt, err := Index(tag, WithTransport(http.DefaultTransport))
	if err != nil {
		t.Errorf("Index() = %v", err)
	}
	rmtChild, err := rmt.Image(childDigest)
	if err != nil {
		t.Errorf("remoteIndex.Image(%s) = %v", childDigest, err)
	}

	// Test that index works as expected.
	if got, want := mustRawManifest(t, rmt), mustRawManifest(t, idx); bytes.Compare(got, want) != 0 {
		t.Errorf("RawManifest() = %v, want %v", got, want)
	}
	if diff := cmp.Diff(mustIndexManifest(t, idx), mustIndexManifest(t, rmt)); diff != "" {
		t.Errorf("IndexManifest() (-want +got) = %v", diff)
	}
	if got, want := mustMediaType(t, rmt), mustMediaType(t, idx); got != want {
		t.Errorf("MediaType() = %v, want %v", got, want)
	}
	if got, want := mustDigest(t, rmt), mustDigest(t, idx); got != want {
		t.Errorf("Digest() = %v, want %v", got, want)
	}
	// Make sure caching the manifest works for index.
	if manifestReqCount != 1 {
		t.Errorf("RawManifest made %v requests, expected 1", manifestReqCount)
	}

	// Test that child works as expected.
	if got, want := mustRawManifest(t, rmtChild), mustRawManifest(t, child); bytes.Compare(got, want) != 0 {
		t.Errorf("RawManifest() = %v, want %v", got, want)
	}
	if got, want := mustRawConfigFile(t, rmtChild), mustRawConfigFile(t, child); bytes.Compare(got, want) != 0 {
		t.Errorf("RawConfigFile() = %v, want %v", got, want)
	}
	// Make sure caching the manifest works for child.
	if childReqCount != 1 {
		t.Errorf("RawManifest made %v requests, expected 1", childReqCount)
	}

	// Try to fetch bogus children.
	bogusHash := mustHash(t, bogusDigest)

	if _, err := rmt.Image(bogusHash); err == nil {
		t.Errorf("remoteIndex.Image(bogusDigest) err = %v, wanted err", err)
	}
	if _, err := rmt.ImageIndex(bogusHash); err == nil {
		t.Errorf("remoteIndex.ImageIndex(bogusDigest) err = %v, wanted err", err)
	}
}

// TestMatchesPlatform runs test cases on the matchesPlatform function which verifies
// whether the given platform can run on the required platform by checking the
// compatibility of architecture, OS, OS version, OS features, variant and features.
func TestMatchesPlatform(t *testing.T) {
	t.Parallel()
	tests := []struct {
		// want is the expected return value from matchesPlatform
		// when the given platform is 'given' and the required platform is 'required'.
		given    v1.Platform
		required v1.Platform
		want     bool
	}{{ // The given & required platforms are identical. matchesPlatform expected to return true.
		given: v1.Platform{
			Architecture: "amd64",
			OS:           "linux",
			OSVersion:    "10.0.10586",
			OSFeatures:   []string{"win32k"},
			Variant:      "armv6l",
			Features:     []string{"sse4"},
		},
		required: v1.Platform{
			Architecture: "amd64",
			OS:           "linux",
			OSVersion:    "10.0.10586",
			OSFeatures:   []string{"win32k"},
			Variant:      "armv6l",
			Features:     []string{"sse4"},
		},
		want: true,
	},
		{ // OS and Architecture must exactly match. matchesPlatform expected to return false.
			given: v1.Platform{
				Architecture: "arm",
				OS:           "linux",
				OSVersion:    "10.0.10586",
				OSFeatures:   []string{"win64k"},
				Variant:      "armv6l",
				Features:     []string{"sse4"},
			},
			required: v1.Platform{
				Architecture: "amd64",
				OS:           "linux",
				OSVersion:    "10.0.10586",
				OSFeatures:   []string{"win32k"},
				Variant:      "armv6l",
				Features:     []string{"sse4"},
			},
			want: false,
		},
		{ // OS version must exactly match
			given: v1.Platform{
				Architecture: "amd64",
				OS:           "linux",
				OSVersion:    "10.0.10586",
				OSFeatures:   []string{"win64k"},
				Variant:      "armv6l",
				Features:     []string{"sse4"},
			},
			required: v1.Platform{
				Architecture: "amd64",
				OS:           "linux",
				OSVersion:    "10.0.10587",
				OSFeatures:   []string{"win64k"},
				Variant:      "armv6l",
				Features:     []string{"sse4"},
			},
			want: false,
		},
		{ // OS Features must exactly match. matchesPlatform expected to return false.
			given: v1.Platform{
				Architecture: "arm",
				OS:           "linux",
				OSVersion:    "10.0.10586",
				OSFeatures:   []string{"win64k"},
				Variant:      "armv6l",
				Features:     []string{"sse4"},
			},
			required: v1.Platform{
				Architecture: "arm",
				OS:           "linux",
				OSVersion:    "10.0.10586",
				OSFeatures:   []string{"win32k"},
				Variant:      "armv6l",
				Features:     []string{"sse4"},
			},
			want: false,
		},
		{ // Variant must exactly match. matchesPlatform expected to return false.
			given: v1.Platform{
				Architecture: "amd64",
				OS:           "linux",
				OSVersion:    "10.0.10586",
				OSFeatures:   []string{"win64k"},
				Variant:      "armv6l",
				Features:     []string{"sse4"},
			},
			required: v1.Platform{
				Architecture: "amd64",
				OS:           "linux",
				OSVersion:    "10.0.10586",
				OSFeatures:   []string{"win64k"},
				Variant:      "armv7l",
				Features:     []string{"sse4"},
			},
			want: false,
		},
		{ // OS must exactly match, and is case sensative. matchesPlatform expected to return false.
			given: v1.Platform{
				Architecture: "arm",
				OS:           "linux",
				OSVersion:    "10.0.10586",
				OSFeatures:   []string{"win64k"},
				Variant:      "armv6l",
				Features:     []string{"sse4"},
			},
			required: v1.Platform{
				Architecture: "arm",
				OS:           "LinuX",
				OSVersion:    "10.0.10586",
				OSFeatures:   []string{"win64k"},
				Variant:      "armv6l",
				Features:     []string{"sse4"},
			},
			want: false,
		},
		{ // OSVersion and Variant are specified in given but not in required.
			// matchesPlatform expected to return true.
			given: v1.Platform{
				Architecture: "arm",
				OS:           "linux",
				OSVersion:    "10.0.10586",
				OSFeatures:   []string{"win64k"},
				Variant:      "armv6l",
				Features:     []string{"sse4"},
			},
			required: v1.Platform{
				Architecture: "arm",
				OS:           "linux",
				OSVersion:    "",
				OSFeatures:   []string{"win64k"},
				Variant:      "",
				Features:     []string{"sse4"},
			},
			want: true,
		},
		{ // Ensure the optional field OSVersion & Variant match exactly if specified as required.
			given: v1.Platform{
				Architecture: "amd64",
				OS:           "linux",
				OSVersion:    "",
				OSFeatures:   []string{},
				Variant:      "",
				Features:     []string{},
			},
			required: v1.Platform{
				Architecture: "amd64",
				OS:           "linux",
				OSVersion:    "10.0.10586",
				OSFeatures:   []string{"win32k"},
				Variant:      "armv6l",
				Features:     []string{"sse4"},
			},
			want: false,
		},
		{ // Checking subset validity when required less features than given features.
			// matchesPlatform expected to return true.
			given: v1.Platform{
				Architecture: "",
				OS:           "linux",
				OSVersion:    "10.0.10586",
				OSFeatures:   []string{"win32k"},
				Variant:      "armv6l",
				Features:     []string{"sse4"},
			},
			required: v1.Platform{
				Architecture: "",
				OS:           "linux",
				OSVersion:    "",
				OSFeatures:   []string{},
				Variant:      "",
				Features:     []string{},
			},
			want: true,
		},
		{ // Checking subset validity when required features are subset of given features.
			// matchesPlatform expected to return true.
			given: v1.Platform{
				Architecture: "arm",
				OS:           "linux",
				OSVersion:    "10.0.10586",
				OSFeatures:   []string{"win64k", "f1", "f2"},
				Variant:      "",
				Features:     []string{"sse4", "f1"},
			},
			required: v1.Platform{
				Architecture: "arm",
				OS:           "linux",
				OSVersion:    "10.0.10586",
				OSFeatures:   []string{"win64k"},
				Variant:      "",
				Features:     []string{"sse4"},
			},
			want: true,
		},
		{ // Checking subset validity when some required features is not subset of given features.
			// matchesPlatform expected to return false.
			given: v1.Platform{
				Architecture: "arm",
				OS:           "linux",
				OSVersion:    "10.0.10586",
				OSFeatures:   []string{"win64k", "f1", "f2"},
				Variant:      "",
				Features:     []string{"sse4", "f1"},
			},
			required: v1.Platform{
				Architecture: "arm",
				OS:           "linux",
				OSVersion:    "10.0.10586",
				OSFeatures:   []string{"win64k"},
				Variant:      "",
				Features:     []string{"sse4", "f2"},
			},
			want: false,
		},
		{ // Checking subset validity when OS features not required,
			// and required features is indeed a subset of given features.
			// matchesPlatform expected to return true.
			given: v1.Platform{
				Architecture: "arm",
				OS:           "linux",
				OSVersion:    "10.0.10586",
				OSFeatures:   []string{"win64k", "f1", "f2"},
				Variant:      "armv6l",
				Features:     []string{"sse4"},
			},
			required: v1.Platform{
				Architecture: "arm",
				OS:           "linux",
				OSVersion:    "10.0.10586",
				OSFeatures:   []string{},
				Variant:      "armv6l",
				Features:     []string{"sse4"},
			},
			want: true,
		},
	}

	for _, test := range tests {
		got := matchesPlatform(test.given, test.required)
		if got != test.want {
			t.Errorf("matchesPlatform(%v, %v); got %v, want %v", test.given, test.required, got, test.want)
		}
	}

}
