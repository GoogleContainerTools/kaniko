// Copyright 2019 The original author or authors
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

package layout

import (
	"testing"
)

func TestRead(t *testing.T) {
	lp, err := FromPath(testPath)
	if err != nil {
		t.Fatalf("FromPath() = %v", err)
	}
	if testPath != lp.path() {
		t.Errorf("unexpected path %s", lp.path())
	}
}

func TestReadErrors(t *testing.T) {
	if _, err := FromPath(bogusPath); err == nil {
		t.Errorf("FromPath(%s) = nil, expected err", bogusPath)
	}

	// Found this here:
	// https://github.com/golang/go/issues/24195
	invalidPath := "double-null-padded-string\x00\x00"
	if _, err := FromPath(invalidPath); err == nil {
		t.Errorf("FromPath(%s) = nil, expected err", bogusPath)
	}
}
