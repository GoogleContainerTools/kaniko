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

package storage

import (
	"bytes"
	"github.com/GoogleCloudPlatform/k8s-container-builder/pkg/util"
	"github.com/sirupsen/logrus"
	"golang.org/x/net/context"
	"google.golang.org/api/iterator"
	"io"
	"io/ioutil"
	"os"

	"cloud.google.com/go/storage"
)

// UploadFilesToBucket uploads files (given as a list of filepaths) to the given bucket
func UploadFilesToBucket(files []string, bucketName string) error {
	ctx := context.Background()
	client, err := storage.NewClient(ctx)
	if err != nil {
		return err
	}
	// Creates a Bucket instance.
	bucket := client.Bucket(bucketName)
	for _, file := range files {
		if dir, err := util.IsDir(file); dir || err != nil {
			logrus.Debugf("%s is directory, continue to upload context", file)
			if err != nil {
				return err
			}
			continue
		}
		f, err := os.Open(file)
		if err != nil {
			logrus.Debugf("Could not open %s, err: %v", file, err)
			return err
		}
		defer f.Close()
		buf := bytes.NewBuffer(nil)
		_, err = io.Copy(buf, f)
		if err != nil {
			logrus.Debugf("Could not copy contents of %s, err: %v", file, err)
			return err
		}
		w := bucket.Object(file).NewWriter(ctx)
		if _, err := w.Write(buf.Bytes()); err != nil {
			logrus.Errorf("createFile: unable to write file %s: %v", file, err)
			return err
		}
		if err := w.Close(); err != nil {
			logrus.Errorf("createFile: unable to close bucket: %v", err)
			return err
		}
	}
	return nil
}

// GetFilesFromStorageBucket returns a map [filename]:[file contents] of all
// files in the specified bucket at the given path
func GetFilesFromStorageBucket(bucketName string, path string) (map[string][]byte, error) {
	ctx := context.Background()
	client, err := storage.NewClient(ctx)
	if err != nil {
		return nil, err
	}
	bucket := client.Bucket(bucketName)
	// First, get all the files located at that path in the bucket
	files, err := listFilesInBucket(bucket, bucketName, path)
	if err != nil {
		return nil, err
	}
	// Next, get the contents for each file and add to fileMap
	fileMap := make(map[string][]byte)
	for _, file := range files {
		reader, err := bucket.Object(file).NewReader(ctx)
		if err != nil {
			return nil, err
		}
		defer reader.Close()
		contents, err := ioutil.ReadAll(reader)
		if err != nil {
			return nil, err
		}
		fileMap[file] = contents
	}
	return fileMap, err
}

func listFilesInBucket(bucket *storage.BucketHandle, bucketName, path string) ([]string, error) {
	ctx := context.Background()
	query := &storage.Query{Prefix: path}
	if path == "" {
		query = nil
	}
	logrus.Infof("Querying %s for %s", bucketName, path)
	it := bucket.Objects(ctx, query)
	var files []string
	for {
		obj, err := it.Next()
		if err == iterator.Done {
			break
		}
		if err != nil {
			logrus.Errorf("listBucket: unable to list files in bucket at %s, err: %v", path, err)
			return nil, err
		}
		files = append(files, obj.Name)
	}
	return files, nil
}
