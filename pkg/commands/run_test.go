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

	"github.com/GoogleContainerTools/kaniko/testutil"
)

func Test_addDefaultHOME(t *testing.T) {
	tests := []struct {
		name     string
		user     string
		mockUser *user.User
		initial  []string
		expected []string
	}{
		{
			name: "HOME already set",
			user: "",
			initial: []string{
				"HOME=/something",
				"PATH=/something/else",
			},
			expected: []string{
				"HOME=/something",
				"PATH=/something/else",
			},
		},
		{
			name: "HOME not set, user not set",
			user: "",
			initial: []string{
				"PATH=/something/else",
			},
			expected: []string{
				"PATH=/something/else",
				"HOME=/root",
			},
		},
		{
			name: "HOME not set, user and homedir for the user set",
			user: "www-add",
			mockUser: &user.User{
				Username: "www-add",
				HomeDir:  "some-other",
			},
			initial: []string{
				"PATH=/something/else",
			},
			expected: []string{
				"PATH=/something/else",
				"HOME=/home/www-add",
			},
		},
		{
			name: "HOME not set, user set",
			user: "www-add",
			mockUser: &user.User{
				Username: "www-add",
			},
			initial: []string{
				"PATH=/something/else",
			},
			expected: []string{
				"PATH=/something/else",
				"HOME=/home/www-add",
			},
		},
		{
			name: "HOME not set, user is set",
			user: "newuser",
			mockUser: &user.User{
				Username: "newuser",
			},
			initial: []string{
				"PATH=/something/else",
			},
			expected: []string{
				"PATH=/something/else",
				"HOME=/home/newuser",
			},
		},
		{
			name: "HOME not set, user is set to root",
			user: "root",
			mockUser: &user.User{
				Username: "root",
			},
			initial: []string{
				"PATH=/something/else",
			},
			expected: []string{
				"PATH=/something/else",
				"HOME=/root",
			},
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			userLookup = func(username string) (*user.User, error) { return test.mockUser, nil }
			defer func() { userLookup = user.Lookup }()
			actual := addDefaultHOME(test.user, test.initial)
			testutil.CheckErrorAndDeepEqual(t, false, nil, test.expected, actual)
		})
	}
}
