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

	"github.com/GoogleContainerTools/kaniko/testutil"
	"github.com/google/go-containerregistry/pkg/v1"
	"github.com/moby/buildkit/frontend/dockerfile/instructions"
)

var copyTests = []struct {
	SourcesAndDest      []string
	expectedDest        []string
}{
	{
		SourcesAndDest:   []string{"/usr/bin/bash", "/tmp"},
		expectedDest:     []string{"/tmp/bash"},
	},
	{
		SourcesAndDest:   []string{"/usr/bin/bash", "/tmp/"},
		expectedDest:     []string{"/tmp/bash"},
	},
}

func TestCopyExecuteCmd(t *testing.T) {

	cfg := &v1.Config{
		Cmd: nil,
	}

	for _, test := range copyTests {
		cmd := CopyCommand{
			cmd: &instructions.CopyCommand{
				SourcesAndDest: instructions.SourcesAndDest{
					SourcesAndDest:      test.SourcesAndDest,
			},
		}
		err := cmd.ExecuteCommand(cfg, nil)
		testutil.CheckErrorAndDeepEqual(t, false, err, test.expectedDest, cfg.WorkingDir)
	}
}
