package global

import (
	"testing"
	"time"

	"github.com/docker/go-events"
	"github.com/docker/swarmkit/api"
	"github.com/docker/swarmkit/manager/orchestrator/testutils"
	"github.com/docker/swarmkit/manager/state"
	"github.com/docker/swarmkit/manager/state/store"
	"github.com/docker/swarmkit/protobuf/ptypes"
	gogotypes "github.com/gogo/protobuf/types"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"golang.org/x/net/context"
)

var (
	node1 = &api.Node{
		ID: "nodeid1",
		Spec: api.NodeSpec{
			Annotations: api.Annotations{
				Name: "name1",
			},
			Availability: api.NodeAvailabilityActive,
		},
		Status: api.NodeStatus{
			State: api.NodeStatus_READY,
		},
		Description: &api.NodeDescription{
			Hostname: "name1",
		},
		Role: api.NodeRoleWorker,
	}
	node2 = &api.Node{
		ID: "nodeid2",
		Spec: api.NodeSpec{
			Annotations: api.Annotations{
				Name: "name2",
			},
			Availability: api.NodeAvailabilityActive,
		},
		Status: api.NodeStatus{
			State: api.NodeStatus_READY,
		},
		Description: &api.NodeDescription{
			Hostname: "name2",
		},
		Role: api.NodeRoleWorker,
	}

	restartDelay = 50 * time.Millisecond

	service1 = &api.Service{
		ID: "serviceid1",
		Spec: api.ServiceSpec{
			Annotations: api.Annotations{
				Name: "name1",
			},
			Task: api.TaskSpec{
				Runtime: &api.TaskSpec_Container{
					Container: &api.ContainerSpec{},
				},
				Restart: &api.RestartPolicy{
					Condition: api.RestartOnAny,
					Delay:     gogotypes.DurationProto(restartDelay),
				},
			},
			Mode: &api.ServiceSpec_Global{
				Global: &api.GlobalService{},
			},
		},
	}

	service2 = &api.Service{
		ID: "serviceid2",
		Spec: api.ServiceSpec{
			Annotations: api.Annotations{
				Name: "name2",
			},
			Task: api.TaskSpec{
				Runtime: &api.TaskSpec_Container{
					Container: &api.ContainerSpec{},
				},
			},
			Mode: &api.ServiceSpec_Global{
				Global: &api.GlobalService{},
			},
		},
	}

	serviceNoRestart = &api.Service{
		ID: "serviceid3",
		Spec: api.ServiceSpec{
			Annotations: api.Annotations{
				Name: "norestart",
			},
			Task: api.TaskSpec{
				Runtime: &api.TaskSpec_Container{
					Container: &api.ContainerSpec{},
				},
				Restart: &api.RestartPolicy{
					Condition: api.RestartOnNone,
				},
			},
			Mode: &api.ServiceSpec_Global{
				Global: &api.GlobalService{},
			},
		},
	}
)

func setup(t *testing.T, store *store.MemoryStore, watch chan events.Event) *Orchestrator {
	ctx := context.Background()
	// Start the global orchestrator.
	global := NewGlobalOrchestrator(store)
	go func() {
		assert.NoError(t, global.Run(ctx))
	}()

	addService(t, store, service1)
	testutils.Expect(t, watch, api.EventCreateService{})
	testutils.Expect(t, watch, state.EventCommit{})

	addNode(t, store, node1)
	testutils.Expect(t, watch, api.EventCreateNode{})
	testutils.Expect(t, watch, state.EventCommit{})

	return global
}

func TestSetup(t *testing.T) {
	store := store.NewMemoryStore(nil)
	assert.NotNil(t, store)
	defer store.Close()

	watch, cancel := state.Watch(store.WatchQueue() /*state.EventCreateTask{}, state.EventUpdateTask{}*/)
	defer cancel()

	orchestrator := setup(t, store, watch)
	defer orchestrator.Stop()

	observedTask1 := testutils.WatchTaskCreate(t, watch)

	assert.Equal(t, observedTask1.Status.State, api.TaskStateNew)
	assert.Equal(t, observedTask1.ServiceAnnotations.Name, "name1")
	assert.Equal(t, observedTask1.NodeID, "nodeid1")
}

func TestAddNode(t *testing.T) {
	store := store.NewMemoryStore(nil)
	assert.NotNil(t, store)
	defer store.Close()

	watch, cancel := state.Watch(store.WatchQueue())
	defer cancel()

	orchestrator := setup(t, store, watch)
	defer orchestrator.Stop()

	testutils.WatchTaskCreate(t, watch)

	addNode(t, store, node2)
	observedTask2 := testutils.WatchTaskCreate(t, watch)
	assert.Equal(t, observedTask2.Status.State, api.TaskStateNew)
	assert.Equal(t, observedTask2.ServiceAnnotations.Name, "name1")
	assert.Equal(t, observedTask2.NodeID, "nodeid2")
}

func TestDeleteNode(t *testing.T) {
	store := store.NewMemoryStore(nil)
	assert.NotNil(t, store)
	defer store.Close()

	watch, cancel := state.Watch(store.WatchQueue())
	defer cancel()

	orchestrator := setup(t, store, watch)
	defer orchestrator.Stop()

	testutils.WatchTaskCreate(t, watch)

	deleteNode(t, store, node1)
	// task should be set to dead
	observedTask := testutils.WatchTaskDelete(t, watch)
	assert.Equal(t, observedTask.ServiceAnnotations.Name, "name1")
	assert.Equal(t, observedTask.NodeID, "nodeid1")
}

