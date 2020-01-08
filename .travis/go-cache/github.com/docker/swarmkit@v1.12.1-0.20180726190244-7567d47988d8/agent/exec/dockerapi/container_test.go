package dockerapi

import (
	"reflect"
	"testing"
	"time"

	enginecontainer "github.com/docker/docker/api/types/container"
	enginemount "github.com/docker/docker/api/types/mount"
	"github.com/docker/swarmkit/api"
	gogotypes "github.com/gogo/protobuf/types"
)

func TestVolumesAndBinds(t *testing.T) {
	type testCase struct {
		explain string
		config  api.Mount
		x       enginemount.Mount
	}

	cases := []testCase{
		{"Simple bind mount", api.Mount{Type: api.MountTypeBind, Source: "/banana", Target: "/kerfluffle"},
			enginemount.Mount{Type: enginemount.TypeBind, Source: "/banana", Target: "/kerfluffle"}},
		{"Bind mound with propagation", api.Mount{Type: api.MountTypeBind, Source: "/banana", Target: "/kerfluffle", BindOptions: &api.Mount_BindOptions{Propagation: api.MountPropagationRPrivate}},
			enginemount.Mount{Type: enginemount.TypeBind, Source: "/banana", Target: "/kerfluffle", BindOptions: &enginemount.BindOptions{Propagation: enginemount.PropagationRPrivate}}},
		{"Simple volume with source", api.Mount{Type: api.MountTypeVolume, Source: "banana", Target: "/kerfluffle"},
			enginemount.Mount{Type: enginemount.TypeVolume, Source: "banana", Target: "/kerfluffle"}},
		{"Volume with options", api.Mount{Type: api.MountTypeVolume, Source: "banana", Target: "/kerfluffle", VolumeOptions: &api.Mount_VolumeOptions{NoCopy: true}},
			enginemount.Mount{Type: enginemount.TypeVolume, Source: "banana", Target: "/kerfluffle", VolumeOptions: &enginemount.VolumeOptions{NoCopy: true}}},
		{"Volume with no source", api.Mount{Type: api.MountTypeVolume, Target: "/kerfluffle"},
			enginemount.Mount{Type: enginemount.TypeVolume, Target: "/kerfluffle"}},
	}

	for _, c := range cases {
		cfg := containerConfig{
			task: &api.Task{
				Spec: api.TaskSpec{Runtime: &api.TaskSpec_Container{
					Container: &api.ContainerSpec{
						Mounts: []api.Mount{c.config},
					},
				}},
			},
		}

		if vols := cfg.config().Volumes; len(vols) != 0 {
			t.Fatalf("expected no anonymous volumes: %v", vols)
		}
		mounts := cfg.hostConfig().Mounts
		if len(mounts) != 1 {
			t.Fatalf("expected 1 mount: %v", mounts)
		}

		if !reflect.DeepEqual(mounts[0], c.x) {
			t.Log(c.explain)
			t.Logf("expected: %+v, got: %+v", c.x, mounts[0])
			switch c.x.Type {
			case enginemount.TypeVolume:
				t.Logf("expected volume opts: %+v, got: %+v", c.x.VolumeOptions, mounts[0].VolumeOptions)
				if c.x.VolumeOptions.DriverConfig != nil {
					t.Logf("expected volume driver config: %+v, got: %+v", c.x.VolumeOptions.DriverConfig, mounts[0].VolumeOptions.DriverConfig)
				}
			case enginemount.TypeBind:
				t.Logf("expected bind opts: %+v, got: %+v", c.x.BindOptions, mounts[0].BindOptions)
			}
			t.Fail()
		}
	}
}

