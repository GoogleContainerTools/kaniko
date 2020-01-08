package controlapi

import (
	"fmt"
	"strings"
	"testing"
	"time"

	"github.com/docker/swarmkit/api"
	"github.com/docker/swarmkit/manager/state/store"
	gogotypes "github.com/gogo/protobuf/types"
	"github.com/stretchr/testify/assert"
	"golang.org/x/net/context"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
)

func createGenericSpec(name, runtime string) *api.ServiceSpec {
	spec := createSpec(name, runtime, 0)
	spec.Task.Runtime = &api.TaskSpec_Generic{
		Generic: &api.GenericRuntimeSpec{
			Kind: runtime,
			Payload: &gogotypes.Any{
				TypeUrl: "com.docker.custom.runtime",
				Value:   []byte{0},
			},
		},
	}
	return spec
}

func createSpec(name, image string, instances uint64) *api.ServiceSpec {
	return &api.ServiceSpec{
		Annotations: api.Annotations{
			Name: name,
			Labels: map[string]string{
				"common": "yes",
				"unique": name,
			},
		},
		Task: api.TaskSpec{
			Runtime: &api.TaskSpec_Container{
				Container: &api.ContainerSpec{
					Image: image,
				},
			},
		},
		Mode: &api.ServiceSpec_Replicated{
			Replicated: &api.ReplicatedService{
				Replicas: instances,
			},
		},
	}
}

func createSpecWithDuplicateMounts(name string) *api.ServiceSpec {
	service := createSpec("", "image", 1)
	mounts := []api.Mount{
		{
			Target: "/foo",
			Source: "/mnt/mount1",
		},
		{
			Target: "/foo",
			Source: "/mnt/mount2",
		},
	}

	service.Task.GetContainer().Mounts = mounts

	return service
}

func createSpecWithHostnameTemplate(serviceName, hostnameTmpl string) *api.ServiceSpec {
	service := createSpec(serviceName, "image", 1)
	service.Task.GetContainer().Hostname = hostnameTmpl
	return service
}

func createSecret(t *testing.T, ts *testServer, secretName, target string) *api.SecretReference {
	secretSpec := createSecretSpec(secretName, []byte(secretName), nil)
	secret := &api.Secret{
		ID:   fmt.Sprintf("ID%v", secretName),
		Spec: *secretSpec,
	}
	err := ts.Store.Update(func(tx store.Tx) error {
		return store.CreateSecret(tx, secret)
	})
	assert.NoError(t, err)

	return &api.SecretReference{
		SecretName: secret.Spec.Annotations.Name,
		SecretID:   secret.ID,
		Target: &api.SecretReference_File{
			File: &api.FileTarget{
				Name: target,
				UID:  "0",
				GID:  "0",
				Mode: 0666,
			},
		},
	}
}

func createServiceSpecWithSecrets(serviceName string, secretRefs ...*api.SecretReference) *api.ServiceSpec {
	service := createSpec(serviceName, fmt.Sprintf("image%v", serviceName), 1)
	service.Task.GetContainer().Secrets = secretRefs

	return service
}

func createConfig(t *testing.T, ts *testServer, configName, target string) *api.ConfigReference {
	configSpec := createConfigSpec(configName, []byte(configName), nil)
	config := &api.Config{
		ID:   fmt.Sprintf("ID%v", configName),
		Spec: *configSpec,
	}
	err := ts.Store.Update(func(tx store.Tx) error {
		return store.CreateConfig(tx, config)
	})
	assert.NoError(t, err)

	return &api.ConfigReference{
		ConfigName: config.Spec.Annotations.Name,
		ConfigID:   config.ID,
		Target: &api.ConfigReference_File{
			File: &api.FileTarget{
				Name: target,
				UID:  "0",
				GID:  "0",
				Mode: 0666,
			},
		},
	}
}

func createServiceSpecWithConfigs(serviceName string, configRefs ...*api.ConfigReference) *api.ServiceSpec {
	service := createSpec(serviceName, fmt.Sprintf("image%v", serviceName), 1)
	service.Task.GetContainer().Configs = configRefs

	return service
}

func createService(t *testing.T, ts *testServer, name, image string, instances uint64) *api.Service {
	spec := createSpec(name, image, instances)
	r, err := ts.Client.CreateService(context.Background(), &api.CreateServiceRequest{Spec: spec})
	assert.NoError(t, err)
	return r.Service
}

func createGenericService(t *testing.T, ts *testServer, name, runtime string) *api.Service {
	spec := createGenericSpec(name, runtime)
	r, err := ts.Client.CreateService(context.Background(), &api.CreateServiceRequest{Spec: spec})
	assert.NoError(t, err)
	return r.Service
}

func getIngressTargetID(t *testing.T, ts *testServer) string {
	rsp, err := ts.Client.ListNetworks(context.Background(), &api.ListNetworksRequest{})
	assert.NoError(t, err)
	for _, n := range rsp.Networks {
		if n.Spec.Ingress {
			return n.ID
		}
	}
	t.Fatal("unable to find ingress")
	return ""
}

func TestValidateResources(t *testing.T) {
	bad := []*api.Resources{
		{MemoryBytes: 1},
		{NanoCPUs: 42},
	}

	good := []*api.Resources{
		{MemoryBytes: 4096 * 1024 * 1024},
		{NanoCPUs: 1e9},
	}

	for _, b := range bad {
		err := validateResources(b)
		assert.Error(t, err)
		assert.Equal(t, codes.InvalidArgument, grpc.Code(err))
	}

	for _, g := range good {
		assert.NoError(t, validateResources(g))
	}
}

func TestValidateResourceRequirements(t *testing.T) {
	bad := []*api.ResourceRequirements{
		{Limits: &api.Resources{MemoryBytes: 1}},
		{Reservations: &api.Resources{MemoryBytes: 1}},
	}
	good := []*api.ResourceRequirements{
		{Limits: &api.Resources{NanoCPUs: 1e9}},
		{Reservations: &api.Resources{NanoCPUs: 1e9}},
	}
	for _, b := range bad {
		err := validateResourceRequirements(b)
		assert.Error(t, err)
		assert.Equal(t, codes.InvalidArgument, grpc.Code(err))
	}

	for _, g := range good {
		assert.NoError(t, validateResourceRequirements(g))
	}
}

