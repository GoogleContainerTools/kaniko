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
	"fmt"
	"github.com/GoogleCloudPlatform/k8s-container-builder/pkg/util"
	"github.com/GoogleCloudPlatform/k8s-container-builder/testutil"
	"github.com/sirupsen/logrus"
	"golang.org/x/net/context"
	"io/ioutil"
	"os"
	"path/filepath"
	"testing"
	"time"

	"cloud.google.com/go/storage"
)

func TestStorage(t *testing.T) {
	// First, create a test bucket
	bucketName := fmt.Sprintf("kbuild-test-buckets-%d", time.Now().Unix())
	if err := createStorageBucket(bucketName); err != nil {
		t.Fatalf("Unable to create GCS storage bucket: %s", err)
	}

	// Second, create temp dir of files and test uploading them to the bucket
	testDir, err := ioutil.TempDir("", "")
	if err != nil {
		t.Fatalf("Error creating tempdir: %s", err)
	}
	t.Log(testDir)
	defer os.RemoveAll(testDir)

	fileMap := map[string]string{
		"foo":     "baz1",
		"bar/bat": "baz2",
	}
	if err := testutil.SetupFiles(testDir, fileMap); err != nil {
		t.Fatalf("Unable to setup files: %s", err)
	}
	files, err := util.Files(testDir)
	fmt.Println(files)
	if err != nil {
		t.Fatalf("Unable to get files at test dir %s", "")
	}
	if err := UploadFilesToBucket(files, bucketName); err != nil {
		t.Fatalf("Unable to upload files to bucket, %s", err)
	}

	// Make sure the files actually exist, so get all files from the bucket
	bucketFilesMap, err := GetFilesFromStorageBucket(bucketName, testDir)
	if err != nil {
		t.Fatalf("Unable to get files from bucket, %s", err)
	}
	for filename, contents := range bucketFilesMap {
		relPath, err := filepath.Rel(testDir, filename)
		if err != nil {
			t.Fatalf("Couldn't get relative path: %s", err)
		}
		if _, exists := fileMap[relPath]; !exists {
			t.Fatalf("Found file %s unexpectedly in bucket", filename)
		}
		if string(contents) != fileMap[relPath] {
			t.Fatalf("File contents wrong, expected: %s actual: %s", fileMap[filename], string(contents))
		}
	}

	//Make sure getting all the files from the bucket is equivalent
	allBucketFilesMap, err := GetFilesFromStorageBucket(bucketName, "")
	if err != nil {
		t.Fatalf("Unable to get files from bucket, %s", err)
	}
	for filename, contents := range allBucketFilesMap {
		if _, exists := bucketFilesMap[filename]; !exists {
			t.Fatalf("Found file %s unexpectedly in bucket", filename)
		}
		if string(contents) != string(bucketFilesMap[filename]) {
			t.Fatalf("File contents wrong, expected: %s actual: %s", fileMap[filename], string(contents))
		}
	}

}

// CreateStorageBucket creates a storage bucket to store the source context in
func createStorageBucket(bucketName string) error {
	ctx := context.Background()

	client, err := storage.NewClient(ctx)
	if err != nil {
		return err
	}
	projectID := "kbuild-test"
	// Creates a Bucket instance.
	bucket := client.Bucket(bucketName)

	// Creates the new bucket.
	if err := bucket.Create(ctx, projectID, nil); err != nil {
		logrus.Errorf("Failed to create bucket: %v", err)
		return err
	}
	logrus.Info("Created bucket ", bucketName)
	return nil
}
