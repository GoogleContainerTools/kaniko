package agent

import (
	"testing"

	"github.com/boltdb/bolt"
	"github.com/docker/swarmkit/agent/exec"
	"github.com/docker/swarmkit/api"
	"github.com/docker/swarmkit/log"
	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
	"golang.org/x/net/context"
)

type testPublisherProvider struct {
}

func (tpp *testPublisherProvider) Publisher(ctx context.Context, subscriptionID string) (exec.LogPublisher, func(), error) {
	return exec.LogPublisherFunc(func(ctx context.Context, message api.LogMessage) error {
			log.G(ctx).WithFields(logrus.Fields{
				"subscription": subscriptionID,
				"task.id":      message.Context.TaskID,
				"node.id":      message.Context.NodeID,
				"service.id":   message.Context.ServiceID,
			}).Info(message.Data)
			return nil
		}), func() {
		}, nil
}

func TestWorkerAssign(t *testing.T) {
	db, cleanup := storageTestEnv(t)
	defer cleanup()

	ctx := context.Background()
	executor := &mockExecutor{t: t, dependencies: NewDependencyManager()}
	worker := newWorker(db, executor, &testPublisherProvider{})
	reporter := statusReporterFunc(func(ctx context.Context, taskID string, status *api.TaskStatus) error {
		log.G(ctx).WithFields(logrus.Fields{"task.id": taskID, "status": status}).Info("status update received")
		return nil
	})

	worker.Listen(ctx, reporter)

	for _, testcase := range []struct {
		changeSet        []*api.AssignmentChange
		expectedTasks    []*api.Task
		expectedSecrets  []*api.Secret
		expectedConfigs  []*api.Config
		expectedAssigned []*api.Task
	}{
		{}, // handle nil case.
		{
			changeSet: []*api.AssignmentChange{
				{
					Assignment: &api.Assignment{
						Item: &api.Assignment_Task{
							Task: &api.Task{ID: "task-1"},
						},
					},
					Action: api.AssignmentChange_AssignmentActionUpdate,
				},
				{
					Assignment: &api.Assignment{
						Item: &api.Assignment_Secret{
							Secret: &api.Secret{ID: "secret-1"},
						},
					},
					Action: api.AssignmentChange_AssignmentActionUpdate,
				},
				{
					Assignment: &api.Assignment{
						Item: &api.Assignment_Config{
							Config: &api.Config{ID: "config-1"},
						},
					},
					Action: api.AssignmentChange_AssignmentActionUpdate,
				},
				// these should be ignored
				{
					Assignment: &api.Assignment{
						Item: &api.Assignment_Secret{
							Secret: &api.Secret{ID: "secret-2"},
						},
					},
					Action: api.AssignmentChange_AssignmentActionRemove,
				},
				{
					Assignment: &api.Assignment{
						Item: &api.Assignment_Task{
							Task: &api.Task{ID: "task-2"},
						},
					},
					Action: api.AssignmentChange_AssignmentActionRemove,
				},
				{
					Assignment: &api.Assignment{
						Item: &api.Assignment_Config{
							Config: &api.Config{ID: "config-2"},
						},
					},
					Action: api.AssignmentChange_AssignmentActionRemove,
				},
			},
			expectedTasks: []*api.Task{
				{ID: "task-1"},
			},
			expectedSecrets: []*api.Secret{
				{ID: "secret-1"},
			},
			expectedConfigs: []*api.Config{
				{ID: "config-1"},
			},
			expectedAssigned: []*api.Task{
				{ID: "task-1"},
			},
		},
		{ // completely replaces the existing tasks and secrets
			changeSet: []*api.AssignmentChange{
				{
					Assignment: &api.Assignment{
						Item: &api.Assignment_Task{
							Task: &api.Task{ID: "task-2"},
						},
					},
					Action: api.AssignmentChange_AssignmentActionUpdate,
				},
				{
					Assignment: &api.Assignment{
						Item: &api.Assignment_Secret{
							Secret: &api.Secret{ID: "secret-2"},
						},
					},
					Action: api.AssignmentChange_AssignmentActionUpdate,
				},
				{
					Assignment: &api.Assignment{
						Item: &api.Assignment_Config{
							Config: &api.Config{ID: "config-2"},
						},
					},
					Action: api.AssignmentChange_AssignmentActionUpdate,
				},
			},
			expectedTasks: []*api.Task{
				{ID: "task-1"},
				{ID: "task-2"},
			},
			expectedSecrets: []*api.Secret{
				{ID: "secret-2"},
			},
			expectedConfigs: []*api.Config{
				{ID: "config-2"},
			},
			expectedAssigned: []*api.Task{
				{ID: "task-2"},
			},
		},
		{
			// remove assigned tasks, secret and config no longer present
			expectedTasks: []*api.Task{
				{ID: "task-1"},
				{ID: "task-2"},
			},
		},

		// TODO(stevvooe): There are a few more states here we need to get
		// covered to ensure correct during code changes.
	} {
		assert.NoError(t, worker.Assign(ctx, testcase.changeSet))

		var (
			tasks    []*api.Task
			assigned []*api.Task
		)
		assert.NoError(t, worker.db.View(func(tx *bolt.Tx) error {
			return WalkTasks(tx, func(task *api.Task) error {
				tasks = append(tasks, task)
				if TaskAssigned(tx, task.ID) {
					assigned = append(assigned, task)
				}
				return nil
			})
		}))

		assert.Equal(t, testcase.expectedTasks, tasks)
		assert.Equal(t, testcase.expectedAssigned, assigned)
		for _, secret := range testcase.expectedSecrets {
			secret, err := executor.Secrets().Get(secret.ID)
			assert.NoError(t, err)
			assert.NotNil(t, secret)
		}
		for _, config := range testcase.expectedConfigs {
			config, err := executor.Configs().Get(config.ID)
			assert.NoError(t, err)
			assert.NotNil(t, config)
		}
	}
}