func TestNodeAvailability(t *testing.T) {
	t.Parallel()

	store := store.NewMemoryStore(nil)
	assert.NotNil(t, store)
	defer store.Close()

	watch, cancel := state.Watch(store.WatchQueue())
	defer cancel()

	orchestrator := setup(t, store, watch)
	defer orchestrator.Stop()

	testutils.WatchTaskCreate(t, watch)

	// set node1 to drain
	updateNodeAvailability(t, store, node1, api.NodeAvailabilityDrain)

	// task should be set to dead
	observedTask1 := testutils.WatchShutdownTask(t, watch)
	assert.Equal(t, observedTask1.ServiceAnnotations.Name, "name1")
	assert.Equal(t, observedTask1.NodeID, "nodeid1")
	testutils.Expect(t, watch, state.EventCommit{})

	// updating the service shouldn't restart the task
	updateService(t, store, service1, true)
	testutils.Expect(t, watch, api.EventUpdateService{})
	testutils.Expect(t, watch, state.EventCommit{})
	select {
	case event := <-watch:
		t.Fatalf("got unexpected event %T: %+v", event, event)
	case <-time.After(100 * time.Millisecond):
	}

	// set node1 to active
	updateNodeAvailability(t, store, node1, api.NodeAvailabilityActive)
	// task should be added back
	observedTask2 := testutils.WatchTaskCreate(t, watch)
	assert.Equal(t, observedTask2.Status.State, api.TaskStateNew)
	assert.Equal(t, observedTask2.ServiceAnnotations.Name, "name1")
	assert.Equal(t, observedTask2.NodeID, "nodeid1")
	testutils.Expect(t, watch, state.EventCommit{})

	// set node1 to pause
	updateNodeAvailability(t, store, node1, api.NodeAvailabilityPause)

	failTask(t, store, observedTask2)
	observedTask3 := testutils.WatchShutdownTask(t, watch)
	assert.Equal(t, observedTask3.ServiceAnnotations.Name, "name1")
	assert.Equal(t, observedTask3.NodeID, "nodeid1")
	testutils.Expect(t, watch, state.EventCommit{})

	// updating the service shouldn't restart the task
	updateService(t, store, service1, true)
	testutils.Expect(t, watch, api.EventUpdateService{})
	testutils.Expect(t, watch, state.EventCommit{})
	select {
	case event := <-watch:
		t.Fatalf("got unexpected event %T: %+v", event, event)
	case <-time.After(100 * time.Millisecond):
	}

}

func TestNodeState(t *testing.T) {
	t.Parallel()

	store := store.NewMemoryStore(nil)
	assert.NotNil(t, store)
	defer store.Close()

	watch, cancel := state.Watch(store.WatchQueue())
	defer cancel()

	orchestrator := setup(t, store, watch)
	defer orchestrator.Stop()

	testutils.WatchTaskCreate(t, watch)

	// set node1 to down
	updateNodeState(t, store, node1, api.NodeStatus_DOWN)

	// task should be set to dead
	observedTask1 := testutils.WatchShutdownTask(t, watch)
	assert.Equal(t, observedTask1.ServiceAnnotations.Name, "name1")
	assert.Equal(t, observedTask1.NodeID, "nodeid1")
	testutils.Expect(t, watch, state.EventCommit{})

	// updating the service shouldn't restart the task
	updateService(t, store, service1, true)
	testutils.Expect(t, watch, api.EventUpdateService{})
	testutils.Expect(t, watch, state.EventCommit{})
	select {
	case event := <-watch:
		t.Fatalf("got unexpected event %T: %+v", event, event)
	case <-time.After(100 * time.Millisecond):
	}

	// set node1 to ready
	updateNodeState(t, store, node1, api.NodeStatus_READY)
	// task should be added back
	observedTask2 := testutils.WatchTaskCreate(t, watch)
	assert.Equal(t, observedTask2.Status.State, api.TaskStateNew)
	assert.Equal(t, observedTask2.ServiceAnnotations.Name, "name1")
	assert.Equal(t, observedTask2.NodeID, "nodeid1")
}

func TestAddService(t *testing.T) {
	store := store.NewMemoryStore(nil)
	assert.NotNil(t, store)
	defer store.Close()

	watch, cancel := state.Watch(store.WatchQueue())
	defer cancel()

	orchestrator := setup(t, store, watch)
	defer orchestrator.Stop()

	testutils.WatchTaskCreate(t, watch)

	addService(t, store, service2)
	observedTask := testutils.WatchTaskCreate(t, watch)
	assert.Equal(t, observedTask.Status.State, api.TaskStateNew)
	assert.Equal(t, observedTask.ServiceAnnotations.Name, "name2")
	assert.True(t, observedTask.NodeID == "nodeid1")
}

func TestDeleteService(t *testing.T) {
	store := store.NewMemoryStore(nil)
	assert.NotNil(t, store)
	defer store.Close()

	watch, cancel := state.Watch(store.WatchQueue())
	defer cancel()

	orchestrator := setup(t, store, watch)
	defer orchestrator.Stop()

	testutils.WatchTaskCreate(t, watch)

	deleteService(t, store, service1)
	// task should be deleted
	observedTask := testutils.WatchTaskUpdate(t, watch)
	assert.Equal(t, observedTask.ServiceAnnotations.Name, "name1")
	assert.Equal(t, observedTask.NodeID, "nodeid1")
}

