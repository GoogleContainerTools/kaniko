package scheduler

import (
	"fmt"
	"runtime"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/docker/go-events"
	"github.com/docker/swarmkit/api"
	"github.com/docker/swarmkit/api/genericresource"
	"github.com/docker/swarmkit/identity"
	"github.com/docker/swarmkit/manager/state"
	"github.com/docker/swarmkit/manager/state/store"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"golang.org/x/net/context"
)

func TestScheduler(t *testing.T) {
	ctx := context.Background()
	initialNodeSet := []*api.Node{
		{
			ID: "id1",
			Spec: api.NodeSpec{
				Annotations: api.Annotations{
					Name: "name1",
				},
			},
			Status: api.NodeStatus{
				State: api.NodeStatus_READY,
			},
		},
		{
			ID: "id2",
			Spec: api.NodeSpec{
				Annotations: api.Annotations{
					Name: "name2",
				},
			},
			Status: api.NodeStatus{
				State: api.NodeStatus_READY,
			},
		},
		{
			ID: "id3",
			Spec: api.NodeSpec{
				Annotations: api.Annotations{
					Name: "name2",
				},
			},
			Status: api.NodeStatus{
				State: api.NodeStatus_READY,
			},
		},
	}

	initialTaskSet := []*api.Task{
		{
			ID:           "id1",
			DesiredState: api.TaskStateRunning,
			ServiceAnnotations: api.Annotations{
				Name: "name1",
			},

			Status: api.TaskStatus{
				State: api.TaskStateAssigned,
			},
			NodeID: initialNodeSet[0].ID,
		},
		{
			ID:           "id2",
			DesiredState: api.TaskStateRunning,
			ServiceAnnotations: api.Annotations{
				Name: "name2",
			},
			Status: api.TaskStatus{
				State: api.TaskStatePending,
			},
		},
		{
			ID:           "id3",
			DesiredState: api.TaskStateRunning,
			ServiceAnnotations: api.Annotations{
				Name: "name2",
			},
			Status: api.TaskStatus{
				State: api.TaskStatePending,
			},
		},
	}

	s := store.NewMemoryStore(nil)
	assert.NotNil(t, s)
	defer s.Close()

	err := s.Update(func(tx store.Tx) error {
		// Prepopulate nodes
		for _, n := range initialNodeSet {
			assert.NoError(t, store.CreateNode(tx, n))
		}

		// Prepopulate tasks
		for _, task := range initialTaskSet {
			assert.NoError(t, store.CreateTask(tx, task))
		}
		return nil
	})
	assert.NoError(t, err)

	scheduler := New(s)

	watch, cancel := state.Watch(s.WatchQueue(), api.EventUpdateTask{})
	defer cancel()

	go func() {
		assert.NoError(t, scheduler.Run(ctx))
	}()
	defer scheduler.Stop()

	assignment1 := watchAssignment(t, watch)
	// must assign to id2 or id3 since id1 already has a task
	assert.Regexp(t, assignment1.NodeID, "(id2|id3)")

	assignment2 := watchAssignment(t, watch)
	// must assign to id2 or id3 since id1 already has a task
	if assignment1.NodeID == "id2" {
		assert.Equal(t, "id3", assignment2.NodeID)
	} else {
		assert.Equal(t, "id2", assignment2.NodeID)
	}

	err = s.Update(func(tx store.Tx) error {
		// Update each node to make sure this doesn't mess up the
		// scheduler's state.
		for _, n := range initialNodeSet {
			assert.NoError(t, store.UpdateNode(tx, n))
		}
		return nil
	})
	assert.NoError(t, err)

	err = s.Update(func(tx store.Tx) error {
		// Delete the task associated with node 1 so it's now the most lightly
		// loaded node.
		assert.NoError(t, store.DeleteTask(tx, "id1"))

		// Create a new task. It should get assigned to id1.
		t4 := &api.Task{
			ID:           "id4",
			DesiredState: api.TaskStateRunning,
			ServiceAnnotations: api.Annotations{
				Name: "name4",
			},
			Status: api.TaskStatus{
				State: api.TaskStatePending,
			},
		}
		assert.NoError(t, store.CreateTask(tx, t4))
		return nil
	})
	assert.NoError(t, err)

	assignment3 := watchAssignment(t, watch)
	assert.Equal(t, "id1", assignment3.NodeID)

	// Update a task to make it unassigned. It should get assigned by the
	// scheduler.
	err = s.Update(func(tx store.Tx) error {
		// Remove assignment from task id4. It should get assigned
		// to node id1.
		t4 := &api.Task{
			ID:           "id4",
			DesiredState: api.TaskStateRunning,
			ServiceAnnotations: api.Annotations{
				Name: "name4",
			},
			Status: api.TaskStatus{
				State: api.TaskStatePending,
			},
		}
		assert.NoError(t, store.UpdateTask(tx, t4))
		return nil
	})
	assert.NoError(t, err)

	assignment4 := watchAssignment(t, watch)
	assert.Equal(t, "id1", assignment4.NodeID)

	err = s.Update(func(tx store.Tx) error {
		// Create a ready node, then remove it. No tasks should ever
		// be assigned to it.
		node := &api.Node{
			ID: "removednode",
			Spec: api.NodeSpec{
				Annotations: api.Annotations{
					Name: "removednode",
				},
			},
			Status: api.NodeStatus{
				State: api.NodeStatus_DOWN,
			},
		}
		assert.NoError(t, store.CreateNode(tx, node))
		assert.NoError(t, store.DeleteNode(tx, node.ID))

		// Create an unassigned task.
		task := &api.Task{
			ID:           "removednode",
			DesiredState: api.TaskStateRunning,
			ServiceAnnotations: api.Annotations{
				Name: "removednode",
			},
			Status: api.TaskStatus{
				State: api.TaskStatePending,
			},
		}
		assert.NoError(t, store.CreateTask(tx, task))
		return nil
	})
	assert.NoError(t, err)

	assignmentRemovedNode := watchAssignment(t, watch)
	assert.NotEqual(t, "removednode", assignmentRemovedNode.NodeID)

	err = s.Update(func(tx store.Tx) error {
		// Create a ready node. It should be used for the next
		// assignment.
		n4 := &api.Node{
			ID: "id4",
			Spec: api.NodeSpec{
				Annotations: api.Annotations{
					Name: "name4",
				},
			},
			Status: api.NodeStatus{
				State: api.NodeStatus_READY,
			},
		}
		assert.NoError(t, store.CreateNode(tx, n4))

		// Create an unassigned task.
		t5 := &api.Task{
			ID:           "id5",
			DesiredState: api.TaskStateRunning,
			ServiceAnnotations: api.Annotations{
				Name: "name5",
			},
			Status: api.TaskStatus{
				State: api.TaskStatePending,
			},
		}
		assert.NoError(t, store.CreateTask(tx, t5))
		return nil
	})
	assert.NoError(t, err)

	assignment5 := watchAssignment(t, watch)
	assert.Equal(t, "id4", assignment5.NodeID)

	err = s.Update(func(tx store.Tx) error {
		// Create a non-ready node. It should NOT be used for the next
		// assignment.
		n5 := &api.Node{
			ID: "id5",
			Spec: api.NodeSpec{
				Annotations: api.Annotations{
					Name: "name5",
				},
			},
			Status: api.NodeStatus{
				State: api.NodeStatus_DOWN,
			},
		}
		assert.NoError(t, store.CreateNode(tx, n5))

		// Create an unassigned task.
		t6 := &api.Task{
			ID:           "id6",
			DesiredState: api.TaskStateRunning,
			ServiceAnnotations: api.Annotations{
				Name: "name6",
			},
			Status: api.TaskStatus{
				State: api.TaskStatePending,
			},
		}
		assert.NoError(t, store.CreateTask(tx, t6))
		return nil
	})
	assert.NoError(t, err)

	assignment6 := watchAssignment(t, watch)
	assert.NotEqual(t, "id5", assignment6.NodeID)

	err = s.Update(func(tx store.Tx) error {
		// Update node id5 to put it in the READY state.
		n5 := &api.Node{
			ID: "id5",
			Spec: api.NodeSpec{
				Annotations: api.Annotations{
					Name: "name5",
				},
			},
			Status: api.NodeStatus{
				State: api.NodeStatus_READY,
			},
		}
		assert.NoError(t, store.UpdateNode(tx, n5))

		// Create an unassigned task. Should be assigned to the
		// now-ready node.
		t7 := &api.Task{
			ID:           "id7",
			DesiredState: api.TaskStateRunning,
			ServiceAnnotations: api.Annotations{
				Name: "name7",
			},
			Status: api.TaskStatus{
				State: api.TaskStatePending,
			},
		}
		assert.NoError(t, store.CreateTask(tx, t7))
		return nil
	})
	assert.NoError(t, err)

	assignment7 := watchAssignment(t, watch)
	assert.Equal(t, "id5", assignment7.NodeID)

	err = s.Update(func(tx store.Tx) error {
		// Create a ready node, then immediately take it down. The next
		// unassigned task should NOT be assigned to it.
		n6 := &api.Node{
			ID: "id6",
			Spec: api.NodeSpec{
				Annotations: api.Annotations{
					Name: "name6",
				},
			},
			Status: api.NodeStatus{
				State: api.NodeStatus_READY,
			},
		}
		assert.NoError(t, store.CreateNode(tx, n6))
		n6.Status.State = api.NodeStatus_DOWN
		assert.NoError(t, store.UpdateNode(tx, n6))

		// Create an unassigned task.
		t8 := &api.Task{
			ID:           "id8",
			DesiredState: api.TaskStateRunning,
			ServiceAnnotations: api.Annotations{
				Name: "name8",
			},
			Status: api.TaskStatus{
				State: api.TaskStatePending,
			},
		}
		assert.NoError(t, store.CreateTask(tx, t8))
		return nil
	})
	assert.NoError(t, err)

	assignment8 := watchAssignment(t, watch)
	assert.NotEqual(t, "id6", assignment8.NodeID)
}

