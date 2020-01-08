package remote

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"net/url"
	"testing"

	"github.com/google/go-containerregistry/pkg/authn"
	"github.com/google/go-containerregistry/pkg/name"
)

func TestCheckPushPermission(t *testing.T) {
	for _, c := range []struct {
		status  int
		wantErr bool
	}{{
		http.StatusCreated,
		false,
	}, {
		http.StatusForbidden,
		true,
	}, {
		http.StatusBadRequest,
		true,
	}} {

		expectedRepo := "write/time"
		initiatePath := fmt.Sprintf("/v2/%s/blobs/uploads/", expectedRepo)
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			switch r.URL.Path {
			case "/v2/":
				w.WriteHeader(http.StatusOK)
			case initiatePath:
				if r.Method != http.MethodPost {
					t.Errorf("Method; got %v, want %v", r.Method, http.MethodPost)
				}
				w.Header().Set("Location", "somewhere/else")
				http.Error(w, "", c.status)
			default:
				t.Fatalf("Unexpected path: %v", r.URL.Path)
			}
		}))
		defer server.Close()
		u, err := url.Parse(server.URL)
		if err != nil {
			t.Fatalf("url.Parse(%v) = %v", server.URL, err)
		}

		ref := mustNewTag(t, fmt.Sprintf("%s/%s:latest", u.Host, expectedRepo))
		if err := CheckPushPermission(ref, authn.DefaultKeychain, http.DefaultTransport); (err != nil) != c.wantErr {
			t.Errorf("CheckPermission(%d): got error = %v, want err = %t", c.status, err, c.wantErr)
		}
	}
}

func TestCheckPushPermission_Real(t *testing.T) {
	// Tests should not run in an environment where these registries can
	// be pushed to.
	for _, r := range []name.Reference{
		mustNewTag(t, "ubuntu"),
		mustNewTag(t, "google/cloud-sdk"),
		mustNewTag(t, "microsoft/dotnet:sdk"),
		mustNewTag(t, "gcr.io/non-existent-project/made-up"),
		mustNewTag(t, "gcr.io/google-containers/foo"),
		mustNewTag(t, "quay.io/username/reponame"),
	} {
		if err := CheckPushPermission(r, authn.DefaultKeychain, http.DefaultTransport); err == nil {
			t.Errorf("CheckPushPermission(%s) returned nil", r)
		}
	}
}