func TestHealthcheck(t *testing.T) {
	c := containerConfig{
		task: &api.Task{
			Spec: api.TaskSpec{Runtime: &api.TaskSpec_Container{
				Container: &api.ContainerSpec{
					Healthcheck: &api.HealthConfig{
						Test:        []string{"a", "b", "c"},
						Interval:    gogotypes.DurationProto(time.Second),
						Timeout:     gogotypes.DurationProto(time.Minute),
						Retries:     10,
						StartPeriod: gogotypes.DurationProto(time.Minute),
					},
				},
			}},
		},
	}
	config := c.config()
	expected := &enginecontainer.HealthConfig{
		Test:        []string{"a", "b", "c"},
		Interval:    time.Second,
		Timeout:     time.Minute,
		Retries:     10,
		StartPeriod: time.Minute,
	}
	if !reflect.DeepEqual(config.Healthcheck, expected) {
		t.Fatalf("expected %#v, got %#v", expected, config.Healthcheck)
	}
}

func TestExtraHosts(t *testing.T) {
	c := containerConfig{
		task: &api.Task{
			Spec: api.TaskSpec{Runtime: &api.TaskSpec_Container{
				Container: &api.ContainerSpec{
					Hosts: []string{
						"1.2.3.4 example.com",
						"5.6.7.8 example.org",
						"127.0.0.1 mylocal",
					},
				},
			}},
		},
	}

	hostConfig := c.hostConfig()
	if len(hostConfig.ExtraHosts) != 3 {
		t.Fatalf("expected 3 extra hosts: %v", hostConfig.ExtraHosts)
	}

	expected := "example.com:1.2.3.4"
	actual := hostConfig.ExtraHosts[0]
	if actual != expected {
		t.Fatalf("expected %s, got %s", expected, actual)
	}

	expected = "example.org:5.6.7.8"
	actual = hostConfig.ExtraHosts[1]
	if actual != expected {
		t.Fatalf("expected %s, got %s", expected, actual)
	}

	expected = "mylocal:127.0.0.1"
	actual = hostConfig.ExtraHosts[2]
	if actual != expected {
		t.Fatalf("expected %s, got %s", expected, actual)
	}
}

func TestPidLimit(t *testing.T) {
	c := containerConfig{
		task: &api.Task{
			Spec: api.TaskSpec{Runtime: &api.TaskSpec_Container{
				Container: &api.ContainerSpec{
					PidsLimit: 10,
				},
			}},
		},
	}

	hostConfig := c.hostConfig()
	expected := int64(10)
	actual := hostConfig.PidsLimit

	if expected != actual {
		t.Fatalf("expected %d, got %d", expected, actual)
	}
}

func TestStopSignal(t *testing.T) {
	c := containerConfig{
		task: &api.Task{
			Spec: api.TaskSpec{Runtime: &api.TaskSpec_Container{
				Container: &api.ContainerSpec{
					StopSignal: "SIGWINCH",
				},
			}},
		},
	}

	expected := "SIGWINCH"
	actual := c.config().StopSignal
	if actual != expected {
		t.Fatalf("expected %s, got %s", expected, actual)
	}
}

func TestInit(t *testing.T) {
	c := containerConfig{
		task: &api.Task{
			Spec: api.TaskSpec{Runtime: &api.TaskSpec_Container{
				Container: &api.ContainerSpec{
					StopSignal: "SIGWINCH",
				},
			}},
		},
	}
	var expected *bool
	actual := c.hostConfig().Init
	if actual != expected {
		t.Fatalf("expected %v, got %v", expected, actual)
	}
	c.task.Spec.GetContainer().Init = &gogotypes.BoolValue{
		Value: true,
	}
	actual = c.hostConfig().Init
	if actual == nil || !*actual {
		t.Fatalf("expected &true, got %v", actual)
	}
}

func TestIsolation(t *testing.T) {
	c := containerConfig{
		task: &api.Task{
			Spec: api.TaskSpec{
				Runtime: &api.TaskSpec_Container{
					Container: &api.ContainerSpec{
						Isolation: api.ContainerIsolationHyperV,
					},
				},
			},
		},
	}

	expected := "hyperv"
	actual := string(c.hostConfig().Isolation)
	if actual != expected {
		t.Fatalf("expected %s, got %s", expected, actual)
	}
}