func TestRemoveTask(t *testing.T) {
	t.Parallel()

	store := store.NewMemoryStore(nil)
	assert.NotNil(t, store)
	defer store.Close()

	watch, cancel := state.Watch(store.WatchQueue() /*api.EventCreateTask{}, api.EventUpdateTask{}*/)
	defer cancel()

	orchestrator := setup(t, store, watch)
	defer orchestrator.Stop()

	observedTask1 := testutils.WatchTaskCreate(t, watch)
	testutils.Expect(t, watch, state.EventCommit{})

	assert.Equal(t, observedTask1.Status.State, api.TaskStateNew)
	assert.Equal(t, observedTask1.ServiceAnnotations.Name, "name1")
	assert.Equal(t, observedTask1.NodeID, "nodeid1")

	deleteTask(t, store, observedTask1)
	testutils.Expect(t, watch, api.EventDeleteTask{})
	testutils.Expect(t, watch, state.EventCommit{})

	// the task should not be recreated
	select {
	case event := <-watch:
		t.Fatalf("got unexpected event %T: %+v", event, event)
	case <-time.After(100 * time.Millisecond):
	}
}

func TestTaskFailure(t *testing.T) {
	t.Parallel()

	store := store.NewMemoryStore(nil)
	assert.NotNil(t, store)
	defer store.Close()

	watch, cancel := state.Watch(store.WatchQueue() /*api.EventCreateTask{}, api.EventUpdateTask{}*/)
	defer cancel()

	// first, try a "restart on any" policy
	orchestrator := setup(t, store, watch)
	defer orchestrator.Stop()

	observedTask1 := testutils.WatchTaskCreate(t, watch)

	assert.Equal(t, observedTask1.Status.State, api.TaskStateNew)
	assert.Equal(t, observedTask1.ServiceAnnotations.Name, "name1")
	assert.Equal(t, observedTask1.NodeID, "nodeid1")

	failTask(t, store, observedTask1)

	testutils.WatchShutdownTask(t, watch)

	// the task should be recreated
	observedTask2 := testutils.WatchTaskCreate(t, watch)
	assert.Equal(t, observedTask2.Status.State, api.TaskStateNew)
	assert.Equal(t, observedTask2.ServiceAnnotations.Name, "name1")
	assert.Equal(t, observedTask2.NodeID, "nodeid1")
	testutils.Expect(t, watch, state.EventCommit{})
	testutils.Expect(t, watch, api.EventUpdateTask{}) // ready->running
	testutils.Expect(t, watch, state.EventCommit{})

	// repeat with service set up not to restart
	addService(t, store, serviceNoRestart)
	testutils.Expect(t, watch, api.EventCreateService{})
	testutils.Expect(t, watch, state.EventCommit{})

	observedTask3 := testutils.WatchTaskCreate(t, watch)
	assert.Equal(t, observedTask3.Status.State, api.TaskStateNew)
	assert.Equal(t, observedTask3.ServiceAnnotations.Name, "norestart")
	assert.Equal(t, observedTask3.NodeID, "nodeid1")
	testutils.Expect(t, watch, state.EventCommit{})

	failTask(t, store, observedTask3)
	testutils.Expect(t, watch, api.EventUpdateTask{})
	testutils.Expect(t, watch, state.EventCommit{})
	observedTask4 := testutils.WatchTaskUpdate(t, watch)
	assert.Equal(t, observedTask4.DesiredState, api.TaskStateShutdown)
	testutils.Expect(t, watch, state.EventCommit{})

	// the task should not be recreated
	select {
	case event := <-watch:
		t.Fatalf("got unexpected event %T: %+v", event, event)
	case <-time.After(100 * time.Millisecond):
	}

	// update the service with no spec changes, to trigger a
	// reconciliation. the task should still not be updated.
	updateService(t, store, serviceNoRestart, false)
	testutils.Expect(t, watch, api.EventUpdateService{})
	testutils.Expect(t, watch, state.EventCommit{})

	select {
	case event := <-watch:
		t.Fatalf("got unexpected event %T: %+v", event, event)
	case <-time.After(100 * time.Millisecond):
	}

	// update the service with spec changes. now the task should be recreated.
	updateService(t, store, serviceNoRestart, true)
	testutils.Expect(t, watch, api.EventUpdateService{})
	testutils.Expect(t, watch, state.EventCommit{})

	observedTask5 := testutils.WatchTaskCreate(t, watch)
	assert.Equal(t, observedTask5.Status.State, api.TaskStateNew)
	assert.Equal(t, observedTask5.ServiceAnnotations.Name, "norestart")
	assert.Equal(t, observedTask5.NodeID, "nodeid1")
	testutils.Expect(t, watch, state.EventCommit{})
}

func addService(t *testing.T, s *store.MemoryStore, service *api.Service) {
	s.Update(func(tx store.Tx) error {
		assert.NoError(t, store.CreateService(tx, service.Copy()))
		return nil
	})
}

func updateService(t *testing.T, s *store.MemoryStore, service *api.Service, force bool) {
	s.Update(func(tx store.Tx) error {
		service := store.GetService(tx, service.ID)
		require.NotNil(t, service)
		if force {
			service.Spec.Task.ForceUpdate++
		}
		assert.NoError(t, store.UpdateService(tx, service))
		return nil
	})
}

func deleteService(t *testing.T, s *store.MemoryStore, service *api.Service) {
	s.Update(func(tx store.Tx) error {
		assert.NoError(t, store.DeleteService(tx, service.ID))
		return nil
	})
}

func addNode(t *testing.T, s *store.MemoryStore, node *api.Node) {
	s.Update(func(tx store.Tx) error {
		assert.NoError(t, store.CreateNode(tx, node.Copy()))
		return nil
	})
}

