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
	"errors"
	"strings"
)

var buildContextMap = map[string]BuildContext{
	"gs://":  &GCS{},
	"s3://":  &S3{},
	"dir://": &File{},
}

// BuildContext unifies calls to download and unpack the build context.
type BuildContext interface {
	// Gets context.tar.gz from the build context and unpacks to the directory
	UnpackTarFromBuildContext(buildContext string, directory string) error
}

// GetBuildContext parses srcContext for the prefix and returns related buildcontext
// parser
func GetBuildContext(srcContext string) (BuildContext, error) {
	for prefix, bc := range buildContextMap {
		if strings.HasPrefix(srcContext, prefix) {
			return bc, nil
		}
	}
	return nil, errors.New("unknown prefix")
}
