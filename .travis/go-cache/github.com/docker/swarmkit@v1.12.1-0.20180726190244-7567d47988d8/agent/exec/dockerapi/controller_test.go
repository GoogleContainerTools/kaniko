package dockerapi

import (
	"bytes"
	"fmt"
	"io"
	"io/ioutil"
	"reflect"
	"runtime"
	"testing"
	"time"

	"github.com/docker/docker/api/types"
	containertypes "github.com/docker/docker/api/types/container"
	"github.com/docker/docker/api/types/events"
	"github.com/docker/docker/api/types/network"
	"github.com/docker/swarmkit/agent/exec"
	"github.com/docker/swarmkit/api"
	"github.com/docker/swarmkit/identity"
	"github.com/docker/swarmkit/log"
	gogotypes "github.com/gogo/protobuf/types"
	"github.com/stretchr/testify/assert"
	"golang.org/x/net/context"
)

var tenSecond = 10 * time.Second

func TestControllerPrepare(t *testing.T) {
	task := genTask(t)
	ctx, client, ctlr, config, finish := genTestControllerEnv(t, task)
	defer func() {
		finish()
		assert.Equal(t, 1, client.calls["ImagePull"])
		assert.Equal(t, 1, client.calls["ContainerCreate"])
	}()

	client.ImagePullFn = func(_ context.Context, refStr string, options types.ImagePullOptions) (io.ReadCloser, error) {
		if refStr == config.image() {
			return ioutil.NopCloser(bytes.NewBuffer([]byte{})), nil
		}
		panic("unexpected call of ImagePull")
	}

	client.ContainerCreateFn = func(_ context.Context, cConfig *containertypes.Config, hConfig *containertypes.HostConfig, nConfig *network.NetworkingConfig, containerName string) (containertypes.ContainerCreateCreatedBody, error) {
		if reflect.DeepEqual(*cConfig, *config.config()) &&
			reflect.DeepEqual(*hConfig, *config.hostConfig()) &&
			reflect.DeepEqual(*nConfig, *config.networkingConfig()) &&
			containerName == config.name() {
			return containertypes.ContainerCreateCreatedBody{ID: "container-id-" + task.ID}, nil
		}
		panic("unexpected call to ContainerCreate")
	}

	assert.NoError(t, ctlr.Prepare(ctx))
}

func TestControllerPrepareAlreadyPrepared(t *testing.T) {
	task := genTask(t)
	ctx, client, ctlr, config, finish := genTestControllerEnv(t, task)
	defer func() {
		finish()
		assert.Equal(t, 1, client.calls["ImagePull"])
		assert.Equal(t, 1, client.calls["ContainerCreate"])
		assert.Equal(t, 1, client.calls["ContainerInspect"])
	}()

	client.ImagePullFn = func(_ context.Context, refStr string, options types.ImagePullOptions) (io.ReadCloser, error) {
		if refStr == config.image() {
			return ioutil.NopCloser(bytes.NewBuffer([]byte{})), nil
		}
		panic("unexpected call of ImagePull")
	}

	client.ContainerCreateFn = func(_ context.Context, cConfig *containertypes.Config, hostConfig *containertypes.HostConfig, networking *network.NetworkingConfig, containerName string) (containertypes.ContainerCreateCreatedBody, error) {
		if reflect.DeepEqual(*cConfig, *config.config()) &&
			reflect.DeepEqual(*networking, *config.networkingConfig()) &&
			containerName == config.name() {
			return containertypes.ContainerCreateCreatedBody{}, fmt.Errorf("Conflict. The name")
		}
		panic("unexpected call of ContainerCreate")
	}

	client.ContainerInspectFn = func(_ context.Context, containerName string) (types.ContainerJSON, error) {
		if containerName == config.name() {
			return types.ContainerJSON{}, nil
		}
		panic("unexpected call of ContainerInspect")
	}

	// ensure idempotence
	if err := ctlr.Prepare(ctx); err != exec.ErrTaskPrepared {
		t.Fatalf("expected error %v, got %v", exec.ErrTaskPrepared, err)
	}
}

