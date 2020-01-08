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

package transport

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"

	"github.com/google/go-containerregistry/pkg/authn"
	"github.com/google/go-containerregistry/pkg/name"
)

func TestBearerRefresh(t *testing.T) {
	expectedToken := "Sup3rDup3rS3cr3tz"
	expectedScope := "this-is-your-scope"
	expectedService := "my-service.io"

	cases := []struct {
		tokenKey string
		wantErr  bool
	}{{
		tokenKey: "token",
		wantErr:  false,
	}, {
		tokenKey: "access_token",
		wantErr:  false,
	}, {
		tokenKey: "tolkien",
		wantErr:  true,
	}}

	for _, tc := range cases {
		t.Run(tc.tokenKey, func(t *testing.T) {
			server := httptest.NewServer(
				http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
					hdr := r.Header.Get("Authorization")
					if !strings.HasPrefix(hdr, "Basic ") {
						t.Errorf("Header.Get(Authorization); got %v, want Basic prefix", hdr)
					}
					if got, want := r.FormValue("scope"), expectedScope; got != want {
						t.Errorf("FormValue(scope); got %v, want %v", got, want)
					}
					if got, want := r.FormValue("service"), expectedService; got != want {
						t.Errorf("FormValue(service); got %v, want %v", got, want)
					}
					w.Write([]byte(fmt.Sprintf(`{%q: %q}`, tc.tokenKey, expectedToken)))
				}))
			defer server.Close()

			basic := &authn.Basic{Username: "foo", Password: "bar"}
			registry, err := name.NewRegistry(expectedService, name.WeakValidation)
			if err != nil {
				t.Errorf("Unexpected error during NewRegistry: %v", err)
			}

			bt := &bearerTransport{
				inner:    http.DefaultTransport,
				basic:    basic,
				registry: registry,
				realm:    server.URL,
				scopes:   []string{expectedScope},
				service:  expectedService,
				scheme:   "http",
			}

			if err := bt.refresh(); (err != nil) != tc.wantErr {
				t.Errorf("refresh() = %v", err)
			}
		})
	}
}

func TestBearerTransport(t *testing.T) {
	expectedToken := "sdkjhfskjdhfkjshdf"

	blobServer := httptest.NewServer(
		http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// We don't expect the blobServer to receive bearer tokens.
			if got := r.Header.Get("Authorization"); got != "" {
				t.Errorf("Header.Get(Authorization); got %v, want empty string", got)
			}
			w.WriteHeader(http.StatusOK)
		}))
	defer blobServer.Close()

	server := httptest.NewServer(
		http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if got, want := r.Header.Get("Authorization"), "Bearer "+expectedToken; got != want {
				t.Errorf("Header.Get(Authorization); got %v, want %v", got, want)
			}
			if r.URL.Path == "/v2/auth" {
				http.Redirect(w, r, "/redirect", http.StatusMovedPermanently)
				return
			}
			if strings.Contains(r.URL.Path, "blobs") {
				http.Redirect(w, r, blobServer.URL, http.StatusFound)
				return
			}
			w.WriteHeader(http.StatusOK)
		}))
	defer server.Close()

	u, err := url.Parse(server.URL)
	if err != nil {
		t.Errorf("Unexpected error during url.Parse: %v", err)
	}
	registry, err := name.NewRegistry(u.Host, name.WeakValidation)
	if err != nil {
		t.Errorf("Unexpected error during NewRegistry: %v", err)
	}

	bearer := &authn.Bearer{Token: expectedToken}
	client := http.Client{Transport: &bearerTransport{
		inner:    &http.Transport{},
		bearer:   bearer,
		registry: registry,
		scheme:   "http",
	}}

	_, err = client.Get(fmt.Sprintf("http://%s/v2/auth", u.Host))
	if err != nil {
		t.Errorf("Unexpected error during Get: %v", err)
	}

	_, err = client.Get(fmt.Sprintf("http://%s/v2/foo/bar/blobs/blah", u.Host))
	if err != nil {
		t.Errorf("Unexpected error during Get: %v", err)
	}
}

func TestBearerTransportTokenRefresh(t *testing.T) {
	initialToken := "foo"
	refreshedToken := "bar"

	server := httptest.NewServer(
		http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			hdr := r.Header.Get("Authorization")
			if hdr == "Bearer "+refreshedToken {
				w.WriteHeader(http.StatusOK)
				return
			}
			if strings.HasPrefix(hdr, "Basic ") {
				w.Write([]byte(fmt.Sprintf(`{"token": %q}`, refreshedToken)))
			}

			w.WriteHeader(http.StatusUnauthorized)
			return
		}))
	defer server.Close()

	u, err := url.Parse(server.URL)
	registry, err := name.NewRegistry(u.Host, name.WeakValidation)
	if err != nil {
		t.Errorf("Unexpected error during NewRegistry: %v", err)
	}

	bearer := &authn.Bearer{Token: initialToken}
	transport := &bearerTransport{
		inner:    http.DefaultTransport,
		bearer:   bearer,
		basic:    &authn.Basic{},
		registry: registry,
		realm:    server.URL,
		scheme:   "http",
	}
	client := http.Client{Transport: transport}

	res, err := client.Get(fmt.Sprintf("http://%s/v2/foo/bar/blobs/blah", u.Host))
	if err != nil {
		t.Errorf("Unexpected error during client.Get: %v", err)
	}
	if res.StatusCode != http.StatusOK {
		t.Errorf("client.Get final StatusCode got %v, want: %v", res.StatusCode, http.StatusOK)
	}
	if transport.bearer.Token != refreshedToken {
		t.Errorf("Expected Bearer token to be refreshed, got %v, want %v", bearer.Token, refreshedToken)
	}
}

