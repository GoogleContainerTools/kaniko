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

package authn

import (
	"errors"
	"os/exec"
	"testing"

	"github.com/google/go-containerregistry/pkg/name"
)

var (
	testDomain, _ = name.NewRegistry("foo.dev", name.WeakValidation)
)

// errorRunner implements runner to always return an execution error.
type errorRunner struct {
	err error
	msg string
}

// Run implements runner
func (er *errorRunner) Run(cmd *exec.Cmd) error {
	_, err := cmd.Stdout.Write([]byte(er.msg))
	if err != nil {
		return err
	}

	return er.err
}

// printRunner implements runner to write a fixed message to stdout.
type printRunner struct {
	msg string
}

// Run implements runner
func (pr *printRunner) Run(cmd *exec.Cmd) error {
	_, err := cmd.Stdout.Write([]byte(pr.msg))
	return err
}

// errorPrintRunner implements runner to write a fixed message to stdout
// and exit with an error code.
type errorPrintRunner struct {
	msg string
}

// Run implements runner
func (pr *errorPrintRunner) Run(cmd *exec.Cmd) error {
	_, err := cmd.Stdout.Write([]byte(pr.msg))
	if err != nil {
		return err
	}

	return &exec.ExitError{}
}

func TestHelperError(t *testing.T) {
	cases := []struct {
		err    string
		output string
		want   string
	}{{
		// We should show useful output.
		err:    "i am a useless error",
		output: "harmless and helpful output",
		want:   "invoking docker-credential-test: i am a useless error; output: harmless and helpful output",
	}, {
		// We should not show credentials.
		err:    "i am a useless error",
		output: `{"Username":"AzureDiamond","Password":"hunter2"}`,
		want:   "invoking docker-credential-test: i am a useless error",
	}}

	for _, tc := range cases {
		h := &helper{name: "test", domain: testDomain, r: &errorRunner{err: errors.New(tc.err), msg: tc.output}}

		if _, got := h.Authorization(); got.Error() != tc.want {
			t.Errorf("Authorization(); got %v, want %s", got, tc.want)
		}
	}
}

func TestMagicString(t *testing.T) {
	h := &helper{name: "test", domain: testDomain, r: &errorPrintRunner{msg: magicNotFoundMessage}}

	got, err := h.Authorization()
	if err != nil {
		t.Errorf("Authorization() = %v", err)
	}

	// When we get the magic not found message we should fall back on anonymous authentication.
	want, _ := Anonymous.Authorization()
	if got != want {
		t.Errorf("Authorization(); got %v, want %v", got, want)
	}
}

func TestGoodOutput(t *testing.T) {
	output := `{"Username": "foo", "Secret": "bar"}`
	h := &helper{name: "test", domain: testDomain, r: &printRunner{msg: output}}

	got, err := h.Authorization()
	if err != nil {
		t.Errorf("Authorization() = %v", err)
	}

	// When we get the magic not found message we should fall back on anonymous authentication.
	want := "Basic Zm9vOmJhcg=="
	if got != want {
		t.Errorf("Authorization(); got %v, want %v", got, want)
	}
}

func TestBadOutput(t *testing.T) {
	// That extra comma will get ya every time.
	output := `{"Username": "foo", "Secret": "bar",}`
	h := &helper{name: "test", domain: testDomain, r: &printRunner{msg: output}}

	got, err := h.Authorization()
	if err == nil {
		t.Errorf("Authorization() = %v", got)
	}
}
