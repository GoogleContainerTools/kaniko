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

package image

import (
	"os"
	"strings"

	v1 "github.com/google/go-containerregistry/pkg/v1"
	"github.com/sirupsen/logrus"
)

// SetEnvVariables sets environment variables as specified in the image
func SetEnvVariables(img v1.Image) error {
	cfg, err := img.ConfigFile()
	if err != nil {
		return err
	}
	envVars := cfg.Config.Env
	for _, envVar := range envVars {
		split := strings.SplitN(envVar, "=", 2)
		if err := os.Setenv(split[0], split[1]); err != nil {
			return err
		}
		logrus.Infof("Setting environment variable %s", envVar)
	}
	return nil
}