func updateNodeAvailability(t *testing.T, s *store.MemoryStore, node *api.Node, avail api.NodeSpec_Availability) {
	s.Update(func(tx store.Tx) error {
		node := store.GetNode(tx, node.ID)
		require.NotNil(t, node)
		node.Spec.Availability = avail
		assert.NoError(t, store.UpdateNode(tx, node))
		return nil
	})
}

func updateNodeState(t *testing.T, s *store.MemoryStore, node *api.Node, state api.NodeStatus_State) {
	s.Update(func(tx store.Tx) error {
		node := store.GetNode(tx, node.ID)
		require.NotNil(t, node)
		node.Status.State = state
		assert.NoError(t, store.UpdateNode(tx, node))
		return nil
	})
}

func deleteNode(t *testing.T, s *store.MemoryStore, node *api.Node) {
	s.Update(func(tx store.Tx) error {
		assert.NoError(t, store.DeleteNode(tx, node.ID))
		return nil
	})
}

func addTask(t *testing.T, s *store.MemoryStore, task *api.Task) {
	s.Update(func(tx store.Tx) error {
		assert.NoError(t, store.CreateTask(tx, task))
		return nil
	})
}

func deleteTask(t *testing.T, s *store.MemoryStore, task *api.Task) {
	s.Update(func(tx store.Tx) error {
		assert.NoError(t, store.DeleteTask(tx, task.ID))
		return nil
	})
}

func failTask(t *testing.T, s *store.MemoryStore, task *api.Task) {
	s.Update(func(tx store.Tx) error {
		task := store.GetTask(tx, task.ID)
		require.NotNil(t, task)
		task.Status.State = api.TaskStateFailed
		assert.NoError(t, store.UpdateTask(tx, task))
		return nil
	})
}

func TestInitializationRejectedTasks(t *testing.T) {
	ctx := context.Background()
	s := store.NewMemoryStore(nil)
	assert.NotNil(t, s)
	defer s.Close()

	// create nodes, services and tasks in store directly
	// where orchestrator runs, it should fix tasks to declarative state
	addNode(t, s, node1)
	addService(t, s, service1)
	tasks := []*api.Task{
		// nodeid1 has a rejected task for serviceid1
		{
			ID:           "task1",
			Slot:         0,
			DesiredState: api.TaskStateReady,
			Status: api.TaskStatus{
				State: api.TaskStateRejected,
			},
			Spec: api.TaskSpec{
				Runtime: &api.TaskSpec_Container{
					Container: &api.ContainerSpec{},
				},
				Restart: &api.RestartPolicy{
					Condition: api.RestartOnAny,
					Delay:     gogotypes.DurationProto(restartDelay),
				},
			},
			ServiceAnnotations: api.Annotations{
				Name: "task1",
			},
			ServiceID: "serviceid1",
			NodeID:    "nodeid1",
		},
	}
	for _, task := range tasks {
		addTask(t, s, task)
	}

	// watch orchestration events
	watch, cancel := state.Watch(s.WatchQueue(), api.EventCreateTask{}, api.EventUpdateTask{}, api.EventDeleteTask{})
	defer cancel()

	orchestrator := NewGlobalOrchestrator(s)
	defer orchestrator.Stop()

	go func() {
		assert.NoError(t, orchestrator.Run(ctx))
	}()

	observedTask1 := testutils.WatchTaskUpdate(t, watch)
	assert.Equal(t, observedTask1.ID, "task1")
	assert.Equal(t, observedTask1.Status.State, api.TaskStateRejected)
	assert.Equal(t, observedTask1.DesiredState, api.TaskStateShutdown)

	observedTask2 := testutils.WatchTaskCreate(t, watch)
	assert.Equal(t, observedTask2.NodeID, "nodeid1")
	assert.Equal(t, observedTask2.Status.State, api.TaskStateNew)
	assert.Equal(t, observedTask2.DesiredState, api.TaskStateReady)
}

func TestInitializationFailedTasks(t *testing.T) {
	ctx := context.Background()
	s := store.NewMemoryStore(nil)
	assert.NotNil(t, s)
	defer s.Close()

	// create nodes, services and tasks in store directly
	// where orchestrator runs, it should fix tasks to declarative state
	addNode(t, s, node1)
	addService(t, s, service1)
	before := time.Now()
	tasks := []*api.Task{
		// nodeid1 has a failed task for serviceid1
		{
			ID:           "task1",
			Slot:         0,
			DesiredState: api.TaskStateRunning,
			Status: api.TaskStatus{
				State:     api.TaskStateFailed,
				Timestamp: ptypes.MustTimestampProto(before),
			},
			Spec: api.TaskSpec{
				Runtime: &api.TaskSpec_Container{
					Container: &api.ContainerSpec{},
				},
				Restart: &api.RestartPolicy{
					Condition: api.RestartOnAny,
					Delay:     gogotypes.DurationProto(restartDelay),
				},
			},
			ServiceAnnotations: api.Annotations{
				Name: "task1",
			},
			ServiceID: "serviceid1",
			NodeID:    "nodeid1",
		},
	}
	for _, task := range tasks {
		addTask(t, s, task)
	}

	// watch orchestration events
	watch, cancel := state.Watch(s.WatchQueue(), api.EventCreateTask{}, api.EventUpdateTask{}, api.EventDeleteTask{})
	defer cancel()

	orchestrator := NewGlobalOrchestrator(s)
	defer orchestrator.Stop()

	go func() {
		assert.NoError(t, orchestrator.Run(ctx))
	}()

	observedTask1 := testutils.WatchTaskUpdate(t, watch)
	assert.Equal(t, observedTask1.ID, "task1")
	assert.Equal(t, observedTask1.Status.State, api.TaskStateFailed)
	assert.Equal(t, observedTask1.DesiredState, api.TaskStateShutdown)

	observedTask2 := testutils.WatchTaskCreate(t, watch)
	assert.Equal(t, observedTask2.NodeID, "nodeid1")
	assert.Equal(t, observedTask2.Status.State, api.TaskStateNew)
	assert.Equal(t, observedTask2.DesiredState, api.TaskStateReady)

	observedTask3 := testutils.WatchTaskUpdate(t, watch)
	after := time.Now()
	assert.Equal(t, observedTask3.NodeID, "nodeid1")
	assert.Equal(t, observedTask3.DesiredState, api.TaskStateRunning)

	if after.Sub(before) < restartDelay {
		t.Fatalf("restart delay should have elapsed. Got: %v", after.Sub(before))
	}
}

