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
	"crypto/tls"
	"crypto/x509"
	"fmt"
	"io/ioutil"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"testing"

	"github.com/GoogleContainerTools/kaniko/pkg/config"
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
			fs = afero.NewMemMapFs()
			if c.want == "" {
				fs = afero.NewReadOnlyFs(fs) // No files should be written.
			}

			os.Setenv("BUILDER_OUTPUT", c.env)
			if err := writeImageOutputs(img, c.tags); err != nil {
				t.Fatalf("writeImageOutputs: %v", err)
			}

			if c.want == "" {
				return
			}

			b, err := afero.ReadFile(fs, filepath.Join(c.env, "images"))
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
			req, err := http.NewRequest("GET", "dummy", nil)
			if err != nil {
				t.Fatalf("culd not create a req due to %s", err)
			}
			resp, err := rt.RoundTrip(req)
			testutil.CheckError(t, false, err)
			defer resp.Body.Close()
			body, err := ioutil.ReadAll(resp.Body)
			testutil.CheckErrorAndDeepEqual(t, false, err, test.expected, string(body))
		})
	}

}

type mockRoundTripper struct {
}

func (m *mockRoundTripper) RoundTrip(r *http.Request) (*http.Response, error) {
	ua := r.UserAgent()
	return &http.Response{Body: ioutil.NopCloser(bytes.NewBufferString(ua))}, nil
}

func TestOCILayoutPath(t *testing.T) {
	tmpDir, err := ioutil.TempDir("", "")
	if err != nil {
		t.Fatalf("could not create temp dir: %s", err)
	}
	defer os.RemoveAll(tmpDir)

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

	got, err := ioutil.ReadFile("tmpFile")

	testutil.CheckErrorAndDeepEqual(t, false, err, want, got)

}

type mockedCertPool struct {
	certificatesPath []string
}

func (m *mockedCertPool) value() *x509.CertPool {
	return &x509.CertPool{}
}

func (m *mockedCertPool) append(path string) error {
	m.certificatesPath = append(m.certificatesPath, path)
	return nil
}

func Test_makeTransport(t *testing.T) {
	registryName := "my.registry.name"

	tests := []struct {
		name  string
		opts  *config.KanikoOptions
		check func(*tls.Config, *mockedCertPool)
	}{
		{
			name: "SkipTLSVerify set",
			opts: &config.KanikoOptions{SkipTLSVerify: true},
			check: func(config *tls.Config, pool *mockedCertPool) {
				if !config.InsecureSkipVerify {
					t.Errorf("makeTransport().TLSClientConfig.InsecureSkipVerify not set while SkipTLSVerify set")
				}
			},
		},
		{
			name: "SkipTLSVerifyRegistries set with expected registry",
			opts: &config.KanikoOptions{SkipTLSVerifyRegistries: []string{registryName}},
			check: func(config *tls.Config, pool *mockedCertPool) {
				if !config.InsecureSkipVerify {
					t.Errorf("makeTransport().TLSClientConfig.InsecureSkipVerify not set while SkipTLSVerifyRegistries set with registry name")
				}
			},
		},
		{
			name: "SkipTLSVerifyRegistries set with other registry",
			opts: &config.KanikoOptions{SkipTLSVerifyRegistries: []string{fmt.Sprintf("other.%s", registryName)}},
			check: func(config *tls.Config, pool *mockedCertPool) {
				if config.InsecureSkipVerify {
					t.Errorf("makeTransport().TLSClientConfig.InsecureSkipVerify set while SkipTLSVerifyRegistries not set with registry name")
				}
			},
		},
		{
			name: "RegistriesCertificates set for registry",
			opts: &config.KanikoOptions{RegistriesCertificates: map[string]string{registryName: "/path/to/the/certificate.cert"}},
			check: func(config *tls.Config, pool *mockedCertPool) {
				if len(pool.certificatesPath) != 1 || pool.certificatesPath[0] != "/path/to/the/certificate.cert" {
					t.Errorf("makeTransport().RegistriesCertificates certificate not appended to system certificates")
				}
			},
		},
		{
			name: "RegistriesCertificates set for another registry",
			opts: &config.KanikoOptions{RegistriesCertificates: map[string]string{fmt.Sprintf("other.%s=", registryName): "/path/to/the/certificate.cert"}},
			check: func(config *tls.Config, pool *mockedCertPool) {
				if len(pool.certificatesPath) != 0 {
					t.Errorf("makeTransport().RegistriesCertificates certificate appended to system certificates while added for other registry")
				}
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var certificatesPath []string
			certPool := mockedCertPool{
				certificatesPath: certificatesPath,
			}
			var mockedSystemCertLoader systemCertLoader = func() CertPool {
				return &certPool
			}
			transport := makeTransport(tt.opts, registryName, mockedSystemCertLoader)
			tt.check(transport.(*http.Transport).TLSClientConfig, &certPool)
		})
	}
}

var calledExecCommand = false
var calledCheckPushPermission = false

func setCalledFalse() {
	calledExecCommand = false
	calledCheckPushPermission = false
}

func fakeExecCommand(command string, args ...string) *exec.Cmd {
	calledExecCommand = true
	cs := []string{"-test.run=TestHelperProcess", "--", command}
	cs = append(cs, args...)
	cmd := exec.Command(os.Args[0], cs...)
	cmd.Env = []string{"GO_WANT_HELPER_PROCESS=1"}
	return cmd
}

func fakeCheckPushPermission(ref name.Reference, kc authn.Keychain, t http.RoundTripper) error {
	calledCheckPushPermission = true
	return nil
}

func TestCheckPushPermissions(t *testing.T) {
	tests := []struct {
		Destination           string
		ShouldCallExecCommand bool
		ExistingConfig        bool
	}{
		{"gcr.io/test-image", true, false},
		{"gcr.io/test-image", false, true},
		{"localhost:5000/test-image", false, false},
		{"localhost:5000/test-image", false, true},
	}

	execCommand = fakeExecCommand
	checkRemotePushPermission = fakeCheckPushPermission
	for _, test := range tests {
		testName := fmt.Sprintf("%s_ExistingDockerConf_%v", test.Destination, test.ExistingConfig)
		t.Run(testName, func(t *testing.T) {
			fs = afero.NewMemMapFs()
			opts := config.KanikoOptions{
				Destinations: []string{test.Destination},
			}
			if test.ExistingConfig {
				afero.WriteFile(fs, DockerConfLocation, []byte(""), os.FileMode(0644))
			}
			CheckPushPermissions(&opts)
			if test.ShouldCallExecCommand != calledExecCommand {
				t.Errorf("Expected calledExecCommand to be %v however it was %v",
					calledExecCommand, test.ShouldCallExecCommand)
			}
			setCalledFalse()
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
