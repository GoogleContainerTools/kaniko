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

package util

import (
	"testing"

	"github.com/GoogleContainerTools/kaniko/testutil"
)

func Test_ValidAzureBlobStorageHost(t *testing.T) {
	tests := []struct {
		name           string
		context        string
		expectedResult bool
	}{
		{
			name:           "AzureCloud",
			context:        "https://myaccount.blob.core.windows.net/fairingcontext/context.tar.gz",
			expectedResult: true,
		},
		{
			name:           "AzureChinaCloud",
			context:        "https://myaccount.blob.core.chinacloudapi.cn/fairingcontext/context.tar.gz",
			expectedResult: true,
		},
		{
			name:           "AzureGermanCloud",
			context:        "https://myaccount.blob.core.cloudapi.de/fairingcontext/context.tar.gz",
			expectedResult: true,
		},
		{
			name:           "AzureUSGovernment",
			context:        "https://myaccount.blob.core.usgovcloudapi.net/fairingcontext/context.tar.gz",
			expectedResult: true,
		},
		{
			name:           "Invalid Azure Blob Storage Hostname",
			context:        "https://myaccount.anything.core.windows.net/fairingcontext/context.tar.gz",
			expectedResult: false,
		},
		{
			name:           "URL Missing Accountname",
			context:        "https://blob.core.windows.net/fairingcontext/context.tar.gz",
			expectedResult: false,
		},
		{
			name:           "URL Missing Containername",
			context:        "https://myaccount.blob.core.windows.net/",
			expectedResult: false,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			result := ValidAzureBlobStorageHost(test.context)
			testutil.CheckDeepEqual(t, test.expectedResult, result)

		})
	}
}
