package store

import (
	"errors"
	"strconv"
	"sync"
	"testing"
	"time"

	events "github.com/docker/go-events"
	"github.com/docker/swarmkit/api"
	"github.com/docker/swarmkit/identity"
	"github.com/docker/swarmkit/manager/state"
	"github.com/docker/swarmkit/manager/state/testutils"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

var (
	clusterSet = []*api.Cluster{
		{
			ID: "id1",
			Spec: api.ClusterSpec{
				Annotations: api.Annotations{
					Name: "name1",
				},
			},
		},
		{
			ID: "id2",
			Spec: api.ClusterSpec{
				Annotations: api.Annotations{
					Name: "name2",
				},
			},
		},
		{
			ID: "id3",
			Spec: api.ClusterSpec{
				Annotations: api.Annotations{
					Name: "name3",
				},
			},
		},
	}
	altClusterSet = []*api.Cluster{
		{
			ID: "alt-id1",
			Spec: api.ClusterSpec{
				Annotations: api.Annotations{
					Name: "alt-name1",
				},
			},
		},
	}

	nodeSet = []*api.Node{
		{
			ID: "id1",
			Spec: api.NodeSpec{
				Membership: api.NodeMembershipPending,
			},
			Description: &api.NodeDescription{
				Hostname: "name1",
			},
			Role: api.NodeRoleManager,
		},
		{
			ID: "id2",
			Spec: api.NodeSpec{
				Membership: api.NodeMembershipAccepted,
			},
			Description: &api.NodeDescription{
				Hostname: "name2",
			},
			Role: api.NodeRoleWorker,
		},
		{
			ID: "id3",
			Spec: api.NodeSpec{
				Membership: api.NodeMembershipAccepted,
			},
			Description: &api.NodeDescription{
				// intentionally conflicting hostname
				Hostname: "name2",
			},
			Role: api.NodeRoleWorker,
		},
	}
	altNodeSet = []*api.Node{
		{
			ID: "alt-id1",
			Spec: api.NodeSpec{
				Membership: api.NodeMembershipPending,
			},
			Description: &api.NodeDescription{
				Hostname: "alt-name1",
			},
			Role: api.NodeRoleManager,
		},
	}

	serviceSet = []*api.Service{
		{
			ID: "id1",
			Spec: api.ServiceSpec{
				Annotations: api.Annotations{
					Name: "name1",
				},
			},
		},
		{
			ID: "id2",
			Spec: api.ServiceSpec{
				Annotations: api.Annotations{
					Name: "name2",
				},
				Mode: &api.ServiceSpec_Global{
					Global: &api.GlobalService{},
				},
			},
		},
		{
			ID: "id3",
			Spec: api.ServiceSpec{
				Annotations: api.Annotations{
					Name: "name3",
				},
			},
		},
	}
	altServiceSet = []*api.Service{
		{
			ID: "alt-id1",
			Spec: api.ServiceSpec{
				Annotations: api.Annotations{
					Name: "alt-name1",
				},
			},
		},
	}

	taskSet = []*api.Task{
		{
			ID: "id1",
			Annotations: api.Annotations{
				Name: "name1",
			},
			ServiceAnnotations: api.Annotations{
				Name: "name1",
			},
			DesiredState: api.TaskStateRunning,
			NodeID:       nodeSet[0].ID,
		},
		{
			ID: "id2",
			Annotations: api.Annotations{
				Name: "name2.1",
			},
			ServiceAnnotations: api.Annotations{
				Name: "name2",
			},
			DesiredState: api.TaskStateRunning,
			ServiceID:    serviceSet[0].ID,
		},
		{
			ID: "id3",
			Annotations: api.Annotations{
				Name: "name2.2",
			},
			ServiceAnnotations: api.Annotations{
				Name: "name2",
			},
			DesiredState: api.TaskStateShutdown,
		},
	}
	altTaskSet = []*api.Task{
		{
			ID: "alt-id1",
			Annotations: api.Annotations{
				Name: "alt-name1",
			},
			ServiceAnnotations: api.Annotations{
				Name: "alt-name1",
			},
			DesiredState: api.TaskStateRunning,
			NodeID:       altNodeSet[0].ID,
		},
	}

	networkSet = []*api.Network{
		{
			ID: "id1",
			Spec: api.NetworkSpec{
				Annotations: api.Annotations{
					Name: "name1",
				},
			},
		},
		{
			ID: "id2",
			Spec: api.NetworkSpec{
				Annotations: api.Annotations{
					Name: "name2",
				},
			},
		},
		{
			ID: "id3",
			Spec: api.NetworkSpec{
				Annotations: api.Annotations{
					Name: "name3",
				},
			},
		},
	}
	altNetworkSet = []*api.Network{
		{
			ID: "alt-id1",
			Spec: api.NetworkSpec{
				Annotations: api.Annotations{
					Name: "alt-name1",
				},
			},
		},
	}

	configSet = []*api.Config{
		{
			ID: "id1",
			Spec: api.ConfigSpec{
				Annotations: api.Annotations{
					Name: "name1",
				},
			},
		},
		{
			ID: "id2",
			Spec: api.ConfigSpec{
				Annotations: api.Annotations{
					Name: "name2",
				},
			},
		},
		{
			ID: "id3",
			Spec: api.ConfigSpec{
				Annotations: api.Annotations{
					Name: "name3",
				},
			},
		},
	}
	altConfigSet = []*api.Config{
		{
			ID: "alt-id1",
			Spec: api.ConfigSpec{
				Annotations: api.Annotations{
					Name: "alt-name1",
				},
			},
		},
	}

	secretSet = []*api.Secret{
		{
			ID: "id1",
			Spec: api.SecretSpec{
				Annotations: api.Annotations{
					Name: "name1",
				},
			},
		},
		{
			ID: "id2",
			Spec: api.SecretSpec{
				Annotations: api.Annotations{
					Name: "name2",
				},
			},
		},
		{
			ID: "id3",
			Spec: api.SecretSpec{
				Annotations: api.Annotations{
					Name: "name3",
				},
			},
		},
	}
	altSecretSet = []*api.Secret{
		{
			ID: "alt-id1",
			Spec: api.SecretSpec{
				Annotations: api.Annotations{
					Name: "alt-name1",
				},
			},
		},
	}

	extensionSet = []*api.Extension{
		{
			ID: "id1",
			Annotations: api.Annotations{
				Name: "name1",
			},
		},
		{
			ID: "id2",
			Annotations: api.Annotations{
				Name: "name2",
			},
		},
		{
			ID: "id3",
			Annotations: api.Annotations{
				Name: "name3",
			},
		},
	}
	altExtensionSet = []*api.Extension{
		{
			ID: "alt-id1",
			Annotations: api.Annotations{
				Name: "alt-name1",
			},
		},
	}

	resourceSet = []*api.Resource{
		{
			ID: "id1",
			Annotations: api.Annotations{
				Name: "name1",
			},
			Kind: "name1", // corresponds to extension id1
		},
		{
			ID: "id2",
			Annotations: api.Annotations{
				Name: "name2",
			},
			Kind: "name2", // corresponds to extension id2
		},
		{
			ID: "id3",
			Annotations: api.Annotations{
				Name: "name3",
			},
			Kind: "name3", // corresponds to extension id3
		},
	}
	altResourceSet = []*api.Resource{
		{
			ID: "alt-id1",
			Annotations: api.Annotations{
				Name: "alt-name1",
			},
			Kind: "alt-name1", // corresponds to extension alt-id1
		},
	}
)

func setupTestStore(t *testing.T, s *MemoryStore) {
	populateTestStore(t, s,
		clusterSet, nodeSet, serviceSet, taskSet, networkSet, configSet, secretSet,
		extensionSet, resourceSet)
}

func populateTestStore(t *testing.T, s *MemoryStore,
	clusters []*api.Cluster, nodes []*api.Node, services []*api.Service, tasks []*api.Task, networks []*api.Network,
	configs []*api.Config, secrets []*api.Secret, extensions []*api.Extension, resources []*api.Resource) {
	err := s.Update(func(tx Tx) error {
		// Prepoulate clusters
		for _, c := range clusters {
			assert.NoError(t, CreateCluster(tx, c))
		}

		// Prepoulate nodes
		for _, n := range nodes {
			assert.NoError(t, CreateNode(tx, n))
		}

		// Prepopulate services
		for _, s := range services {
			assert.NoError(t, CreateService(tx, s))
		}
		// Prepopulate tasks
		for _, task := range tasks {
			assert.NoError(t, CreateTask(tx, task))
		}
		// Prepopulate networks
		for _, n := range networks {
			assert.NoError(t, CreateNetwork(tx, n))
		}
		// Prepopulate configs
		for _, c := range configs {
			assert.NoError(t, CreateConfig(tx, c))
		}
		// Prepopulate secrets
		for _, s := range secrets {
			assert.NoError(t, CreateSecret(tx, s))
		}
		// Prepopulate extensions
		for _, c := range extensions {
			assert.NoError(t, CreateExtension(tx, c))
		}
		// Prepopulate resources
		for _, s := range resources {
			assert.NoError(t, CreateResource(tx, s))
		}
		return nil
	})
	assert.NoError(t, err)
}

func TestStoreNode(t *testing.T) {
	s := NewMemoryStore(nil)
	assert.NotNil(t, s)

	s.View(func(readTx ReadTx) {
		allNodes, err := FindNodes(readTx, All)
		assert.NoError(t, err)
		assert.Empty(t, allNodes)
	})

	setupTestStore(t, s)

	err := s.Update(func(tx Tx) error {
		allNodes, err := FindNodes(tx, All)
		assert.NoError(t, err)
		assert.Len(t, allNodes, len(nodeSet))

		assert.Error(t, CreateNode(tx, nodeSet[0]), "duplicate IDs must be rejected")
		return nil
	})
	assert.NoError(t, err)

	s.View(func(readTx ReadTx) {
		assert.Equal(t, nodeSet[0], GetNode(readTx, "id1"))
		assert.Equal(t, nodeSet[1], GetNode(readTx, "id2"))
		assert.Equal(t, nodeSet[2], GetNode(readTx, "id3"))

		foundNodes, err := FindNodes(readTx, ByName("name1"))
		assert.NoError(t, err)
		assert.Len(t, foundNodes, 1)
		foundNodes, err = FindNodes(readTx, ByName("name2"))
		assert.NoError(t, err)
		assert.Len(t, foundNodes, 2)
		foundNodes, err = FindNodes(readTx, Or(ByName("name1"), ByName("name2")))
		assert.NoError(t, err)
		assert.Len(t, foundNodes, 3)
		foundNodes, err = FindNodes(readTx, ByName("invalid"))
		assert.NoError(t, err)
		assert.Len(t, foundNodes, 0)

		foundNodes, err = FindNodes(readTx, ByIDPrefix("id"))
		assert.NoError(t, err)
		assert.Len(t, foundNodes, 3)

		foundNodes, err = FindNodes(readTx, ByRole(api.NodeRoleManager))
		assert.NoError(t, err)
		assert.Len(t, foundNodes, 1)

		foundNodes, err = FindNodes(readTx, ByRole(api.NodeRoleWorker))
		assert.NoError(t, err)
		assert.Len(t, foundNodes, 2)

		foundNodes, err = FindNodes(readTx, ByMembership(api.NodeMembershipPending))
		assert.NoError(t, err)
		assert.Len(t, foundNodes, 1)

		foundNodes, err = FindNodes(readTx, ByMembership(api.NodeMembershipAccepted))
		assert.NoError(t, err)
		assert.Len(t, foundNodes, 2)
	})

	// Update.
	update := &api.Node{
		ID: "id3",
		Description: &api.NodeDescription{
			Hostname: "name3",
		},
	}
	err = s.Update(func(tx Tx) error {
		assert.NotEqual(t, update, GetNode(tx, "id3"))
		assert.NoError(t, UpdateNode(tx, update))
		assert.Equal(t, update, GetNode(tx, "id3"))

		foundNodes, err := FindNodes(tx, ByName("name2"))
		assert.NoError(t, err)
		assert.Len(t, foundNodes, 1)
		foundNodes, err = FindNodes(tx, ByName("name3"))
		assert.NoError(t, err)
		assert.Len(t, foundNodes, 1)

		invalidUpdate := *nodeSet[0]
		invalidUpdate.ID = "invalid"
		assert.Error(t, UpdateNode(tx, &invalidUpdate), "invalid IDs should be rejected")

		// Delete
		assert.NotNil(t, GetNode(tx, "id1"))
		assert.NoError(t, DeleteNode(tx, "id1"))
		assert.Nil(t, GetNode(tx, "id1"))
		foundNodes, err = FindNodes(tx, ByName("name1"))
		assert.NoError(t, err)
		assert.Empty(t, foundNodes)

		assert.Equal(t, DeleteNode(tx, "nonexistent"), ErrNotExist)
		return nil
	})
	assert.NoError(t, err)
}

func TestStoreService(t *testing.T) {
	s := NewMemoryStore(nil)
	assert.NotNil(t, s)

	s.View(func(readTx ReadTx) {
		allServices, err := FindServices(readTx, All)
		assert.NoError(t, err)
		assert.Empty(t, allServices)
	})

	setupTestStore(t, s)

	err := s.Update(func(tx Tx) error {
		assert.Equal(t,
			CreateService(tx, &api.Service{
				ID: "id1",
				Spec: api.ServiceSpec{
					Annotations: api.Annotations{
						Name: "name4",
					},
				},
			}), ErrExist, "duplicate IDs must be rejected")

		assert.Equal(t,
			CreateService(tx, &api.Service{
				ID: "id4",
				Spec: api.ServiceSpec{
					Annotations: api.Annotations{
						Name: "name1",
					},
				},
			}), ErrNameConflict, "duplicate names must be rejected")

		assert.Equal(t,
			CreateService(tx, &api.Service{
				ID: "id4",
				Spec: api.ServiceSpec{
					Annotations: api.Annotations{
						Name: "NAME1",
					},
				},
			}), ErrNameConflict, "duplicate check should be case insensitive")
		return nil
	})
	assert.NoError(t, err)

	s.View(func(readTx ReadTx) {
		assert.Equal(t, serviceSet[0], GetService(readTx, "id1"))
		assert.Equal(t, serviceSet[1], GetService(readTx, "id2"))
		assert.Equal(t, serviceSet[2], GetService(readTx, "id3"))

		foundServices, err := FindServices(readTx, ByNamePrefix("name1"))
		assert.NoError(t, err)
		assert.Len(t, foundServices, 1)
		foundServices, err = FindServices(readTx, ByNamePrefix("NAME1"))
		assert.NoError(t, err)
		assert.Len(t, foundServices, 1)
		foundServices, err = FindServices(readTx, ByNamePrefix("invalid"))
		assert.NoError(t, err)
		assert.Len(t, foundServices, 0)
		foundServices, err = FindServices(readTx, Or(ByNamePrefix("name1"), ByNamePrefix("name2")))
		assert.NoError(t, err)
		assert.Len(t, foundServices, 2)
		foundServices, err = FindServices(readTx, Or(ByNamePrefix("name1"), ByNamePrefix("name2"), ByNamePrefix("name4")))
		assert.NoError(t, err)
		assert.Len(t, foundServices, 2)

		foundServices, err = FindServices(readTx, ByIDPrefix("id"))
		assert.NoError(t, err)
		assert.Len(t, foundServices, 3)
	})

	// Update.
	err = s.Update(func(tx Tx) error {
		// Regular update.
		update := serviceSet[0].Copy()
		update.Spec.Annotations.Labels = map[string]string{
			"foo": "bar",
		}

		assert.NotEqual(t, update, GetService(tx, update.ID))
		assert.NoError(t, UpdateService(tx, update))
		assert.Equal(t, update, GetService(tx, update.ID))

		// Name conflict.
		update = GetService(tx, update.ID)
		update.Spec.Annotations.Name = "name2"
		assert.Equal(t, UpdateService(tx, update), ErrNameConflict, "duplicate names should be rejected")
		update = GetService(tx, update.ID)
		update.Spec.Annotations.Name = "NAME2"
		assert.Equal(t, UpdateService(tx, update), ErrNameConflict, "duplicate check should be case insensitive")

		// Name change.
		update = GetService(tx, update.ID)
		foundServices, err := FindServices(tx, ByNamePrefix("name1"))
		assert.NoError(t, err)
		assert.Len(t, foundServices, 1)
		foundServices, err = FindServices(tx, ByNamePrefix("name4"))
		assert.NoError(t, err)
		assert.Empty(t, foundServices)

		update.Spec.Annotations.Name = "name4"
		assert.NoError(t, UpdateService(tx, update))
		foundServices, err = FindServices(tx, ByNamePrefix("name1"))
		assert.NoError(t, err)
		assert.Empty(t, foundServices)
		foundServices, err = FindServices(tx, ByNamePrefix("name4"))
		assert.NoError(t, err)
		assert.Len(t, foundServices, 1)

		// Invalid update.
		invalidUpdate := serviceSet[0].Copy()
		invalidUpdate.ID = "invalid"
		assert.Error(t, UpdateService(tx, invalidUpdate), "invalid IDs should be rejected")

		return nil
	})
	assert.NoError(t, err)

	// Delete
	err = s.Update(func(tx Tx) error {
		assert.NotNil(t, GetService(tx, "id1"))
		assert.NoError(t, DeleteService(tx, "id1"))
		assert.Nil(t, GetService(tx, "id1"))
		foundServices, err := FindServices(tx, ByNamePrefix("name1"))
		assert.NoError(t, err)
		assert.Empty(t, foundServices)

		assert.Equal(t, DeleteService(tx, "nonexistent"), ErrNotExist)
		return nil
	})
	assert.NoError(t, err)
}

func TestStoreNetwork(t *testing.T) {
	s := NewMemoryStore(nil)
	assert.NotNil(t, s)

	s.View(func(readTx ReadTx) {
		allNetworks, err := FindNetworks(readTx, All)
		assert.NoError(t, err)
		assert.Empty(t, allNetworks)
	})

	setupTestStore(t, s)

	err := s.Update(func(tx Tx) error {
		allNetworks, err := FindNetworks(tx, All)
		assert.NoError(t, err)
		assert.Len(t, allNetworks, len(networkSet))

		assert.Error(t, CreateNetwork(tx, networkSet[0]), "duplicate IDs must be rejected")
		return nil
	})
	assert.NoError(t, err)

	s.View(func(readTx ReadTx) {
		assert.Equal(t, networkSet[0], GetNetwork(readTx, "id1"))
		assert.Equal(t, networkSet[1], GetNetwork(readTx, "id2"))
		assert.Equal(t, networkSet[2], GetNetwork(readTx, "id3"))

		foundNetworks, err := FindNetworks(readTx, ByName("name1"))
		assert.NoError(t, err)
		assert.Len(t, foundNetworks, 1)
		foundNetworks, err = FindNetworks(readTx, ByName("name2"))
		assert.NoError(t, err)
		assert.Len(t, foundNetworks, 1)
		foundNetworks, err = FindNetworks(readTx, ByName("invalid"))
		assert.NoError(t, err)
		assert.Len(t, foundNetworks, 0)
	})

	err = s.Update(func(tx Tx) error {
		// Delete
		assert.NotNil(t, GetNetwork(tx, "id1"))
		assert.NoError(t, DeleteNetwork(tx, "id1"))
		assert.Nil(t, GetNetwork(tx, "id1"))
		foundNetworks, err := FindNetworks(tx, ByName("name1"))
		assert.NoError(t, err)
		assert.Empty(t, foundNetworks)

		assert.Equal(t, DeleteNetwork(tx, "nonexistent"), ErrNotExist)
		return nil
	})

	assert.NoError(t, err)
}

func TestStoreTask(t *testing.T) {
	s := NewMemoryStore(nil)
	assert.NotNil(t, s)

	s.View(func(tx ReadTx) {
		allTasks, err := FindTasks(tx, All)
		assert.NoError(t, err)
		assert.Empty(t, allTasks)
	})

	setupTestStore(t, s)

	err := s.Update(func(tx Tx) error {
		allTasks, err := FindTasks(tx, All)
		assert.NoError(t, err)
		assert.Len(t, allTasks, len(taskSet))

		assert.Error(t, CreateTask(tx, taskSet[0]), "duplicate IDs must be rejected")
		return nil
	})
	assert.NoError(t, err)

	s.View(func(readTx ReadTx) {
		assert.Equal(t, taskSet[0], GetTask(readTx, "id1"))
		assert.Equal(t, taskSet[1], GetTask(readTx, "id2"))
		assert.Equal(t, taskSet[2], GetTask(readTx, "id3"))

		foundTasks, err := FindTasks(readTx, ByNamePrefix("name1"))
		assert.NoError(t, err)
		assert.Len(t, foundTasks, 1)
		foundTasks, err = FindTasks(readTx, ByNamePrefix("name2"))
		assert.NoError(t, err)
		assert.Len(t, foundTasks, 2)
		foundTasks, err = FindTasks(readTx, ByNamePrefix("invalid"))
		assert.NoError(t, err)
		assert.Len(t, foundTasks, 0)

		foundTasks, err = FindTasks(readTx, ByNodeID(nodeSet[0].ID))
		assert.NoError(t, err)
		assert.Len(t, foundTasks, 1)
		assert.Equal(t, foundTasks[0], taskSet[0])
		foundTasks, err = FindTasks(readTx, ByNodeID("invalid"))
		assert.NoError(t, err)
		assert.Len(t, foundTasks, 0)

		foundTasks, err = FindTasks(readTx, ByServiceID(serviceSet[0].ID))
		assert.NoError(t, err)
		assert.Len(t, foundTasks, 1)
		assert.Equal(t, foundTasks[0], taskSet[1])
		foundTasks, err = FindTasks(readTx, ByServiceID("invalid"))
		assert.NoError(t, err)
		assert.Len(t, foundTasks, 0)

		foundTasks, err = FindTasks(readTx, ByDesiredState(api.TaskStateRunning))
		assert.NoError(t, err)
		assert.Len(t, foundTasks, 2)
		assert.Equal(t, foundTasks[0].DesiredState, api.TaskStateRunning)
		assert.Equal(t, foundTasks[0].DesiredState, api.TaskStateRunning)
		foundTasks, err = FindTasks(readTx, ByDesiredState(api.TaskStateShutdown))
		assert.NoError(t, err)
		assert.Len(t, foundTasks, 1)
		assert.Equal(t, foundTasks[0], taskSet[2])
		foundTasks, err = FindTasks(readTx, ByDesiredState(api.TaskStatePending))
		assert.NoError(t, err)
		assert.Len(t, foundTasks, 0)
	})

	// Update.
	update := &api.Task{
		ID: "id3",
		Annotations: api.Annotations{
			Name: "name3",
		},
		ServiceAnnotations: api.Annotations{
			Name: "name3",
		},
	}
	err = s.Update(func(tx Tx) error {
		assert.NotEqual(t, update, GetTask(tx, "id3"))
		assert.NoError(t, UpdateTask(tx, update))
		assert.Equal(t, update, GetTask(tx, "id3"))

		foundTasks, err := FindTasks(tx, ByNamePrefix("name2"))
		assert.NoError(t, err)
		assert.Len(t, foundTasks, 1)
		foundTasks, err = FindTasks(tx, ByNamePrefix("name3"))
		assert.NoError(t, err)
		assert.Len(t, foundTasks, 1)

		invalidUpdate := *taskSet[0]
		invalidUpdate.ID = "invalid"
		assert.Error(t, UpdateTask(tx, &invalidUpdate), "invalid IDs should be rejected")

		// Delete
		assert.NotNil(t, GetTask(tx, "id1"))
		assert.NoError(t, DeleteTask(tx, "id1"))
		assert.Nil(t, GetTask(tx, "id1"))
		foundTasks, err = FindTasks(tx, ByNamePrefix("name1"))
		assert.NoError(t, err)
		assert.Empty(t, foundTasks)

		assert.Equal(t, DeleteTask(tx, "nonexistent"), ErrNotExist)
		return nil
	})
	assert.NoError(t, err)
}

func TestStoreSnapshot(t *testing.T) {
	s1 := NewMemoryStore(nil)
	assert.NotNil(t, s1)

	setupTestStore(t, s1)

	s2 := NewMemoryStore(nil)
	assert.NotNil(t, s2)

	copyToS2 := func(readTx ReadTx) error {
		return s2.Update(func(tx Tx) error {
			// Copy over new data
			nodes, err := FindNodes(readTx, All)
			if err != nil {
				return err
			}
			for _, n := range nodes {
				if err := CreateNode(tx, n); err != nil {
					return err
				}
			}

			tasks, err := FindTasks(readTx, All)
			if err != nil {
				return err
			}
			for _, t := range tasks {
				if err := CreateTask(tx, t); err != nil {
					return err
				}
			}

			services, err := FindServices(readTx, All)
			if err != nil {
				return err
			}
			for _, s := range services {
				if err := CreateService(tx, s); err != nil {
					return err
				}
			}

			networks, err := FindNetworks(readTx, All)
			if err != nil {
				return err
			}
			for _, n := range networks {
				if err := CreateNetwork(tx, n); err != nil {
					return err
				}
			}

			return nil
		})
	}

	// Fork
	watcher, cancel, err := ViewAndWatch(s1, copyToS2)
	defer cancel()
	assert.NoError(t, err)

	s2.View(func(tx2 ReadTx) {
		assert.Equal(t, nodeSet[0], GetNode(tx2, "id1"))
		assert.Equal(t, nodeSet[1], GetNode(tx2, "id2"))
		assert.Equal(t, nodeSet[2], GetNode(tx2, "id3"))

		assert.Equal(t, serviceSet[0], GetService(tx2, "id1"))
		assert.Equal(t, serviceSet[1], GetService(tx2, "id2"))
		assert.Equal(t, serviceSet[2], GetService(tx2, "id3"))

		assert.Equal(t, taskSet[0], GetTask(tx2, "id1"))
		assert.Equal(t, taskSet[1], GetTask(tx2, "id2"))
		assert.Equal(t, taskSet[2], GetTask(tx2, "id3"))
	})

	// Create node
	createNode := &api.Node{
		ID: "id4",
		Spec: api.NodeSpec{
			Annotations: api.Annotations{
				Name: "name4",
			},
		},
	}

	err = s1.Update(func(tx1 Tx) error {
		assert.NoError(t, CreateNode(tx1, createNode))
		return nil
	})
	assert.NoError(t, err)

	assert.NoError(t, Apply(s2, <-watcher))
	<-watcher // consume commit event

	s2.View(func(tx2 ReadTx) {
		assert.Equal(t, createNode, GetNode(tx2, "id4"))
	})

	// Update node
	updateNode := &api.Node{
		ID: "id3",
		Spec: api.NodeSpec{
			Annotations: api.Annotations{
				Name: "name3",
			},
		},
	}

	err = s1.Update(func(tx1 Tx) error {
		assert.NoError(t, UpdateNode(tx1, updateNode))
		return nil
	})
	assert.NoError(t, err)

	assert.NoError(t, Apply(s2, <-watcher))
	<-watcher // consume commit event

	s2.View(func(tx2 ReadTx) {
		assert.Equal(t, updateNode, GetNode(tx2, "id3"))
	})

	err = s1.Update(func(tx1 Tx) error {
		// Delete node
		assert.NoError(t, DeleteNode(tx1, "id1"))
		return nil
	})
	assert.NoError(t, err)

	assert.NoError(t, Apply(s2, <-watcher))
	<-watcher // consume commit event

	s2.View(func(tx2 ReadTx) {
		assert.Nil(t, GetNode(tx2, "id1"))
	})

	// Create service
	createService := &api.Service{
		ID: "id4",
		Spec: api.ServiceSpec{
			Annotations: api.Annotations{
				Name: "name4",
			},
		},
	}

	err = s1.Update(func(tx1 Tx) error {
		assert.NoError(t, CreateService(tx1, createService))
		return nil
	})
	assert.NoError(t, err)

	assert.NoError(t, Apply(s2, <-watcher))
	<-watcher // consume commit event

	s2.View(func(tx2 ReadTx) {
		assert.Equal(t, createService, GetService(tx2, "id4"))
	})

	// Update service
	updateService := serviceSet[2].Copy()
	updateService.Spec.Annotations.Name = "new-name"
	err = s1.Update(func(tx1 Tx) error {
		assert.NotEqual(t, updateService, GetService(tx1, updateService.ID))
		assert.NoError(t, UpdateService(tx1, updateService))
		return nil
	})
	assert.NoError(t, err)

	assert.NoError(t, Apply(s2, <-watcher))
	<-watcher // consume commit event

	s2.View(func(tx2 ReadTx) {
		assert.Equal(t, updateService, GetService(tx2, "id3"))
	})

	err = s1.Update(func(tx1 Tx) error {
		// Delete service
		assert.NoError(t, DeleteService(tx1, "id1"))
		return nil
	})
	assert.NoError(t, err)

	assert.NoError(t, Apply(s2, <-watcher))
	<-watcher // consume commit event

	s2.View(func(tx2 ReadTx) {
		assert.Nil(t, GetService(tx2, "id1"))
	})

	// Create task
	createTask := &api.Task{
		ID: "id4",
		ServiceAnnotations: api.Annotations{
			Name: "name4",
		},
	}

	err = s1.Update(func(tx1 Tx) error {
		assert.NoError(t, CreateTask(tx1, createTask))
		return nil
	})
	assert.NoError(t, err)

	assert.NoError(t, Apply(s2, <-watcher))
	<-watcher // consume commit event

	s2.View(func(tx2 ReadTx) {
		assert.Equal(t, createTask, GetTask(tx2, "id4"))
	})

	// Update task
	updateTask := &api.Task{
		ID: "id3",
		ServiceAnnotations: api.Annotations{
			Name: "name3",
		},
	}

	err = s1.Update(func(tx1 Tx) error {
		assert.NoError(t, UpdateTask(tx1, updateTask))
		return nil
	})
	assert.NoError(t, err)
	assert.NoError(t, Apply(s2, <-watcher))
	<-watcher // consume commit event

	s2.View(func(tx2 ReadTx) {
		assert.Equal(t, updateTask, GetTask(tx2, "id3"))
	})

	err = s1.Update(func(tx1 Tx) error {
		// Delete task
		assert.NoError(t, DeleteTask(tx1, "id1"))
		return nil
	})
	assert.NoError(t, err)
	assert.NoError(t, Apply(s2, <-watcher))
	<-watcher // consume commit event

	s2.View(func(tx2 ReadTx) {
		assert.Nil(t, GetTask(tx2, "id1"))
	})
}

func TestCustomIndex(t *testing.T) {
	s := NewMemoryStore(nil)
	assert.NotNil(t, s)

	setupTestStore(t, s)

	// Add a custom index entry to each node
	err := s.Update(func(tx Tx) error {
		allNodes, err := FindNodes(tx, All)
		assert.NoError(t, err)
		assert.Len(t, allNodes, len(nodeSet))

		for _, n := range allNodes {
			switch n.ID {
			case "id2":
				n.Spec.Annotations.Indices = []api.IndexEntry{
					{Key: "nodesbefore", Val: "id1"},
				}
				assert.NoError(t, UpdateNode(tx, n))
			case "id3":
				n.Spec.Annotations.Indices = []api.IndexEntry{
					{Key: "nodesbefore", Val: "id1"},
					{Key: "nodesbefore", Val: "id2"},
				}
				assert.NoError(t, UpdateNode(tx, n))
			}
		}
		return nil
	})
	assert.NoError(t, err)

	s.View(func(readTx ReadTx) {
		foundNodes, err := FindNodes(readTx, ByCustom("", "nodesbefore", "id2"))
		require.NoError(t, err)
		require.Len(t, foundNodes, 1)
		assert.Equal(t, "id3", foundNodes[0].ID)

		foundNodes, err = FindNodes(readTx, ByCustom("", "nodesbefore", "id1"))
		require.NoError(t, err)
		require.Len(t, foundNodes, 2)

		foundNodes, err = FindNodes(readTx, ByCustom("", "nodesbefore", "id3"))
		require.NoError(t, err)
		require.Len(t, foundNodes, 0)

		foundNodes, err = FindNodes(readTx, ByCustomPrefix("", "nodesbefore", "id"))
		require.NoError(t, err)
		require.Len(t, foundNodes, 2)

		foundNodes, err = FindNodes(readTx, ByCustomPrefix("", "nodesbefore", "id6"))
		require.NoError(t, err)
		require.Len(t, foundNodes, 0)
	})
}

func TestFailedTransaction(t *testing.T) {
	s := NewMemoryStore(nil)
	assert.NotNil(t, s)

	// Create one node
	err := s.Update(func(tx Tx) error {
		n := &api.Node{
			ID: "id1",
			Description: &api.NodeDescription{
				Hostname: "name1",
			},
		}

		assert.NoError(t, CreateNode(tx, n))
		return nil
	})
	assert.NoError(t, err)

	// Create a second node, but then roll back the transaction
	err = s.Update(func(tx Tx) error {
		n := &api.Node{
			ID: "id2",
			Description: &api.NodeDescription{
				Hostname: "name2",
			},
		}

		assert.NoError(t, CreateNode(tx, n))
		return errors.New("rollback")
	})
	assert.Error(t, err)

	s.View(func(tx ReadTx) {
		foundNodes, err := FindNodes(tx, All)
		assert.NoError(t, err)
		assert.Len(t, foundNodes, 1)
		foundNodes, err = FindNodes(tx, ByName("name1"))
		assert.NoError(t, err)
		assert.Len(t, foundNodes, 1)
		foundNodes, err = FindNodes(tx, ByName("name2"))
		assert.NoError(t, err)
		assert.Len(t, foundNodes, 0)
	})
}

func TestVersion(t *testing.T) {
	s := NewMemoryStore(&testutils.MockProposer{})
	assert.NotNil(t, s)

	var (
		retrievedNode  *api.Node
		retrievedNode2 *api.Node
	)

	// Create one node
	n := &api.Node{
		ID: "id1",
		Spec: api.NodeSpec{
			Annotations: api.Annotations{
				Name: "name1",
			},
		},
	}
	err := s.Update(func(tx Tx) error {
		assert.NoError(t, CreateNode(tx, n))
		return nil
	})
	assert.NoError(t, err)

	// Update the node using an object fetched from the store.
	n.Spec.Annotations.Name = "name2"
	err = s.Update(func(tx Tx) error {
		assert.NoError(t, UpdateNode(tx, n))
		retrievedNode = GetNode(tx, n.ID)
		return nil
	})
	assert.NoError(t, err)

	// Make sure the store is updating our local copy with the version.
	assert.Equal(t, n.Meta.Version, retrievedNode.Meta.Version)

	// Try again, this time using the retrieved node.
	retrievedNode.Spec.Annotations.Name = "name2"
	err = s.Update(func(tx Tx) error {
		assert.NoError(t, UpdateNode(tx, retrievedNode))
		retrievedNode2 = GetNode(tx, n.ID)
		return nil
	})
	assert.NoError(t, err)

	// Try to update retrievedNode again. This should fail because it was
	// already used to perform an update.
	retrievedNode.Spec.Annotations.Name = "name3"
	err = s.Update(func(tx Tx) error {
		assert.Equal(t, ErrSequenceConflict, UpdateNode(tx, n))
		return nil
	})
	assert.NoError(t, err)

	// But using retrievedNode2 should work, since it has the latest
	// sequence information.
	retrievedNode2.Spec.Annotations.Name = "name3"
	err = s.Update(func(tx Tx) error {
		assert.NoError(t, UpdateNode(tx, retrievedNode2))
		return nil
	})
	assert.NoError(t, err)
}

func TestTimestamps(t *testing.T) {
	s := NewMemoryStore(&testutils.MockProposer{})
	assert.NotNil(t, s)

	var (
		retrievedNode *api.Node
		updatedNode   *api.Node
	)

	// Create one node
	n := &api.Node{
		ID: "id1",
		Spec: api.NodeSpec{
			Annotations: api.Annotations{
				Name: "name1",
			},
		},
	}
	err := s.Update(func(tx Tx) error {
		assert.NoError(t, CreateNode(tx, n))
		return nil
	})
	assert.NoError(t, err)

	// Make sure our local copy got updated.
	assert.NotZero(t, n.Meta.CreatedAt)
	assert.NotZero(t, n.Meta.UpdatedAt)
	// Since this is a new node, CreatedAt should equal UpdatedAt.
	assert.Equal(t, n.Meta.CreatedAt, n.Meta.UpdatedAt)

	// Fetch the node from the store and make sure timestamps match.
	s.View(func(tx ReadTx) {
		retrievedNode = GetNode(tx, n.ID)
	})
	assert.Equal(t, retrievedNode.Meta.CreatedAt, n.Meta.CreatedAt)
	assert.Equal(t, retrievedNode.Meta.UpdatedAt, n.Meta.UpdatedAt)

	// Make an update.
	retrievedNode.Spec.Annotations.Name = "name2"
	err = s.Update(func(tx Tx) error {
		assert.NoError(t, UpdateNode(tx, retrievedNode))
		updatedNode = GetNode(tx, n.ID)
		return nil
	})
	assert.NoError(t, err)

	// Ensure `CreatedAt` is the same after the update and `UpdatedAt` got updated.
	assert.Equal(t, updatedNode.Meta.CreatedAt, n.Meta.CreatedAt)
	assert.NotEqual(t, updatedNode.Meta.CreatedAt, updatedNode.Meta.UpdatedAt)
}

func TestBatch(t *testing.T) {
	s := NewMemoryStore(&testutils.MockProposer{})
	assert.NotNil(t, s)

	watch, cancel := s.WatchQueue().Watch()
	defer cancel()

	// Create 405 nodes. Should get split across 3 transactions.
	err := s.Batch(func(batch *Batch) error {
		for i := 0; i != 2*MaxChangesPerTransaction+5; i++ {
			n := &api.Node{
				ID: "id" + strconv.Itoa(i),
				Spec: api.NodeSpec{
					Annotations: api.Annotations{
						Name: "name" + strconv.Itoa(i),
					},
				},
			}

			batch.Update(func(tx Tx) error {
				assert.NoError(t, CreateNode(tx, n))
				return nil
			})
		}

		return nil
	})
	assert.NoError(t, err)

	for i := 0; i != MaxChangesPerTransaction; i++ {
		event := <-watch
		if _, ok := event.(api.EventCreateNode); !ok {
			t.Fatalf("expected EventCreateNode; got %#v", event)
		}
	}
	event := <-watch
	if _, ok := event.(state.EventCommit); !ok {
		t.Fatalf("expected EventCommit; got %#v", event)
	}
	for i := 0; i != MaxChangesPerTransaction; i++ {
		event := <-watch
		if _, ok := event.(api.EventCreateNode); !ok {
			t.Fatalf("expected EventCreateNode; got %#v", event)
		}
	}
	event = <-watch
	if _, ok := event.(state.EventCommit); !ok {
		t.Fatalf("expected EventCommit; got %#v", event)
	}
	for i := 0; i != 5; i++ {
		event := <-watch
		if _, ok := event.(api.EventCreateNode); !ok {
			t.Fatalf("expected EventCreateNode; got %#v", event)
		}
	}
	event = <-watch
	if _, ok := event.(state.EventCommit); !ok {
		t.Fatalf("expected EventCommit; got %#v", event)
	}
}

func TestBatchFailure(t *testing.T) {
	s := NewMemoryStore(&testutils.MockProposer{})
	assert.NotNil(t, s)

	watch, cancel := s.WatchQueue().Watch()
	defer cancel()

	// Return an error partway through a transaction.
	err := s.Batch(func(batch *Batch) error {
		for i := 0; ; i++ {
			n := &api.Node{
				ID: "id" + strconv.Itoa(i),
				Spec: api.NodeSpec{
					Annotations: api.Annotations{
						Name: "name" + strconv.Itoa(i),
					},
				},
			}

			batch.Update(func(tx Tx) error {
				assert.NoError(t, CreateNode(tx, n))
				return nil
			})
			if i == MaxChangesPerTransaction+8 {
				return errors.New("failing the current tx")
			}
		}
	})
	assert.Error(t, err)

	for i := 0; i != MaxChangesPerTransaction; i++ {
		event := <-watch
		if _, ok := event.(api.EventCreateNode); !ok {
			t.Fatalf("expected EventCreateNode; got %#v", event)
		}
	}
	event := <-watch
	if _, ok := event.(state.EventCommit); !ok {
		t.Fatalf("expected EventCommit; got %#v", event)
	}

	// Shouldn't be anything after the first transaction
	select {
	case <-watch:
		t.Fatal("unexpected additional events")
	case <-time.After(50 * time.Millisecond):
	}
}

func TestStoreSaveRestore(t *testing.T) {
	s1 := NewMemoryStore(nil)
	assert.NotNil(t, s1)

	setupTestStore(t, s1)

	var snapshot *api.StoreSnapshot
	s1.View(func(tx ReadTx) {
		var err error
		snapshot, err = s1.Save(tx)
		assert.NoError(t, err)
	})

	s2 := NewMemoryStore(nil)
	assert.NotNil(t, s2)
	// setup s2 with the first element of each of the object sets (which should be
	// updated on restore), as well as one extraneous object (which should be deleted
	// on restore).  We also want to bump the version on all the ones that will be
	// updated just to make sure that restoration works.
	version := api.Version{Index: 100}
	c := clusterSet[0].Copy()
	c.Meta.Version = version
	n := nodeSet[0].Copy()
	n.Meta.Version = version
	s := serviceSet[0].Copy()
	s.Meta.Version = version
	task := taskSet[0].Copy()
	task.Meta.Version = version
	nw := networkSet[0].Copy()
	nw.Meta.Version = version
	cf := configSet[0].Copy()
	cf.Meta.Version = version
	sk := secretSet[0].Copy()
	sk.Meta.Version = version
	ext := extensionSet[0].Copy()
	ext.Meta.Version = version
	r := resourceSet[0].Copy()
	r.Meta.Version = version
	populateTestStore(t, s2,
		append(altClusterSet, c),
		append(altNodeSet, n),
		append(altServiceSet, s),
		append(altTaskSet, task),
		append(altNetworkSet, nw),
		append(altConfigSet, cf),
		append(altSecretSet, sk),
		append(altExtensionSet, ext),
		append(altResourceSet, r),
	)

	watcher, cancel, err := ViewAndWatch(s2, func(ReadTx) error {
		return nil
	})
	assert.NoError(t, err)
	defer cancel()

	err = s2.Restore(snapshot)
	assert.NoError(t, err)

	// s2 should end up looking just like s1
	s2.View(func(tx ReadTx) {
		allClusters, err := FindClusters(tx, All)
		assert.NoError(t, err)
		assert.Len(t, allClusters, len(clusterSet))
		for i := range allClusters {
			assert.Equal(t, allClusters[i], clusterSet[i])
		}

		allTasks, err := FindTasks(tx, All)
		assert.NoError(t, err)
		assert.Len(t, allTasks, len(taskSet))
		for i := range allTasks {
			assert.Equal(t, allTasks[i], taskSet[i])
		}

		allNodes, err := FindNodes(tx, All)
		assert.NoError(t, err)
		assert.Len(t, allNodes, len(nodeSet))
		for i := range allNodes {
			assert.Equal(t, allNodes[i], nodeSet[i])
		}

		allNetworks, err := FindNetworks(tx, All)
		assert.NoError(t, err)
		assert.Len(t, allNetworks, len(networkSet))
		for i := range allNetworks {
			assert.Equal(t, allNetworks[i], networkSet[i])
		}

		allServices, err := FindServices(tx, All)
		assert.NoError(t, err)
		assert.Len(t, allServices, len(serviceSet))
		for i := range allServices {
			assert.Equal(t, allServices[i], serviceSet[i])
		}

		allConfigs, err := FindConfigs(tx, All)
		assert.NoError(t, err)
		assert.Len(t, allConfigs, len(configSet))
		for i := range allConfigs {
			assert.Equal(t, allConfigs[i], configSet[i])
		}

		allSecrets, err := FindSecrets(tx, All)
		assert.NoError(t, err)
		assert.Len(t, allSecrets, len(secretSet))
		for i := range allSecrets {
			assert.Equal(t, allSecrets[i], secretSet[i])
		}

		allExtensions, err := FindExtensions(tx, All)
		assert.NoError(t, err)
		assert.Len(t, allExtensions, len(extensionSet))
		for i := range allExtensions {
			assert.Equal(t, allExtensions[i], extensionSet[i])
		}

		allResources, err := FindResources(tx, All)
		assert.NoError(t, err)
		assert.Len(t, allResources, len(resourceSet))
		for i := range allResources {
			assert.Equal(t, allResources[i], resourceSet[i])
		}
	})

	timeout := time.After(time.Second)

	// make sure we have 1 update event, 2 create events, and 1 delete event for each
	// object type
	var (
		clusterUpdates, clusterCreates, clusterDeletes,
		nodeUpdates, nodeCreates, nodeDeletes,
		serviceUpdates, serviceCreates, serviceDeletes,
		taskUpdates, taskCreates, taskDeletes,
		networkUpdates, networkCreates, networkDeletes,
		configUpdates, configCreates, configDeletes,
		secretUpdates, secretCreates, secretDeletes,
		extensionUpdates, extensionCreates, extensionDeletes,
		resourceUpdates, resourceCreates, resourceDeletes []api.StoreObject
	)

waitForAllEvents:
	for {
		var update events.Event
		select {
		case update = <-watcher:
		case <-timeout:
			assert.FailNow(t, "did not get all the events we were expecting after a snapshot was restored")
		}

		switch e := update.(type) {

		case api.EventUpdateCluster:
			clusterUpdates = append(clusterUpdates, e.Cluster)
		case api.EventCreateCluster:
			clusterCreates = append(clusterCreates, e.Cluster)
		case api.EventDeleteCluster:
			clusterDeletes = append(clusterDeletes, e.Cluster)

		case api.EventUpdateNode:
			nodeUpdates = append(nodeUpdates, e.Node)
		case api.EventCreateNode:
			nodeCreates = append(nodeCreates, e.Node)
		case api.EventDeleteNode:
			nodeDeletes = append(nodeDeletes, e.Node)

		case api.EventUpdateService:
			serviceUpdates = append(serviceUpdates, e.Service)
		case api.EventCreateService:
			serviceCreates = append(serviceCreates, e.Service)
		case api.EventDeleteService:
			serviceDeletes = append(serviceDeletes, e.Service)

		case api.EventUpdateTask:
			taskUpdates = append(taskUpdates, e.Task)
		case api.EventCreateTask:
			taskCreates = append(taskCreates, e.Task)
		case api.EventDeleteTask:
			taskDeletes = append(taskDeletes, e.Task)

		case api.EventUpdateNetwork:
			networkUpdates = append(networkUpdates, e.Network)
		case api.EventCreateNetwork:
			networkCreates = append(networkCreates, e.Network)
		case api.EventDeleteNetwork:
			networkDeletes = append(networkDeletes, e.Network)

		case api.EventUpdateConfig:
			configUpdates = append(configUpdates, e.Config)
		case api.EventCreateConfig:
			configCreates = append(configCreates, e.Config)
		case api.EventDeleteConfig:
			configDeletes = append(configDeletes, e.Config)

		case api.EventUpdateSecret:
			secretUpdates = append(secretUpdates, e.Secret)
		case api.EventCreateSecret:
			secretCreates = append(secretCreates, e.Secret)
		case api.EventDeleteSecret:
			secretDeletes = append(secretDeletes, e.Secret)

		case api.EventUpdateExtension:
			extensionUpdates = append(extensionUpdates, e.Extension)
		case api.EventCreateExtension:
			extensionCreates = append(extensionCreates, e.Extension)
		case api.EventDeleteExtension:
			extensionDeletes = append(extensionDeletes, e.Extension)

		case api.EventUpdateResource:
			resourceUpdates = append(resourceUpdates, e.Resource)
		case api.EventCreateResource:
			resourceCreates = append(resourceCreates, e.Resource)
		case api.EventDeleteResource:
			resourceDeletes = append(resourceDeletes, e.Resource)
		}

		// wait until we have all the events we want
		for _, x := range [][]api.StoreObject{
			clusterUpdates, clusterDeletes,
			nodeUpdates, nodeDeletes,
			serviceUpdates, serviceDeletes,
			taskUpdates, taskDeletes,
			networkUpdates, networkDeletes,
			configUpdates, configDeletes,
			secretUpdates, secretDeletes,
			extensionUpdates, extensionDeletes,
			resourceUpdates, resourceDeletes,
		} {
			if len(x) < 1 {
				continue waitForAllEvents
			}
		}

		for _, x := range [][]api.StoreObject{
			clusterCreates,
			nodeCreates,
			serviceCreates,
			taskCreates,
			networkCreates,
			configCreates,
			secretCreates,
			extensionCreates,
			resourceCreates,
		} {
			if len(x) < 2 {
				continue waitForAllEvents
			}
		}
		break
	}

	assertHasSameIDs := func(changes []api.StoreObject, expected ...api.StoreObject) {
		assert.Equal(t, len(expected), len(changes))
		expectedIDs := make(map[string]struct{})
		for _, s := range expected {
			expectedIDs[s.GetID()] = struct{}{}
		}
		for _, s := range changes {
			_, ok := expectedIDs[s.GetID()]
			assert.True(t, ok)
		}
	}

	assertHasSameIDs(clusterUpdates, clusterSet[0])
	assertHasSameIDs(clusterDeletes, altClusterSet[0])
	cantCastArrays := make([]api.StoreObject, len(clusterSet[1:]))
	for i, x := range clusterSet[1:] {
		cantCastArrays[i] = x
	}
	assertHasSameIDs(clusterCreates, cantCastArrays...)

	assertHasSameIDs(nodeUpdates, nodeSet[0])
	assertHasSameIDs(nodeDeletes, altNodeSet[0])
	cantCastArrays = make([]api.StoreObject, len(nodeSet[1:]))
	for i, x := range nodeSet[1:] {
		cantCastArrays[i] = x
	}
	assertHasSameIDs(nodeCreates, cantCastArrays...)

	assertHasSameIDs(serviceUpdates, serviceSet[0])
	assertHasSameIDs(serviceDeletes, altServiceSet[0])
	cantCastArrays = make([]api.StoreObject, len(serviceSet[1:]))
	for i, x := range serviceSet[1:] {
		cantCastArrays[i] = x
	}
	assertHasSameIDs(serviceCreates, cantCastArrays...)

	assertHasSameIDs(taskUpdates, taskSet[0])
	assertHasSameIDs(taskDeletes, altTaskSet[0])
	cantCastArrays = make([]api.StoreObject, len(taskSet[1:]))
	for i, x := range taskSet[1:] {
		cantCastArrays[i] = x
	}
	assertHasSameIDs(taskCreates, cantCastArrays...)

	assertHasSameIDs(networkUpdates, networkSet[0])
	assertHasSameIDs(networkDeletes, altNetworkSet[0])
	cantCastArrays = make([]api.StoreObject, len(networkSet[1:]))
	for i, x := range networkSet[1:] {
		cantCastArrays[i] = x
	}
	assertHasSameIDs(networkCreates, cantCastArrays...)

	assertHasSameIDs(configUpdates, configSet[0])
	assertHasSameIDs(configDeletes, altConfigSet[0])
	cantCastArrays = make([]api.StoreObject, len(configSet[1:]))
	for i, x := range configSet[1:] {
		cantCastArrays[i] = x
	}
	assertHasSameIDs(configCreates, cantCastArrays...)

	assertHasSameIDs(secretUpdates, secretSet[0])
	assertHasSameIDs(secretDeletes, altSecretSet[0])
	cantCastArrays = make([]api.StoreObject, len(secretSet[1:]))
	for i, x := range secretSet[1:] {
		cantCastArrays[i] = x
	}
	assertHasSameIDs(secretCreates, cantCastArrays...)

	assertHasSameIDs(extensionUpdates, extensionSet[0])
	assertHasSameIDs(extensionDeletes, altExtensionSet[0])
	cantCastArrays = make([]api.StoreObject, len(extensionSet[1:]))
	for i, x := range extensionSet[1:] {
		cantCastArrays[i] = x
	}
	assertHasSameIDs(extensionCreates, cantCastArrays...)

	assertHasSameIDs(resourceUpdates, resourceSet[0])
	assertHasSameIDs(resourceDeletes, altResourceSet[0])
	cantCastArrays = make([]api.StoreObject, len(resourceSet[1:]))
	for i, x := range resourceSet[1:] {
		cantCastArrays[i] = x
	}
	assertHasSameIDs(resourceCreates, cantCastArrays...)
}

func TestWatchFrom(t *testing.T) {
	s := NewMemoryStore(&testutils.MockProposer{})
	assert.NotNil(t, s)

	// Create a few nodes, 2 per transaction
	for i := 0; i != 5; i++ {
		err := s.Batch(func(batch *Batch) error {
			node := &api.Node{
				ID: "id" + strconv.Itoa(i),
				Spec: api.NodeSpec{
					Annotations: api.Annotations{
						Name: "name" + strconv.Itoa(i),
					},
				},
			}

			service := &api.Service{
				ID: "id" + strconv.Itoa(i),
				Spec: api.ServiceSpec{
					Annotations: api.Annotations{
						Name: "name" + strconv.Itoa(i),
					},
				},
			}

			batch.Update(func(tx Tx) error {
				assert.NoError(t, CreateNode(tx, node))
				return nil
			})
			batch.Update(func(tx Tx) error {
				assert.NoError(t, CreateService(tx, service))
				return nil
			})
			return nil
		})
		assert.NoError(t, err)
	}

	// Try to watch from an invalid index
	_, _, err := WatchFrom(s, &api.Version{Index: 5000})
	assert.Error(t, err)

	watch1, cancel1, err := WatchFrom(s, &api.Version{Index: 10}, api.EventCreateNode{}, state.EventCommit{})
	require.NoError(t, err)
	defer cancel1()

	for i := 0; i != 2; i++ {
		select {
		case event := <-watch1:
			nodeEvent, ok := event.(api.EventCreateNode)
			if !ok {
				t.Fatal("wrong event type - expected node create")
			}

			if i == 0 {
				assert.Equal(t, "id3", nodeEvent.Node.ID)
			} else {
				assert.Equal(t, "id4", nodeEvent.Node.ID)
			}
		case <-time.After(time.Second):
			t.Fatal("timed out waiting for event")
		}
		select {
		case event := <-watch1:
			if _, ok := event.(state.EventCommit); !ok {
				t.Fatal("wrong event type - expected commit")
			}
		case <-time.After(time.Second):
			t.Fatal("timed out waiting for event")
		}
	}

	watch2, cancel2, err := WatchFrom(s, &api.Version{Index: 13}, api.EventCreateService{}, state.EventCommit{})
	require.NoError(t, err)
	defer cancel2()

	select {
	case event := <-watch2:
		serviceEvent, ok := event.(api.EventCreateService)
		if !ok {
			t.Fatal("wrong event type - expected service create")
		}
		assert.Equal(t, "id4", serviceEvent.Service.ID)
	case <-time.After(time.Second):
		t.Fatal("timed out waiting for event")
	}
	select {
	case event := <-watch2:
		if _, ok := event.(state.EventCommit); !ok {
			t.Fatal("wrong event type - expected commit")
		}
	case <-time.After(time.Second):
		t.Fatal("timed out waiting for event")
	}

	// Create some new objects and make sure they show up in the watches.
	assert.NoError(t, s.Update(func(tx Tx) error {
		node := &api.Node{
			ID: "newnode",
			Spec: api.NodeSpec{
				Annotations: api.Annotations{
					Name: "newnode",
				},
			},
		}

		service := &api.Service{
			ID: "newservice",
			Spec: api.ServiceSpec{
				Annotations: api.Annotations{
					Name: "newservice",
				},
			},
		}

		assert.NoError(t, CreateNode(tx, node))
		assert.NoError(t, CreateService(tx, service))
		return nil
	}))

	select {
	case event := <-watch1:
		nodeEvent, ok := event.(api.EventCreateNode)
		if !ok {
			t.Fatalf("wrong event type - expected node create, got %T", event)
		}
		assert.Equal(t, "newnode", nodeEvent.Node.ID)
	case <-time.After(time.Second):
		t.Fatal("timed out waiting for event")
	}
	select {
	case event := <-watch1:
		if _, ok := event.(state.EventCommit); !ok {
			t.Fatal("wrong event type - expected commit")
		}
	case <-time.After(time.Second):
		t.Fatal("timed out waiting for event")
	}

	select {
	case event := <-watch2:
		serviceEvent, ok := event.(api.EventCreateService)
		if !ok {
			t.Fatalf("wrong event type - expected service create, got %T", event)
		}
		assert.Equal(t, "newservice", serviceEvent.Service.ID)
	case <-time.After(time.Second):
		t.Fatal("timed out waiting for event")
	}
	select {
	case event := <-watch2:
		if _, ok := event.(state.EventCommit); !ok {
			t.Fatal("wrong event type - expected commit")
		}
	case <-time.After(time.Second):
		t.Fatal("timed out waiting for event")
	}

	assert.NoError(t, s.Update(func(tx Tx) error {
		node := &api.Node{
			ID: "newnode2",
			Spec: api.NodeSpec{
				Annotations: api.Annotations{
					Name: "newnode2",
				},
			},
		}

		assert.NoError(t, CreateNode(tx, node))
		return nil
	}))

	select {
	case event := <-watch1:
		nodeEvent, ok := event.(api.EventCreateNode)
		if !ok {
			t.Fatalf("wrong event type - expected node create, got %T", event)
		}
		assert.Equal(t, "newnode2", nodeEvent.Node.ID)
	case <-time.After(time.Second):
		t.Fatal("timed out waiting for event")
	}
	select {
	case event := <-watch1:
		if _, ok := event.(state.EventCommit); !ok {
			t.Fatal("wrong event type - expected commit")
		}
	case <-time.After(time.Second):
		t.Fatal("timed out waiting for event")
	}

	select {
	case event := <-watch2:
		if _, ok := event.(state.EventCommit); !ok {
			t.Fatal("wrong event type - expected commit")
		}
	case <-time.After(time.Second):
		t.Fatal("timed out waiting for event")
	}
}

const benchmarkNumNodes = 10000

func setupNodes(b *testing.B, n int) (*MemoryStore, []string) {
	s := NewMemoryStore(nil)

	nodeIDs := make([]string, n)

	for i := 0; i < n; i++ {
		nodeIDs[i] = identity.NewID()
	}

	b.ResetTimer()

	_ = s.Update(func(tx1 Tx) error {
		for i := 0; i < n; i++ {
			_ = CreateNode(tx1, &api.Node{
				ID: nodeIDs[i],
				Spec: api.NodeSpec{
					Annotations: api.Annotations{
						Name: "name" + strconv.Itoa(i),
					},
				},
			})
		}
		return nil
	})

	return s, nodeIDs
}

func BenchmarkCreateNode(b *testing.B) {
	setupNodes(b, b.N)
}

func BenchmarkUpdateNode(b *testing.B) {
	s, nodeIDs := setupNodes(b, benchmarkNumNodes)
	b.ResetTimer()
	_ = s.Update(func(tx1 Tx) error {
		for i := 0; i < b.N; i++ {
			_ = UpdateNode(tx1, &api.Node{
				ID: nodeIDs[i%benchmarkNumNodes],
				Spec: api.NodeSpec{
					Annotations: api.Annotations{
						Name: nodeIDs[i%benchmarkNumNodes] + "_" + strconv.Itoa(i),
					},
				},
			})
		}
		return nil
	})
}

func BenchmarkUpdateNodeTransaction(b *testing.B) {
	s, nodeIDs := setupNodes(b, benchmarkNumNodes)
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = s.Update(func(tx1 Tx) error {
			_ = UpdateNode(tx1, &api.Node{
				ID: nodeIDs[i%benchmarkNumNodes],
				Spec: api.NodeSpec{
					Annotations: api.Annotations{
						Name: nodeIDs[i%benchmarkNumNodes] + "_" + strconv.Itoa(i),
					},
				},
			})
			return nil
		})
	}
}

