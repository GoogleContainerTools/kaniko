//go:build !linux
// +build !linux

package chrootuser

import (
	"errors"
	"os/user"
)

func lookupUserInContainer(rootdir, username string) (*lookupPasswdEntry, error) {
	return nil, errors.New("lookupUserInContainer is only available on linux")
}

func lookupGroupInContainer(rootdir, groupname string) (*lookupGroupEntry, error) {
	return nil, errors.New("lookupGroupInContainer is only available on linux")
}

func lookupHomedirInContainer(rootdir string, uid uint64) (string, error) {
	return "", errors.New("lookupHomedirInContainer is only available on linux")
}

func lookupAdditionalGroupsForUser(rootdir string, user *user.User) (gids []uint32, err error) {
	return nil, errors.New("lookupAdditionalGroupsForUser is only available on linux")
}
