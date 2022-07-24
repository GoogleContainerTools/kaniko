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
	"fmt"
	"strconv"
	"syscall"

	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
)

func SyscallCredentials(rootDir string, userGroupStr string) (*syscall.Credential, error) {
	uid, gid, err := getUIDAndGIDFromString(rootDir, userGroupStr, true)
	if err != nil {
		return nil, err
	}
	u, err := LookupUser(rootDir, fmt.Sprint(uid))
	if err != nil {
		return nil, errors.Wrap(err, "lookup")
	}
	logrus.Infof("Util.Lookup returned: %+v", u)

	// initiliaze empty
	groups := []uint32{}

	additionalGids, err := groupIDs(rootDir, u)
	if err != nil {
		return nil, errors.Wrap(err, "group ids for user")
	}

	for _, g := range additionalGids {
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
