package exec

import (
	"errors"
	"fmt"
	"runtime"
	"testing"

	"github.com/docker/swarmkit/api"
	"github.com/docker/swarmkit/log"
	gogotypes "github.com/gogo/protobuf/types"
	"github.com/stretchr/testify/assert"
	"golang.org/x/net/context"
)

func TestResolve(t *testing.T) {
	var (
		ctx      = context.Background()
		executor = &mockExecutor{}
		task     = newTestTask(t, api.TaskStateAssigned, api.TaskStateRunning)
	)

	_, status, err := Resolve(ctx, task, executor)
	assert.NoError(t, err)
	assert.Equal(t, api.TaskStateAccepted, status.State)
	assert.Equal(t, "accepted", status.Message)

	task.Status = *status
	// now, we get no status update.
	_, status, err = Resolve(ctx, task, executor)
	assert.NoError(t, err)
	assert.Equal(t, task.Status, *status)

	// now test an error causing rejection
	executor.err = errors.New("some error")
	task = newTestTask(t, api.TaskStateAssigned, api.TaskStateRunning)
	_, status, err = Resolve(ctx, task, executor)
	assert.Equal(t, executor.err, err)
	assert.Equal(t, api.TaskStateRejected, status.State)

	// on Resolve failure, tasks already started should be considered failed
	task = newTestTask(t, api.TaskStateStarting, api.TaskStateRunning)
	_, status, err = Resolve(ctx, task, executor)
	assert.Equal(t, executor.err, err)
	assert.Equal(t, api.TaskStateFailed, status.State)

	// on Resolve failure, tasks already in terminated state don't need update
	task = newTestTask(t, api.TaskStateCompleted, api.TaskStateRunning)
	_, status, err = Resolve(ctx, task, executor)
	assert.Equal(t, executor.err, err)
	assert.Equal(t, api.TaskStateCompleted, status.State)

	// task is now foobared, from a reporting perspective but we can now
	// resolve the controller for some reason. Ensure the task state isn't
	// touched.
	task.Status = *status
	executor.err = nil
	_, status, err = Resolve(ctx, task, executor)
	assert.NoError(t, err)
	assert.Equal(t, task.Status, *status)
}

func TestAcceptPrepare(t *testing.T) {
	var (
		task              = newTestTask(t, api.TaskStateAssigned, api.TaskStateRunning)
		ctx, ctlr, finish = buildTestEnv(t, task)
	)
	defer func() {
		finish()
		assert.Equal(t, 1, ctlr.calls["Prepare"])
	}()

	ctlr.PrepareFn = func(_ context.Context) error {
		return nil
	}

	// Report acceptance.
	status := checkDo(ctx, t, task, ctlr, &api.TaskStatus{
		State:   api.TaskStateAccepted,
		Message: "accepted",
	})

	// Actually prepare the task.
	task.Status = *status

	status = checkDo(ctx, t, task, ctlr, &api.TaskStatus{
		State:   api.TaskStatePreparing,
		Message: "preparing",
	})

	task.Status = *status

	checkDo(ctx, t, task, ctlr, &api.TaskStatus{
		State:   api.TaskStateReady,
		Message: "prepared",
	})
}

func TestPrepareAlready(t *testing.T) {
	var (
		task              = newTestTask(t, api.TaskStateAssigned, api.TaskStateRunning)
		ctx, ctlr, finish = buildTestEnv(t, task)
	)
	defer func() {
		finish()
		assert.Equal(t, 1, ctlr.calls["Prepare"])
	}()
	ctlr.PrepareFn = func(_ context.Context) error {
		return ErrTaskPrepared
	}

	// Report acceptance.
	status := checkDo(ctx, t, task, ctlr, &api.TaskStatus{
		State:   api.TaskStateAccepted,
		Message: "accepted",
	})

	// Actually prepare the task.
	task.Status = *status

	status = checkDo(ctx, t, task, ctlr, &api.TaskStatus{
		State:   api.TaskStatePreparing,
		Message: "preparing",
	})

	task.Status = *status

	checkDo(ctx, t, task, ctlr, &api.TaskStatus{
		State:   api.TaskStateReady,
		Message: "prepared",
	})
}