func TestValidateMode(t *testing.T) {
	negative := -4
	bad := []*api.ServiceSpec{
		// -4 jammed into the replicas field, underflowing the uint64
		{Mode: &api.ServiceSpec_Replicated{Replicated: &api.ReplicatedService{Replicas: uint64(negative)}}},
		{},
	}

	good := []*api.ServiceSpec{
		{Mode: &api.ServiceSpec_Replicated{Replicated: &api.ReplicatedService{Replicas: 2}}},
		{Mode: &api.ServiceSpec_Global{}},
	}

	for _, b := range bad {
		err := validateMode(b)
		assert.Error(t, err)
		assert.Equal(t, codes.InvalidArgument, grpc.Code(err))
	}

	for _, g := range good {
		assert.NoError(t, validateMode(g))
	}
}

func TestValidateTaskSpec(t *testing.T) {
	type badSource struct {
		s api.TaskSpec
		c codes.Code
	}

	for _, bad := range []badSource{
		{
			s: api.TaskSpec{
				Runtime: &api.TaskSpec_Container{
					Container: &api.ContainerSpec{},
				},
			},
			c: codes.InvalidArgument,
		},
		{
			s: api.TaskSpec{
				Runtime: &api.TaskSpec_Attachment{
					Attachment: &api.NetworkAttachmentSpec{},
				},
			},
			c: codes.Unimplemented,
		},
		{
			s: createSpec("", "", 0).Task,
			c: codes.InvalidArgument,
		},
		{
			s: createSpec("", "busybox###", 0).Task,
			c: codes.InvalidArgument,
		},
		{
			s: createGenericSpec("name", "").Task,
			c: codes.InvalidArgument,
		},
		{
			s: createGenericSpec("name", "c").Task,
			c: codes.InvalidArgument,
		},
		{
			s: createSpecWithDuplicateMounts("test").Task,
			c: codes.InvalidArgument,
		},
		{
			s: createSpecWithHostnameTemplate("", "{{.Nothing.here}}").Task,
			c: codes.InvalidArgument,
		},
	} {
		err := validateTaskSpec(bad.s)
		assert.Error(t, err)
		assert.Equal(t, bad.c, grpc.Code(err))
	}

	for _, good := range []api.TaskSpec{
		createSpec("", "image", 0).Task,
		createGenericSpec("", "custom").Task,
		createSpecWithHostnameTemplate("service", "{{.Service.Name}}-{{.Task.Slot}}").Task,
	} {
		err := validateTaskSpec(good)
		assert.NoError(t, err)
	}
}

func TestValidateContainerSpec(t *testing.T) {
	type BadSpec struct {
		spec api.TaskSpec
		c    codes.Code
	}

	bad1 := api.TaskSpec{
		Runtime: &api.TaskSpec_Container{
			Container: &api.ContainerSpec{
				Image: "", // image name should not be empty
			},
		},
	}

	bad2 := api.TaskSpec{
		Runtime: &api.TaskSpec_Container{
			Container: &api.ContainerSpec{
				Image: "image",
				Mounts: []api.Mount{
					{
						Type:   api.Mount_MountType(0),
						Source: "/data",
						Target: "/data",
					},
					{
						Type:   api.Mount_MountType(0),
						Source: "/data2",
						Target: "/data", // duplicate mount point
					},
				},
			},
		},
	}

	bad3 := api.TaskSpec{
		Runtime: &api.TaskSpec_Container{
			Container: &api.ContainerSpec{
				Image: "image",
				Healthcheck: &api.HealthConfig{
					Test:        []string{"curl 127.0.0.1:3000"},
					Interval:    gogotypes.DurationProto(time.Duration(-1 * time.Second)), // invalid negative duration
					Timeout:     gogotypes.DurationProto(time.Duration(-1 * time.Second)), // invalid negative duration
					Retries:     -1,                                                       // invalid negative integer
					StartPeriod: gogotypes.DurationProto(time.Duration(-1 * time.Second)), // invalid negative duration
				},
			},
		},
	}

	for _, bad := range []BadSpec{
		{
			spec: bad1,
			c:    codes.InvalidArgument,
		},
		{
			spec: bad2,
			c:    codes.InvalidArgument,
		},
		{
			spec: bad3,
			c:    codes.InvalidArgument,
		},
	} {
		err := validateContainerSpec(bad.spec)
		assert.Error(t, err)
		assert.Equal(t, bad.c, grpc.Code(err), grpc.ErrorDesc(err))
	}

	good1 := api.TaskSpec{
		Runtime: &api.TaskSpec_Container{
			Container: &api.ContainerSpec{
				Image: "image",
				Mounts: []api.Mount{
					{
						Type:   api.Mount_MountType(0),
						Source: "/data",
						Target: "/data",
					},
					{
						Type:   api.Mount_MountType(0),
						Source: "/data2",
						Target: "/data2",
					},
				},
				Healthcheck: &api.HealthConfig{
					Test:        []string{"curl 127.0.0.1:3000"},
					Interval:    gogotypes.DurationProto(time.Duration(1 * time.Second)),
					Timeout:     gogotypes.DurationProto(time.Duration(3 * time.Second)),
					Retries:     5,
					StartPeriod: gogotypes.DurationProto(time.Duration(1 * time.Second)),
				},
			},
		},
	}

	for _, good := range []api.TaskSpec{good1} {
		err := validateContainerSpec(good)
		assert.NoError(t, err)
	}
}

func TestValidateServiceSpec(t *testing.T) {
	type BadServiceSpec struct {
		spec *api.ServiceSpec
		c    codes.Code
	}

	for _, bad := range []BadServiceSpec{
		{
			spec: nil,
			c:    codes.InvalidArgument,
		},
		{
			spec: &api.ServiceSpec{Annotations: api.Annotations{Name: "name"}},
			c:    codes.InvalidArgument,
		},
		{
			spec: createSpec("", "", 1),
			c:    codes.InvalidArgument,
		},
		{
			spec: createSpec("name", "", 1),
			c:    codes.InvalidArgument,
		},
		{
			spec: createSpec("", "image", 1),
			c:    codes.InvalidArgument,
		},
		{
			spec: createSpec(strings.Repeat("longname", 8), "image", 1),
			c:    codes.InvalidArgument,
		},
	} {
		err := validateServiceSpec(bad.spec)
		assert.Error(t, err)
		assert.Equal(t, bad.c, grpc.Code(err), grpc.ErrorDesc(err))
	}

	for _, good := range []*api.ServiceSpec{
		createSpec("name", "image", 1),
	} {
		err := validateServiceSpec(good)
		assert.NoError(t, err)
	}
}

