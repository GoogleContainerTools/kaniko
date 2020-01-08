package replicated

import (
	"testing"
	"time"

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

func TestReplicatedOrchestrator(t *testing.T) {
	ctx := context.Background()
	s := store.NewMemoryStore(nil)
	assert.NotNil(t, s)
	defer s.Close()

	orchestrator := NewReplicatedOrchestrator(s)
	defer orchestrator.Stop()

	watch, cancel := state.Watch(s.WatchQueue() /*api.EventCreateTask{}, api.EventUpdateTask{}*/)
	defer cancel()

	// Create a service with two instances specified before the orchestrator is
	// started. This should result in two tasks when the orchestrator
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
						Container: &api.ContainerSpec{},
					},
				},
				Mode: &api.ServiceSpec_Replicated{
					Replicated: &api.ReplicatedService{
						Replicas: 2,
					},
				},
			},
		}
		assert.NoError(t, store.CreateService(tx, s1))
		return nil
	})
	assert.NoError(t, err)

	// Start the orchestrator.
	go func() {
		assert.NoError(t, orchestrator.Run(ctx))
	}()

	observedTask1 := testutils.WatchTaskCreate(t, watch)
	assert.Equal(t, observedTask1.Status.State, api.TaskStateNew)
	assert.Equal(t, observedTask1.ServiceAnnotations.Name, "name1")

	observedTask2 := testutils.WatchTaskCreate(t, watch)
	assert.Equal(t, observedTask2.Status.State, api.TaskStateNew)
	assert.Equal(t, observedTask2.ServiceAnnotations.Name, "name1")

	// Create a second service.
	err = s.Update(func(tx store.Tx) error {
		s2 := &api.Service{
			ID: "id2",
			Spec: api.ServiceSpec{
				Annotations: api.Annotations{
					Name: "name2",
				},
				Task: api.TaskSpec{
					Runtime: &api.TaskSpec_Container{
						Container: &api.ContainerSpec{},
					},
				},
				Mode: &api.ServiceSpec_Replicated{
					Replicated: &api.ReplicatedService{
						Replicas: 1,
					},
				},
			},
		}
		assert.NoError(t, store.CreateService(tx, s2))
		return nil
	})
	assert.NoError(t, err)

	observedTask3 := testutils.WatchTaskCreate(t, watch)
	assert.Equal(t, observedTask3.Status.State, api.TaskStateNew)
	assert.Equal(t, observedTask3.ServiceAnnotations.Name, "name2")

	// Update a service to scale it out to 3 instances
	err = s.Update(func(tx store.Tx) error {
		s2 := &api.Service{
			ID: "id2",
			Spec: api.ServiceSpec{
				Annotations: api.Annotations{
					Name: "name2",
				},
				Task: api.TaskSpec{
					Runtime: &api.TaskSpec_Container{
						Container: &api.ContainerSpec{},
					},
				},
				Mode: &api.ServiceSpec_Replicated{
					Replicated: &api.ReplicatedService{
						Replicas: 3,
					},
				},
			},
		}
		assert.NoError(t, store.UpdateService(tx, s2))
		return nil
	})
	assert.NoError(t, err)

	observedTask4 := testutils.WatchTaskCreate(t, watch)
	assert.Equal(t, observedTask4.Status.State, api.TaskStateNew)
	assert.Equal(t, observedTask4.ServiceAnnotations.Name, "name2")

	observedTask5 := testutils.WatchTaskCreate(t, watch)
	assert.Equal(t, observedTask5.Status.State, api.TaskStateNew)
	assert.Equal(t, observedTask5.ServiceAnnotations.Name, "name2")

	// Now scale it back down to 1 instance
	err = s.Update(func(tx store.Tx) error {
		s2 := &api.Service{
			ID: "id2",
			Spec: api.ServiceSpec{
				Annotations: api.Annotations{
					Name: "name2",
				},
				Task: api.TaskSpec{
					Runtime: &api.TaskSpec_Container{
						Container: &api.ContainerSpec{},
					},
				},
				Mode: &api.ServiceSpec_Replicated{
					Replicated: &api.ReplicatedService{
						Replicas: 1,
					},
				},
			},
		}
		assert.NoError(t, store.UpdateService(tx, s2))
		return nil
	})
	assert.NoError(t, err)

	observedUpdateRemove1 := testutils.WatchTaskUpdate(t, watch)
	assert.Equal(t, observedUpdateRemove1.DesiredState, api.TaskStateRemove)
	assert.Equal(t, observedUpdateRemove1.ServiceAnnotations.Name, "name2")

	observedUpdateRemove2 := testutils.WatchTaskUpdate(t, watch)
	assert.Equal(t, observedUpdateRemove2.DesiredState, api.TaskStateRemove)
	assert.Equal(t, observedUpdateRemove2.ServiceAnnotations.Name, "name2")

	// There should be one remaining task attached to service id2/name2.
	var liveTasks []*api.Task
	s.View(func(readTx store.ReadTx) {
		var tasks []*api.Task
		tasks, err = store.FindTasks(readTx, store.ByServiceID("id2"))
		for _, t := range tasks {
			if t.DesiredState == api.TaskStateRunning {
				liveTasks = append(liveTasks, t)
			}
		}
	})
	assert.NoError(t, err)
	assert.Len(t, liveTasks, 1)

	// Delete the remaining task directly. It should be recreated by the
	// orchestrator.
	err = s.Update(func(tx store.Tx) error {
		assert.NoError(t, store.DeleteTask(tx, liveTasks[0].ID))
		return nil
	})
	assert.NoError(t, err)

	observedTask6 := testutils.WatchTaskCreate(t, watch)
	assert.Equal(t, observedTask6.Status.State, api.TaskStateNew)
	assert.Equal(t, observedTask6.ServiceAnnotations.Name, "name2")

	// Delete the service. Its remaining task should go away.
	err = s.Update(func(tx store.Tx) error {
		assert.NoError(t, store.DeleteService(tx, "id2"))
		return nil
	})
	assert.NoError(t, err)

	deletedTask := testutils.WatchTaskDelete(t, watch)
	assert.Equal(t, deletedTask.Status.State, api.TaskStateNew)
	assert.Equal(t, deletedTask.ServiceAnnotations.Name, "name2")
}

