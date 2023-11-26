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

package util

import (
	"bufio"
	"bytes"
	"fmt"
	"testing"

	"github.com/GoogleContainerTools/kaniko/testutil"
)

func TestGetInputFrom(t *testing.T) {
	validInput := []byte("Valid\n")
	validReader := bufio.NewReader(bytes.NewReader((validInput)))
	validValue, err := GetInputFrom(validReader)
	testutil.CheckErrorAndDeepEqual(t, false, err, validInput, validValue)
}

func makeRetryFunc(numFailures int) retryFunc {
	i := -1

	return func() error {
		i++
		if i < numFailures {
			return fmt.Errorf("Failing with i=%v", i)
		}
		return nil
	}
}

func TestRetry(t *testing.T) {
	// test with a function that does not return an error
	if err := Retry(makeRetryFunc(0), 0, 10); err != nil {
		t.Fatalf("Not expecting error: %v", err)
	}
	if err := Retry(makeRetryFunc(0), 3, 10); err != nil {
		t.Fatalf("Not expecting error: %v", err)
	}

	// test with a function that returns an error twice
	if err := Retry(makeRetryFunc(2), 0, 10); err == nil {
		t.Fatal("Expecting error", err)
	}
	if err := Retry(makeRetryFunc(2), 1, 10); err == nil {
		t.Fatal("Expecting error", err)
	}
	if err := Retry(makeRetryFunc(2), 2, 10); err != nil {
		t.Fatalf("Not expecting error: %v", err)
	}
}

func makeRetryFuncWithResult(numFailures int) func() (int, error) {
	i := -1

	return func() (int, error) {
		i++
		if i < numFailures {
			return i, fmt.Errorf("Failing with i=%v", i)
		}
		return i, nil
	}
}

func TestRetryWithResult(t *testing.T) {
	// test with a function that does not return an error
	result, err := RetryWithResult(makeRetryFuncWithResult(0), 0, 10)
	if err != nil || result != 0 {
		t.Fatalf("Got result %d and error: %v", result, err)
	}
	result, err = RetryWithResult(makeRetryFuncWithResult(0), 3, 10)
	if err != nil || result != 0 {
		t.Fatalf("Got result %d and error: %v", result, err)
	}

	// test with a function that returns an error twice
	result, err = RetryWithResult(makeRetryFuncWithResult(2), 0, 10)
	if err == nil || result != 0 {
		t.Fatalf("Got result %d and error: %v", result, err)
	}
	result, err = RetryWithResult(makeRetryFuncWithResult(2), 1, 10)
	if err == nil || result != 1 {
		t.Fatalf("Got result %d and error: %v", result, err)
	}
	result, err = RetryWithResult(makeRetryFuncWithResult(2), 2, 10)
	if err != nil || result != 2 {
		t.Fatalf("Got result %d and error: %v", result, err)
	}
}
