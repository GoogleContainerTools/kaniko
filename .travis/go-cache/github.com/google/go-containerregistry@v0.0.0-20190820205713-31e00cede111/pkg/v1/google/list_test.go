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

package google

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"net/url"
	"testing"
	"time"

	"github.com/google/go-cmp/cmp"
	"github.com/google/go-containerregistry/pkg/name"
)

func mustParseDuration(t *testing.T, d string) time.Duration {
	dur, err := time.ParseDuration(d)
	if err != nil {
		t.Fatal(err)
	}
	return dur
}

func TestList(t *testing.T) {
	cases := []struct {
		name         string
		responseBody []byte
		wantErr      bool
		wantTags     *Tags
	}{{
		name:         "success",
		responseBody: []byte(`{"tags":["foo","bar"]}`),
		wantErr:      false,
		wantTags:     &Tags{Tags: []string{"foo", "bar"}},
	}, {
		name:         "gcr success",
		responseBody: []byte(`{"child":["hello", "world"],"manifest":{"digest1":{"imageSizeBytes":"1","mediaType":"mainstream","timeCreatedms":"1","timeUploadedMs":"2","tag":["foo"]},"digest2":{"imageSizeBytes":"2","mediaType":"indie","timeCreatedMs":"3","timeUploadedMs":"4","tag":["bar","baz"]}},"tags":["foo","bar","baz"]}`),
		wantErr:      false,
		wantTags: &Tags{
			Children: []string{"hello", "world"},
			Manifests: map[string]ManifestInfo{
				"digest1": {
					Size:      1,
					MediaType: "mainstream",
					Created:   time.Unix(0, 0).Add(mustParseDuration(t, "1ms")),
					Uploaded:  time.Unix(0, 0).Add(mustParseDuration(t, "2ms")),
					Tags:      []string{"foo"},
				},
				"digest2": {
					Size:      2,
					MediaType: "indie",
					Created:   time.Unix(0, 0).Add(mustParseDuration(t, "3ms")),
					Uploaded:  time.Unix(0, 0).Add(mustParseDuration(t, "4ms")),
					Tags:      []string{"bar", "baz"},
				},
			},
			Tags: []string{"foo", "bar", "baz"},
		},
	}, {
		name:         "not json",
		responseBody: []byte("notjson"),
		wantErr:      true,
	}}

	repoName := "ubuntu"

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			tagsPath := fmt.Sprintf("/v2/%s/tags/list", repoName)
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				switch r.URL.Path {
				case "/v2/":
					w.WriteHeader(http.StatusOK)
				case tagsPath:
					if r.Method != http.MethodGet {
						t.Errorf("Method; got %v, want %v", r.Method, http.MethodGet)
					}

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

			repo, err := name.NewRepository(fmt.Sprintf("%s/%s", u.Host, repoName), name.WeakValidation)
			if err != nil {
				t.Fatalf("name.NewRepository(%v) = %v", repoName, err)
			}

			tags, err := List(repo)
			if (err != nil) != tc.wantErr {
				t.Errorf("List() wrong error: %v, want %v: %v\n", (err != nil), tc.wantErr, err)
			}

			if diff := cmp.Diff(tc.wantTags, tags); diff != "" {
				t.Errorf("List() wrong tags (-want +got) = %s", diff)
			}
		})
	}
}

type recorder struct {
	Tags []*Tags
	Errs []error
}

func (r *recorder) walk(repo name.Repository, tags *Tags, err error) error {
	r.Tags = append(r.Tags, tags)
	r.Errs = append(r.Errs, err)

	return nil
}

func TestWalk(t *testing.T) {
	cases := []struct {
		name         string
		responseBody []byte
		wantResult   recorder
	}{{
		name:         "gcr success",
		responseBody: []byte(`{"child":["hello", "world"],"manifest":{"digest1":{"imageSizeBytes":"1","mediaType":"mainstream","timeCreatedms":"1","timeUploadedMs":"2","tag":["foo"]},"digest2":{"imageSizeBytes":"2","mediaType":"indie","timeCreatedMs":"3","timeUploadedMs":"4","tag":["bar","baz"]}},"tags":["foo","bar","baz"]}`),
		wantResult: recorder{
			Tags: []*Tags{{
				Children: []string{"hello", "world"},
				Manifests: map[string]ManifestInfo{
					"digest1": {
						Size:      1,
						MediaType: "mainstream",
						Created:   time.Unix(0, 0).Add(mustParseDuration(t, "1ms")),
						Uploaded:  time.Unix(0, 0).Add(mustParseDuration(t, "2ms")),
						Tags:      []string{"foo"},
					},
					"digest2": {
						Size:      2,
						MediaType: "indie",
						Created:   time.Unix(0, 0).Add(mustParseDuration(t, "3ms")),
						Uploaded:  time.Unix(0, 0).Add(mustParseDuration(t, "4ms")),
						Tags:      []string{"bar", "baz"},
					},
				},
				Tags: []string{"foo", "bar", "baz"},
			}, {
				Tags: []string{"hello"},
			}, {
				Tags: []string{"world"},
			}},
			Errs: []error{nil, nil, nil},
		},
	}}

	repoName := "ubuntu"

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			rootPath := fmt.Sprintf("/v2/%s/tags/list", repoName)
			helloPath := fmt.Sprintf("/v2/%s/hello/tags/list", repoName)
			worldPath := fmt.Sprintf("/v2/%s/world/tags/list", repoName)
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				switch r.URL.Path {
				case "/v2/":
					w.WriteHeader(http.StatusOK)
				case rootPath:
					if r.Method != http.MethodGet {
						t.Errorf("Method; got %v, want %v", r.Method, http.MethodGet)
					}

					w.Write(tc.responseBody)
				case helloPath:
					w.Write([]byte(`{"tags":["hello"]}`))
				case worldPath:
					w.Write([]byte(`{"tags":["world"]}`))
				default:
					t.Fatalf("Unexpected path: %v", r.URL.Path)
				}
			}))
			defer server.Close()
			u, err := url.Parse(server.URL)
			if err != nil {
				t.Fatalf("url.Parse(%v) = %v", server.URL, err)
			}

			repo, err := name.NewRepository(fmt.Sprintf("%s/%s", u.Host, repoName), name.WeakValidation)
			if err != nil {
				t.Fatalf("name.NewRepository(%v) = %v", repoName, err)
			}

			r := recorder{}
			err = Walk(repo, r.walk)

			if diff := cmp.Diff(tc.wantResult, r); diff != "" {
				t.Errorf("Walk() wrong tags (-want +got) = %s", diff)
			}
		})
	}
}
