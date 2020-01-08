package dockerapi

import (
	"github.com/docker/docker/api/types"
	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/api/types/events"
	"github.com/docker/docker/api/types/network"
	"github.com/docker/docker/client"
	"golang.org/x/net/context"
	"io"
	"runtime"
	"strings"
	"time"
)

// StubAPIClient implements the client.APIClient interface, but allows
// you to specify the behavior of each of the methods.
type StubAPIClient struct {
	client.APIClient
	calls              map[string]int
	ContainerCreateFn  func(_ context.Context, config *container.Config, hostConfig *container.HostConfig, networking *network.NetworkingConfig, containerName string) (container.ContainerCreateCreatedBody, error)
	ContainerInspectFn func(_ context.Context, containerID string) (types.ContainerJSON, error)
	ContainerKillFn    func(_ context.Context, containerID, signal string) error
	ContainerRemoveFn  func(_ context.Context, containerID string, options types.ContainerRemoveOptions) error
	ContainerStartFn   func(_ context.Context, containerID string, options types.ContainerStartOptions) error
	ContainerStopFn    func(_ context.Context, containerID string, timeout *time.Duration) error
	ImagePullFn        func(_ context.Context, refStr string, options types.ImagePullOptions) (io.ReadCloser, error)
	EventsFn           func(_ context.Context, options types.EventsOptions) (<-chan events.Message, <-chan error)
}

// NewStubAPIClient returns an initialized StubAPIClient
func NewStubAPIClient() *StubAPIClient {
	return &StubAPIClient{
		calls: make(map[string]int),
	}
}

// If function A calls updateCountsForSelf,
// The callCount[A] value will be incremented
func (sa *StubAPIClient) called() {
	pc, _, _, ok := runtime.Caller(1)
	if !ok {
		panic("failed to update counts")
	}
	// longName looks like 'github.com/docker/swarmkit/agent/exec.(*StubController).Prepare:1'
	longName := runtime.FuncForPC(pc).Name()
	parts := strings.Split(longName, ".")
	tail := strings.Split(parts[len(parts)-1], ":")
	sa.calls[tail[0]]++
}

// ContainerCreate is part of the APIClient interface
func (sa *StubAPIClient) ContainerCreate(ctx context.Context, config *container.Config, hostConfig *container.HostConfig, networking *network.NetworkingConfig, containerName string) (container.ContainerCreateCreatedBody, error) {
	sa.called()
	return sa.ContainerCreateFn(ctx, config, hostConfig, networking, containerName)
}

// ContainerInspect is part of the APIClient interface
func (sa *StubAPIClient) ContainerInspect(ctx context.Context, containerID string) (types.ContainerJSON, error) {
	sa.called()
	return sa.ContainerInspectFn(ctx, containerID)
}

// ContainerKill is part of the APIClient interface
func (sa *StubAPIClient) ContainerKill(ctx context.Context, containerID, signal string) error {
	sa.called()
	return sa.ContainerKillFn(ctx, containerID, signal)
}

// ContainerRemove is part of the APIClient interface
func (sa *StubAPIClient) ContainerRemove(ctx context.Context, containerID string, options types.ContainerRemoveOptions) error {
	sa.called()
	return sa.ContainerRemoveFn(ctx, containerID, options)
}

// ContainerStart is part of the APIClient interface
func (sa *StubAPIClient) ContainerStart(ctx context.Context, containerID string, options types.ContainerStartOptions) error {
	sa.called()
	return sa.ContainerStartFn(ctx, containerID, options)
}

// ContainerStop is part of the APIClient interface
func (sa *StubAPIClient) ContainerStop(ctx context.Context, containerID string, timeout *time.Duration) error {
	sa.called()
	return sa.ContainerStopFn(ctx, containerID, timeout)
}

// ImagePull is part of the APIClient interface
func (sa *StubAPIClient) ImagePull(ctx context.Context, refStr string, options types.ImagePullOptions) (io.ReadCloser, error) {
	sa.called()
	return sa.ImagePullFn(ctx, refStr, options)
}

// Events is part of the APIClient interface
func (sa *StubAPIClient) Events(ctx context.Context, options types.EventsOptions) (<-chan events.Message, <-chan error) {
	sa.called()
	return sa.EventsFn(ctx, options)
}