func TestValidateRestartPolicy(t *testing.T) {
	bad := []*api.RestartPolicy{
		{
			Delay:  gogotypes.DurationProto(time.Duration(-1 * time.Second)),
			Window: gogotypes.DurationProto(time.Duration(-1 * time.Second)),
		},
		{
			Delay:  gogotypes.DurationProto(time.Duration(20 * time.Second)),
			Window: gogotypes.DurationProto(time.Duration(-4 * time.Second)),
		},
	}

	good := []*api.RestartPolicy{
		{
			Delay:  gogotypes.DurationProto(time.Duration(10 * time.Second)),
			Window: gogotypes.DurationProto(time.Duration(1 * time.Second)),
		},
	}

	for _, b := range bad {
		err := validateRestartPolicy(b)
		assert.Error(t, err)
		assert.Equal(t, codes.InvalidArgument, grpc.Code(err))
	}

	for _, g := range good {
		assert.NoError(t, validateRestartPolicy(g))
	}
}

func TestValidateUpdate(t *testing.T) {
	bad := []*api.UpdateConfig{
		{Delay: -1 * time.Second},
		{Delay: -1000 * time.Second},
		{Monitor: gogotypes.DurationProto(time.Duration(-1 * time.Second))},
		{Monitor: gogotypes.DurationProto(time.Duration(-1000 * time.Second))},
		{MaxFailureRatio: -0.1},
		{MaxFailureRatio: 1.1},
	}

	good := []*api.UpdateConfig{
		{Delay: time.Second},
		{Monitor: gogotypes.DurationProto(time.Duration(time.Second))},
		{MaxFailureRatio: 0.5},
	}

	for _, b := range bad {
		err := validateUpdate(b)
		assert.Error(t, err)
		assert.Equal(t, codes.InvalidArgument, grpc.Code(err))
	}

	for _, g := range good {
		assert.NoError(t, validateUpdate(g))
	}
}

func TestCreateService(t *testing.T) {
	ts := newTestServer(t)
	defer ts.Stop()
	_, err := ts.Client.CreateService(context.Background(), &api.CreateServiceRequest{})
	assert.Error(t, err)
	assert.Equal(t, codes.InvalidArgument, grpc.Code(err))

	spec := createSpec("name", "image", 1)
	r, err := ts.Client.CreateService(context.Background(), &api.CreateServiceRequest{Spec: spec})
	assert.NoError(t, err)
	assert.NotEmpty(t, r.Service.ID)

	// test port conflicts
	spec = createSpec("name2", "image", 1)
	spec.Endpoint = &api.EndpointSpec{Ports: []*api.PortConfig{
		{PublishedPort: uint32(9000), TargetPort: uint32(9000), Protocol: api.PortConfig_Protocol(api.ProtocolTCP)},
	}}
	r, err = ts.Client.CreateService(context.Background(), &api.CreateServiceRequest{Spec: spec})
	assert.NoError(t, err)
	assert.NotEmpty(t, r.Service.ID)

	spec2 := createSpec("name3", "image", 1)
	spec2.Endpoint = &api.EndpointSpec{Ports: []*api.PortConfig{
		{PublishedPort: uint32(9000), TargetPort: uint32(9000), Protocol: api.PortConfig_Protocol(api.ProtocolTCP)},
	}}
	_, err = ts.Client.CreateService(context.Background(), &api.CreateServiceRequest{Spec: spec2})
	assert.Error(t, err)
	assert.Equal(t, codes.InvalidArgument, grpc.Code(err))

	// test no port conflicts when no publish port is specified
	spec3 := createSpec("name4", "image", 1)
	spec3.Endpoint = &api.EndpointSpec{Ports: []*api.PortConfig{
		{TargetPort: uint32(9000), Protocol: api.PortConfig_Protocol(api.ProtocolTCP)},
	}}
	r, err = ts.Client.CreateService(context.Background(), &api.CreateServiceRequest{Spec: spec3})
	assert.NoError(t, err)
	assert.NotEmpty(t, r.Service.ID)
	spec4 := createSpec("name5", "image", 1)
	spec4.Endpoint = &api.EndpointSpec{Ports: []*api.PortConfig{
		{TargetPort: uint32(9001), Protocol: api.PortConfig_Protocol(api.ProtocolTCP)},
	}}
	_, err = ts.Client.CreateService(context.Background(), &api.CreateServiceRequest{Spec: spec4})
	assert.NoError(t, err)

	// ensure no port conflict when different protocols are used
	spec = createSpec("name6", "image", 1)
	spec.Endpoint = &api.EndpointSpec{Ports: []*api.PortConfig{
		{PublishedPort: uint32(9100), TargetPort: uint32(9100), Protocol: api.PortConfig_Protocol(api.ProtocolTCP)},
	}}
	r, err = ts.Client.CreateService(context.Background(), &api.CreateServiceRequest{Spec: spec})
	assert.NoError(t, err)
	assert.NotEmpty(t, r.Service.ID)

	spec2 = createSpec("name7", "image", 1)
	spec2.Endpoint = &api.EndpointSpec{Ports: []*api.PortConfig{
		{PublishedPort: uint32(9100), TargetPort: uint32(9100), Protocol: api.PortConfig_Protocol(api.ProtocolUDP)},
	}}
	_, err = ts.Client.CreateService(context.Background(), &api.CreateServiceRequest{Spec: spec2})
	assert.NoError(t, err)

	// ensure no port conflict when host ports overlap
	spec = createSpec("name8", "image", 1)
	spec.Endpoint = &api.EndpointSpec{Ports: []*api.PortConfig{
		{PublishMode: api.PublishModeHost, PublishedPort: uint32(9101), TargetPort: uint32(9101), Protocol: api.PortConfig_Protocol(api.ProtocolTCP)},
	}}
	r, err = ts.Client.CreateService(context.Background(), &api.CreateServiceRequest{Spec: spec})
	assert.NoError(t, err)
	assert.NotEmpty(t, r.Service.ID)

	spec2 = createSpec("name9", "image", 1)
	spec2.Endpoint = &api.EndpointSpec{Ports: []*api.PortConfig{
		{PublishMode: api.PublishModeHost, PublishedPort: uint32(9101), TargetPort: uint32(9101), Protocol: api.PortConfig_Protocol(api.ProtocolTCP)},
	}}
	_, err = ts.Client.CreateService(context.Background(), &api.CreateServiceRequest{Spec: spec2})
	assert.NoError(t, err)

	// ensure port conflict when host ports overlaps with ingress port (host port first)
	spec = createSpec("name10", "image", 1)
	spec.Endpoint = &api.EndpointSpec{Ports: []*api.PortConfig{
		{PublishMode: api.PublishModeHost, PublishedPort: uint32(9102), TargetPort: uint32(9102), Protocol: api.PortConfig_Protocol(api.ProtocolTCP)},
	}}
	r, err = ts.Client.CreateService(context.Background(), &api.CreateServiceRequest{Spec: spec})
	assert.NoError(t, err)
	assert.NotEmpty(t, r.Service.ID)

	spec2 = createSpec("name11", "image", 1)
	spec2.Endpoint = &api.EndpointSpec{Ports: []*api.PortConfig{
		{PublishMode: api.PublishModeIngress, PublishedPort: uint32(9102), TargetPort: uint32(9102), Protocol: api.PortConfig_Protocol(api.ProtocolTCP)},
	}}
	_, err = ts.Client.CreateService(context.Background(), &api.CreateServiceRequest{Spec: spec2})
	assert.Error(t, err)
	assert.Equal(t, codes.InvalidArgument, grpc.Code(err))

	// ensure port conflict when host ports overlaps with ingress port (ingress port first)
	spec = createSpec("name12", "image", 1)
	spec.Endpoint = &api.EndpointSpec{Ports: []*api.PortConfig{
		{PublishMode: api.PublishModeIngress, PublishedPort: uint32(9103), TargetPort: uint32(9103), Protocol: api.PortConfig_Protocol(api.ProtocolTCP)},
	}}
	r, err = ts.Client.CreateService(context.Background(), &api.CreateServiceRequest{Spec: spec})
	assert.NoError(t, err)
	assert.NotEmpty(t, r.Service.ID)

	spec2 = createSpec("name13", "image", 1)
	spec2.Endpoint = &api.EndpointSpec{Ports: []*api.PortConfig{
		{PublishMode: api.PublishModeHost, PublishedPort: uint32(9103), TargetPort: uint32(9103), Protocol: api.PortConfig_Protocol(api.ProtocolTCP)},
	}}
	_, err = ts.Client.CreateService(context.Background(), &api.CreateServiceRequest{Spec: spec2})
	assert.Error(t, err)
	assert.Equal(t, codes.InvalidArgument, grpc.Code(err))

	// ingress network cannot be attached explicitly
	spec = createSpec("name14", "image", 1)
	spec.Task.Networks = []*api.NetworkAttachmentConfig{{Target: getIngressTargetID(t, ts)}}
	_, err = ts.Client.CreateService(context.Background(), &api.CreateServiceRequest{Spec: spec})
	assert.Error(t, err)
	assert.Equal(t, codes.InvalidArgument, grpc.Code(err))
}

