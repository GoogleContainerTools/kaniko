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
	"testing"

	"github.com/google/go-containerregistry/pkg/crane"
)

func TestLayer(t *testing.T) {
	tcs := []struct {
		Name    string
		FileMap map[string][]byte
		Digest  string
	}{{
		Name:   "Empty contents",
		Digest: "sha256:89732bc7504122601f40269fc9ddfb70982e633ea9caf641ae45736f2846b004",
	}, {
		Name: "One file",
		FileMap: map[string][]byte{
			"/test": []byte("testy"),
		},
		Digest: "sha256:ec3ff19f471b99a76fb1c339c1dfdaa944a4fba25be6bcdc99fe7e772103079e",
	}, {
		Name: "Two files",
		FileMap: map[string][]byte{
			"/test":    []byte("testy"),
			"/testalt": []byte("footesty"),
		},
		Digest: "sha256:a48bcb7be3ab3ec608ee56eb80901224e19e31dc096cc06a8fd3a8dae1aa8947",
	}, {
		Name: "Many files",
		FileMap: map[string][]byte{
			"/1": []byte("1"),
			"/2": []byte("2"),
			"/3": []byte("3"),
			"/4": []byte("4"),
			"/5": []byte("5"),
			"/6": []byte("6"),
			"/7": []byte("7"),
			"/8": []byte("8"),
			"/9": []byte("9"),
		},
		Digest: "sha256:1e637602abbcab2dcedcc24e0b7c19763454a47261f1658b57569530b369ccb9",
	}}

	for _, tc := range tcs {
		t.Run(tc.Name, func(t *testing.T) {
			l, err := crane.Layer(tc.FileMap)
			if err != nil {
				t.Fatalf("Error calling layer: %v", err)
			}

			d, err := l.Digest()
			if err != nil {
				t.Fatalf("Error calling digest: %v", err)
			}
			if d.String() != tc.Digest {
				t.Fatalf("Incorrect digest, want %q, got %q", tc.Digest, d.String())
			}
		})
		t.Run(tc.Name+" is reproducible", func(t *testing.T) {
			l1, _ := crane.Layer(tc.FileMap)
			l2, _ := crane.Layer(tc.FileMap)
			d1, _ := l1.Digest()
			d2, _ := l2.Digest()
			if d1 != d2 {
				t.Fatalf("Non matching digests, want %q, got %q", d1, d2)
			}
		})
	}
}

func TestImage(t *testing.T) {
	tcs := []struct {
		Name    string
		FileMap map[string][]byte
		Digest  string
	}{{
		Name:   "Empty contents",
		Digest: "sha256:ea0bfd91e6495d74ae70510e91074289e391db7046769a46f7886a9c348b8726",
	}, {
		Name: "One file",
		FileMap: map[string][]byte{
			"/test": []byte("testy"),
		},
		Digest: "sha256:d1fd83b38f973d31da3ca7298f9e490e7715c9387bc609cd349ffc3909c20c8a",
	}, {
		Name: "Two files",
		FileMap: map[string][]byte{
			"/test": []byte("testy"),
			"/bar":  []byte("not useful"),
		},
		Digest: "sha256:d66dff1eaab5184591bb43a0f7c0ce24ffcab731a38a760e6631431966aaea2b",
	}, {
		Name: "Many files",
		FileMap: map[string][]byte{
			"/1": []byte("1"),
			"/2": []byte("2"),
			"/3": []byte("3"),
			"/4": []byte("4"),
			"/5": []byte("5"),
			"/6": []byte("6"),
			"/7": []byte("7"),
			"/8": []byte("8"),
			"/9": []byte("9"),
		},
		Digest: "sha256:6a79a016f70ff3d574612f7d5ccc4329ee1d573c239e3aeef1e4014fb7294b01",
	}}
	for _, tc := range tcs {
		t.Run(tc.Name, func(t *testing.T) {
			i, err := crane.Image(tc.FileMap)
			if err != nil {
				t.Fatalf("Error calling image: %v", err)
			}
			d, err := i.Digest()
			if err != nil {
				t.Fatalf("Error calling digest: %v", err)
			}
			if d.String() != tc.Digest {
				t.Fatalf("Incorrect digest, want %q, got %q", tc.Digest, d.String())
			}
		})
		t.Run(tc.Name+" is reproducible", func(t *testing.T) {
			i1, _ := crane.Image(tc.FileMap)
			i2, _ := crane.Image(tc.FileMap)
			d1, _ := i1.Digest()
			d2, _ := i2.Digest()
			if d1 != d2 {
				t.Fatalf("Non matching digests, want %q, got %q", d1, d2)
			}
		})
	}
}
