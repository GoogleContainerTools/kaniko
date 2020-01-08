package scheduler

import (
	"testing"

	"github.com/docker/swarmkit/api"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

var (
	task1 *api.Task
	ni    *NodeInfo
)

func setupEnv() {
	task1 = &api.Task{
		ID:           "id1",
		DesiredState: api.TaskStateRunning,
		ServiceAnnotations: api.Annotations{
			Name: "name1",
		},

		Spec: api.TaskSpec{
			Runtime: &api.TaskSpec_Container{
				Container: &api.ContainerSpec{
					Command: []string{"sh", "-c", "sleep 5"},
					Image:   "alpine",
				},
			},
		},

		Status: api.TaskStatus{
			State: api.TaskStateAssigned,
		},
	}

	ni = &NodeInfo{
		Node: &api.Node{
			ID: "nodeid-1",
			Spec: api.NodeSpec{
				Annotations: api.Annotations{
					Labels: make(map[string]string),
				},
				DesiredRole: api.NodeRoleWorker,
			},
			Description: &api.NodeDescription{
				Engine: &api.EngineDescription{
					Labels: make(map[string]string),
				},
			},
			Status: api.NodeStatus{
				State: api.NodeStatus_READY,
				Addr:  "186.17.9.41",
			},
		},
		Tasks: make(map[string]*api.Task),
		ActiveTasksCountByService: make(map[string]int),
	}
}

func TestConstraintSetTask(t *testing.T) {
	setupEnv()
	f := ConstraintFilter{}
	assert.False(t, f.SetTask(task1))

	task1.Spec.Placement = &api.Placement{
		Constraints: []string{"node.hostname == node-2", "node.labels.security != low"},
	}
	assert.True(t, f.SetTask(task1))

	task1.Spec.Placement = &api.Placement{
		Constraints: []string{"node.id == nodeid-2", "engine.labels.operatingsystem != ubuntu"},
	}
	assert.True(t, f.SetTask(task1))
}

func TestWrongSyntax(t *testing.T) {
	setupEnv()
	f := ConstraintFilter{}
	task1.Spec.Placement = &api.Placement{
		Constraints: []string{"node.abc.bcd == high"},
	}
	require.True(t, f.SetTask(task1))
	assert.False(t, f.Check(ni))

	task1.Spec.Placement = &api.Placement{
		Constraints: []string{"node.abc.bcd != high"},
	}
	require.True(t, f.SetTask(task1))
	assert.False(t, f.Check(ni))
}

func TestNodeHostname(t *testing.T) {
	setupEnv()
	f := ConstraintFilter{}
	task1.Spec.Placement = &api.Placement{
		Constraints: []string{"node.hostname != node-1"},
	}
	require.True(t, f.SetTask(task1))

	// the node without hostname passes constraint
	assert.True(t, f.Check(ni))

	// add a not matching hostname
	ni.Description.Hostname = "node-2"
	assert.True(t, f.Check(ni))

	// matching engine name
	ni.Description.Hostname = "node-1"
	assert.False(t, f.Check(ni))

	// case insensitive
	ni.Node.Description.Hostname = "NODe-1"
	assert.False(t, f.Check(ni))
}

func TestNodeIP(t *testing.T) {
	setupEnv()
	f := ConstraintFilter{}

	type testcase struct {
		constraints    []string
		requireVerdict bool
		assertVerdict  bool
	}

	testFunc := func(tc testcase) {
		task1.Spec.Placement = &api.Placement{
			Constraints: tc.constraints,
		}
		require.Equal(t, f.SetTask(task1), tc.requireVerdict)
		if tc.requireVerdict {
			assert.Equal(t, f.Check(ni), tc.assertVerdict)
		}
	}

	ipv4tests := []testcase{
		{[]string{"node.ip == 186.17.9.41"}, true, true},
		{[]string{"node.ip != 186.17.9.41"}, true, false},
		{[]string{"node.ip == 186.17.9.42"}, true, false},
		{[]string{"node.ip == 186.17.9.4/24"}, true, true},
		{[]string{"node.ip == 186.17.8.41/24"}, true, false},
		// invalid CIDR format
		{[]string{"node.ip == 186.17.9.41/34"}, true, false},
		// malformed IP
		{[]string{"node.ip != 266.17.9.41"}, true, false},
		// zero
		{[]string{"node.ip != 0.0.0.0"}, true, true},
		// invalid input, detected by SetTask
		{[]string{"node.ip == "}, false, true},
		// invalid input, not detected by SetTask
		{[]string{"node.ip == not_ip_addr"}, true, false},
	}

	for _, tc := range ipv4tests {
		testFunc(tc)
	}

	// IPv6 address
	ni.Status.Addr = "2001:db8::2"
	ipv6tests := []testcase{
		{[]string{"node.ip == 2001:db8::2"}, true, true},
		// same IPv6 address, different format
		{[]string{"node.ip == 2001:db8:0::2"}, true, true},
		{[]string{"node.ip != 2001:db8::2/128"}, true, false},
		{[]string{"node.ip == 2001:db8::/64"}, true, true},
		{[]string{"node.ip == 2001:db9::/64"}, true, false},
		{[]string{"node.ip != 2001:db9::/64"}, true, true},
	}

	for _, tc := range ipv6tests {
		testFunc(tc)
	}

	// node doesn't have address
	ni.Status.Addr = ""
	edgetests := []testcase{
		{[]string{"node.ip == 0.0.0.0"}, true, false},
		{[]string{"node.ip != 0.0.0.0"}, true, true},
	}

	for _, tc := range edgetests {
		testFunc(tc)
	}
}

func TestNodeID(t *testing.T) {
	setupEnv()
	f := ConstraintFilter{}
	task1.Spec.Placement = &api.Placement{
		Constraints: []string{"node.id == nodeid-1"},
	}
	require.True(t, f.SetTask(task1))
	assert.True(t, f.Check(ni))

	// full text match, cannot be longer
	task1.Spec.Placement = &api.Placement{
		Constraints: []string{"node.id == nodeid-1-extra"},
	}
	require.True(t, f.SetTask(task1))
	assert.False(t, f.Check(ni))

	// cannot be shorter
	task1.Spec.Placement = &api.Placement{
		Constraints: []string{"node.id == nodeid-"},
	}
	require.True(t, f.SetTask(task1))
	assert.False(t, f.Check(ni))
}

func TestNodeRole(t *testing.T) {
	setupEnv()
	f := ConstraintFilter{}
	task1.Spec.Placement = &api.Placement{
		Constraints: []string{"node.role == worker"},
	}
	require.True(t, f.SetTask(task1))
	assert.True(t, f.Check(ni))

	task1.Spec.Placement = &api.Placement{
		Constraints: []string{"node.role == manager"},
	}
	require.True(t, f.SetTask(task1))
	assert.False(t, f.Check(ni))

	// no such role as worker-manage
	task1.Spec.Placement = &api.Placement{
		Constraints: []string{"node.role == worker-manager"},
	}
	require.True(t, f.SetTask(task1))
	assert.False(t, f.Check(ni))
}

func TestNodePlatform(t *testing.T) {
	setupEnv()
	f := ConstraintFilter{}
	task1.Spec.Placement = &api.Placement{
		Constraints: []string{"node.platform.os == linux"},
	}
	require.True(t, f.SetTask(task1))
	//node info doesn't have platform yet
	assert.False(t, f.Check(ni))

	ni.Node.Description.Platform = &api.Platform{
		Architecture: "x86_64",
		OS:           "linux",
	}
	assert.True(t, f.Check(ni))

	ni.Node.Description.Platform = &api.Platform{
		Architecture: "x86_64",
		OS:           "windows",
	}
	assert.False(t, f.Check(ni))

	task1.Spec.Placement = &api.Placement{
		Constraints: []string{"node.platform.arch == amd64"},
	}
	require.True(t, f.SetTask(task1))
	assert.False(t, f.Check(ni))

	task1.Spec.Placement = &api.Placement{
		Constraints: []string{"node.platform.arch != amd64"},
	}
	require.True(t, f.SetTask(task1))
	assert.True(t, f.Check(ni))
}

func TestNodeLabel(t *testing.T) {
	setupEnv()
	f := ConstraintFilter{}
	task1.Spec.Placement = &api.Placement{
		Constraints: []string{"node.labels.security == high"},
	}
	require.True(t, f.SetTask(task1))
	assert.False(t, f.Check(ni))

	// engine label is not node label
	ni.Description.Engine.Labels["security"] = "high"
	assert.False(t, f.Check(ni))

	ni.Spec.Annotations.Labels["security"] = "high"
	assert.True(t, f.Check(ni))
}

func TestEngineLabel(t *testing.T) {
	setupEnv()
	f := ConstraintFilter{}
	task1.Spec.Placement = &api.Placement{
		Constraints: []string{"engine.labels.disk != ssd"},
	}
	require.True(t, f.SetTask(task1))
	// no such label matches !=
	assert.True(t, f.Check(ni))

	// node label is not engine label
	ni.Spec.Annotations.Labels["disk"] = "ssd"
	assert.True(t, f.Check(ni))

	ni.Description.Engine.Labels["disk"] = "ssd"
	assert.False(t, f.Check(ni))

	// extra label doesn't interfere
	ni.Description.Engine.Labels["memory"] = "large"
	assert.False(t, f.Check(ni))
}

func TestMultipleConstraints(t *testing.T) {
	setupEnv()
	f := ConstraintFilter{}
	task1.Spec.Placement = &api.Placement{
		Constraints: []string{"node.hostname == node-1", "engine.labels.operatingsystem != Ubuntu 14.04"},
	}
	require.True(t, f.SetTask(task1))
	assert.False(t, f.Check(ni))

	ni.Description.Hostname = "node-1"
	assert.True(t, f.Check(ni))

	// set node operating system
	ni.Node.Description.Engine.Labels["operatingsystem"] = "Ubuntu 14.04"
	assert.False(t, f.Check(ni))

	// case insensitive
	ni.Node.Description.Engine.Labels["operatingsystem"] = "ubuntu 14.04"
	assert.False(t, f.Check(ni))

	ni.Node.Description.Engine.Labels["operatingsystem"] = "ubuntu 15.04"
	assert.True(t, f.Check(ni))

	// add one more label requirement to task
	task1.Spec.Placement = &api.Placement{
		Constraints: []string{"node.hostname == node-1",
			"engine.labels.operatingsystem != Ubuntu 14.04",
			"node.labels.security == high"},
	}
	require.True(t, f.SetTask(task1))
	assert.False(t, f.Check(ni))

	// add label to Spec.Annotations.Labels
	ni.Spec.Annotations.Labels["security"] = "low"
	assert.False(t, f.Check(ni))
	ni.Spec.Annotations.Labels["security"] = "high"
	assert.True(t, f.Check(ni))

	// extra label doesn't interfere
	ni.Description.Engine.Labels["memory"] = "large"
	assert.True(t, f.Check(ni))
}