func TestSecretValidation(t *testing.T) {
	ts := newTestServer(t)
	defer ts.Stop()

	// test creating service with a secret that doesn't exist fails
	secretRef := createSecret(t, ts, "secret", "secret.txt")
	secretRef.SecretID = "404"
	secretRef.SecretName = "404"
	serviceSpec := createServiceSpecWithSecrets("service", secretRef)
	_, err := ts.Client.CreateService(context.Background(), &api.CreateServiceRequest{Spec: serviceSpec})
	assert.Equal(t, codes.InvalidArgument, grpc.Code(err))

	// test creating service with a secretRef that has an existing secret
	// but mismatched SecretName fails.
	secretRef1 := createSecret(t, ts, "secret1", "secret1.txt")
	secretRef1.SecretName = "secret2"
	serviceSpec = createServiceSpecWithSecrets("service1", secretRef1)
	_, err = ts.Client.CreateService(context.Background(), &api.CreateServiceRequest{Spec: serviceSpec})
	assert.Equal(t, codes.InvalidArgument, grpc.Code(err))

	// test secret target conflicts
	secretRef2 := createSecret(t, ts, "secret2", "secret2.txt")
	secretRef3 := createSecret(t, ts, "secret3", "secret2.txt")
	serviceSpec = createServiceSpecWithSecrets("service2", secretRef2, secretRef3)
	_, err = ts.Client.CreateService(context.Background(), &api.CreateServiceRequest{Spec: serviceSpec})
	assert.Equal(t, codes.InvalidArgument, grpc.Code(err))

	// test secret target conflicts with same secret and two references
	secretRef3.SecretID = secretRef2.SecretID
	secretRef3.SecretName = secretRef2.SecretName
	serviceSpec = createServiceSpecWithSecrets("service3", secretRef2, secretRef3)
	_, err = ts.Client.CreateService(context.Background(), &api.CreateServiceRequest{Spec: serviceSpec})
	assert.Equal(t, codes.InvalidArgument, grpc.Code(err))

	// test two different secretReferences with using the same secret
	secretRef5 := secretRef2.Copy()
	secretRef5.Target = &api.SecretReference_File{
		File: &api.FileTarget{
			Name: "different-target",
		},
	}

	serviceSpec = createServiceSpecWithSecrets("service4", secretRef2, secretRef5)
	_, err = ts.Client.CreateService(context.Background(), &api.CreateServiceRequest{Spec: serviceSpec})
	assert.NoError(t, err)

	// test secret References with invalid filenames
	secretRefBlank := createSecret(t, ts, "", "")

	serviceSpec = createServiceSpecWithSecrets("invalid-blank", secretRefBlank)
	_, err = ts.Client.CreateService(context.Background(), &api.CreateServiceRequest{Spec: serviceSpec})
	assert.Equal(t, codes.InvalidArgument, grpc.Code(err))

	// Test secret References with valid filenames
	// Note: "../secretfile.txt", "../../secretfile.txt" will be rejected
	// by the executor, but controlapi presently doesn't reject those names.
	// Such validation would be platform-specific.
	validFileNames := []string{"file.txt", ".file.txt", "_file-txt_.txt", "../secretfile.txt", "../../secretfile.txt", "file../.txt", "subdir/file.txt", "/file.txt"}
	for i, validName := range validFileNames {
		secretRef := createSecret(t, ts, validName, validName)

		serviceSpec = createServiceSpecWithSecrets(fmt.Sprintf("valid%v", i), secretRef)
		_, err = ts.Client.CreateService(context.Background(), &api.CreateServiceRequest{Spec: serviceSpec})
		assert.NoError(t, err)
	}

	// test secret target conflicts on update
	serviceSpec1 := createServiceSpecWithSecrets("service5", secretRef2, secretRef3)
	// Copy this service, but delete the secrets for creation
	serviceSpec2 := serviceSpec1.Copy()
	serviceSpec2.Task.GetContainer().Secrets = nil
	rs, err := ts.Client.CreateService(context.Background(), &api.CreateServiceRequest{Spec: serviceSpec2})
	assert.NoError(t, err)

	// Attempt to update to the originally intended (conflicting) spec
	_, err = ts.Client.UpdateService(context.Background(), &api.UpdateServiceRequest{
		ServiceID:      rs.Service.ID,
		Spec:           serviceSpec1,
		ServiceVersion: &rs.Service.Meta.Version,
	})
	assert.Equal(t, codes.InvalidArgument, grpc.Code(err))
}

