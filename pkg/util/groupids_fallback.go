//go:build (linux || darwin) && !cgo
// +build linux darwin
// +build !cgo

/*
Copyright 2020 Google LLC

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

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
	logrus.Infof("Performing slow lookup of group ids for %s", u.Username)

	// user can have no gid if it's a non existing user
	if u.Gid == "" {
		return []string{}, nil
	}

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
