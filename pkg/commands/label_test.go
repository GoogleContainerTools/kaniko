package commands

import (
	"github.com/GoogleCloudPlatform/k8s-container-builder/testutil"
	"github.com/containers/image/manifest"
	"github.com/docker/docker/builder/dockerfile/instructions"
	"testing"
)

func TestUpdateLabels(t *testing.T) {
	cfg := &manifest.Schema2Config{
		Labels: map[string]string {
			"foo": "bar",
		},
	}

	labels := []instructions.KeyValuePair{
		{
			Key: "foo",
			Value: "override",
		},
		{
			Key: "bar",
			Value: "baz",
		},
	}

	expectedLabels := map[string]string {
		"foo": "override",
		"bar": "baz",
	}
	updateLabels(labels, cfg)
	testutil.CheckErrorAndDeepEqual(t, false, nil, expectedLabels, cfg.Labels)
}
