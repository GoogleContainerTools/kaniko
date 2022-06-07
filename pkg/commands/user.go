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
	"fmt"
	"strings"

	"github.com/GoogleContainerTools/kaniko/pkg/dockerfile"
	"github.com/GoogleContainerTools/kaniko/pkg/util"
	v1 "github.com/google/go-containerregistry/pkg/v1"
	"github.com/moby/buildkit/frontend/dockerfile/instructions"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
)

type UserCommand struct {
	BaseCommand
	cmd *instructions.UserCommand
}

func (r *UserCommand) ExecuteCommand(config *v1.Config, buildArgs *dockerfile.BuildArgs) error {
	logrus.Info("Cmd: USER")
	u := r.cmd.User
	userAndGroup := strings.Split(u, ":")
	replacementEnvs := buildArgs.ReplacementEnvs(config.Env)
	userStr, err := util.ResolveEnvironmentReplacement(userAndGroup[0], replacementEnvs, false)
	if err != nil {
		return errors.Wrap(err, fmt.Sprintf("resolving user %s", userAndGroup[0]))
	}

	if len(userAndGroup) > 1 {
		groupStr, err := util.ResolveEnvironmentReplacement(userAndGroup[1], replacementEnvs, false)
		if err != nil {
			return errors.Wrap(err, fmt.Sprintf("resolving group %s", userAndGroup[1]))
		}
		userStr = userStr + ":" + groupStr
	}

	config.User = userStr
	return nil
}

func (r *UserCommand) String() string {
	return r.cmd.String()
}
