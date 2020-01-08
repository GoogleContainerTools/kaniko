package flagparser

import (
	"fmt"
	"strings"

	"github.com/docker/swarmkit/api"
	"github.com/spf13/pflag"
)

// parseVolume only supports a very simple version of anonymous volumes for
// testing the most basic of data flows. Replace with a --mount flag, similar
// to what we have in docker service.
func parseVolume(flags *pflag.FlagSet, spec *api.ServiceSpec) error {
	if flags.Changed("volume") {
		volumes, err := flags.GetStringSlice("volume")
		if err != nil {
			return err
		}

		container := spec.Task.GetContainer()

		for _, volume := range volumes {
			if strings.Contains(volume, ":") {
				return fmt.Errorf("volume format %q not supported", volume)
			}
			container.Mounts = append(container.Mounts, api.Mount{
				Type:   api.MountTypeVolume,
				Target: volume,
			})
		}
	}

	return nil
}
