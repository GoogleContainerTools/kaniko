package dockerapi

import (
	"encoding/json"
	"fmt"
	"io"
	"strings"
	"time"

	"github.com/docker/docker/api/types"
	"github.com/docker/docker/api/types/events"
	engineapi "github.com/docker/docker/client"
	"github.com/docker/swarmkit/agent/exec"
	"github.com/docker/swarmkit/api"
	"github.com/docker/swarmkit/log"
	gogotypes "github.com/gogo/protobuf/types"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
	"golang.org/x/net/context"
	"golang.org/x/time/rate"
)

// containerAdapter conducts remote operations for a container. All calls
// are mostly naked calls to the client API, seeded with information from
// containerConfig.
type containerAdapter struct {
	client    engineapi.APIClient
	container *containerConfig
	secrets   exec.SecretGetter
}

func newContainerAdapter(client engineapi.APIClient, nodeDescription *api.NodeDescription, task *api.Task, secrets exec.SecretGetter) (*containerAdapter, error) {
	ctnr, err := newContainerConfig(nodeDescription, task)
	if err != nil {
		return nil, err
	}

	return &containerAdapter{
		client:    client,
		container: ctnr,
		secrets:   secrets,
	}, nil
}

func noopPrivilegeFn() (string, error) { return "", nil }

func (c *containerConfig) imagePullOptions() types.ImagePullOptions {
	var registryAuth string

	if c.spec().PullOptions != nil {
		registryAuth = c.spec().PullOptions.RegistryAuth
	}

	return types.ImagePullOptions{
		// if the image needs to be pulled, the auth config will be retrieved and updated
		RegistryAuth:  registryAuth,
		PrivilegeFunc: noopPrivilegeFn,
	}
}

func (c *containerAdapter) pullImage(ctx context.Context) error {
	rc, err := c.client.ImagePull(ctx, c.container.image(), c.container.imagePullOptions())
	if err != nil {
		return err
	}

	dec := json.NewDecoder(rc)
	dec.UseNumber()
	m := map[string]interface{}{}
	spamLimiter := rate.NewLimiter(rate.Every(1000*time.Millisecond), 1)

	lastStatus := ""
	for {
		if err := dec.Decode(&m); err != nil {
			if err == io.EOF {
				break
			}
			return err
		}
		l := log.G(ctx)
		// limit pull progress logs unless the status changes
		if spamLimiter.Allow() || lastStatus != m["status"] {
			// if we have progress details, we have everything we need
			if progress, ok := m["progressDetail"].(map[string]interface{}); ok {
				// first, log the image and status
				l = l.WithFields(logrus.Fields{
					"image":  c.container.image(),
					"status": m["status"],
				})
				// then, if we have progress, log the progress
				if progress["current"] != nil && progress["total"] != nil {
					l = l.WithFields(logrus.Fields{
						"current": progress["current"],
						"total":   progress["total"],
					})
				}
			}
			l.Debug("pull in progress")
		}
		// sometimes, we get no useful information at all, and add no fields
		if status, ok := m["status"].(string); ok {
			lastStatus = status
		}
	}
	// if the final stream object contained an error, return it
	if errMsg, ok := m["error"]; ok {
		return errors.Errorf("%v", errMsg)
	}
	return nil
}

func (c *containerAdapter) createNetworks(ctx context.Context) error {
	for _, network := range c.container.networks() {
		opts, err := c.container.networkCreateOptions(network)
		if err != nil {
			return err
		}

		if _, err := c.client.NetworkCreate(ctx, network, opts); err != nil {
			if isNetworkExistError(err, network) {
				continue
			}

			return err
		}
	}

	return nil
}

func (c *containerAdapter) removeNetworks(ctx context.Context) error {
	for _, nid := range c.container.networks() {
		if err := c.client.NetworkRemove(ctx, nid); err != nil {
			if isActiveEndpointError(err) {
				continue
			}

			log.G(ctx).Errorf("network %s remove failed", nid)
			return err
		}
	}

	return nil
}

func (c *containerAdapter) create(ctx context.Context) error {
	if _, err := c.client.ContainerCreate(ctx,
		c.container.config(),
		c.container.hostConfig(),
		c.container.networkingConfig(),
		c.container.name()); err != nil {
		return err
	}

	return nil
}

func (c *containerAdapter) start(ctx context.Context) error {
	// TODO(nishanttotla): Consider adding checkpoint handling later
	return c.client.ContainerStart(ctx, c.container.name(), types.ContainerStartOptions{})
}

func (c *containerAdapter) inspect(ctx context.Context) (types.ContainerJSON, error) {
	return c.client.ContainerInspect(ctx, c.container.name())
}

