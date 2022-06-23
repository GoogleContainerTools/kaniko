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

package bucket

import (
	"context"
	"fmt"
	"io"
	"net/url"
	"strings"

	"cloud.google.com/go/storage"
	"github.com/GoogleContainerTools/kaniko/pkg/constants"
	"google.golang.org/api/option"
)

// Upload uploads everything from Reader to the bucket under path
func Upload(ctx context.Context, bucketName string, path string, r io.Reader, client *storage.Client) error {
	bucket := client.Bucket(bucketName)
	w := bucket.Object(path).NewWriter(ctx)
	if _, err := io.Copy(w, r); err != nil {
		return err
	}
	if err := w.Close(); err != nil {
		return err
	}
	return nil
}

// Delete will remove the content at path. path should be the full path
// to a file in GCS.
func Delete(ctx context.Context, bucketName string, path string, client *storage.Client) error {
	err := client.Bucket(bucketName).Object(path).Delete(ctx)
	if err != nil {
		return fmt.Errorf("failed to delete file at %s in gcs bucket %v: %w", path, bucketName, err)
	}
	return err
}

// ReadCloser will create io.ReadCloser for the specified bucket and path
func ReadCloser(ctx context.Context, bucketName string, path string, client *storage.Client) (io.ReadCloser, error) {
	bucket := client.Bucket(bucketName)
	r, err := bucket.Object(path).NewReader(ctx)
	if err != nil {
		return nil, err
	}
	return r, nil
}

// NewClient returns a new google storage client
func NewClient(ctx context.Context, opts ...option.ClientOption) (*storage.Client, error) {
	client, err := storage.NewClient(ctx, opts...)
	if err != nil {
		return nil, err
	}
	return client, err
}

// GetNameAndFilepathFromURI returns the bucketname and the path to the item inside.
// Will error if provided URI is not a valid URL.
// If the filepath is empty, returns the contextTar filename
func GetNameAndFilepathFromURI(bucketURI string) (bucketName string, path string, err error) {
	url, err := url.Parse(bucketURI)
	if err != nil {
		return "", "", err
	}
	bucketName = url.Host
	// remove leading slash
	filePath := strings.TrimPrefix(url.Path, "/")
	if filePath == "" {
		filePath = constants.ContextTar
	}
	return bucketName, filePath, nil
}
