package test

import (
	"os/exec"
	"testing"
)

func Test_Exec(t *testing.T) {
	imageName := "gcr.io/kaniko-dev/kaniko-dockerfile-test-user-home"

	cmd := exec.Command(
		"docker",
		"run",
		"--rm",
		"--entrypoint", "",
		imageName,
		"/bin/sh",
		"-c",
		"echo $HOME",
	)

	out, err := cmd.CombinedOutput()

	if err != nil {
		t.Errorf("%s: %s %s", cmd, err, out)
	}
}