func TestReplicatedScaleDown(t *testing.T) {
	ctx := context.Background()
	s := store.NewMemoryStore(nil)
	assert.NotNil(t, s)
	defer s.Close()

	orchestrator := NewReplicatedOrchestrator(s)
	defer orchestrator.Stop()

	watch, cancel := state.Watch(s.WatchQueue(), api.EventUpdateTask{}, api.EventDeleteTask{})
	defer cancel()

	s1 := &api.Service{
		ID: "id1",
		Spec: api.ServiceSpec{
			Annotations: api.Annotations{
				Name: "name1",
			},
			Mode: &api.ServiceSpec_Replicated{
				Replicated: &api.ReplicatedService{
					Replicas: 6,
				},
			},
		},
	}

	err := s.Update(func(tx store.Tx) error {
		assert.NoError(t, store.CreateService(tx, s1))

		nodes := []*api.Node{
			{
				ID: "node1",
				Spec: api.NodeSpec{
					Annotations: api.Annotations{
						Name: "name1",
					},
					Availability: api.NodeAvailabilityActive,
				},
				Status: api.NodeStatus{
					State: api.NodeStatus_READY,
				},
			},
			{
				ID: "node2",
				Spec: api.NodeSpec{
					Annotations: api.Annotations{
						Name: "name2",
					},
					Availability: api.NodeAvailabilityActive,
				},
				Status: api.NodeStatus{
					State: api.NodeStatus_READY,
				},
			},
			{
				ID: "node3",
				Spec: api.NodeSpec{
					Annotations: api.Annotations{
						Name: "name3",
					},
					Availability: api.NodeAvailabilityActive,
				},
				Status: api.NodeStatus{
					State: api.NodeStatus_READY,
				},
			},
		}
		for _, node := range nodes {
			assert.NoError(t, store.CreateNode(tx, node))
		}

		// task1 is assigned to node1
		// task2 - task3 are assigned to node2
		// task4 - task6 are assigned to node3
		// task7 is unassigned

		tasks := []*api.Task{
			{
				ID:           "task1",
				Slot:         1,
				DesiredState: api.TaskStateRunning,
				Status: api.TaskStatus{
					State: api.TaskStateStarting,
				},
				ServiceAnnotations: api.Annotations{
					Name: "task1",
				},
				ServiceID: "id1",
				NodeID:    "node1",
			},
			{
				ID:           "task2",
				Slot:         2,
				DesiredState: api.TaskStateRunning,
				Status: api.TaskStatus{
					State: api.TaskStateRunning,
				},
				ServiceAnnotations: api.Annotations{
					Name: "task2",
				},
				ServiceID: "id1",
				NodeID:    "node2",
			},
			{
				ID:           "task3",
				Slot:         3,
				DesiredState: api.TaskStateRunning,
				Status: api.TaskStatus{
					State: api.TaskStateRunning,
				},
				ServiceAnnotations: api.Annotations{
					Name: "task3",
				},
				ServiceID: "id1",
				NodeID:    "node2",
			},
			{
				ID:           "task4",
				Slot:         4,
				DesiredState: api.TaskStateRunning,
				Status: api.TaskStatus{
					State: api.TaskStateRunning,
				},
				ServiceAnnotations: api.Annotations{
					Name: "task4",
				},
				ServiceID: "id1",
				NodeID:    "node3",
			},
			{
				ID:           "task5",
				Slot:         5,
				DesiredState: api.TaskStateRunning,
				Status: api.TaskStatus{
					State: api.TaskStateRunning,
				},
				ServiceAnnotations: api.Annotations{
					Name: "task5",
				},
				ServiceID: "id1",
				NodeID:    "node3",
			},
			{
				ID:           "task6",
				Slot:         6,
				DesiredState: api.TaskStateRunning,
				Status: api.TaskStatus{
					State: api.TaskStateRunning,
				},
				ServiceAnnotations: api.Annotations{
					Name: "task6",
				},
				ServiceID: "id1",
				NodeID:    "node3",
			},
			{
				ID:           "task7",
				Slot:         7,
				DesiredState: api.TaskStateRunning,
				Status: api.TaskStatus{
					State: api.TaskStateNew,
				},
				ServiceAnnotations: api.Annotations{
					Name: "task7",
				},
				ServiceID: "id1",
			},
		}
		for _, task := range tasks {
			assert.NoError(t, store.CreateTask(tx, task))
		}

		return nil
	})
	assert.NoError(t, err)

	// Start the orchestrator.
	go func() {
		assert.NoError(t, orchestrator.Run(ctx))
	}()

	// Replicas was set to 6, but we started with 7 tasks. task7 should
	// be the one the orchestrator chose to shut down because it was not
	// assigned yet. The desired state of task7 will be set to "REMOVE"

	observedUpdateRemove := testutils.WatchTaskUpdate(t, watch)
	assert.Equal(t, api.TaskStateRemove, observedUpdateRemove.DesiredState)
	assert.Equal(t, "task7", observedUpdateRemove.ID)

	// Now scale down to 4 instances.
	err = s.Update(func(tx store.Tx) error {
		s1.Spec.Mode = &api.ServiceSpec_Replicated{
			Replicated: &api.ReplicatedService{
				Replicas: 4,
			},
		}
		assert.NoError(t, store.UpdateService(tx, s1))
		return nil
	})
	assert.NoError(t, err)

	// Tasks should be shut down in a way that balances the remaining tasks.
	// node2 should be preferred over node3 because node2's tasks have
	// lower Slot numbers than node3's tasks.

	shutdowns := make(map[string]int)
	for i := 0; i != 2; i++ {
		observedUpdateDesiredRemove := testutils.WatchTaskUpdate(t, watch)
		assert.Equal(t, api.TaskStateRemove, observedUpdateDesiredRemove.DesiredState)
		shutdowns[observedUpdateDesiredRemove.NodeID]++
	}

	assert.Equal(t, 0, shutdowns["node1"])
	assert.Equal(t, 0, shutdowns["node2"])
	assert.Equal(t, 2, shutdowns["node3"])

	// task4 should be preferred over task5 and task6.
	s.View(func(readTx store.ReadTx) {
		tasks, err := store.FindTasks(readTx, store.ByNodeID("node3"))
		require.NoError(t, err)
		for _, task := range tasks {
			if task.DesiredState == api.TaskStateRunning {
				assert.Equal(t, "task4", task.ID)
			}
		}
	})

	// Now scale down to 2 instances.
	err = s.Update(func(tx store.Tx) error {
		s1.Spec.Mode = &api.ServiceSpec_Replicated{
			Replicated: &api.ReplicatedService{
				Replicas: 2,
			},
		}
		assert.NoError(t, store.UpdateService(tx, s1))
		return nil
	})
	assert.NoError(t, err)

	// Tasks should be shut down in a way that balances the remaining tasks.
	// node2 and node3 should be preferred over node1 because node1's task
	// is not running yet.

	shutdowns = make(map[string]int)
	for i := 0; i != 2; i++ {
		observedUpdateDesiredRemove := testutils.WatchTaskUpdate(t, watch)
		assert.Equal(t, api.TaskStateRemove, observedUpdateDesiredRemove.DesiredState)
		shutdowns[observedUpdateDesiredRemove.NodeID]++
	}

	assert.Equal(t, 1, shutdowns["node1"])
	assert.Equal(t, 1, shutdowns["node2"])
	assert.Equal(t, 0, shutdowns["node3"])

	// There should be remaining tasks on node2 and node3. task2 should be
	// preferred over task3 on node2.
	s.View(func(readTx store.ReadTx) {
		tasks, err := store.FindTasks(readTx, store.ByDesiredState(api.TaskStateRunning))
		require.NoError(t, err)
		require.Len(t, tasks, 2)
		if tasks[0].NodeID == "node2" {
			assert.Equal(t, "task2", tasks[0].ID)
			assert.Equal(t, "node3", tasks[1].NodeID)
		} else {
			assert.Equal(t, "node3", tasks[0].NodeID)
			assert.Equal(t, "node2", tasks[1].NodeID)
			assert.Equal(t, "task2", tasks[1].ID)
		}
	})
}

