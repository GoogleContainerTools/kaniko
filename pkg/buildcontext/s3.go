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
	"github.com/GoogleContainerTools/kaniko/pkg/constants"
	"github.com/GoogleContainerTools/kaniko/pkg/util"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/s3"
	"github.com/aws/aws-sdk-go/service/s3/s3manager"
	// "github.com/sirupsen/logrus"
	// "io/ioutil"
	// "net/url"
	"os"
	"path/filepath"
	// "strings"
)

// S3 unifies calls to download and unpack the build context.
type S3 struct {
	context string
}

// UnpackTarFromBuildContext download and untar a file from s3
func (s *S3) UnpackTarFromBuildContext(directory string) error {
	bucket := "kaniko"
	item := "context.tar.gz"
	sess, _ := session.NewSession(&aws.Config{
		Region: aws.String("us-east")},
	)

	downloader := s3manager.NewDownloader(sess)
	tarPath := filepath.Join(directory, constants.ContextTar)
	if err := os.MkdirAll(directory, 0750); err != nil {
		return err
	}
	file, err := os.Create(tarPath)
	if err != nil {
		return err
	}
	_, err = downloader.Download(file,
		&s3.GetObjectInput{
			Bucket: aws.String(bucket),
			Key:    aws.String(item),
		})
	if err != nil {
		return err
	}

	if err := util.UnpackCompressedTar(tarPath, directory); err != nil {
		return err
	}

	return nil
}

func (s *S3) SetContext(srcContext string) {
	s.context = srcContext
}
