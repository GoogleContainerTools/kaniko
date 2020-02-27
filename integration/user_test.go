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

package integration

import (
	"io/ioutil"
	"os"
	"os/exec"
	"path/filepath"
	"reflect"
	"testing"
)

func Test_UserHome(t *testing.T) {
	builder := NewDockerFileBuilder()

	testDir, err := ioutil.TempDir("", "")
	if err != nil {
		t.Fatal(err)
	}

	defer os.RemoveAll(testDir)

	dockerfileContent := `
FROM alpine
RUN adduser -S -D myuser

USER myuser
`

	dockerfileName := "Dockerfile-test-user-home"
	dockerfilePath := filepath.Join(testDir, dockerfileName)

	if err := ioutil.WriteFile(dockerfilePath, []byte(dockerfileContent), 0777); err != nil {
		t.Fatal(err)
	}

	if err := builder.BuildImageWithContext(
		config, testDir, dockerfileName, testDir,
	); err != nil {
		t.Fatal(err)
	}

	kanikoImageName := GetKanikoImage(config.imageRepo, dockerfileName)
	dockerImageName := GetDockerImage(config.imageRepo, dockerfileName)

	userCmd := func(imageName string) *exec.Cmd {
		return exec.Command(
			"docker",
			"run",
			"--rm",
			"--entrypoint", "",
			imageName,
			"/bin/sh",
			"-c",
			"echo $HOME",
		)
	}

	kanikoOut, err := RunCommandWithoutTest(userCmd(kanikoImageName))
	if err != nil {
		t.Fatalf("%s %s: %s", userCmd(kanikoImageName), err, kanikoOut)
	}

	dockerOut, err := RunCommandWithoutTest(userCmd(dockerImageName))
	if err != nil {
		t.Fatalf("%s %s: %s", userCmd(dockerImageName), err, dockerOut)
	}

	if !reflect.DeepEqual(kanikoOut, dockerOut) {
		t.Errorf("want\n%+v\ngot\n%+v", dockerOut, kanikoOut)
	}
}