func TestInitializationRejectedTasks(t *testing.T) {
	ctx := context.Background()
	s := store.NewMemoryStore(nil)
	assert.NotNil(t, s)
	defer s.Close()

	service1 := &api.Service{
		ID: "serviceid1",
		Spec: api.ServiceSpec{
			Annotations: api.Annotations{
				Name: "name1",
			},
			Task: api.TaskSpec{
				Runtime: &api.TaskSpec_Container{
					Container: &api.ContainerSpec{},
				},
			},
			Mode: &api.ServiceSpec_Replicated{
				Replicated: &api.ReplicatedService{
					Replicas: 1,
				},
			},
		},
	}

	err := s.Update(func(tx store.Tx) error {
		assert.NoError(t, store.CreateService(tx, service1))

		nodes := []*api.Node{
			{
				ID: "node1",
				Spec: api.NodeSpec{
					Annotations: api.Annotations{
						Name: "name1",
					},
					Availability: api.NodeAvailabilityActive,
				},
				Status: api.NodeStatus{
					State: api.NodeStatus_READY,
				},
			},
		}
		for _, node := range nodes {
			assert.NoError(t, store.CreateNode(tx, node))
		}

		// 1 rejected task is in store before orchestrator starts
		tasks := []*api.Task{
			{
				ID:           "task1",
				Slot:         1,
				DesiredState: api.TaskStateReady,
				Status: api.TaskStatus{
					State: api.TaskStateRejected,
				},
				Spec: api.TaskSpec{
					Runtime: &api.TaskSpec_Container{
						Container: &api.ContainerSpec{},
					},
				},
				ServiceAnnotations: api.Annotations{
					Name: "task1",
				},
				ServiceID: "serviceid1",
				NodeID:    "node1",
			},
		}
		for _, task := range tasks {
			assert.NoError(t, store.CreateTask(tx, task))
		}

		return nil
	})
	assert.NoError(t, err)

	// watch orchestration events
	watch, cancel := state.Watch(s.WatchQueue(), api.EventCreateTask{}, api.EventUpdateTask{}, api.EventDeleteTask{})
	defer cancel()

	orchestrator := NewReplicatedOrchestrator(s)
	defer orchestrator.Stop()

	go func() {
		assert.NoError(t, orchestrator.Run(ctx))
	}()

	// initTask triggers an update event
	observedTask1 := testutils.WatchTaskUpdate(t, watch)
	assert.Equal(t, observedTask1.ID, "task1")
	assert.Equal(t, observedTask1.Status.State, api.TaskStateRejected)
	assert.Equal(t, observedTask1.DesiredState, api.TaskStateShutdown)

	// a new task is created
	observedTask2 := testutils.WatchTaskCreate(t, watch)
	assert.Equal(t, observedTask2.ServiceID, "serviceid1")
	// it has not been scheduled
	assert.Equal(t, observedTask2.NodeID, "")
	assert.Equal(t, observedTask2.Status.State, api.TaskStateNew)
	assert.Equal(t, observedTask2.DesiredState, api.TaskStateReady)

	var deadCnt, liveCnt int
	s.View(func(readTx store.ReadTx) {
		var tasks []*api.Task
		tasks, err = store.FindTasks(readTx, store.ByServiceID("serviceid1"))
		for _, task := range tasks {
			if task.DesiredState == api.TaskStateShutdown {
				assert.Equal(t, task.ID, "task1")
				deadCnt++
			} else {
				liveCnt++
			}
		}
	})
	assert.NoError(t, err)
	assert.Equal(t, deadCnt, 1)
	assert.Equal(t, liveCnt, 1)
}

