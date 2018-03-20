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
	"bytes"
	"github.com/containers/image/manifest"
	"github.com/docker/docker/builder/dockerfile/instructions"
	"github.com/docker/docker/builder/dockerfile/parser"
	"github.com/docker/docker/builder/dockerfile/shell"
	"github.com/sirupsen/logrus"
	"os"
	"strings"
)

type EnvCommand struct {
	cmd *instructions.EnvCommand
}

func (e *EnvCommand) ExecuteCommand(config *manifest.Schema2Config) error {
	logrus.Info("cmd: ENV")
	// The dockerfile/shell package handles processing env values
	// It handles escape characters and supports expansion from the config.Env array
	// Shlex handles some of the following use cases (these and more are tested in integration tests)
	// ""a'b'c"" -> "a'b'c"
	// "Rex\ The\ Dog \" -> "Rex The Dog"
	// "a\"b" -> "a"b"
	envString := envToString(e.cmd)
	p, err := parser.Parse(bytes.NewReader([]byte(envString)))
	if err != nil {
		return err
	}
	shlex := shell.NewLex(p.EscapeToken)
	newEnvs := e.cmd.Env
	for index, pair := range newEnvs {
		expandedValue, err := shlex.ProcessWord(pair.Value, config.Env)
		if err != nil {
			return err
		}
		newEnvs[index] = instructions.KeyValuePair{
			Key:   pair.Key,
			Value: expandedValue,
		}
		logrus.Infof("Setting environment variable %s=%s", pair.Key, expandedValue)
		if err := os.Setenv(pair.Key, expandedValue); err != nil {
			return err
		}
	}
	return updateConfigEnv(newEnvs, config)
}

func updateConfigEnv(newEnvs []instructions.KeyValuePair, config *manifest.Schema2Config) error {
	// First, convert config.Env array to []instruction.KeyValuePair
	var kvps []instructions.KeyValuePair
	for _, env := range config.Env {
		entry := strings.Split(env, "=")
		kvps = append(kvps, instructions.KeyValuePair{
			Key:   entry[0],
			Value: entry[1],
		})
	}
	// Iterate through new environment variables, and replace existing keys
	// We can't use a map because we need to preserve the order of the environment variables
Loop:
	for _, newEnv := range newEnvs {
		for index, kvp := range kvps {
			// If key exists, replace the KeyValuePair...
			if kvp.Key == newEnv.Key {
				logrus.Debugf("Replacing environment variable %v with %v in config", kvp, newEnv)
				kvps[index] = newEnv
				continue Loop
			}
		}
		// ... Else, append it as a new env variable
		kvps = append(kvps, newEnv)
	}
	// Convert back to array and set in config
	envArray := []string{}
	for _, kvp := range kvps {
		entry := kvp.Key + "=" + kvp.Value
		envArray = append(envArray, entry)
	}
	config.Env = envArray
	return nil
}

func envToString(cmd *instructions.EnvCommand) string {
	env := []string{"ENV"}
	for _, kvp := range cmd.Env {
		env = append(env, kvp.Key+"="+kvp.Value)
	}
	return strings.Join(env, " ")
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
