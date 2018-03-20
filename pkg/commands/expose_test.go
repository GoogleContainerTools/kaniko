package commands

import (
	"github.com/GoogleCloudPlatform/k8s-container-builder/testutil"
	"github.com/containers/image/manifest"
	"testing"
)

func TestUpdateExposedPorts(t *testing.T) {
	cfg := &manifest.Schema2Config{
		ExposedPorts: manifest.Schema2PortSet{
			"8080/tcp": {},
		},
	}

	ports := []string{
		"8080",
		"8081/tcp",
		"8082",
		"8083/udp",
	}

	expectedPorts := manifest.Schema2PortSet{
		"8080/tcp": {},
		"8081/tcp": {},
		"8082/tcp": {},
		"8083/udp": {},
	}

	updateExposedPorts(ports, cfg)
	testutil.CheckErrorAndDeepEqual(t, false, nil, expectedPorts, cfg.ExposedPorts)
}
