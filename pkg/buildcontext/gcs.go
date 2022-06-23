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
	"fmt"
	"io"
	"os"
	"path/filepath"

	kConfig "github.com/GoogleContainerTools/kaniko/pkg/config"
	"github.com/GoogleContainerTools/kaniko/pkg/constants"
	"github.com/GoogleContainerTools/kaniko/pkg/util"
	"github.com/GoogleContainerTools/kaniko/pkg/util/bucket"
	"github.com/sirupsen/logrus"
	"golang.org/x/net/context"
)

// GCS struct for Google Cloud Storage processing
type GCS struct {
	context string
}

func (g *GCS) UnpackTarFromBuildContext() (string, error) {
	bucketName, filepath, err := bucket.GetNameAndFilepathFromURI(g.context)
	if err != nil {
		return "", fmt.Errorf("getting bucketname and filepath from context: %w", err)
	}
	return kConfig.BuildContextDir, unpackTarFromGCSBucket(bucketName, filepath, kConfig.BuildContextDir)
}

func UploadToBucket(r io.Reader, dest string) error {
	ctx := context.Background()
	bucketName, filepath, err := bucket.GetNameAndFilepathFromURI(dest)
	if err != nil {
		return fmt.Errorf("getting bucketname and filepath from dest: %w", err)
	}
	client, err := bucket.NewClient(ctx)
	if err != nil {
		return err
	}
	return bucket.Upload(ctx, bucketName, filepath, r, client)
}

// unpackTarFromGCSBucket unpacks the context.tar.gz file in the given bucket to the given directory
func unpackTarFromGCSBucket(bucketName, item, directory string) error {
	// Get the tar from the bucket
	tarPath, err := getTarFromBucket(bucketName, item, directory)
	if err != nil {
		return err
	}
	logrus.Debug("Unpacking source context tar...")
	if err := util.UnpackCompressedTar(tarPath, directory); err != nil {
		return err
	}
	// Remove the tar so it doesn't interfere with subsequent commands
	logrus.Debugf("Deleting %s", tarPath)
	return os.Remove(tarPath)
}

// getTarFromBucket gets context.tar.gz from the GCS bucket and saves it to the filesystem
// It returns the path to the tar file
func getTarFromBucket(bucketName, filepathInBucket, directory string) (string, error) {
	ctx := context.Background()
	client, err := bucket.NewClient(ctx)
	if err != nil {
		return "", err
	}
	// Get the tarfile context.tar.gz from the GCS bucket, and save it to a tar object
	reader, err := bucket.ReadCloser(ctx, bucketName, filepathInBucket, client)
	if err != nil {
		return "", err
	}
	defer reader.Close()
	tarPath := filepath.Join(directory, constants.ContextTar)
	if err := util.CreateFile(tarPath, reader, 0600, 0, 0); err != nil {
		return "", err
	}
	logrus.Debugf("Copied tarball %s from GCS bucket %s to %s", constants.ContextTar, bucketName, tarPath)
	return tarPath, nil
}