func TestPrepareFailure(t *testing.T) {
	var (
		task              = newTestTask(t, api.TaskStateAssigned, api.TaskStateRunning)
		ctx, ctlr, finish = buildTestEnv(t, task)
	)
	defer func() {
		finish()
		assert.Equal(t, ctlr.calls["Prepare"], 1)
	}()
	ctlr.PrepareFn = func(_ context.Context) error {
		return errors.New("test error")
	}

	// Report acceptance.
	status := checkDo(ctx, t, task, ctlr, &api.TaskStatus{
		State:   api.TaskStateAccepted,
		Message: "accepted",
	})

	// Actually prepare the task.
	task.Status = *status

	status = checkDo(ctx, t, task, ctlr, &api.TaskStatus{
		State:   api.TaskStatePreparing,
		Message: "preparing",
	})

	task.Status = *status

	checkDo(ctx, t, task, ctlr, &api.TaskStatus{
		State:   api.TaskStateRejected,
		Message: "preparing",
		Err:     "test error",
	})
}

func TestReadyRunning(t *testing.T) {
	var (
		task              = newTestTask(t, api.TaskStateReady, api.TaskStateRunning)
		ctx, ctlr, finish = buildTestEnv(t, task)
	)
	defer func() {
		finish()
		assert.Equal(t, 1, ctlr.calls["Start"])
		assert.Equal(t, 2, ctlr.calls["Wait"])
	}()

	ctlr.StartFn = func(ctx context.Context) error {
		return nil
	}
	ctlr.WaitFn = func(ctx context.Context) error {
		if ctlr.calls["Wait"] == 1 {
			return context.Canceled
		} else if ctlr.calls["Wait"] == 2 {
			return nil
		} else {
			panic("unexpected call!")
		}
	}

	// Report starting
	status := checkDo(ctx, t, task, ctlr, &api.TaskStatus{
		State:   api.TaskStateStarting,
		Message: "starting",
	})

	task.Status = *status

	// start the container
	status = checkDo(ctx, t, task, ctlr, &api.TaskStatus{
		State:   api.TaskStateRunning,
		Message: "started",
	})

	task.Status = *status

	// resume waiting
	status = checkDo(ctx, t, task, ctlr, &api.TaskStatus{
		State:   api.TaskStateRunning,
		Message: "started",
	}, ErrTaskRetry)

	task.Status = *status
	// wait and cancel
	dctlr := &StatuserController{
		StubController: ctlr,
		cstatus: &api.ContainerStatus{
			ExitCode: 0,
		},
	}
	checkDo(ctx, t, task, dctlr, &api.TaskStatus{
		State:   api.TaskStateCompleted,
		Message: "finished",
		RuntimeStatus: &api.TaskStatus_Container{
			Container: &api.ContainerStatus{
				ExitCode: 0,
			},
		},
	})
}

func TestReadyRunningExitFailure(t *testing.T) {
	var (
		task              = newTestTask(t, api.TaskStateReady, api.TaskStateRunning)
		ctx, ctlr, finish = buildTestEnv(t, task)
	)
	defer func() {
		finish()
		assert.Equal(t, 1, ctlr.calls["Start"])
		assert.Equal(t, 1, ctlr.calls["Wait"])
	}()

	ctlr.StartFn = func(ctx context.Context) error {
		return nil
	}
	ctlr.WaitFn = func(ctx context.Context) error {
		return newExitError(1)
	}

	// Report starting
	status := checkDo(ctx, t, task, ctlr, &api.TaskStatus{
		State:   api.TaskStateStarting,
		Message: "starting",
	})

	task.Status = *status

	// start the container
	status = checkDo(ctx, t, task, ctlr, &api.TaskStatus{
		State:   api.TaskStateRunning,
		Message: "started",
	})

	task.Status = *status
	dctlr := &StatuserController{
		StubController: ctlr,
		cstatus: &api.ContainerStatus{
			ExitCode: 1,
		},
	}
	checkDo(ctx, t, task, dctlr, &api.TaskStatus{
		State: api.TaskStateFailed,
		RuntimeStatus: &api.TaskStatus_Container{
			Container: &api.ContainerStatus{
				ExitCode: 1,
			},
		},
		Message: "started",
		Err:     "test error, exit code=1",
	})
}

