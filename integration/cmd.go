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

package integration

import (
	"bytes"
	"os/exec"
	"testing"
)

// RunCommandWithoutTest will run cmd and if it fails will output relevant info
// for debugging before returning an error. It can be run outside the context of a test.
func RunCommandWithoutTest(cmd *exec.Cmd) ([]byte, error) {
	output, err := cmd.CombinedOutput()
	return output, err
}

// RunCommand will run cmd and if it fails will output relevant info for debugging
// before it fails. It must be run within the context of a test t and if the command
// fails, it will fail the test. Returns the output from the command.
func RunCommand(cmd *exec.Cmd, t *testing.T) []byte {
	var stderr bytes.Buffer
	cmd.Stderr = &stderr
	output, err := cmd.Output()
	if err != nil {
		t.Log(cmd.Args)
		t.Log(stderr.String())
		t.Log(string(output))
		t.Error(err)
		t.FailNow()
	}
	return output
}
