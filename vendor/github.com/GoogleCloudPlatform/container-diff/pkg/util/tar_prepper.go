/*
Copyright 2017 Google, Inc. All rights reserved.

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
	"encoding/json"
	"errors"
	"github.com/containers/image/docker/tarfile"
	"github.com/docker/docker/client"
	"github.com/sirupsen/logrus"
	"io/ioutil"
	"os"
	"path/filepath"
)

type TarPrepper struct {
	Source string
	Client *client.Client
}

func (p TarPrepper) Name() string {
	return "Tar Archive"
}

func (p TarPrepper) GetSource() string {
	return p.Source
}

func (p TarPrepper) GetImage() (Image, error) {
	image, err := getImage(p)
	image.Type = ImageTypeTar
	return image, err
}

func (p TarPrepper) GetFileSystem() (string, error) {
	return getImageFromTar(p.Source)
}

func (p TarPrepper) GetConfig() (ConfigSchema, error) {
	tempDir, err := ioutil.TempDir("", ".container-diff")
	if err != nil {
		return ConfigSchema{}, nil
	}
	defer os.RemoveAll(tempDir)
	f, err := os.Open(p.Source)
	if err != nil {
		return ConfigSchema{}, err
	}
	defer f.Close()
	if err := UnTar(f, tempDir, nil); err != nil {
		return ConfigSchema{}, err
	}

	var config ConfigSchema
	// First open the manifest, then find the referenced config.
	manifestPath := filepath.Join(tempDir, "manifest.json")
	contents, err := ioutil.ReadFile(manifestPath)
	if err != nil {
		return ConfigSchema{}, err
	}

	manifests := []tarfile.ManifestItem{}
	if err := json.Unmarshal(contents, &manifests); err != nil {
		return ConfigSchema{}, err
	}

	if len(manifests) != 1 {
		return ConfigSchema{}, errors.New("specified tar file contains multiple images")
	}

	cfgFilename := filepath.Join(tempDir, manifests[0].Config)
	file, err := ioutil.ReadFile(cfgFilename)
	if err != nil {
		logrus.Errorf("Could not read config file %s: %s", cfgFilename, err)
		return ConfigSchema{}, errors.New("Could not obtain image config")
	}
	err = json.Unmarshal(file, &config)
	if err != nil {
		logrus.Errorf("Could not marshal config file %s: %s", cfgFilename, err)
		return ConfigSchema{}, errors.New("Could not obtain image config")
	}

	return config, nil
}