func TestWorkerWait(t *testing.T) {
	db, cleanup := storageTestEnv(t)
	defer cleanup()

	ctx := context.Background()
	executor := &mockExecutor{t: t, dependencies: NewDependencyManager()}
	worker := newWorker(db, executor, &testPublisherProvider{})
	reporter := statusReporterFunc(func(ctx context.Context, taskID string, status *api.TaskStatus) error {
		log.G(ctx).WithFields(logrus.Fields{"task.id": taskID, "status": status}).Info("status update received")
		return nil
	})

	worker.Listen(ctx, reporter)

	changeSet := []*api.AssignmentChange{
		{
			Assignment: &api.Assignment{
				Item: &api.Assignment_Task{
					Task: &api.Task{ID: "task-1"},
				},
			},
			Action: api.AssignmentChange_AssignmentActionUpdate,
		},
		{
			Assignment: &api.Assignment{
				Item: &api.Assignment_Task{
					Task: &api.Task{ID: "task-2"},
				},
			},
			Action: api.AssignmentChange_AssignmentActionUpdate,
		},
		{
			Assignment: &api.Assignment{
				Item: &api.Assignment_Secret{
					Secret: &api.Secret{ID: "secret-1"},
				},
			},
			Action: api.AssignmentChange_AssignmentActionUpdate,
		},
		{
			Assignment: &api.Assignment{
				Item: &api.Assignment_Config{
					Config: &api.Config{ID: "config-1"},
				},
			},
			Action: api.AssignmentChange_AssignmentActionUpdate,
		},
	}

	expectedTasks := []*api.Task{
		{ID: "task-1"},
		{ID: "task-2"},
	}

	expectedSecrets := []*api.Secret{
		{ID: "secret-1"},
	}

	expectedConfigs := []*api.Config{
		{ID: "config-1"},
	}

	expectedAssigned := []*api.Task{
		{ID: "task-1"},
		{ID: "task-2"},
	}

	var (
		tasks    []*api.Task
		assigned []*api.Task
	)
	assert.NoError(t, worker.Assign(ctx, changeSet))

	assert.NoError(t, worker.db.View(func(tx *bolt.Tx) error {
		return WalkTasks(tx, func(task *api.Task) error {
			tasks = append(tasks, task)
			if TaskAssigned(tx, task.ID) {
				assigned = append(assigned, task)
			}
			return nil
		})
	}))

	assert.Equal(t, expectedTasks, tasks)
	assert.Equal(t, expectedAssigned, assigned)
	for _, secret := range expectedSecrets {
		secret, err := executor.Secrets().Get(secret.ID)
		assert.NoError(t, err)
		assert.NotNil(t, secret)
	}
	for _, config := range expectedConfigs {
		config, err := executor.Configs().Get(config.ID)
		assert.NoError(t, err)
		assert.NotNil(t, config)
	}

	err := worker.Assign(ctx, nil)
	assert.Nil(t, err)

	err = worker.Wait(ctx)
	assert.Nil(t, err)

	assigned = assigned[:0]

	assert.NoError(t, worker.db.View(func(tx *bolt.Tx) error {
		return WalkTasks(tx, func(task *api.Task) error {
			if TaskAssigned(tx, task.ID) {
				assigned = append(assigned, task)
			}
			return nil
		})
	}))
	assert.Equal(t, len(assigned), 0)
}

