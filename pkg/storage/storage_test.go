package storage

import (
	"fmt"
	"github.com/GoogleCloudPlatform/k8s-container-builder/pkg/util"
	"github.com/GoogleCloudPlatform/k8s-container-builder/testutil"
	"io/ioutil"
	"os"
	"path/filepath"
	"testing"
)

func TestStorage(t *testing.T) {
	// First, test creating the bucket

	bucket, bucketName, err := CreateStorageBucket()
	if err != nil {
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
	if err := UploadContextToBucket(files, bucket); err != nil {
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
