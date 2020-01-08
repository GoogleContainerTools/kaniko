package allocator

import (
	"net"
	"runtime/debug"
	"strconv"
	"testing"
	"time"

	"golang.org/x/net/context"

	"github.com/docker/go-events"
	"github.com/docker/swarmkit/api"
	"github.com/docker/swarmkit/manager/state"
	"github.com/docker/swarmkit/manager/state/store"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func init() {
	// set artificially low retry interval for testing
	retryInterval = 5 * time.Millisecond
}

func TestAllocator(t *testing.T) {
	s := store.NewMemoryStore(nil)
	assert.NotNil(t, s)
	defer s.Close()

	a, err := New(s, nil)
	assert.NoError(t, err)
	assert.NotNil(t, a)

	// Predefined node-local network
	p := &api.Network{
		ID: "one_unIque_id",
		Spec: api.NetworkSpec{
			Annotations: api.Annotations{
				Name: "pred_bridge_network",
				Labels: map[string]string{
					"com.docker.swarm.predefined": "true",
				},
			},
			DriverConfig: &api.Driver{Name: "bridge"},
		},
	}

	// Node-local swarm scope network
	nln := &api.Network{
		ID: "another_unIque_id",
		Spec: api.NetworkSpec{
			Annotations: api.Annotations{
				Name: "swarm-macvlan",
			},
			DriverConfig: &api.Driver{Name: "macvlan"},
		},
	}

	// Try adding some objects to store before allocator is started
	assert.NoError(t, s.Update(func(tx store.Tx) error {
		// populate ingress network
		in := &api.Network{
			ID: "ingress-nw-id",
			Spec: api.NetworkSpec{
				Annotations: api.Annotations{
					Name: "default-ingress",
				},
				Ingress: true,
			},
		}
		assert.NoError(t, store.CreateNetwork(tx, in))

		n1 := &api.Network{
			ID: "testID1",
			Spec: api.NetworkSpec{
				Annotations: api.Annotations{
					Name: "test1",
				},
			},
		}
		assert.NoError(t, store.CreateNetwork(tx, n1))

		s1 := &api.Service{
			ID: "testServiceID1",
			Spec: api.ServiceSpec{
				Annotations: api.Annotations{
					Name: "service1",
				},
				Task: api.TaskSpec{
					Networks: []*api.NetworkAttachmentConfig{
						{
							Target: "testID1",
						},
					},
				},
				Endpoint: &api.EndpointSpec{
					Mode: api.ResolutionModeVirtualIP,
					Ports: []*api.PortConfig{
						{
							Name:          "portName",
							Protocol:      api.ProtocolTCP,
							TargetPort:    8000,
							PublishedPort: 8001,
						},
					},
				},
			},
		}
		assert.NoError(t, store.CreateService(tx, s1))

		t1 := &api.Task{
			ID: "testTaskID1",
			Status: api.TaskStatus{
				State: api.TaskStateNew,
			},
			Networks: []*api.NetworkAttachment{
				{
					Network: n1,
				},
			},
		}
		assert.NoError(t, store.CreateTask(tx, t1))

		t2 := &api.Task{
			ID: "testTaskIDPreInit",
			Status: api.TaskStatus{
				State: api.TaskStateNew,
			},
			ServiceID:    "testServiceID1",
			DesiredState: api.TaskStateRunning,
		}
		assert.NoError(t, store.CreateTask(tx, t2))

		// Create the predefined node-local network with one service
		assert.NoError(t, store.CreateNetwork(tx, p))

		sp1 := &api.Service{
			ID: "predServiceID1",
			Spec: api.ServiceSpec{
				Annotations: api.Annotations{
					Name: "predService1",
				},
				Task: api.TaskSpec{
					Networks: []*api.NetworkAttachmentConfig{
						{
							Target: p.ID,
						},
					},
				},
				Endpoint: &api.EndpointSpec{Mode: api.ResolutionModeDNSRoundRobin},
			},
		}
		assert.NoError(t, store.CreateService(tx, sp1))

		tp1 := &api.Task{
			ID: "predTaskID1",
			Status: api.TaskStatus{
				State: api.TaskStateNew,
			},
			Networks: []*api.NetworkAttachment{
				{
					Network: p,
				},
			},
		}
		assert.NoError(t, store.CreateTask(tx, tp1))

		// Create the the swarm level node-local network with one service
		assert.NoError(t, store.CreateNetwork(tx, nln))

		sp2 := &api.Service{
			ID: "predServiceID2",
			Spec: api.ServiceSpec{
				Annotations: api.Annotations{
					Name: "predService2",
				},
				Task: api.TaskSpec{
					Networks: []*api.NetworkAttachmentConfig{
						{
							Target: nln.ID,
						},
					},
				},
				Endpoint: &api.EndpointSpec{Mode: api.ResolutionModeDNSRoundRobin},
			},
		}
		assert.NoError(t, store.CreateService(tx, sp2))

		tp2 := &api.Task{
			ID: "predTaskID2",
			Status: api.TaskStatus{
				State: api.TaskStateNew,
			},
			Networks: []*api.NetworkAttachment{
				{
					Network: nln,
				},
			},
		}
		assert.NoError(t, store.CreateTask(tx, tp2))

		return nil
	}))

	netWatch, cancel := state.Watch(s.WatchQueue(), api.EventUpdateNetwork{}, api.EventDeleteNetwork{})
	defer cancel()
	taskWatch, cancel := state.Watch(s.WatchQueue(), api.EventUpdateTask{}, api.EventDeleteTask{})
	defer cancel()
	serviceWatch, cancel := state.Watch(s.WatchQueue(), api.EventUpdateService{}, api.EventDeleteService{})
	defer cancel()

	// Start allocator
	go func() {
		assert.NoError(t, a.Run(context.Background()))
	}()
	defer a.Stop()

	// Now verify if we get network and tasks updated properly
	watchNetwork(t, netWatch, false, isValidNetwork)
	watchTask(t, s, taskWatch, false, isValidTask) // t1
	watchTask(t, s, taskWatch, false, isValidTask) // t2
	watchService(t, serviceWatch, false, nil)

	// Verify no allocation was done for the node-local networks
	var (
		ps *api.Network
		sn *api.Network
	)
	s.View(func(tx store.ReadTx) {
		ps = store.GetNetwork(tx, p.ID)
		sn = store.GetNetwork(tx, nln.ID)

	})
	assert.NotNil(t, ps)
	assert.NotNil(t, sn)
	// Verify no allocation was done for tasks on node-local networks
	var (
		tp1 *api.Task
		tp2 *api.Task
	)
	s.View(func(tx store.ReadTx) {
		tp1 = store.GetTask(tx, "predTaskID1")
		tp2 = store.GetTask(tx, "predTaskID2")
	})
	assert.NotNil(t, tp1)
	assert.NotNil(t, tp2)
	assert.Equal(t, tp1.Networks[0].Network.ID, p.ID)
	assert.Equal(t, tp2.Networks[0].Network.ID, nln.ID)
	assert.Nil(t, tp1.Networks[0].Addresses, "Non nil addresses for task on node-local network")
	assert.Nil(t, tp2.Networks[0].Addresses, "Non nil addresses for task on node-local network")

	// Add new networks/tasks/services after allocator is started.
	assert.NoError(t, s.Update(func(tx store.Tx) error {
		n2 := &api.Network{
			ID: "testID2",
			Spec: api.NetworkSpec{
				Annotations: api.Annotations{
					Name: "test2",
				},
			},
		}
		assert.NoError(t, store.CreateNetwork(tx, n2))
		return nil
	}))

	watchNetwork(t, netWatch, false, isValidNetwork)

	assert.NoError(t, s.Update(func(tx store.Tx) error {
		s2 := &api.Service{
			ID: "testServiceID2",
			Spec: api.ServiceSpec{
				Annotations: api.Annotations{
					Name: "service2",
				},
				Networks: []*api.NetworkAttachmentConfig{
					{
						Target: "testID2",
					},
				},
				Endpoint: &api.EndpointSpec{},
			},
		}
		assert.NoError(t, store.CreateService(tx, s2))
		return nil
	}))

	watchService(t, serviceWatch, false, nil)

	assert.NoError(t, s.Update(func(tx store.Tx) error {
		t2 := &api.Task{
			ID: "testTaskID2",
			Status: api.TaskStatus{
				State: api.TaskStateNew,
			},
			ServiceID:    "testServiceID2",
			DesiredState: api.TaskStateRunning,
		}
		assert.NoError(t, store.CreateTask(tx, t2))
		return nil
	}))

	watchTask(t, s, taskWatch, false, isValidTask)

	// Now try adding a task which depends on a network before adding the network.
	n3 := &api.Network{
		ID: "testID3",
		Spec: api.NetworkSpec{
			Annotations: api.Annotations{
				Name: "test3",
			},
		},
	}

	assert.NoError(t, s.Update(func(tx store.Tx) error {
		t3 := &api.Task{
			ID: "testTaskID3",
			Status: api.TaskStatus{
				State: api.TaskStateNew,
			},
			DesiredState: api.TaskStateRunning,
			Networks: []*api.NetworkAttachment{
				{
					Network: n3,
				},
			},
		}
		assert.NoError(t, store.CreateTask(tx, t3))
		return nil
	}))

	// Wait for a little bit of time before adding network just to
	// test network is not available while task allocation is
	// going through
	time.Sleep(10 * time.Millisecond)

	assert.NoError(t, s.Update(func(tx store.Tx) error {
		assert.NoError(t, store.CreateNetwork(tx, n3))
		return nil
	}))

	watchNetwork(t, netWatch, false, isValidNetwork)
	watchTask(t, s, taskWatch, false, isValidTask)

	assert.NoError(t, s.Update(func(tx store.Tx) error {
		assert.NoError(t, store.DeleteTask(tx, "testTaskID3"))
		return nil
	}))
	watchTask(t, s, taskWatch, false, isValidTask)

	assert.NoError(t, s.Update(func(tx store.Tx) error {
		t5 := &api.Task{
			ID: "testTaskID5",
			Spec: api.TaskSpec{
				Networks: []*api.NetworkAttachmentConfig{
					{
						Target: "testID2",
					},
				},
			},
			Status: api.TaskStatus{
				State: api.TaskStateNew,
			},
			DesiredState: api.TaskStateRunning,
			ServiceID:    "testServiceID2",
		}
		assert.NoError(t, store.CreateTask(tx, t5))
		return nil
	}))
	watchTask(t, s, taskWatch, false, isValidTask)

	assert.NoError(t, s.Update(func(tx store.Tx) error {
		assert.NoError(t, store.DeleteNetwork(tx, "testID3"))
		return nil
	}))
	watchNetwork(t, netWatch, false, isValidNetwork)

	assert.NoError(t, s.Update(func(tx store.Tx) error {
		assert.NoError(t, store.DeleteService(tx, "testServiceID2"))
		return nil
	}))
	watchService(t, serviceWatch, false, nil)

	// Try to create a task with no network attachments and test
	// that it moves to ALLOCATED state.
	assert.NoError(t, s.Update(func(tx store.Tx) error {
		t4 := &api.Task{
			ID: "testTaskID4",
			Status: api.TaskStatus{
				State: api.TaskStateNew,
			},
			DesiredState: api.TaskStateRunning,
		}
		assert.NoError(t, store.CreateTask(tx, t4))
		return nil
	}))
	watchTask(t, s, taskWatch, false, isValidTask)

	assert.NoError(t, s.Update(func(tx store.Tx) error {
		n2 := store.GetNetwork(tx, "testID2")
		require.NotEqual(t, nil, n2)
		assert.NoError(t, store.UpdateNetwork(tx, n2))
		return nil
	}))
	watchNetwork(t, netWatch, false, isValidNetwork)
	watchNetwork(t, netWatch, true, nil)

	// Try updating service which is already allocated with no endpointSpec
	assert.NoError(t, s.Update(func(tx store.Tx) error {
		s := store.GetService(tx, "testServiceID1")
		s.Spec.Endpoint = nil

		assert.NoError(t, store.UpdateService(tx, s))
		return nil
	}))
	watchService(t, serviceWatch, false, nil)

	// Try updating task which is already allocated
	assert.NoError(t, s.Update(func(tx store.Tx) error {
		t2 := store.GetTask(tx, "testTaskID2")
		require.NotEqual(t, nil, t2)
		assert.NoError(t, store.UpdateTask(tx, t2))
		return nil
	}))
	watchTask(t, s, taskWatch, false, isValidTask)
	watchTask(t, s, taskWatch, true, nil)

	// Try adding networks with conflicting network resources and
	// add task which attaches to a network which gets allocated
	// later and verify if task reconciles and moves to ALLOCATED.
	n4 := &api.Network{
		ID: "testID4",
		Spec: api.NetworkSpec{
			Annotations: api.Annotations{
				Name: "test4",
			},
			DriverConfig: &api.Driver{
				Name: "overlay",
				Options: map[string]string{
					"com.docker.network.driver.overlay.vxlanid_list": "328",
				},
			},
		},
	}

	n5 := n4.Copy()
	n5.ID = "testID5"
	n5.Spec.Annotations.Name = "test5"
	assert.NoError(t, s.Update(func(tx store.Tx) error {
		assert.NoError(t, store.CreateNetwork(tx, n4))
		return nil
	}))
	watchNetwork(t, netWatch, false, isValidNetwork)

	assert.NoError(t, s.Update(func(tx store.Tx) error {
		assert.NoError(t, store.CreateNetwork(tx, n5))
		return nil
	}))
	watchNetwork(t, netWatch, true, nil)

	assert.NoError(t, s.Update(func(tx store.Tx) error {
		t6 := &api.Task{
			ID: "testTaskID6",
			Status: api.TaskStatus{
				State: api.TaskStateNew,
			},
			DesiredState: api.TaskStateRunning,
			Networks: []*api.NetworkAttachment{
				{
					Network: n5,
				},
			},
		}
		assert.NoError(t, store.CreateTask(tx, t6))
		return nil
	}))
	watchTask(t, s, taskWatch, true, nil)

	// Now remove the conflicting network.
	assert.NoError(t, s.Update(func(tx store.Tx) error {
		assert.NoError(t, store.DeleteNetwork(tx, n4.ID))
		return nil
	}))
	watchNetwork(t, netWatch, false, isValidNetwork)
	watchTask(t, s, taskWatch, false, isValidTask)

	// Try adding services with conflicting port configs and add
	// task which is part of the service whose allocation hasn't
	// happened and when that happens later and verify if task
	// reconciles and moves to ALLOCATED.
	s3 := &api.Service{
		ID: "testServiceID3",
		Spec: api.ServiceSpec{
			Annotations: api.Annotations{
				Name: "service3",
			},
			Endpoint: &api.EndpointSpec{
				Ports: []*api.PortConfig{
					{
						Name:          "http",
						TargetPort:    80,
						PublishedPort: 8080,
					},
					{
						PublishMode: api.PublishModeHost,
						Name:        "http",
						TargetPort:  80,
					},
				},
			},
		},
	}

	s4 := s3.Copy()
	s4.ID = "testServiceID4"
	s4.Spec.Annotations.Name = "service4"
	assert.NoError(t, s.Update(func(tx store.Tx) error {
		assert.NoError(t, store.CreateService(tx, s3))
		return nil
	}))
	watchService(t, serviceWatch, false, nil)
	assert.NoError(t, s.Update(func(tx store.Tx) error {
		assert.NoError(t, store.CreateService(tx, s4))
		return nil
	}))
	watchService(t, serviceWatch, true, nil)

	assert.NoError(t, s.Update(func(tx store.Tx) error {
		t7 := &api.Task{
			ID: "testTaskID7",
			Status: api.TaskStatus{
				State: api.TaskStateNew,
			},
			ServiceID:    "testServiceID4",
			DesiredState: api.TaskStateRunning,
		}
		assert.NoError(t, store.CreateTask(tx, t7))
		return nil
	}))
	watchTask(t, s, taskWatch, true, nil)

	// Now remove the conflicting service.
	assert.NoError(t, s.Update(func(tx store.Tx) error {
		assert.NoError(t, store.DeleteService(tx, s3.ID))
		return nil
	}))
	watchService(t, serviceWatch, false, nil)
	watchTask(t, s, taskWatch, false, isValidTask)
}

func TestNoDuplicateIPs(t *testing.T) {
	s := store.NewMemoryStore(nil)
	assert.NotNil(t, s)
	defer s.Close()

	// Try adding some objects to store before allocator is started
	assert.NoError(t, s.Update(func(tx store.Tx) error {
		// populate ingress network
		in := &api.Network{
			ID: "ingress-nw-id",
			Spec: api.NetworkSpec{
				Annotations: api.Annotations{
					Name: "default-ingress",
				},
				Ingress: true,
			},
			IPAM: &api.IPAMOptions{
				Driver: &api.Driver{},
				Configs: []*api.IPAMConfig{
					{
						Subnet:  "10.0.0.0/24",
						Gateway: "10.0.0.1",
					},
				},
			},
			DriverState: &api.Driver{},
		}
		assert.NoError(t, store.CreateNetwork(tx, in))
		n1 := &api.Network{
			ID: "testID1",
			Spec: api.NetworkSpec{
				Annotations: api.Annotations{
					Name: "test1",
				},
			},
			IPAM: &api.IPAMOptions{
				Driver: &api.Driver{},
				Configs: []*api.IPAMConfig{
					{
						Subnet:  "10.1.0.0/24",
						Gateway: "10.1.0.1",
					},
				},
			},
			DriverState: &api.Driver{},
		}
		assert.NoError(t, store.CreateNetwork(tx, n1))

		s1 := &api.Service{
			ID: "testServiceID1",
			Spec: api.ServiceSpec{
				Annotations: api.Annotations{
					Name: "service1",
				},
				Task: api.TaskSpec{
					Networks: []*api.NetworkAttachmentConfig{
						{
							Target: "testID1",
						},
					},
				},
				Endpoint: &api.EndpointSpec{
					Mode: api.ResolutionModeVirtualIP,
					Ports: []*api.PortConfig{
						{
							Name:          "portName",
							Protocol:      api.ProtocolTCP,
							TargetPort:    8000,
							PublishedPort: 8001,
						},
					},
				},
			},
		}
		assert.NoError(t, store.CreateService(tx, s1))

		return nil
	}))

	taskWatch, cancel := state.Watch(s.WatchQueue(), api.EventUpdateTask{}, api.EventDeleteTask{})
	defer cancel()

	assignedIPs := make(map[string]string)
	hasUniqueIP := func(fakeT assert.TestingT, s *store.MemoryStore, task *api.Task) bool {
		if len(task.Networks) == 0 {
			panic("missing networks")
		}
		if len(task.Networks[0].Addresses) == 0 {
			panic("missing network address")
		}

		assignedIP := task.Networks[0].Addresses[0]
		oldTaskID, present := assignedIPs[assignedIP]
		if present && task.ID != oldTaskID {
			t.Fatalf("task %s assigned duplicate IP %s, previously assigned to task %s", task.ID, assignedIP, oldTaskID)
		}
		assignedIPs[assignedIP] = task.ID
		return true
	}

	reps := 100
	for i := 0; i != reps; i++ {
		assert.NoError(t, s.Update(func(tx store.Tx) error {
			t2 := &api.Task{
				// The allocator iterates over the tasks in
				// lexical order, so number tasks in descending
				// order. Note that the problem this test was
				// meant to trigger also showed up with tasks
				// numbered in ascending order, but it took
				// until the 52nd task.
				ID: "testTaskID" + strconv.Itoa(reps-i),
				Status: api.TaskStatus{
					State: api.TaskStateNew,
				},
				ServiceID:    "testServiceID1",
				DesiredState: api.TaskStateRunning,
			}
			assert.NoError(t, store.CreateTask(tx, t2))

			return nil
		}))
		a, err := New(s, nil)
		assert.NoError(t, err)
		assert.NotNil(t, a)

		// Start allocator
		go func() {
			assert.NoError(t, a.Run(context.Background()))
		}()

		// Confirm task gets a unique IP
		watchTask(t, s, taskWatch, false, hasUniqueIP)
		a.Stop()
	}
}

func TestAllocatorRestoreForDuplicateIPs(t *testing.T) {
	s := store.NewMemoryStore(nil)
	assert.NotNil(t, s)
	defer s.Close()
	// Create 3 services with 1 task each
	numsvcstsks := 3
	assert.NoError(t, s.Update(func(tx store.Tx) error {
		// populate ingress network
		in := &api.Network{
			ID: "ingress-nw-id",
			Spec: api.NetworkSpec{
				Annotations: api.Annotations{
					Name: "default-ingress",
				},
				Ingress: true,
			},
			IPAM: &api.IPAMOptions{
				Driver: &api.Driver{},
				Configs: []*api.IPAMConfig{
					{
						Subnet:  "10.0.0.0/24",
						Gateway: "10.0.0.1",
					},
				},
			},
		}
		assert.NoError(t, store.CreateNetwork(tx, in))

		for i := 0; i != numsvcstsks; i++ {
			svc := &api.Service{
				ID: "testServiceID" + strconv.Itoa(i),
				Spec: api.ServiceSpec{
					Annotations: api.Annotations{
						Name: "service" + strconv.Itoa(i),
					},
					Endpoint: &api.EndpointSpec{
						Mode: api.ResolutionModeVirtualIP,

						Ports: []*api.PortConfig{
							{
								Name:          "",
								Protocol:      api.ProtocolTCP,
								TargetPort:    8000,
								PublishedPort: uint32(8001 + i),
							},
						},
					},
				},
				Endpoint: &api.Endpoint{
					Ports: []*api.PortConfig{
						{
							Name:          "",
							Protocol:      api.ProtocolTCP,
							TargetPort:    8000,
							PublishedPort: uint32(8001 + i),
						},
					},
					VirtualIPs: []*api.Endpoint_VirtualIP{
						{
							NetworkID: "ingress-nw-id",
							Addr:      "10.0.0." + strconv.Itoa(2+i) + "/24",
						},
					},
				},
			}
			assert.NoError(t, store.CreateService(tx, svc))
		}
		return nil
	}))

	for i := 0; i != numsvcstsks; i++ {
		assert.NoError(t, s.Update(func(tx store.Tx) error {
			tsk := &api.Task{
				ID: "testTaskID" + strconv.Itoa(i),
				Status: api.TaskStatus{
					State: api.TaskStateNew,
				},
				ServiceID:    "testServiceID" + strconv.Itoa(i),
				DesiredState: api.TaskStateRunning,
			}
			assert.NoError(t, store.CreateTask(tx, tsk))
			return nil
		}))
	}

	assignedVIPs := make(map[string]bool)
	assignedIPs := make(map[string]bool)
	hasNoIPOverlapServices := func(fakeT assert.TestingT, service *api.Service) bool {
		assert.NotEqual(fakeT, len(service.Endpoint.VirtualIPs), 0)
		assert.NotEqual(fakeT, len(service.Endpoint.VirtualIPs[0].Addr), 0)

		assignedVIP := service.Endpoint.VirtualIPs[0].Addr
		if assignedVIPs[assignedVIP] {
			t.Fatalf("service %s assigned duplicate IP %s", service.ID, assignedVIP)
		}
		assignedVIPs[assignedVIP] = true
		if assignedIPs[assignedVIP] {
			t.Fatalf("a task and service %s have the same IP %s", service.ID, assignedVIP)
		}
		return true
	}

	hasNoIPOverlapTasks := func(fakeT assert.TestingT, s *store.MemoryStore, task *api.Task) bool {
		assert.NotEqual(fakeT, len(task.Networks), 0)
		assert.NotEqual(fakeT, len(task.Networks[0].Addresses), 0)

		assignedIP := task.Networks[0].Addresses[0]
		if assignedIPs[assignedIP] {
			t.Fatalf("task %s assigned duplicate IP %s", task.ID, assignedIP)
		}
		assignedIPs[assignedIP] = true
		if assignedVIPs[assignedIP] {
			t.Fatalf("a service and task %s have the same IP %s", task.ID, assignedIP)
		}
		return true
	}

	a, err := New(s, nil)
	assert.NoError(t, err)
	assert.NotNil(t, a)
	// Start allocator
	go func() {
		assert.NoError(t, a.Run(context.Background()))
	}()
	defer a.Stop()

	taskWatch, cancel := state.Watch(s.WatchQueue(), api.EventUpdateTask{}, api.EventDeleteTask{})
	defer cancel()

	serviceWatch, cancel := state.Watch(s.WatchQueue(), api.EventUpdateService{}, api.EventDeleteService{})
	defer cancel()

	// Confirm tasks have no IPs that overlap with the services VIPs on restart
	for i := 0; i != numsvcstsks; i++ {
		watchTask(t, s, taskWatch, false, hasNoIPOverlapTasks)
		watchService(t, serviceWatch, false, hasNoIPOverlapServices)
	}
}

// TestAllocatorRestartNoEndpointSpec covers the leader election case when the service Spec
// does not contain the EndpointSpec.
// The expected behavior is that the VIP(s) are still correctly populated inside
// the IPAM and that no configuration on the service is changed.
func TestAllocatorRestartNoEndpointSpec(t *testing.T) {
	s := store.NewMemoryStore(nil)
	assert.NotNil(t, s)
	defer s.Close()
	// Create 3 services with 1 task each
	numsvcstsks := 3
	assert.NoError(t, s.Update(func(tx store.Tx) error {
		// populate ingress network
		in := &api.Network{
			ID: "overlay1",
			Spec: api.NetworkSpec{
				Annotations: api.Annotations{
					Name: "net1",
				},
			},
			IPAM: &api.IPAMOptions{
				Driver: &api.Driver{},
				Configs: []*api.IPAMConfig{
					{
						Subnet:  "10.0.0.0/24",
						Gateway: "10.0.0.1",
					},
				},
			},
			DriverState: &api.Driver{},
		}
		assert.NoError(t, store.CreateNetwork(tx, in))

		for i := 0; i != numsvcstsks; i++ {
			svc := &api.Service{
				ID: "testServiceID" + strconv.Itoa(i),
				Spec: api.ServiceSpec{
					Annotations: api.Annotations{
						Name: "service" + strconv.Itoa(i),
					},
					// Endpoint: &api.EndpointSpec{
					// 	Mode: api.ResolutionModeVirtualIP,
					// },
					Task: api.TaskSpec{
						Networks: []*api.NetworkAttachmentConfig{
							{
								Target: "overlay1",
							},
						},
					},
				},
				Endpoint: &api.Endpoint{
					Spec: &api.EndpointSpec{
						Mode: api.ResolutionModeVirtualIP,
					},
					VirtualIPs: []*api.Endpoint_VirtualIP{
						{
							NetworkID: "overlay1",
							Addr:      "10.0.0." + strconv.Itoa(2+2*i) + "/24",
						},
					},
				},
			}
			assert.NoError(t, store.CreateService(tx, svc))
		}
		return nil
	}))

	for i := 0; i != numsvcstsks; i++ {
		assert.NoError(t, s.Update(func(tx store.Tx) error {
			tsk := &api.Task{
				ID: "testTaskID" + strconv.Itoa(i),
				Status: api.TaskStatus{
					State: api.TaskStateNew,
				},
				ServiceID:    "testServiceID" + strconv.Itoa(i),
				DesiredState: api.TaskStateRunning,
				Networks: []*api.NetworkAttachment{
					{
						Network: &api.Network{
							ID: "overlay1",
						},
					},
				},
			}
			assert.NoError(t, store.CreateTask(tx, tsk))
			return nil
		}))
	}

	expectedIPs := map[string]string{
		"testServiceID0": "10.0.0.2/24",
		"testServiceID1": "10.0.0.4/24",
		"testServiceID2": "10.0.0.6/24",
		"testTaskID0":    "10.0.0.3/24",
		"testTaskID1":    "10.0.0.5/24",
		"testTaskID2":    "10.0.0.7/24",
	}
	assignedIPs := make(map[string]bool)
	hasNoIPOverlapServices := func(fakeT assert.TestingT, service *api.Service) bool {
		assert.NotEqual(fakeT, len(service.Endpoint.VirtualIPs), 0)
		assert.NotEqual(fakeT, len(service.Endpoint.VirtualIPs[0].Addr), 0)
		assignedVIP := service.Endpoint.VirtualIPs[0].Addr
		if assignedIPs[assignedVIP] {
			t.Fatalf("service %s assigned duplicate IP %s", service.ID, assignedVIP)
		}
		assignedIPs[assignedVIP] = true
		ip, ok := expectedIPs[service.ID]
		assert.True(t, ok)
		assert.Equal(t, ip, assignedVIP)
		delete(expectedIPs, service.ID)
		return true
	}

	hasNoIPOverlapTasks := func(fakeT assert.TestingT, s *store.MemoryStore, task *api.Task) bool {
		assert.NotEqual(fakeT, len(task.Networks), 0)
		assert.NotEqual(fakeT, len(task.Networks[0].Addresses), 0)
		assignedIP := task.Networks[0].Addresses[0]
		if assignedIPs[assignedIP] {
			t.Fatalf("task %s assigned duplicate IP %s", task.ID, assignedIP)
		}
		assignedIPs[assignedIP] = true
		ip, ok := expectedIPs[task.ID]
		assert.True(t, ok)
		assert.Equal(t, ip, assignedIP)
		delete(expectedIPs, task.ID)
		return true
	}

	a, err := New(s, nil)
	assert.NoError(t, err)
	assert.NotNil(t, a)
	// Start allocator
	go func() {
		assert.NoError(t, a.Run(context.Background()))
	}()
	defer a.Stop()

	taskWatch, cancel := state.Watch(s.WatchQueue(), api.EventUpdateTask{}, api.EventDeleteTask{})
	defer cancel()

	serviceWatch, cancel := state.Watch(s.WatchQueue(), api.EventUpdateService{}, api.EventDeleteService{})
	defer cancel()

	// Confirm tasks have no IPs that overlap with the services VIPs on restart
	for i := 0; i != numsvcstsks; i++ {
		watchTask(t, s, taskWatch, false, hasNoIPOverlapTasks)
		watchService(t, serviceWatch, false, hasNoIPOverlapServices)
	}
	assert.Len(t, expectedIPs, 0)
}

// TestAllocatorRestoreForUnallocatedNetwork tests allocator restart
// scenarios where there is a combination of allocated and unallocated
// networks and tests whether the restore logic ensures the networks
// services and tasks that were preallocated are allocated correctly
// followed by the allocation of unallocated networks prior to the
// restart.
func TestAllocatorRestoreForUnallocatedNetwork(t *testing.T) {
	s := store.NewMemoryStore(nil)
	assert.NotNil(t, s)
	defer s.Close()
	// Create 3 services with 1 task each
	numsvcstsks := 3
	var n1 *api.Network
	var n2 *api.Network
	assert.NoError(t, s.Update(func(tx store.Tx) error {
		// populate ingress network
		in := &api.Network{
			ID: "ingress-nw-id",
			Spec: api.NetworkSpec{
				Annotations: api.Annotations{
					Name: "default-ingress",
				},
				Ingress: true,
			},
			IPAM: &api.IPAMOptions{
				Driver: &api.Driver{},
				Configs: []*api.IPAMConfig{
					{
						Subnet:  "10.0.0.0/24",
						Gateway: "10.0.0.1",
					},
				},
			},
		}
		assert.NoError(t, store.CreateNetwork(tx, in))

		n1 = &api.Network{
			ID: "testID1",
			Spec: api.NetworkSpec{
				Annotations: api.Annotations{
					Name: "test1",
				},
			},
			IPAM: &api.IPAMOptions{
				Driver: &api.Driver{},
				Configs: []*api.IPAMConfig{
					{
						Subnet:  "10.1.0.0/24",
						Gateway: "10.1.0.1",
					},
				},
			},
			DriverState: &api.Driver{},
		}
		assert.NoError(t, store.CreateNetwork(tx, n1))

		n2 = &api.Network{
			// Intentionally named testID0 so that in restore this network
			// is looked into first
			ID: "testID0",
			Spec: api.NetworkSpec{
				Annotations: api.Annotations{
					Name: "test2",
				},
			},
		}
		assert.NoError(t, store.CreateNetwork(tx, n2))

		for i := 0; i != numsvcstsks; i++ {
			svc := &api.Service{
				ID: "testServiceID" + strconv.Itoa(i),
				Spec: api.ServiceSpec{
					Annotations: api.Annotations{
						Name: "service" + strconv.Itoa(i),
					},
					Task: api.TaskSpec{
						Networks: []*api.NetworkAttachmentConfig{
							{
								Target: "testID1",
							},
						},
					},
					Endpoint: &api.EndpointSpec{
						Mode: api.ResolutionModeVirtualIP,
						Ports: []*api.PortConfig{
							{
								Name:          "",
								Protocol:      api.ProtocolTCP,
								TargetPort:    8000,
								PublishedPort: uint32(8001 + i),
							},
						},
					},
				},
				Endpoint: &api.Endpoint{
					Ports: []*api.PortConfig{
						{
							Name:          "",
							Protocol:      api.ProtocolTCP,
							TargetPort:    8000,
							PublishedPort: uint32(8001 + i),
						},
					},
					VirtualIPs: []*api.Endpoint_VirtualIP{
						{
							NetworkID: "ingress-nw-id",
							Addr:      "10.0.0." + strconv.Itoa(2+i) + "/24",
						},
						{
							NetworkID: "testID1",
							Addr:      "10.1.0." + strconv.Itoa(2+i) + "/24",
						},
					},
				},
			}
			assert.NoError(t, store.CreateService(tx, svc))
		}
		return nil
	}))

	for i := 0; i != numsvcstsks; i++ {
		assert.NoError(t, s.Update(func(tx store.Tx) error {
			tsk := &api.Task{
				ID: "testTaskID" + strconv.Itoa(i),
				Status: api.TaskStatus{
					State: api.TaskStateNew,
				},
				Spec: api.TaskSpec{
					Networks: []*api.NetworkAttachmentConfig{
						{
							Target: "testID1",
						},
					},
				},
				ServiceID:    "testServiceID" + strconv.Itoa(i),
				DesiredState: api.TaskStateRunning,
			}
			assert.NoError(t, store.CreateTask(tx, tsk))
			return nil
		}))
	}

	assignedIPs := make(map[string]bool)
	expectedIPs := map[string]string{
		"testServiceID0": "10.1.0.2/24",
		"testServiceID1": "10.1.0.3/24",
		"testServiceID2": "10.1.0.4/24",
		"testTaskID0":    "10.1.0.5/24",
		"testTaskID1":    "10.1.0.6/24",
		"testTaskID2":    "10.1.0.7/24",
	}
	hasNoIPOverlapServices := func(fakeT assert.TestingT, service *api.Service) bool {
		assert.NotEqual(fakeT, len(service.Endpoint.VirtualIPs), 0)
		assert.NotEqual(fakeT, len(service.Endpoint.VirtualIPs[0].Addr), 0)
		assignedVIP := service.Endpoint.VirtualIPs[1].Addr
		if assignedIPs[assignedVIP] {
			t.Fatalf("service %s assigned duplicate IP %s", service.ID, assignedVIP)
		}
		assignedIPs[assignedVIP] = true
		ip, ok := expectedIPs[service.ID]
		assert.True(t, ok)
		assert.Equal(t, ip, assignedVIP)
		delete(expectedIPs, service.ID)
		return true
	}

	hasNoIPOverlapTasks := func(fakeT assert.TestingT, s *store.MemoryStore, task *api.Task) bool {
		assert.NotEqual(fakeT, len(task.Networks), 0)
		assert.NotEqual(fakeT, len(task.Networks[0].Addresses), 0)
		assignedIP := task.Networks[1].Addresses[0]
		if assignedIPs[assignedIP] {
			t.Fatalf("task %s assigned duplicate IP %s", task.ID, assignedIP)
		}
		assignedIPs[assignedIP] = true
		ip, ok := expectedIPs[task.ID]
		assert.True(t, ok)
		assert.Equal(t, ip, assignedIP)
		delete(expectedIPs, task.ID)
		return true
	}

	a, err := New(s, nil)
	assert.NoError(t, err)
	assert.NotNil(t, a)
	// Start allocator
	go func() {
		assert.NoError(t, a.Run(context.Background()))
	}()
	defer a.Stop()

	taskWatch, cancel := state.Watch(s.WatchQueue(), api.EventUpdateTask{}, api.EventDeleteTask{})
	defer cancel()

	serviceWatch, cancel := state.Watch(s.WatchQueue(), api.EventUpdateService{}, api.EventDeleteService{})
	defer cancel()

	// Confirm tasks have no IPs that overlap with the services VIPs on restart
	for i := 0; i != numsvcstsks; i++ {
		watchTask(t, s, taskWatch, false, hasNoIPOverlapTasks)
		watchService(t, serviceWatch, false, hasNoIPOverlapServices)
	}
}

func TestNodeAllocator(t *testing.T) {
	s := store.NewMemoryStore(nil)
	assert.NotNil(t, s)
	defer s.Close()

	a, err := New(s, nil)
	assert.NoError(t, err)
	assert.NotNil(t, a)

	var node1FromStore *api.Node
	node1 := &api.Node{
		ID: "nodeID1",
	}

	// Try adding some objects to store before allocator is started
	assert.NoError(t, s.Update(func(tx store.Tx) error {
		// populate ingress network
		in := &api.Network{
			ID: "ingress",
			Spec: api.NetworkSpec{
				Annotations: api.Annotations{
					Name: "ingress",
				},
				Ingress: true,
			},
		}
		assert.NoError(t, store.CreateNetwork(tx, in))

		n1 := &api.Network{
			ID: "overlayID1",
			Spec: api.NetworkSpec{
				Annotations: api.Annotations{
					Name: "overlayID1",
				},
			},
		}
		assert.NoError(t, store.CreateNetwork(tx, n1))

		assert.NoError(t, store.CreateNode(tx, node1))
		return nil
	}))

	nodeWatch, cancel := state.Watch(s.WatchQueue(), api.EventUpdateNode{}, api.EventDeleteNode{})
	defer cancel()
	netWatch, cancel := state.Watch(s.WatchQueue(), api.EventUpdateNetwork{}, api.EventDeleteNetwork{})
	defer cancel()

	// Start allocator
	go func() {
		assert.NoError(t, a.Run(context.Background()))
	}()
	defer a.Stop()

	// Validate node has 2 LB IP address (1 for each network).
	watchNetwork(t, netWatch, false, isValidNetwork)                                      // ingress
	watchNetwork(t, netWatch, false, isValidNetwork)                                      // overlayID1
	watchNode(t, nodeWatch, false, isValidNode, node1, []string{"ingress", "overlayID1"}) // node1

	// Add a node and validate it gets a LB ip on each network.
	node2 := &api.Node{
		ID: "nodeID2",
	}
	assert.NoError(t, s.Update(func(tx store.Tx) error {
		assert.NoError(t, store.CreateNode(tx, node2))
		return nil
	}))
	watchNode(t, nodeWatch, false, isValidNode, node2, []string{"ingress", "overlayID1"}) // node2

	// Add a network and validate each node has 3 LB IP addresses
	n2 := &api.Network{
		ID: "overlayID2",
		Spec: api.NetworkSpec{
			Annotations: api.Annotations{
				Name: "overlayID2",
			},
		},
	}
	assert.NoError(t, s.Update(func(tx store.Tx) error {
		assert.NoError(t, store.CreateNetwork(tx, n2))
		return nil
	}))
	watchNetwork(t, netWatch, false, isValidNetwork)                                                    // overlayID2
	watchNode(t, nodeWatch, false, isValidNode, node1, []string{"ingress", "overlayID1", "overlayID3"}) // node1
	watchNode(t, nodeWatch, false, isValidNode, node2, []string{"ingress", "overlayID1", "overlayID3"}) // node2

	// Remove a network and validate each node has 2 LB IP addresses
	assert.NoError(t, s.Update(func(tx store.Tx) error {
		assert.NoError(t, store.DeleteNetwork(tx, n2.ID))
		return nil
	}))
	watchNetwork(t, netWatch, false, isValidNetwork)                                      // overlayID2
	watchNode(t, nodeWatch, false, isValidNode, node1, []string{"ingress", "overlayID1"}) // node1
	watchNode(t, nodeWatch, false, isValidNode, node2, []string{"ingress", "overlayID1"}) // node2

	// Remove a node and validate remaining node has 2 LB IP addresses
	assert.NoError(t, s.Update(func(tx store.Tx) error {
		assert.NoError(t, store.DeleteNode(tx, node2.ID))
		return nil
	}))
	watchNode(t, nodeWatch, false, nil, nil, nil) // node2
	s.View(func(tx store.ReadTx) {
		node1FromStore = store.GetNode(tx, node1.ID)
	})

	isValidNode(t, node1, node1FromStore, []string{"ingress", "overlayID1"})

	// Validate that a LB IP address is not allocated for node-local networks
	p := &api.Network{
		ID: "bridge",
		Spec: api.NetworkSpec{
			Annotations: api.Annotations{
				Name: "pred_bridge_network",
				Labels: map[string]string{
					"com.docker.swarm.predefined": "true",
				},
			},
			DriverConfig: &api.Driver{Name: "bridge"},
		},
	}
	assert.NoError(t, s.Update(func(tx store.Tx) error {
		assert.NoError(t, store.CreateNetwork(tx, p))
		return nil
	}))
	watchNetwork(t, netWatch, false, isValidNetwork) // bridge

	s.View(func(tx store.ReadTx) {
		node1FromStore = store.GetNode(tx, node1.ID)
	})

	isValidNode(t, node1, node1FromStore, []string{"ingress", "overlayID1"})
}

func isValidNode(t assert.TestingT, originalNode, updatedNode *api.Node, networks []string) bool {

	if !assert.Equal(t, originalNode.ID, updatedNode.ID) {
		return false
	}

	if !assert.Equal(t, len(updatedNode.Attachments), len(networks)) {
		return false
	}

	for _, na := range updatedNode.Attachments {
		if !assert.Equal(t, len(na.Addresses), 1) {
			return false
		}
	}

	return true
}

func isValidNetwork(t assert.TestingT, n *api.Network) bool {
	if _, ok := n.Spec.Annotations.Labels["com.docker.swarm.predefined"]; ok {
		return true
	}
	return assert.NotEqual(t, n.IPAM.Configs, nil) &&
		assert.Equal(t, len(n.IPAM.Configs), 1) &&
		assert.Equal(t, n.IPAM.Configs[0].Range, "") &&
		assert.Equal(t, len(n.IPAM.Configs[0].Reserved), 0) &&
		isValidSubnet(t, n.IPAM.Configs[0].Subnet) &&
		assert.NotEqual(t, net.ParseIP(n.IPAM.Configs[0].Gateway), nil)
}

func isValidTask(t assert.TestingT, s *store.MemoryStore, task *api.Task) bool {
	return isValidNetworkAttachment(t, task) &&
		isValidEndpoint(t, s, task) &&
		assert.Equal(t, task.Status.State, api.TaskStatePending)
}

func isValidNetworkAttachment(t assert.TestingT, task *api.Task) bool {
	if len(task.Networks) != 0 {
		return assert.Equal(t, len(task.Networks[0].Addresses), 1) &&
			isValidSubnet(t, task.Networks[0].Addresses[0])
	}

	return true
}

func isValidEndpoint(t assert.TestingT, s *store.MemoryStore, task *api.Task) bool {
	if task.ServiceID != "" {
		var service *api.Service
		s.View(func(tx store.ReadTx) {
			service = store.GetService(tx, task.ServiceID)
		})

		if service == nil {
			return true
		}

		return assert.Equal(t, service.Endpoint, task.Endpoint)

	}

	return true
}

func isValidSubnet(t assert.TestingT, subnet string) bool {
	_, _, err := net.ParseCIDR(subnet)
	return assert.NoError(t, err)
}

type mockTester struct{}

func (m mockTester) Errorf(format string, args ...interface{}) {
}

// Returns a timeout given whether we should expect a timeout:  In the case where we do expect a timeout,
// the timeout should be short, because it's not very useful to wait long amounts of time just in case
// an unexpected event comes in - a short timeout should catch an incorrect event at least often enough
// to make the test flaky and alert us to the problem. But in the cases where we don't expect a timeout,
// the timeout should be on the order of several seconds, so the test doesn't fail just because it's run
// on a relatively slow system, or there's a load spike.
func getWatchTimeout(expectTimeout bool) time.Duration {
	if expectTimeout {
		return 350 * time.Millisecond
	}
	return 5 * time.Second
}

func watchNode(t *testing.T, watch chan events.Event, expectTimeout bool,
	fn func(t assert.TestingT, originalNode, updatedNode *api.Node, networks []string) bool,
	originalNode *api.Node,
	networks []string) {
	for {

		var node *api.Node
		select {
		case event := <-watch:
			if n, ok := event.(api.EventUpdateNode); ok {
				node = n.Node.Copy()
				if fn == nil || (fn != nil && fn(mockTester{}, originalNode, node, networks)) {
					return
				}
			}

			if n, ok := event.(api.EventDeleteNode); ok {
				node = n.Node.Copy()
				if fn == nil || (fn != nil && fn(mockTester{}, originalNode, node, networks)) {
					return
				}
			}

		case <-time.After(1 * time.Millisecond):
			if !expectTimeout {
				if node != nil && fn != nil {
					fn(t, originalNode, node, networks)
				}

				t.Fatal("timed out before watchNode found expected node state")
			}

			return
		}
	}
}

func watchNetwork(t *testing.T, watch chan events.Event, expectTimeout bool, fn func(t assert.TestingT, n *api.Network) bool) {
	for {
		var network *api.Network
		select {
		case event := <-watch:
			if n, ok := event.(api.EventUpdateNetwork); ok {
				network = n.Network.Copy()
				if fn == nil || (fn != nil && fn(mockTester{}, network)) {
					return
				}
			}

			if n, ok := event.(api.EventDeleteNetwork); ok {
				network = n.Network.Copy()
				if fn == nil || (fn != nil && fn(mockTester{}, network)) {
					return
				}
			}

		case <-time.After(getWatchTimeout(expectTimeout)):
			if !expectTimeout {
				if network != nil && fn != nil {
					fn(t, network)
				}

				t.Fatal("timed out before watchNetwork found expected network state")
			}

			return
		}
	}
}

func watchService(t *testing.T, watch chan events.Event, expectTimeout bool, fn func(t assert.TestingT, n *api.Service) bool) {
	for {
		var service *api.Service
		select {
		case event := <-watch:
			if s, ok := event.(api.EventUpdateService); ok {
				service = s.Service.Copy()
				if fn == nil || (fn != nil && fn(mockTester{}, service)) {
					return
				}
			}

			if s, ok := event.(api.EventDeleteService); ok {
				service = s.Service.Copy()
				if fn == nil || (fn != nil && fn(mockTester{}, service)) {
					return
				}
			}

		case <-time.After(getWatchTimeout(expectTimeout)):
			if !expectTimeout {
				if service != nil && fn != nil {
					fn(t, service)
				}

				t.Fatalf("timed out before watchService found expected service state\n stack = %s", string(debug.Stack()))
			}

			return
		}
	}
}

func watchTask(t *testing.T, s *store.MemoryStore, watch chan events.Event, expectTimeout bool, fn func(t assert.TestingT, s *store.MemoryStore, n *api.Task) bool) {
	for {
		var task *api.Task
		select {
		case event := <-watch:
			if t, ok := event.(api.EventUpdateTask); ok {
				task = t.Task.Copy()
				if fn == nil || (fn != nil && fn(mockTester{}, s, task)) {
					return
				}
			}

			if t, ok := event.(api.EventDeleteTask); ok {
				task = t.Task.Copy()
				if fn == nil || (fn != nil && fn(mockTester{}, s, task)) {
					return
				}
			}

		case <-time.After(getWatchTimeout(expectTimeout)):
			if !expectTimeout {
				if task != nil && fn != nil {
					fn(t, s, task)
				}

				t.Fatalf("timed out before watchTask found expected task state %s", debug.Stack())
			}

			return
		}
	}
}