func testHA(t *testing.T, useSpecVersion bool) {
	ctx := context.Background()
	initialNodeSet := []*api.Node{
		{
			ID: "id1",
			Status: api.NodeStatus{
				State: api.NodeStatus_READY,
			},
		},
		{
			ID: "id2",
			Status: api.NodeStatus{
				State: api.NodeStatus_READY,
			},
		},
		{
			ID: "id3",
			Status: api.NodeStatus{
				State: api.NodeStatus_READY,
			},
		},
		{
			ID: "id4",
			Status: api.NodeStatus{
				State: api.NodeStatus_READY,
			},
		},
		{
			ID: "id5",
			Status: api.NodeStatus{
				State: api.NodeStatus_READY,
			},
		},
	}

	taskTemplate1 := &api.Task{
		DesiredState: api.TaskStateRunning,
		ServiceID:    "service1",
		Spec: api.TaskSpec{
			Runtime: &api.TaskSpec_Container{
				Container: &api.ContainerSpec{
					Image: "v:1",
				},
			},
		},
		Status: api.TaskStatus{
			State: api.TaskStatePending,
		},
	}

	taskTemplate2 := &api.Task{
		DesiredState: api.TaskStateRunning,
		ServiceID:    "service2",
		Spec: api.TaskSpec{
			Runtime: &api.TaskSpec_Container{
				Container: &api.ContainerSpec{
					Image: "v:2",
				},
			},
		},
		Status: api.TaskStatus{
			State: api.TaskStatePending,
		},
	}

	if useSpecVersion {
		taskTemplate1.SpecVersion = &api.Version{Index: 1}
		taskTemplate2.SpecVersion = &api.Version{Index: 1}
	}

	s := store.NewMemoryStore(nil)
	assert.NotNil(t, s)
	defer s.Close()

	t1Instances := 18

	err := s.Update(func(tx store.Tx) error {
		// Prepopulate nodes
		for _, n := range initialNodeSet {
			assert.NoError(t, store.CreateNode(tx, n))
		}

		// Prepopulate tasks from template 1
		for i := 0; i != t1Instances; i++ {
			taskTemplate1.ID = fmt.Sprintf("t1id%d", i)
			assert.NoError(t, store.CreateTask(tx, taskTemplate1))
		}
		return nil
	})
	assert.NoError(t, err)

	scheduler := New(s)

	watch, cancel := state.Watch(s.WatchQueue(), api.EventUpdateTask{})
	defer cancel()

	go func() {
		assert.NoError(t, scheduler.Run(ctx))
	}()
	defer scheduler.Stop()

	t1Assignments := make(map[string]int)
	for i := 0; i != t1Instances; i++ {
		assignment := watchAssignment(t, watch)
		if !strings.HasPrefix(assignment.ID, "t1") {
			t.Fatal("got assignment for different kind of task")
		}
		t1Assignments[assignment.NodeID]++
	}

	assert.Len(t, t1Assignments, 5)

	nodesWith3T1Tasks := 0
	nodesWith4T1Tasks := 0
	for nodeID, taskCount := range t1Assignments {
		if taskCount == 3 {
			nodesWith3T1Tasks++
		} else if taskCount == 4 {
			nodesWith4T1Tasks++
		} else {
			t.Fatalf("unexpected number of tasks %d on node %s", taskCount, nodeID)
		}
	}

	assert.Equal(t, 3, nodesWith4T1Tasks)
	assert.Equal(t, 2, nodesWith3T1Tasks)

	t2Instances := 2

	// Add a new service with two instances. They should fill the nodes
	// that only have two tasks.
	err = s.Update(func(tx store.Tx) error {
		for i := 0; i != t2Instances; i++ {
			taskTemplate2.ID = fmt.Sprintf("t2id%d", i)
			assert.NoError(t, store.CreateTask(tx, taskTemplate2))
		}
		return nil
	})
	assert.NoError(t, err)

	t2Assignments := make(map[string]int)
	for i := 0; i != t2Instances; i++ {
		assignment := watchAssignment(t, watch)
		if !strings.HasPrefix(assignment.ID, "t2") {
			t.Fatal("got assignment for different kind of task")
		}
		t2Assignments[assignment.NodeID]++
	}

	assert.Len(t, t2Assignments, 2)

	for nodeID := range t2Assignments {
		assert.Equal(t, 3, t1Assignments[nodeID])
	}

	// Scale up service 1 to 21 tasks. It should cover the two nodes that
	// service 2 was assigned to, and also one other node.
	err = s.Update(func(tx store.Tx) error {
		for i := t1Instances; i != t1Instances+3; i++ {
			taskTemplate1.ID = fmt.Sprintf("t1id%d", i)
			assert.NoError(t, store.CreateTask(tx, taskTemplate1))
		}
		return nil
	})
	assert.NoError(t, err)

	var sharedNodes [2]string

	for i := 0; i != 3; i++ {
		assignment := watchAssignment(t, watch)
		if !strings.HasPrefix(assignment.ID, "t1") {
			t.Fatal("got assignment for different kind of task")
		}
		if t1Assignments[assignment.NodeID] == 5 {
			t.Fatal("more than one new task assigned to the same node")
		}
		t1Assignments[assignment.NodeID]++

		if t2Assignments[assignment.NodeID] != 0 {
			if sharedNodes[0] == "" {
				sharedNodes[0] = assignment.NodeID
			} else if sharedNodes[1] == "" {
				sharedNodes[1] = assignment.NodeID
			} else {
				t.Fatal("all three assignments went to nodes with service2 tasks")
			}
		}
	}

	assert.NotEmpty(t, sharedNodes[0])
	assert.NotEmpty(t, sharedNodes[1])
	assert.NotEqual(t, sharedNodes[0], sharedNodes[1])

	nodesWith4T1Tasks = 0
	nodesWith5T1Tasks := 0
	for nodeID, taskCount := range t1Assignments {
		if taskCount == 4 {
			nodesWith4T1Tasks++
		} else if taskCount == 5 {
			nodesWith5T1Tasks++
		} else {
			t.Fatalf("unexpected number of tasks %d on node %s", taskCount, nodeID)
		}
	}

	assert.Equal(t, 4, nodesWith4T1Tasks)
	assert.Equal(t, 1, nodesWith5T1Tasks)

	// Add another task from service2. It must not land on the node that
	// has 5 service1 tasks.
	err = s.Update(func(tx store.Tx) error {
		taskTemplate2.ID = "t2id4"
		assert.NoError(t, store.CreateTask(tx, taskTemplate2))
		return nil
	})
	assert.NoError(t, err)

	assignment := watchAssignment(t, watch)
	if assignment.ID != "t2id4" {
		t.Fatal("got assignment for different task")
	}

	if t2Assignments[assignment.NodeID] != 0 {
		t.Fatal("was scheduled on a node that already has a service2 task")
	}
	if t1Assignments[assignment.NodeID] == 5 {
		t.Fatal("was scheduled on the node that has the most service1 tasks")
	}
	t2Assignments[assignment.NodeID]++

	// Remove all tasks on node id1.
	err = s.Update(func(tx store.Tx) error {
		tasks, err := store.FindTasks(tx, store.ByNodeID("id1"))
		assert.NoError(t, err)
		for _, task := range tasks {
			assert.NoError(t, store.DeleteTask(tx, task.ID))
		}
		return nil
	})
	assert.NoError(t, err)

	t1Assignments["id1"] = 0
	t2Assignments["id1"] = 0

	// Add four instances of service1 and two instances of service2.
	// All instances of service1 should land on node "id1", and one
	// of the two service2 instances should as well.
	// Put these in a map to randomize the order in which they are
	// created.
	err = s.Update(func(tx store.Tx) error {
		tasksMap := make(map[string]*api.Task)
		for i := 22; i <= 25; i++ {
			taskTemplate1.ID = fmt.Sprintf("t1id%d", i)
			tasksMap[taskTemplate1.ID] = taskTemplate1.Copy()
		}
		for i := 5; i <= 6; i++ {
			taskTemplate2.ID = fmt.Sprintf("t2id%d", i)
			tasksMap[taskTemplate2.ID] = taskTemplate2.Copy()
		}
		for _, task := range tasksMap {
			assert.NoError(t, store.CreateTask(tx, task))
		}
		return nil
	})
	assert.NoError(t, err)

	for i := 0; i != 4+2; i++ {
		assignment := watchAssignment(t, watch)
		if strings.HasPrefix(assignment.ID, "t1") {
			t1Assignments[assignment.NodeID]++
		} else if strings.HasPrefix(assignment.ID, "t2") {
			t2Assignments[assignment.NodeID]++
		}
	}

	assert.Equal(t, 4, t1Assignments["id1"])
	assert.Equal(t, 1, t2Assignments["id1"])
}

func TestHA(t *testing.T) {
	t.Run("useSpecVersion=false", func(t *testing.T) { testHA(t, false) })
	t.Run("useSpecVersion=true", func(t *testing.T) { testHA(t, true) })
}

func testPreferences(t *testing.T, useSpecVersion bool) {
	ctx := context.Background()
	initialNodeSet := []*api.Node{
		{
			ID: "id1",
			Status: api.NodeStatus{
				State: api.NodeStatus_READY,
			},
			Spec: api.NodeSpec{
				Annotations: api.Annotations{
					Labels: map[string]string{
						"az": "az1",
					},
				},
			},
		},
		{
			ID: "id2",
			Status: api.NodeStatus{
				State: api.NodeStatus_READY,
			},
			Spec: api.NodeSpec{
				Annotations: api.Annotations{
					Labels: map[string]string{
						"az": "az2",
					},
				},
			},
		},
		{
			ID: "id3",
			Status: api.NodeStatus{
				State: api.NodeStatus_READY,
			},
			Spec: api.NodeSpec{
				Annotations: api.Annotations{
					Labels: map[string]string{
						"az": "az2",
					},
				},
			},
		},
		{
			ID: "id4",
			Status: api.NodeStatus{
				State: api.NodeStatus_READY,
			},
			Spec: api.NodeSpec{
				Annotations: api.Annotations{
					Labels: map[string]string{
						"az": "az2",
					},
				},
			},
		},
		{
			ID: "id5",
			Status: api.NodeStatus{
				State: api.NodeStatus_READY,
			},
			Spec: api.NodeSpec{
				Annotations: api.Annotations{
					Labels: map[string]string{
						"az": "az2",
					},
				},
			},
		},
	}

	taskTemplate1 := &api.Task{
		DesiredState: api.TaskStateRunning,
		ServiceID:    "service1",
		Spec: api.TaskSpec{
			Runtime: &api.TaskSpec_Container{
				Container: &api.ContainerSpec{
					Image: "v:1",
				},
			},
			Placement: &api.Placement{
				Preferences: []*api.PlacementPreference{
					{
						Preference: &api.PlacementPreference_Spread{
							Spread: &api.SpreadOver{
								SpreadDescriptor: "node.labels.az",
							},
						},
					},
				},
			},
		},
		Status: api.TaskStatus{
			State: api.TaskStatePending,
		},
	}

	if useSpecVersion {
		taskTemplate1.SpecVersion = &api.Version{Index: 1}
	}

	s := store.NewMemoryStore(nil)
	assert.NotNil(t, s)
	defer s.Close()

	t1Instances := 8

	err := s.Update(func(tx store.Tx) error {
		// Prepoulate nodes
		for _, n := range initialNodeSet {
			assert.NoError(t, store.CreateNode(tx, n))
		}

		// Prepopulate tasks from template 1
		for i := 0; i != t1Instances; i++ {
			taskTemplate1.ID = fmt.Sprintf("t1id%d", i)
			assert.NoError(t, store.CreateTask(tx, taskTemplate1))
		}
		return nil
	})
	assert.NoError(t, err)

	scheduler := New(s)

	watch, cancel := state.Watch(s.WatchQueue(), api.EventUpdateTask{})
	defer cancel()

	go func() {
		assert.NoError(t, scheduler.Run(ctx))
	}()
	defer scheduler.Stop()

	t1Assignments := make(map[string]int)
	for i := 0; i != t1Instances; i++ {
		assignment := watchAssignment(t, watch)
		if !strings.HasPrefix(assignment.ID, "t1") {
			t.Fatal("got assignment for different kind of task")
		}
		t1Assignments[assignment.NodeID]++
	}

	assert.Len(t, t1Assignments, 5)
	assert.Equal(t, 4, t1Assignments["id1"])
	assert.Equal(t, 1, t1Assignments["id2"])
	assert.Equal(t, 1, t1Assignments["id3"])
	assert.Equal(t, 1, t1Assignments["id4"])
	assert.Equal(t, 1, t1Assignments["id5"])
}

func TestPreferences(t *testing.T) {
	t.Run("useSpecVersion=false", func(t *testing.T) { testPreferences(t, false) })
	t.Run("useSpecVersion=true", func(t *testing.T) { testPreferences(t, true) })
}

