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
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/google/go-containerregistry/pkg/authn"
	"github.com/google/go-containerregistry/pkg/name"
	"golang.org/x/oauth2"
)

const (
	// Fails to parse as JSON at all.
	badoutput = ""

	// Fails to parse token_expiry format.
	badexpiry = `
{
  "credential": {
    "access_token": "mytoken",
    "token_expiry": "most-definitely-not-a-date"
  }
}`

	// Expires in 6,000 years. Hopefully nobody is using software then.
	success = `
{
  "credential": {
    "access_token": "mytoken",
    "token_expiry": "8018-12-02T04:08:13Z"
  }
}`
)

// We'll invoke ourselves with a special environment variable in order to mock
// out the gcloud dependency of gcloudSource. The exec package does this, too.
//
// See: https://www.joeshaw.org/testing-with-os-exec-and-testmain/
func TestMain(m *testing.M) {
	switch os.Getenv("GO_TEST_MODE") {
	case "":
		// Normal test mode
		os.Exit(m.Run())

	case "error":
		// Makes cmd.Run() return an error.
		os.Exit(2)

	case "badoutput":
		// Makes the gcloudOutput Unmarshaler fail.
		fmt.Println(badoutput)

	case "badexpiry":
		// Makes the token_expiry time parser fail.
		fmt.Println(badexpiry)

	case "success":
		// Returns a seemingly valid token.
		fmt.Println(success)
	}
}

func newGcloudCmdMock(env string) func() *exec.Cmd {
	return func() *exec.Cmd {
		cmd := exec.Command(os.Args[0])
		cmd.Env = []string{fmt.Sprintf("GO_TEST_MODE=%s", env)}
		return cmd
	}
}

func TestGcloudErrors(t *testing.T) {
	cases := []struct {
		env string

		// Just look for the prefix because we can't control other packages' errors.
		wantPrefix string
	}{{
		env:        "error",
		wantPrefix: "error executing `gcloud config config-helper`:",
	}, {
		env:        "badoutput",
		wantPrefix: "failed to parse `gcloud config config-helper` output:",
	}, {
		env:        "badexpiry",
		wantPrefix: "failed to parse gcloud token expiry:",
	}}

	for _, tc := range cases {
		t.Run(tc.env, func(t *testing.T) {
			GetGcloudCmd = newGcloudCmdMock(tc.env)

			if _, err := NewGcloudAuthenticator(); err == nil {
				t.Errorf("wanted error, got nil")
			} else if got := err.Error(); !strings.HasPrefix(got, tc.wantPrefix) {
				t.Errorf("wanted error prefix %q, got %q", tc.wantPrefix, got)
			}
		})
	}
}

func TestGcloudSuccess(t *testing.T) {
	GetGcloudCmd = newGcloudCmdMock("success")

	auth, err := NewGcloudAuthenticator()
	if err != nil {
		t.Fatalf("NewGcloudAuthenticator got error %v", err)
	}

	token, err := auth.Authorization()
	if err != nil {
		t.Fatalf("Authorization got error %v", err)
	}

	if want, got := "Bearer mytoken", token; want != got {
		t.Errorf("wanted token %q, got %q", want, got)
	}
}

//
// Keychain tests are in here so we can reuse the fake gcloud stuff.
//

func mustRegistry(r string) name.Registry {
	reg, err := name.NewRegistry(r, name.StrictValidation)
	if err != nil {
		panic(err)
	}
	return reg
}

func TestKeychainDockerHub(t *testing.T) {
	if auth, err := Keychain.Resolve(mustRegistry("index.docker.io")); err != nil {
		t.Errorf("expected success, got: %v", err)
	} else if auth != authn.Anonymous {
		t.Errorf("expected anonymous, got: %v", auth)
	}
}

func TestKeychainGCR(t *testing.T) {
	cases := []string{
		"gcr.io",
		"us.gcr.io",
		"asia.gcr.io",
		"eu.gcr.io",
		"staging-k8s.gcr.io",
		"global.gcr.io",
	}

	// Env should fail.
	if err := os.Setenv("GOOGLE_APPLICATION_CREDENTIALS", "/dev/null"); err != nil {
		t.Fatalf("unexpected err os.Setenv: %v", err)
	}

	// Gcloud should succeed.
	GetGcloudCmd = newGcloudCmdMock("success")

	for i, tc := range cases {
		t.Run(fmt.Sprintf("cases[%d]", i), func(t *testing.T) {
			if auth, err := Keychain.Resolve(mustRegistry(tc)); err != nil {
				t.Errorf("expected success, got: %v", err)
			} else if auth == authn.Anonymous {
				t.Errorf("expected not anonymous auth, got: %v", auth)
			}
		})
	}
}

func TestKeychainEnv(t *testing.T) {
	wd, err := os.Getwd()
	if err != nil {
		t.Fatalf("unexpected err os.Getwd: %v", err)
	}

	keyFile := filepath.Join(wd, "testdata", "key.json")

	if err := os.Setenv("GOOGLE_APPLICATION_CREDENTIALS", keyFile); err != nil {
		t.Fatalf("unexpected err os.Setenv: %v", err)
	}

	if auth, err := Keychain.Resolve(mustRegistry("gcr.io")); err != nil {
		t.Errorf("expected success, got: %v", err)
	} else if auth == authn.Anonymous {
		t.Errorf("expected not anonymous auth, got: %v", auth)
	}
}

func TestKeychainError(t *testing.T) {
	if err := os.Setenv("GOOGLE_APPLICATION_CREDENTIALS", "/dev/null"); err != nil {
		t.Fatalf("unexpected err os.Setenv: %v", err)
	}

	GetGcloudCmd = newGcloudCmdMock("badoutput")

	if _, err := Keychain.Resolve(mustRegistry("gcr.io")); err == nil {
		t.Fatalf("expected err, got: %v", err)
	}
}

type badSource struct{}

func (bs badSource) Token() (*oauth2.Token, error) {
	return nil, fmt.Errorf("oops")
}

// This test is silly, but coverage.
func TestTokenSourceAuthError(t *testing.T) {
	auth := tokenSourceAuth{badSource{}}

	_, err := auth.Authorization()
	if err == nil {
		t.Errorf("expected err, got nil")
	}
}

func TestNewEnvAuthenticatorFailure(t *testing.T) {
	if err := os.Setenv("GOOGLE_APPLICATION_CREDENTIALS", "/dev/null"); err != nil {
		t.Fatalf("unexpected err os.Setenv: %v", err)
	}

	// Expect error.
	_, err := NewEnvAuthenticator()
	if err == nil {
		t.Errorf("expected err, got nil")
	}
}

func TestNewEnvAuthenticatorSuccess(t *testing.T) {
	wd, err := os.Getwd()
	if err != nil {
		t.Fatalf("unexpected err os.Getwd: %v", err)
	}

	keyFile := filepath.Join(wd, "testdata", "key.json")

	if err := os.Setenv("GOOGLE_APPLICATION_CREDENTIALS", keyFile); err != nil {
		t.Fatalf("unexpected err os.Setenv: %v", err)
	}

	_, err = NewEnvAuthenticator()
	if err != nil {
		t.Fatalf("unexpected err NewEnvAuthenticator: %v", err)
	}
}
