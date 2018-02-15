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
	"archive/tar"
	"encoding/json"
	"errors"
	"io/ioutil"
	"os"
	"strings"

	"github.com/GoogleCloudPlatform/container-diff/cmd/util/output"
	"github.com/containers/image/docker"
	"github.com/containers/image/manifest"
	"github.com/containers/image/pkg/compression"
	"github.com/containers/image/types"
	"github.com/sirupsen/logrus"
)

type Prepper interface {
	Name() string
	GetConfig() (ConfigSchema, error)
	GetFileSystem() (string, error)
	GetImage() (Image, error)
	GetSource() string
}

type ImageType int

const (
	ImageTypeTar ImageType = iota
	ImageTypeDaemon
	ImageTypeCloud
)

type Image struct {
	Source string
	FSPath string
	Config ConfigSchema
	Type   ImageType
}

func (i *Image) IsTar() bool {
	return i.Type == ImageTypeTar
}

func (i *Image) IsDaemon() bool {
	return i.Type == ImageTypeDaemon
}

func (i *Image) IsCloud() bool {
	return i.Type == ImageTypeCloud
}

func (i *Image) GetRemoteDigest() (string, error) {
	ref, err := docker.ParseReference("//" + i.Source)
	if err != nil {
		return "", err
	}
	return getDigestFromReference(ref, i.Source)
}

func (i *Image) GetName() string {
	return strings.Split(i.Source, ":")[0]
}

type ImageHistoryItem struct {
	CreatedBy string `json:"created_by"`
}

type ConfigObject struct {
	Env          []string            `json:"Env"`
	Entrypoint   []string            `json:"Entrypoint"`
	ExposedPorts map[string]struct{} `json:"ExposedPorts"`
	Cmd          []string            `json:"Cmd"`
	Volumes      map[string]struct{} `json:"Volumes"`
	Workdir      string              `json:"WorkingDir"`
	Labels       map[string]string   `json:"Labels"`
}

type ConfigSchema struct {
	Config  ConfigObject       `json:"config"`
	History []ImageHistoryItem `json:"history"`
}

func getImage(p Prepper) (Image, error) {
	output.PrintToStdErr("Retrieving image %s from source %s\n", p.GetSource(), p.Name())
	imgPath, err := p.GetFileSystem()
	if err != nil {
		return Image{}, err
	}

	config, err := p.GetConfig()
	if err != nil {
		logrus.Error("Error retrieving History: ", err)
	}

	logrus.Infof("Finished prepping image %s", p.GetSource())
	return Image{
		Source: p.GetSource(),
		FSPath: imgPath,
		Config: config,
	}, nil
}

func getImageFromTar(tarPath string) (string, error) {
	logrus.Info("Extracting image tar to obtain image file system")
	tempPath, err := ioutil.TempDir("", ".container-diff")
	if err != nil {
		return "", err
	}
	return tempPath, unpackDockerSave(tarPath, tempPath)
}

func getFileSystemFromReference(ref types.ImageReference, imgSrc types.ImageSource, path string) error {
	img, err := ref.NewImage(nil)
	if err != nil {
		return err
	}
	defer img.Close()
	for _, b := range img.LayerInfos() {
		bi, _, err := imgSrc.GetBlob(b)
		if err != nil {
			return err
		}
		defer bi.Close()
		f, reader, err := compression.DetectCompression(bi)
		if err != nil {
			return err
		}
		// Decompress if necessary.
		if f != nil {
			reader, err = f(reader)
			if err != nil {
				return err
			}
		}
		tr := tar.NewReader(reader)
		if err := unpackTar(tr, path); err != nil {
			return err
		}
	}
	return nil
}

func getDigestFromReference(ref types.ImageReference, source string) (string, error) {
	img, err := ref.NewImage(nil)
	if err != nil {
		logrus.Errorf("Error referencing image %s from registry: %s", source, err)
		return "", errors.New("Could not obtain image digest")
	}
	defer img.Close()

	rawManifest, _, err := img.Manifest()
	if err != nil {
		logrus.Errorf("Error referencing image %s from registry: %s", source, err)
		return "", errors.New("Could not obtain image digest")
	}

	digest, err := manifest.Digest(rawManifest)
	if err != nil {
		logrus.Errorf("Error referencing image %s from registry: %s", source, err)
		return "", errors.New("Could not obtain image digest")
	}

	return digest.String(), nil
}

func getConfigFromReference(ref types.ImageReference, source string) (ConfigSchema, error) {
	img, err := ref.NewImage(nil)
	if err != nil {
		logrus.Errorf("Error referencing image %s from registry: %s", source, err)
		return ConfigSchema{}, errors.New("Could not obtain image config")
	}
	defer img.Close()

	configBlob, err := img.ConfigBlob()
	if err != nil {
		logrus.Errorf("Error obtaining config blob for image %s from registry: %s", source, err)
		return ConfigSchema{}, errors.New("Could not obtain image config")
	}

	var config ConfigSchema
	err = json.Unmarshal(configBlob, &config)
	if err != nil {
		logrus.Errorf("Error with config file struct for image %s: %s", source, err)
		return ConfigSchema{}, errors.New("Could not obtain image config")
	}
	return config, nil
}

func CleanupImage(image Image) {
	if image.FSPath != "" {
		logrus.Infof("Removing image filesystem directory %s from system", image.FSPath)
		if err := os.RemoveAll(image.FSPath); err != nil {
			logrus.Error(err.Error())
		}
	}
}
