package validate

import (
	"testing"

	"github.com/opencontainers/runc/libcontainer/configs"
)

func init() {
	geteuid = func() int { return 1337 }
	getegid = func() int { return 7331 }
}

func rootlessConfig() *configs.Config {
	return &configs.Config{
		Rootfs:   "/var",
		Rootless: true,
		Namespaces: configs.Namespaces(
			[]configs.Namespace{
				{Type: configs.NEWUSER},
			},
		),
		UidMappings: []configs.IDMap{
			{
				HostID:      geteuid(),
				ContainerID: 0,
				Size:        1,
			},
		},
		GidMappings: []configs.IDMap{
			{
				HostID:      getegid(),
				ContainerID: 0,
				Size:        1,
			},
		},
	}
}

func TestValidateRootless(t *testing.T) {
	validator := New()

	config := rootlessConfig()
	if err := validator.Validate(config); err != nil {
		t.Errorf("Expected error to not occur: %+v", err)
	}
}

/* rootlessMappings() */

func TestValidateRootlessUserns(t *testing.T) {
	validator := New()

	config := rootlessConfig()
	config.Namespaces = nil
	if err := validator.Validate(config); err == nil {
		t.Errorf("Expected error to occur if user namespaces not set")
	}
}

func TestValidateRootlessMappingUid(t *testing.T) {
	validator := New()

	config := rootlessConfig()
	config.UidMappings = nil
	if err := validator.Validate(config); err == nil {
		t.Errorf("Expected error to occur if no uid mappings provided")
	}
}

func TestValidateRootlessMappingGid(t *testing.T) {
	validator := New()

	config := rootlessConfig()
	config.GidMappings = nil
	if err := validator.Validate(config); err == nil {
		t.Errorf("Expected error to occur if no gid mappings provided")
	}
}

/* rootlessMount() */

func TestValidateRootlessMountUid(t *testing.T) {
	config := rootlessConfig()
	validator := New()

	config.Mounts = []*configs.Mount{
		{
			Source:      "devpts",
			Destination: "/dev/pts",
			Device:      "devpts",
		},
	}

	if err := validator.Validate(config); err != nil {
		t.Errorf("Expected error to not occur when uid= not set in mount options: %+v", err)
	}

	config.Mounts[0].Data = "uid=5"
	if err := validator.Validate(config); err == nil {
		t.Errorf("Expected error to occur when setting uid=5 in mount options")
	}

	config.Mounts[0].Data = "uid=0"
	if err := validator.Validate(config); err != nil {
		t.Errorf("Expected error to not occur when setting uid=0 in mount options: %+v", err)
	}

	config.Mounts[0].Data = "uid=2"
	config.UidMappings[0].Size = 10
	if err := validator.Validate(config); err != nil {
		t.Errorf("Expected error to not occur when setting uid=2 in mount options and UidMapping[0].size is 10")
	}

	config.Mounts[0].Data = "uid=20"
	config.UidMappings[0].Size = 10
	if err := validator.Validate(config); err == nil {
		t.Errorf("Expected error to occur when setting uid=20 in mount options and UidMapping[0].size is 10")
	}
}

func TestValidateRootlessMountGid(t *testing.T) {
	config := rootlessConfig()
	validator := New()

	config.Mounts = []*configs.Mount{
		{
			Source:      "devpts",
			Destination: "/dev/pts",
			Device:      "devpts",
		},
	}

	if err := validator.Validate(config); err != nil {
		t.Errorf("Expected error to not occur when gid= not set in mount options: %+v", err)
	}

	config.Mounts[0].Data = "gid=5"
	if err := validator.Validate(config); err == nil {
		t.Errorf("Expected error to occur when setting gid=5 in mount options")
	}

	config.Mounts[0].Data = "gid=0"
	if err := validator.Validate(config); err != nil {
		t.Errorf("Expected error to not occur when setting gid=0 in mount options: %+v", err)
	}

	config.Mounts[0].Data = "gid=5"
	config.GidMappings[0].Size = 10
	if err := validator.Validate(config); err != nil {
		t.Errorf("Expected error to not occur when setting gid=5 in mount options and GidMapping[0].size is 10")
	}

	config.Mounts[0].Data = "gid=11"
	config.GidMappings[0].Size = 10
	if err := validator.Validate(config); err == nil {
		t.Errorf("Expected error to occur when setting gid=11 in mount options and GidMapping[0].size is 10")
	}
}
