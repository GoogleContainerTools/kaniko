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
	v1 "github.com/google/go-containerregistry/pkg/v1"
	"github.com/moby/buildkit/frontend/dockerfile/instructions"
)

var stopsignalTests = []struct {
	signal         string
	expectedSignal string
}{
	{
		signal:         "SIGKILL",
		expectedSignal: "SIGKILL",
	},
	{
		signal:         "${STOPSIG}",
		expectedSignal: "SIGKILL",
	},
	{
		signal:         "1",
		expectedSignal: "1",
	},
}

func TestStopsignalExecuteCmd(t *testing.T) {

	cfg := &v1.Config{
		StopSignal: "",
		Env:        []string{"STOPSIG=SIGKILL"},
	}

	for _, test := range stopsignalTests {
		cmd := StopSignalCommand{
			cmd: &instructions.StopSignalCommand{
				Signal: test.signal,
			},
		}
		b := dockerfile.NewBuildArgs([]string{})
		err := cmd.ExecuteCommand(cfg, b)
		testutil.CheckErrorAndDeepEqual(t, false, err, test.expectedSignal, cfg.StopSignal)
	}
}
