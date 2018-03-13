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

package buildcontext

import (
	"github.com/GoogleCloudPlatform/k8s-container-builder/pkg/util"
	"io/ioutil"
	"os"
	"path/filepath"
)

type LocalDirectory struct {
	root string
}

func (ld *LocalDirectory) Files(path string) ([]string, error) {
	return util.Files(path, ld.root)
}

func (ld *LocalDirectory) Exists(path string) bool {
	fullPath := filepath.Join(ld.root, path)
	return util.FilepathExists(fullPath)
}

func (ld *LocalDirectory) Stat(path string) (os.FileInfo, error) {
	fullPath := filepath.Join(ld.root, path)
	return os.Stat(fullPath)
}

func (ld *LocalDirectory) Contents(path string) ([]byte, error) {
	fullPath := filepath.Join(ld.root, path)
	return ioutil.ReadFile(fullPath)
}
