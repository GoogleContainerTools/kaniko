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
package commands

import (
	"github.com/GoogleCloudPlatform/kaniko/testutil"
	"github.com/containers/image/manifest"
	"github.com/docker/docker/builder/dockerfile/instructions"
	"testing"
)

var userTests = []struct {
	user        string
	expectedUid string
	shouldError bool
}{
	{
		user:        "root",
		expectedUid: "0",
		shouldError: false,
	},
	{
		user:        "0",
		expectedUid: "0",
		shouldError: false,
	},
	{
		user:        "fakeUser",
		expectedUid: "",
		shouldError: true,
	},
	{
		user:        "root:root",
		expectedUid: "0:0",
		shouldError: false,
	},
	{
		user:        "0:root",
		expectedUid: "0:0",
		shouldError: false,
	},
	{
		user:        "root:0",
		expectedUid: "0:0",
		shouldError: false,
	},
	{
		user:        "0:0",
		expectedUid: "0:0",
		shouldError: false,
	},
	{
		user:        "root:fakeGroup",
		expectedUid: "",
		shouldError: true,
	},
	{
		user:        "$envuser",
		expectedUid: "0",
		shouldError: false,
	},
	{
		user:        "root:$envgroup",
		expectedUid: "0:0",
		shouldError: false,
	},
}

func TestUpdateUser(t *testing.T) {
	for _, test := range userTests {
		cfg := &manifest.Schema2Config{
			Env: []string{
				"envuser=root",
				"envgroup=root",
			},
		}
		cmd := UserCommand{
			&instructions.UserCommand{
				User: test.user,
			},
		}
		err := cmd.ExecuteCommand(cfg)
		testutil.CheckErrorAndDeepEqual(t, test.shouldError, err, test.expectedUid, cfg.User)
	}
}