func TestInitializationFailedTasks(t *testing.T) {
	ctx := context.Background()
	s := store.NewMemoryStore(nil)
	assert.NotNil(t, s)
	defer s.Close()

	service1 := &api.Service{
		ID: "serviceid1",
		Spec: api.ServiceSpec{
			Annotations: api.Annotations{
				Name: "name1",
			},
			Task: api.TaskSpec{
				Runtime: &api.TaskSpec_Container{
					Container: &api.ContainerSpec{},
				},
			},
			Mode: &api.ServiceSpec_Replicated{
				Replicated: &api.ReplicatedService{
					Replicas: 2,
				},
			},
		},
	}

	err := s.Update(func(tx store.Tx) error {
		assert.NoError(t, store.CreateService(tx, service1))

		nodes := []*api.Node{
			{
				ID: "node1",
				Spec: api.NodeSpec{
					Annotations: api.Annotations{
						Name: "name1",
					},
					Availability: api.NodeAvailabilityActive,
				},
				Status: api.NodeStatus{
					State: api.NodeStatus_READY,
				},
			},
		}
		for _, node := range nodes {
			assert.NoError(t, store.CreateNode(tx, node))
		}

		// 1 failed task is in store before orchestrator starts
		tasks := []*api.Task{
			{
				ID:           "task1",
				Slot:         1,
				DesiredState: api.TaskStateRunning,
				Status: api.TaskStatus{
					State: api.TaskStateFailed,
				},
				Spec: api.TaskSpec{
					Runtime: &api.TaskSpec_Container{
						Container: &api.ContainerSpec{},
					},
				},
				ServiceAnnotations: api.Annotations{
					Name: "task1",
				},
				ServiceID: "serviceid1",
				NodeID:    "node1",
			},
			{
				ID:           "task2",
				Slot:         2,
				DesiredState: api.TaskStateRunning,
				Status: api.TaskStatus{
					State: api.TaskStateStarting,
				},
				Spec: api.TaskSpec{
					Runtime: &api.TaskSpec_Container{
						Container: &api.ContainerSpec{},
					},
				},
				ServiceAnnotations: api.Annotations{
					Name: "task2",
				},
				ServiceID: "serviceid1",
				NodeID:    "node1",
			},
		}
		for _, task := range tasks {
			assert.NoError(t, store.CreateTask(tx, task))
		}

		return nil
	})
	assert.NoError(t, err)

	// watch orchestration events
	watch, cancel := state.Watch(s.WatchQueue(), api.EventCreateTask{}, api.EventUpdateTask{}, api.EventDeleteTask{})
	defer cancel()

	orchestrator := NewReplicatedOrchestrator(s)
	defer orchestrator.Stop()

	go func() {
		assert.NoError(t, orchestrator.Run(ctx))
	}()

	// initTask triggers an update
	observedTask1 := testutils.WatchTaskUpdate(t, watch)
	assert.Equal(t, observedTask1.ID, "task1")
	assert.Equal(t, observedTask1.Status.State, api.TaskStateFailed)
	assert.Equal(t, observedTask1.DesiredState, api.TaskStateShutdown)

	// a new task is created
	observedTask2 := testutils.WatchTaskCreate(t, watch)
	assert.Equal(t, observedTask2.ServiceID, "serviceid1")
	assert.Equal(t, observedTask2.Status.State, api.TaskStateNew)
	assert.Equal(t, observedTask2.DesiredState, api.TaskStateReady)

	var deadCnt, liveCnt int
	s.View(func(readTx store.ReadTx) {
		var tasks []*api.Task
		tasks, err = store.FindTasks(readTx, store.ByServiceID("serviceid1"))
		for _, task := range tasks {
			if task.DesiredState == api.TaskStateShutdown {
				assert.Equal(t, task.ID, "task1")
				deadCnt++
			} else {
				liveCnt++
			}
		}
	})
	assert.NoError(t, err)
	assert.Equal(t, deadCnt, 1)
	assert.Equal(t, liveCnt, 2)
}