func testMultiplePreferences(t *testing.T, useSpecVersion bool) {
	ctx := context.Background()
	initialNodeSet := []*api.Node{
		{
			ID: "id0",
			Status: api.NodeStatus{
				State: api.NodeStatus_READY,
			},
			Spec: api.NodeSpec{
				Annotations: api.Annotations{
					Labels: map[string]string{
						"az":   "az1",
						"rack": "rack1",
					},
				},
			},
			Description: &api.NodeDescription{
				Resources: &api.Resources{
					NanoCPUs:    1e9,
					MemoryBytes: 1e8,
					Generic: []*api.GenericResource{
						genericresource.NewDiscrete("apple", 1),
					},
				},
			},
		},
		{
			ID: "id1",
			Status: api.NodeStatus{
				State: api.NodeStatus_READY,
			},
			Spec: api.NodeSpec{
				Annotations: api.Annotations{
					Labels: map[string]string{
						"az":   "az1",
						"rack": "rack1",
					},
				},
			},
			Description: &api.NodeDescription{
				Resources: &api.Resources{
					NanoCPUs:    1e9,
					MemoryBytes: 1e9,
					Generic: []*api.GenericResource{
						genericresource.NewDiscrete("apple", 10),
					},
				},
			},
		},
		{
			ID: "id2",
			Status: api.NodeStatus{
				State: api.NodeStatus_READY,
			},
			Spec: api.NodeSpec{
				Annotations: api.Annotations{
					Labels: map[string]string{
						"az":   "az2",
						"rack": "rack1",
					},
				},
			},
			Description: &api.NodeDescription{
				Resources: &api.Resources{
					NanoCPUs:    1e9,
					MemoryBytes: 1e9,
					Generic: []*api.GenericResource{
						genericresource.NewDiscrete("apple", 6),
					},
				},
			},
		},
		{
			ID: "id3",
			Status: api.NodeStatus{
				State: api.NodeStatus_READY,
			},
			Spec: api.NodeSpec{
				Annotations: api.Annotations{
					Labels: map[string]string{
						"az":   "az2",
						"rack": "rack1",
					},
				},
			},
			Description: &api.NodeDescription{
				Resources: &api.Resources{
					NanoCPUs:    1e9,
					MemoryBytes: 1e9,
					Generic: []*api.GenericResource{
						genericresource.NewDiscrete("apple", 6),
					},
				},
			},
		},
		{
			ID: "id4",
			Status: api.NodeStatus{
				State: api.NodeStatus_READY,
			},
			Spec: api.NodeSpec{
				Annotations: api.Annotations{
					Labels: map[string]string{
						"az":   "az2",
						"rack": "rack1",
					},
				},
			},
			Description: &api.NodeDescription{
				Resources: &api.Resources{
					NanoCPUs:    1e9,
					MemoryBytes: 1e9,
					Generic: []*api.GenericResource{
						genericresource.NewDiscrete("apple", 6),
					},
				},
			},
		},
		{
			ID: "id5",
			Status: api.NodeStatus{
				State: api.NodeStatus_READY,
			},
			Spec: api.NodeSpec{
				Annotations: api.Annotations{
					Labels: map[string]string{
						"az":   "az2",
						"rack": "rack2",
					},
				},
			},
			Description: &api.NodeDescription{
				Resources: &api.Resources{
					NanoCPUs:    1e9,
					MemoryBytes: 1e9,
					Generic: []*api.GenericResource{
						genericresource.NewDiscrete("apple", 6),
					},
				},
			},
		},
		{
			ID: "id6",
			Status: api.NodeStatus{
				State: api.NodeStatus_READY,
			},
			Spec: api.NodeSpec{
				Annotations: api.Annotations{
					Labels: map[string]string{
						"az":   "az2",
						"rack": "rack2",
					},
				},
			},
			Description: &api.NodeDescription{
				Resources: &api.Resources{
					NanoCPUs:    1e9,
					MemoryBytes: 1e9,
					Generic: []*api.GenericResource{
						genericresource.NewDiscrete("apple", 6),
					},
				},
			},
		},
	}

	taskTemplate1 := &api.Task{
		DesiredState: api.TaskStateRunning,
		ServiceID:    "service1",
		Spec: api.TaskSpec{
			Runtime: &api.TaskSpec_Container{
				Container: &api.ContainerSpec{
					Image: "v:1",
				},
			},
			Placement: &api.Placement{
				Preferences: []*api.PlacementPreference{
					{
						Preference: &api.PlacementPreference_Spread{
							Spread: &api.SpreadOver{
								SpreadDescriptor: "node.labels.az",
							},
						},
					},
					{
						Preference: &api.PlacementPreference_Spread{
							Spread: &api.SpreadOver{
								SpreadDescriptor: "node.labels.rack",
							},
						},
					},
				},
			},
			Resources: &api.ResourceRequirements{
				Reservations: &api.Resources{
					MemoryBytes: 2e8,
					Generic: []*api.GenericResource{
						genericresource.NewDiscrete("apple", 2),
					},
				},
			},
		},
		Status: api.TaskStatus{
			State: api.TaskStatePending,
		},
	}

	if useSpecVersion {
		taskTemplate1.SpecVersion = &api.Version{Index: 1}
	}

	s := store.NewMemoryStore(nil)
	assert.NotNil(t, s)
	defer s.Close()

	t1Instances := 12

	err := s.Update(func(tx store.Tx) error {
		// Prepoulate nodes
		for _, n := range initialNodeSet {
			assert.NoError(t, store.CreateNode(tx, n))
		}

		// Prepopulate tasks from template 1
		for i := 0; i != t1Instances; i++ {
			taskTemplate1.ID = fmt.Sprintf("t1id%d", i)
			assert.NoError(t, store.CreateTask(tx, taskTemplate1))
		}
		return nil
	})
	assert.NoError(t, err)

	scheduler := New(s)

	watch, cancel := state.Watch(s.WatchQueue(), api.EventUpdateTask{})
	defer cancel()

	go func() {
		assert.NoError(t, scheduler.Run(ctx))
	}()
	defer scheduler.Stop()

	t1Assignments := make(map[string]int)
	for i := 0; i != t1Instances; i++ {
		assignment := watchAssignment(t, watch)
		if !strings.HasPrefix(assignment.ID, "t1") {
			t.Fatal("got assignment for different kind of task")
		}
		t1Assignments[assignment.NodeID]++
	}

	assert.Len(t, t1Assignments, 6)

	// There should be no tasks assigned to id0 because it doesn't meet the
	// resource requirements.
	assert.Equal(t, 0, t1Assignments["id0"])

	// There should be 5 tasks assigned to id1 because half of the 12 tasks
	// should ideally end up in az1, but id1 can only accommodate 5 due to
	// resource requirements.
	assert.Equal(t, 5, t1Assignments["id1"])

	// The remaining 7 tasks should be spread across rack1 and rack2 of
	// az2.

	if t1Assignments["id2"]+t1Assignments["id3"]+t1Assignments["id4"] == 4 {
		// If rack1 gets 4 and rack2 gets 3, then one of id[2-4] will have two
		// tasks and the others will have one.
		if t1Assignments["id2"] == 2 {
			assert.Equal(t, 1, t1Assignments["id3"])
			assert.Equal(t, 1, t1Assignments["id4"])
		} else if t1Assignments["id3"] == 2 {
			assert.Equal(t, 1, t1Assignments["id2"])
			assert.Equal(t, 1, t1Assignments["id4"])
		} else {
			assert.Equal(t, 1, t1Assignments["id2"])
			assert.Equal(t, 1, t1Assignments["id3"])
			assert.Equal(t, 2, t1Assignments["id4"])
		}

		// either id5 or id6 should end up with 2 tasks
		if t1Assignments["id5"] == 1 {
			assert.Equal(t, 2, t1Assignments["id6"])
		} else {
			assert.Equal(t, 2, t1Assignments["id5"])
			assert.Equal(t, 1, t1Assignments["id6"])
		}
	} else if t1Assignments["id2"]+t1Assignments["id3"]+t1Assignments["id4"] == 3 {
		// If rack2 gets 4 and rack1 gets 3, then id[2-4] will each get
		// 1 task and id[5-6] will each get 2 tasks.
		assert.Equal(t, 1, t1Assignments["id2"])
		assert.Equal(t, 1, t1Assignments["id3"])
		assert.Equal(t, 1, t1Assignments["id4"])
		assert.Equal(t, 2, t1Assignments["id5"])
		assert.Equal(t, 2, t1Assignments["id6"])
	} else {
		t.Fatal("unexpected task layout")
	}
}

func TestMultiplePreferences(t *testing.T) {
	t.Run("useSpecVersion=false", func(t *testing.T) { testMultiplePreferences(t, false) })
	t.Run("useSpecVersion=true", func(t *testing.T) { testMultiplePreferences(t, true) })
}

func TestSchedulerNoReadyNodes(t *testing.T) {
	ctx := context.Background()
	initialTask := &api.Task{
		ID:           "id1",
		ServiceID:    "serviceID1",
		DesiredState: api.TaskStateRunning,
		ServiceAnnotations: api.Annotations{
			Name: "name1",
		},
		Status: api.TaskStatus{
			State: api.TaskStatePending,
		},
	}

	s := store.NewMemoryStore(nil)
	assert.NotNil(t, s)
	defer s.Close()

	err := s.Update(func(tx store.Tx) error {
		// Add initial service and task
		assert.NoError(t, store.CreateService(tx, &api.Service{ID: "serviceID1"}))
		assert.NoError(t, store.CreateTask(tx, initialTask))
		return nil
	})
	assert.NoError(t, err)

	scheduler := New(s)

	watch, cancel := state.Watch(s.WatchQueue(), api.EventUpdateTask{})
	defer cancel()

	go func() {
		assert.NoError(t, scheduler.Run(ctx))
	}()
	defer scheduler.Stop()

	failure := watchAssignmentFailure(t, watch)
	assert.Equal(t, "no suitable node", failure.Status.Err)

	err = s.Update(func(tx store.Tx) error {
		// Create a ready node. The task should get assigned to this
		// node.
		node := &api.Node{
			ID: "newnode",
			Spec: api.NodeSpec{
				Annotations: api.Annotations{
					Name: "newnode",
				},
			},
			Status: api.NodeStatus{
				State: api.NodeStatus_READY,
			},
		}
		assert.NoError(t, store.CreateNode(tx, node))
		return nil
	})
	assert.NoError(t, err)

	assignment := watchAssignment(t, watch)
	assert.Equal(t, "newnode", assignment.NodeID)
}

