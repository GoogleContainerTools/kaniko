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

package dockerfile

import (
	"fmt"
	"io/ioutil"
	"os"
	"os/exec"

	"github.com/pkg/errors"
	"gopkg.in/yaml.v2"
)

const (
	hadolintConfigPath = "/kaniko/.hadolint.yaml"
)

type LintConfig struct {
	IgnoredRules      []string `yaml:"ignored"`
	TrustedRegistries []string `yaml:"trustedRegistries"`
}

func lintDockerfile(dockerfilePath string, ignoredRules, trustedRegistries []string) ([]byte, error) {
	var cfg LintConfig

	args := []string{dockerfilePath}

	if _, hadolintErr := os.Stat(hadolintConfigPath); hadolintErr == nil {
		d, err := ioutil.ReadFile(hadolintConfigPath)
		if err != nil {
			return nil, errors.Wrap(err, fmt.Sprintf("reading hadolint config at path %s", hadolintConfigPath))
		}
		if err := yaml.Unmarshal(d, &cfg); err != nil {
			return nil, errors.Wrap(err, "unmarshaling hadolint config")
		}
	} else if len(ignoredRules) > 0 || len(trustedRegistries) > 0 {
		f, err := os.Create(hadolintConfigPath)
		defer f.Close()
		if err != nil {
			return nil, errors.Wrap(err, fmt.Sprintf("creating hadolint config at path %s", hadolintConfigPath))
		}
	} else {
		return exec.Command("hadolint", args...).CombinedOutput()
	}

	// It's fine if Ignores Rules or Trusted Registries have dupplicates
	cfg.IgnoredRules = append(cfg.IgnoredRules, ignoredRules...)
	cfg.TrustedRegistries = append(cfg.TrustedRegistries, trustedRegistries...)

	data, err := yaml.Marshal(&cfg)
	if err != nil {
		return nil, errors.Wrap(err, fmt.Sprintf("marshaling hadolint config %v", cfg))
	}

	if err := ioutil.WriteFile(hadolintConfigPath, data, 0644); err != nil {
		return nil, errors.Wrap(err, fmt.Sprintf("writing hadolint config data into file at path %s", hadolintConfigPath))
	}

	args = append(args, "-c", hadolintConfigPath)

	return exec.Command("hadolint", args...).CombinedOutput()
}