func TestInitializationExtraTask(t *testing.T) {
	ctx := context.Background()
	s := store.NewMemoryStore(nil)
	assert.NotNil(t, s)
	defer s.Close()

	// create nodes, services and tasks in store directly
	// where orchestrator runs, it should fix tasks to declarative state
	addNode(t, s, node1)
	addService(t, s, service1)
	tasks := []*api.Task{
		// nodeid1 has 2 tasks for serviceid1
		{
			ID:           "task1",
			Slot:         0,
			DesiredState: api.TaskStateRunning,
			Status: api.TaskStatus{
				State: api.TaskStateRunning,
			},
			Spec: api.TaskSpec{
				Runtime: &api.TaskSpec_Container{
					Container: &api.ContainerSpec{},
				},
				Restart: &api.RestartPolicy{
					Condition: api.RestartOnAny,
					Delay:     gogotypes.DurationProto(restartDelay),
				},
			},
			ServiceAnnotations: api.Annotations{
				Name: "task1",
			},
			ServiceID: "serviceid1",
			NodeID:    "nodeid1",
		},
		{
			ID:           "task2",
			Slot:         0,
			DesiredState: api.TaskStateRunning,
			Status: api.TaskStatus{
				State: api.TaskStateRunning,
			},
			Spec: api.TaskSpec{
				Runtime: &api.TaskSpec_Container{
					Container: &api.ContainerSpec{},
				},
				Restart: &api.RestartPolicy{
					Condition: api.RestartOnAny,
					Delay:     gogotypes.DurationProto(restartDelay),
				},
			},
			ServiceAnnotations: api.Annotations{
				Name: "task2",
			},
			ServiceID: "serviceid1",
			NodeID:    "nodeid1",
		},
	}
	for _, task := range tasks {
		addTask(t, s, task)
	}

	// watch orchestration events
	watch, cancel := state.Watch(s.WatchQueue(), api.EventCreateTask{}, api.EventUpdateTask{}, api.EventDeleteTask{})
	defer cancel()

	orchestrator := NewGlobalOrchestrator(s)
	defer orchestrator.Stop()

	go func() {
		assert.NoError(t, orchestrator.Run(ctx))
	}()

	observedTask1 := testutils.WatchTaskUpdate(t, watch)
	assert.True(t, observedTask1.ID == "task1" || observedTask1.ID == "task2")
	assert.Equal(t, observedTask1.Status.State, api.TaskStateRunning)
	assert.Equal(t, observedTask1.DesiredState, api.TaskStateShutdown)

	var deadCnt, liveCnt int
	var err error
	s.View(func(readTx store.ReadTx) {
		var tasks []*api.Task
		tasks, err = store.FindTasks(readTx, store.ByServiceID("serviceid1"))
		for _, task := range tasks {
			if task.DesiredState == api.TaskStateShutdown {
				assert.Equal(t, task.ID, observedTask1.ID)
				deadCnt++
			} else {
				assert.Equal(t, task.DesiredState, api.TaskStateRunning)
				liveCnt++
			}
		}
	})
	assert.NoError(t, err)
	assert.Equal(t, deadCnt, 1)
	assert.Equal(t, liveCnt, 1)
}

