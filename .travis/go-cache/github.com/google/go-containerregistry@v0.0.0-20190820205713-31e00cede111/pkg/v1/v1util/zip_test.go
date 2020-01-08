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

package v1util

import (
	"bytes"
	"io/ioutil"
	"testing"
)

func TestReader(t *testing.T) {
	want := "This is the input string."
	buf := bytes.NewBufferString(want)
	zipped, err := GzipReadCloser(ioutil.NopCloser(buf))
	if err != nil {
		t.Errorf("GzipReadCloser() = %v", err)
	}
	unzipped, err := GunzipReadCloser(zipped)
	if err != nil {
		t.Errorf("GunzipReadCloser() = %v", err)
	}

	b, err := ioutil.ReadAll(unzipped)
	if err != nil {
		t.Errorf("ReadAll() = %v", err)
	}
	if got := string(b); got != want {
		t.Errorf("ReadAll(); got %q, want %q", got, want)
	}
}

func TestIsGzipped(t *testing.T) {
	tests := []struct {
		in  []byte
		out bool
		err error
	}{
		{[]byte{}, false, nil},
		{[]byte{'\x00', '\x00', '\x00'}, false, nil},
		{[]byte{'\x1f', '\x8b', '\x1b'}, true, nil},
	}
	for _, test := range tests {
		reader := bytes.NewReader(test.in)
		got, err := IsGzipped(reader)
		if got != test.out {
			t.Errorf("IsGzipped; n: got %v, wanted %v\n", got, test.out)
		}
		if err != test.err {
			t.Errorf("IsGzipped; err: got %v, wanted %v\n", err, test.err)
		}
	}
}