func TestConfigValidation(t *testing.T) {
	ts := newTestServer(t)
	defer ts.Stop()

	// test creating service with a config that doesn't exist fails
	configRef := createConfig(t, ts, "config", "config.txt")
	configRef.ConfigID = "404"
	configRef.ConfigName = "404"
	serviceSpec := createServiceSpecWithConfigs("service", configRef)
	_, err := ts.Client.CreateService(context.Background(), &api.CreateServiceRequest{Spec: serviceSpec})
	assert.Equal(t, codes.InvalidArgument, grpc.Code(err))

	// test creating service with a configRef that has an existing config
	// but mismatched ConfigName fails.
	configRef1 := createConfig(t, ts, "config1", "config1.txt")
	configRef1.ConfigName = "config2"
	serviceSpec = createServiceSpecWithConfigs("service1", configRef1)
	_, err = ts.Client.CreateService(context.Background(), &api.CreateServiceRequest{Spec: serviceSpec})
	assert.Equal(t, codes.InvalidArgument, grpc.Code(err))

	// test config target conflicts
	configRef2 := createConfig(t, ts, "config2", "config2.txt")
	configRef3 := createConfig(t, ts, "config3", "config2.txt")
	serviceSpec = createServiceSpecWithConfigs("service2", configRef2, configRef3)
	_, err = ts.Client.CreateService(context.Background(), &api.CreateServiceRequest{Spec: serviceSpec})
	assert.Equal(t, codes.InvalidArgument, grpc.Code(err))

	// test config target conflicts with same config and two references
	configRef3.ConfigID = configRef2.ConfigID
	configRef3.ConfigName = configRef2.ConfigName
	serviceSpec = createServiceSpecWithConfigs("service3", configRef2, configRef3)
	_, err = ts.Client.CreateService(context.Background(), &api.CreateServiceRequest{Spec: serviceSpec})
	assert.Equal(t, codes.InvalidArgument, grpc.Code(err))

	// test two different configReferences with using the same config
	configRef5 := configRef2.Copy()
	configRef5.Target = &api.ConfigReference_File{
		File: &api.FileTarget{
			Name: "different-target",
		},
	}

	serviceSpec = createServiceSpecWithConfigs("service4", configRef2, configRef5)
	_, err = ts.Client.CreateService(context.Background(), &api.CreateServiceRequest{Spec: serviceSpec})
	assert.NoError(t, err)

	// Test config References with valid filenames
	// TODO(aaronl): Should some of these be disallowed? How can we deal
	// with Windows-style paths on a Linux manager or vice versa?
	validFileNames := []string{"../configfile.txt", "../../configfile.txt", "file../.txt", "subdir/file.txt", "file.txt", ".file.txt", "_file-txt_.txt"}
	for i, validName := range validFileNames {
		configRef := createConfig(t, ts, validName, validName)

		serviceSpec = createServiceSpecWithConfigs(fmt.Sprintf("valid%v", i), configRef)
		_, err = ts.Client.CreateService(context.Background(), &api.CreateServiceRequest{Spec: serviceSpec})
		assert.NoError(t, err)
	}

	// test config target conflicts on update
	serviceSpec1 := createServiceSpecWithConfigs("service5", configRef2, configRef3)
	// Copy this service, but delete the configs for creation
	serviceSpec2 := serviceSpec1.Copy()
	serviceSpec2.Task.GetContainer().Configs = nil
	rs, err := ts.Client.CreateService(context.Background(), &api.CreateServiceRequest{Spec: serviceSpec2})
	assert.NoError(t, err)

	// Attempt to update to the originally intended (conflicting) spec
	_, err = ts.Client.UpdateService(context.Background(), &api.UpdateServiceRequest{
		ServiceID:      rs.Service.ID,
		Spec:           serviceSpec1,
		ServiceVersion: &rs.Service.Meta.Version,
	})
	assert.Equal(t, codes.InvalidArgument, grpc.Code(err))
}

func TestGetService(t *testing.T) {
	ts := newTestServer(t)
	defer ts.Stop()
	_, err := ts.Client.GetService(context.Background(), &api.GetServiceRequest{})
	assert.Error(t, err)
	assert.Equal(t, codes.InvalidArgument, grpc.Code(err))

	_, err = ts.Client.GetService(context.Background(), &api.GetServiceRequest{ServiceID: "invalid"})
	assert.Error(t, err)
	assert.Equal(t, codes.NotFound, grpc.Code(err))

	service := createService(t, ts, "name", "image", 1)
	r, err := ts.Client.GetService(context.Background(), &api.GetServiceRequest{ServiceID: service.ID})
	assert.NoError(t, err)
	service.Meta.Version = r.Service.Meta.Version
	assert.Equal(t, service, r.Service)
}

