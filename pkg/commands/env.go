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
	"github.com/containers/image/manifest"
	"github.com/docker/docker/builder/dockerfile/instructions"
	"github.com/sirupsen/logrus"
	"os"
	"strings"
)

type EnvCommand struct {
	cmd *instructions.EnvCommand
}

func (e *EnvCommand) ExecuteCommand(config *manifest.Schema2Config) error {
	logrus.Info("cmd: ENV")
	newEnvs := e.cmd.Env
	for _, pair := range newEnvs {
		logrus.Infof("Setting environment variable %s:%s", pair.Key, pair.Value)
		if err := os.Setenv(pair.Key, pair.Value); err != nil {
			return err
		}
	}
	return updateConfig(newEnvs, config)
}

func updateConfig(newEnvs []instructions.KeyValuePair, config *manifest.Schema2Config) error {
	// First, create map of current environment variables to prevent duplicates
	envMap := make(map[string]string)
	for _, env := range config.Env {
		entry := strings.Split(env, "=")
		envMap[entry[0]] = entry[1]
	}
	// Now, add new envs to the map
	for _, pair := range newEnvs {
		envMap[pair.Key] = pair.Value
	}

	// Now, convert back to array and modify config
	envArray := []string{}
	for key, value := range envMap {
		entry := key + "=" + value
		envArray = append(envArray, entry)
	}
	logrus.Debugf("Setting Env in config to %v", envArray)
	config.Env = envArray
	return nil
}

// We know that no files have changed, so return an empty array
func (e *EnvCommand) FilesToSnapshot() []string {
	return []string{}
}

// CreatedBy returns some information about the command for the image config history
func (e *EnvCommand) CreatedBy() string {
	envArray := []string{e.cmd.Name()}
	for _, pair := range e.cmd.Env {
		envArray = append(envArray, pair.Key+"="+pair.Value)
	}
	return strings.Join(envArray, " ")
}
