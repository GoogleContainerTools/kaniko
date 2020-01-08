package flagparser

import (
	"fmt"
	"strings"

	"github.com/docker/swarmkit/api"
	"github.com/docker/swarmkit/cmd/swarmctl/common"
	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
)

// AddServiceFlags add all supported service flags to the flagset.
func AddServiceFlags(flags *pflag.FlagSet) {
	flags.String("name", "", "service name")
	flags.StringSlice("label", nil, "service label (key=value)")

	flags.Uint64("replicas", 1, "number of replicas for the service (only works in replicated service mode)")

	flags.String("image", "", "container image")
	flags.String("hostname", "", "container hostname")
	flags.StringSlice("command", nil, "override entrypoint")
	flags.StringSlice("args", nil, "container args")
	flags.StringSlice("env", nil, "container env")
	flags.Bool("tty", false, "open a tty on standard streams")
	flags.Bool("open-stdin", false, "open standard input")

	flags.StringSlice("ports", nil, "ports")
	flags.String("network", "", "network name")

	flags.String("memory-reservation", "", "amount of reserved memory (e.g. 512m)")
	flags.String("memory-limit", "", "memory limit (e.g. 512m)")
	flags.String("cpu-reservation", "", "number of CPU cores reserved (e.g. 0.5)")
	flags.String("cpu-limit", "", "CPU cores limit (e.g. 0.5)")
	flags.String("generic-resources", "", "user defined resources request (e.g. gpu=3,fpga=1)")

	flags.Uint64("update-parallelism", 0, "task update parallelism (0 = all at once)")
	flags.String("update-delay", "0s", "delay between task updates (0s = none)")
	flags.String("update-on-failure", "pause", "action on failure during update (pause|continue|rollback)")
	flags.String("update-order", "stop-first", "order of shutting down old task and starting updated task (stop-first|start-first)")

	flags.Uint64("rollback-parallelism", 0, "task update parallelism during rollback (0 = all at once)")
	flags.String("rollback-delay", "0s", "delay between task updates during rollback (0s = none)")
	flags.String("rollback-on-failure", "pause", "action on failure during rollback (pause|continue)")
	flags.String("rollback-order", "stop-first", "order of shutting down old task and starting rolled-back task (stop-first|start-first)")

	flags.String("restart-condition", "any", "condition to restart the task (any, failure, none)")
	flags.String("restart-delay", "5s", "delay between task restarts")
	flags.Uint64("restart-max-attempts", 0, "maximum number of restart attempts (0 = unlimited)")
	flags.String("restart-window", "0s", "time window to evaluate restart attempts (0 = unbound)")

	flags.StringSlice("constraint", nil, "Placement constraint (e.g. node.labels.key==value)")

	// TODO(stevvooe): Replace these with a more interesting mount flag.
	flags.StringSlice("bind", nil, "define a bind mount")
	flags.StringSlice("volume", nil, "define a volume mount")
	flags.StringSlice("tmpfs", nil, "define a tmpfs mount")

	flags.String("log-driver", "", "specify a log driver")
	flags.StringSlice("log-opt", nil, "log driver options, as key value pairs")

	flags.Bool("init", false, "Run an init inside the container that forwards signals and reaps processes")
}

// Merge merges a flagset into a service spec.
func Merge(cmd *cobra.Command, spec *api.ServiceSpec, c api.ControlClient) error {
	flags := cmd.Flags()

	if flags.Changed("force") {
		force, err := flags.GetBool("force")
		if err != nil {
			return err
		}
		if force {
			spec.Task.ForceUpdate++
		}
	}

	if flags.Changed("name") {
		name, err := flags.GetString("name")
		if err != nil {
			return err
		}
		spec.Annotations.Name = name
	}

	if flags.Changed("label") {
		labels, err := flags.GetStringSlice("label")
		if err != nil {
			return err
		}
		spec.Annotations.Labels = map[string]string{}
		for _, l := range labels {
			parts := strings.SplitN(l, "=", 2)
			if len(parts) != 2 {
				return fmt.Errorf("malformed label: %s", l)
			}
			spec.Annotations.Labels[strings.TrimSpace(parts[0])] = strings.TrimSpace(parts[1])
		}
	}

	if err := parseMode(flags, spec); err != nil {
		return err
	}

	if err := parseContainer(flags, spec); err != nil {
		return err
	}

	if err := parseResource(flags, spec); err != nil {
		return err
	}

	if err := parsePorts(flags, spec); err != nil {
		return err
	}

	if err := parseNetworks(cmd, spec, c); err != nil {
		return err
	}

	if err := parseRestart(flags, spec); err != nil {
		return err
	}

	if err := parseUpdate(flags, spec); err != nil {
		return err
	}

	if err := parsePlacement(flags, spec); err != nil {
		return err
	}

	if err := parseBind(flags, spec); err != nil {
		return err
	}

	if err := parseVolume(flags, spec); err != nil {
		return err
	}

	if err := parseTmpfs(flags, spec); err != nil {
		return err
	}

	driver, err := common.ParseLogDriverFlags(flags)
	if err != nil {
		return err
	}
	spec.Task.LogDriver = driver

	return nil
}
