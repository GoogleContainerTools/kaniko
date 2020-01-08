package controlapi

import (
	"strings"
	"testing"

	"golang.org/x/net/context"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"

	"github.com/docker/swarmkit/api"
	"github.com/docker/swarmkit/identity"
	"github.com/docker/swarmkit/manager/state/store"
	"github.com/stretchr/testify/assert"
)

func createTask(t *testing.T, ts *testServer, desiredState api.TaskState) *api.Task {
	task := &api.Task{
		ID:           identity.NewID(),
		DesiredState: desiredState,
		Spec: api.TaskSpec{
			Runtime: &api.TaskSpec_Container{
				Container: &api.ContainerSpec{},
			},
		},
	}
	err := ts.Store.Update(func(tx store.Tx) error {
		return store.CreateTask(tx, task)
	})
	assert.NoError(t, err)
	return task
}

func TestGetTask(t *testing.T) {
	ts := newTestServer(t)
	defer ts.Stop()

	_, err := ts.Client.GetTask(context.Background(), &api.GetTaskRequest{})
	assert.Error(t, err)
	assert.Equal(t, codes.InvalidArgument, grpc.Code(err))

	_, err = ts.Client.GetTask(context.Background(), &api.GetTaskRequest{TaskID: "invalid"})
	assert.Error(t, err)
	assert.Equal(t, codes.NotFound, grpc.Code(err))

	task := createTask(t, ts, api.TaskStateRunning)
	r, err := ts.Client.GetTask(context.Background(), &api.GetTaskRequest{TaskID: task.ID})
	assert.NoError(t, err)
	assert.Equal(t, task.ID, r.Task.ID)
}

func TestRemoveTask(t *testing.T) {
	// TODO
}

func TestListTasks(t *testing.T) {
	ts := newTestServer(t)
	defer ts.Stop()
	r, err := ts.Client.ListTasks(context.Background(), &api.ListTasksRequest{})
	assert.NoError(t, err)
	assert.Empty(t, r.Tasks)

	t1 := createTask(t, ts, api.TaskStateRunning)
	r, err = ts.Client.ListTasks(context.Background(), &api.ListTasksRequest{})
	assert.NoError(t, err)
	assert.Equal(t, 1, len(r.Tasks))

	createTask(t, ts, api.TaskStateRunning)
	createTask(t, ts, api.TaskStateShutdown)
	r, err = ts.Client.ListTasks(context.Background(), &api.ListTasksRequest{})
	assert.NoError(t, err)
	assert.Equal(t, 3, len(r.Tasks))

	// List with an ID prefix.
	r, err = ts.Client.ListTasks(context.Background(), &api.ListTasksRequest{
		Filters: &api.ListTasksRequest_Filters{
			IDPrefixes: []string{t1.ID[0:4]},
		},
	})
	assert.NoError(t, err)
	assert.NotEmpty(t, r.Tasks)
	for _, task := range r.Tasks {
		assert.True(t, strings.HasPrefix(task.ID, t1.ID[0:4]))
	}

	// List by desired state.
	r, err = ts.Client.ListTasks(context.Background(),
		&api.ListTasksRequest{
			Filters: &api.ListTasksRequest_Filters{
				DesiredStates: []api.TaskState{api.TaskStateRunning},
			},
		},
	)
	assert.NoError(t, err)
	assert.Equal(t, 2, len(r.Tasks))
	r, err = ts.Client.ListTasks(context.Background(),
		&api.ListTasksRequest{
			Filters: &api.ListTasksRequest_Filters{
				DesiredStates: []api.TaskState{api.TaskStateShutdown},
			},
		},
	)
	assert.NoError(t, err)
	assert.Equal(t, 1, len(r.Tasks))
	r, err = ts.Client.ListTasks(context.Background(),
		&api.ListTasksRequest{
			Filters: &api.ListTasksRequest_Filters{
				DesiredStates: []api.TaskState{api.TaskStateRunning, api.TaskStateShutdown},
			},
		},
	)
	assert.NoError(t, err)
	assert.Equal(t, 3, len(r.Tasks))
}
