package agent

import (
	"io/ioutil"
	"math/rand"
	"os"
	"path/filepath"
	"sort"
	"testing"

	"github.com/boltdb/bolt"
	"github.com/docker/swarmkit/api"
	"github.com/docker/swarmkit/identity"
	"github.com/stretchr/testify/assert"
)

func TestStorageInit(t *testing.T) {
	db, cleanup := storageTestEnv(t)
	defer cleanup()

	assert.NoError(t, InitDB(db)) // ensure idempotence.
	assert.NoError(t, db.View(func(tx *bolt.Tx) error {
		bkt := tx.Bucket(bucketKeyStorageVersion)
		assert.NotNil(t, bkt)

		tbkt := bkt.Bucket([]byte("tasks"))
		assert.NotNil(t, tbkt)

		return nil
	}))
}

func TestStoragePutGet(t *testing.T) {
	db, cleanup := storageTestEnv(t)
	defer cleanup()

	tasks := genTasks(20)

	assert.NoError(t, db.Update(func(tx *bolt.Tx) error {
		for i, task := range tasks {
			assert.NoError(t, PutTask(tx, task))
			// remove status to make comparison work
			tasks[i].Status = api.TaskStatus{}
		}

		return nil
	}))

	assert.NoError(t, db.View(func(tx *bolt.Tx) error {
		for _, task := range tasks {
			retrieved, err := GetTask(tx, task.ID)
			assert.NoError(t, err)
			assert.Equal(t, task, retrieved)
		}

		return nil
	}))
}

func TestStoragePutGetStatusAssigned(t *testing.T) {
	db, cleanup := storageTestEnv(t)
	defer cleanup()

	tasks := genTasks(20)

	// set task, status and assignment for all tasks.
	assert.NoError(t, db.Update(func(tx *bolt.Tx) error {
		for _, task := range tasks {
			assert.NoError(t, PutTaskStatus(tx, task.ID, &task.Status))
			assert.NoError(t, PutTask(tx, task))
			assert.NoError(t, SetTaskAssignment(tx, task.ID, true))
		}

		return nil
	}))

	assert.NoError(t, db.View(func(tx *bolt.Tx) error {
		for _, task := range tasks {
			status, err := GetTaskStatus(tx, task.ID)
			assert.NoError(t, err)
			assert.Equal(t, &task.Status, status)

			retrieved, err := GetTask(tx, task.ID)
			assert.NoError(t, err)

			task.Status = api.TaskStatus{}
			assert.Equal(t, task, retrieved)

			assert.True(t, TaskAssigned(tx, task.ID))
		}

		return nil
	}))

	// set evens to unassigned and updates all states plus one
	assert.NoError(t, db.Update(func(tx *bolt.Tx) error {
		for i, task := range tasks {
			task.Status.State++
			assert.NoError(t, PutTaskStatus(tx, task.ID, &task.Status))

			if i%2 == 0 {
				assert.NoError(t, SetTaskAssignment(tx, task.ID, false))
			}
		}

		return nil
	}))

	assert.NoError(t, db.View(func(tx *bolt.Tx) error {
		for i, task := range tasks {
			status, err := GetTaskStatus(tx, task.ID)
			assert.NoError(t, err)
			assert.Equal(t, &task.Status, status)

			retrieved, err := GetTask(tx, task.ID)
			assert.NoError(t, err)

			task.Status = api.TaskStatus{}
			assert.Equal(t, task, retrieved)

			if i%2 == 0 {
				assert.False(t, TaskAssigned(tx, task.ID))
			} else {
				assert.True(t, TaskAssigned(tx, task.ID))
			}

		}

		return nil
	}))
}

func genTasks(n int) []*api.Task {
	var tasks []*api.Task
	for i := 0; i < n; i++ {
		tasks = append(tasks, genTask())
	}

	sort.Stable(tasksByID(tasks))

	return tasks
}

func genTask() *api.Task {
	return &api.Task{
		ID:        identity.NewID(),
		ServiceID: identity.NewID(),
		Status:    *genTaskStatus(),
		Spec: api.TaskSpec{
			Runtime: &api.TaskSpec_Container{
				Container: &api.ContainerSpec{
					Image:   "foo",
					Command: []string{"this", "-w", "works"},
				},
			},
		},
	}
}

var taskStates = []api.TaskState{
	api.TaskStateAssigned, api.TaskStateAccepted,
	api.TaskStatePreparing, api.TaskStateReady,
	api.TaskStateStarting, api.TaskStateRunning,
	api.TaskStateCompleted, api.TaskStateFailed,
	api.TaskStateRejected, api.TaskStateShutdown,
}

func genTaskStatus() *api.TaskStatus {
	return &api.TaskStatus{
		State:   taskStates[rand.Intn(len(taskStates))],
		Message: identity.NewID(), // just put some garbage here.
	}
}

// storageTestEnv returns an initialized db and cleanup function for use in
// tests.
func storageTestEnv(t *testing.T) (*bolt.DB, func()) {
	var cleanup []func()
	dir, err := ioutil.TempDir("", "agent-TestStorage-")
	assert.NoError(t, err)

	dbpath := filepath.Join(dir, "tasks.db")
	assert.NoError(t, os.MkdirAll(dir, 0777))
	cleanup = append(cleanup, func() { os.RemoveAll(dir) })

	db, err := bolt.Open(dbpath, 0666, nil)
	assert.NoError(t, err)
	cleanup = append(cleanup, func() { db.Close() })

	assert.NoError(t, InitDB(db))
	return db, func() {
		// iterate in reverse so it works like defer
		for i := len(cleanup) - 1; i >= 0; i-- {
			cleanup[i]()
		}
	}
}

type tasksByID []*api.Task

func (ts tasksByID) Len() int           { return len(ts) }
func (ts tasksByID) Less(i, j int) bool { return ts[i].ID < ts[j].ID }
func (ts tasksByID) Swap(i, j int)      { ts[i], ts[j] = ts[j], ts[i] }
