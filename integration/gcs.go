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

package integration

import (
	"context"
	"fmt"
	"io/ioutil"
	"log"
	"net/url"
	"os"
	"os/exec"
	"time"

	"cloud.google.com/go/storage"
	"google.golang.org/api/option"
)

// CreateIntegrationTarball will take the contents of the integration directory and write
// them to a tarball in a temmporary dir. It will return a path to the tarball.
func CreateIntegrationTarball() (string, error) {
	log.Println("Creating tarball of integration test files to use as build context")
	dir, err := os.Getwd()
	if err != nil {
		return "", fmt.Errorf("Failed find path to integration dir: %w", err)
	}
	tempDir, err := ioutil.TempDir("", "")
	if err != nil {
		return "", fmt.Errorf("Failed to create temporary directory to hold tarball: %w", err)
	}
	contextFile := fmt.Sprintf("%s/context_%d.tar.gz", tempDir, time.Now().UnixNano())
	cmd := exec.Command("tar", "-C", dir, "-zcvf", contextFile, ".")
	_, err = RunCommandWithoutTest(cmd)
	if err != nil {
		return "", fmt.Errorf("Failed to create build context tarball from integration dir: %w", err)
	}
	return contextFile, err
}

// UploadFileToBucket will upload the at filePath to gcsBucket. It will return the path
// of the file in bucket.
func UploadFileToBucket(ctx context.Context, bucket string, filePath string, gcsPath string, client *storage.Client) (string, error) {
	dst := fmt.Sprintf("%s/%s", bucket, gcsPath)
	log.Printf("Uploading file at %s to GCS bucket at %s\n", filePath, dst)

	writer := client.Bucket(bucket).Object(gcsPath).NewWriter(ctx)
	filedata, err := os.ReadFile(filePath)
	if err != nil {
		return "", fmt.Errorf("reading data at %s: %w", filePath, err)
	}
	_, err = writer.Write(filedata)
	if err != nil {
		log.Printf("Error uploading file %s to GCS at %s: %s", filePath, dst, err)
		return "", fmt.Errorf("failed to copy tarball to GCS bucket %s: %w", bucket, err)
	}
	return dst, nil
}

// DeleteFromBucket will remove the content at path. path should be the full path
// to a file in GCS.
func DeleteFromBucket(ctx context.Context, bucket string, path string, client *storage.Client) error {
	err := client.Bucket(bucket).Object(path).Delete(ctx)
	if err != nil {
		return fmt.Errorf("Failed to delete file %s from GCS: %w", path, err)
	}
	return err
}

func newStorageClient(ctx context.Context, opts ...option.ClientOption) (*storage.Client, error) {
	client, err := storage.NewClient(ctx, opts...)
	if err != nil {
		return nil, err
	}
	return client, err
}

func bucketNameFromUri(bucketURI string) (string, error) {
	url, err := url.Parse(bucketURI)
	if err != nil {
		return "", err
	}
	if url.Scheme != "gs" {
		return "", fmt.Errorf("%v is not a valid google cloud storage uri", bucketURI)
	}
	return url.Host, nil
}
