package replicated

import (
	"sync/atomic"
	"testing"
	"time"

	"github.com/docker/swarmkit/api"
	"github.com/docker/swarmkit/manager/orchestrator/testutils"
	"github.com/docker/swarmkit/manager/state"
	"github.com/docker/swarmkit/manager/state/store"
	gogotypes "github.com/gogo/protobuf/types"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"golang.org/x/net/context"
)

func TestUpdaterRollback(t *testing.T) {
	t.Run("pause/monitor_set/spec_version_unset", func(t *testing.T) { testUpdaterRollback(t, api.UpdateConfig_PAUSE, true, false) })
	t.Run("pause/monitor_set/spec_version_set", func(t *testing.T) { testUpdaterRollback(t, api.UpdateConfig_PAUSE, true, true) })
	// skipped, see #2137
	// t.Run("pause/monitor_unset/spec_version_unset", func(t *testing.T) { testUpdaterRollback(t, api.UpdateConfig_PAUSE, false, false) })
	// t.Run("pause/monitor_unset/spec_version_set", func(t *testing.T) { testUpdaterRollback(t, api.UpdateConfig_PAUSE, false, true) })
	t.Run("continue/spec_version_unset", func(t *testing.T) { testUpdaterRollback(t, api.UpdateConfig_CONTINUE, true, false) })
	t.Run("continue/spec_version_set", func(t *testing.T) { testUpdaterRollback(t, api.UpdateConfig_CONTINUE, true, true) })
}