func TestInitializationNodeDown(t *testing.T) {
	ctx := context.Background()
	s := store.NewMemoryStore(nil)
	assert.NotNil(t, s)
	defer s.Close()

	service1 := &api.Service{
		ID: "serviceid1",
		Spec: api.ServiceSpec{
			Annotations: api.Annotations{
				Name: "name1",
			},
			Task: api.TaskSpec{
				Runtime: &api.TaskSpec_Container{
					Container: &api.ContainerSpec{},
				},
			},
			Mode: &api.ServiceSpec_Replicated{
				Replicated: &api.ReplicatedService{
					Replicas: 1,
				},
			},
		},
	}

	err := s.Update(func(tx store.Tx) error {
		assert.NoError(t, store.CreateService(tx, service1))

		nodes := []*api.Node{
			{
				ID: "node1",
				Spec: api.NodeSpec{
					Annotations: api.Annotations{
						Name: "name1",
					},
					Availability: api.NodeAvailabilityActive,
				},
				Status: api.NodeStatus{
					State: api.NodeStatus_DOWN,
				},
			},
		}
		for _, node := range nodes {
			assert.NoError(t, store.CreateNode(tx, node))
		}

		// 1 failed task is in store before orchestrator starts
		tasks := []*api.Task{
			{
				ID:           "task1",
				Slot:         1,
				DesiredState: api.TaskStateRunning,
				Status: api.TaskStatus{
					State: api.TaskStateRunning,
				},
				Spec: api.TaskSpec{
					Runtime: &api.TaskSpec_Container{
						Container: &api.ContainerSpec{},
					},
				},
				ServiceAnnotations: api.Annotations{
					Name: "task1",
				},
				ServiceID: "serviceid1",
				NodeID:    "node1",
			},
		}
		for _, task := range tasks {
			assert.NoError(t, store.CreateTask(tx, task))
		}

		return nil
	})
	assert.NoError(t, err)

	// watch orchestration events
	watch, cancel := state.Watch(s.WatchQueue(), api.EventCreateTask{}, api.EventUpdateTask{}, api.EventDeleteTask{})
	defer cancel()

	orchestrator := NewReplicatedOrchestrator(s)
	defer orchestrator.Stop()

	go func() {
		assert.NoError(t, orchestrator.Run(ctx))
	}()

	// initTask triggers an update
	observedTask1 := testutils.WatchTaskUpdate(t, watch)
	assert.Equal(t, observedTask1.ID, "task1")
	assert.Equal(t, observedTask1.Status.State, api.TaskStateRunning)
	assert.Equal(t, observedTask1.DesiredState, api.TaskStateShutdown)

	// a new task is created
	observedTask2 := testutils.WatchTaskCreate(t, watch)
	assert.Equal(t, observedTask2.ServiceID, "serviceid1")
	assert.Equal(t, observedTask2.Status.State, api.TaskStateNew)
	assert.Equal(t, observedTask2.DesiredState, api.TaskStateReady)
}