func TestControllerStart(t *testing.T) {
	task := genTask(t)
	ctx, client, ctlr, config, finish := genTestControllerEnv(t, task)
	defer func() {
		finish()
		assert.Equal(t, 1, client.calls["ContainerInspect"])
		assert.Equal(t, 1, client.calls["ContainerStart"])
	}()

	client.ContainerInspectFn = func(_ context.Context, containerName string) (types.ContainerJSON, error) {
		if containerName == config.name() {
			return types.ContainerJSON{
				ContainerJSONBase: &types.ContainerJSONBase{
					State: &types.ContainerState{
						Status: "created",
					},
				},
			}, nil
		}
		panic("unexpected call of ContainerInspect")
	}

	client.ContainerStartFn = func(_ context.Context, containerName string, options types.ContainerStartOptions) error {
		if containerName == config.name() && reflect.DeepEqual(options, types.ContainerStartOptions{}) {
			return nil
		}
		panic("unexpected call of ContainerStart")
	}

	assert.NoError(t, ctlr.Start(ctx))
}

func TestControllerStartAlreadyStarted(t *testing.T) {
	task := genTask(t)
	ctx, client, ctlr, config, finish := genTestControllerEnv(t, task)
	defer func() {
		finish()
		assert.Equal(t, 1, client.calls["ContainerInspect"])
	}()

	client.ContainerInspectFn = func(_ context.Context, containerName string) (types.ContainerJSON, error) {
		if containerName == config.name() {
			return types.ContainerJSON{
				ContainerJSONBase: &types.ContainerJSONBase{
					State: &types.ContainerState{
						Status: "notcreated", // can be anything but created
					},
				},
			}, nil
		}
		panic("unexpected call of ContainerInspect")
	}

	// ensure idempotence
	if err := ctlr.Start(ctx); err != exec.ErrTaskStarted {
		t.Fatalf("expected error %v, got %v", exec.ErrTaskPrepared, err)
	}
}

func TestControllerWait(t *testing.T) {
	task := genTask(t)
	ctx, client, ctlr, config, finish := genTestControllerEnv(t, task)
	defer func() {
		finish()
		assert.Equal(t, 2, client.calls["ContainerInspect"])
		assert.Equal(t, 1, client.calls["Events"])
	}()

	client.ContainerInspectFn = func(_ context.Context, container string) (types.ContainerJSON, error) {
		if client.calls["ContainerInspect"] == 1 && container == config.name() {
			return types.ContainerJSON{
				ContainerJSONBase: &types.ContainerJSONBase{
					State: &types.ContainerState{
						Status: "running",
					},
				},
			}, nil
		} else if client.calls["ContainerInspect"] == 2 && container == config.name() {
			return types.ContainerJSON{
				ContainerJSONBase: &types.ContainerJSONBase{
					State: &types.ContainerState{
						Status: "stopped", // can be anything but created
					},
				},
			}, nil
		}
		panic("unexpected call of ContainerInspect")
	}

	client.EventsFn = func(_ context.Context, options types.EventsOptions) (<-chan events.Message, <-chan error) {
		if reflect.DeepEqual(options, types.EventsOptions{
			Since:   "0",
			Filters: config.eventFilter(),
		}) {
			return makeEvents(t, config, "create", "die")
		}
		panic("unexpected call of Events")
	}

	assert.NoError(t, ctlr.Wait(ctx))
}

func TestControllerWaitUnhealthy(t *testing.T) {
	task := genTask(t)
	ctx, client, ctlr, config, finish := genTestControllerEnv(t, task)
	defer func() {
		finish()
		assert.Equal(t, 1, client.calls["ContainerInspect"])
		assert.Equal(t, 1, client.calls["Events"])
		assert.Equal(t, 1, client.calls["ContainerStop"])
	}()
	client.ContainerInspectFn = func(_ context.Context, containerName string) (types.ContainerJSON, error) {
		if containerName == config.name() {
			return types.ContainerJSON{
				ContainerJSONBase: &types.ContainerJSONBase{
					State: &types.ContainerState{
						Status: "running",
					},
				},
			}, nil
		}
		panic("unexpected call ContainerInspect")
	}
	evs, errs := makeEvents(t, config, "create", "health_status: unhealthy")
	client.EventsFn = func(_ context.Context, options types.EventsOptions) (<-chan events.Message, <-chan error) {
		if reflect.DeepEqual(options, types.EventsOptions{
			Since:   "0",
			Filters: config.eventFilter(),
		}) {
			return evs, errs
		}
		panic("unexpected call of Events")
	}
	client.ContainerStopFn = func(_ context.Context, containerName string, timeout *time.Duration) error {
		if containerName == config.name() && *timeout == tenSecond {
			return nil
		}
		panic("unexpected call of ContainerStop")
	}

	assert.Equal(t, ctlr.Wait(ctx), ErrContainerUnhealthy)
}