// events issues a call to the events API and returns a channel with all
// events. The stream of events can be shutdown by cancelling the context.
//
// A chan struct{} is returned that will be closed if the event processing
// fails and needs to be restarted.
func (c *containerAdapter) events(ctx context.Context) (<-chan events.Message, <-chan struct{}, error) {
	// TODO(stevvooe): Move this to a single, global event dispatch. For
	// now, we create a connection per container.
	var (
		eventsq = make(chan events.Message)
		closed  = make(chan struct{})
	)

	log.G(ctx).Debugf("waiting on events")
	// TODO(stevvooe): For long running tasks, it is likely that we will have
	// to restart this under failure.
	eventCh, errCh := c.client.Events(ctx, types.EventsOptions{
		Since:   "0",
		Filters: c.container.eventFilter(),
	})

	go func() {
		defer close(closed)

		for {
			select {
			case msg := <-eventCh:
				select {
				case eventsq <- msg:
				case <-ctx.Done():
					return
				}
			case err := <-errCh:
				log.G(ctx).WithError(err).Error("error from events stream")
				return
			case <-ctx.Done():
				// exit
				return
			}
		}
	}()

	return eventsq, closed, nil
}

func (c *containerAdapter) shutdown(ctx context.Context) error {
	// Default stop grace period to 10s.
	stopgrace := 10 * time.Second
	spec := c.container.spec()
	if spec.StopGracePeriod != nil {
		stopgrace, _ = gogotypes.DurationFromProto(spec.StopGracePeriod)
	}
	return c.client.ContainerStop(ctx, c.container.name(), &stopgrace)
}

func (c *containerAdapter) terminate(ctx context.Context) error {
	return c.client.ContainerKill(ctx, c.container.name(), "")
}

func (c *containerAdapter) remove(ctx context.Context) error {
	return c.client.ContainerRemove(ctx, c.container.name(), types.ContainerRemoveOptions{
		RemoveVolumes: true,
		Force:         true,
	})
}

func (c *containerAdapter) createVolumes(ctx context.Context) error {
	// Create plugin volumes that are embedded inside a Mount
	for _, mount := range c.container.spec().Mounts {
		if mount.Type != api.MountTypeVolume {
			continue
		}

		// we create volumes when there is a volume driver available volume options
		if mount.VolumeOptions == nil {
			continue
		}

		if mount.VolumeOptions.DriverConfig == nil {
			continue
		}

		req := c.container.volumeCreateRequest(&mount)
		if _, err := c.client.VolumeCreate(ctx, *req); err != nil {
			// TODO(amitshukla): Today, volume create through the engine api does not return an error
			// when the named volume with the same parameters already exists.
			// It returns an error if the driver name is different - that is a valid error
			return err
		}
	}

	return nil
}

func (c *containerAdapter) logs(ctx context.Context, options api.LogSubscriptionOptions) (io.ReadCloser, error) {
	conf := c.container.config()
	if conf != nil && conf.Tty {
		return nil, errors.New("logs not supported on services with TTY")
	}

	apiOptions := types.ContainerLogsOptions{
		Follow:     options.Follow,
		Timestamps: true,
		Details:    false,
	}

	if options.Since != nil {
		since, err := gogotypes.TimestampFromProto(options.Since)
		if err != nil {
			return nil, err
		}
		apiOptions.Since = fmt.Sprintf("%d.%09d", since.Unix(), int64(since.Nanosecond()))
	}

	if options.Tail < 0 {
		// See protobuf documentation for details of how this works.
		apiOptions.Tail = fmt.Sprint(-options.Tail - 1)
	} else if options.Tail > 0 {
		return nil, fmt.Errorf("tail relative to start of logs not supported via docker API")
	}

	if len(options.Streams) == 0 {
		// empty == all
		apiOptions.ShowStdout, apiOptions.ShowStderr = true, true
	} else {
		for _, stream := range options.Streams {
			switch stream {
			case api.LogStreamStdout:
				apiOptions.ShowStdout = true
			case api.LogStreamStderr:
				apiOptions.ShowStderr = true
			}
		}
	}

	return c.client.ContainerLogs(ctx, c.container.name(), apiOptions)
}

// TODO(mrjana/stevvooe): There is no proper error code for network not found
// error in engine-api. Resort to string matching until engine-api is fixed.

func isActiveEndpointError(err error) bool {
	return strings.Contains(err.Error(), "has active endpoints")
}

func isNetworkExistError(err error, name string) bool {
	return strings.Contains(err.Error(), fmt.Sprintf("network with name %s already exists", name))
}

func isContainerCreateNameConflict(err error) bool {
	return strings.Contains(err.Error(), "Conflict. The name")
}

func isUnknownContainer(err error) bool {
	return strings.Contains(err.Error(), "No such container:")
}

func isStoppedContainer(err error) bool {
	return strings.Contains(err.Error(), "is already stopped")
}
