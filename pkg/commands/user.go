/*
Copyright 2018 Google LLC

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

package commands

import (
	"github.com/GoogleCloudPlatform/k8s-container-builder/pkg/util"
	"github.com/containers/image/manifest"
	"github.com/docker/docker/builder/dockerfile/instructions"
	"github.com/sirupsen/logrus"
	"os/user"
	"strings"
)

type UserCommand struct {
	cmd *instructions.UserCommand
}

func (r *UserCommand) ExecuteCommand(config *manifest.Schema2Config) error {
	logrus.Info("cmd: USER")
	u := r.cmd.User
	userAndGroup := strings.Split(u, ":")
	userStr, err := util.ResolveEnvironmentReplacement(userAndGroup[0], config.Env, false)
	if err != nil {
		return err
	}
	var groupStr string
	if len(userAndGroup) > 1 {
		groupStr, err = util.ResolveEnvironmentReplacement(userAndGroup[1], config.Env, false)
		if err != nil {
			return err
		}
	}

	// Lookup by username
	userObj, err := user.Lookup(userStr)
	if err != nil {
		if _, ok := err.(user.UnknownUserError); ok {
			// Lookup by id
			userObj, err = user.LookupId(userStr)
			if err != nil {
				return err
			}
		} else {
			return err
		}
	}

	// Same dance with groups
	var group *user.Group
	if groupStr != "" {
		group, err = user.LookupGroup(groupStr)
		if err != nil {
			if _, ok := err.(user.UnknownGroupError); ok {
				group, err = user.LookupGroupId(groupStr)
				if err != nil {
					return err
				}
			} else {
				return err
			}
		}
	}

	uid := userObj.Uid
	if group != nil {
		uid = uid + ":" + group.Gid
	}

	logrus.Infof("Setting user to %s", uid)
	config.User = uid
	return nil
}

func (r *UserCommand) FilesToSnapshot() []string {
	return []string{}
}

func (r *UserCommand) CreatedBy() string {
	s := []string{r.cmd.Name(), r.cmd.User}
	return strings.Join(s, " ")
}