func TestInitializationMultipleServices(t *testing.T) {
	ctx := context.Background()
	s := store.NewMemoryStore(nil)
	assert.NotNil(t, s)
	defer s.Close()

	// create nodes, services and tasks in store directly
	// where orchestrator runs, it should fix tasks to declarative state
	addNode(t, s, node1)
	addService(t, s, service1)
	addService(t, s, service2)
	tasks := []*api.Task{
		// nodeid1 has 1 task for serviceid1 and 1 task for serviceid2
		{
			ID:           "task1",
			DesiredState: api.TaskStateRunning,
			Status: api.TaskStatus{
				State: api.TaskStateRunning,
			},
			Spec: service1.Spec.Task,
			ServiceAnnotations: api.Annotations{
				Name: "task1",
			},
			ServiceID: "serviceid1",
			NodeID:    "nodeid1",
		},
		{
			ID:           "task2",
			DesiredState: api.TaskStateRunning,
			Status: api.TaskStatus{
				State: api.TaskStateRunning,
			},
			Spec: service2.Spec.Task,
			ServiceAnnotations: api.Annotations{
				Name: "task2",
			},
			ServiceID: "serviceid2",
			NodeID:    "nodeid1",
		},
	}
	for _, task := range tasks {
		addTask(t, s, task)
	}

	// watch orchestration events
	watch, cancel := state.Watch(s.WatchQueue(), api.EventCreateTask{}, api.EventUpdateTask{}, api.EventDeleteTask{})
	defer cancel()

	orchestrator := NewGlobalOrchestrator(s)
	defer orchestrator.Stop()

	go func() {
		assert.NoError(t, orchestrator.Run(ctx))
	}()

	// Nothing should happen because both tasks are up to date.
	select {
	case e := <-watch:
		t.Fatalf("Received unexpected event (type: %T) %+v", e, e)
	case <-time.After(100 * time.Millisecond):
	}

	// Update service 1. Make sure only service 1's task is restarted.

	s.Update(func(tx store.Tx) error {
		s1 := store.GetService(tx, "serviceid1")
		require.NotNil(t, s1)

		s1.Spec.Task.Restart.Delay = gogotypes.DurationProto(70 * time.Millisecond)

		assert.NoError(t, store.UpdateService(tx, s1))
		return nil
	})

	observedCreation1 := testutils.WatchTaskCreate(t, watch)
	assert.Equal(t, "serviceid1", observedCreation1.ServiceID)
	assert.Equal(t, "nodeid1", observedCreation1.NodeID)
	assert.Equal(t, api.TaskStateReady, observedCreation1.DesiredState)

	observedUpdate1 := testutils.WatchTaskUpdate(t, watch)
	assert.Equal(t, "serviceid1", observedUpdate1.ServiceID)
	assert.Equal(t, "nodeid1", observedUpdate1.NodeID)
	assert.Equal(t, api.TaskStateShutdown, observedUpdate1.DesiredState)

	// Nothing else should happen
	select {
	case e := <-watch:
		t.Fatalf("Received unexpected event (type: %T) %+v", e, e)
	case <-time.After(100 * time.Millisecond):
	}

	// Fail a task from service 2. Make sure only service 2's task is restarted.

	s.Update(func(tx store.Tx) error {
		t2 := store.GetTask(tx, "task2")
		require.NotNil(t, t2)

		t2.Status.State = api.TaskStateFailed

		assert.NoError(t, store.UpdateTask(tx, t2))
		return nil
	})

	// Consume our own task update event
	<-watch

	observedUpdate2 := testutils.WatchTaskUpdate(t, watch)
	assert.Equal(t, "serviceid2", observedUpdate2.ServiceID)
	assert.Equal(t, "nodeid1", observedUpdate2.NodeID)
	assert.Equal(t, api.TaskStateShutdown, observedUpdate2.DesiredState)

	observedCreation2 := testutils.WatchTaskCreate(t, watch)
	assert.Equal(t, "serviceid2", observedCreation2.ServiceID)
	assert.Equal(t, "nodeid1", observedCreation2.NodeID)
	assert.Equal(t, api.TaskStateReady, observedCreation2.DesiredState)

	// Nothing else should happen
	select {
	case e := <-watch:
		t.Fatalf("Received unexpected event (type: %T) %+v", e, e)
	case <-time.After(100 * time.Millisecond):
	}
}

func TestInitializationTaskWithoutService(t *testing.T) {
	ctx := context.Background()
	s := store.NewMemoryStore(nil)
	assert.NotNil(t, s)
	defer s.Close()

	// create nodes, services and tasks in store directly
	// where orchestrator runs, it should fix tasks to declarative state
	addNode(t, s, node1)
	addService(t, s, service1)
	tasks := []*api.Task{
		// nodeid1 has 1 task for serviceid1 and 1 task for serviceid2
		{
			ID:           "task1",
			Slot:         0,
			DesiredState: api.TaskStateRunning,
			Status: api.TaskStatus{
				State: api.TaskStateRunning,
			},
			Spec: api.TaskSpec{
				Runtime: &api.TaskSpec_Container{
					Container: &api.ContainerSpec{},
				},
				Restart: &api.RestartPolicy{
					Condition: api.RestartOnAny,
					Delay:     gogotypes.DurationProto(restartDelay),
				},
			},
			ServiceAnnotations: api.Annotations{
				Name: "task1",
			},
			ServiceID: "serviceid1",
			NodeID:    "nodeid1",
		},
		{
			ID:           "task2",
			Slot:         0,
			DesiredState: api.TaskStateRunning,
			Status: api.TaskStatus{
				State: api.TaskStateRunning,
			},
			Spec: api.TaskSpec{
				Runtime: &api.TaskSpec_Container{
					Container: &api.ContainerSpec{},
				},
				Restart: &api.RestartPolicy{
					Condition: api.RestartOnAny,
					Delay:     gogotypes.DurationProto(restartDelay),
				},
			},
			ServiceAnnotations: api.Annotations{
				Name: "task2",
			},
			ServiceID: "serviceid2",
			NodeID:    "nodeid1",
		},
	}
	for _, task := range tasks {
		addTask(t, s, task)
	}

	// watch orchestration events
	watch, cancel := state.Watch(s.WatchQueue(), api.EventCreateTask{}, api.EventUpdateTask{}, api.EventDeleteTask{})
	defer cancel()

	orchestrator := NewGlobalOrchestrator(s)
	defer orchestrator.Stop()

	go func() {
		assert.NoError(t, orchestrator.Run(ctx))
	}()

	observedTask1 := testutils.WatchTaskDelete(t, watch)
	assert.Equal(t, observedTask1.ID, "task2")
	assert.Equal(t, observedTask1.Status.State, api.TaskStateRunning)
	assert.Equal(t, observedTask1.DesiredState, api.TaskStateRunning)
}

