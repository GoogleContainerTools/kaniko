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
	"regexp"

	"github.com/GoogleContainerTools/kaniko/pkg/constants"
)

//Validate if the host url provided is with correct suffix for AzureCloud, AzureChinaCloud, AzureGermanCloud and AzureUSGovernment
//RegEX for supported suffix defined in constants.AzureBlobStorageHostRegEx
func ValidAzureBlobStorageHost(context string) bool {
	for _, re := range constants.AzureBlobStorageHostRegEx {
		validBlobURL := regexp.MustCompile(re)
		if validBlobURL.MatchString(context) {
			return true
		}
	}

	return false
}
