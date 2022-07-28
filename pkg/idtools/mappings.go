package idtools

const (
	subuidFile = "/etc/subuid"
	subgidFile = "/etc/subgid"
)

type Mapping struct {
	// ContainerID is the starting ID in the user namespace
	ContainerID uint32
	// HostID is the starting ID outside of the user namespace
	HostID uint32
	// Size is the number of IDs that can be mapped on top of ContainerID
	Size uint32
}
