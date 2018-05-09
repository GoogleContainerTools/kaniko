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
	"github.com/docker/docker/builder/dockerfile"
	"strings"

	"github.com/GoogleContainerTools/kaniko/pkg/util"
	"github.com/docker/docker/builder/dockerfile/instructions"
	"github.com/google/go-containerregistry/v1"
	"github.com/sirupsen/logrus"
)

type LabelCommand struct {
	cmd *instructions.LabelCommand
}

func (r *LabelCommand) ExecuteCommand(config *v1.Config, buildArgs *dockerfile.BuildArgs) error {
	logrus.Info("cmd: LABEL")
	return updateLabels(r.cmd.Labels, config, buildArgs)
}

func updateLabels(labels []instructions.KeyValuePair, config *v1.Config, buildArgs *dockerfile.BuildArgs) error {
	existingLabels := config.Labels
	if existingLabels == nil {
		existingLabels = make(map[string]string)
	}
	// Let's unescape values before setting the label
	replacementEnvs := util.ReplacementEnvs(config, buildArgs)
	for index, kvp := range labels {
		key, err := util.ResolveEnvironmentReplacement(kvp.Key, replacementEnvs, false)
		if err != nil {
			return err
		}
		unescaped, err := util.ResolveEnvironmentReplacement(kvp.Value, replacementEnvs, false)
		if err != nil {
			return err
		}
		labels[index] = instructions.KeyValuePair{
			Key:   key,
			Value: unescaped,
		}
	}
	for _, kvp := range labels {
		logrus.Infof("Applying label %s=%s", kvp.Key, kvp.Value)
		existingLabels[kvp.Key] = kvp.Value
	}

	config.Labels = existingLabels
	return nil

}

// No files have changed, this command only touches metadata.
func (r *LabelCommand) FilesToSnapshot() []string {
	return []string{}
}

// CreatedBy returns some information about the command for the image config history
func (r *LabelCommand) CreatedBy() string {
	l := []string{r.cmd.Name()}
	for _, kvp := range r.cmd.Labels {
		l = append(l, kvp.String())
	}
	return strings.Join(l, " ")
}
