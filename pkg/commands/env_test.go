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
	"github.com/GoogleContainerTools/kaniko/pkg/dockerfile"
	"github.com/GoogleContainerTools/kaniko/testutil"
	"github.com/docker/docker/builder/dockerfile/instructions"
	"github.com/google/go-containerregistry/v1"
	"testing"
)

func TestUpdateEnvConfig(t *testing.T) {
	cfg := &v1.Config{
		Env: []string{
			"PATH=/path/to/dir",
			"hey=hey",
		},
	}

	newEnvs := []instructions.KeyValuePair{
		{
			Key:   "foo",
			Value: "foo2",
		},
		{
			Key:   "PATH",
			Value: "/new/path/",
		},
		{
			Key:   "foo",
			Value: "newfoo",
		},
	}

	expectedEnvArray := []string{
		"PATH=/new/path/",
		"hey=hey",
		"foo=newfoo",
	}
	updateConfigEnv(newEnvs, cfg)
	testutil.CheckErrorAndDeepEqual(t, false, nil, expectedEnvArray, cfg.Env)
}
func Test_EnvExecute(t *testing.T) {
	cfg := &v1.Config{
		Env: []string{
			"path=/usr/",
			"home=/root",
		},
	}

	envCmd := &EnvCommand{
		&instructions.EnvCommand{
			Env: []instructions.KeyValuePair{
				{
					Key:   "path",
					Value: "/some/path",
				},
				{
					Key:   "HOME",
					Value: "$home",
				},
				{
					Key:   "$path",
					Value: "$home/",
				},
				{
					Key:   "$buildArg1",
					Value: "$buildArg2",
				},
			},
		},
	}

	expectedEnvs := []string{
		"path=/some/path",
		"home=/root",
		"HOME=/root",
		"/usr/=/root/",
		"foo=foo2",
	}
	buildArgs := setUpBuildArgs()
	err := envCmd.ExecuteCommand(cfg, buildArgs)
	testutil.CheckErrorAndDeepEqual(t, false, err, expectedEnvs, cfg.Env)
}

func setUpBuildArgs() *dockerfile.BuildArgs {
	buildArgs := dockerfile.NewBuildArgs([]string{
		"buildArg1=foo",
		"buildArg2=foo2",
	})
	buildArgs.AddArg("buildArg1", nil)
	d := "default"
	buildArgs.AddArg("buildArg2", &d)
	return buildArgs
}
