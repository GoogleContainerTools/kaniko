//go:build !linux
// +build !linux

package idtools

import "errors"

func SetUidMap(pid int, uidmap Mapping) error {
	return errors.New("SetUidMap is only supported on linux")
}

func SetGidMap(pid int, gidmap Mapping) error {
	return errors.New("SetGidMap is only supported on linux")
}

func GetSubIDMappings(uid, gid uint32, user, group string) (Mapping, Mapping, error) {
	return errors.New("GetSubIDMappings is only supported on linux")
}
