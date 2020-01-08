package flagparser

import (
	"github.com/docker/swarmkit/api"
	"github.com/spf13/pflag"
)

func parsePlacement(flags *pflag.FlagSet, spec *api.ServiceSpec) error {
	if flags.Changed("constraint") {
		constraints, err := flags.GetStringSlice("constraint")
		if err != nil {
			return err
		}
		if spec.Task.Placement == nil {
			spec.Task.Placement = &api.Placement{}
		}
		spec.Task.Placement.Constraints = constraints
	}

	return nil
}
