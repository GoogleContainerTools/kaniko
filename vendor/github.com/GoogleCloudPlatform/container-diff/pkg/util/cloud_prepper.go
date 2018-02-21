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
	"io/ioutil"
	"strings"

	"github.com/containers/image/docker"
	"github.com/containers/image/types"
	"github.com/docker/docker/client"
)

// CloudPrepper prepares images sourced from a Cloud registry
type CloudPrepper struct {
	Source      string
	Client      *client.Client
	ImageSource types.ImageSource
}

func (p CloudPrepper) Name() string {
	return "Cloud Registry"
}

func (p CloudPrepper) GetSource() string {
	return p.Source
}

func (p CloudPrepper) GetImage() (Image, error) {
	image, err := getImage(p)
	image.Type = ImageTypeCloud
	return image, err
}

func (p CloudPrepper) GetFileSystem() (string, error) {
	ref, err := docker.ParseReference("//" + p.Source)
	if err != nil {
		return "", err
	}
	sanitizedName := strings.Replace(p.Source, ":", "", -1)
	sanitizedName = strings.Replace(sanitizedName, "/", "", -1)

	path, err := ioutil.TempDir("", sanitizedName)
	if err != nil {
		return "", err
	}

	return path, GetFileSystemFromReference(ref, p.ImageSource, path, nil)
}

func (p CloudPrepper) GetConfig() (ConfigSchema, error) {
	ref, err := docker.ParseReference("//" + p.Source)
	if err != nil {
		return ConfigSchema{}, err
	}

	return getConfigFromReference(ref, p.Source)
}
