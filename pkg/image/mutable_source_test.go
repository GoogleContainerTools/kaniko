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
	"encoding/json"
	"io/ioutil"
	"testing"

	"github.com/containers/image/manifest"
	"github.com/containers/image/types"
	digest "github.com/opencontainers/go-digest"
)

type fields struct {
	mfst *manifest.Schema2
	cfg  *manifest.Schema2Image
}
type args struct {
	content string
}

var testCases = []struct {
	name    string
	fields  fields
	args    args
	wantErr bool
}{
	{
		name: "add layer",
		fields: fields{
			mfst: &manifest.Schema2{
				LayersDescriptors: []manifest.Schema2Descriptor{
					{
						Digest: digest.Digest("abc123"),
					},
				},
			},
			cfg: &manifest.Schema2Image{
				RootFS: &manifest.Schema2RootFS{
					DiffIDs: []digest.Digest{digest.Digest("bcd234")},
				},
				History: []manifest.Schema2History{
					{
						CreatedBy: "foo",
					},
				},
			},
		},
		args: args{
			content: "myextralayer",
		},
		wantErr: false,
	},
}

func TestMutableSource_appendLayer(t *testing.T) {
	for _, tt := range testCases {
		t.Run(tt.name, func(t *testing.T) {
			m := &MutableSource{
				mfst:       tt.fields.mfst,
				cfg:        tt.fields.cfg,
				extraBlobs: make(map[string][]byte),
			}

			if err := m.AppendLayer([]byte(tt.args.content)); (err != nil) != tt.wantErr {
				t.Fatalf("MutableSource.appendLayer() error = %v, wantErr %v", err, tt.wantErr)
			}
			if err := m.SaveConfig(); err != nil {
				t.Fatalf("Error saving config: %v", err)
			}
			// One blob for the new layer, one for the new config.
			if len(m.extraBlobs) != 2 {
				t.Fatal("No extra blob stored after appending layer.")
			}

			r, _, err := m.GetBlob(types.BlobInfo{Digest: m.mfst.ConfigDescriptor.Digest})
			if err != nil {
				t.Fatal("Not able to get new config blob.")
			}

			cfgBytes, err := ioutil.ReadAll(r)
			if err != nil {
				t.Fatal("Unable to read config.")
			}
			cfg := manifest.Schema2Image{}
			if err := json.Unmarshal(cfgBytes, &cfg); err != nil {
				t.Fatal("Unable to parse config.")
			}

			if len(cfg.History) != 2 {
				t.Fatalf("No layer added to image history: %v", cfg.History)
			}

			if len(cfg.RootFS.DiffIDs) != 2 {
				t.Fatalf("No layer added to Diff IDs: %v", cfg.RootFS.DiffIDs)
			}
			if cfg.RootFS.DiffIDs[1] != digest.FromString(tt.args.content) {
				t.Fatalf("Incorrect diffid for content. Expected %s, got %s", digest.FromString(tt.args.content), cfg.RootFS.DiffIDs[1])
			}
		})
	}
}