func TestUpdateService(t *testing.T) {
	ts := newTestServer(t)
	defer ts.Stop()
	service := createService(t, ts, "name", "image", 1)

	_, err := ts.Client.UpdateService(context.Background(), &api.UpdateServiceRequest{})
	assert.Error(t, err)
	assert.Equal(t, codes.InvalidArgument, grpc.Code(err))

	_, err = ts.Client.UpdateService(context.Background(), &api.UpdateServiceRequest{ServiceID: "invalid", Spec: &service.Spec, ServiceVersion: &api.Version{}})
	assert.Error(t, err)
	assert.Equal(t, codes.NotFound, grpc.Code(err))

	// No update options.
	_, err = ts.Client.UpdateService(context.Background(), &api.UpdateServiceRequest{ServiceID: service.ID, Spec: &service.Spec})
	assert.Error(t, err)
	assert.Equal(t, codes.InvalidArgument, grpc.Code(err))

	_, err = ts.Client.UpdateService(context.Background(), &api.UpdateServiceRequest{ServiceID: service.ID, Spec: &service.Spec, ServiceVersion: &service.Meta.Version})
	assert.NoError(t, err)

	r, err := ts.Client.GetService(context.Background(), &api.GetServiceRequest{ServiceID: service.ID})
	assert.NoError(t, err)
	assert.Equal(t, service.Spec.Annotations.Name, r.Service.Spec.Annotations.Name)
	mode, ok := r.Service.Spec.GetMode().(*api.ServiceSpec_Replicated)
	assert.Equal(t, ok, true)
	assert.True(t, mode.Replicated.Replicas == 1)

	mode.Replicated.Replicas = 42
	_, err = ts.Client.UpdateService(context.Background(), &api.UpdateServiceRequest{
		ServiceID:      service.ID,
		Spec:           &r.Service.Spec,
		ServiceVersion: &r.Service.Meta.Version,
	})
	assert.NoError(t, err)

	r, err = ts.Client.GetService(context.Background(), &api.GetServiceRequest{ServiceID: service.ID})
	assert.NoError(t, err)
	assert.Equal(t, service.Spec.Annotations.Name, r.Service.Spec.Annotations.Name)
	mode, ok = r.Service.Spec.GetMode().(*api.ServiceSpec_Replicated)
	assert.Equal(t, ok, true)
	assert.True(t, mode.Replicated.Replicas == 42)

	// mode change not allowed
	r, err = ts.Client.GetService(context.Background(), &api.GetServiceRequest{ServiceID: service.ID})
	assert.NoError(t, err)
	r.Service.Spec.Mode = &api.ServiceSpec_Global{
		Global: &api.GlobalService{},
	}
	_, err = ts.Client.UpdateService(context.Background(), &api.UpdateServiceRequest{
		ServiceID:      service.ID,
		Spec:           &r.Service.Spec,
		ServiceVersion: &r.Service.Meta.Version,
	})
	assert.Error(t, err)
	assert.True(t, strings.Contains(err.Error(), errModeChangeNotAllowed.Error()))

	// Versioning.
	r, err = ts.Client.GetService(context.Background(), &api.GetServiceRequest{ServiceID: service.ID})
	assert.NoError(t, err)
	version := &r.Service.Meta.Version

	_, err = ts.Client.UpdateService(context.Background(), &api.UpdateServiceRequest{
		ServiceID:      service.ID,
		Spec:           &r.Service.Spec,
		ServiceVersion: version,
	})
	assert.NoError(t, err)

	// Perform an update with the "old" version.
	_, err = ts.Client.UpdateService(context.Background(), &api.UpdateServiceRequest{
		ServiceID:      service.ID,
		Spec:           &r.Service.Spec,
		ServiceVersion: version,
	})
	assert.Error(t, err)

	// test port conflicts
	spec2 := createSpec("name2", "image", 1)
	spec2.Endpoint = &api.EndpointSpec{Ports: []*api.PortConfig{
		{PublishedPort: uint32(9000), TargetPort: uint32(9000), Protocol: api.PortConfig_Protocol(api.ProtocolTCP)},
	}}
	_, err = ts.Client.CreateService(context.Background(), &api.CreateServiceRequest{Spec: spec2})
	assert.NoError(t, err)

	spec3 := createSpec("name3", "image", 1)
	rs, err := ts.Client.CreateService(context.Background(), &api.CreateServiceRequest{Spec: spec3})
	assert.NoError(t, err)

	spec3.Endpoint = &api.EndpointSpec{Ports: []*api.PortConfig{
		{PublishedPort: uint32(9000), TargetPort: uint32(9000), Protocol: api.PortConfig_Protocol(api.ProtocolTCP)},
	}}
	_, err = ts.Client.UpdateService(context.Background(), &api.UpdateServiceRequest{
		ServiceID:      rs.Service.ID,
		Spec:           spec3,
		ServiceVersion: &rs.Service.Meta.Version,
	})
	assert.Error(t, err)
	assert.Equal(t, codes.InvalidArgument, grpc.Code(err))
	spec3.Endpoint = &api.EndpointSpec{Ports: []*api.PortConfig{
		{PublishedPort: uint32(9001), TargetPort: uint32(9000), Protocol: api.PortConfig_Protocol(api.ProtocolTCP)},
	}}
	_, err = ts.Client.UpdateService(context.Background(), &api.UpdateServiceRequest{
		ServiceID:      rs.Service.ID,
		Spec:           spec3,
		ServiceVersion: &rs.Service.Meta.Version,
	})
	assert.NoError(t, err)

	// ingress network cannot be attached explicitly
	spec4 := createSpec("name4", "image", 1)
	rs, err = ts.Client.CreateService(context.Background(), &api.CreateServiceRequest{Spec: spec4})
	assert.NoError(t, err)
	spec4.Task.Networks = []*api.NetworkAttachmentConfig{{Target: getIngressTargetID(t, ts)}}
	_, err = ts.Client.UpdateService(context.Background(), &api.UpdateServiceRequest{
		ServiceID:      rs.Service.ID,
		Spec:           spec4,
		ServiceVersion: &rs.Service.Meta.Version,
	})
	assert.Error(t, err)
	assert.Equal(t, codes.InvalidArgument, grpc.Code(err))
}