func TestAlreadyStarted(t *testing.T) {
	var (
		task              = newTestTask(t, api.TaskStateReady, api.TaskStateRunning)
		ctx, ctlr, finish = buildTestEnv(t, task)
	)
	defer func() {
		finish()
		assert.Equal(t, 1, ctlr.calls["Start"])
		assert.Equal(t, 2, ctlr.calls["Wait"])
	}()

	ctlr.StartFn = func(ctx context.Context) error {
		return ErrTaskStarted
	}
	ctlr.WaitFn = func(ctx context.Context) error {
		if ctlr.calls["Wait"] == 1 {
			return context.Canceled
		} else if ctlr.calls["Wait"] == 2 {
			return newExitError(1)
		} else {
			panic("unexpected call!")
		}
	}

	// Before we can move to running, we have to move to startin.
	status := checkDo(ctx, t, task, ctlr, &api.TaskStatus{
		State:   api.TaskStateStarting,
		Message: "starting",
	})

	task.Status = *status

	// start the container
	status = checkDo(ctx, t, task, ctlr, &api.TaskStatus{
		State:   api.TaskStateRunning,
		Message: "started",
	})

	task.Status = *status

	status = checkDo(ctx, t, task, ctlr, &api.TaskStatus{
		State:   api.TaskStateRunning,
		Message: "started",
	}, ErrTaskRetry)

	task.Status = *status

	// now take the real exit to test wait cancelling.
	dctlr := &StatuserController{
		StubController: ctlr,
		cstatus: &api.ContainerStatus{
			ExitCode: 1,
		},
	}
	checkDo(ctx, t, task, dctlr, &api.TaskStatus{
		State: api.TaskStateFailed,
		RuntimeStatus: &api.TaskStatus_Container{
			Container: &api.ContainerStatus{
				ExitCode: 1,
			},
		},
		Message: "started",
		Err:     "test error, exit code=1",
	})

}
func TestShutdown(t *testing.T) {
	var (
		task              = newTestTask(t, api.TaskStateNew, api.TaskStateShutdown)
		ctx, ctlr, finish = buildTestEnv(t, task)
	)
	defer func() {
		finish()
		assert.Equal(t, 1, ctlr.calls["Shutdown"])
	}()
	ctlr.ShutdownFn = func(_ context.Context) error {
		return nil
	}

	checkDo(ctx, t, task, ctlr, &api.TaskStatus{
		State:   api.TaskStateShutdown,
		Message: "shutdown",
	})
}

// TestDesiredStateRemove checks that the agent maintains SHUTDOWN as the
// maximum state in the agent. This is particularly relevant for the case
// where a service scale down or deletion sets the desired state of tasks
// that are supposed to be removed to REMOVE.
func TestDesiredStateRemove(t *testing.T) {
	var (
		task              = newTestTask(t, api.TaskStateNew, api.TaskStateRemove)
		ctx, ctlr, finish = buildTestEnv(t, task)
	)
	defer func() {
		finish()
		assert.Equal(t, 1, ctlr.calls["Shutdown"])
	}()
	ctlr.ShutdownFn = func(_ context.Context) error {
		return nil
	}

	checkDo(ctx, t, task, ctlr, &api.TaskStatus{
		State:   api.TaskStateShutdown,
		Message: "shutdown",
	})
}