func BenchmarkDeleteNodeTransaction(b *testing.B) {
	s, nodeIDs := setupNodes(b, benchmarkNumNodes)
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = s.Update(func(tx1 Tx) error {
			_ = DeleteNode(tx1, nodeIDs[0])
			// Don't actually commit deletions, so we can delete
			// things repeatedly to satisfy the benchmark structure.
			return errors.New("don't commit this")
		})
	}
}

func BenchmarkGetNode(b *testing.B) {
	s, nodeIDs := setupNodes(b, benchmarkNumNodes)
	b.ResetTimer()
	s.View(func(tx1 ReadTx) {
		for i := 0; i < b.N; i++ {
			_ = GetNode(tx1, nodeIDs[i%benchmarkNumNodes])
		}
	})
}

func BenchmarkFindAllNodes(b *testing.B) {
	s, _ := setupNodes(b, benchmarkNumNodes)
	b.ResetTimer()
	s.View(func(tx1 ReadTx) {
		for i := 0; i < b.N; i++ {
			_, _ = FindNodes(tx1, All)
		}
	})
}

func BenchmarkFindNodeByName(b *testing.B) {
	s, _ := setupNodes(b, benchmarkNumNodes)
	b.ResetTimer()
	s.View(func(tx1 ReadTx) {
		for i := 0; i < b.N; i++ {
			_, _ = FindNodes(tx1, ByName("name"+strconv.Itoa(i)))
		}
	})
}

func BenchmarkNodeConcurrency(b *testing.B) {
	s, nodeIDs := setupNodes(b, benchmarkNumNodes)
	b.ResetTimer()

	// Run 5 writer goroutines and 5 reader goroutines
	var wg sync.WaitGroup
	for c := 0; c != 5; c++ {
		wg.Add(1)
		go func(c int) {
			defer wg.Done()
			for i := 0; i < b.N; i++ {
				_ = s.Update(func(tx1 Tx) error {
					_ = UpdateNode(tx1, &api.Node{
						ID: nodeIDs[i%benchmarkNumNodes],
						Spec: api.NodeSpec{
							Annotations: api.Annotations{
								Name: nodeIDs[i%benchmarkNumNodes] + "_" + strconv.Itoa(c) + "_" + strconv.Itoa(i),
							},
						},
					})
					return nil
				})
			}
		}(c)
	}

	for c := 0; c != 5; c++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			s.View(func(tx1 ReadTx) {
				for i := 0; i < b.N; i++ {
					_ = GetNode(tx1, nodeIDs[i%benchmarkNumNodes])
				}
			})
		}()
	}

	wg.Wait()
}
