package testutils

import (
	"fmt"
	"testing"
	"time"

	"github.com/docker/go-events"
	"github.com/docker/swarmkit/api"
	"github.com/docker/swarmkit/manager/state"
	"github.com/stretchr/testify/assert"
)

// WatchTaskCreate waits for a task to be created.
func WatchTaskCreate(t *testing.T, watch chan events.Event) *api.Task {
	for {
		select {
		case event := <-watch:
			if task, ok := event.(api.EventCreateTask); ok {
				return task.Task
			}
			if _, ok := event.(api.EventUpdateTask); ok {
				assert.FailNow(t, "got EventUpdateTask when expecting EventCreateTask", fmt.Sprint(event))
			}
		case <-time.After(time.Second):
			assert.FailNow(t, "no task creation")
		}
	}
}

// WatchTaskUpdate waits for a task to be updated.
func WatchTaskUpdate(t *testing.T, watch chan events.Event) *api.Task {
	for {
		select {
		case event := <-watch:
			if task, ok := event.(api.EventUpdateTask); ok {
				return task.Task
			}
			if _, ok := event.(api.EventCreateTask); ok {
				assert.FailNow(t, "got EventCreateTask when expecting EventUpdateTask", fmt.Sprint(event))
			}
		case <-time.After(time.Second):
			assert.FailNow(t, "no task update")
		}
	}
}

// WatchTaskDelete waits for a task to be deleted.
func WatchTaskDelete(t *testing.T, watch chan events.Event) *api.Task {
	for {
		select {
		case event := <-watch:
			if task, ok := event.(api.EventDeleteTask); ok {
				return task.Task
			}
		case <-time.After(time.Second):
			assert.FailNow(t, "no task deletion")
		}
	}
}

// WatchShutdownTask fails the test if the next event is not a task having its
// desired state changed to Shutdown.
func WatchShutdownTask(t *testing.T, watch chan events.Event) *api.Task {
	for {
		select {
		case event := <-watch:
			if task, ok := event.(api.EventUpdateTask); ok && task.Task.DesiredState == api.TaskStateShutdown {
				return task.Task
			}
			if _, ok := event.(api.EventCreateTask); ok {
				assert.FailNow(t, "got EventCreateTask when expecting EventUpdateTask", fmt.Sprint(event))
			}
		case <-time.After(time.Second):
			assert.FailNow(t, "no task shutdown")
		}
	}
}

// Expect fails the test if the next event is not one of the specified events.
func Expect(t *testing.T, watch chan events.Event, specifiers ...api.Event) {
	matcher := state.Matcher(specifiers...)
	for {
		select {
		case event := <-watch:
			if !matcher.Match(event) {
				assert.FailNow(t, fmt.Sprintf("unexpected event: %T", event))
			}
			return
		case <-time.After(time.Second):
			assert.FailNow(t, "no matching event")
		}
	}
}