func TestWorkerUpdate(t *testing.T) {
	db, cleanup := storageTestEnv(t)
	defer cleanup()

	ctx := context.Background()
	executor := &mockExecutor{t: t, dependencies: NewDependencyManager()}
	worker := newWorker(db, executor, &testPublisherProvider{})
	reporter := statusReporterFunc(func(ctx context.Context, taskID string, status *api.TaskStatus) error {
		log.G(ctx).WithFields(logrus.Fields{"task.id": taskID, "status": status}).Info("status update received")
		return nil
	})

	worker.Listen(ctx, reporter)

	// create existing task/secret/config
	assert.NoError(t, worker.Assign(ctx, []*api.AssignmentChange{
		{
			Assignment: &api.Assignment{
				Item: &api.Assignment_Task{
					Task: &api.Task{ID: "task-1"},
				},
			},
			Action: api.AssignmentChange_AssignmentActionUpdate,
		},
		{
			Assignment: &api.Assignment{
				Item: &api.Assignment_Secret{
					Secret: &api.Secret{ID: "secret-1"},
				},
			},
			Action: api.AssignmentChange_AssignmentActionUpdate,
		},
		{
			Assignment: &api.Assignment{
				Item: &api.Assignment_Config{
					Config: &api.Config{ID: "config-1"},
				},
			},
			Action: api.AssignmentChange_AssignmentActionUpdate,
		},
	}))

	for _, testcase := range []struct {
		changeSet        []*api.AssignmentChange
		expectedTasks    []*api.Task
		expectedSecrets  []*api.Secret
		expectedConfigs  []*api.Config
		expectedAssigned []*api.Task
	}{
		{ // handle nil changeSet case.
			expectedTasks: []*api.Task{
				{ID: "task-1"},
			},
			expectedSecrets: []*api.Secret{
				{ID: "secret-1"},
			},
			expectedConfigs: []*api.Config{
				{ID: "config-1"},
			},
			expectedAssigned: []*api.Task{
				{ID: "task-1"},
			},
		},
		{
			// no changes
			changeSet: []*api.AssignmentChange{
				{
					Assignment: &api.Assignment{
						Item: &api.Assignment_Task{
							Task: &api.Task{ID: "task-1"},
						},
					},
					Action: api.AssignmentChange_AssignmentActionUpdate,
				},
			},
			expectedTasks: []*api.Task{
				{ID: "task-1"},
			},
			expectedSecrets: []*api.Secret{
				{ID: "secret-1"},
			},
			expectedConfigs: []*api.Config{
				{ID: "config-1"},
			},
			expectedAssigned: []*api.Task{
				{ID: "task-1"},
			},
		},
		{
			// adding a secret and task
			changeSet: []*api.AssignmentChange{
				{
					Assignment: &api.Assignment{
						Item: &api.Assignment_Task{
							Task: &api.Task{ID: "task-2"},
						},
					},
					Action: api.AssignmentChange_AssignmentActionUpdate,
				},
				{
					Assignment: &api.Assignment{
						Item: &api.Assignment_Secret{
							Secret: &api.Secret{ID: "secret-2"},
						},
					},
					Action: api.AssignmentChange_AssignmentActionUpdate,
				},
				{
					Assignment: &api.Assignment{
						Item: &api.Assignment_Config{
							Config: &api.Config{ID: "config-2"},
						},
					},
					Action: api.AssignmentChange_AssignmentActionUpdate,
				},
			},
			expectedTasks: []*api.Task{
				{ID: "task-1"},
				{ID: "task-2"},
			},
			expectedSecrets: []*api.Secret{
				{ID: "secret-1"},
				{ID: "secret-2"},
			},
			expectedConfigs: []*api.Config{
				{ID: "config-1"},
				{ID: "config-2"},
			},
			expectedAssigned: []*api.Task{
				{ID: "task-1"},
				{ID: "task-2"},
			},
		},
		{
			// remove assigned task and secret, updating existing secret
			changeSet: []*api.AssignmentChange{
				{
					Assignment: &api.Assignment{
						Item: &api.Assignment_Task{
							Task: &api.Task{ID: "task-1"},
						},
					},
					Action: api.AssignmentChange_AssignmentActionRemove,
				},
				{
					Assignment: &api.Assignment{
						Item: &api.Assignment_Secret{
							Secret: &api.Secret{ID: "secret-1"},
						},
					},
					Action: api.AssignmentChange_AssignmentActionRemove,
				},
				{
					Assignment: &api.Assignment{
						Item: &api.Assignment_Secret{
							Secret: &api.Secret{ID: "secret-2"},
						},
					},
					Action: api.AssignmentChange_AssignmentActionUpdate,
				},
				{
					Assignment: &api.Assignment{
						Item: &api.Assignment_Config{
							Config: &api.Config{ID: "config-1"},
						},
					},
					Action: api.AssignmentChange_AssignmentActionRemove,
				},
				{
					Assignment: &api.Assignment{
						Item: &api.Assignment_Config{
							Config: &api.Config{ID: "config-2"},
						},
					},
					Action: api.AssignmentChange_AssignmentActionUpdate,
				},
			},
			expectedTasks: []*api.Task{
				{ID: "task-1"},
				{ID: "task-2"},
			},
			expectedSecrets: []*api.Secret{
				{ID: "secret-2"},
			},
			expectedConfigs: []*api.Config{
				{ID: "config-2"},
			},
			expectedAssigned: []*api.Task{
				{ID: "task-2"},
			},
		},
		{
			// removing nonexistent items doesn't fail
			changeSet: []*api.AssignmentChange{
				{
					Assignment: &api.Assignment{
						Item: &api.Assignment_Task{
							Task: &api.Task{ID: "task-1"},
						},
					},
					Action: api.AssignmentChange_AssignmentActionRemove,
				},
				{
					Assignment: &api.Assignment{
						Item: &api.Assignment_Secret{
							Secret: &api.Secret{ID: "secret-1"},
						},
					},
					Action: api.AssignmentChange_AssignmentActionRemove,
				},
				{
					Assignment: &api.Assignment{
						Item: &api.Assignment_Task{
							Task: &api.Task{ID: "task-2"},
						},
					},
					Action: api.AssignmentChange_AssignmentActionRemove,
				},
				{
					Assignment: &api.Assignment{
						Item: &api.Assignment_Secret{
							Secret: &api.Secret{ID: "secret-2"},
						},
					},
					Action: api.AssignmentChange_AssignmentActionRemove,
				},
				{
					Assignment: &api.Assignment{
						Item: &api.Assignment_Config{
							Config: &api.Config{ID: "config-1"},
						},
					},
					Action: api.AssignmentChange_AssignmentActionRemove,
				},
				{
					Assignment: &api.Assignment{
						Item: &api.Assignment_Config{
							Config: &api.Config{ID: "config-2"},
						},
					},
					Action: api.AssignmentChange_AssignmentActionRemove,
				},
			},
			expectedTasks: []*api.Task{
				{ID: "task-1"},
				{ID: "task-2"},
			},
		},
	} {
		assert.NoError(t, worker.Update(ctx, testcase.changeSet))

		var (
			tasks    []*api.Task
			assigned []*api.Task
		)
		assert.NoError(t, worker.db.View(func(tx *bolt.Tx) error {
			return WalkTasks(tx, func(task *api.Task) error {
				tasks = append(tasks, task)
				if TaskAssigned(tx, task.ID) {
					assigned = append(assigned, task)
				}
				return nil
			})
		}))

		assert.Equal(t, testcase.expectedTasks, tasks)
		assert.Equal(t, testcase.expectedAssigned, assigned)
		for _, secret := range testcase.expectedSecrets {
			secret, err := executor.Secrets().Get(secret.ID)
			assert.NoError(t, err)
			assert.NotNil(t, secret)
		}
		for _, config := range testcase.expectedConfigs {
			config, err := executor.Configs().Get(config.ID)
			assert.NoError(t, err)
			assert.NotNil(t, config)
		}
	}
}

type mockTaskController struct {
	t *testing.T
	exec.Controller
	task         *api.Task
	dependencies exec.DependencyGetter
}

func (mtc *mockTaskController) Remove(ctx context.Context) error {
	mtc.t.Log("(*mockTestController).Remove")
	return nil
}

func (mtc *mockTaskController) Close() error {
	mtc.t.Log("(*mockTestController).Close")
	return nil
}

type mockExecutor struct {
	t *testing.T
	exec.Executor
	dependencies exec.DependencyManager
}

func (m *mockExecutor) Controller(task *api.Task) (exec.Controller, error) {
	return &mockTaskController{t: m.t, task: task, dependencies: Restrict(m.dependencies, task)}, nil
}

func (m *mockExecutor) Secrets() exec.SecretsManager {
	return m.dependencies.Secrets()
}

func (m *mockExecutor) Configs() exec.ConfigsManager {
	return m.dependencies.Configs()
}
