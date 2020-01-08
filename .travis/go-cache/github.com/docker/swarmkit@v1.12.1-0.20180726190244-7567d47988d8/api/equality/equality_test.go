package equality

import (
	"testing"

	"github.com/docker/swarmkit/api"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestTasksEqualStable(t *testing.T) {
	const taskCount = 5
	var tasks [taskCount]*api.Task

	for i := 0; i < taskCount; i++ {
		tasks[i] = &api.Task{
			ID:   "task-id",
			Meta: api.Meta{Version: api.Version{Index: 6}},
			Spec: api.TaskSpec{
				Runtime: &api.TaskSpec_Container{
					Container: &api.ContainerSpec{
						Image: "redis:3.0.7",
					},
				},
			},
			ServiceID:    "service-id",
			Slot:         3,
			NodeID:       "node-id",
			Status:       api.TaskStatus{State: api.TaskStateAssigned},
			DesiredState: api.TaskStateReady,
		}
	}

	tasks[1].Status.State = api.TaskStateFailed
	tasks[2].Meta.Version.Index = 7
	tasks[3].DesiredState = api.TaskStateRunning
	tasks[4].Spec.Runtime = &api.TaskSpec_Container{
		Container: &api.ContainerSpec{
			Image: "redis:3.2.1",
		},
	}

	var tests = []struct {
		task        *api.Task
		expected    bool
		failureText string
	}{
		{tasks[1], true, "Tasks with different Status should be equal"},
		{tasks[2], true, "Tasks with different Meta should be equal"},
		{tasks[3], false, "Tasks with different DesiredState are not equal"},
		{tasks[4], false, "Tasks with different Spec are not equal"},
	}
	for _, test := range tests {
		assert.Equal(t, TasksEqualStable(tasks[0], test.task), test.expected, test.failureText)
	}
}

func TestRootCAEqualStable(t *testing.T) {
	root1 := api.RootCA{
		CACert:     []byte("1"),
		CAKey:      []byte("2"),
		CACertHash: "hash",
	}
	root2 := root1
	root2.JoinTokens = api.JoinTokens{
		Worker:  "worker",
		Manager: "manager",
	}
	root3 := root1
	root3.RootRotation = &api.RootRotation{
		CACert:            []byte("3"),
		CAKey:             []byte("4"),
		CrossSignedCACert: []byte("5"),
	}

	for _, v := range []struct{ a, b *api.RootCA }{
		{a: nil, b: nil},
		{a: &root1, b: &root1},
		{a: &root1, b: &root2},
		{a: &root3, b: &root3},
	} {
		require.True(t, RootCAEqualStable(v.a, v.b), "should be equal:\n%v\n%v\n", v.a, v.b)
	}

	root1Permutations := []api.RootCA{root1, root1, root1}
	root3Permutations := []api.RootCA{root3, root3, root3}
	for _, r := range root3Permutations {
		copy := *r.RootRotation
		root3.RootRotation = &copy
	}
	root1Permutations[0].CACert = []byte("nope")
	root1Permutations[1].CAKey = []byte("nope")
	root1Permutations[2].CACertHash = "nope"
	root3Permutations[0].RootRotation.CACert = []byte("nope")
	root3Permutations[1].RootRotation.CAKey = []byte("nope")
	root3Permutations[2].RootRotation.CrossSignedCACert = []byte("nope")

	for _, v := range []struct{ a, b *api.RootCA }{
		{a: &root1, b: &root3},
		{a: &root1, b: &root1Permutations[0]},
		{a: &root1, b: &root1Permutations[1]},
		{a: &root1, b: &root1Permutations[2]},
		{a: &root3, b: &root3Permutations[0]},
		{a: &root3, b: &root3Permutations[1]},
		{a: &root3, b: &root3Permutations[2]},
	} {
		require.False(t, RootCAEqualStable(v.a, v.b), "should not be equal:\n%v\n%v\n", v.a, v.b)
	}
}

func TestExternalCAsEqualStable(t *testing.T) {
	externals := []*api.ExternalCA{
		{URL: "1"},
		{
			URL:    "1",
			CACert: []byte("cacert"),
		},
		{
			URL:      "1",
			CACert:   []byte("cacert"),
			Protocol: 1,
		},
		{
			URL:    "1",
			CACert: []byte("cacert"),
			Options: map[string]string{
				"hello": "there",
			},
		},
		{
			URL:    "1",
			CACert: []byte("cacert"),
			Options: map[string]string{
				"hello": "world",
			},
		},
	}
	// equal
	for _, v := range []struct{ a, b []*api.ExternalCA }{
		{a: nil, b: []*api.ExternalCA{}},
		{a: externals, b: externals},
		{a: externals[0:1], b: externals[0:1]},
	} {
		require.True(t, ExternalCAsEqualStable(v.a, v.b), "should be equal:\n%v\n%v\n", v.a, v.b)
	}
	// not equal
	for _, v := range []struct{ a, b []*api.ExternalCA }{
		{a: nil, b: externals},
		{a: externals[2:3], b: externals[3:4]},
		{a: externals[2:3], b: externals[4:5]},
		{a: externals[3:4], b: externals[4:5]},
	} {
		require.False(t, ExternalCAsEqualStable(v.a, v.b), "should not be equal:\n%v\n%v\n", v.a, v.b)
	}
}