func TestServiceUpdateRejectNetworkChange(t *testing.T) {
	ts := newTestServer(t)
	defer ts.Stop()
	spec := createSpec("name1", "image", 1)
	spec.Networks = []*api.NetworkAttachmentConfig{
		{
			Target: "net20",
		},
	}
	cr, err := ts.Client.CreateService(context.Background(), &api.CreateServiceRequest{Spec: spec})
	assert.NoError(t, err)

	ur, err := ts.Client.GetService(context.Background(), &api.GetServiceRequest{ServiceID: cr.Service.ID})
	assert.NoError(t, err)
	service := ur.Service

	service.Spec.Networks[0].Target = "net30"

	_, err = ts.Client.UpdateService(context.Background(), &api.UpdateServiceRequest{
		ServiceID:      service.ID,
		Spec:           &service.Spec,
		ServiceVersion: &service.Meta.Version,
	})
	assert.Error(t, err)
	assert.True(t, strings.Contains(err.Error(), errNetworkUpdateNotSupported.Error()))

	// Changes to TaskSpec.Networks are allowed
	spec = createSpec("name2", "image", 1)
	spec.Task.Networks = []*api.NetworkAttachmentConfig{
		{
			Target: "net20",
		},
	}
	cr, err = ts.Client.CreateService(context.Background(), &api.CreateServiceRequest{Spec: spec})
	assert.NoError(t, err)

	ur, err = ts.Client.GetService(context.Background(), &api.GetServiceRequest{ServiceID: cr.Service.ID})
	assert.NoError(t, err)
	service = ur.Service

	service.Spec.Task.Networks[0].Target = "net30"

	_, err = ts.Client.UpdateService(context.Background(), &api.UpdateServiceRequest{
		ServiceID:      service.ID,
		Spec:           &service.Spec,
		ServiceVersion: &service.Meta.Version,
	})
	assert.NoError(t, err)

	// Migrate networks from ServiceSpec.Networks to TaskSpec.Networks
	spec = createSpec("name3", "image", 1)
	spec.Networks = []*api.NetworkAttachmentConfig{
		{
			Target: "net20",
		},
	}
	cr, err = ts.Client.CreateService(context.Background(), &api.CreateServiceRequest{Spec: spec})
	assert.NoError(t, err)

	ur, err = ts.Client.GetService(context.Background(), &api.GetServiceRequest{ServiceID: cr.Service.ID})
	assert.NoError(t, err)
	service = ur.Service

	service.Spec.Task.Networks = spec.Networks
	service.Spec.Networks = nil

	_, err = ts.Client.UpdateService(context.Background(), &api.UpdateServiceRequest{
		ServiceID:      service.ID,
		Spec:           &service.Spec,
		ServiceVersion: &service.Meta.Version,
	})
	assert.NoError(t, err)
}

func TestRemoveService(t *testing.T) {
	ts := newTestServer(t)
	defer ts.Stop()
	_, err := ts.Client.RemoveService(context.Background(), &api.RemoveServiceRequest{})
	assert.Error(t, err)
	assert.Equal(t, codes.InvalidArgument, grpc.Code(err))

	service := createService(t, ts, "name", "image", 1)
	r, err := ts.Client.RemoveService(context.Background(), &api.RemoveServiceRequest{ServiceID: service.ID})
	assert.NoError(t, err)
	assert.NotNil(t, r)
}

func TestValidateEndpointSpec(t *testing.T) {
	endPointSpec1 := &api.EndpointSpec{
		Mode: api.ResolutionModeDNSRoundRobin,
		Ports: []*api.PortConfig{
			{
				Name:       "http",
				TargetPort: 80,
			},
		},
	}

	endPointSpec2 := &api.EndpointSpec{
		Mode: api.ResolutionModeVirtualIP,
		Ports: []*api.PortConfig{
			{
				Name:          "http",
				TargetPort:    81,
				PublishedPort: 8001,
			},
			{
				Name:          "http",
				TargetPort:    80,
				PublishedPort: 8000,
			},
		},
	}

	// has duplicated published port, invalid
	endPointSpec3 := &api.EndpointSpec{
		Mode: api.ResolutionModeVirtualIP,
		Ports: []*api.PortConfig{
			{
				Name:          "http",
				TargetPort:    81,
				PublishedPort: 8001,
			},
			{
				Name:          "http",
				TargetPort:    80,
				PublishedPort: 8001,
			},
		},
	}

	// duplicated published port but different protocols, valid
	endPointSpec4 := &api.EndpointSpec{
		Mode: api.ResolutionModeVirtualIP,
		Ports: []*api.PortConfig{
			{
				Name:          "dns",
				TargetPort:    53,
				PublishedPort: 8002,
				Protocol:      api.ProtocolTCP,
			},
			{
				Name:          "dns",
				TargetPort:    53,
				PublishedPort: 8002,
				Protocol:      api.ProtocolUDP,
			},
		},
	}

	// multiple randomly assigned published ports
	endPointSpec5 := &api.EndpointSpec{
		Mode: api.ResolutionModeVirtualIP,
		Ports: []*api.PortConfig{
			{
				Name:       "http",
				TargetPort: 80,
				Protocol:   api.ProtocolTCP,
			},
			{
				Name:       "dns",
				TargetPort: 53,
				Protocol:   api.ProtocolUDP,
			},
			{
				Name:       "dns",
				TargetPort: 53,
				Protocol:   api.ProtocolTCP,
			},
		},
	}

	err := validateEndpointSpec(endPointSpec1)
	assert.Error(t, err)
	assert.Equal(t, codes.InvalidArgument, grpc.Code(err))

	err = validateEndpointSpec(endPointSpec2)
	assert.NoError(t, err)

	err = validateEndpointSpec(endPointSpec3)
	assert.Error(t, err)
	assert.Equal(t, codes.InvalidArgument, grpc.Code(err))

	err = validateEndpointSpec(endPointSpec4)
	assert.NoError(t, err)

	err = validateEndpointSpec(endPointSpec5)
	assert.NoError(t, err)
}

func TestServiceEndpointSpecUpdate(t *testing.T) {
	ts := newTestServer(t)
	defer ts.Stop()
	spec := &api.ServiceSpec{
		Annotations: api.Annotations{
			Name: "name",
		},
		Task: api.TaskSpec{
			Runtime: &api.TaskSpec_Container{
				Container: &api.ContainerSpec{
					Image: "image",
				},
			},
		},
		Mode: &api.ServiceSpec_Replicated{
			Replicated: &api.ReplicatedService{
				Replicas: 1,
			},
		},
		Endpoint: &api.EndpointSpec{
			Ports: []*api.PortConfig{
				{
					Name:       "http",
					TargetPort: 80,
				},
			},
		},
	}

	r, err := ts.Client.CreateService(context.Background(),
		&api.CreateServiceRequest{Spec: spec})
	assert.NoError(t, err)
	assert.NotNil(t, r)

	// Update the service with duplicate ports
	spec.Endpoint.Ports = append(spec.Endpoint.Ports, &api.PortConfig{
		Name:       "fakehttp",
		TargetPort: 80,
	})
	_, err = ts.Client.UpdateService(context.Background(),
		&api.UpdateServiceRequest{Spec: spec})
	assert.Error(t, err)
	assert.Equal(t, codes.InvalidArgument, grpc.Code(err))
}

