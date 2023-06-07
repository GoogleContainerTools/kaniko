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
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	kConfig "github.com/GoogleContainerTools/kaniko/pkg/config"
	"github.com/GoogleContainerTools/kaniko/pkg/constants"
	"github.com/GoogleContainerTools/kaniko/pkg/util"
	"github.com/GoogleContainerTools/kaniko/pkg/util/bucket"
	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	s3manager "github.com/aws/aws-sdk-go-v2/feature/s3/manager"
	"github.com/aws/aws-sdk-go-v2/service/s3"
)

// S3 unifies calls to download and unpack the build context.
type S3 struct {
	context string
}

// UnpackTarFromBuildContext download and untar a file from s3
func (s *S3) UnpackTarFromBuildContext() (string, error) {
	bucket, item, err := bucket.GetNameAndFilepathFromURI(s.context)
	if err != nil {
		return "", fmt.Errorf("getting bucketname and filepath from context: %w", err)
	}

	endpoint := os.Getenv(constants.S3EndpointEnv)
	forcePath := false
	if strings.ToLower(os.Getenv(constants.S3ForcePathStyle)) == "true" {
		forcePath = true
	}

	customResolver := aws.EndpointResolverWithOptionsFunc(func(service, region string, options ...interface{}) (aws.Endpoint, error) {
		if endpoint != "" {
			return aws.Endpoint{
				URL: endpoint,
			}, nil
		}
		return aws.Endpoint{}, &aws.EndpointNotFoundError{}
	})

	cfg, err := config.LoadDefaultConfig(context.TODO(), config.WithEndpointResolverWithOptions(customResolver))
	if err != nil {
		return bucket, err
	}
	client := s3.NewFromConfig(cfg, func(options *s3.Options) {
		if endpoint != "" {
			options.UsePathStyle = forcePath
		}
	})
	downloader := s3manager.NewDownloader(client)
	directory := kConfig.BuildContextDir
	tarPath := filepath.Join(directory, constants.ContextTar)
	if err := os.MkdirAll(directory, 0750); err != nil {
		return directory, err
	}
	file, err := os.Create(tarPath)
	if err != nil {
		return directory, err
	}
	_, err = downloader.Download(context.TODO(), file,
		&s3.GetObjectInput{
			Bucket: aws.String(bucket),
			Key:    aws.String(item),
		})
	if err != nil {
		return directory, err
	}

	return directory, util.UnpackCompressedTar(tarPath, directory)
}
