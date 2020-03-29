package util

import (
	"strconv"
	"syscall"

	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
)

func SyscallCredentials(userStr string) (*syscall.Credential, error) {
	uid, gid, err := GetUIDAndGIDFromString(userStr, true)
	if err != nil {
		return nil, errors.Wrap(err, "get uid/gid")
	}

	u, err := Lookup(userStr)
	if err != nil {
		return nil, errors.Wrap(err, "lookup")
	}
	logrus.Infof("util.Lookup returned: %+v", u)

	var groups []uint32

	gidStr, err := groupIDs(u)
	if err != nil {
		return nil, errors.Wrap(err, "group ids for user")
	}

	for _, g := range gidStr {
		i, err := strconv.ParseUint(g, 10, 32)
		if err != nil {
			return nil, errors.Wrap(err, "parseuint")
		}

		groups = append(groups, uint32(i))
	}

	return &syscall.Credential{
		Uid:    uid,
		Gid:    gid,
		Groups: groups,
	}, nil
}
