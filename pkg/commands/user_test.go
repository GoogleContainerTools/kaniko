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
	"testing"

	"github.com/GoogleContainerTools/kaniko/pkg/dockerfile"

	"github.com/GoogleContainerTools/kaniko/testutil"
	"github.com/google/go-containerregistry/pkg/v1"
	"github.com/moby/buildkit/frontend/dockerfile/instructions"
)

var userTests = []struct {
	user        string
	expectedUID string
	shouldError bool
}{
	{
		user:        "root",
		expectedUID: "root",
		shouldError: false,
	},
	{
		user:        "0",
		expectedUID: "0",
		shouldError: false,
	},
	{
		user:        "fakeUser",
		expectedUID: "",
		shouldError: true,
	},
	{
		user:        "root:root",
		expectedUID: "root:root",
		shouldError: false,
	},
	{
		user:        "0:root",
		expectedUID: "0:root",
		shouldError: false,
	},
	{
		user:        "root:0",
		expectedUID: "root:0",
		shouldError: false,
	},
	{
		user:        "0:0",
		expectedUID: "0:0",
		shouldError: false,
	},
	{
		user:        "root:fakeGroup",
		expectedUID: "",
		shouldError: true,
	},
	{
		user:        "$envuser",
		expectedUID: "root",
		shouldError: false,
	},
	{
		user:        "root:$envgroup",
		expectedUID: "root:root",
		shouldError: false,
	},
}

func TestUpdateUser(t *testing.T) {
	for _, test := range userTests {
		cfg := &v1.Config{
			Env: []string{
				"envuser=root",
				"envgroup=root",
			},
		}
		cmd := UserCommand{
			cmd: &instructions.UserCommand{
				User: test.user,
			},
		}
		buildArgs := dockerfile.NewBuildArgs([]string{})
		err := cmd.ExecuteCommand(cfg, buildArgs)
		testutil.CheckErrorAndDeepEqual(t, test.shouldError, err, test.expectedUID, cfg.User)
	}
}
