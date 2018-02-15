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
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"strings"

	"github.com/docker/docker/client"
)

type Event struct {
	Status         string `json:"status"`
	Error          string `json:"error"`
	Progress       string `json:"progress"`
	ProgressDetail struct {
		Current int `json:"current"`
		Total   int `json:"total"`
	} `json:"progressDetail"`
}

func NewClient() (*client.Client, error) {
	cli, err := client.NewEnvClient()
	if err != nil {
		return nil, fmt.Errorf("Error getting docker client: %s", err)
	}
	cli.NegotiateAPIVersion(context.Background())

	return cli, nil
}

func getLayersFromManifest(r io.Reader) ([]string, error) {
	type Manifest struct {
		Layers []string
	}

	manifestJSON, err := ioutil.ReadAll(r)
	if err != nil {
		return nil, err
	}

	var imageManifest []Manifest
	if err := json.Unmarshal(manifestJSON, &imageManifest); err != nil {
		return []string{}, fmt.Errorf("Could not unmarshal manifest to get layer order: %s", err)
	}
	return imageManifest[0].Layers, nil
}

func unpackDockerSave(tarPath string, target string) error {
	if _, ok := os.Stat(target); ok != nil {
		os.MkdirAll(target, 0775)
	}
	f, err := os.Open(tarPath)
	if err != nil {
		return err
	}

	tr := tar.NewReader(f)

	// Unpack the layers into a map, since we need to sort out the order later.
	var layers []string
	layerMap := map[string][]byte{}
	for {
		hdr, err := tr.Next()
		if err == io.EOF {
			break
		}
		if err != nil {
			return err
		}

		// Docker save contains files and directories. Ignore the directories.
		// We care about the layers and the manifest. The layers look like:
		// $SHA/layer.tar
		// and they are referenced that way in the manifest.
		switch t := hdr.Typeflag; t {
		case tar.TypeReg:
			if hdr.Name == "manifest.json" {
				layers, err = getLayersFromManifest(tr)
				if err != nil {
					return err
				}
			} else if strings.HasSuffix(hdr.Name, ".tar") {
				layerMap[hdr.Name], err = ioutil.ReadAll(tr)
				if err != nil {
					return err
				}
			}
		case tar.TypeDir:
			continue
		default:
			return fmt.Errorf("unsupported file type %v found in file %s tar %s", t, hdr.Name, tarPath)
		}
	}

	for _, layer := range layers {
		if err = UnTar(bytes.NewReader(layerMap[layer]), target); err != nil {
			return fmt.Errorf("Could not unpack layer %s: %s", layer, err)
		}
	}
	return nil
}
