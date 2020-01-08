package orchestrator

import (
	google_protobuf "github.com/gogo/protobuf/types"
	"github.com/stretchr/testify/assert"
	"sort"
	"strconv"
	"testing"

	"github.com/docker/swarmkit/api"
)

// Test IsTaskDirty() for placement constraints.
func TestIsTaskDirty(t *testing.T) {
	service := &api.Service{
		ID:          "id1",
		SpecVersion: &api.Version{Index: 1},
		Spec: api.ServiceSpec{
			Annotations: api.Annotations{
				Name: "name1",
			},
			Task: api.TaskSpec{
				Runtime: &api.TaskSpec_Container{
					Container: &api.ContainerSpec{
						Image: "v:1",
					},
				},
			},
		},
	}

	task := &api.Task{
		ID: "task1",
		Spec: api.TaskSpec{
			Runtime: &api.TaskSpec_Container{
				Container: &api.ContainerSpec{
					Image: "v:1",
				},
			},
		},
	}

	node := &api.Node{
		ID: "node1",
	}

	assert.False(t, IsTaskDirty(service, task, node))

	// Update only placement constraints.
	service.SpecVersion.Index++
	service.Spec.Task.Placement = &api.Placement{}
	service.Spec.Task.Placement.Constraints = append(service.Spec.Task.Placement.Constraints, "node=node1")
	assert.False(t, IsTaskDirty(service, task, node))

	// Update only placement constraints again.
	service.SpecVersion.Index++
	service.Spec.Task.Placement = &api.Placement{}
	service.Spec.Task.Placement.Constraints = append(service.Spec.Task.Placement.Constraints, "node!=node1")
	assert.True(t, IsTaskDirty(service, task, node))

	// Update only placement constraints
	service.SpecVersion.Index++
	service.Spec.Task.Placement = &api.Placement{}
	service.Spec.Task.GetContainer().Image = "v:2"
	assert.True(t, IsTaskDirty(service, task, node))
}

func TestIsTaskDirtyPlacementConstraintsOnly(t *testing.T) {
	service := &api.Service{
		ID: "id1",
		Spec: api.ServiceSpec{
			Annotations: api.Annotations{
				Name: "name1",
			},
			Task: api.TaskSpec{
				Runtime: &api.TaskSpec_Container{
					Container: &api.ContainerSpec{
						Image: "v:1",
					},
				},
			},
		},
	}

	task := &api.Task{
		ID: "task1",
		Spec: api.TaskSpec{
			Runtime: &api.TaskSpec_Container{
				Container: &api.ContainerSpec{
					Image: "v:1",
				},
			},
		},
	}

	assert.False(t, IsTaskDirtyPlacementConstraintsOnly(service.Spec.Task, task))

	// Update only placement constraints.
	service.Spec.Task.Placement = &api.Placement{}
	service.Spec.Task.Placement.Constraints = append(service.Spec.Task.Placement.Constraints, "node==*")
	assert.True(t, IsTaskDirtyPlacementConstraintsOnly(service.Spec.Task, task))

	// Update something else in the task spec.
	service.Spec.Task.GetContainer().Image = "v:2"
	assert.False(t, IsTaskDirtyPlacementConstraintsOnly(service.Spec.Task, task))

	// Clear out placement constraints.
	service.Spec.Task.Placement.Constraints = nil
	assert.False(t, IsTaskDirtyPlacementConstraintsOnly(service.Spec.Task, task))
}

// Test Task sorting, which is currently based on
// Status.AppliedAt, and then on Status.Timestamp.
func TestTaskSort(t *testing.T) {
	var tasks []*api.Task
	size := 5
	seconds := int64(size)
	for i := 0; i < size; i++ {
		task := &api.Task{
			ID: "id_" + strconv.Itoa(i),
			Status: api.TaskStatus{
				Timestamp: &google_protobuf.Timestamp{Seconds: seconds},
			},
		}

		seconds--
		tasks = append(tasks, task)
	}

	sort.Sort(TasksByTimestamp(tasks))
	for i, task := range tasks {
		expected := &google_protobuf.Timestamp{Seconds: int64(i + 1)}
		assert.Equal(t, expected, task.Status.Timestamp)
		assert.Equal(t, "id_"+strconv.Itoa(size-(i+1)), task.ID)
	}

	for i, task := range tasks {
		task.Status.AppliedAt = &google_protobuf.Timestamp{Seconds: int64(size - i)}
	}

	sort.Sort(TasksByTimestamp(tasks))
	sort.Sort(TasksByTimestamp(tasks))
	for i, task := range tasks {
		expected := &google_protobuf.Timestamp{Seconds: int64(i + 1)}
		assert.Equal(t, expected, task.Status.AppliedAt)
		assert.Equal(t, "id_"+strconv.Itoa(i), task.ID)
	}
}