func TestInitializationDelayStart(t *testing.T) {
	ctx := context.Background()
	s := store.NewMemoryStore(nil)
	assert.NotNil(t, s)
	defer s.Close()

	service1 := &api.Service{
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
					Delay:     gogotypes.DurationProto(100 * time.Millisecond),
				},
			},
			Mode: &api.ServiceSpec_Replicated{
				Replicated: &api.ReplicatedService{
					Replicas: 1,
				},
			},
		},
	}

	before := time.Now()
	err := s.Update(func(tx store.Tx) error {
		assert.NoError(t, store.CreateService(tx, service1))

		nodes := []*api.Node{
			{
				ID: "node1",
				Spec: api.NodeSpec{
					Annotations: api.Annotations{
						Name: "name1",
					},
					Availability: api.NodeAvailabilityActive,
				},
				Status: api.NodeStatus{
					State: api.NodeStatus_READY,
				},
			},
		}
		for _, node := range nodes {
			assert.NoError(t, store.CreateNode(tx, node))
		}

		// 1 failed task is in store before orchestrator starts
		tasks := []*api.Task{
			{
				ID:           "task1",
				Slot:         1,
				DesiredState: api.TaskStateReady,
				Status: api.TaskStatus{
					State:     api.TaskStateReady,
					Timestamp: ptypes.MustTimestampProto(before),
				},
				Spec: api.TaskSpec{
					Runtime: &api.TaskSpec_Container{
						Container: &api.ContainerSpec{},
					},
					Restart: &api.RestartPolicy{
						Condition: api.RestartOnAny,
						Delay:     gogotypes.DurationProto(100 * time.Millisecond),
					},
				},
				ServiceAnnotations: api.Annotations{
					Name: "task1",
				},
				ServiceID: "serviceid1",
				NodeID:    "node1",
			},
		}
		for _, task := range tasks {
			assert.NoError(t, store.CreateTask(tx, task))
		}

		return nil
	})
	assert.NoError(t, err)

	// watch orchestration events
	watch, cancel := state.Watch(s.WatchQueue(), api.EventCreateTask{}, api.EventUpdateTask{}, api.EventDeleteTask{})
	defer cancel()

	orchestrator := NewReplicatedOrchestrator(s)
	defer orchestrator.Stop()

	go func() {
		assert.NoError(t, orchestrator.Run(ctx))
	}()

	// initTask triggers an update
	observedTask1 := testutils.WatchTaskUpdate(t, watch)
	after := time.Now()
	assert.Equal(t, observedTask1.ID, "task1")
	assert.Equal(t, observedTask1.Status.State, api.TaskStateReady)
	assert.Equal(t, observedTask1.DesiredState, api.TaskStateRunning)

	// At least 100 ms should have elapsed
	if after.Sub(before) < 100*time.Millisecond {
		t.Fatalf("restart delay should have elapsed. Got: %v", after.Sub(before))
	}
}
