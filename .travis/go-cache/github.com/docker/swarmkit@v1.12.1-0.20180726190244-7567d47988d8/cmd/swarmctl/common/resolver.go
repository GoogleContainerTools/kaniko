package common

import (
	"fmt"

	"github.com/docker/swarmkit/api"
	"github.com/spf13/cobra"
	"golang.org/x/net/context"
)

// Resolver provides ID to Name resolution.
type Resolver struct {
	cmd   *cobra.Command
	c     api.ControlClient
	ctx   context.Context
	cache map[string]string
}

// NewResolver creates a new Resolver.
func NewResolver(cmd *cobra.Command, c api.ControlClient) *Resolver {
	return &Resolver{
		cmd:   cmd,
		c:     c,
		ctx:   Context(cmd),
		cache: make(map[string]string),
	}
}

func (r *Resolver) get(t interface{}, id string) string {
	switch t.(type) {
	case api.Node:
		res, err := r.c.GetNode(r.ctx, &api.GetNodeRequest{NodeID: id})
		if err != nil {
			return id
		}
		if res.Node.Spec.Annotations.Name != "" {
			return res.Node.Spec.Annotations.Name
		}
		if res.Node.Description == nil {
			return id
		}
		return res.Node.Description.Hostname
	case api.Service:
		res, err := r.c.GetService(r.ctx, &api.GetServiceRequest{ServiceID: id})
		if err != nil {
			return id
		}
		return res.Service.Spec.Annotations.Name
	case api.Task:
		res, err := r.c.GetTask(r.ctx, &api.GetTaskRequest{TaskID: id})
		if err != nil {
			return id
		}
		svc := r.get(api.Service{}, res.Task.ServiceID)
		return fmt.Sprintf("%s.%d", svc, res.Task.Slot)
	default:
		return id
	}
}

// Resolve will attempt to resolve an ID to a Name by querying the manager.
// Results are stored into a cache.
// If the `-n` flag is used in the command-line, resolution is disabled.
func (r *Resolver) Resolve(t interface{}, id string) string {
	if r.cmd.Flags().Changed("no-resolve") {
		return id
	}
	if name, ok := r.cache[id]; ok {
		return name
	}
	name := r.get(t, id)
	r.cache[id] = name
	return name
}