func TestInitializationTaskOnDrainedNode(t *testing.T) {
	ctx := context.Background()
	s := store.NewMemoryStore(nil)
	assert.NotNil(t, s)
	defer s.Close()

	// create nodes, services and tasks in store directly
	// where orchestrator runs, it should fix tasks to declarative state
	n1 := *node1
	n1.Spec.Availability = api.NodeAvailabilityDrain
	addNode(t, s, &n1)
	addService(t, s, service1)
	tasks := []*api.Task{
		// nodeid1 has 1 task for serviceid1
		{
			ID:           "task1",
			Slot:         0,
			DesiredState: api.TaskStateRunning,
			Status: api.TaskStatus{
				State: api.TaskStateRunning,
			},
			Spec: api.TaskSpec{
				Runtime: &api.TaskSpec_Container{
					Container: &api.ContainerSpec{},
				},
				Restart: &api.RestartPolicy{
					Condition: api.RestartOnAny,
					Delay:     gogotypes.DurationProto(restartDelay),
				},
			},
			ServiceAnnotations: api.Annotations{
				Name: "task1",
			},
			ServiceID: "serviceid1",
			NodeID:    "nodeid1",
		},
	}
	for _, task := range tasks {
		addTask(t, s, task)
	}

	// watch orchestration events
	watch, cancel := state.Watch(s.WatchQueue(), api.EventCreateTask{}, api.EventUpdateTask{}, api.EventDeleteTask{})
	defer cancel()

	orchestrator := NewGlobalOrchestrator(s)
	defer orchestrator.Stop()

	go func() {
		assert.NoError(t, orchestrator.Run(ctx))
	}()

	observedTask1 := testutils.WatchTaskUpdate(t, watch)
	assert.Equal(t, observedTask1.ID, "task1")
	assert.Equal(t, observedTask1.Status.State, api.TaskStateRunning)
	assert.Equal(t, observedTask1.DesiredState, api.TaskStateShutdown)

	var deadCnt, liveCnt int
	var err error
	s.View(func(readTx store.ReadTx) {
		var tasks []*api.Task
		tasks, err = store.FindTasks(readTx, store.ByServiceID("serviceid1"))
		for _, task := range tasks {
			if task.DesiredState == api.TaskStateShutdown {
				deadCnt++
			} else {
				liveCnt++
			}
		}
	})
	assert.NoError(t, err)
	assert.Equal(t, deadCnt, 1)
	assert.Equal(t, liveCnt, 0)
}

func TestInitializationTaskOnNonexistentNode(t *testing.T) {
	ctx := context.Background()
	s := store.NewMemoryStore(nil)
	assert.NotNil(t, s)
	defer s.Close()

	// create nodes, services and tasks in store directly
	// where orchestrator runs, it should fix tasks to declarative state
	addService(t, s, service1)
	tasks := []*api.Task{
		// 1 task for serviceid1 on nonexistent nodeid1
		{
			ID:           "task1",
			Slot:         0,
			DesiredState: api.TaskStateRunning,
			Status: api.TaskStatus{
				State: api.TaskStateRunning,
			},
			Spec: api.TaskSpec{
				Runtime: &api.TaskSpec_Container{
					Container: &api.ContainerSpec{},
				},
				Restart: &api.RestartPolicy{
					Condition: api.RestartOnAny,
					Delay:     gogotypes.DurationProto(restartDelay),
				},
			},
			ServiceAnnotations: api.Annotations{
				Name: "task1",
			},
			ServiceID: "serviceid1",
			NodeID:    "nodeid1",
		},
	}
	for _, task := range tasks {
		addTask(t, s, task)
	}

	// watch orchestration events
	watch, cancel := state.Watch(s.WatchQueue(), api.EventCreateTask{}, api.EventUpdateTask{}, api.EventDeleteTask{})
	defer cancel()

	orchestrator := NewGlobalOrchestrator(s)
	defer orchestrator.Stop()

	go func() {
		assert.NoError(t, orchestrator.Run(ctx))
	}()

	observedTask1 := testutils.WatchTaskUpdate(t, watch)
	assert.Equal(t, observedTask1.ID, "task1")
	assert.Equal(t, observedTask1.Status.State, api.TaskStateRunning)
	assert.Equal(t, observedTask1.DesiredState, api.TaskStateShutdown)

	var deadCnt, liveCnt int
	var err error
	s.View(func(readTx store.ReadTx) {
		var tasks []*api.Task
		tasks, err = store.FindTasks(readTx, store.ByServiceID("serviceid1"))
		for _, task := range tasks {
			if task.DesiredState == api.TaskStateShutdown {
				deadCnt++
			} else {
				liveCnt++
			}
		}
	})
	assert.NoError(t, err)
	assert.Equal(t, deadCnt, 1)
	assert.Equal(t, liveCnt, 0)
}