func TestSchedulerFaultyNode(t *testing.T) {
	ctx := context.Background()

	replicatedTaskTemplate := &api.Task{
		ServiceID:    "service1",
		DesiredState: api.TaskStateRunning,
		ServiceAnnotations: api.Annotations{
			Name: "name1",
		},
		Status: api.TaskStatus{
			State: api.TaskStatePending,
		},
	}

	preassignedTaskTemplate := &api.Task{
		ServiceID:    "service2",
		NodeID:       "id1",
		DesiredState: api.TaskStateRunning,
		ServiceAnnotations: api.Annotations{
			Name: "name2",
		},
		Status: api.TaskStatus{
			State: api.TaskStatePending,
		},
	}

	node1 := &api.Node{
		ID: "id1",
		Spec: api.NodeSpec{
			Annotations: api.Annotations{
				Name: "id1",
			},
		},
		Status: api.NodeStatus{
			State: api.NodeStatus_READY,
		},
	}

	node2 := &api.Node{
		ID: "id2",
		Spec: api.NodeSpec{
			Annotations: api.Annotations{
				Name: "id2",
			},
		},
		Status: api.NodeStatus{
			State: api.NodeStatus_READY,
		},
	}

	s := store.NewMemoryStore(nil)
	assert.NotNil(t, s)
	defer s.Close()

	err := s.Update(func(tx store.Tx) error {
		// Add initial nodes, and one task of each type assigned to node id1
		assert.NoError(t, store.CreateNode(tx, node1))
		assert.NoError(t, store.CreateNode(tx, node2))

		task1 := replicatedTaskTemplate.Copy()
		task1.ID = "id1"
		task1.NodeID = "id1"
		task1.Status.State = api.TaskStateRunning
		assert.NoError(t, store.CreateTask(tx, task1))

		task2 := preassignedTaskTemplate.Copy()
		task2.ID = "id2"
		task2.NodeID = "id1"
		task2.Status.State = api.TaskStateRunning
		assert.NoError(t, store.CreateTask(tx, task2))
		return nil
	})
	assert.NoError(t, err)

	scheduler := New(s)

	watch, cancel := state.Watch(s.WatchQueue(), api.EventUpdateTask{})
	defer cancel()

	go func() {
		assert.NoError(t, scheduler.Run(ctx))
	}()
	defer scheduler.Stop()

	for i := 0; i != 8; i++ {
		// Simulate a task failure cycle
		newReplicatedTask := replicatedTaskTemplate.Copy()
		newReplicatedTask.ID = identity.NewID()

		err = s.Update(func(tx store.Tx) error {
			assert.NoError(t, store.CreateTask(tx, newReplicatedTask))
			return nil
		})
		assert.NoError(t, err)

		assignment := watchAssignment(t, watch)
		assert.Equal(t, newReplicatedTask.ID, assignment.ID)

		if i < 5 {
			// The first 5 attempts should be assigned to node id2 because
			// it has no replicas of the service.
			assert.Equal(t, "id2", assignment.NodeID)
		} else {
			// The next ones should be assigned to id1, since we'll
			// flag id2 as potentially faulty.
			assert.Equal(t, "id1", assignment.NodeID)
		}

		node2Info, err := scheduler.nodeSet.nodeInfo("id2")
		assert.NoError(t, err)
		expectedNode2Failures := i
		if i > 5 {
			expectedNode2Failures = 5
		}
		assert.Len(t, node2Info.recentFailures[versionedService{serviceID: "service1"}], expectedNode2Failures)

		node1Info, err := scheduler.nodeSet.nodeInfo("id1")
		assert.NoError(t, err)

		expectedNode1Failures := i - 5
		if i < 5 {
			expectedNode1Failures = 0
		}
		assert.Len(t, node1Info.recentFailures[versionedService{serviceID: "service1"}], expectedNode1Failures)

		newPreassignedTask := preassignedTaskTemplate.Copy()
		newPreassignedTask.ID = identity.NewID()

		err = s.Update(func(tx store.Tx) error {
			assert.NoError(t, store.CreateTask(tx, newPreassignedTask))
			return nil
		})
		assert.NoError(t, err)

		assignment = watchAssignment(t, watch)
		assert.Equal(t, newPreassignedTask.ID, assignment.ID)

		// The preassigned task is always assigned to node id1
		assert.Equal(t, "id1", assignment.NodeID)

		// The service associated with the preassigned task will not be
		// marked as
		nodeInfo, err := scheduler.nodeSet.nodeInfo("id1")
		assert.NoError(t, err)
		assert.Len(t, nodeInfo.recentFailures[versionedService{serviceID: "service2"}], 0)

		err = s.Update(func(tx store.Tx) error {
			newReplicatedTask := store.GetTask(tx, newReplicatedTask.ID)
			require.NotNil(t, newReplicatedTask)
			newReplicatedTask.Status.State = api.TaskStateFailed
			assert.NoError(t, store.UpdateTask(tx, newReplicatedTask))

			newPreassignedTask := store.GetTask(tx, newPreassignedTask.ID)
			require.NotNil(t, newPreassignedTask)
			newPreassignedTask.Status.State = api.TaskStateFailed
			assert.NoError(t, store.UpdateTask(tx, newPreassignedTask))

			return nil
		})
		assert.NoError(t, err)
	}
}

func TestSchedulerFaultyNodeSpecVersion(t *testing.T) {
	ctx := context.Background()

	taskTemplate := &api.Task{
		ServiceID:    "service1",
		SpecVersion:  &api.Version{Index: 1},
		DesiredState: api.TaskStateRunning,
		ServiceAnnotations: api.Annotations{
			Name: "name1",
		},
		Status: api.TaskStatus{
			State: api.TaskStatePending,
		},
	}

	node1 := &api.Node{
		ID: "id1",
		Spec: api.NodeSpec{
			Annotations: api.Annotations{
				Name: "id1",
			},
		},
		Status: api.NodeStatus{
			State: api.NodeStatus_READY,
		},
	}

	node2 := &api.Node{
		ID: "id2",
		Spec: api.NodeSpec{
			Annotations: api.Annotations{
				Name: "id2",
			},
		},
		Status: api.NodeStatus{
			State: api.NodeStatus_READY,
		},
	}

	s := store.NewMemoryStore(nil)
	assert.NotNil(t, s)
	defer s.Close()

	err := s.Update(func(tx store.Tx) error {
		// Add initial nodes, and one task assigned to node id1
		assert.NoError(t, store.CreateNode(tx, node1))
		assert.NoError(t, store.CreateNode(tx, node2))

		task1 := taskTemplate.Copy()
		task1.ID = "id1"
		task1.NodeID = "id1"
		task1.Status.State = api.TaskStateRunning
		assert.NoError(t, store.CreateTask(tx, task1))
		return nil
	})
	assert.NoError(t, err)

	scheduler := New(s)

	watch, cancel := state.Watch(s.WatchQueue(), api.EventUpdateTask{})
	defer cancel()

	go func() {
		assert.NoError(t, scheduler.Run(ctx))
	}()
	defer scheduler.Stop()

	for i := 0; i != 15; i++ {
		// Simulate a task failure cycle
		newTask := taskTemplate.Copy()
		newTask.ID = identity.NewID()

		// After the condition for node faultiness has been reached,
		// bump the spec version to simulate a service update.
		if i > 5 {
			newTask.SpecVersion.Index++
		}

		err = s.Update(func(tx store.Tx) error {
			assert.NoError(t, store.CreateTask(tx, newTask))
			return nil
		})
		assert.NoError(t, err)

		assignment := watchAssignment(t, watch)
		assert.Equal(t, newTask.ID, assignment.ID)

		if i < 5 || (i > 5 && i < 11) {
			// The first 5 attempts should be assigned to node id2 because
			// it has no replicas of the service.
			// Same with i=6 to i=10 inclusive, which is repeating the
			// same behavior with a different SpecVersion.
			assert.Equal(t, "id2", assignment.NodeID)
		} else {
			// The next ones should be assigned to id1, since we'll
			// flag id2 as potentially faulty.
			assert.Equal(t, "id1", assignment.NodeID)
		}

		node1Info, err := scheduler.nodeSet.nodeInfo("id1")
		assert.NoError(t, err)
		node2Info, err := scheduler.nodeSet.nodeInfo("id2")
		assert.NoError(t, err)
		expectedNode1Spec1Failures := 0
		expectedNode1Spec2Failures := 0
		expectedNode2Spec1Failures := i
		expectedNode2Spec2Failures := 0
		if i > 5 {
			expectedNode1Spec1Failures = 1
			expectedNode2Spec1Failures = 5
			expectedNode2Spec2Failures = i - 6
		}
		if i > 11 {
			expectedNode1Spec2Failures = i - 11
			expectedNode2Spec2Failures = 5
		}
		assert.Len(t, node1Info.recentFailures[versionedService{serviceID: "service1", specVersion: api.Version{Index: 1}}], expectedNode1Spec1Failures)
		assert.Len(t, node1Info.recentFailures[versionedService{serviceID: "service1", specVersion: api.Version{Index: 2}}], expectedNode1Spec2Failures)
		assert.Len(t, node2Info.recentFailures[versionedService{serviceID: "service1", specVersion: api.Version{Index: 1}}], expectedNode2Spec1Failures)
		assert.Len(t, node2Info.recentFailures[versionedService{serviceID: "service1", specVersion: api.Version{Index: 2}}], expectedNode2Spec2Failures)

		err = s.Update(func(tx store.Tx) error {
			newTask := store.GetTask(tx, newTask.ID)
			require.NotNil(t, newTask)
			newTask.Status.State = api.TaskStateFailed
			assert.NoError(t, store.UpdateTask(tx, newTask))
			return nil
		})
		assert.NoError(t, err)
	}
}

func TestSchedulerResourceConstraint(t *testing.T) {
	ctx := context.Background()
	// Create a ready node without enough memory to run the task.
	underprovisionedNode := &api.Node{
		ID: "underprovisioned",
		Spec: api.NodeSpec{
			Annotations: api.Annotations{
				Name: "underprovisioned",
			},
		},
		Status: api.NodeStatus{
			State: api.NodeStatus_READY,
		},
		Description: &api.NodeDescription{
			Resources: &api.Resources{
				NanoCPUs:    1e9,
				MemoryBytes: 1e9,
				Generic: append(
					genericresource.NewSet("orange", "blue"),
					genericresource.NewDiscrete("apple", 1),
				),
			},
		},
	}

	// Non-ready nodes that satisfy the constraints but shouldn't be used
	nonready1 := &api.Node{
		ID: "nonready1",
		Spec: api.NodeSpec{
			Annotations: api.Annotations{
				Name: "nonready1",
			},
		},
		Status: api.NodeStatus{
			State: api.NodeStatus_UNKNOWN,
		},
		Description: &api.NodeDescription{
			Resources: &api.Resources{
				NanoCPUs:    2e9,
				MemoryBytes: 2e9,
				Generic: append(
					genericresource.NewSet("orange", "blue", "red"),
					genericresource.NewDiscrete("apple", 2),
				),
			},
		},
	}
	nonready2 := &api.Node{
		ID: "nonready2",
		Spec: api.NodeSpec{
			Annotations: api.Annotations{
				Name: "nonready2",
			},
		},
		Status: api.NodeStatus{
			State: api.NodeStatus_UNKNOWN,
		},
		Description: &api.NodeDescription{
			Resources: &api.Resources{
				NanoCPUs:    2e9,
				MemoryBytes: 2e9,
				Generic: append(
					genericresource.NewSet("orange", "blue", "red"),
					genericresource.NewDiscrete("apple", 2),
				),
			},
		},
	}

	initialTask := &api.Task{
		ID:           "id1",
		ServiceID:    "serviceID1",
		DesiredState: api.TaskStateRunning,
		Spec: api.TaskSpec{
			Runtime: &api.TaskSpec_Container{
				Container: &api.ContainerSpec{},
			},
			Resources: &api.ResourceRequirements{
				Reservations: &api.Resources{
					MemoryBytes: 2e9,
					Generic: []*api.GenericResource{
						genericresource.NewDiscrete("orange", 2),
						genericresource.NewDiscrete("apple", 2),
					},
				},
			},
		},
		ServiceAnnotations: api.Annotations{
			Name: "name1",
		},
		Status: api.TaskStatus{
			State: api.TaskStatePending,
		},
	}

	initialService := &api.Service{
		ID: "serviceID1",
	}

	s := store.NewMemoryStore(nil)
	assert.NotNil(t, s)
	defer s.Close()

	err := s.Update(func(tx store.Tx) error {
		// Add initial node, service and task
		assert.NoError(t, store.CreateService(tx, initialService))
		assert.NoError(t, store.CreateTask(tx, initialTask))
		assert.NoError(t, store.CreateNode(tx, underprovisionedNode))
		assert.NoError(t, store.CreateNode(tx, nonready1))
		assert.NoError(t, store.CreateNode(tx, nonready2))
		return nil
	})
	assert.NoError(t, err)

	scheduler := New(s)

	watch, cancel := state.Watch(s.WatchQueue(), api.EventUpdateTask{})
	defer cancel()

	go func() {
		assert.NoError(t, scheduler.Run(ctx))
	}()
	defer scheduler.Stop()

	failure := watchAssignmentFailure(t, watch)
	assert.Equal(t, "no suitable node (2 nodes not available for new tasks; insufficient resources on 1 node)", failure.Status.Err)

	err = s.Update(func(tx store.Tx) error {
		// Create a node with enough memory. The task should get
		// assigned to this node.
		node := &api.Node{
			ID: "bignode",
			Spec: api.NodeSpec{
				Annotations: api.Annotations{
					Name: "bignode",
				},
			},
			Description: &api.NodeDescription{
				Resources: &api.Resources{
					NanoCPUs:    4e9,
					MemoryBytes: 8e9,
					Generic: append(
						genericresource.NewSet("orange", "blue", "red", "green"),
						genericresource.NewDiscrete("apple", 4),
					),
				},
			},
			Status: api.NodeStatus{
				State: api.NodeStatus_READY,
			},
		}
		assert.NoError(t, store.CreateNode(tx, node))
		return nil
	})
	assert.NoError(t, err)

	assignment := watchAssignment(t, watch)
	assert.Equal(t, "bignode", assignment.NodeID)
}

