package commands

import (
	"github.com/containers/image/manifest"
	"github.com/docker/docker/builder/dockerfile/instructions"
	"strings"
)

type ExposeCommand struct {
	cmd *instructions.ExposeCommand
}

func (r *ExposeCommand) ExecuteCommand(config *manifest.Schema2Config) error {
	return updateExposedPorts(r.Ports, config)
}

func updateExposedPorts(ports []string, config *manifest.Schema2Config) error {
	// Grab the currently exposed ports
	existingPorts := config.ExposedPorts

	// Add any new ones in
	for _, p := range ports {
		// Add the default protocol if one isn't specified
		if !strings.Contains(p, "/") {
			p = p + "/tcp"
		}
		existingPorts[p] = {}
	}
	config.ExposedPorts = existingPorts
	return nil
}

func (r *ExposeCommand) FilesToSnapshot() []string {
	return []string{}
}

func (r *ExposeCommand) CreatedBy() string {
	s := []string{"/bin/sh", "-c"}
	return strings.Join(append(s, r.Ports...), " ")
}