func TestControllerWaitExitError(t *testing.T) {
	task := genTask(t)
	ctx, client, ctlr, config, finish := genTestControllerEnv(t, task)
	defer func() {
		finish()
		assert.Equal(t, 2, client.calls["ContainerInspect"])
		assert.Equal(t, 1, client.calls["Events"])
	}()

	client.ContainerInspectFn = func(_ context.Context, containerName string) (types.ContainerJSON, error) {
		if client.calls["ContainerInspect"] == 1 && containerName == config.name() {
			return types.ContainerJSON{
				ContainerJSONBase: &types.ContainerJSONBase{
					State: &types.ContainerState{
						Status: "running",
					},
				},
			}, nil
		} else if client.calls["ContainerInspect"] == 2 && containerName == config.name() {
			return types.ContainerJSON{
				ContainerJSONBase: &types.ContainerJSONBase{
					ID: "cid",
					State: &types.ContainerState{
						Status:   "exited", // can be anything but created
						ExitCode: 1,
						Pid:      1,
					},
				},
			}, nil
		}
		panic("unexpected call of ContainerInspect")
	}

	client.EventsFn = func(_ context.Context, options types.EventsOptions) (<-chan events.Message, <-chan error) {
		if reflect.DeepEqual(options, types.EventsOptions{
			Since:   "0",
			Filters: config.eventFilter(),
		}) {
			return makeEvents(t, config, "create", "die")
		}
		panic("unexpected call of Events")
	}

	err := ctlr.Wait(ctx)
	checkExitError(t, 1, err)
}

func checkExitError(t *testing.T, expectedCode int, err error) {
	ec, ok := err.(exec.ExitCoder)
	if !ok {
		t.Fatalf("expected an exit error, got: %v", err)
	}

	assert.Equal(t, expectedCode, ec.ExitCode())
}

func TestControllerWaitExitedClean(t *testing.T) {
	task := genTask(t)
	ctx, client, ctlr, config, finish := genTestControllerEnv(t, task)
	defer func() {
		finish()
		assert.Equal(t, 1, client.calls["ContainerInspect"])
	}()

	client.ContainerInspectFn = func(_ context.Context, container string) (types.ContainerJSON, error) {
		if container == config.name() {
			return types.ContainerJSON{
				ContainerJSONBase: &types.ContainerJSONBase{
					State: &types.ContainerState{
						Status: "exited",
					},
				},
			}, nil
		}
		panic("unexpected call of ContainerInspect")
	}

	err := ctlr.Wait(ctx)
	assert.Nil(t, err)
}

func TestControllerWaitExitedError(t *testing.T) {
	task := genTask(t)
	ctx, client, ctlr, config, finish := genTestControllerEnv(t, task)
	defer func() {
		finish()
		assert.Equal(t, 1, client.calls["ContainerInspect"])
	}()

	client.ContainerInspectFn = func(_ context.Context, containerName string) (types.ContainerJSON, error) {
		if containerName == config.name() {
			return types.ContainerJSON{
				ContainerJSONBase: &types.ContainerJSONBase{
					ID: "cid",
					State: &types.ContainerState{
						Status:   "exited",
						ExitCode: 1,
						Pid:      1,
					},
				},
			}, nil
		}
		panic("unexpected call of ContainerInspect")
	}

	err := ctlr.Wait(ctx)
	checkExitError(t, 1, err)
}