func TestSchedulerResourceConstraintHA(t *testing.T) {
	// node 1 starts with 1 task, node 2 starts with 3 tasks.
	// however, node 1 only has enough memory to schedule one more task.

	ctx := context.Background()
	node1 := &api.Node{
		ID: "id1",
		Spec: api.NodeSpec{
			Annotations: api.Annotations{
				Name: "id1",
			},
		},
		Status: api.NodeStatus{
			State: api.NodeStatus_READY,
		},
		Description: &api.NodeDescription{
			Resources: &api.Resources{
				MemoryBytes: 1e9,
				Generic: []*api.GenericResource{
					genericresource.NewDiscrete("apple", 2),
				},
			},
		},
	}
	node2 := &api.Node{
		ID: "id2",
		Spec: api.NodeSpec{
			Annotations: api.Annotations{
				Name: "id2",
			},
		},
		Status: api.NodeStatus{
			State: api.NodeStatus_READY,
		},
		Description: &api.NodeDescription{
			Resources: &api.Resources{
				MemoryBytes: 1e11,
				Generic: []*api.GenericResource{
					genericresource.NewDiscrete("apple", 5),
				},
			},
		},
	}

	taskTemplate := &api.Task{
		DesiredState: api.TaskStateRunning,
		Spec: api.TaskSpec{
			Runtime: &api.TaskSpec_Container{
				Container: &api.ContainerSpec{},
			},
			Resources: &api.ResourceRequirements{
				Reservations: &api.Resources{
					MemoryBytes: 5e8,
					Generic: []*api.GenericResource{
						genericresource.NewDiscrete("apple", 1),
					},
				},
			},
		},
		ServiceAnnotations: api.Annotations{
			Name: "name1",
		},
		Status: api.TaskStatus{
			State: api.TaskStatePending,
		},
	}

	s := store.NewMemoryStore(nil)
	assert.NotNil(t, s)
	defer s.Close()

	err := s.Update(func(tx store.Tx) error {
		// Add initial node and task
		assert.NoError(t, store.CreateNode(tx, node1))
		assert.NoError(t, store.CreateNode(tx, node2))

		// preassigned tasks
		task1 := taskTemplate.Copy()
		task1.ID = "id1"
		task1.NodeID = "id1"
		task1.Status.State = api.TaskStateRunning
		assert.NoError(t, store.CreateTask(tx, task1))

		task2 := taskTemplate.Copy()
		task2.ID = "id2"
		task2.NodeID = "id2"
		task2.Status.State = api.TaskStateRunning
		assert.NoError(t, store.CreateTask(tx, task2))

		task3 := taskTemplate.Copy()
		task3.ID = "id3"
		task3.NodeID = "id2"
		task3.Status.State = api.TaskStateRunning
		assert.NoError(t, store.CreateTask(tx, task3))

		task4 := taskTemplate.Copy()
		task4.ID = "id4"
		task4.NodeID = "id2"
		task4.Status.State = api.TaskStateRunning
		assert.NoError(t, store.CreateTask(tx, task4))

		// tasks to assign
		task5 := taskTemplate.Copy()
		task5.ID = "id5"
		assert.NoError(t, store.CreateTask(tx, task5))

		task6 := taskTemplate.Copy()
		task6.ID = "id6"
		assert.NoError(t, store.CreateTask(tx, task6))

		return nil
	})
	assert.NoError(t, err)

	scheduler := New(s)

	watch, cancel := state.Watch(s.WatchQueue(), api.EventUpdateTask{})
	defer cancel()

	go func() {
		assert.NoError(t, scheduler.Run(ctx))
	}()
	defer scheduler.Stop()

	assignment1 := watchAssignment(t, watch)
	if assignment1.ID != "id5" && assignment1.ID != "id6" {
		t.Fatal("assignment for unexpected task")
	}
	assignment2 := watchAssignment(t, watch)
	if assignment1.ID == "id5" {
		assert.Equal(t, "id6", assignment2.ID)
	} else {
		assert.Equal(t, "id5", assignment2.ID)
	}

	if assignment1.NodeID == "id1" {
		assert.Equal(t, "id2", assignment2.NodeID)
	} else {
		assert.Equal(t, "id1", assignment2.NodeID)
	}
}

func TestSchedulerResourceConstraintDeadTask(t *testing.T) {
	ctx := context.Background()
	// Create a ready node without enough memory to run the task.
	node := &api.Node{
		ID: "id1",
		Spec: api.NodeSpec{
			Annotations: api.Annotations{
				Name: "node",
			},
		},
		Status: api.NodeStatus{
			State: api.NodeStatus_READY,
		},
		Description: &api.NodeDescription{
			Resources: &api.Resources{
				NanoCPUs:    1e9,
				MemoryBytes: 1e9,
				Generic: []*api.GenericResource{
					genericresource.NewDiscrete("apple", 4),
				},
			},
		},
	}

	bigTask1 := &api.Task{
		DesiredState: api.TaskStateRunning,
		ID:           "id1",
		ServiceID:    "serviceID1",
		Spec: api.TaskSpec{
			Resources: &api.ResourceRequirements{
				Reservations: &api.Resources{
					MemoryBytes: 8e8,
					Generic: []*api.GenericResource{
						genericresource.NewDiscrete("apple", 3),
					},
				},
			},
		},
		ServiceAnnotations: api.Annotations{
			Name: "big",
		},
		Status: api.TaskStatus{
			State: api.TaskStatePending,
		},
	}

	bigTask2 := bigTask1.Copy()
	bigTask2.ID = "id2"

	bigService := &api.Service{
		ID: "serviceID1",
	}

	s := store.NewMemoryStore(nil)
	assert.NotNil(t, s)
	defer s.Close()

	err := s.Update(func(tx store.Tx) error {
		// Add initial node, service and task
		assert.NoError(t, store.CreateService(tx, bigService))
		assert.NoError(t, store.CreateNode(tx, node))
		assert.NoError(t, store.CreateTask(tx, bigTask1))
		return nil
	})
	assert.NoError(t, err)

	scheduler := New(s)

	watch, cancel := state.Watch(s.WatchQueue(), api.EventUpdateTask{})
	defer cancel()

	go func() {
		assert.NoError(t, scheduler.Run(ctx))
	}()
	defer scheduler.Stop()

	// The task fits, so it should get assigned
	assignment := watchAssignment(t, watch)
	assert.Equal(t, "id1", assignment.ID)
	assert.Equal(t, "id1", assignment.NodeID)

	err = s.Update(func(tx store.Tx) error {
		// Add a second task. It shouldn't get assigned because of
		// resource constraints.
		return store.CreateTask(tx, bigTask2)
	})
	assert.NoError(t, err)

	failure := watchAssignmentFailure(t, watch)
	assert.Equal(t, "no suitable node (insufficient resources on 1 node)", failure.Status.Err)

	err = s.Update(func(tx store.Tx) error {
		// The task becomes dead
		updatedTask := store.GetTask(tx, bigTask1.ID)
		updatedTask.Status.State = api.TaskStateShutdown
		return store.UpdateTask(tx, updatedTask)
	})
	assert.NoError(t, err)

	// With the first task no longer consuming resources, the second
	// one can be scheduled.
	assignment = watchAssignment(t, watch)
	assert.Equal(t, "id2", assignment.ID)
	assert.Equal(t, "id1", assignment.NodeID)
}

func TestSchedulerPreexistingDeadTask(t *testing.T) {
	ctx := context.Background()
	// Create a ready node without enough memory to run two tasks at once.
	node := &api.Node{
		ID: "id1",
		Spec: api.NodeSpec{
			Annotations: api.Annotations{
				Name: "node",
			},
		},
		Status: api.NodeStatus{
			State: api.NodeStatus_READY,
		},
		Description: &api.NodeDescription{
			Resources: &api.Resources{
				NanoCPUs:    1e9,
				MemoryBytes: 1e9,
				Generic: []*api.GenericResource{
					genericresource.NewDiscrete("apple", 1),
				},
			},
		},
	}

	deadTask := &api.Task{
		DesiredState: api.TaskStateRunning,
		ID:           "id1",
		NodeID:       "id1",
		Spec: api.TaskSpec{
			Resources: &api.ResourceRequirements{
				Reservations: &api.Resources{
					MemoryBytes: 8e8,
					Generic: []*api.GenericResource{
						genericresource.NewDiscrete("apple", 1),
					},
				},
			},
		},
		ServiceAnnotations: api.Annotations{
			Name: "big",
		},
		Status: api.TaskStatus{
			State: api.TaskStateShutdown,
		},
	}

	bigTask2 := deadTask.Copy()
	bigTask2.ID = "id2"
	bigTask2.Status.State = api.TaskStatePending

	s := store.NewMemoryStore(nil)
	assert.NotNil(t, s)
	defer s.Close()

	err := s.Update(func(tx store.Tx) error {
		// Add initial node and task
		assert.NoError(t, store.CreateNode(tx, node))
		assert.NoError(t, store.CreateTask(tx, deadTask))
		return nil
	})
	assert.NoError(t, err)

	scheduler := New(s)

	watch, cancel := state.Watch(s.WatchQueue(), api.EventUpdateTask{})
	defer cancel()

	go func() {
		assert.NoError(t, scheduler.Run(ctx))
	}()
	defer scheduler.Stop()

	err = s.Update(func(tx store.Tx) error {
		// Add a second task. It should get assigned because the task
		// using the resources is past the running state.
		return store.CreateTask(tx, bigTask2)
	})
	assert.NoError(t, err)

	assignment := watchAssignment(t, watch)
	assert.Equal(t, "id2", assignment.ID)
	assert.Equal(t, "id1", assignment.NodeID)
}

