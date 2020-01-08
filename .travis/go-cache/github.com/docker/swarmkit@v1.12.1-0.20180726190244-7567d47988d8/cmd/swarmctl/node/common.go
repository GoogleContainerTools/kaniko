package node

import (
	"errors"
	"fmt"
	"reflect"
	"strings"

	"golang.org/x/net/context"

	"github.com/docker/swarmkit/api"
	"github.com/docker/swarmkit/cmd/swarmctl/common"
	"github.com/spf13/cobra"
)

var (
	errNoChange = errors.New("node attribute was already set to the requested value")
	flagLabel   = "label"
)

func changeNodeAvailability(cmd *cobra.Command, args []string, availability api.NodeSpec_Availability) error {
	if len(args) == 0 {
		return errors.New("missing node ID")
	}

	if len(args) > 1 {
		return errors.New("command takes exactly 1 argument")
	}

	c, err := common.Dial(cmd)
	if err != nil {
		return err
	}
	node, err := getNode(common.Context(cmd), c, args[0])
	if err != nil {
		return err
	}
	spec := &node.Spec

	if spec.Availability == availability {
		return errNoChange
	}

	spec.Availability = availability

	_, err = c.UpdateNode(common.Context(cmd), &api.UpdateNodeRequest{
		NodeID:      node.ID,
		NodeVersion: &node.Meta.Version,
		Spec:        spec,
	})

	if err != nil {
		return err
	}

	return nil
}

func changeNodeMembership(cmd *cobra.Command, args []string, membership api.NodeSpec_Membership) error {
	if len(args) == 0 {
		return errors.New("missing node ID")
	}

	if len(args) > 1 {
		return errors.New("command takes exactly 1 argument")
	}

	c, err := common.Dial(cmd)
	if err != nil {
		return err
	}
	node, err := getNode(common.Context(cmd), c, args[0])
	if err != nil {
		return err
	}
	spec := &node.Spec

	if spec.Membership == membership {
		return errNoChange
	}

	spec.Membership = membership

	_, err = c.UpdateNode(common.Context(cmd), &api.UpdateNodeRequest{
		NodeID:      node.ID,
		NodeVersion: &node.Meta.Version,
		Spec:        spec,
	})

	if err != nil {
		return err
	}

	return nil
}

func changeNodeRole(cmd *cobra.Command, args []string, role api.NodeRole) error {
	if len(args) == 0 {
		return errors.New("missing node ID")
	}

	if len(args) > 1 {
		return errors.New("command takes exactly 1 argument")
	}

	c, err := common.Dial(cmd)
	if err != nil {
		return err
	}
	node, err := getNode(common.Context(cmd), c, args[0])
	if err != nil {
		return err
	}
	spec := &node.Spec

	if spec.DesiredRole == role {
		return errNoChange
	}

	spec.DesiredRole = role

	_, err = c.UpdateNode(common.Context(cmd), &api.UpdateNodeRequest{
		NodeID:      node.ID,
		NodeVersion: &node.Meta.Version,
		Spec:        spec,
	})

	if err != nil {
		return err
	}

	return nil
}

func getNode(ctx context.Context, c api.ControlClient, input string) (*api.Node, error) {
	// GetNode to match via full ID.
	rg, err := c.GetNode(ctx, &api.GetNodeRequest{NodeID: input})
	if err != nil {
		// If any error (including NotFound), ListServices to match via full name.
		rl, err := c.ListNodes(ctx,
			&api.ListNodesRequest{
				Filters: &api.ListNodesRequest_Filters{
					Names: []string{input},
				},
			},
		)
		if err != nil {
			return nil, err
		}

		if len(rl.Nodes) == 0 {
			return nil, fmt.Errorf("node %s not found", input)
		}

		if l := len(rl.Nodes); l > 1 {
			return nil, fmt.Errorf("node %s is ambiguous (%d matches found)", input, l)
		}

		return rl.Nodes[0], nil
	}
	return rg.Node, nil
}

func updateNode(cmd *cobra.Command, args []string) error {
	if len(args) == 0 {
		return errors.New("node ID missing")
	}

	if len(args) > 1 {
		return errors.New("command takes exactly 1 argument")
	}

	c, err := common.Dial(cmd)
	if err != nil {
		return err
	}

	node, err := getNode(common.Context(cmd), c, args[0])
	if err != nil {
		return err
	}
	spec := node.Spec.Copy()

	flags := cmd.Flags()
	if flags.Changed(flagLabel) {
		labels, err := flags.GetStringSlice(flagLabel)
		if err != nil {
			return err
		}
		// overwrite existing labels
		spec.Annotations.Labels = map[string]string{}
		for _, l := range labels {
			parts := strings.SplitN(l, "=", 2)
			if len(parts) != 2 {
				return fmt.Errorf("malformed label for node %s", l)
			}
			spec.Annotations.Labels[strings.TrimSpace(parts[0])] = strings.TrimSpace(parts[1])
		}
	}

	if reflect.DeepEqual(spec, &node.Spec) {
		return errNoChange
	}

	_, err = c.UpdateNode(common.Context(cmd), &api.UpdateNodeRequest{
		NodeID:      node.ID,
		NodeVersion: &node.Meta.Version,
		Spec:        spec,
	})

	if err != nil {
		return err
	}

	return nil
}
