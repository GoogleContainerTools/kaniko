//go:build !linux
// +build !linux
package chrootuser

import (
	"errors"
)

func lookupUserInContainer(rootdir, username string) (uid uint64, gid uint64, err error) {
	return 0, 0, errors.New("lookupUserInContainer is only available on linux")
}	

func lookupGroupForUIDInContainer(rootdir string, userid uint64) (username string, gid uint64, err error) {
	return "", 0, errors.New("lookupGroupForUIDInContainer is only available on linux")
}

func lookupGroupInContainer(rootdir, groupname string) (gid uint64, err error) {
	return 0, errors.New("lookupGroupInContainer is only available on linux")
}

func lookupHomedirInContainer(rootdir string, uid uint64) (string, error) {
	return "", errors.New("lookupHomedirInContainer is only available on linux")
}
