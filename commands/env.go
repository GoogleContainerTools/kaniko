/*
Copyright 2018 Google, Inc. All rights reserved.

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
	"github.com/docker/docker/builder/dockerfile/instructions"
	"github.com/sirupsen/logrus"
	"os"
)

// EnvCommand represents the ENV command in a Dockerfile
type EnvCommand struct {
	cmd *instructions.EnvCommand
}

// ExecuteCommand executes the env command
func (e EnvCommand) ExecuteCommand() error {
	for _, env := range e.cmd.Env {
		key := env.Key
		value := env.Value
		if err := os.Setenv(key, value); err != nil {
			logrus.Debugf("Unable to set env variable %s: %s", key, value)
			return err
		}
	}
	return nil
}
