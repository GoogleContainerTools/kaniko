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
	"github.com/GoogleCloudPlatform/kaniko/testutil"
	"github.com/containers/image/manifest"
	"github.com/docker/docker/builder/dockerfile/instructions"
	"testing"
)

func TestUpdateLabels(t *testing.T) {
	cfg := &manifest.Schema2Config{
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
	}

	expectedLabels := map[string]string{
		"foo":         "override",
		"bar":         "baz",
		"multiword":   "lots of words",
		"backslashes": "lots\\ of\\ words",
	}
	updateLabels(labels, cfg)
	testutil.CheckErrorAndDeepEqual(t, false, nil, expectedLabels, cfg.Labels)
}
