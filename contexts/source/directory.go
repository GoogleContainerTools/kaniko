/*
Copyright 2018 Google, Inc. All rights reserved.

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

package source

import (
	"github.com/GoogleCloudPlatform/k8s-container-builder/pkg/storage"
)

type DirectoryContext struct {
}

func (d DirectoryContext) Name() string {
	return "directory"
}

// Copy local directory into a GCS storage bucket
func (d DirectoryContext) CopyFilesToContext(files []string) (string, error) {
	// Create GCS storage bucket
	bucket, bucketName, err := storage.CreateStorageBucket()
	if err != nil {
		return "", err
	}
	if err := storage.UploadContextToBucket(files, bucket); err != nil {
		return "", err
	}
	return bucketName, err
}

func (d DirectoryContext) GetFilesFromSource(path, source string) (map[string][]byte, error) {
	return storage.GetFilesFromStorageBucket(source, path)
}

func (d DirectoryContext) CleanupContext() error {
	return nil
}
