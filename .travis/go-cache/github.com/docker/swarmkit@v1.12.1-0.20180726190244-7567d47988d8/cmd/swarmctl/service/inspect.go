package service

import (
	"errors"
	"fmt"
	"io"
	"os"
	"sort"
	"strings"
	"text/tabwriter"

	"github.com/docker/swarmkit/api"
	"github.com/docker/swarmkit/api/genericresource"
	"github.com/docker/swarmkit/cmd/swarmctl/common"
	"github.com/docker/swarmkit/cmd/swarmctl/task"
	"github.com/dustin/go-humanize"
	gogotypes "github.com/gogo/protobuf/types"
	"github.com/spf13/cobra"
)

func printServiceSummary(service *api.Service, running int) {
	w := tabwriter.NewWriter(os.Stdout, 8, 8, 8, ' ', 0)
	defer w.Flush()

	task := service.Spec.Task
	common.FprintfIfNotEmpty(w, "ID\t: %s\n", service.ID)
	common.FprintfIfNotEmpty(w, "Name\t: %s\n", service.Spec.Annotations.Name)
	if len(service.Spec.Annotations.Labels) > 0 {
		fmt.Fprintln(w, "Labels\t")
		for k, v := range service.Spec.Annotations.Labels {
			fmt.Fprintf(w, "  %s\t: %s\n", k, v)
		}
	}
	common.FprintfIfNotEmpty(w, "Replicas\t: %s\n", getServiceReplicasTxt(service, running))

	if service.UpdateStatus != nil {
		fmt.Fprintln(w, "Update Status\t")
		fmt.Fprintln(w, " State\t:", service.UpdateStatus.State)
		started, err := gogotypes.TimestampFromProto(service.UpdateStatus.StartedAt)
		if err == nil {
			fmt.Fprintln(w, " Started\t:", humanize.Time(started))
		}
		if service.UpdateStatus.State == api.UpdateStatus_COMPLETED {
			completed, err := gogotypes.TimestampFromProto(service.UpdateStatus.CompletedAt)
			if err == nil {
				fmt.Fprintln(w, " Completed\t:", humanize.Time(completed))
			}
		}
		fmt.Fprintln(w, " Message\t:", service.UpdateStatus.Message)
	}

	fmt.Fprintln(w, "Template\t")
	fmt.Fprintln(w, " Container\t")
	ctr := service.Spec.Task.GetContainer()
	common.FprintfIfNotEmpty(w, "  Image\t: %s\n", ctr.Image)
	common.FprintfIfNotEmpty(w, "  Command\t: %q\n", strings.Join(ctr.Command, " "))
	common.FprintfIfNotEmpty(w, "  Args\t: [%s]\n", strings.Join(ctr.Args, ", "))
	common.FprintfIfNotEmpty(w, "  Env\t: [%s]\n", strings.Join(ctr.Env, ", "))
	if task.Placement != nil {
		common.FprintfIfNotEmpty(w, "  Constraints\t: %s\n", strings.Join(task.Placement.Constraints, ", "))
	}

	if task.Resources != nil {
		res := task.Resources
		fmt.Fprintln(w, "  Resources\t")
		printResources := func(w io.Writer, r *api.Resources) {
			if r.NanoCPUs != 0 {
				fmt.Fprintf(w, "      CPU\t: %g\n", float64(r.NanoCPUs)/1e9)
			}
			if r.MemoryBytes != 0 {
				fmt.Fprintf(w, "      Memory\t: %s\n", humanize.IBytes(uint64(r.MemoryBytes)))
			}
			if len(r.Generic) != 0 {
				fmt.Fprintln(w, "      Generic Resources\t")
			}

			for _, r := range r.Generic {
				k := genericresource.Kind(r)
				v := genericresource.Value(r)
				fmt.Fprintf(w, "        %s\t: %s\n", k, v)
			}

		}
		if res.Reservations != nil {
			fmt.Fprintln(w, "    Reservations:\t")
			printResources(w, res.Reservations)
		}
		if res.Limits != nil {
			fmt.Fprintln(w, "    Limits:\t")
			printResources(w, res.Limits)
		}
	}
	if len(service.Spec.Task.Networks) > 0 {
		fmt.Fprint(w, "  Networks:")
		for _, n := range service.Spec.Task.Networks {
			fmt.Fprintf(w, " %s", n.Target)
		}
	}

	if service.Endpoint != nil && len(service.Endpoint.Ports) > 0 {
		fmt.Fprintln(w, "\nPorts:")
		for _, port := range service.Endpoint.Ports {
			fmt.Fprintf(w, "    - Name\t= %s\n", port.Name)
			fmt.Fprintf(w, "      Protocol\t= %s\n", port.Protocol)
			fmt.Fprintf(w, "      Port\t= %d\n", port.TargetPort)
			fmt.Fprintf(w, "      SwarmPort\t= %d\n", port.PublishedPort)
		}
	}

	if len(ctr.Mounts) > 0 {
		fmt.Fprintln(w, "  Mounts:")
		for _, v := range ctr.Mounts {
			fmt.Fprintf(w, "    - target = %s\n", v.Target)
			fmt.Fprintf(w, "      source = %s\n", v.Source)
			fmt.Fprintf(w, "      readonly = %v\n", v.ReadOnly)
			fmt.Fprintf(w, "      type = %v\n", strings.ToLower(v.Type.String()))
		}
	}

	if len(ctr.Secrets) > 0 {
		fmt.Fprintln(w, "  Secrets:")
		for _, sr := range ctr.Secrets {
			var targetName, mode string
			if sr.GetFile() != nil {
				targetName = sr.GetFile().Name
				mode = "FILE"
			}
			fmt.Fprintf(w, "    [%s] %s@%s:%s\n", mode, sr.SecretName, sr.SecretID, targetName)
		}
	}

	if len(ctr.Configs) > 0 {
		fmt.Fprintln(w, "  Configs:")
		for _, cr := range ctr.Configs {
			var targetName, mode string
			if cr.GetFile() != nil {
				targetName = cr.GetFile().Name
				mode = "FILE"
			}
			fmt.Fprintf(w, "    [%s] %s@%s:%s\n", mode, cr.ConfigName, cr.ConfigID, targetName)
		}
	}

	if task.LogDriver != nil {
		fmt.Fprintf(w, "  LogDriver\t: %s\n", task.LogDriver.Name)
		var keys []string

		if task.LogDriver.Options != nil {
			for k := range task.LogDriver.Options {
				keys = append(keys, k)
			}
			sort.Strings(keys)

			for _, k := range keys {
				v := task.LogDriver.Options[k]
				if v != "" {
					fmt.Fprintf(w, "    %s\t: %s\n", k, v)
				} else {
					fmt.Fprintf(w, "    %s\t\n", k)

				}
			}
		}
	}
}

