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

const (
	DefaultKanikoDockerConfigJSON = "/kaniko/.docker/config.json"
	DockerConfigEnvKey            = "DOCKER_CONFIG"
)

func mustTag(t *testing.T, s string) name.Tag {
	tag, err := name.NewTag(s, name.StrictValidation)
	if err != nil {
		t.Fatalf("NewTag: %v", err)
	}
	return tag
}

func TestDockerConfLocationWithFileLocation(t *testing.T) {
	originalDockerConfig := os.Getenv(DockerConfigEnvKey)
	if err := os.Unsetenv(DockerConfigEnvKey); err != nil {
		t.Fatalf("Failed to unset DOCKER_CONFIG: %v", err)
	}
	file, err := ioutil.TempFile("", "docker.conf")
	if err != nil {
		t.Fatalf("could not create temp file: %s", err)
	}
	defer os.Remove(file.Name())
	if err := os.Setenv(DockerConfigEnvKey, file.Name()); err != nil {
		t.Fatalf("Failed to unset DOCKER_CONFIG: %v", err)
	}
	unset := DockerConfLocation()
	unsetExpected := file.Name()
	if unset != unsetExpected {
		t.Errorf("Unexpected default Docker configuration file location: expected:'%s' got:'%s'", unsetExpected, unset)
	}
	restoreOriginalDockerConfigEnv(t, originalDockerConfig)
}

func TestDockerConfLocationWithInvalidFileLocation(t *testing.T) {
	originalDockerConfig := os.Getenv(DockerConfigEnvKey)
	if err := os.Unsetenv(DockerConfigEnvKey); err != nil {
		t.Fatalf("Failed to unset DOCKER_CONFIG: %v", err)
	}
	tmpDir, err := ioutil.TempDir("", "*")
	if err != nil {
		t.Fatalf("could not create temp dir: %s", err)
	}
	defer os.RemoveAll(tmpDir)
	random := "fdgdsfrdfgdf-fdfsf-24dsgfd" //replace with a really random string
	file := filepath.Join(tmpDir, random)  // an random file name, shouldn't exist
	if err := os.Setenv(DockerConfigEnvKey, file); err != nil {
		t.Fatalf("Failed to unset DOCKER_CONFIG: %v", err)
	}
	unset := DockerConfLocation()
	// as file doesn't point to a real file,  DockerConfLocation() should return the default kaniko path for config.json
	unsetExpected := DefaultKanikoDockerConfigJSON
	if unset != unsetExpected {
		t.Errorf("Unexpected default Docker configuration file location: expected:'%s' got:'%s'", unsetExpected, unset)
	}
	restoreOriginalDockerConfigEnv(t, originalDockerConfig)
}

func TestDockerConfLocation(t *testing.T) {
	originalDockerConfig := os.Getenv(DockerConfigEnvKey)

	if err := os.Unsetenv(DockerConfigEnvKey); err != nil {
		t.Fatalf("Failed to unset DOCKER_CONFIG: %v", err)
	}
	unset := DockerConfLocation()
	unsetExpected := DefaultKanikoDockerConfigJSON // will fail on Windows
	if unset != unsetExpected {
		t.Errorf("Unexpected default Docker configuration file location: expected:'%s' got:'%s'", unsetExpected, unset)
	}
	tmpDir, err := ioutil.TempDir("", "*")
	if err != nil {
		t.Fatalf("could not create temp dir: %s", err)
	}
	defer os.RemoveAll(tmpDir)

	dir := filepath.Join(tmpDir, "/kaniko/.docker")
	os.MkdirAll(dir, os.ModePerm)
	if err := os.Setenv(DockerConfigEnvKey, dir); err != nil {
		t.Fatalf("Failed to set DOCKER_CONFIG: %v", err)
	}
	kanikoDefault := DockerConfLocation()
	kanikoDefaultExpected := filepath.Join(tmpDir, DefaultKanikoDockerConfigJSON) // will fail on Windows
	if kanikoDefault != kanikoDefaultExpected {
		t.Errorf("Unexpected kaniko default Docker conf file location: expected:'%s' got:'%s'", kanikoDefaultExpected, kanikoDefault)
	}

	differentPath, err := ioutil.TempDir("", "differentPath")
	if err != nil {
		t.Fatalf("could not create temp dir: %s", err)
	}
	defer os.RemoveAll(differentPath)
	if err := os.Setenv(DockerConfigEnvKey, differentPath); err != nil {
		t.Fatalf("Failed to set DOCKER_CONFIG: %v", err)
	}
	set := DockerConfLocation()
	setExpected := filepath.Join(differentPath, "config.json") // will fail on Windows ?
	if set != setExpected {
		t.Errorf("Unexpected DOCKER_CONF-based file location: expected:'%s' got:'%s'", setExpected, set)
	}
	restoreOriginalDockerConfigEnv(t, originalDockerConfig)
}

