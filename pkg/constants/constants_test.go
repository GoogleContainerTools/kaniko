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

package constants

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/GoogleContainerTools/kaniko/testutil"
)

func Test_SettingEnv(t *testing.T) {
	tests := []struct {
		description                 string
		envVariable                 string
		rootDir                     string
		kanikoDir                   string
		dockerfile                  string
		buildContextDir             string
		kanikoIntermediateStagesDir string
	}{
		{
			description:                 "env variable not set",
			rootDir:                     "/",
			kanikoDir:                   "/kaniko",
			dockerfile:                  "/kaniko/Dockerfile",
			buildContextDir:             "/kaniko/buildcontext",
			kanikoIntermediateStagesDir: "/kaniko/stages",
		},
		{
			description:                 "env variable set with absolute path",
			envVariable:                 "/test",
			rootDir:                     "/test",
			kanikoDir:                   "/test/kaniko",
			dockerfile:                  "/test/kaniko/Dockerfile",
			buildContextDir:             "/test/kaniko/buildcontext",
			kanikoIntermediateStagesDir: "/test/kaniko/stages",
		},
		{
			description:                 "env variable set with ending /",
			envVariable:                 "/test/",
			rootDir:                     "/test",
			kanikoDir:                   "/test/kaniko",
			dockerfile:                  "/test/kaniko/Dockerfile",
			buildContextDir:             "/test/kaniko/buildcontext",
			kanikoIntermediateStagesDir: "/test/kaniko/stages",
		},
		{
			description:                 "env variable set with relative path",
			envVariable:                 "./..//test",
			rootDir:                     "../test",
			kanikoDir:                   "../test/kaniko",
			dockerfile:                  "../test/kaniko/Dockerfile",
			buildContextDir:             "../test/kaniko/buildcontext",
			kanikoIntermediateStagesDir: "../test/kaniko/stages",
		},
	}
	for _, test := range tests {
		t.Run(test.description, func(t *testing.T) {
			original := os.Getenv(KanikoRootEnv)
			defer func() {
				setEnvAndRevalGlobal(original)
			}()

			setEnvAndRevalGlobal(test.envVariable)
			testutil.CheckDeepEqual(t, RootDir, test.rootDir)
			testutil.CheckDeepEqual(t, KanikoDir, test.kanikoDir)
			testutil.CheckDeepEqual(t, DockerfilePath, test.dockerfile)
			testutil.CheckDeepEqual(t, BuildContextDir, test.buildContextDir)
			testutil.CheckDeepEqual(t, KanikoIntermediateStagesDir, test.kanikoIntermediateStagesDir)
		})
	}
}

func setEnvAndRevalGlobal(env string) {
	os.Setenv(KanikoRootEnv, env)
	RootDir = filepath.Join(os.Getenv(KanikoRootEnv), "/")

	//KanikoDir is the path to the Kaniko directory
	KanikoDir = filepath.Join(os.Getenv(KanikoRootEnv), "/kaniko")

	// DockerfilePath is the path the Dockerfile is copied to
	DockerfilePath = filepath.Join(os.Getenv(KanikoRootEnv), "/kaniko/Dockerfile")

	// BuildContextDir is the directory a build context will be unpacked into,
	// for example, a tarball from a GCS bucket will be unpacked here
	BuildContextDir = filepath.Join(os.Getenv(KanikoRootEnv), "/kaniko/buildcontext")

	// KanikoIntermediateStagesDir is where we will store intermediate stages
	// as tarballs in case they are needed later on
	KanikoIntermediateStagesDir = filepath.Join(os.Getenv(KanikoRootEnv), "/kaniko/stages")
}
