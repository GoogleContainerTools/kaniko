package node

import (
	"errors"
	"fmt"
	"os"
	"sort"
	"text/tabwriter"

	"github.com/docker/swarmkit/api"
	"github.com/docker/swarmkit/api/genericresource"
	"github.com/docker/swarmkit/cmd/swarmctl/common"
	"github.com/docker/swarmkit/cmd/swarmctl/task"
	"github.com/dustin/go-humanize"
	"github.com/spf13/cobra"
)

func printNodeSummary(node *api.Node) {
	w := tabwriter.NewWriter(os.Stdout, 8, 8, 8, ' ', 0)
	defer func() {
		// Ignore flushing errors - there's nothing we can do.
		_ = w.Flush()
	}()
	spec := &node.Spec
	desc := node.Description
	if desc == nil {
		desc = &api.NodeDescription{}
	}
	common.FprintfIfNotEmpty(w, "ID\t: %s\n", node.ID)
	if node.Description != nil {
		common.FprintfIfNotEmpty(w, "Hostname\t: %s\n", node.Description.Hostname)
	}
	if len(spec.Annotations.Labels) != 0 {
		fmt.Fprint(w, "Node Labels\t:")
		// sort label output for readability
		var keys []string
		for k := range spec.Annotations.Labels {
			keys = append(keys, k)
		}
		sort.Strings(keys)
		for _, k := range keys {
			fmt.Fprintf(w, " %s=%s", k, spec.Annotations.Labels[k])
		}
		fmt.Fprintln(w)
	}
	fmt.Fprintln(w, "Status:\t")
	common.FprintfIfNotEmpty(w, "  State\t: %s\n", node.Status.State.String())
	common.FprintfIfNotEmpty(w, "  Message\t: %s\n", node.Status.Message)
	common.FprintfIfNotEmpty(w, "  Availability\t: %s\n", spec.Availability.String())
	common.FprintfIfNotEmpty(w, "  Address\t: %s\n", node.Status.Addr)

	if node.ManagerStatus != nil {
		fmt.Fprintln(w, "Manager status:\t")
		common.FprintfIfNotEmpty(w, "  Address\t: %s\n", node.ManagerStatus.Addr)
		common.FprintfIfNotEmpty(w, "  Raft status\t: %s\n", node.ManagerStatus.Reachability.String())
		leader := "no"
		if node.ManagerStatus.Leader {
			leader = "yes"
		}
		common.FprintfIfNotEmpty(w, "  Leader\t: %s\n", leader)
	}

	if desc.Platform != nil {
		fmt.Fprintln(w, "Platform:\t")
		common.FprintfIfNotEmpty(w, "  Operating System\t: %s\n", desc.Platform.OS)
		common.FprintfIfNotEmpty(w, "  Architecture\t: %s\n", desc.Platform.Architecture)
	}

	if desc.Resources != nil {
		fmt.Fprintln(w, "Resources:\t")
		fmt.Fprintf(w, "  CPUs\t: %d\n", desc.Resources.NanoCPUs/1e9)
		fmt.Fprintf(w, "  Memory\t: %s\n", humanize.IBytes(uint64(desc.Resources.MemoryBytes)))
		fmt.Fprintln(w, "  Generic Resources:\t")
		for _, r := range desc.Resources.Generic {
			k := genericresource.Kind(r)
			v := genericresource.Value(r)
			fmt.Fprintf(w, "    %s\t: %s\n", k, v)
		}
	}

	if desc.Engine != nil {
		fmt.Fprintln(w, "Plugins:\t")
		var pluginTypes []string
		pluginNamesByType := map[string][]string{}
		for _, p := range desc.Engine.Plugins {
			// append to pluginTypes only if not done previously
			if _, ok := pluginNamesByType[p.Type]; !ok {
				pluginTypes = append(pluginTypes, p.Type)
			}
			pluginNamesByType[p.Type] = append(pluginNamesByType[p.Type], p.Name)
		}

		sort.Strings(pluginTypes) // ensure stable output
		for _, pluginType := range pluginTypes {
			fmt.Fprintf(w, "  %s\t: %v\n", pluginType, pluginNamesByType[pluginType])
		}
	}

	if desc.Engine != nil {
		common.FprintfIfNotEmpty(w, "Engine Version\t: %s\n", desc.Engine.EngineVersion)

		if len(desc.Engine.Labels) != 0 {
			fmt.Fprint(w, "Engine Labels\t:")
			var keys []string
			for k := range desc.Engine.Labels {
				keys = append(keys, k)
			}
			sort.Strings(keys)
			for _, k := range keys {
				fmt.Fprintf(w, " %s=%s", k, desc.Engine.Labels[k])
			}
			fmt.Fprintln(w)
		}
	}
}

var (
	inspectCmd = &cobra.Command{
		Use:   "inspect <node ID>",
		Short: "Inspect a node",
		RunE: func(cmd *cobra.Command, args []string) error {
			if len(args) == 0 {
				return errors.New("node ID missing")
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

			node, err := getNode(common.Context(cmd), c, args[0])
			if err != nil {
				return err
			}

			r, err := c.ListTasks(common.Context(cmd),
				&api.ListTasksRequest{
					Filters: &api.ListTasksRequest_Filters{
						NodeIDs: []string{node.ID},
					},
				})
			if err != nil {
				return err
			}

			printNodeSummary(node)
			if len(r.Tasks) > 0 {
				fmt.Println()
				task.Print(r.Tasks, all, common.NewResolver(cmd, c))
			}

			return nil
		},
	}
)

func init() {
	inspectCmd.Flags().BoolP("all", "a", false, "Show all tasks (default shows just running)")
}
