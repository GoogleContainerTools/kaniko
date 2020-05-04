// +build linux darwin
// +build !cgo

package util

import (
	"bufio"
	"bytes"
	"io"
	"os"
	"os/user"
	"strconv"
	"strings"

	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
)

var groupFile = "/etc/group"

type group struct {
	id      string   // group ID
	name    string   // group name
	members []string // secondary group ids
}

// groupIDs returns all of the group ID's a user is a member of
func groupIDs(u *user.User) ([]string, error) {
	logrus.Infof("performing slow lookup of group ids for %s", u.Username)

	f, err := os.Open(groupFile)
	if err != nil {
		return nil, errors.Wrap(err, "open")
	}
	defer f.Close()

	gids := []string{u.Gid}

	for _, g := range localGroups(f) {
		for _, m := range g.members {
			if m == u.Username {
				gids = append(gids, g.id)
			}
		}
	}

	return gids, nil
}

// localGroups parses a reader in /etc/group form, returning parsed group data
// based on src/os/user/lookup_unix.go - but extended to include secondary groups
func localGroups(r io.Reader) []*group {
	var groups []*group

	bs := bufio.NewScanner(r)
	for bs.Scan() {
		line := bs.Bytes()

		// There's no spec for /etc/passwd or /etc/group, but we try to follow
		// the same rules as the glibc parser, which allows comments and blank
		// space at the beginning of a line.
		line = bytes.TrimSpace(line)
		if len(line) == 0 || line[0] == '#' {
			continue
		}

		// wheel:*:0:root,anotherGrp
		parts := strings.SplitN(string(line), ":", 4)
		if _, err := strconv.Atoi(parts[2]); err != nil {
			continue
		}

		groups = append(groups, &group{name: parts[0], id: parts[2], members: strings.Split(parts[3], ",")})
	}
	return groups
}
