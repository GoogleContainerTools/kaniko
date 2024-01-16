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
	"errors"
	"os"
	"path/filepath"
	"strings"

	"github.com/Azure/azure-sdk-for-go/sdk/storage/azblob"
	kConfig "github.com/GoogleContainerTools/kaniko/pkg/config"
	"github.com/GoogleContainerTools/kaniko/pkg/constants"
	"github.com/GoogleContainerTools/kaniko/pkg/util"
)

// AzureBlob struct for Azure Blob Storage processing
type AzureBlob struct {
	context string
}

// Download context file from given azure blob storage url and unpack it to BuildContextDir
func (b *AzureBlob) UnpackTarFromBuildContext() (string, error) {

	// Get Azure_STORAGE_ACCESS_KEY from environment variables
	accountKey := os.Getenv("AZURE_STORAGE_ACCESS_KEY")
	if len(accountKey) == 0 {
		return "", errors.New("AZURE_STORAGE_ACCESS_KEY environment variable is not set")
	}

	// Get storage accountName for Azure Blob Storage
	parts, err := azblob.ParseURL(b.context)
	if err != nil {
		return parts.Host, err
	}
	accountName := strings.Split(parts.Host, ".")[0]

	// Generate credential with accountName and accountKey
	credential, err := azblob.NewSharedKeyCredential(accountName, accountKey)
	if err != nil {
		return parts.Host, err
	}

	// Create directory and target file for downloading the context file
	directory := kConfig.BuildContextDir
	tarPath := filepath.Join(directory, constants.ContextTar)
	file, err := util.CreateTargetTarfile(tarPath)
	if err != nil {
		return tarPath, err
	}

	// Downloading context file from Azure Blob Storage
	client, err := azblob.NewClientWithSharedKeyCredential(b.context, credential, nil)
	if err != nil {
		return parts.Host, err
	}
	ctx := context.Background()

	if _, err := client.DownloadFile(ctx, parts.ContainerName, parts.BlobName, file, nil); err != nil {
		return parts.Host, err
	}

	if err := util.UnpackCompressedTar(tarPath, directory); err != nil {
		return tarPath, err
	}
	// Remove the tar so it doesn't interfere with subsequent commands
	return directory, os.Remove(tarPath)
}
