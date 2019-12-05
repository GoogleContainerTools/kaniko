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

func TestUpdateLabels(t *testing.T) {
	cfg := &v1.Config{
		Labels: map[string]string{
			"foo": "bar",
		},
	}

	labels := []instructions.KeyValuePair{
		{
			Key:   "foo",
			Value: "override",
		},
		{
			Key:   "bar",
			Value: "baz",
		},
		{
			Key:   "multiword",
			Value: "lots\\ of\\ words",
		},
		{
			Key:   "backslashes",
			Value: "lots\\\\ of\\\\ words",
		},
		{
			Key:   "$label",
			Value: "foo",
		},
	}

	arguments := []string{
		"label=build_arg_label",
	}

	buildArgs := dockerfile.NewBuildArgs(arguments)
	buildArgs.AddArg("label", nil)
	expectedLabels := map[string]string{
		"foo":             "override",
		"bar":             "baz",
		"multiword":       "lots of words",
		"backslashes":     "lots\\ of\\ words",
		"build_arg_label": "foo",
	}
	updateLabels(labels, cfg, buildArgs)
	testutil.CheckErrorAndDeepEqual(t, false, nil, expectedLabels, cfg.Labels)
}
