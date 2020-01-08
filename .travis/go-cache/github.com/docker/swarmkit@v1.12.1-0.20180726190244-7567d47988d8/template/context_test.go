package template

import (
	"strings"
	"testing"

	"github.com/docker/swarmkit/api"
	"github.com/stretchr/testify/assert"
)

func TestTemplateContext(t *testing.T) {
	for _, testcase := range []struct {
		Test            string
		Task            *api.Task
		Context         Context
		Expected        *api.ContainerSpec
		Err             error
		NodeDescription *api.NodeDescription
	}{
		{
			Test: "Identity",
			Task: modifyTask(func(t *api.Task) {
				t.Spec = api.TaskSpec{
					Runtime: &api.TaskSpec_Container{
						Container: &api.ContainerSpec{
							Env: []string{
								"NOTOUCH=dont",
							},
							Mounts: []api.Mount{
								{
									Target: "foo",
									Source: "bar",
								},
							},
						},
					},
				}
			}),
			NodeDescription: modifyNode(func(n *api.NodeDescription) {
			}),
			Expected: &api.ContainerSpec{
				Env: []string{
					"NOTOUCH=dont",
				},
				Mounts: []api.Mount{
					{
						Target: "foo",
						Source: "bar",
					},
				},
			},
		},
		{
			Test: "Env",
			Task: modifyTask(func(t *api.Task) {
				t.Spec = api.TaskSpec{
					Runtime: &api.TaskSpec_Container{
						Container: &api.ContainerSpec{
							Labels: map[string]string{
								"ContainerLabel": "should-NOT-end-up-as-task",
							},
							Env: []string{
								"MYENV=notemplate",
								"{{.NotExpanded}}=foo",
								"SERVICE_ID={{.Service.ID}}",
								"SERVICE_NAME={{.Service.Name}}",
								"TASK_ID={{.Task.ID}}",
								"TASK_NAME={{.Task.Name}}",
								"NODE_ID={{.Node.ID}}",
								"SERVICE_LABELS={{range $k, $v := .Service.Labels}}{{$k}}={{$v}},{{end}}",
							},
						},
					},
				}
			}),
			NodeDescription: modifyNode(func(n *api.NodeDescription) {
			}),
			Expected: &api.ContainerSpec{
				Labels: map[string]string{
					"ContainerLabel": "should-NOT-end-up-as-task",
				},
				Env: []string{
					"MYENV=notemplate",
					"{{.NotExpanded}}=foo",
					"SERVICE_ID=serviceID",
					"SERVICE_NAME=serviceName",
					"TASK_ID=taskID",
					"TASK_NAME=serviceName.10.taskID",
					"NODE_ID=nodeID",
					"SERVICE_LABELS=ServiceLabelOneKey=service-label-one-value,ServiceLabelTwoKey=service-label-two-value,com.example.ServiceLabelThreeKey=service-label-three-value,",
				},
			},
		},
		{
			Test: "Mount",
			Task: modifyTask(func(t *api.Task) {
				t.Spec = api.TaskSpec{
					Runtime: &api.TaskSpec_Container{
						Container: &api.ContainerSpec{
							Mounts: []api.Mount{
								{
									Source: "bar-{{.Node.ID}}-{{.Task.Name}}",
									Target: "foo-{{.Service.ID}}-{{.Service.Name}}",
								},
								{
									Source: "bar-{{.Node.ID}}-{{.Service.Name}}",
									Target: "foo-{{.Task.Slot}}-{{.Task.ID}}",
								},
							},
						},
					},
				}
			}),
			NodeDescription: modifyNode(func(n *api.NodeDescription) {
			}),
			Expected: &api.ContainerSpec{
				Mounts: []api.Mount{
					{
						Source: "bar-nodeID-serviceName.10.taskID",
						Target: "foo-serviceID-serviceName",
					},
					{
						Source: "bar-nodeID-serviceName",
						Target: "foo-10-taskID",
					},
				},
			},
		},
		{
			Test: "Hostname",
			Task: modifyTask(func(t *api.Task) {
				t.Spec = api.TaskSpec{
					Runtime: &api.TaskSpec_Container{
						Container: &api.ContainerSpec{
							Hostname: "myhost-{{.Task.Slot}}",
						},
					},
				}
			}),
			NodeDescription: modifyNode(func(n *api.NodeDescription) {
			}),
			Expected: &api.ContainerSpec{
				Hostname: "myhost-10",
			},
		},
		{
			Test: "Node hostname",
			Task: modifyTask(func(t *api.Task) {
				t.Spec = api.TaskSpec{
					Runtime: &api.TaskSpec_Container{
						Container: &api.ContainerSpec{
							Hostname: "myservice-{{.Node.Hostname}}",
						},
					},
				}
			}),
			NodeDescription: modifyNode(func(n *api.NodeDescription) {
				n.Hostname = "mynode"
			}),
			Expected: &api.ContainerSpec{
				Hostname: "myservice-mynode",
			},
		},
		{
			Test: "Node architecture",
			Task: modifyTask(func(t *api.Task) {
				t.Spec = api.TaskSpec{
					Runtime: &api.TaskSpec_Container{
						Container: &api.ContainerSpec{
							Hostname: "{{.Node.Hostname}}-{{.Node.Platform.OS}}-{{.Node.Platform.Architecture}}",
						},
					},
				}
			}),
			NodeDescription: modifyNode(func(n *api.NodeDescription) {
				n.Hostname = "mynode"
				n.Platform.Architecture = "myarchitecture"
				n.Platform.OS = "myos"
			}),
			Expected: &api.ContainerSpec{
				Hostname: "mynode-myos-myarchitecture",
			},
		},
	} {
		t.Run(testcase.Test, func(t *testing.T) {
			spec, err := ExpandContainerSpec(testcase.NodeDescription, testcase.Task)
			if err != nil {
				if testcase.Err == nil {
					t.Fatalf("unexpected error: %v", err)
				} else {
					if err != testcase.Err {
						t.Fatalf("unexpected error: %v != %v", err, testcase.Err)
					}
				}
			}

			assert.Equal(t, testcase.Expected, spec)

			for k, v := range testcase.Task.Annotations.Labels {
				// make sure that that task.annotations.labels didn't make an appearance.
				visitAllTemplatedFields(spec, func(s string) {
					if strings.Contains(s, k) || strings.Contains(s, v) {
						t.Fatalf("string value from task labels found in expanded spec: %q or %q found in %q, on %#v", k, v, s, spec)
					}
				})
			}
		})
	}
}