func TestControllerShutdown(t *testing.T) {
	task := genTask(t)
	ctx, client, ctlr, config, finish := genTestControllerEnv(t, task)
	defer func() {
		finish()
		assert.Equal(t, 1, client.calls["ContainerStop"])
	}()

	client.ContainerStopFn = func(_ context.Context, containerName string, timeout *time.Duration) error {
		if containerName == config.name() && *timeout == tenSecond {
			return nil
		}
		panic("unexpected call of ContainerStop")
	}

	assert.NoError(t, ctlr.Shutdown(ctx))
}

func TestControllerTerminate(t *testing.T) {
	task := genTask(t)
	ctx, client, ctlr, config, finish := genTestControllerEnv(t, task)
	defer func() {
		finish()
		assert.Equal(t, 1, client.calls["ContainerKill"])
	}()

	client.ContainerKillFn = func(_ context.Context, containerName, signal string) error {
		if containerName == config.name() && signal == "" {
			return nil
		}
		panic("unexpected call of ContainerKill")
	}

	assert.NoError(t, ctlr.Terminate(ctx))
}

func TestControllerRemove(t *testing.T) {
	task := genTask(t)
	ctx, client, ctlr, config, finish := genTestControllerEnv(t, task)
	defer func() {
		finish()
		assert.Equal(t, 1, client.calls["ContainerStop"])
		assert.Equal(t, 1, client.calls["ContainerRemove"])
	}()

	client.ContainerStopFn = func(_ context.Context, container string, timeout *time.Duration) error {
		if container == config.name() && *timeout == tenSecond {
			return nil
		}
		panic("unexpected call of ContainerStop")
	}

	client.ContainerRemoveFn = func(_ context.Context, container string, options types.ContainerRemoveOptions) error {
		if container == config.name() && reflect.DeepEqual(options, types.ContainerRemoveOptions{
			RemoveVolumes: true,
			Force:         true,
		}) {
			return nil
		}
		panic("unexpected call of ContainerRemove")
	}

	assert.NoError(t, ctlr.Remove(ctx))
}

func genTestControllerEnv(t *testing.T, task *api.Task) (context.Context, *StubAPIClient, exec.Controller, *containerConfig, func()) {
	testNodeDescription := &api.NodeDescription{
		Hostname: "testHostname",
		Platform: &api.Platform{
			OS:           "linux",
			Architecture: "x86_64",
		},
	}

	client := NewStubAPIClient()
	ctlr, err := newController(client, testNodeDescription, task, nil)
	assert.NoError(t, err)

	config, err := newContainerConfig(testNodeDescription, task)
	assert.NoError(t, err)
	assert.NotNil(t, config)

	ctx := context.Background()

	// Put test name into log messages. Awesome!
	pc, _, _, ok := runtime.Caller(1)
	if ok {
		fn := runtime.FuncForPC(pc)
		ctx = log.WithLogger(ctx, log.L.WithField("test", fn.Name()))
	}

	ctx, cancel := context.WithCancel(ctx)
	return ctx, client, ctlr, config, cancel
}

func genTask(t *testing.T) *api.Task {
	const (
		nodeID    = "dockerexec-test-node-id"
		serviceID = "dockerexec-test-service"
		reference = "stevvooe/foo:latest"
	)

	return &api.Task{
		ID:        identity.NewID(),
		ServiceID: serviceID,
		NodeID:    nodeID,
		Spec: api.TaskSpec{
			Runtime: &api.TaskSpec_Container{
				Container: &api.ContainerSpec{
					Image:           reference,
					StopGracePeriod: gogotypes.DurationProto(10 * time.Second),
				},
			},
		},
	}
}

func makeEvents(t *testing.T, container *containerConfig, actions ...string) (<-chan events.Message, <-chan error) {
	evs := make(chan events.Message, len(actions))
	for _, action := range actions {
		evs <- events.Message{
			Type:   events.ContainerEventType,
			Action: action,
			Actor: events.Actor{
				// TODO(stevvooe): Resolve container id.
				Attributes: map[string]string{
					"name": container.name(),
				},
			},
		}
	}
	close(evs)

	return evs, nil
}