func TestSchedulerCompatiblePlatform(t *testing.T) {
	ctx := context.Background()
	// create tasks
	// task1 - has a node it can run on
	task1 := &api.Task{
		ID:           "id1",
		ServiceID:    "serviceID1",
		DesiredState: api.TaskStateRunning,
		ServiceAnnotations: api.Annotations{
			Name: "name1",
		},
		Status: api.TaskStatus{
			State: api.TaskStatePending,
		},
		Spec: api.TaskSpec{
			Placement: &api.Placement{
				Platforms: []*api.Platform{
					{
						Architecture: "amd64",
						OS:           "linux",
					},
				},
			},
		},
	}

	// task2 - has no node it can run on
	task2 := &api.Task{
		ID:           "id2",
		ServiceID:    "serviceID1",
		DesiredState: api.TaskStateRunning,
		ServiceAnnotations: api.Annotations{
			Name: "name2",
		},
		Status: api.TaskStatus{
			State: api.TaskStatePending,
		},
		Spec: api.TaskSpec{
			Placement: &api.Placement{
				Platforms: []*api.Platform{
					{
						Architecture: "arm",
						OS:           "linux",
					},
				},
			},
		},
	}

	// task3 - no platform constraints, should run on any node
	task3 := &api.Task{
		ID:           "id3",
		ServiceID:    "serviceID1",
		DesiredState: api.TaskStateRunning,
		ServiceAnnotations: api.Annotations{
			Name: "name3",
		},
		Status: api.TaskStatus{
			State: api.TaskStatePending,
		},
	}

	// task4 - only OS constraint, is runnable on any linux node
	task4 := &api.Task{
		ID:           "id4",
		ServiceID:    "serviceID1",
		DesiredState: api.TaskStateRunning,
		ServiceAnnotations: api.Annotations{
			Name: "name4",
		},
		Status: api.TaskStatus{
			State: api.TaskStatePending,
		},
		Spec: api.TaskSpec{
			Placement: &api.Placement{
				Platforms: []*api.Platform{
					{
						Architecture: "",
						OS:           "linux",
					},
				},
			},
		},
	}

	// task5 - supported on multiple platforms
	task5 := &api.Task{
		ID:           "id5",
		ServiceID:    "serviceID1",
		DesiredState: api.TaskStateRunning,
		ServiceAnnotations: api.Annotations{
			Name: "name5",
		},
		Status: api.TaskStatus{
			State: api.TaskStatePending,
		},
		Spec: api.TaskSpec{
			Placement: &api.Placement{
				Platforms: []*api.Platform{
					{
						Architecture: "amd64",
						OS:           "linux",
					},
					{
						Architecture: "x86_64",
						OS:           "windows",
					},
				},
			},
		},
	}

	node1 := &api.Node{
		ID: "node1",
		Spec: api.NodeSpec{
			Annotations: api.Annotations{
				Name: "node1",
			},
		},
		Status: api.NodeStatus{
			State: api.NodeStatus_READY,
		},
		Description: &api.NodeDescription{
			Platform: &api.Platform{
				Architecture: "x86_64",
				OS:           "linux",
			},
		},
	}

	node2 := &api.Node{
		ID: "node2",
		Spec: api.NodeSpec{
			Annotations: api.Annotations{
				Name: "node2",
			},
		},
		Status: api.NodeStatus{
			State: api.NodeStatus_READY,
		},
		Description: &api.NodeDescription{
			Platform: &api.Platform{
				Architecture: "amd64",
				OS:           "windows",
			},
		},
	}

	// node with nil platform description, cannot schedule anything
	// with a platform constraint
	node3 := &api.Node{
		ID: "node3",
		Spec: api.NodeSpec{
			Annotations: api.Annotations{
				Name: "node3",
			},
		},
		Status: api.NodeStatus{
			State: api.NodeStatus_READY,
		},
		Description: &api.NodeDescription{},
	}

	service1 := &api.Service{
		ID: "serviceID1",
	}
	s := store.NewMemoryStore(nil)
	assert.NotNil(t, s)
	defer s.Close()

	err := s.Update(func(tx store.Tx) error {
		// Add initial task, service and nodes to the store
		assert.NoError(t, store.CreateService(tx, service1))
		assert.NoError(t, store.CreateTask(tx, task1))
		assert.NoError(t, store.CreateNode(tx, node1))
		assert.NoError(t, store.CreateNode(tx, node2))
		assert.NoError(t, store.CreateNode(tx, node3))
		return nil
	})
	assert.NoError(t, err)

	scheduler := New(s)

	watch, cancel := state.Watch(s.WatchQueue(), api.EventUpdateTask{})
	defer cancel()

	go func() {
		assert.NoError(t, scheduler.Run(ctx))
	}()
	defer scheduler.Stop()

	// task1 should get assigned
	assignment1 := watchAssignment(t, watch)
	assert.Equal(t, "node1", assignment1.NodeID)

	// add task2
	err = s.Update(func(tx store.Tx) error {
		assert.NoError(t, store.CreateTask(tx, task2))
		return nil
	})
	assert.NoError(t, err)
	failure := watchAssignmentFailure(t, watch)
	assert.Equal(t, "no suitable node (unsupported platform on 3 nodes)", failure.Status.Err)

	// add task3
	err = s.Update(func(tx store.Tx) error {
		assert.NoError(t, store.CreateTask(tx, task3))
		return nil
	})
	assert.NoError(t, err)
	assignment2 := watchAssignment(t, watch)
	assert.Regexp(t, assignment2.NodeID, "(node2|node3)")

	// add task4
	err = s.Update(func(tx store.Tx) error {
		assert.NoError(t, store.CreateTask(tx, task4))
		return nil
	})
	assert.NoError(t, err)
	assignment3 := watchAssignment(t, watch)
	assert.Equal(t, "node1", assignment3.NodeID)

	// add task5
	err = s.Update(func(tx store.Tx) error {
		assert.NoError(t, store.CreateTask(tx, task5))
		return nil
	})
	assert.NoError(t, err)
	assignment4 := watchAssignment(t, watch)
	assert.Regexp(t, assignment4.NodeID, "(node1|node2)")
}

// TestSchedulerUnassignedMap tests that unassigned tasks are deleted from unassignedTasks when the service is removed
func TestSchedulerUnassignedMap(t *testing.T) {
	ctx := context.Background()
	// create a service and a task with OS constraint that is not met
	task1 := &api.Task{
		ID:           "id1",
		ServiceID:    "serviceID1",
		DesiredState: api.TaskStateRunning,
		ServiceAnnotations: api.Annotations{
			Name: "name1",
		},
		Status: api.TaskStatus{
			State: api.TaskStatePending,
		},
		Spec: api.TaskSpec{
			Placement: &api.Placement{
				Platforms: []*api.Platform{
					{
						Architecture: "amd64",
						OS:           "windows",
					},
				},
			},
		},
	}

	node1 := &api.Node{
		ID: "node1",
		Spec: api.NodeSpec{
			Annotations: api.Annotations{
				Name: "node1",
			},
		},
		Status: api.NodeStatus{
			State: api.NodeStatus_READY,
		},
		Description: &api.NodeDescription{
			Platform: &api.Platform{
				Architecture: "x86_64",
				OS:           "linux",
			},
		},
	}

	service1 := &api.Service{
		ID: "serviceID1",
	}

	s := store.NewMemoryStore(nil)
	assert.NotNil(t, s)
	defer s.Close()

	err := s.Update(func(tx store.Tx) error {
		// Add initial task, service and nodes to the store
		assert.NoError(t, store.CreateService(tx, service1))
		assert.NoError(t, store.CreateTask(tx, task1))
		assert.NoError(t, store.CreateNode(tx, node1))
		return nil
	})
	assert.NoError(t, err)

	scheduler := New(s)
	scheduler.unassignedTasks["id1"] = task1

	scheduler.tick(ctx)
	// task1 is in the unassigned map
	assert.Contains(t, scheduler.unassignedTasks, task1.ID)

	// delete the service of an unassigned task
	err = s.Update(func(tx store.Tx) error {
		assert.NoError(t, store.DeleteService(tx, service1.ID))
		return nil
	})
	assert.NoError(t, err)

	scheduler.tick(ctx)
	// task1 is removed from the unassigned map
	assert.NotContains(t, scheduler.unassignedTasks, task1.ID)
}

func TestPreassignedTasks(t *testing.T) {
	ctx := context.Background()
	initialNodeSet := []*api.Node{
		{
			ID: "node1",
			Spec: api.NodeSpec{
				Annotations: api.Annotations{
					Name: "name1",
				},
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
			},
			Status: api.NodeStatus{
				State: api.NodeStatus_READY,
			},
		},
	}

	initialTaskSet := []*api.Task{
		{
			ID:           "task1",
			DesiredState: api.TaskStateRunning,
			ServiceAnnotations: api.Annotations{
				Name: "name1",
			},

			Status: api.TaskStatus{
				State: api.TaskStatePending,
			},
		},
		{
			ID:           "task2",
			DesiredState: api.TaskStateRunning,
			ServiceAnnotations: api.Annotations{
				Name: "name2",
			},
			Status: api.TaskStatus{
				State: api.TaskStatePending,
			},
			NodeID: initialNodeSet[0].ID,
		},
		{
			ID:           "task3",
			DesiredState: api.TaskStateRunning,
			ServiceAnnotations: api.Annotations{
				Name: "name2",
			},
			Status: api.TaskStatus{
				State: api.TaskStatePending,
			},
			NodeID: initialNodeSet[0].ID,
		},
	}

	s := store.NewMemoryStore(nil)
	assert.NotNil(t, s)
	defer s.Close()

	err := s.Update(func(tx store.Tx) error {
		// Prepopulate nodes
		for _, n := range initialNodeSet {
			assert.NoError(t, store.CreateNode(tx, n))
		}

		// Prepopulate tasks
		for _, task := range initialTaskSet {
			assert.NoError(t, store.CreateTask(tx, task))
		}
		return nil
	})
	assert.NoError(t, err)

	scheduler := New(s)

	watch, cancel := state.Watch(s.WatchQueue(), api.EventUpdateTask{})
	defer cancel()

	go func() {
		assert.NoError(t, scheduler.Run(ctx))
	}()

	//preassigned tasks would be processed first
	assignment1 := watchAssignment(t, watch)
	// task2 and task3 are preassigned to node1
	assert.Equal(t, assignment1.NodeID, "node1")
	assert.Regexp(t, assignment1.ID, "(task2|task3)")

	assignment2 := watchAssignment(t, watch)
	if assignment1.ID == "task2" {
		assert.Equal(t, "task3", assignment2.ID)
	} else {
		assert.Equal(t, "task2", assignment2.ID)
	}

	// task1 would be assigned to node2 because node1 has 2 tasks already
	assignment3 := watchAssignment(t, watch)
	assert.Equal(t, assignment3.ID, "task1")
	assert.Equal(t, assignment3.NodeID, "node2")
}

