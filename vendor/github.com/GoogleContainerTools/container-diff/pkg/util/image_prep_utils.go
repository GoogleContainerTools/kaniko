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

package util

import (
	"archive/tar"
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"os"
	"sort"
	"strings"

	"github.com/GoogleContainerTools/container-diff/cmd/util/output"
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
	SetSource(string)
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
	Hostname     string
	Domainname   string
	User         string
	AttachStdin  bool
	AttachStdout bool
	AttachStderr bool
	ExposedPorts map[string]struct{} `json:"ExposedPorts"`
	Tty          bool
	OpenStdin    bool
	StdinOnce    bool
	Env          []string `json:"Env"`
	Cmd          []string `json:"Cmd"`
	// Healthcheck *HealthConfig
	ArgsEscaped     bool `json:",omitempty"`
	Image           string
	Volumes         map[string]struct{} `json:"Volumes"`
	Workdir         string              `json:"WorkingDir"`
	Entrypoint      []string            `json:"Entrypoint"`
	NetworkDisabled bool                `json:",omitempty"`
	MacAddress      string              `json:",omitempty"`
	OnBuild         []string
	Labels          map[string]string `json:"Labels"`
	StopSignal      string            `json:",omitempty"`
	StopTimeout     *int              `json:",omitempty"`
	Shell           []string          `json:",omitempty"`
}

func (c ConfigObject) AsList() []string {
	return []string{
		fmt.Sprintf("Hostname: %s", c.Hostname),
		fmt.Sprintf("Domainname: %s", c.Domainname),
		fmt.Sprintf("User: %s", c.User),
		fmt.Sprintf("AttachStdin: %t", c.AttachStdin),
		fmt.Sprintf("AttachStdout: %t", c.AttachStdout),
		fmt.Sprintf("AttachStderr: %t", c.AttachStderr),
		fmt.Sprintf("ExposedPorts: %v", sortMap(c.ExposedPorts)),
		fmt.Sprintf("Tty: %t", c.Tty),
		fmt.Sprintf("OpenStdin: %t", c.OpenStdin),
		fmt.Sprintf("StdinOnce: %t", c.StdinOnce),
		fmt.Sprintf("Env: %s", strings.Join(c.Env, ",")),
		fmt.Sprintf("Cmd: %s", strings.Join(c.Cmd, ",")),
		fmt.Sprintf("ArgsEscaped: %t", c.ArgsEscaped),
		fmt.Sprintf("Image: %s", c.Image),
		fmt.Sprintf("Volumes: %v", sortMap(c.Volumes)),
		fmt.Sprintf("Workdir: %s", c.Workdir),
		fmt.Sprintf("Entrypoint: %s", strings.Join(c.Entrypoint, ",")),
		fmt.Sprintf("NetworkDisabled: %t", c.NetworkDisabled),
		fmt.Sprintf("MacAddress: %s", c.MacAddress),
		fmt.Sprintf("OnBuild: %s", strings.Join(c.OnBuild, ",")),
		fmt.Sprintf("Labels: %v", c.Labels),
		fmt.Sprintf("StopSignal: %s", c.StopSignal),
		fmt.Sprintf("StopTimeout: %d", c.StopTimeout),
		fmt.Sprintf("Shell: %s", strings.Join(c.Shell, ",")),
	}
}

type ConfigSchema struct {
	Config  ConfigObject       `json:"config"`
	History []ImageHistoryItem `json:"history"`
}

func getImage(p Prepper) (Image, error) {
	// see if the image name has tag provided, if not add latest as tag
	if !IsTar(p.GetSource()) && !HasTag(p.GetSource()) {
		p.SetSource(p.GetSource() + LatestTag)
	}
	output.PrintToStdErr("Retrieving image %s from source %s\n", p.GetSource(), p.Name())
	imgPath, err := p.GetFileSystem()
	if err != nil {
		// return image with FSPath so it can be cleaned up
		return Image{
			FSPath: imgPath,
		}, err
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

func GetFileSystemFromReference(ref types.ImageReference, imgSrc types.ImageSource, path string, whitelist []string) error {
	var err error
	if imgSrc == nil {
		imgSrc, err = ref.NewImageSource(nil)
	}
	if err != nil {
		return err
	}
	defer imgSrc.Close()
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
		if err := unpackTar(tr, path, whitelist); err != nil {
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
			logrus.Warn(err.Error())
		}
	}
}

func sortMap(m map[string]struct{}) string {
	pairs := make([]string, 0)
	for key := range m {
		pairs = append(pairs, fmt.Sprintf("%s:%s", key, m[key]))
	}
	sort.Strings(pairs)
	return strings.Join(pairs, " ")
}
