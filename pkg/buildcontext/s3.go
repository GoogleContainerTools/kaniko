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
	"io/ioutil"
	"net/url"
	"path/filepath"
	"strings"

	"github.com/GoogleContainerTools/kaniko/pkg/constants"
	"github.com/GoogleContainerTools/kaniko/pkg/util"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/s3"
)

// S3 unifies calls to download and unpack the build context.
type S3 struct {
}

// UnpackTarFromBuildContext download and untar a file from s3
func (s *S3) UnpackTarFromBuildContext(buildContext string, directory string) error {
	// if no context is set, add default file context.tar.gz
	if !strings.HasSuffix(buildContext, ".tar.gz") {
		buildContext += "/" + constants.ContextTar
	}

	u, err := url.Parse(buildContext)
	if err != nil {
		return err
	}

	bucket := strings.TrimSuffix(u.Host, "/")
	key := strings.TrimSuffix(u.Path, "/")
	tarPath := filepath.Join(directory, constants.ContextTar)

	svc := s3.New(session.New())
	input := &s3.GetObjectInput{
		Bucket: aws.String(bucket),
		Key:    aws.String(key),
	}

	response, err := svc.GetObject(input)
	if err != nil {
		return err
	}
	defer response.Body.Close()
	body, err := ioutil.ReadAll(response.Body)
	if err != nil {
		return err
	}
	ioutil.WriteFile(tarPath, body, 0600)

	if err := util.UnpackCompressedTar(tarPath, directory); err != nil {
		return err
	}

	return nil
}
