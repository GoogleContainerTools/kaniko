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

package contexts

import (
	"github.com/GoogleCloudPlatform/k8s-container-builder/pkg/util"
	"github.com/sirupsen/logrus"
)

type Context interface {
	// Returns a map of [file path]:[file contents] of all files rooted at path
	GetFilesFromPath(path string) (map[string][]byte, error)
}

func GetContext(source string) Context {
	if util.FilepathExists(source) {
		logrus.Infof("Using local directory context at path %s", source)
		return &LocalDirectory{root: source}
	}
	logrus.Errorf("Unable to find valid context for %s", source)
	return nil
}