func testUpdaterRollback(t *testing.T, rollbackFailureAction api.UpdateConfig_FailureAction, setMonitor bool, useSpecVersion bool) {
	ctx := context.Background()
	s := store.NewMemoryStore(nil)
	assert.NotNil(t, s)
	defer s.Close()

	orchestrator := NewReplicatedOrchestrator(s)
	defer orchestrator.Stop()

	var (
		failImage1 uint32
		failImage2 uint32
	)

	watchCreate, cancelCreate := state.Watch(s.WatchQueue(), api.EventCreateTask{})
	defer cancelCreate()

	watchServiceUpdate, cancelServiceUpdate := state.Watch(s.WatchQueue(), api.EventUpdateService{})
	defer cancelServiceUpdate()

	// Fail new tasks the updater tries to run
	watchUpdate, cancelUpdate := state.Watch(s.WatchQueue(), api.EventUpdateTask{})
	defer cancelUpdate()
	go func() {
		failedLast := false
		for {
			select {
			case e := <-watchUpdate:
				task := e.(api.EventUpdateTask).Task
				if task.DesiredState == task.Status.State {
					continue
				}
				if task.DesiredState == api.TaskStateRunning && task.Status.State != api.TaskStateFailed && task.Status.State != api.TaskStateRunning {
					err := s.Update(func(tx store.Tx) error {
						task = store.GetTask(tx, task.ID)
						// Never fail two image2 tasks in a row, so there's a mix of
						// failed and successful tasks for the rollback.
						if task.Spec.GetContainer().Image == "image1" && atomic.LoadUint32(&failImage1) == 1 {
							task.Status.State = api.TaskStateFailed
							failedLast = true
						} else if task.Spec.GetContainer().Image == "image2" && atomic.LoadUint32(&failImage2) == 1 && !failedLast {
							task.Status.State = api.TaskStateFailed
							failedLast = true
						} else {
							task.Status.State = task.DesiredState
							failedLast = false
						}
						return store.UpdateTask(tx, task)
					})
					assert.NoError(t, err)
				} else if task.DesiredState > api.TaskStateRunning {
					err := s.Update(func(tx store.Tx) error {
						task = store.GetTask(tx, task.ID)
						task.Status.State = task.DesiredState
						return store.UpdateTask(tx, task)
					})
					assert.NoError(t, err)
				}
			}
		}
	}()

	// Create a service with four replicas specified before the orchestrator
	// is started. This should result in two tasks when the orchestrator
	// starts up.
	err := s.Update(func(tx store.Tx) error {
		s1 := &api.Service{
			ID: "id1",
			Spec: api.ServiceSpec{
				Annotations: api.Annotations{
					Name: "name1",
				},
				Task: api.TaskSpec{
					Runtime: &api.TaskSpec_Container{
						Container: &api.ContainerSpec{
							Image: "image1",
						},
					},
					Restart: &api.RestartPolicy{
						Condition: api.RestartOnNone,
					},
				},
				Mode: &api.ServiceSpec_Replicated{
					Replicated: &api.ReplicatedService{
						Replicas: 4,
					},
				},
				Update: &api.UpdateConfig{
					FailureAction:   api.UpdateConfig_ROLLBACK,
					Parallelism:     1,
					Delay:           10 * time.Millisecond,
					MaxFailureRatio: 0.4,
				},
				Rollback: &api.UpdateConfig{
					FailureAction:   rollbackFailureAction,
					Parallelism:     1,
					Delay:           10 * time.Millisecond,
					MaxFailureRatio: 0.4,
				},
			},
		}

		if setMonitor {
			s1.Spec.Update.Monitor = gogotypes.DurationProto(500 * time.Millisecond)
			s1.Spec.Rollback.Monitor = gogotypes.DurationProto(500 * time.Millisecond)
		}
		if useSpecVersion {
			s1.SpecVersion = &api.Version{
				Index: 1,
			}
		}

		assert.NoError(t, store.CreateService(tx, s1))
		return nil
	})
	assert.NoError(t, err)

	// Start the orchestrator.
	go func() {
		assert.NoError(t, orchestrator.Run(ctx))
	}()

	observedTask := testutils.WatchTaskCreate(t, watchCreate)
	assert.Equal(t, observedTask.Status.State, api.TaskStateNew)
	assert.Equal(t, observedTask.Spec.GetContainer().Image, "image1")

	observedTask = testutils.WatchTaskCreate(t, watchCreate)
	assert.Equal(t, observedTask.Status.State, api.TaskStateNew)
	assert.Equal(t, observedTask.Spec.GetContainer().Image, "image1")

	observedTask = testutils.WatchTaskCreate(t, watchCreate)
	assert.Equal(t, observedTask.Status.State, api.TaskStateNew)
	assert.Equal(t, observedTask.Spec.GetContainer().Image, "image1")

	observedTask = testutils.WatchTaskCreate(t, watchCreate)
	assert.Equal(t, observedTask.Status.State, api.TaskStateNew)
	assert.Equal(t, observedTask.Spec.GetContainer().Image, "image1")

	atomic.StoreUint32(&failImage2, 1)

	// Start a rolling update
	err = s.Update(func(tx store.Tx) error {
		s1 := store.GetService(tx, "id1")
		require.NotNil(t, s1)
		s1.PreviousSpec = s1.Spec.Copy()
		s1.PreviousSpecVersion = s1.SpecVersion.Copy()
		s1.UpdateStatus = nil
		s1.Spec.Task.GetContainer().Image = "image2"
		if s1.SpecVersion != nil {
			s1.SpecVersion.Index = 2
		}
		assert.NoError(t, store.UpdateService(tx, s1))
		return nil
	})
	assert.NoError(t, err)

	// Should see three tasks started, then a rollback

	observedTask = testutils.WatchTaskCreate(t, watchCreate)
	assert.Equal(t, observedTask.Status.State, api.TaskStateNew)
	assert.Equal(t, observedTask.Spec.GetContainer().Image, "image2")

	observedTask = testutils.WatchTaskCreate(t, watchCreate)
	assert.Equal(t, observedTask.Status.State, api.TaskStateNew)
	assert.Equal(t, observedTask.Spec.GetContainer().Image, "image2")

	observedTask = testutils.WatchTaskCreate(t, watchCreate)
	assert.Equal(t, observedTask.Status.State, api.TaskStateNew)
	assert.Equal(t, observedTask.Spec.GetContainer().Image, "image2")

	// Should get to the ROLLBACK_STARTED state
	for {
		e := <-watchServiceUpdate
		if e.(api.EventUpdateService).Service.UpdateStatus == nil {
			continue
		}
		if e.(api.EventUpdateService).Service.UpdateStatus.State == api.UpdateStatus_ROLLBACK_STARTED {
			break
		}
	}

	observedTask = testutils.WatchTaskCreate(t, watchCreate)
	assert.Equal(t, observedTask.Status.State, api.TaskStateNew)
	assert.Equal(t, observedTask.Spec.GetContainer().Image, "image1")

	observedTask = testutils.WatchTaskCreate(t, watchCreate)
	assert.Equal(t, observedTask.Status.State, api.TaskStateNew)
	assert.Equal(t, observedTask.Spec.GetContainer().Image, "image1")

	observedTask = testutils.WatchTaskCreate(t, watchCreate)
	assert.Equal(t, observedTask.Status.State, api.TaskStateNew)
	assert.Equal(t, observedTask.Spec.GetContainer().Image, "image1")

	if !setMonitor {
		// Exit early in this case, since it would take a long time for
		// the service to reach the "*_COMPLETED" states.
		return
	}

	// Should end up in ROLLBACK_COMPLETED state
	for {
		e := <-watchServiceUpdate
		if e.(api.EventUpdateService).Service.UpdateStatus.State == api.UpdateStatus_ROLLBACK_COMPLETED {
			break
		}
	}

	atomic.StoreUint32(&failImage1, 1)

	// Repeat the rolling update but this time fail the tasks that the
	// rollback creates.
	err = s.Update(func(tx store.Tx) error {
		s1 := store.GetService(tx, "id1")
		require.NotNil(t, s1)
		s1.PreviousSpec = s1.Spec.Copy()
		s1.PreviousSpecVersion = s1.SpecVersion.Copy()
		s1.UpdateStatus = nil
		s1.Spec.Task.GetContainer().Image = "image2"
		if s1.SpecVersion != nil {
			s1.SpecVersion.Index = 2
		}
		assert.NoError(t, store.UpdateService(tx, s1))
		return nil
	})
	assert.NoError(t, err)

	// Should see three tasks started, then a rollback

	observedTask = testutils.WatchTaskCreate(t, watchCreate)
	assert.Equal(t, observedTask.Status.State, api.TaskStateNew)
	assert.Equal(t, observedTask.Spec.GetContainer().Image, "image2")

	observedTask = testutils.WatchTaskCreate(t, watchCreate)
	assert.Equal(t, observedTask.Status.State, api.TaskStateNew)
	assert.Equal(t, observedTask.Spec.GetContainer().Image, "image2")

	observedTask = testutils.WatchTaskCreate(t, watchCreate)
	assert.Equal(t, observedTask.Status.State, api.TaskStateNew)
	assert.Equal(t, observedTask.Spec.GetContainer().Image, "image2")

	// Should get to the ROLLBACK_STARTED state
	for {
		e := <-watchServiceUpdate
		if e.(api.EventUpdateService).Service.UpdateStatus == nil {
			continue
		}
		if e.(api.EventUpdateService).Service.UpdateStatus.State == api.UpdateStatus_ROLLBACK_STARTED {
			break
		}
	}

	observedTask = testutils.WatchTaskCreate(t, watchCreate)
	assert.Equal(t, observedTask.Status.State, api.TaskStateNew)
	assert.Equal(t, observedTask.Spec.GetContainer().Image, "image1")

	observedTask = testutils.WatchTaskCreate(t, watchCreate)
	assert.Equal(t, observedTask.Status.State, api.TaskStateNew)
	assert.Equal(t, observedTask.Spec.GetContainer().Image, "image1")

	observedTask = testutils.WatchTaskCreate(t, watchCreate)
	assert.Equal(t, observedTask.Status.State, api.TaskStateNew)
	assert.Equal(t, observedTask.Spec.GetContainer().Image, "image1")

	switch rollbackFailureAction {
	case api.UpdateConfig_PAUSE:
		// Should end up in ROLLBACK_PAUSED state
		for {
			e := <-watchServiceUpdate
			if e.(api.EventUpdateService).Service.UpdateStatus.State == api.UpdateStatus_ROLLBACK_PAUSED {
				return
			}
		}
	case api.UpdateConfig_CONTINUE:
		// Should end up in ROLLBACK_COMPLETE state
		for {
			e := <-watchServiceUpdate
			if e.(api.EventUpdateService).Service.UpdateStatus.State == api.UpdateStatus_ROLLBACK_COMPLETED {
				return
			}
		}
	}
}