func restoreOriginalDockerConfigEnv(t *testing.T, originalDockerConfig string) {
	if originalDockerConfig != "" {
		if err := os.Setenv(DockerConfigEnvKey, originalDockerConfig); err != nil {
			t.Fatalf("Failed to set DOCKER_CONFIG back to original value '%s': %v", originalDockerConfig, err)
		}
	} else {
		if err := os.Unsetenv(DockerConfigEnvKey); err != nil {
			t.Fatalf("Failed to unset DOCKER_CONFIG after testing: %v", err)
		}
	}
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

var calledExecCommand = []bool{}
var calledCheckPushPermission = false

func setCalledFalse() {
	calledExecCommand = []bool{}
	calledCheckPushPermission = false
}

func fakeExecCommand(command string, args ...string) *exec.Cmd {
	calledExecCommand = append(calledExecCommand, true)
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
		description           string
		Destination           []string
		ShouldCallExecCommand []bool
		ExistingConfig        bool
	}{
		{"a gcr image without config", []string{"gcr.io/test-image"}, []bool{true}, false},
		{"a gcr image with config", []string{"gcr.io/test-image"}, []bool{false}, true},
		{"a pkg.dev image without config", []string{"us-docker.pkg.dev/test-image"}, []bool{true}, false},
		{"a pkg.dev image with config", []string{"us-docker.pkg.dev/test-image"}, []bool{false}, true},
		{"localhost registry with config", []string{"localhost:5000/test-image"}, []bool{false}, false},
		{"localhost registry without config", []string{"localhost:5000/test-image"}, []bool{false}, true},
		{"any other registry", []string{"notgcr.io/test-image"}, []bool{false}, false},
		{"multiple destinations pushed to different registry",
			[]string{
				"us-central1-docker.pkg.dev/prj/test-image",
				"us-west-docker.pkg.dev/prj/test-image",
			},
			[]bool{true, true}, false,
		},
		{"same image names with different tags",
			[]string{
				"us-central1-docker.pkg.dev/prj/test-image:tag1",
				"us-central1-docker.pkg.dev/prj/test-image:tag2",
			},
			[]bool{true, true}, false,
		},
		{"same destination image multiple times",
			[]string{
				"us-central1-docker.pkg.dev/prj/test-image",
				"us-central1-docker.pkg.dev/prj/test-image",
			},
			[]bool{true, false}, false,
		},
	}

	execCommand = fakeExecCommand
	checkRemotePushPermission = fakeCheckPushPermission
	for _, test := range tests {
		t.Run(test.description, func(t *testing.T) {
			setCalledFalse()
			fs = afero.NewMemMapFs()
			opts := config.KanikoOptions{
				Destinations: test.Destination,
			}
			if test.ExistingConfig {
				afero.WriteFile(fs, DockerConfLocation(), []byte(""), os.FileMode(0644))
				defer fs.Remove(DockerConfLocation())
			}
			CheckPushPermissions(&opts)
			for i, shdCall := range test.ShouldCallExecCommand {
				if i < len(calledExecCommand) && shdCall != calledExecCommand[i] {
					t.Errorf("Expected calledExecCommand to be %v however it was %v",
						calledExecCommand, shdCall)
				}
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
