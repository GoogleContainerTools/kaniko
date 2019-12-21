/*
Copyright 2019 Google LLC

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

package cache

import (
	"bytes"
	"log"

	"github.com/GoogleContainerTools/kaniko/pkg/config"
	"github.com/google/go-containerregistry/pkg/v1/remote"
)

func ExampleWarmer_Warm() {
	tarBuf := new(bytes.Buffer)
	manifestBuf := new(bytes.Buffer)
	w := &Warmer{
		Remote:         remote.Image,
		Local:          LocalSource,
		TarWriter:      tarBuf,
		ManifestWriter: manifestBuf,
	}

	options := &config.WarmerOptions{}

	digest, err := w.Warm("ubuntu:latest", options)
	if err != nil {
		if !IsAlreadyCached(err) {
			log.Fatal(err)
		}
	}

	log.Printf("digest %v tar len %d\nmanifest:\n%s\n", digest, tarBuf.Len(), manifestBuf.String())
}
