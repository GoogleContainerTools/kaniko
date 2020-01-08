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

package retry

import (
	"fmt"
	"testing"

	"k8s.io/apimachinery/pkg/util/wait"
)

type temp struct{}

func (e temp) Error() string {
	return "temporary error"
}

func (e temp) Temporary() bool {
	return true
}

func TestRetry(t *testing.T) {
	for i, test := range []struct {
		predicate   Predicate
		err         error
		shouldRetry bool
	}{{
		predicate:   IsTemporary,
		err:         nil,
		shouldRetry: false,
	}, {
		predicate:   IsTemporary,
		err:         fmt.Errorf("not temporary"),
		shouldRetry: false,
	}, {
		predicate:   IsNotNil,
		err:         fmt.Errorf("not temporary"),
		shouldRetry: true,
	}, {
		predicate:   IsTemporary,
		err:         temp{},
		shouldRetry: true,
	}} {
		// Make sure we retry 5 times if we shouldRetry.
		steps := 5
		backoff := wait.Backoff{
			Steps: steps,
		}

		// Count how many times this function is invoked.
		count := 0
		f := func() error {
			count++
			return test.err
		}

		Retry(f, test.predicate, backoff)

		if test.shouldRetry && count != steps {
			t.Errorf("expected %d to retry %v, did not", i, test.err)
		} else if !test.shouldRetry && count == steps {
			t.Errorf("expected %d not to retry %v, but did", i, test.err)
		}
	}
}

// Make sure we don't panic.
func TestNil(t *testing.T) {
	if err := Retry(nil, nil, wait.Backoff{}); err == nil {
		t.Errorf("got nil when passing in nil f")
	}
	if err := Retry(func() error { return nil }, nil, wait.Backoff{}); err == nil {
		t.Errorf("got nil when passing in nil p")
	}
}
