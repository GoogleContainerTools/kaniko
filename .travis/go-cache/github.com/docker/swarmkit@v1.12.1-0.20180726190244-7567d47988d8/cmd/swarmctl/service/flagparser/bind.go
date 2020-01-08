package flagparser

import (
	"fmt"
	"strings"

	"github.com/docker/swarmkit/api"
	"github.com/spf13/pflag"
)

// parseBind only supports a very simple version of bind for testing the most
// basic of data flows. Replace with a --mount flag, similar to what we have in
// docker service.
func parseBind(flags *pflag.FlagSet, spec *api.ServiceSpec) error {
	if flags.Changed("bind") {
		binds, err := flags.GetStringSlice("bind")
		if err != nil {
			return err
		}

		container := spec.Task.GetContainer()

		for _, bind := range binds {
			parts := strings.SplitN(bind, ":", 2)
			if len(parts) != 2 {
				return fmt.Errorf("bind format %q not supported", bind)
			}
			container.Mounts = append(container.Mounts, api.Mount{
				Type:   api.MountTypeBind,
				Source: parts[0],
				Target: parts[1],
			})
		}
	}

	return nil
}