func TestIgnoreTasks(t *testing.T) {
	ctx := context.Background()
	initialNodeSet := []*api.Node{
		{
			ID: "node1",
			Spec: api.NodeSpec{
				Annotations: api.Annotations{
					Name: "name1",
				},
			},
			Status: api.NodeStatus{
				State: api.NodeStatus_READY,
			},
		},
	}

	// Tasks with desired state running, shutdown, remove.
	initialTaskSet := []*api.Task{
		{
			ID:           "task1",
			DesiredState: api.TaskStateRunning,
			ServiceAnnotations: api.Annotations{
				Name: "name1",
			},

			Status: api.TaskStatus{
				State: api.TaskStatePending,
			},
		},
		{
			ID:           "task2",
			DesiredState: api.TaskStateShutdown,
			ServiceAnnotations: api.Annotations{
				Name: "name2",
			},
			Status: api.TaskStatus{
				State: api.TaskStatePending,
			},
			NodeID: initialNodeSet[0].ID,
		},
		{
			ID:           "task3",
			DesiredState: api.TaskStateRemove,
			ServiceAnnotations: api.Annotations{
				Name: "name2",
			},
			Status: api.TaskStatus{
				State: api.TaskStatePending,
			},
			NodeID: initialNodeSet[0].ID,
		},
	}

	s := store.NewMemoryStore(nil)
	assert.NotNil(t, s)
	defer s.Close()

	err := s.Update(func(tx store.Tx) error {
		// Prepopulate nodes
		for _, n := range initialNodeSet {
			assert.NoError(t, store.CreateNode(tx, n))
		}

		// Prepopulate tasks
		for _, task := range initialTaskSet {
			assert.NoError(t, store.CreateTask(tx, task))
		}
		return nil
	})
	assert.NoError(t, err)

	scheduler := New(s)

	watch, cancel := state.Watch(s.WatchQueue(), api.EventUpdateTask{})
	defer cancel()

	go func() {
		assert.NoError(t, scheduler.Run(ctx))
	}()

	// task1 is the only task that gets assigned since other two tasks
	// are ignored by the scheduler.
	// Normally task2/task3 should get assigned first since its a preassigned task.
	assignment3 := watchAssignment(t, watch)
	assert.Equal(t, assignment3.ID, "task1")
	assert.Equal(t, assignment3.NodeID, "node1")
}

func watchAssignmentFailure(t *testing.T, watch chan events.Event) *api.Task {
	for {
		select {
		case event := <-watch:
			if task, ok := event.(api.EventUpdateTask); ok {
				if task.Task.Status.State < api.TaskStateAssigned {
					return task.Task
				}
			}
		case <-time.After(time.Second):
			t.Fatal("no task assignment failure")
		}
	}
}

func watchAssignment(t *testing.T, watch chan events.Event) *api.Task {
	for {
		select {
		case event := <-watch:
			if task, ok := event.(api.EventUpdateTask); ok {
				if task.Task.Status.State >= api.TaskStateAssigned &&
					task.Task.Status.State <= api.TaskStateRunning &&
					task.Task.NodeID != "" {
					return task.Task
				}
			}
		case <-time.After(time.Second):
			t.Fatal("no task assignment")
		}
	}
}

func TestSchedulerPluginConstraint(t *testing.T) {
	ctx := context.Background()

	// Node1: vol plugin1
	n1 := &api.Node{
		ID: "node1_ID",
		Spec: api.NodeSpec{
			Annotations: api.Annotations{
				Name: "node1",
			},
		},
		Description: &api.NodeDescription{
			Engine: &api.EngineDescription{
				Plugins: []api.PluginDescription{
					{
						Type: "Volume",
						Name: "plugin1",
					},
					{
						Type: "Log",
						Name: "default",
					},
				},
			},
		},
		Status: api.NodeStatus{
			State: api.NodeStatus_READY,
		},
	}

	// Node2: vol plugin1, vol plugin2
	n2 := &api.Node{
		ID: "node2_ID",
		Spec: api.NodeSpec{
			Annotations: api.Annotations{
				Name: "node2",
			},
		},
		Description: &api.NodeDescription{
			Engine: &api.EngineDescription{
				Plugins: []api.PluginDescription{
					{
						Type: "Volume",
						Name: "plugin1",
					},
					{
						Type: "Volume",
						Name: "plugin2",
					},
					{
						Type: "Log",
						Name: "default",
					},
				},
			},
		},
		Status: api.NodeStatus{
			State: api.NodeStatus_READY,
		},
	}

	// Node3: vol plugin1, network plugin1
	n3 := &api.Node{
		ID: "node3_ID",
		Spec: api.NodeSpec{
			Annotations: api.Annotations{
				Name: "node3",
			},
		},
		Description: &api.NodeDescription{
			Engine: &api.EngineDescription{
				Plugins: []api.PluginDescription{
					{
						Type: "Volume",
						Name: "plugin1",
					},
					{
						Type: "Network",
						Name: "plugin1",
					},
					{
						Type: "Log",
						Name: "default",
					},
				},
			},
		},
		Status: api.NodeStatus{
			State: api.NodeStatus_READY,
		},
	}

	// Node4: log plugin1
	n4 := &api.Node{
		ID: "node4_ID",
		Spec: api.NodeSpec{
			Annotations: api.Annotations{
				Name: "node4",
			},
		},
		Description: &api.NodeDescription{
			Engine: &api.EngineDescription{
				Plugins: []api.PluginDescription{
					{
						Type: "Log",
						Name: "plugin1",
					},
				},
			},
		},
		Status: api.NodeStatus{
			State: api.NodeStatus_READY,
		},
	}

	volumeOptionsDriver := func(driver string) *api.Mount_VolumeOptions {
		return &api.Mount_VolumeOptions{
			DriverConfig: &api.Driver{
				Name: driver,
			},
		}
	}

	// Task0: bind mount
	t0 := &api.Task{
		ID:           "task0_ID",
		ServiceID:    "serviceID1",
		DesiredState: api.TaskStateRunning,
		Spec: api.TaskSpec{
			Runtime: &api.TaskSpec_Container{
				Container: &api.ContainerSpec{
					Mounts: []api.Mount{
						{
							Source: "/src",
							Target: "/foo",
							Type:   api.MountTypeBind,
						},
					},
				},
			},
		},
		ServiceAnnotations: api.Annotations{
			Name: "task0",
		},
		Status: api.TaskStatus{
			State: api.TaskStatePending,
		},
	}

	// Task1: vol plugin1
	t1 := &api.Task{
		ID:           "task1_ID",
		ServiceID:    "serviceID1",
		DesiredState: api.TaskStateRunning,
		Spec: api.TaskSpec{
			Runtime: &api.TaskSpec_Container{
				Container: &api.ContainerSpec{
					Mounts: []api.Mount{
						{
							Source:        "testVol1",
							Target:        "/foo",
							Type:          api.MountTypeVolume,
							VolumeOptions: volumeOptionsDriver("plugin1"),
						},
					},
				},
			},
		},
		ServiceAnnotations: api.Annotations{
			Name: "task1",
		},
		Status: api.TaskStatus{
			State: api.TaskStatePending,
		},
	}

	// Task2: vol plugin1, vol plugin2
	t2 := &api.Task{
		ID:           "task2_ID",
		ServiceID:    "serviceID1",
		DesiredState: api.TaskStateRunning,
		Spec: api.TaskSpec{
			Runtime: &api.TaskSpec_Container{
				Container: &api.ContainerSpec{
					Mounts: []api.Mount{
						{
							Source:        "testVol1",
							Target:        "/foo",
							Type:          api.MountTypeVolume,
							VolumeOptions: volumeOptionsDriver("plugin1"),
						},
						{
							Source:        "testVol2",
							Target:        "/foo",
							Type:          api.MountTypeVolume,
							VolumeOptions: volumeOptionsDriver("plugin2"),
						},
					},
				},
			},
		},
		ServiceAnnotations: api.Annotations{
			Name: "task2",
		},
		Status: api.TaskStatus{
			State: api.TaskStatePending,
		},
	}

	// Task3: vol plugin1, network plugin1
	t3 := &api.Task{
		ID:           "task3_ID",
		ServiceID:    "serviceID1",
		DesiredState: api.TaskStateRunning,
		Networks: []*api.NetworkAttachment{
			{
				Network: &api.Network{
					ID: "testNwID1",
					Spec: api.NetworkSpec{
						Annotations: api.Annotations{
							Name: "testVol1",
						},
					},
					DriverState: &api.Driver{
						Name: "plugin1",
					},
				},
			},
		},
		Spec: api.TaskSpec{
			Runtime: &api.TaskSpec_Container{
				Container: &api.ContainerSpec{
					Mounts: []api.Mount{
						{
							Source:        "testVol1",
							Target:        "/foo",
							Type:          api.MountTypeVolume,
							VolumeOptions: volumeOptionsDriver("plugin1"),
						},
					},
				},
			},
		},
		ServiceAnnotations: api.Annotations{
			Name: "task2",
		},
		Status: api.TaskStatus{
			State: api.TaskStatePending,
		},
	}
	// Task4: log plugin1
	t4 := &api.Task{
		ID:           "task4_ID",
		ServiceID:    "serviceID1",
		DesiredState: api.TaskStateRunning,
		Spec: api.TaskSpec{
			Runtime: &api.TaskSpec_Container{
				Container: &api.ContainerSpec{},
			},
			LogDriver: &api.Driver{Name: "plugin1"},
		},
		ServiceAnnotations: api.Annotations{
			Name: "task4",
		},
		Status: api.TaskStatus{
			State: api.TaskStatePending,
		},
	}
	// Task5: log plugin1
	t5 := &api.Task{
		ID:           "task5_ID",
		ServiceID:    "serviceID1",
		DesiredState: api.TaskStateRunning,
		Spec: api.TaskSpec{
			Runtime: &api.TaskSpec_Container{
				Container: &api.ContainerSpec{},
			},
			LogDriver: &api.Driver{Name: "plugin1"},
		},
		ServiceAnnotations: api.Annotations{
			Name: "task5",
		},
		Status: api.TaskStatus{
			State: api.TaskStatePending,
		},
	}

	// no logging
	t6 := &api.Task{
		ID:           "task6_ID",
		ServiceID:    "serviceID1",
		DesiredState: api.TaskStateRunning,
		Spec: api.TaskSpec{
			Runtime: &api.TaskSpec_Container{
				Container: &api.ContainerSpec{},
			},
			LogDriver: &api.Driver{Name: "none"},
		},
		ServiceAnnotations: api.Annotations{
			Name: "task6",
		},
		Status: api.TaskStatus{
			State: api.TaskStatePending,
		},
	}

	// log driver with no name
	t7 := &api.Task{
		ID:           "task7_ID",
		ServiceID:    "serviceID1",
		DesiredState: api.TaskStateRunning,
		Spec: api.TaskSpec{
			Runtime: &api.TaskSpec_Container{
				Container: &api.ContainerSpec{},
			},
			LogDriver: &api.Driver{
				Options: map[string]string{
					"max-size": "50k",
				},
			},
		},
		ServiceAnnotations: api.Annotations{
			Name: "task7",
		},
		Status: api.TaskStatus{
			State: api.TaskStatePending,
		},
	}

	s1 := &api.Service{
		ID: "serviceID1",
	}
	s := store.NewMemoryStore(nil)
	assert.NotNil(t, s)
	defer s.Close()

	// Add initial node, service and task
	err := s.Update(func(tx store.Tx) error {
		assert.NoError(t, store.CreateService(tx, s1))
		assert.NoError(t, store.CreateTask(tx, t1))
		assert.NoError(t, store.CreateNode(tx, n1))
		return nil
	})
	assert.NoError(t, err)

	scheduler := New(s)

	watch, cancel := state.Watch(s.WatchQueue(), api.EventUpdateTask{})
	defer cancel()

	go func() {
		assert.NoError(t, scheduler.Run(ctx))
	}()
	defer scheduler.Stop()

	// t1 should get assigned
	assignment := watchAssignment(t, watch)
	assert.Equal(t, assignment.NodeID, "node1_ID")

	// Create t0; it should get assigned because the plugin filter shouldn't
	// be enabled for tasks that have bind mounts
	err = s.Update(func(tx store.Tx) error {
		assert.NoError(t, store.CreateTask(tx, t0))
		return nil
	})
	assert.NoError(t, err)

	assignment0 := watchAssignment(t, watch)
	assert.Equal(t, assignment0.ID, "task0_ID")
	assert.Equal(t, assignment0.NodeID, "node1_ID")

	// Create t2; it should stay in the pending state because there is
	// no node that with volume plugin `plugin2`
	err = s.Update(func(tx store.Tx) error {
		assert.NoError(t, store.CreateTask(tx, t2))
		return nil
	})
	assert.NoError(t, err)

	failure := watchAssignmentFailure(t, watch)
	assert.Equal(t, "no suitable node (missing plugin on 1 node)", failure.Status.Err)

	// Now add the second node
	err = s.Update(func(tx store.Tx) error {
		assert.NoError(t, store.CreateNode(tx, n2))
		return nil
	})
	assert.NoError(t, err)

	// Check that t2 has been assigned
	assignment1 := watchAssignment(t, watch)
	assert.Equal(t, assignment1.ID, "task2_ID")
	assert.Equal(t, assignment1.NodeID, "node2_ID")

	// Create t3; it should stay in the pending state because there is
	// no node that with network plugin `plugin1`
	err = s.Update(func(tx store.Tx) error {
		assert.NoError(t, store.CreateTask(tx, t3))
		return nil
	})
	assert.NoError(t, err)

	failure = watchAssignmentFailure(t, watch)
	assert.Equal(t, "no suitable node (missing plugin on 2 nodes)", failure.Status.Err)

	// Now add the node3
	err = s.Update(func(tx store.Tx) error {
		assert.NoError(t, store.CreateNode(tx, n3))
		return nil
	})
	assert.NoError(t, err)

	// Check that t3 has been assigned
	assignment2 := watchAssignment(t, watch)
	assert.Equal(t, assignment2.ID, "task3_ID")
	assert.Equal(t, assignment2.NodeID, "node3_ID")

	// Create t4; it should stay in the pending state because there is
	// no node that with log plugin `plugin1`
	err = s.Update(func(tx store.Tx) error {
		assert.NoError(t, store.CreateTask(tx, t4))
		return nil
	})
	assert.NoError(t, err)

	// check that t4 has been assigned
	failure2 := watchAssignmentFailure(t, watch)
	assert.Equal(t, "no suitable node (missing plugin on 3 nodes)", failure2.Status.Err)

	err = s.Update(func(tx store.Tx) error {
		assert.NoError(t, store.CreateNode(tx, n4))
		return nil
	})
	assert.NoError(t, err)

	// Check that t4 has been assigned
	assignment3 := watchAssignment(t, watch)
	assert.Equal(t, assignment3.ID, "task4_ID")
	assert.Equal(t, assignment3.NodeID, "node4_ID")

	err = s.Update(func(tx store.Tx) error {
		assert.NoError(t, store.CreateTask(tx, t5))
		return nil
	})
	assert.NoError(t, err)
	assignment4 := watchAssignment(t, watch)
	assert.Equal(t, assignment4.ID, "task5_ID")
	assert.Equal(t, assignment4.NodeID, "node4_ID")

	// check that t6 gets assigned to some node
	err = s.Update(func(tx store.Tx) error {
		assert.NoError(t, store.CreateTask(tx, t6))
		return nil
	})
	assert.NoError(t, err)
	assignment5 := watchAssignment(t, watch)
	assert.Equal(t, assignment5.ID, "task6_ID")
	assert.NotEqual(t, assignment5.NodeID, "")

	// check that t7 gets assigned to some node
	err = s.Update(func(tx store.Tx) error {
		assert.NoError(t, store.CreateTask(tx, t7))
		return nil
	})
	assert.NoError(t, err)
	assignment6 := watchAssignment(t, watch)
	assert.Equal(t, assignment6.ID, "task7_ID")
	assert.NotEqual(t, assignment6.NodeID, "")
}

