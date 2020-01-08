package template

import (
	"testing"

	"github.com/docker/swarmkit/agent"
	"github.com/docker/swarmkit/api"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestTemplatedSecret(t *testing.T) {
	templatedSecret := &api.Secret{
		ID: "templatedsecret",
	}

	referencedSecret := &api.Secret{
		ID: "referencedsecret",
		Spec: api.SecretSpec{
			Data: []byte("mysecret"),
		},
	}
	referencedConfig := &api.Config{
		ID: "referencedconfig",
		Spec: api.ConfigSpec{
			Data: []byte("myconfig"),
		},
	}

	type testCase struct {
		desc        string
		secretSpec  api.SecretSpec
		task        *api.Task
		node        *api.NodeDescription
		expected    string
		expectedErr string
	}

	testCases := []testCase{
		{
			desc: "Test expansion of task context",
			secretSpec: api.SecretSpec{
				Data: []byte("SERVICE_ID={{.Service.ID}}\n" +
					"SERVICE_NAME={{.Service.Name}}\n" +
					"TASK_ID={{.Task.ID}}\n" +
					"TASK_NAME={{.Task.Name}}\n" +
					"NODE_ID={{.Node.ID}}\n" +
					"NODE_HOSTNAME={{.Node.Hostname}}\n" +
					"NODE_OS={{.Node.Platform.OS}}\n" +
					"NODE_ARCHITECTURE={{.Node.Platform.Architecture}}"),
				Templating: &api.Driver{Name: "golang"},
			},
			expected: "SERVICE_ID=serviceID\n" +
				"SERVICE_NAME=serviceName\n" +
				"TASK_ID=taskID\n" +
				"TASK_NAME=serviceName.10.taskID\n" +
				"NODE_ID=nodeID\n" +
				"NODE_HOSTNAME=myhostname\n" +
				"NODE_OS=testOS\n" +
				"NODE_ARCHITECTURE=testArchitecture",
			task: modifyTask(func(t *api.Task) {
				t.Spec = api.TaskSpec{
					Runtime: &api.TaskSpec_Container{
						Container: &api.ContainerSpec{
							Secrets: []*api.SecretReference{
								{
									SecretID:   "templatedsecret",
									SecretName: "templatedsecretname",
								},
							},
						},
					},
				}
			}),
			node: modifyNode(func(n *api.NodeDescription) {
				n.Hostname = "myhostname"
				n.Platform.OS = "testOS"
				n.Platform.Architecture = "testArchitecture"
			}),
		},
		{
			desc: "Test expansion of secret, by target",
			secretSpec: api.SecretSpec{
				Data:       []byte("SECRET_VAL={{secret \"referencedsecrettarget\"}}\n"),
				Templating: &api.Driver{Name: "golang"},
			},
			expected: "SECRET_VAL=mysecret\n",
			task: modifyTask(func(t *api.Task) {
				t.Spec = api.TaskSpec{
					Runtime: &api.TaskSpec_Container{
						Container: &api.ContainerSpec{
							Secrets: []*api.SecretReference{
								{
									SecretID:   "templatedsecret",
									SecretName: "templatedsecretname",
								},
								{
									SecretID:   "referencedsecret",
									SecretName: "referencedsecretname",
									Target: &api.SecretReference_File{
										File: &api.FileTarget{
											Name: "referencedsecrettarget",
											UID:  "0",
											GID:  "0",
											Mode: 0666,
										},
									},
								},
							},
						},
					},
				}
			}),
			node: modifyNode(func(n *api.NodeDescription) {
				// use default values
			}),
		},
		{
			desc: "Test expansion of config, by target",
			secretSpec: api.SecretSpec{
				Data:       []byte("CONFIG_VAL={{config \"referencedconfigtarget\"}}\n"),
				Templating: &api.Driver{Name: "golang"},
			},
			expected: "CONFIG_VAL=myconfig\n",
			task: modifyTask(func(t *api.Task) {
				t.Spec = api.TaskSpec{
					Runtime: &api.TaskSpec_Container{
						Container: &api.ContainerSpec{
							Secrets: []*api.SecretReference{
								{
									SecretID:   "templatedsecret",
									SecretName: "templatedsecretname",
								},
							},
							Configs: []*api.ConfigReference{
								{
									ConfigID:   "referencedconfig",
									ConfigName: "referencedconfigname",
									Target: &api.ConfigReference_File{
										File: &api.FileTarget{
											Name: "referencedconfigtarget",
											UID:  "0",
											GID:  "0",
											Mode: 0666,
										},
									},
								},
							},
						},
					},
				}
			}),
			node: modifyNode(func(n *api.NodeDescription) {
				// use default values
			}),
		},
		{
			desc: "Test expansion of secret not available to task",
			secretSpec: api.SecretSpec{
				Data:       []byte("SECRET_VAL={{secret \"unknowntarget\"}}\n"),
				Templating: &api.Driver{Name: "golang"},
			},
			expectedErr: `failed to expand templated secret templatedsecret: template: expansion:1:13: executing "expansion" at <secret "unknowntarge...>: error calling secret: secret target unknowntarget not found`,
			task: modifyTask(func(t *api.Task) {
				t.Spec = api.TaskSpec{
					Runtime: &api.TaskSpec_Container{
						Container: &api.ContainerSpec{
							Secrets: []*api.SecretReference{
								{
									SecretID:   "templatedsecret",
									SecretName: "templatedsecretname",
								},
							},
						},
					},
				}
			}),
			node: modifyNode(func(n *api.NodeDescription) {
				// use default values
			}),
		},
		{
			desc: "Test expansion of config not available to task",
			secretSpec: api.SecretSpec{
				Data:       []byte("CONFIG_VAL={{config \"unknowntarget\"}}\n"),
				Templating: &api.Driver{Name: "golang"},
			},
			expectedErr: `failed to expand templated secret templatedsecret: template: expansion:1:13: executing "expansion" at <config "unknowntarge...>: error calling config: config target unknowntarget not found`,
			task: modifyTask(func(t *api.Task) {
				t.Spec = api.TaskSpec{
					Runtime: &api.TaskSpec_Container{
						Container: &api.ContainerSpec{
							Secrets: []*api.SecretReference{
								{
									SecretID:   "templatedsecret",
									SecretName: "templatedsecretname",
								},
							},
						},
					},
				}
			}),
			node: modifyNode(func(n *api.NodeDescription) {
				// use default values
			}),
		},
		{
			desc: "Test that expansion of the same secret avoids recursion",
			secretSpec: api.SecretSpec{
				Data:       []byte("SECRET_VAL={{secret \"templatedsecrettarget\"}}\n"),
				Templating: &api.Driver{Name: "golang"},
			},
			expected: "SECRET_VAL=SECRET_VAL={{secret \"templatedsecrettarget\"}}\n\n",
			task: modifyTask(func(t *api.Task) {
				t.Spec = api.TaskSpec{
					Runtime: &api.TaskSpec_Container{
						Container: &api.ContainerSpec{
							Secrets: []*api.SecretReference{
								{
									SecretID:   "templatedsecret",
									SecretName: "templatedsecretname",
									Target: &api.SecretReference_File{
										File: &api.FileTarget{
											Name: "templatedsecrettarget",
											UID:  "0",
											GID:  "0",
											Mode: 0666,
										},
									},
								},
							},
						},
					},
				}
			}),
			node: modifyNode(func(n *api.NodeDescription) {
				// use default values
			}),
		},
		{
			desc: "Test env",
			secretSpec: api.SecretSpec{
				Data: []byte("ENV VALUE={{env \"foo\"}}\n" +
					"DOES NOT EXIST={{env \"badname\"}}\n"),
				Templating: &api.Driver{Name: "golang"},
			},
			expected: "ENV VALUE=bar\n" +
				"DOES NOT EXIST=\n",
			task: modifyTask(func(t *api.Task) {
				t.Spec = api.TaskSpec{
					Runtime: &api.TaskSpec_Container{
						Container: &api.ContainerSpec{
							Secrets: []*api.SecretReference{
								{
									SecretID:   "templatedsecret",
									SecretName: "templatedsecretname",
								},
							},
							Env: []string{"foo=bar"},
						},
					},
				}
			}),
			node: modifyNode(func(n *api.NodeDescription) {
				// use default values
			}),
		},
	}

	for _, testCase := range testCases {
		templatedSecret.Spec = testCase.secretSpec

		dependencyManager := agent.NewDependencyManager()
		dependencyManager.Secrets().Add(*templatedSecret, *referencedSecret)
		dependencyManager.Configs().Add(*referencedConfig)

		templatedDependencies := NewTemplatedDependencyGetter(agent.Restrict(dependencyManager, testCase.task), testCase.task, testCase.node)
		expandedSecret, err := templatedDependencies.Secrets().Get("templatedsecret")

		if testCase.expectedErr != "" {
			assert.EqualError(t, err, testCase.expectedErr)
		} else {
			assert.NoError(t, err)
			require.NotNil(t, expandedSecret)
			assert.Equal(t, testCase.expected, string(expandedSecret.Spec.Data), testCase.desc)
		}
	}
}

func TestTemplatedConfig(t *testing.T) {
	templatedConfig := &api.Config{
		ID: "templatedconfig",
	}

	referencedSecret := &api.Secret{
		ID: "referencedsecret",
		Spec: api.SecretSpec{
			Data: []byte("mysecret"),
		},
	}
	referencedConfig := &api.Config{
		ID: "referencedconfig",
		Spec: api.ConfigSpec{
			Data: []byte("myconfig"),
		},
	}

	type testCase struct {
		desc              string
		configSpec        api.ConfigSpec
		task              *api.Task
		expected          string
		expectedErr       string
		expectedSensitive bool
		node              *api.NodeDescription
	}

	testCases := []testCase{
		{
			desc: "Test expansion of task context",
			configSpec: api.ConfigSpec{
				Data: []byte("SERVICE_ID={{.Service.ID}}\n" +
					"SERVICE_NAME={{.Service.Name}}\n" +
					"TASK_ID={{.Task.ID}}\n" +
					"TASK_NAME={{.Task.Name}}\n" +
					"NODE_ID={{.Node.ID}}\n" +
					"NODE_HOSTNAME={{.Node.Hostname}}\n" +
					"NODE_OS={{.Node.Platform.OS}}\n" +
					"NODE_ARCHITECTURE={{.Node.Platform.Architecture}}"),
				Templating: &api.Driver{Name: "golang"},
			},
			expected: "SERVICE_ID=serviceID\n" +
				"SERVICE_NAME=serviceName\n" +
				"TASK_ID=taskID\n" +
				"TASK_NAME=serviceName.10.taskID\n" +
				"NODE_ID=nodeID\n" +
				"NODE_HOSTNAME=myhostname\n" +
				"NODE_OS=testOS\n" +
				"NODE_ARCHITECTURE=testArchitecture",
			task: modifyTask(func(t *api.Task) {
				t.Spec = api.TaskSpec{
					Runtime: &api.TaskSpec_Container{
						Container: &api.ContainerSpec{
							Configs: []*api.ConfigReference{
								{
									ConfigID:   "templatedconfig",
									ConfigName: "templatedconfigname",
								},
							},
						},
					},
				}
			}),
			node: modifyNode(func(n *api.NodeDescription) {
				n.Hostname = "myhostname"
				n.Platform.OS = "testOS"
				n.Platform.Architecture = "testArchitecture"
			}),
		},
		{
			desc: "Test expansion of secret, by target",
			configSpec: api.ConfigSpec{
				Data:       []byte("SECRET_VAL={{secret \"referencedsecrettarget\"}}\n"),
				Templating: &api.Driver{Name: "golang"},
			},
			expected:          "SECRET_VAL=mysecret\n",
			expectedSensitive: true,
			task: modifyTask(func(t *api.Task) {
				t.Spec = api.TaskSpec{
					Runtime: &api.TaskSpec_Container{
						Container: &api.ContainerSpec{
							Secrets: []*api.SecretReference{
								{
									SecretID:   "referencedsecret",
									SecretName: "referencedsecretname",
									Target: &api.SecretReference_File{
										File: &api.FileTarget{
											Name: "referencedsecrettarget",
											UID:  "0",
											GID:  "0",
											Mode: 0666,
										},
									},
								},
							},
							Configs: []*api.ConfigReference{
								{
									ConfigID:   "templatedconfig",
									ConfigName: "templatedconfigname",
								},
							},
						},
					},
				}
			}),
			node: modifyNode(func(n *api.NodeDescription) {
				// use default values
			}),
		},
		{
			desc: "Test expansion of config, by target",
			configSpec: api.ConfigSpec{
				Data:       []byte("CONFIG_VAL={{config \"referencedconfigtarget\"}}\n"),
				Templating: &api.Driver{Name: "golang"},
			},
			expected: "CONFIG_VAL=myconfig\n",
			task: modifyTask(func(t *api.Task) {
				t.Spec = api.TaskSpec{
					Runtime: &api.TaskSpec_Container{
						Container: &api.ContainerSpec{
							Configs: []*api.ConfigReference{
								{
									ConfigID:   "templatedconfig",
									ConfigName: "templatedconfigname",
								},
								{
									ConfigID:   "referencedconfig",
									ConfigName: "referencedconfigname",
									Target: &api.ConfigReference_File{
										File: &api.FileTarget{
											Name: "referencedconfigtarget",
											UID:  "0",
											GID:  "0",
											Mode: 0666,
										},
									},
								},
							},
						},
					},
				}
			}),
			node: modifyNode(func(n *api.NodeDescription) {
				// use default values
			}),
		},
		{
			desc: "Test expansion of secret not available to task",
			configSpec: api.ConfigSpec{
				Data:       []byte("SECRET_VAL={{secret \"unknowntarget\"}}\n"),
				Templating: &api.Driver{Name: "golang"},
			},
			expectedErr: `failed to expand templated config templatedconfig: template: expansion:1:13: executing "expansion" at <secret "unknowntarge...>: error calling secret: secret target unknowntarget not found`,
			task: modifyTask(func(t *api.Task) {
				t.Spec = api.TaskSpec{
					Runtime: &api.TaskSpec_Container{
						Container: &api.ContainerSpec{
							Configs: []*api.ConfigReference{
								{
									ConfigID:   "templatedconfig",
									ConfigName: "templatedconfigname",
								},
							},
						},
					},
				}
			}),
			node: modifyNode(func(n *api.NodeDescription) {
				// use default values
			}),
		},
		{
			desc: "Test expansion of config not available to task",
			configSpec: api.ConfigSpec{
				Data:       []byte("CONFIG_VAL={{config \"unknowntarget\"}}\n"),
				Templating: &api.Driver{Name: "golang"},
			},
			expectedErr: `failed to expand templated config templatedconfig: template: expansion:1:13: executing "expansion" at <config "unknowntarge...>: error calling config: config target unknowntarget not found`,
			task: modifyTask(func(t *api.Task) {
				t.Spec = api.TaskSpec{
					Runtime: &api.TaskSpec_Container{
						Container: &api.ContainerSpec{
							Configs: []*api.ConfigReference{
								{
									ConfigID:   "templatedconfig",
									ConfigName: "templatedconfigname",
								},
							},
						},
					},
				}
			}),
			node: modifyNode(func(n *api.NodeDescription) {
				// use default values
			}),
		},
		{
			desc: "Test that expansion of the same config avoids recursion",
			configSpec: api.ConfigSpec{
				Data:       []byte("CONFIG_VAL={{config \"templatedconfigtarget\"}}\n"),
				Templating: &api.Driver{Name: "golang"},
			},
			expected: "CONFIG_VAL=CONFIG_VAL={{config \"templatedconfigtarget\"}}\n\n",
			task: modifyTask(func(t *api.Task) {
				t.Spec = api.TaskSpec{
					Runtime: &api.TaskSpec_Container{
						Container: &api.ContainerSpec{
							Configs: []*api.ConfigReference{
								{
									ConfigID:   "templatedconfig",
									ConfigName: "templatedconfigname",
									Target: &api.ConfigReference_File{
										File: &api.FileTarget{
											Name: "templatedconfigtarget",
											UID:  "0",
											GID:  "0",
											Mode: 0666,
										},
									},
								},
							},
						},
					},
				}
			}),
			node: modifyNode(func(n *api.NodeDescription) {
				// use default values
			}),
		},
		{
			desc: "Test env",
			configSpec: api.ConfigSpec{
				Data: []byte("ENV VALUE={{env \"foo\"}}\n" +
					"DOES NOT EXIST={{env \"badname\"}}\n"),
				Templating: &api.Driver{Name: "golang"},
			},
			expected: "ENV VALUE=bar\n" +
				"DOES NOT EXIST=\n",
			task: modifyTask(func(t *api.Task) {
				t.Spec = api.TaskSpec{
					Runtime: &api.TaskSpec_Container{
						Container: &api.ContainerSpec{
							Configs: []*api.ConfigReference{
								{
									ConfigID:   "templatedconfig",
									ConfigName: "templatedconfigname",
								},
							},
							Env: []string{"foo=bar"},
						},
					},
				}
			}),
			node: modifyNode(func(n *api.NodeDescription) {
				// use default values
			}),
		},
	}

	for _, testCase := range testCases {
		templatedConfig.Spec = testCase.configSpec

		dependencyManager := agent.NewDependencyManager()
		dependencyManager.Configs().Add(*templatedConfig, *referencedConfig)
		dependencyManager.Secrets().Add(*referencedSecret)

		templatedDependencies := NewTemplatedDependencyGetter(agent.Restrict(dependencyManager, testCase.task), testCase.task, testCase.node)
		expandedConfig1, err1 := templatedDependencies.Configs().Get("templatedconfig")
		expandedConfig2, sensitive, err2 := templatedDependencies.Configs().(TemplatedConfigGetter).GetAndFlagSecretData("templatedconfig")

		if testCase.expectedErr != "" {
			assert.EqualError(t, err1, testCase.expectedErr)
			assert.EqualError(t, err2, testCase.expectedErr)
		} else {
			assert.NoError(t, err1)
			assert.NoError(t, err2)
			require.NotNil(t, expandedConfig1)
			require.NotNil(t, expandedConfig2)
			assert.Equal(t, testCase.expected, string(expandedConfig1.Spec.Data), testCase.desc)
			assert.Equal(t, testCase.expected, string(expandedConfig2.Spec.Data), testCase.desc)
			assert.Equal(t, testCase.expectedSensitive, sensitive, testCase.desc)
		}
	}
}