func TestListServices(t *testing.T) {
	ts := newTestServer(t)
	defer ts.Stop()
	r, err := ts.Client.ListServices(context.Background(), &api.ListServicesRequest{})
	assert.NoError(t, err)
	assert.Empty(t, r.Services)

	s1 := createService(t, ts, "name1", "image", 1)
	r, err = ts.Client.ListServices(context.Background(), &api.ListServicesRequest{})
	assert.NoError(t, err)
	assert.Equal(t, 1, len(r.Services))

	createService(t, ts, "name2", "image", 1)
	s3 := createGenericService(t, ts, "name3", "my-runtime")

	// List all.
	r, err = ts.Client.ListServices(context.Background(), &api.ListServicesRequest{})
	assert.NoError(t, err)
	assert.Equal(t, 3, len(r.Services))

	// List by runtime.
	r, err = ts.Client.ListServices(context.Background(), &api.ListServicesRequest{
		Filters: &api.ListServicesRequest_Filters{
			Runtimes: []string{"container"},
		},
	})
	assert.NoError(t, err)
	assert.Equal(t, 2, len(r.Services))

	r, err = ts.Client.ListServices(context.Background(), &api.ListServicesRequest{
		Filters: &api.ListServicesRequest_Filters{
			Runtimes: []string{"my-runtime"},
		},
	})
	assert.NoError(t, err)
	assert.Equal(t, 1, len(r.Services))
	assert.Equal(t, s3.ID, r.Services[0].ID)

	r, err = ts.Client.ListServices(context.Background(), &api.ListServicesRequest{
		Filters: &api.ListServicesRequest_Filters{
			Runtimes: []string{"invalid"},
		},
	})
	assert.NoError(t, err)
	assert.Empty(t, r.Services)

	// List with an ID prefix.
	r, err = ts.Client.ListServices(context.Background(), &api.ListServicesRequest{
		Filters: &api.ListServicesRequest_Filters{
			IDPrefixes: []string{s1.ID[0:4]},
		},
	})
	assert.NoError(t, err)
	assert.Equal(t, 1, len(r.Services))
	assert.Equal(t, s1.ID, r.Services[0].ID)

	// List with simple filter.
	r, err = ts.Client.ListServices(context.Background(), &api.ListServicesRequest{
		Filters: &api.ListServicesRequest_Filters{
			NamePrefixes: []string{"name1"},
		},
	})
	assert.NoError(t, err)
	assert.Equal(t, 1, len(r.Services))

	// List with union filter.
	r, err = ts.Client.ListServices(context.Background(), &api.ListServicesRequest{
		Filters: &api.ListServicesRequest_Filters{
			NamePrefixes: []string{"name1", "name2"},
		},
	})
	assert.NoError(t, err)
	assert.Equal(t, 2, len(r.Services))

	r, err = ts.Client.ListServices(context.Background(), &api.ListServicesRequest{
		Filters: &api.ListServicesRequest_Filters{
			NamePrefixes: []string{"name1", "name2", "name4"},
		},
	})
	assert.NoError(t, err)
	assert.Equal(t, 2, len(r.Services))

	r, err = ts.Client.ListServices(context.Background(), &api.ListServicesRequest{
		Filters: &api.ListServicesRequest_Filters{
			NamePrefixes: []string{"name4"},
		},
	})
	assert.NoError(t, err)
	assert.Equal(t, 0, len(r.Services))

	// List with filter intersection.
	r, err = ts.Client.ListServices(context.Background(),
		&api.ListServicesRequest{
			Filters: &api.ListServicesRequest_Filters{
				NamePrefixes: []string{"name1"},
				IDPrefixes:   []string{s1.ID},
			},
		},
	)
	assert.NoError(t, err)
	assert.Equal(t, 1, len(r.Services))

	r, err = ts.Client.ListServices(context.Background(),
		&api.ListServicesRequest{
			Filters: &api.ListServicesRequest_Filters{
				NamePrefixes: []string{"name2"},
				IDPrefixes:   []string{s1.ID},
			},
		},
	)
	assert.NoError(t, err)
	assert.Equal(t, 0, len(r.Services))

	r, err = ts.Client.ListServices(context.Background(),
		&api.ListServicesRequest{
			Filters: &api.ListServicesRequest_Filters{
				NamePrefixes: []string{"name3"},
				Runtimes:     []string{"my-runtime"},
			},
		},
	)
	assert.NoError(t, err)
	assert.Equal(t, 1, len(r.Services))

	// List filter by label.
	r, err = ts.Client.ListServices(context.Background(),
		&api.ListServicesRequest{
			Filters: &api.ListServicesRequest_Filters{
				Labels: map[string]string{
					"common": "yes",
				},
			},
		},
	)
	assert.NoError(t, err)
	assert.Equal(t, 3, len(r.Services))

	// Value-less label.
	r, err = ts.Client.ListServices(context.Background(),
		&api.ListServicesRequest{
			Filters: &api.ListServicesRequest_Filters{
				Labels: map[string]string{
					"common": "",
				},
			},
		},
	)
	assert.NoError(t, err)
	assert.Equal(t, 3, len(r.Services))

	// Label intersection.
	r, err = ts.Client.ListServices(context.Background(),
		&api.ListServicesRequest{
			Filters: &api.ListServicesRequest_Filters{
				Labels: map[string]string{
					"common": "",
					"unique": "name1",
				},
			},
		},
	)
	assert.NoError(t, err)
	assert.Equal(t, 1, len(r.Services))

	r, err = ts.Client.ListServices(context.Background(),
		&api.ListServicesRequest{
			Filters: &api.ListServicesRequest_Filters{
				Labels: map[string]string{
					"common": "",
					"unique": "error",
				},
			},
		},
	)
	assert.NoError(t, err)
	assert.Equal(t, 0, len(r.Services))
}