// modifyTask generates a task with interesting values then calls the function
// with it. The caller can then modify the task and return the result.
func modifyTask(fn func(t *api.Task)) *api.Task {
	t := &api.Task{
		ID:        "taskID",
		ServiceID: "serviceID",
		NodeID:    "nodeID",
		Slot:      10,
		Annotations: api.Annotations{
			Labels: map[string]string{
				// SUBTLE(stevvooe): Task labels ARE NOT templated. These are
				// reserved for the system and templated is not really needed.
				// Non of these values show show up in templates.
				"TaskLabelOneKey":               "task-label-one-value",
				"TaskLabelTwoKey":               "task-label-two-value",
				"com.example.TaskLabelThreeKey": "task-label-three-value",
			},
		},
		ServiceAnnotations: api.Annotations{
			Name: "serviceName",
			Labels: map[string]string{
				"ServiceLabelOneKey":               "service-label-one-value",
				"ServiceLabelTwoKey":               "service-label-two-value",
				"com.example.ServiceLabelThreeKey": "service-label-three-value",
			},
		},
	}

	fn(t)

	return t
}

// modifyNode generates a node with interesting values then calls the function
// with it. The caller can then modify the node and return the result.
func modifyNode(fn func(n *api.NodeDescription)) *api.NodeDescription {
	n := &api.NodeDescription{
		Hostname: "nodeHostname",
		Platform: &api.Platform{
			Architecture: "x86_64",
			OS:           "linux",
		},
	}

	fn(n)

	return n
}

// visitAllTemplatedFields does just that.
// TODO(stevvooe): Might be best to make this the actual implementation.
func visitAllTemplatedFields(spec *api.ContainerSpec, fn func(value string)) {
	for _, v := range spec.Env {
		fn(v)
	}

	for _, mount := range spec.Mounts {
		fn(mount.Target)
		fn(mount.Source)

		if mount.VolumeOptions != nil {
			for _, v := range mount.VolumeOptions.Labels {
				fn(v)
			}

			if mount.VolumeOptions.DriverConfig != nil {
				for _, v := range mount.VolumeOptions.DriverConfig.Options {
					fn(v)
				}
			}
		}
	}
}
