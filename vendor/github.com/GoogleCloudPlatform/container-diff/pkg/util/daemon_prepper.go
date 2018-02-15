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
	"context"
	"io/ioutil"
	"strings"

	"github.com/containers/image/docker/daemon"

	"github.com/docker/docker/client"
	"github.com/sirupsen/logrus"
)

type DaemonPrepper struct {
	Source string
	Client *client.Client
}

func (p DaemonPrepper) Name() string {
	return "Local Daemon"
}

func (p DaemonPrepper) GetSource() string {
	return p.Source
}

func (p DaemonPrepper) GetImage() (Image, error) {
	image, err := getImage(p)
	image.Type = ImageTypeDaemon
	return image, err
}

func (p DaemonPrepper) GetFileSystem() (string, error) {
	ref, err := daemon.ParseReference(p.Source)
	if err != nil {
		return "", err
	}

	src, err := ref.NewImageSource(nil)
	if err != nil {
		return "", err
	}

	sanitizedName := strings.Replace(p.Source, ":", "", -1)
	sanitizedName = strings.Replace(sanitizedName, "/", "", -1)

	path, err := ioutil.TempDir("", sanitizedName)
	if err != nil {
		return "", err
	}
	return path, GetFileSystemFromReference(ref, src, path, nil)
}

func (p DaemonPrepper) GetConfig() (ConfigSchema, error) {
	ref, err := daemon.ParseReference(p.Source)
	if err != nil {
		return ConfigSchema{}, err
	}
	return getConfigFromReference(ref, p.Source)
}

func (p DaemonPrepper) GetHistory() []ImageHistoryItem {
	history, err := p.Client.ImageHistory(context.Background(), p.Source)
	if err != nil {
		logrus.Errorf("Could not obtain image history for %s: %s", p.Source, err)
	}
	historyItems := []ImageHistoryItem{}
	for _, item := range history {
		historyItems = append(historyItems, ImageHistoryItem{CreatedBy: item.CreatedBy})
	}
	return historyItems
}
