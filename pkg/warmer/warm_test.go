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
package warmer

import (
	"fmt"
	"os"
	"testing"

	"github.com/GoogleContainerTools/kaniko/pkg/config"
	"github.com/GoogleContainerTools/kaniko/testutil"
)

func TestInsecureRegistry(t *testing.T) {
	gopath := os.Getenv("GOPATH")
	warmPath := fmt.Sprintf("%s/tests/dir/warm", gopath)
	insecureConfig := config.WarmerOptions{
		CacheOptions: config.CacheOptions{
			CacheDir: warmPath,
		},
		SecureOptions: config.SecureOptions{
			InsecurePull:      true,
			SkipTLSVerifyPull: true,
		},
		//TODO: test
		//Images: images
	}
	err := WarmCache(&insecureConfig)
	testutil.CheckError(t, false, err)
	os.RemoveAll(warmPath)
}
