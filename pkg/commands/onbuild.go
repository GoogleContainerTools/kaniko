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
	"github.com/GoogleContainerTools/kaniko/pkg/util"
	"github.com/docker/docker/builder/dockerfile/instructions"
	"github.com/google/go-containerregistry/v1"
	"github.com/sirupsen/logrus"
)

type OnBuildCommand struct {
	cmd *instructions.OnbuildCommand
}

//ExecuteCommand adds the specified expression in Onbuild to the config
func (o *OnBuildCommand) ExecuteCommand(config *v1.Config) error {
	logrus.Info("cmd: ONBUILD")
	logrus.Infof("args: %s", o.cmd.Expression)
	resolvedExpression, err := util.ResolveEnvironmentReplacement(o.cmd.Expression, config.Env, false)
	if err != nil {
		return err
	}
	if config.OnBuild == nil {
		config.OnBuild = []string{resolvedExpression}
	} else {
		config.OnBuild = append(config.OnBuild, resolvedExpression)
	}
	return nil
}

// FilesToSnapshot returns that no files have changed, this command only touches metadata.
func (o *OnBuildCommand) FilesToSnapshot() []string {
	return []string{}
}

// CreatedBy returns some information about the command for the image config history
func (o *OnBuildCommand) CreatedBy() string {
	return o.cmd.Expression
}