func TestInitializationRestartHistory(t *testing.T) {
	ctx := context.Background()
	s := store.NewMemoryStore(nil)
	assert.NotNil(t, s)
	defer s.Close()

	// create nodes, services and tasks in store directly
	addNode(t, s, node1)

	service := &api.Service{
		ID: "serviceid1",
		SpecVersion: &api.Version{
			Index: 2,
		},
		Spec: api.ServiceSpec{
			Annotations: api.Annotations{
				Name: "name1",
			},
			Task: api.TaskSpec{
				Runtime: &api.TaskSpec_Container{
					Container: &api.ContainerSpec{},
				},
				Restart: &api.RestartPolicy{
					Condition:   api.RestartOnAny,
					Delay:       gogotypes.DurationProto(restartDelay),
					MaxAttempts: 3,
					Window:      gogotypes.DurationProto(10 * time.Minute),
				},
			},
			Mode: &api.ServiceSpec_Global{
				Global: &api.GlobalService{},
			},
		},
	}
	addService(t, s, service)

	now := time.Now()

	tasks := []*api.Task{
		// old spec versions should be ignored for restart tracking
		{
			ID: "oldspec",
			Meta: api.Meta{
				CreatedAt: ptypes.MustTimestampProto(now.Add(-5 * time.Minute)),
			},
			DesiredState: api.TaskStateShutdown,
			SpecVersion: &api.Version{
				Index: 1,
			},
			Status: api.TaskStatus{
				State:     api.TaskStateShutdown,
				Timestamp: ptypes.MustTimestampProto(now.Add(-5 * time.Minute)),
			},
			Spec:      service.Spec.Task,
			ServiceID: "serviceid1",
			NodeID:    "nodeid1",
		},
		// this is the first task with the current spec version
		{
			ID: "firstcurrent",
			Meta: api.Meta{
				CreatedAt: ptypes.MustTimestampProto(now.Add(-12 * time.Minute)),
			},
			DesiredState: api.TaskStateShutdown,
			SpecVersion: &api.Version{
				Index: 2,
			},
			Status: api.TaskStatus{
				State:     api.TaskStateFailed,
				Timestamp: ptypes.MustTimestampProto(now.Add(-12 * time.Minute)),
			},
			Spec:      service.Spec.Task,
			ServiceID: "serviceid1",
			NodeID:    "nodeid1",
		},

		// this task falls outside the restart window
		{
			ID: "outsidewindow",
			Meta: api.Meta{
				CreatedAt: ptypes.MustTimestampProto(now.Add(-11 * time.Minute)),
			},
			DesiredState: api.TaskStateShutdown,
			SpecVersion: &api.Version{
				Index: 2,
			},
			Status: api.TaskStatus{
				State:     api.TaskStateFailed,
				Timestamp: ptypes.MustTimestampProto(now.Add(-11 * time.Minute)),
			},
			Spec:      service.Spec.Task,
			ServiceID: "serviceid1",
			NodeID:    "nodeid1",
		},
		// first task inside restart window
		{
			ID: "firstinside",
			Meta: api.Meta{
				CreatedAt: ptypes.MustTimestampProto(now.Add(-9 * time.Minute)),
			},
			DesiredState: api.TaskStateShutdown,
			SpecVersion: &api.Version{
				Index: 2,
			},
			Status: api.TaskStatus{
				State:     api.TaskStateFailed,
				Timestamp: ptypes.MustTimestampProto(now.Add(-9 * time.Minute)),
			},
			Spec:      service.Spec.Task,
			ServiceID: "serviceid1",
			NodeID:    "nodeid1",
		},
		// second task inside restart window, currently running
		{
			ID: "secondinside",
			Meta: api.Meta{
				CreatedAt: ptypes.MustTimestampProto(now.Add(-8 * time.Minute)),
			},
			DesiredState: api.TaskStateRunning,
			SpecVersion: &api.Version{
				Index: 2,
			},
			Status: api.TaskStatus{
				State:     api.TaskStateRunning,
				Timestamp: ptypes.MustTimestampProto(now.Add(-8 * time.Minute)),
			},
			Spec:      service.Spec.Task,
			ServiceID: "serviceid1",
			NodeID:    "nodeid1",
		},
	}
	for _, task := range tasks {
		addTask(t, s, task)
	}

	// watch orchestration events
	watch, cancel := state.Watch(s.WatchQueue(), api.EventCreateTask{}, api.EventUpdateTask{}, api.EventDeleteTask{})
	defer cancel()

	orchestrator := NewGlobalOrchestrator(s)
	defer orchestrator.Stop()

	go func() {
		assert.NoError(t, orchestrator.Run(ctx))
	}()

	// Fail the running task
	s.Update(func(tx store.Tx) error {
		task := store.GetTask(tx, "secondinside")
		require.NotNil(t, task)
		task.Status.State = api.TaskStateFailed
		task.Status.Timestamp = ptypes.MustTimestampProto(time.Now())
		assert.NoError(t, store.UpdateTask(tx, task))
		return nil
	})
	testutils.Expect(t, watch, api.EventUpdateTask{})

	// It should restart, because this will only be the third restart
	// attempt within the time window.
	observedTask1 := testutils.WatchTaskUpdate(t, watch)
	assert.Equal(t, "secondinside", observedTask1.ID)
	assert.Equal(t, api.TaskStateFailed, observedTask1.Status.State)

	observedTask2 := testutils.WatchTaskCreate(t, watch)
	assert.Equal(t, observedTask2.NodeID, "nodeid1")
	assert.Equal(t, api.TaskStateNew, observedTask2.Status.State)
	assert.Equal(t, api.TaskStateReady, observedTask2.DesiredState)

	observedTask3 := testutils.WatchTaskUpdate(t, watch)
	assert.Equal(t, observedTask2.ID, observedTask3.ID)
	assert.Equal(t, api.TaskStateRunning, observedTask3.DesiredState)

	// Reject the new task
	s.Update(func(tx store.Tx) error {
		task := store.GetTask(tx, observedTask2.ID)
		require.NotNil(t, task)
		task.Status.State = api.TaskStateRejected
		task.Status.Timestamp = ptypes.MustTimestampProto(time.Now())
		assert.NoError(t, store.UpdateTask(tx, task))
		return nil
	})
	testutils.Expect(t, watch, api.EventUpdateTask{}) // our update
	testutils.Expect(t, watch, api.EventUpdateTask{}) // orchestrator changes desired state

	// It shouldn't restart - that would exceed MaxAttempts
	select {
	case event := <-watch:
		t.Fatalf("got unexpected event %T: %+v", event, event)
	case <-time.After(100 * time.Millisecond):
	}
}
