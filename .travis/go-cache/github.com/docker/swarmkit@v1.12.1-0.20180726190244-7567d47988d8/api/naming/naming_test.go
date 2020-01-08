package naming

import (
	"testing"

	"github.com/docker/swarmkit/api"
	"github.com/stretchr/testify/assert"
)

func TestTaskNaming(t *testing.T) {
	for _, testcase := range []struct {
		Name     string
		Task     *api.Task
		Expected string
	}{
		{
			Name: "Basic",
			Task: &api.Task{
				ID:     "taskID",
				Slot:   10,
				NodeID: "thenodeID",
				ServiceAnnotations: api.Annotations{
					Name: "theservice",
				},
			},
			Expected: "theservice.10.taskID",
		},
		{
			Name: "Annotations",
			Task: &api.Task{
				ID:     "taskID",
				NodeID: "thenodeID",
				Annotations: api.Annotations{
					Name: "thisisthetaskname",
				},
				ServiceAnnotations: api.Annotations{
					Name: "theservice",
				},
			},
			Expected: "thisisthetaskname",
		},
		{
			Name: "NoSlot",
			Task: &api.Task{
				ID:     "taskID",
				NodeID: "thenodeID",
				ServiceAnnotations: api.Annotations{
					Name: "theservice",
				},
			},
			Expected: "theservice.thenodeID.taskID",
		},
	} {
		t.Run(testcase.Name, func(t *testing.T) {
			t.Parallel()
			name := Task(testcase.Task)
			assert.Equal(t, name, testcase.Expected)
		})
	}
}