var (
	inspectCmd = &cobra.Command{
		Use:   "inspect <service ID>",
		Short: "Inspect a service",
		RunE: func(cmd *cobra.Command, args []string) error {
			if len(args) == 0 {
				return errors.New("service ID missing")
			}

			if len(args) > 1 {
				return errors.New("inspect command takes exactly 1 argument")
			}

			flags := cmd.Flags()

			all, err := flags.GetBool("all")
			if err != nil {
				return err
			}

			c, err := common.Dial(cmd)
			if err != nil {
				return err
			}

			res := common.NewResolver(cmd, c)

			service, err := getService(common.Context(cmd), c, args[0])
			if err != nil {
				return err
			}

			r, err := c.ListTasks(common.Context(cmd),
				&api.ListTasksRequest{
					Filters: &api.ListTasksRequest_Filters{
						ServiceIDs: []string{service.ID},
					},
				})
			if err != nil {
				return err
			}
			var running int
			for _, t := range r.Tasks {
				if t.Status.State == api.TaskStateRunning {
					running++
				}
			}

			printServiceSummary(service, running)
			if len(r.Tasks) > 0 {
				fmt.Println()
				task.Print(r.Tasks, all, res)
			}

			return nil
		},
	}
)

func init() {
	inspectCmd.Flags().BoolP("all", "a", false, "Show all tasks (default shows just running)")
}
