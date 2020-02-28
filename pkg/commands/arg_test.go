/*
Copyright 2020 Google LLC

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
	"github.com/GoogleContainerTools/kaniko/pkg/dockerfile"
	"github.com/GoogleContainerTools/kaniko/testutil"
	v1 "github.com/google/go-containerregistry/pkg/v1"
	"github.com/moby/buildkit/frontend/dockerfile/instructions"
	"testing"
)

func Test_ArgExecute(t *testing.T) {
	tests := []struct {
		name         string
		argCmd       *ArgCommand
		expectedEnvs []string
	}{
		{
			name: "arg command",
			argCmd: &ArgCommand{
				cmd: &instructions.ArgCommand{
					KeyValuePairOptional: instructions.KeyValuePairOptional{
						Key:   "arg",
						Value: testutil.StringPtr("val"),
					},
				},
			},
			expectedEnvs: []string{
				"path=/usr",
				"home=/root",
				"arg=val",
			},
		},
		{
			name: "arg command with env replacement",
			argCmd: &ArgCommand{
				cmd: &instructions.ArgCommand{
					KeyValuePairOptional: instructions.KeyValuePairOptional{
						Key:   "arg",
						Value: testutil.StringPtr("$home/path"),
					},
				},
			},
			expectedEnvs: []string{
				"path=/usr",
				"home=/root",
				"arg=/root/path",
			},
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			cfg := &v1.Config{
				Env: []string{
					"path=/usr",
					"home=/root",
				},
			}
			err := test.argCmd.ExecuteCommand(cfg, dockerfile.NewBuildArgs([]string{}))
			testutil.CheckErrorAndDeepEqual(t, false, err, test.expectedEnvs, cfg.Env)
		})
	}
}