func BenchmarkScheduler1kNodes1kTasks(b *testing.B) {
	benchScheduler(b, 1e3, 1e3, false)
}

func BenchmarkScheduler1kNodes10kTasks(b *testing.B) {
	benchScheduler(b, 1e3, 1e4, false)
}

func BenchmarkScheduler1kNodes100kTasks(b *testing.B) {
	benchScheduler(b, 1e3, 1e5, false)
}

func BenchmarkScheduler100kNodes100kTasks(b *testing.B) {
	benchScheduler(b, 1e5, 1e5, false)
}

func BenchmarkScheduler100kNodes1kTasks(b *testing.B) {
	benchScheduler(b, 1e5, 1e3, false)
}

func BenchmarkScheduler100kNodes1MTasks(b *testing.B) {
	benchScheduler(b, 1e5, 1e6, false)
}

func BenchmarkSchedulerConstraints1kNodes1kTasks(b *testing.B) {
	benchScheduler(b, 1e3, 1e3, true)
}

func BenchmarkSchedulerConstraints1kNodes10kTasks(b *testing.B) {
	benchScheduler(b, 1e3, 1e4, true)
}

func BenchmarkSchedulerConstraints1kNodes100kTasks(b *testing.B) {
	benchScheduler(b, 1e3, 1e5, true)
}

func BenchmarkSchedulerConstraints5kNodes100kTasks(b *testing.B) {
	benchScheduler(b, 5e3, 1e5, true)
}

func benchScheduler(b *testing.B, nodes, tasks int, networkConstraints bool) {
	ctx := context.Background()

	for iters := 0; iters < b.N; iters++ {
		b.StopTimer()
		s := store.NewMemoryStore(nil)
		scheduler := New(s)

		watch, cancel := state.Watch(s.WatchQueue(), api.EventUpdateTask{})

		go func() {
			_ = scheduler.Run(ctx)
		}()

		// Let the scheduler get started
		runtime.Gosched()

		_ = s.Update(func(tx store.Tx) error {
			// Create initial nodes and tasks
			for i := 0; i < nodes; i++ {
				n := &api.Node{
					ID: identity.NewID(),
					Spec: api.NodeSpec{
						Annotations: api.Annotations{
							Name:   "name" + strconv.Itoa(i),
							Labels: make(map[string]string),
						},
					},
					Status: api.NodeStatus{
						State: api.NodeStatus_READY,
					},
					Description: &api.NodeDescription{
						Engine: &api.EngineDescription{},
					},
				}
				// Give every third node a special network
				if i%3 == 0 {
					n.Description.Engine.Plugins = []api.PluginDescription{
						{
							Name: "network",
							Type: "Network",
						},
					}

				}
				err := store.CreateNode(tx, n)
				if err != nil {
					panic(err)
				}
			}
			for i := 0; i < tasks; i++ {
				id := "task" + strconv.Itoa(i)
				t := &api.Task{
					ID:           id,
					DesiredState: api.TaskStateRunning,
					ServiceAnnotations: api.Annotations{
						Name: id,
					},
					Status: api.TaskStatus{
						State: api.TaskStatePending,
					},
				}
				if networkConstraints {
					t.Networks = []*api.NetworkAttachment{
						{
							Network: &api.Network{
								DriverState: &api.Driver{
									Name: "network",
								},
							},
						},
					}
				}
				err := store.CreateTask(tx, t)
				if err != nil {
					panic(err)
				}
			}
			b.StartTimer()
			return nil
		})

		for i := 0; i != tasks; i++ {
			<-watch
		}

		scheduler.Stop()
		cancel()
		s.Close()
	}
}

func TestSchedulerHostPort(t *testing.T) {
	ctx := context.Background()
	node1 := &api.Node{
		ID: "nodeid1",
		Spec: api.NodeSpec{
			Annotations: api.Annotations{
				Name: "node1",
			},
		},
		Status: api.NodeStatus{
			State: api.NodeStatus_READY,
		},
	}
	node2 := &api.Node{
		ID: "nodeid2",
		Spec: api.NodeSpec{
			Annotations: api.Annotations{
				Name: "node2",
			},
		},
		Status: api.NodeStatus{
			State: api.NodeStatus_READY,
		},
	}

	task1 := &api.Task{
		ID:           "id1",
		ServiceID:    "serviceID1",
		DesiredState: api.TaskStateRunning,
		Spec: api.TaskSpec{
			Runtime: &api.TaskSpec_Container{
				Container: &api.ContainerSpec{},
			},
		},
		ServiceAnnotations: api.Annotations{
			Name: "name1",
		},
		Status: api.TaskStatus{
			State: api.TaskStatePending,
		},
		Endpoint: &api.Endpoint{
			Ports: []*api.PortConfig{
				{
					PublishMode:   api.PublishModeHost,
					PublishedPort: 58,
					Protocol:      api.ProtocolTCP,
				},
			},
		},
	}
	task2 := &api.Task{
		ID:           "id2",
		ServiceID:    "serviceID1",
		DesiredState: api.TaskStateRunning,
		Spec: api.TaskSpec{
			Runtime: &api.TaskSpec_Container{
				Container: &api.ContainerSpec{},
			},
		},
		ServiceAnnotations: api.Annotations{
			Name: "name2",
		},
		Status: api.TaskStatus{
			State: api.TaskStatePending,
		},
		Endpoint: &api.Endpoint{
			Ports: []*api.PortConfig{
				{
					PublishMode:   api.PublishModeHost,
					PublishedPort: 58,
					Protocol:      api.ProtocolUDP,
				},
			},
		},
	}
	task3 := &api.Task{
		ID:           "id3",
		ServiceID:    "serviceID1",
		DesiredState: api.TaskStateRunning,
		Spec: api.TaskSpec{
			Runtime: &api.TaskSpec_Container{
				Container: &api.ContainerSpec{},
			},
		},
		ServiceAnnotations: api.Annotations{
			Name: "name3",
		},
		Status: api.TaskStatus{
			State: api.TaskStatePending,
		},
		Endpoint: &api.Endpoint{
			Ports: []*api.PortConfig{
				{
					PublishMode:   api.PublishModeHost,
					PublishedPort: 58,
					Protocol:      api.ProtocolUDP,
				},
				{
					PublishMode:   api.PublishModeHost,
					PublishedPort: 58,
					Protocol:      api.ProtocolTCP,
				},
			},
		},
	}

	service1 := &api.Service{
		ID: "serviceID1",
	}

	s := store.NewMemoryStore(nil)
	assert.NotNil(t, s)
	defer s.Close()

	err := s.Update(func(tx store.Tx) error {
		// Add initial node, service and task
		assert.NoError(t, store.CreateService(tx, service1))
		assert.NoError(t, store.CreateTask(tx, task1))
		assert.NoError(t, store.CreateTask(tx, task2))
		return nil
	})
	assert.NoError(t, err)

	scheduler := New(s)

	watch, cancel := state.Watch(s.WatchQueue(), api.EventUpdateTask{})
	defer cancel()

	go func() {
		assert.NoError(t, scheduler.Run(ctx))
	}()
	defer scheduler.Stop()

	// Tasks shouldn't be scheduled because there are no nodes.
	watchAssignmentFailure(t, watch)
	watchAssignmentFailure(t, watch)

	err = s.Update(func(tx store.Tx) error {
		// Add initial node and task
		assert.NoError(t, store.CreateNode(tx, node1))
		assert.NoError(t, store.CreateNode(tx, node2))
		return nil
	})
	assert.NoError(t, err)

	// Tasks 1 and 2 should be assigned to different nodes.
	assignment1 := watchAssignment(t, watch)
	assignment2 := watchAssignment(t, watch)
	assert.True(t, assignment1 != assignment2)

	// Task 3 should not be schedulable.
	err = s.Update(func(tx store.Tx) error {
		assert.NoError(t, store.CreateTask(tx, task3))
		return nil
	})
	assert.NoError(t, err)

	failure := watchAssignmentFailure(t, watch)
	assert.Equal(t, "no suitable node (host-mode port already in use on 2 nodes)", failure.Status.Err)
}
