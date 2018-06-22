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
	"strings"

	"github.com/GoogleContainerTools/kaniko/pkg/constants"
	"github.com/GoogleContainerTools/kaniko/pkg/util"
)

// GCS struct for Google Cloud Storage processing
type GCS struct {
	context string
}

func (g *GCS) UnpackTarFromBuildContext(directory string) error {
	// if no context is set, add default file context.tar.gz
	if !strings.HasSuffix(g.context, ".tar.gz") {
		g.context += "/" + constants.ContextTar
	}

	if err := util.UnpackTarFromGCSBucket(g.context, directory); err != nil {
		return err
	}

	return nil
}

func (g *GCS) SetContext(srcContext string) {
	g.context = srcContext
}
