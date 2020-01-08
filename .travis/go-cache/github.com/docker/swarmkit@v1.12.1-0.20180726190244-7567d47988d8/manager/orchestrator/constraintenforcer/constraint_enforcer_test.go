package constraintenforcer

import (
	"testing"

	"github.com/docker/swarmkit/api"
	"github.com/docker/swarmkit/manager/orchestrator/testutils"
	"github.com/docker/swarmkit/manager/state"
	"github.com/docker/swarmkit/manager/state/store"
	"github.com/stretchr/testify/assert"
)

func TestConstraintEnforcer(t *testing.T) {
	nodes := []*api.Node{
		{
			ID: "id1",
			Spec: api.NodeSpec{
				Annotations: api.Annotations{
					Name: "name1",
				},
				Availability: api.NodeAvailabilityActive,
			},
			Status: api.NodeStatus{
				State: api.NodeStatus_READY,
			},
			Role: api.NodeRoleWorker,
		},
		{
			ID: "id2",
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
				Resources: &api.Resources{
					NanoCPUs:    1e9,
					MemoryBytes: 1e9,
				},
			},
		},
	}

	tasks := []*api.Task{
		{
			ID:           "id0",
			DesiredState: api.TaskStateRunning,
			Spec: api.TaskSpec{
				Placement: &api.Placement{
					Constraints: []string{"node.role == manager"},
				},
			},
			Status: api.TaskStatus{
				State: api.TaskStateNew,
			},
			NodeID: "id1",
		},
		{
			ID:           "id1",
			DesiredState: api.TaskStateRunning,
			Status: api.TaskStatus{
				State: api.TaskStateNew,
			},
			NodeID: "id1",
		},
		{
			ID:           "id2",
			DesiredState: api.TaskStateRunning,
			Spec: api.TaskSpec{
				Placement: &api.Placement{
					Constraints: []string{"node.role == worker"},
				},
			},
			Status: api.TaskStatus{
				State: api.TaskStateRunning,
			},
			NodeID: "id1",
		},
		{
			ID:           "id3",
			DesiredState: api.TaskStateNew,
			Status: api.TaskStatus{
				State: api.TaskStateNew,
			},
			NodeID: "id2",
		},
		{
			ID:           "id4",
			DesiredState: api.TaskStateReady,
			Spec: api.TaskSpec{
				Resources: &api.ResourceRequirements{
					Reservations: &api.Resources{
						MemoryBytes: 9e8,
					},
				},
			},
			Status: api.TaskStatus{
				State: api.TaskStatePending,
			},
			NodeID: "id2",
		},
	}

	s := store.NewMemoryStore(nil)
	assert.NotNil(t, s)
	defer s.Close()

	err := s.Update(func(tx store.Tx) error {
		// Prepoulate nodes
		for _, n := range nodes {
			assert.NoError(t, store.CreateNode(tx, n))
		}

		// Prepopulate tasks
		for _, task := range tasks {
			assert.NoError(t, store.CreateTask(tx, task))
		}
		return nil
	})
	assert.NoError(t, err)

	watch, cancel := state.Watch(s.WatchQueue(), api.EventUpdateTask{})
	defer cancel()

	constraintEnforcer := New(s)
	defer constraintEnforcer.Stop()

	go constraintEnforcer.Run()

	// id0 should be rejected immediately
	shutdown1 := testutils.WatchTaskUpdate(t, watch)
	assert.Equal(t, "id0", shutdown1.ID)
	assert.Equal(t, api.TaskStateRejected, shutdown1.Status.State)

	// Change node id1 to a manager
	err = s.Update(func(tx store.Tx) error {
		node := store.GetNode(tx, "id1")
		if node == nil {
			t.Fatal("could not get node id1")
		}
		node.Role = api.NodeRoleManager
		assert.NoError(t, store.UpdateNode(tx, node))
		return nil
	})
	assert.NoError(t, err)

	shutdown2 := testutils.WatchTaskUpdate(t, watch)
	assert.Equal(t, "id2", shutdown2.ID)
	assert.Equal(t, api.TaskStateRejected, shutdown2.Status.State)

	// Change resources on node id2
	err = s.Update(func(tx store.Tx) error {
		node := store.GetNode(tx, "id2")
		if node == nil {
			t.Fatal("could not get node id2")
		}
		node.Description.Resources.MemoryBytes = 5e8
		assert.NoError(t, store.UpdateNode(tx, node))
		return nil
	})
	assert.NoError(t, err)

	shutdown3 := testutils.WatchTaskUpdate(t, watch)
	assert.Equal(t, "id4", shutdown3.ID)
	assert.Equal(t, api.TaskStateRejected, shutdown3.Status.State)
}
