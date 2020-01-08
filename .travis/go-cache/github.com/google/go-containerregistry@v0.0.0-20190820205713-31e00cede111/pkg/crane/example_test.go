// Copyright 2018 Google LLC All Rights Reserved.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//    http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package crane_test

import (
	"fmt"

	"github.com/google/go-containerregistry/pkg/crane"
)

func Example() {
	c := map[string][]byte{
		"/binary": []byte("binary contents"),
	}
	i, _ := crane.Image(c)
	d, _ := i.Digest()
	fmt.Println(d)
	// Output: sha256:d68299f4ce695504a7d9288b8f1a6686c0fcb64a7542e898be2d11305bba6717
}
