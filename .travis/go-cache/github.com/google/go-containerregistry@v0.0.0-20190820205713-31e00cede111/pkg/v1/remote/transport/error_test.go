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
	"bytes"
	"encoding/json"
	"io/ioutil"
	"net/http"
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/google/go-cmp/cmp/cmpopts"
)

func TestTemporary(t *testing.T) {
	tests := []struct {
		error *Error
		retry bool
	}{{
		error: &Error{},
		retry: false,
	}, {
		error: &Error{
			Errors: []Diagnostic{{
				Code: BlobUploadInvalidErrorCode,
			}},
		},
		retry: true,
	}, {
		error: &Error{
			Errors: []Diagnostic{{
				Code: BlobUploadInvalidErrorCode,
			}, {
				Code: DeniedErrorCode,
			}},
		},
		retry: false,
	}}

	for _, test := range tests {
		retry := test.error.Temporary()

		if test.retry != retry {
			t.Errorf("Temporary(%s) = %t, wanted %t", test.error, retry, test.retry)
		}
	}
}

func TestCheckErrorNil(t *testing.T) {
	tests := []int{
		http.StatusOK,
		http.StatusAccepted,
		http.StatusCreated,
		http.StatusMovedPermanently,
		http.StatusInternalServerError,
	}

	for _, code := range tests {
		resp := &http.Response{StatusCode: code}

		if err := CheckError(resp, code); err != nil {
			t.Errorf("CheckError(%d) = %v", code, err)
		}
	}
}

func TestCheckErrorNotError(t *testing.T) {
	tests := []struct {
		code int
		body string
		msg  string
	}{{
		code: http.StatusBadRequest,
		body: "",
		msg:  "unsupported status code 400",
	}, {
		code: http.StatusUnauthorized,
		body: "Not JSON",
		msg:  "unsupported status code 401; body: Not JSON",
	}}

	for _, test := range tests {
		resp := &http.Response{
			StatusCode: test.code,
			Body:       ioutil.NopCloser(bytes.NewBufferString(test.body)),
		}

		err := CheckError(resp, http.StatusOK)
		if err == nil {
			t.Fatalf("CheckError(%d, %s) = nil, wanted error", test.code, test.body)
		}
		se, ok := err.(*Error)
		if !ok {
			t.Fatalf("CheckError(%d, %s) = %v, wanted error type", test.code, test.body, err)
		}

		if se.StatusCode != test.code {
			t.Errorf("Incorrect status code, got %d, want %d", se.StatusCode, test.code)
		}

		if se.Error() != test.msg {
			t.Errorf("Incorrect message, got %q, want %q", se.Error(), test.msg)
		}
	}
}

func TestCheckErrorWithError(t *testing.T) {
	tests := []struct {
		code  int
		error *Error
		msg   string
	}{{
		code: http.StatusBadRequest,
		error: &Error{
			Errors: []Diagnostic{{
				Code:    NameInvalidErrorCode,
				Message: "a message for you",
			}},
			StatusCode: 400,
		},
		msg: "NAME_INVALID: a message for you",
	}, {
		code: http.StatusBadRequest,
		error: &Error{
			StatusCode: 400,
		},
		msg: "unsupported status code 400",
	}, {
		code: http.StatusBadRequest,
		error: &Error{
			Errors: []Diagnostic{{
				Code:    NameInvalidErrorCode,
				Message: "a message for you",
			}, {
				Code:    SizeInvalidErrorCode,
				Message: "another message for you",
				Detail:  "with some details",
			}},
			StatusCode: 400,
		},
		msg: "multiple errors returned: NAME_INVALID: a message for you;SIZE_INVALID: another message for you; with some details",
	}}

	for _, test := range tests {
		b, err := json.Marshal(test.error)
		if err != nil {
			t.Errorf("json.Marshal(%v) = %v", test.error, err)
		}
		resp := &http.Response{
			StatusCode: test.code,
			Body:       ioutil.NopCloser(bytes.NewBuffer(b)),
		}

		if err := CheckError(resp, http.StatusOK); err == nil {
			t.Errorf("CheckError(%d, %s) = nil, wanted error", test.code, string(b))
		} else if se, ok := err.(*Error); !ok {
			t.Errorf("CheckError(%d, %s) = %T, wanted *transport.Error", test.code, string(b), se)
		} else if diff := cmp.Diff(test.error, se, cmpopts.IgnoreUnexported(Error{})); diff != "" {
			t.Errorf("CheckError(%d, %s); (-want +got) %s", test.code, string(b), diff)
		} else if diff := cmp.Diff(test.msg, test.error.Error()); diff != "" {
			t.Errorf("CheckError(%d, %s).Error(); (-want +got) %s", test.code, string(b), diff)
		}
	}
}
