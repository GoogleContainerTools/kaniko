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
	"os/user"
	"testing"

	"github.com/GoogleContainerTools/kaniko/pkg/dockerfile"

	"github.com/GoogleContainerTools/kaniko/testutil"
	v1 "github.com/google/go-containerregistry/pkg/v1"
	"github.com/moby/buildkit/frontend/dockerfile/instructions"
)

var userTests = []struct {
	user        string
	userObj     *user.User
	expectedUID string
	expectedGID string
}{
	{
		user:        "root",
		userObj:     &user.User{Uid: "root", Gid: "root"},
		expectedUID: "root",
	},
	{
		user:        "root-add",
		userObj:     &user.User{Uid: "root-add", Gid: "root"},
		expectedUID: "root-add",
	},
	{
		user:        "0",
		userObj:     &user.User{Uid: "0", Gid: "0"},
		expectedUID: "0",
	},
	{
		user:        "fakeUser",
		userObj:     &user.User{Uid: "fakeUser", Gid: "fakeUser"},
		expectedUID: "fakeUser",
	},
	{
		user:        "root",
		userObj:     &user.User{Uid: "root", Gid: "some"},
		expectedUID: "root",
	},
	{
		user:        "0",
		userObj:     &user.User{Uid: "0"},
		expectedUID: "0",
	},
	{
		user:        "root",
		userObj:     &user.User{Uid: "root"},
		expectedUID: "root",
		expectedGID: "f0",
	},
	{
		user:        "0",
		userObj:     &user.User{Uid: "0"},
		expectedUID: "0",
	},
	{
		user:        "$envuser",
		userObj:     &user.User{Uid: "root", Gid: "root"},
		expectedUID: "root",
	},
	{
		user:        "root",
		userObj:     &user.User{Uid: "root"},
		expectedUID: "root",
	},
	{
		user:        "some",
		userObj:     &user.User{Uid: "some"},
		expectedUID: "some",
	},
	{
		user:        "some",
		expectedUID: "some",
	},
}

func TestUpdateUser(t *testing.T) {
	for _, test := range userTests {
		cfg := &v1.Config{
			Env: []string{
				"envuser=root",
				"envgroup=grp",
			},
		}
		cmd := UserCommand{
			cmd: &instructions.UserCommand{
				User: test.user,
			},
		}
		buildArgs := dockerfile.NewBuildArgs([]string{})
		err := cmd.ExecuteCommand(cfg, buildArgs)
		testutil.CheckErrorAndDeepEqual(t, false, err, test.expectedUID, cfg.User)
	}
}
