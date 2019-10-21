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
	"net/url"
	"os"
	"path/filepath"
	"strings"

	"github.com/Azure/azure-storage-blob-go/azblob"
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

	// Get storage accoutname for Azure Blob Storage
	u, _ := url.Parse(b.context)
	parts := azblob.NewBlobURLParts(*u)
	accountName := strings.Split(parts.Host, ".")[0]

	// Generate credentail with accountname and accountkey
	credential, err := azblob.NewSharedKeyCredential(accountName, accountKey)
	if err != nil {
		return parts.Host, err
	}

	//Create directory and file for downloading context file
	directory := constants.BuildContextDir
	tarPath := filepath.Join(directory, constants.ContextTar)
	if err := os.MkdirAll(directory, 0750); err != nil {
		return directory, err
	}
	file, err := os.Create(tarPath)
	if err != nil {
		return directory, err
	}

	//Downloading contextfile from Azure Blob Storage
	p := azblob.NewPipeline(credential, azblob.PipelineOptions{})
	blobURL := azblob.NewBlobURL(*u, p)
	ctx := context.Background()

	if err := azblob.DownloadBlobToFile(ctx, blobURL, 0, 0, file, azblob.DownloadFromBlobOptions{}); err != nil {
		return parts.Host, err
	}

	if err := util.UnpackCompressedTar(tarPath, directory); err != nil {
		return tarPath, err
	}
	// Remove the tar so it doesn't interfere with subsequent commands
	return directory, os.Remove(tarPath)
}
