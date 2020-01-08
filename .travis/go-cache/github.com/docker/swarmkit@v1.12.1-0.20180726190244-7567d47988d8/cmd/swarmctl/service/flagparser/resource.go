package flagparser

import (
	"fmt"
	"math/big"

	"github.com/docker/go-units"
	"github.com/docker/swarmkit/api"
	"github.com/docker/swarmkit/api/genericresource"
	"github.com/spf13/pflag"
)

func parseResourceCPU(flags *pflag.FlagSet, resources *api.Resources, name string) error {
	cpu, err := flags.GetString(name)
	if err != nil {
		return err
	}

	nanoCPUs, ok := new(big.Rat).SetString(cpu)
	if !ok {
		return fmt.Errorf("invalid cpu: %s", cpu)
	}
	cpuRat := new(big.Rat).Mul(nanoCPUs, big.NewRat(1e9, 1))
	if !cpuRat.IsInt() {
		return fmt.Errorf("CPU value cannot have more than 9 decimal places: %s", cpu)
	}
	resources.NanoCPUs = cpuRat.Num().Int64()
	return nil
}

func parseResourceMemory(flags *pflag.FlagSet, resources *api.Resources, name string) error {
	memory, err := flags.GetString(name)
	if err != nil {
		return err
	}

	bytes, err := units.RAMInBytes(memory)
	if err != nil {
		return err
	}

	resources.MemoryBytes = bytes
	return nil
}

func parseResource(flags *pflag.FlagSet, spec *api.ServiceSpec) error {
	if flags.Changed("memory-reservation") {
		if spec.Task.Resources == nil {
			spec.Task.Resources = &api.ResourceRequirements{}
		}
		if spec.Task.Resources.Reservations == nil {
			spec.Task.Resources.Reservations = &api.Resources{}
		}
		if err := parseResourceMemory(flags, spec.Task.Resources.Reservations, "memory-reservation"); err != nil {
			return err
		}
	}

	if flags.Changed("memory-limit") {
		if spec.Task.Resources == nil {
			spec.Task.Resources = &api.ResourceRequirements{}
		}
		if spec.Task.Resources.Limits == nil {
			spec.Task.Resources.Limits = &api.Resources{}
		}
		if err := parseResourceMemory(flags, spec.Task.Resources.Limits, "memory-limit"); err != nil {
			return err
		}
	}

	if flags.Changed("cpu-reservation") {
		if spec.Task.Resources == nil {
			spec.Task.Resources = &api.ResourceRequirements{}
		}
		if spec.Task.Resources.Reservations == nil {
			spec.Task.Resources.Reservations = &api.Resources{}
		}
		if err := parseResourceCPU(flags, spec.Task.Resources.Reservations, "cpu-reservation"); err != nil {
			return err
		}
	}

	if flags.Changed("cpu-limit") {
		if spec.Task.Resources == nil {
			spec.Task.Resources = &api.ResourceRequirements{}
		}
		if spec.Task.Resources.Limits == nil {
			spec.Task.Resources.Limits = &api.Resources{}
		}
		if err := parseResourceCPU(flags, spec.Task.Resources.Limits, "cpu-limit"); err != nil {
			return err
		}
	}

	if flags.Changed("generic-resources") {
		if spec.Task.Resources == nil {
			spec.Task.Resources = &api.ResourceRequirements{}
		}
		if spec.Task.Resources.Reservations == nil {
			spec.Task.Resources.Reservations = &api.Resources{}
		}

		cmd, err := flags.GetString("generic-resources")
		if err != nil {
			return err
		}
		spec.Task.Resources.Reservations.Generic, err = genericresource.ParseCmd(cmd)
		if err != nil {
			return err
		}
		err = genericresource.ValidateTask(spec.Task.Resources.Reservations)
		if err != nil {
			return err
		}
	}

	return nil
}
