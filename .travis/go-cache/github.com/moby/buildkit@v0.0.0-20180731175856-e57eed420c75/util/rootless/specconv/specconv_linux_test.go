package specconv

import (
	"testing"

	"github.com/opencontainers/runc/libcontainer/specconv"
	"github.com/opencontainers/runc/libcontainer/user"
	"github.com/opencontainers/runtime-spec/specs-go"
	"github.com/stretchr/testify/require"
)

func TestToRootless(t *testing.T) {
	spec := specconv.Example()
	uidMap := []user.IDMap{
		{
			ID:       0,
			ParentID: 4242,
			Count:    1,
		},
		{
			ID:       1,
			ParentID: 231072,
			Count:    65536,
		},
	}
	gidMap := uidMap
	expectedUIDMappings := []specs.LinuxIDMapping{
		{
			HostID:      0,
			ContainerID: 0,
			Size:        1,
		},
		{
			HostID:      1,
			ContainerID: 1,
			Size:        65536,
		},
	}
	err := toRootless(spec, uidMap, gidMap)
	require.NoError(t, err)
	require.EqualValues(t, expectedUIDMappings, spec.Linux.UIDMappings)
}
