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
	"testing"

	"github.com/GoogleContainerTools/kaniko/pkg/constants"
	"github.com/GoogleContainerTools/kaniko/testutil"
)

func Test_GetBucketAndItem(t *testing.T) {
	tests := []struct {
		name           string
		context        string
		expectedBucket string
		expectedItem   string
		expectedErr    bool
	}{
		{
			name:           "three slashes",
			context:        "gs://test1/test2/test3",
			expectedBucket: "test1",
			expectedItem:   "test2/test3",
		},
		{
			name:           "two slashes",
			context:        "gs://test1/test2",
			expectedBucket: "test1",
			expectedItem:   "test2",
		},
		{
			name:           "one slash",
			context:        "gs://test1/",
			expectedBucket: "test1",
			expectedItem:   constants.ContextTar,
		},
		{
			name:           "zero slash",
			context:        "gs://test1",
			expectedBucket: "test1",
			expectedItem:   constants.ContextTar,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			gotBucket, gotItem, err := GetNameAndFilepathFromURI(test.context)
			testutil.CheckError(t, test.expectedErr, err)
			testutil.CheckDeepEqual(t, test.expectedBucket, gotBucket)
			testutil.CheckDeepEqual(t, test.expectedItem, gotItem)
		})
	}

}