// TestDesiredStateRemoveOnlyNonterminal checks that the agent will only stop
// a container on REMOVE if it's not already in a terminal state. If the
// container is already in a terminal state, (like COMPLETE) the agent should
// take no action
func TestDesiredStateRemoveOnlyNonterminal(t *testing.T) {
	// go through all terminal states, just for completeness' sake
	for _, state := range []api.TaskState{
		api.TaskStateCompleted,
		api.TaskStateShutdown,
		api.TaskStateFailed,
		api.TaskStateRejected,
		api.TaskStateRemove,
		// no TaskStateOrphaned becaused that's not a state the task can be in
		// on the agent
	} {
		// capture state variable here to run in parallel
		state := state
		t.Run(state.String(), func(t *testing.T) {
			// go parallel to go faster
			t.Parallel()
			var (
				// create a new task, actual state `state`, desired state
				// shutdown
				task              = newTestTask(t, state, api.TaskStateShutdown)
				ctx, ctlr, finish = buildTestEnv(t, task)
			)
			// make the shutdown function a noop
			ctlr.ShutdownFn = func(_ context.Context) error {
				return nil
			}

			// Note we check for error ErrTaskNoop, which will be raised
			// because nothing happens
			checkDo(ctx, t, task, ctlr, &api.TaskStatus{
				State: state,
			}, ErrTaskNoop)
			defer func() {
				finish()
				// we should never have called shutdown
				assert.Equal(t, 0, ctlr.calls["Shutdown"])
			}()
		})
	}
}

// StatuserController is used to create a new Controller, which is also a ContainerStatuser.
// We cannot add ContainerStatus() to the Controller, due to the check in controller.go:242
type StatuserController struct {
	*StubController
	cstatus *api.ContainerStatus
}

func (mc *StatuserController) ContainerStatus(ctx context.Context) (*api.ContainerStatus, error) {
	return mc.cstatus, nil
}

type exitCoder struct {
	code int
}

func newExitError(code int) error { return &exitCoder{code} }

func (ec *exitCoder) Error() string { return fmt.Sprintf("test error, exit code=%v", ec.code) }
func (ec *exitCoder) ExitCode() int { return ec.code }

func checkDo(ctx context.Context, t *testing.T, task *api.Task, ctlr Controller, expected *api.TaskStatus, expectedErr ...error) *api.TaskStatus {
	status, err := Do(ctx, task, ctlr)
	if len(expectedErr) > 0 {
		assert.Equal(t, expectedErr[0], err)
	} else {
		assert.NoError(t, err)
	}

	// if the status and task.Status are different, make sure new timestamp is greater
	if task.Status.Timestamp != nil {
		// crazy timestamp validation follows
		previous, err := gogotypes.TimestampFromProto(task.Status.Timestamp)
		assert.Nil(t, err)

		current, err := gogotypes.TimestampFromProto(status.Timestamp)
		assert.Nil(t, err)

		if current.Before(previous) {
			// ensure that the timestamp always proceeds forward
			t.Fatalf("timestamp must proceed forward: %v < %v", current, previous)
		}
	}

	copy := status.Copy()
	copy.Timestamp = nil // don't check against timestamp
	assert.Equal(t, expected, copy)

	return status
}

func newTestTask(t *testing.T, state, desired api.TaskState) *api.Task {
	return &api.Task{
		ID: "test-task",
		Status: api.TaskStatus{
			State: state,
		},
		DesiredState: desired,
	}
}

func buildTestEnv(t *testing.T, task *api.Task) (context.Context, *StubController, func()) {
	var (
		ctx, cancel = context.WithCancel(context.Background())
		ctlr        = NewStubController()
	)

	// Put test name into log messages. Awesome!
	pc, _, _, ok := runtime.Caller(1)
	if ok {
		fn := runtime.FuncForPC(pc)
		ctx = log.WithLogger(ctx, log.L.WithField("test", fn.Name()))
	}

	return ctx, ctlr, cancel
}

type mockExecutor struct {
	Executor

	err error
}

func (m *mockExecutor) Controller(t *api.Task) (Controller, error) {
	return nil, m.err
}
