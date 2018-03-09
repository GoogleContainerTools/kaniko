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

package image

import (
	"bytes"
	"compress/gzip"
	"encoding/json"
	"io"
	"io/ioutil"
	"strings"
	"time"

	"github.com/containers/image/manifest"
	"github.com/containers/image/types"
	digest "github.com/opencontainers/go-digest"
)

type MutableSource struct {
	ProxySource
	mfst        *manifest.Schema2
	cfg         *manifest.Schema2Image
	extraBlobs  map[string][]byte
	extraLayers []digest.Digest
}

func NewMutableSource(r types.ImageReference) (*MutableSource, error) {
	src, err := r.NewImageSource(nil)
	if err != nil {
		return nil, err
	}
	img, err := r.NewImage(nil)
	if err != nil {
		return nil, err
	}

	ms := &MutableSource{
		ProxySource: ProxySource{
			Ref:         r,
			ImageSource: src,
			img:         img,
		},
		extraBlobs: make(map[string][]byte),
	}
	if err := ms.populateManifestAndConfig(); err != nil {
		return nil, err
	}
	return ms, nil
}

// Manifest marshals the stored manifest to the byte format.
func (m *MutableSource) GetManifest(_ *digest.Digest) ([]byte, string, error) {
	if err := m.saveConfig(); err != nil {
		return nil, "", err
	}
	s, err := json.Marshal(m.mfst)
	if err != nil {
		return nil, "", err
	}
	return s, manifest.DockerV2Schema2MediaType, err
}

// populateManifestAndConfig parses the raw manifest and configs, storing them on the struct.
func (m *MutableSource) populateManifestAndConfig() error {
	mfstBytes, _, err := m.ProxySource.GetManifest(nil)
	if err != nil {
		return err
	}

	m.mfst, err = manifest.Schema2FromManifest(mfstBytes)
	if err != nil {
		return err
	}

	bi := types.BlobInfo{Digest: m.mfst.ConfigDescriptor.Digest}
	r, _, err := m.GetBlob(bi)
	if err != nil {
		return err
	}

	cfgBytes, err := ioutil.ReadAll(r)
	if err != nil {
		return err
	}

	return json.Unmarshal(cfgBytes, &m.cfg)
}

// GetBlob first checks the stored "extra" blobs, then proxies the call to the original source.
func (m *MutableSource) GetBlob(bi types.BlobInfo) (io.ReadCloser, int64, error) {
	if b, ok := m.extraBlobs[bi.Digest.String()]; ok {
		return ioutil.NopCloser(bytes.NewReader(b)), int64(len(b)), nil
	}
	return m.ImageSource.GetBlob(bi)
}

func gzipBytes(b []byte) ([]byte, error) {
	buf := bytes.NewBuffer([]byte{})
	w := gzip.NewWriter(buf)
	_, err := w.Write(b)
	w.Close()
	if err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}

// appendLayer appends an uncompressed blob to the image, preserving the invariants required across the config and manifest.
func (m *MutableSource) AppendLayer(content []byte, author string) error {
	compressedBlob, err := gzipBytes(content)
	if err != nil {
		return err
	}

	dgst := digest.FromBytes(compressedBlob)

	// Add the layer to the manifest.
	descriptor := manifest.Schema2Descriptor{
		MediaType: manifest.DockerV2Schema2LayerMediaType,
		Size:      int64(len(content)),
		Digest:    dgst,
	}
	m.mfst.LayersDescriptors = append(m.mfst.LayersDescriptors, descriptor)

	m.extraBlobs[dgst.String()] = compressedBlob
	m.extraLayers = append(m.extraLayers, dgst)

	// Also add it to the config.
	diffID := digest.FromBytes(content)
	m.cfg.RootFS.DiffIDs = append(m.cfg.RootFS.DiffIDs, diffID)
	m.AppendConfigHistory(author, false)
	return nil
}

// saveConfig marshals the stored image config, and updates the references to it in the manifest.
func (m *MutableSource) saveConfig() error {
	cfgBlob, err := json.Marshal(m.cfg)
	if err != nil {
		return err
	}

	cfgDigest := digest.FromBytes(cfgBlob)
	m.extraBlobs[cfgDigest.String()] = cfgBlob
	m.mfst.ConfigDescriptor = manifest.Schema2Descriptor{
		MediaType: manifest.DockerV2Schema2ConfigMediaType,
		Size:      int64(len(cfgBlob)),
		Digest:    cfgDigest,
	}
	return nil
}

// Env returns a map of environment variables stored in the image config
// Converts each variable from a string of the form KEY=VALUE to a map of KEY:VALUE
func (m *MutableSource) Env() map[string]string {
	envArray := m.cfg.Schema2V1Image.Config.Env
	envMap := make(map[string]string)
	for _, env := range envArray {
		entry := strings.Split(env, "=")
		envMap[entry[0]] = entry[1]
	}
	return envMap
}

// SetEnv takes a map of environment variables, and converts them to an array of strings
// in the form KEY=VALUE, and then sets the image config
func (m *MutableSource) SetEnv(envMap map[string]string, author string) {
	envArray := []string{}
	for key, value := range envMap {
		entry := key + "=" + value
		envArray = append(envArray, entry)
	}
	m.cfg.Schema2V1Image.Config.Env = envArray
	m.AppendConfigHistory(author, true)
}

func (m *MutableSource) Config() *manifest.Schema2Config {
	return m.cfg.Schema2V1Image.Config
}

func (m *MutableSource) SetConfig(config *manifest.Schema2Config, author string, emptyLayer bool) {
	m.cfg.Schema2V1Image.Config = config
	m.AppendConfigHistory(author, emptyLayer)
}

func (m *MutableSource) AppendConfigHistory(author string, emptyLayer bool) {
	history := manifest.Schema2History{
		Created:    time.Now(),
		Author:     author,
		EmptyLayer: emptyLayer,
	}
	m.cfg.History = append(m.cfg.History, history)
}
