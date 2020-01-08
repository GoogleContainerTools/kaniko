package agent

import (
	"testing"
	"time"

	"github.com/docker/swarmkit/agent/exec"
	"github.com/docker/swarmkit/api"
	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
	"golang.org/x/net/context"
)

func init() {
	logrus.SetLevel(logrus.DebugLevel)
}

func TestTaskManager(t *testing.T) {
	ctx := context.Background()
	task := &api.Task{
		Status:       api.TaskStatus{},
		DesiredState: api.TaskStateAccepted,
	}
	accepted := make(chan struct{})
	ready := make(chan struct{})
	shutdown := make(chan struct{})
	ctlr := &controllerStub{t: t, calls: map[string]int{}}

	tm := newTaskManager(ctx, task, ctlr, statusReporterFunc(func(ctx context.Context, taskID string, status *api.TaskStatus) error {
		switch status.State {
		case api.TaskStateAccepted:
			select {
			case <-accepted:
			default:
				close(accepted)
			}
		case api.TaskStatePreparing:
		case api.TaskStateReady:
			select {
			case <-ready:
			default:
				close(ready)
			}
		case api.TaskStateStarting:
		case api.TaskStateRunning:
			select {
			case <-ready:
			default:
				t.Fatal("should be running before ready")
			}
		case api.TaskStateCompleted:
			select {
			case <-shutdown:
			default:
				close(shutdown)
			}
		default:
			t.Fatalf("unexpected state encountered: %v", status.State)
		}

		return nil
	}))

	acceptedWait := accepted
	readyWait := ready
	shutdownWait := shutdown
	for {
		select {
		case <-acceptedWait:
			task.DesiredState = api.TaskStateReady // proceed to ready
			assert.NoError(t, tm.Update(ctx, task))
			acceptedWait = nil
		case <-readyWait:
			time.Sleep(time.Second)
			task.DesiredState = api.TaskStateRunning // proceed to running.
			assert.NoError(t, tm.Update(ctx, task))
			readyWait = nil
		case <-shutdownWait:
			assert.NoError(t, tm.Close())
			select {
			case <-tm.closed:
			default:
				t.Fatal("not actually closed")
			}

			assert.NoError(t, tm.Close()) // hit a second time to make sure it behaves
			assert.Equal(t, tm.Update(ctx, task), ErrClosed)

			assert.Equal(t, map[string]int{
				"start":   1,
				"wait":    1,
				"prepare": 1,
				"update":  2}, ctlr.calls)
			return
		case <-ctx.Done():
			t.Fatal(ctx.Err())
		}
	}
}

type controllerStub struct {
	t *testing.T
	exec.Controller

	calls map[string]int
}

func (cs *controllerStub) Prepare(ctx context.Context) error {
	cs.calls["prepare"]++
	cs.t.Log("(*controllerStub).Prepare")
	return nil
}

func (cs *controllerStub) Start(ctx context.Context) error {
	cs.calls["start"]++
	cs.t.Log("(*controllerStub).Start")
	return nil
}

func (cs *controllerStub) Wait(ctx context.Context) error {
	cs.calls["wait"]++
	cs.t.Log("(*controllerStub).Wait")
	return nil
}

func (cs *controllerStub) Update(ctx context.Context, task *api.Task) error {
	cs.calls["update"]++
	cs.t.Log("(*controllerStub).Update")
	return nil
}

func (cs *controllerStub) Remove(ctx context.Context) error {
	cs.calls["remove"]++
	cs.t.Log("(*controllerStub).Remove")
	return nil
}

func (cs *controllerStub) Close() error {
	cs.calls["close"]++
	cs.t.Log("(*controllerStub).Close")
	return nil
}