type recorder struct {
	reqs []*http.Request
	resp *http.Response
	err  error
}

func newRecorder(resp *http.Response, err error) *recorder {
	return &recorder{
		reqs: []*http.Request{},
		resp: resp,
		err:  err,
	}
}

func (r *recorder) RoundTrip(in *http.Request) (*http.Response, error) {
	r.reqs = append(r.reqs, in)
	return r.resp, r.err
}

func TestSchemeOverride(t *testing.T) {
	// Record the requests we get in the inner transport.
	cannedResponse := http.Response{
		Status:     http.StatusText(http.StatusOK),
		StatusCode: http.StatusOK,
	}
	rec := newRecorder(&cannedResponse, nil)
	registry, err := name.NewRegistry("example.com")
	if err != nil {
		t.Fatalf("Unexpected error during NewRegistry: %v", err)
	}
	bearer := &authn.Bearer{Token: "does-not-matter"}
	bt := &bearerTransport{
		inner:    rec,
		bearer:   bearer,
		basic:    &authn.Basic{},
		registry: registry,
		realm:    "token.example.com",
		scheme:   "http",
	}

	// We should see the scheme be overridden to "http" for the registry, but the
	// scheme for the token server should be unchanged.
	tests := []struct {
		url        string
		wantScheme string
	}{{
		url:        "https://example.com",
		wantScheme: "http",
	}, {
		url:        "https://token.example.com",
		wantScheme: "https",
	}}

	for i, tt := range tests {
		req, err := http.NewRequest("GET", tt.url, nil)
		if err != nil {
			t.Fatalf("Unexpected error during NewRequest: %v", err)
		}

		if _, err := bt.RoundTrip(req); err != nil {
			t.Fatalf("Unexpected error during RoundTrip: %v", err)
		}

		if got, want := rec.reqs[i].URL.Scheme, tt.wantScheme; got != want {
			t.Errorf("Wrong scheme: wanted %v, got %v", want, got)
		}
	}
}

func TestCanonicalAddressResolution(t *testing.T) {
	bearer := &authn.Bearer{Token: "does-not-matter"}

	registry, err := name.NewRegistry("does-not-matter", name.WeakValidation)
	if err != nil {
		t.Errorf("Unexpected error during NewRegistry: %v", err)
	}

	tests := []struct {
		bt      *bearerTransport
		address string
		want    string
	}{{
		bt: &bearerTransport{
			inner:    &http.Transport{},
			bearer:   bearer,
			registry: registry,
			scheme:   "http",
		},
		address: "registry.example.com",
		want:    "registry.example.com:80",
	}, {
		bt: &bearerTransport{
			inner:    &http.Transport{},
			bearer:   bearer,
			registry: registry,
			scheme:   "http",
		},
		address: "registry.example.com:12345",
		want:    "registry.example.com:12345",
	}, {
		bt: &bearerTransport{
			inner:    &http.Transport{},
			bearer:   bearer,
			registry: registry,
			scheme:   "https",
		},
		address: "registry.example.com",
		want:    "registry.example.com:443",
	}, {
		bt: &bearerTransport{
			inner:    &http.Transport{},
			bearer:   bearer,
			registry: registry,
			scheme:   "https",
		},
		address: "registry.example.com:12345",
		want:    "registry.example.com:12345",
	}, {
		bt: &bearerTransport{
			inner:    &http.Transport{},
			bearer:   bearer,
			registry: registry,
			scheme:   "http",
		},
		address: "registry.example.com:",
		want:    "registry.example.com:80",
	}, {
		bt: &bearerTransport{
			inner:    &http.Transport{},
			bearer:   bearer,
			registry: registry,
			scheme:   "https",
		},
		address: "registry.example.com:",
		want:    "registry.example.com:443",
	}, {
		bt: &bearerTransport{
			inner:    &http.Transport{},
			bearer:   bearer,
			registry: registry,
			scheme:   "http",
		},
		address: "2001:db8::1",
		want:    "[2001:db8::1]:80",
	}, {
		bt: &bearerTransport{
			inner:    &http.Transport{},
			bearer:   bearer,
			registry: registry,
			scheme:   "https",
		},
		address: "2001:db8::1",
		want:    "[2001:db8::1]:443",
	}, {
		bt: &bearerTransport{
			inner:    &http.Transport{},
			bearer:   bearer,
			registry: registry,
			scheme:   "http",
		},
		address: "[2001:db8::1]:12345",
		want:    "[2001:db8::1]:12345",
	}, {
		bt: &bearerTransport{
			inner:    &http.Transport{},
			bearer:   bearer,
			registry: registry,
			scheme:   "https",
		},
		address: "[2001:db8::1]:12345",
		want:    "[2001:db8::1]:12345",
	}, {
		bt: &bearerTransport{
			inner:    &http.Transport{},
			bearer:   bearer,
			registry: registry,
			scheme:   "http",
		},
		address: "[2001:db8::1]:",
		want:    "[2001:db8::1]:80",
	}, {
		bt: &bearerTransport{
			inner:    &http.Transport{},
			bearer:   bearer,
			registry: registry,
			scheme:   "https",
		},
		address: "[2001:db8::1]:",
		want:    "[2001:db8::1]:443",
	}}

	for _, tt := range tests {
		got := tt.bt.canonicalAddress(tt.address)
		if got != tt.want {
			t.Errorf("Wrong canonical host: wanted %v got %v", tt.want, got)
		}
	}
}
