/*
Copyright 2018 Google LLC

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

package executor

import (
	"bytes"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"

	"github.com/GoogleContainerTools/kaniko/pkg/config"
	"github.com/GoogleContainerTools/kaniko/pkg/util"
	"github.com/GoogleContainerTools/kaniko/testutil"
	"github.com/google/go-containerregistry/pkg/authn"
	"github.com/google/go-containerregistry/pkg/name"
	"github.com/google/go-containerregistry/pkg/v1/layout"
	"github.com/google/go-containerregistry/pkg/v1/random"
	"github.com/google/go-containerregistry/pkg/v1/validate"
	"github.com/spf13/afero"
)

func mustTag(t *testing.T, s string) name.Tag {
	tag, err := name.NewTag(s, name.StrictValidation)
	if err != nil {
		t.Fatalf("NewTag: %v", err)
	}
	return tag
}

func TestWriteImageOutputs(t *testing.T) {
	img, err := random.Image(1024, 3)
	if err != nil {
		t.Fatalf("random.Image: %v", err)
	}
	d, err := img.Digest()
	if err != nil {
		t.Fatalf("Digest: %v", err)
	}

	for _, c := range []struct {
		desc, env string
		tags      []name.Tag
		want      string
	}{{
		desc: "env unset, no output",
		env:  "",
	}, {
		desc: "env set, one tag",
		env:  "/foo",
		tags: []name.Tag{mustTag(t, "gcr.io/foo/bar:latest")},
		want: fmt.Sprintf(`{"name":"gcr.io/foo/bar:latest","digest":%q}
`, d),
	}, {
		desc: "env set, two tags",
		env:  "/foo",
		tags: []name.Tag{
			mustTag(t, "gcr.io/foo/bar:latest"),
			mustTag(t, "gcr.io/baz/qux:latest"),
		},
		want: fmt.Sprintf(`{"name":"gcr.io/foo/bar:latest","digest":%q}
{"name":"gcr.io/baz/qux:latest","digest":%q}
`, d, d),
	}} {
		t.Run(c.desc, func(t *testing.T) {
			newOsFs = afero.NewMemMapFs()
			if c.want == "" {
				newOsFs = afero.NewReadOnlyFs(newOsFs) // No files should be written.
			}

			os.Setenv("BUILDER_OUTPUT", c.env)
			if err := writeImageOutputs(img, c.tags); err != nil {
				t.Fatalf("writeImageOutputs: %v", err)
			}

			if c.want == "" {
				return
			}

			b, err := afero.ReadFile(newOsFs, filepath.Join(c.env, "images"))
			if err != nil {
				t.Fatalf("ReadFile: %v", err)
			}

			if got := string(b); got != c.want {
				t.Fatalf(" got: %s\nwant: %s", got, c.want)
			}
		})
	}
}

func TestHeaderAdded(t *testing.T) {
	tests := []struct {
		name     string
		upstream string
		expected string
	}{{
		name:     "upstream env variable set",
		upstream: "skaffold-v0.25.45",
		expected: "kaniko/unset,skaffold-v0.25.45",
	}, {
		name:     "upstream env variable not set",
		expected: "kaniko/unset",
	},
	}
	for _, test := range tests {

		t.Run(test.name, func(t *testing.T) {
			rt := &withUserAgent{t: &mockRoundTripper{}}
			if test.upstream != "" {
				os.Setenv("UPSTREAM_CLIENT_TYPE", test.upstream)
				defer func() { os.Unsetenv("UPSTREAM_CLIENT_TYPE") }()
			}
			req, err := http.NewRequest("GET", "dummy", nil) //nolint:noctx
			if err != nil {
				t.Fatalf("culd not create a req due to %s", err)
			}
			resp, err := rt.RoundTrip(req)
			testutil.CheckError(t, false, err)
			defer resp.Body.Close()
			body, err := io.ReadAll(resp.Body)
			testutil.CheckErrorAndDeepEqual(t, false, err, test.expected, string(body))
		})
	}

}

type mockRoundTripper struct {
}

func (m *mockRoundTripper) RoundTrip(r *http.Request) (*http.Response, error) {
	ua := r.UserAgent()
	return &http.Response{Body: io.NopCloser(bytes.NewBufferString(ua))}, nil
}

func TestOCILayoutPath(t *testing.T) {
	tmpDir := t.TempDir()

	image, err := random.Image(1024, 4)
	if err != nil {
		t.Fatalf("could not create image: %s", err)
	}

	digest, err := image.Digest()
	if err != nil {
		t.Fatalf("could not get image digest: %s", err)
	}

	want, err := image.Manifest()
	if err != nil {
		t.Fatalf("could not get image manifest: %s", err)
	}

	opts := config.KanikoOptions{
		NoPush:        true,
		OCILayoutPath: tmpDir,
	}

	if err := DoPush(image, &opts); err != nil {
		t.Fatalf("could not push image: %s", err)
	}

	layoutIndex, err := layout.ImageIndexFromPath(tmpDir)
	if err != nil {
		t.Fatalf("could not get index from layout: %s", err)
	}
	testutil.CheckError(t, false, validate.Index(layoutIndex))

	layoutImage, err := layoutIndex.Image(digest)
	if err != nil {
		t.Fatalf("could not get image from layout: %s", err)
	}

	got, err := layoutImage.Manifest()
	testutil.CheckErrorAndDeepEqual(t, false, err, want, got)
}

func TestImageNameDigestFile(t *testing.T) {
	image, err := random.Image(1024, 4)
	if err != nil {
		t.Fatalf("could not create image: %s", err)
	}

	digest, err := image.Digest()
	if err != nil {
		t.Fatalf("could not get image digest: %s", err)
	}

	opts := config.KanikoOptions{
		NoPush:              true,
		Destinations:        []string{"gcr.io/foo/bar:latest", "bob/image"},
		ImageNameDigestFile: "tmpFile",
	}

	defer os.Remove("tmpFile")

	if err := DoPush(image, &opts); err != nil {
		t.Fatalf("could not push image: %s", err)
	}

	want := []byte("gcr.io/foo/bar@" + digest.String() + "\nindex.docker.io/bob/image@" + digest.String() + "\n")

	got, err := os.ReadFile("tmpFile")

	testutil.CheckErrorAndDeepEqual(t, false, err, want, got)

}

func TestDoPushWithOpts(t *testing.T) {
	tarPath := "image.tar"

	for _, tc := range []struct {
		name        string
		opts        config.KanikoOptions
		expectedErr bool
	}{
		{
			name: "no push with tarPath without destinations",
			opts: config.KanikoOptions{
				NoPush:  true,
				TarPath: tarPath,
			},
			expectedErr: false,
		}, {
			name: "no push with tarPath with destinations",
			opts: config.KanikoOptions{
				NoPush:       true,
				TarPath:      tarPath,
				Destinations: []string{"image"},
			},
			expectedErr: false,
		}, {
			name: "no push with tarPath with destinations empty",
			opts: config.KanikoOptions{
				NoPush:       true,
				TarPath:      tarPath,
				Destinations: []string{},
			},
			expectedErr: false,
		}, {
			name: "tarPath with destinations empty",
			opts: config.KanikoOptions{
				NoPush:       false,
				TarPath:      tarPath,
				Destinations: []string{},
			},
			expectedErr: true,
		}} {
		t.Run(tc.name, func(t *testing.T) {
			image, err := random.Image(1024, 4)
			if err != nil {
				t.Fatalf("could not create image: %s", err)
			}
			defer os.Remove("image.tar")

			err = DoPush(image, &tc.opts)
			if err != nil {
				if !tc.expectedErr {
					t.Errorf("unexpected error with opts: could not push image: %s", err)
				}
			} else {
				if tc.expectedErr {
					t.Error("expected error with opts not found")
				}
			}

		})
	}
}

func TestImageNameTagDigestFile(t *testing.T) {
	image, err := random.Image(1024, 4)
	if err != nil {
		t.Fatalf("could not create image: %s", err)
	}

	digest, err := image.Digest()
	if err != nil {
		t.Fatalf("could not get image digest: %s", err)
	}

	opts := config.KanikoOptions{
		NoPush:                 true,
		Destinations:           []string{"gcr.io/foo/bar:123", "bob/image"},
		ImageNameTagDigestFile: "tmpFile",
	}

	defer os.Remove("tmpFile")

	if err := DoPush(image, &opts); err != nil {
		t.Fatalf("could not push image: %s", err)
	}

	want := []byte("gcr.io/foo/bar:123@" + digest.String() + "\nindex.docker.io/bob/image:latest@" + digest.String() + "\n")

	got, err := os.ReadFile("tmpFile")

	testutil.CheckErrorAndDeepEqual(t, false, err, want, got)
}

var checkPushPermsCallCount = 0

func resetCalledCount() {
	checkPushPermsCallCount = 0
}

func fakeCheckPushPermission(ref name.Reference, kc authn.Keychain, t http.RoundTripper) error {
	checkPushPermsCallCount++
	return nil
}

func TestCheckPushPermissions(t *testing.T) {
	tests := []struct {
		description                     string
		cacheRepo                       string
		checkPushPermsExpectedCallCount int
		destinations                    []string
		existingConfig                  bool
		noPush                          bool
		noPushCache                     bool
	}{
		{description: "a gcr image without config", destinations: []string{"gcr.io/test-image"}, checkPushPermsExpectedCallCount: 1},
		{description: "a gcr image with config", destinations: []string{"gcr.io/test-image"}, existingConfig: true, checkPushPermsExpectedCallCount: 1},
		{description: "a pkg.dev image without config", destinations: []string{"us-docker.pkg.dev/test-image"}, checkPushPermsExpectedCallCount: 1},
		{description: "a pkg.dev image with config", destinations: []string{"us-docker.pkg.dev/test-image"}, existingConfig: true, checkPushPermsExpectedCallCount: 1},
		{description: "localhost registry without config", destinations: []string{"localhost:5000/test-image"}, checkPushPermsExpectedCallCount: 1},
		{description: "localhost registry with config", destinations: []string{"localhost:5000/test-image"}, existingConfig: true, checkPushPermsExpectedCallCount: 1},
		{description: "any other registry", destinations: []string{"notgcr.io/test-image"}, checkPushPermsExpectedCallCount: 1},
		{
			description: "multiple destinations pushed to different registry",
			destinations: []string{
				"us-central1-docker.pkg.dev/prj/test-image",
				"us-west-docker.pkg.dev/prj/test-image",
			},
			checkPushPermsExpectedCallCount: 2,
		},
		{
			description: "same image names with different tags",
			destinations: []string{
				"us-central1-docker.pkg.dev/prj/test-image:tag1",
				"us-central1-docker.pkg.dev/prj/test-image:tag2",
			},
			checkPushPermsExpectedCallCount: 1,
		},
		{
			description: "same destination image multiple times",
			destinations: []string{
				"us-central1-docker.pkg.dev/prj/test-image",
				"us-central1-docker.pkg.dev/prj/test-image",
			},
			checkPushPermsExpectedCallCount: 1,
		},
		{
			description:                     "no push and no push cache",
			destinations:                    []string{"us-central1-docker.pkg.dev/prj/test-image"},
			checkPushPermsExpectedCallCount: 0,
			noPush:                          true,
			noPushCache:                     true,
		},
		{
			description:                     "no push and push cache",
			destinations:                    []string{"us-central1-docker.pkg.dev/prj/test-image"},
			cacheRepo:                       "us-central1-docker.pkg.dev/prj/cache-image",
			checkPushPermsExpectedCallCount: 1,
			noPush:                          true,
		},
		{
			description:                     "no push and cache repo is OCI image layout",
			destinations:                    []string{"us-central1-docker.pkg.dev/prj/test-image"},
			cacheRepo:                       "oci:/some-layout-path",
			checkPushPermsExpectedCallCount: 0,
			noPush:                          true,
		},
	}

	checkRemotePushPermission = fakeCheckPushPermission
	for _, test := range tests {
		t.Run(test.description, func(t *testing.T) {
			resetCalledCount()
			newOsFs = afero.NewMemMapFs()
			opts := config.KanikoOptions{
				CacheRepo:    test.cacheRepo,
				Destinations: test.destinations,
				NoPush:       test.noPush,
				NoPushCache:  test.noPushCache,
			}
			if test.existingConfig {
				afero.WriteFile(newOsFs, util.DockerConfLocation(), []byte(""), os.FileMode(0644))
				defer newOsFs.Remove(util.DockerConfLocation())
			}
			CheckPushPermissions(&opts)
			if checkPushPermsCallCount != test.checkPushPermsExpectedCallCount {
				t.Errorf("expected check push permissions call count to be %d but it was %d", test.checkPushPermsExpectedCallCount, checkPushPermsCallCount)
			}
		})
	}
}

func TestSkipPushPermission(t *testing.T) {
	tests := []struct {
		description                     string
		cacheRepo                       string
		checkPushPermsExpectedCallCount int
		destinations                    []string
		existingConfig                  bool
		noPush                          bool
		noPushCache                     bool
		skipPushPermission              bool
	}{
		{description: "skip push permission enabled", destinations: []string{"test.io/skip"}, checkPushPermsExpectedCallCount: 0, skipPushPermission: true},
		{description: "skip push permission disabled", destinations: []string{"test.io/push"}, checkPushPermsExpectedCallCount: 1, skipPushPermission: false},
	}

	checkRemotePushPermission = fakeCheckPushPermission
	for _, test := range tests {
		t.Run(test.description, func(t *testing.T) {
			resetCalledCount()
			newOsFs = afero.NewMemMapFs()
			opts := config.KanikoOptions{
				CacheRepo:               test.cacheRepo,
				Destinations:            test.destinations,
				NoPush:                  test.noPush,
				NoPushCache:             test.noPushCache,
				SkipPushPermissionCheck: test.skipPushPermission,
			}
			if test.existingConfig {
				afero.WriteFile(newOsFs, util.DockerConfLocation(), []byte(""), os.FileMode(0644))
				defer newOsFs.Remove(util.DockerConfLocation())
			}
			CheckPushPermissions(&opts)
			if checkPushPermsCallCount != test.checkPushPermsExpectedCallCount {
				t.Errorf("expected check push permissions call count to be %d but it was %d", test.checkPushPermsExpectedCallCount, checkPushPermsCallCount)
			}
		})
	}
}

func TestHelperProcess(t *testing.T) {
	if os.Getenv("GO_WANT_HELPER_PROCESS") != "1" {
		return
	}
	fmt.Fprintf(os.Stdout, "fake result")
	os.Exit(0)
}

func TestWriteDigestFile(t *testing.T) {
	tmpDir := t.TempDir()

	t.Run("parent directory does not exist", func(t *testing.T) {
		err := writeDigestFile(tmpDir+"/test/df", []byte("test"))
		if err != nil {
			t.Errorf("expected file to be written successfully, but got error: %v", err)
		}
	})

	t.Run("parent directory exists", func(t *testing.T) {
		err := writeDigestFile(tmpDir+"/df", []byte("test"))
		if err != nil {
			t.Errorf("expected file to be written successfully, but got error: %v", err)
		}
	})

	t.Run("https PUT OK", func(t *testing.T) {
		var uploadedContent []byte

		// Start a test server that checks the PUT request.
		server := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if r.Method != http.MethodPut {
				w.WriteHeader(http.StatusMethodNotAllowed)
				return
			}
			uploadedContent, _ = io.ReadAll(r.Body)
			w.WriteHeader(http.StatusNoContent)
		}))
		defer server.Close()

		// Temporarily replace the default client with the test server client to avoid TLS verification errors.
		oldClient := http.DefaultClient
		defer func() { http.DefaultClient = oldClient }()
		http.DefaultClient = server.Client()

		err := writeDigestFile(server.URL+"/df?sig=1234", []byte("test"))
		if err != nil {
			t.Fatalf("expected file to be written successfully, but got error: %v", err)
		}
		if string(uploadedContent) != "test" {
			t.Errorf("expected uploaded content to be 'test', but got '%s'", uploadedContent)
		}
	})
}
