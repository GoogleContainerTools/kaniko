/*
Copyright 2020 Google LLC

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

package filesystem

import (
	"io/ioutil"
	"os"
	"path/filepath"
	"reflect"
	"sort"
	"testing"

	"github.com/GoogleContainerTools/kaniko/pkg/util"
)

func Test_ResolvePaths(t *testing.T) {
	validateResults := func(
		t *testing.T,
		actualFiles,
		expectedFiles []string,
		err error,
	) {
		if err != nil {
			t.Errorf("expected err to be nil but was %s", err)
		}

		// Sort so that comparison is against consistent order
		sort.Strings(actualFiles)
		sort.Strings(expectedFiles)

		if !reflect.DeepEqual(actualFiles, expectedFiles) {
			t.Errorf("expected files to equal %s but was %s",
				expectedFiles, actualFiles,
			)
		}
	}

	t.Run("list of files", func(t *testing.T) {
		dir, err := ioutil.TempDir("", "snapshot-test")
		if err != nil {
			t.Fatal(err)
		}

		defer os.RemoveAll(dir)

		files := []string{
			"/foo/bar.txt",
			"/baz/boom.txt",
		}

		t.Run("all are symlinks", func(t *testing.T) {
			for _, f := range files {
				fLink := filepath.Join(dir, "link", f)
				fTarget := filepath.Join(dir, "target", f)

				if err := os.MkdirAll(filepath.Dir(fTarget), 0777); err != nil {
					t.Fatal(err)
				}

				if err := ioutil.WriteFile(fTarget, []byte{}, 0777); err != nil {
					t.Fatal(err)
				}

				if err := os.MkdirAll(filepath.Dir(fLink), 0777); err != nil {
					t.Fatal(err)
				}

				if err := os.Symlink(fTarget, fLink); err != nil {
					t.Fatal(err)
				}
			}

			t.Run("none are whitelisted", func(t *testing.T) {
				wl := []util.WhitelistEntry{}

				inputFiles := []string{}
				expectedFiles := []string{}

				for _, f := range files {
					link := filepath.Join(dir, "link", f)
					expectedFiles = append(expectedFiles, link)
					inputFiles = append(inputFiles, link)

					target := filepath.Join(dir, "target", f)
					expectedFiles = append(expectedFiles, target)
				}

				expectedFiles = filesWithParentDirs(expectedFiles)

				files, err := ResolvePaths(inputFiles, wl)

				validateResults(t, files, expectedFiles, err)
			})

			t.Run("some are whitelisted", func(t *testing.T) {
				wl := []util.WhitelistEntry{
					{
						Path: filepath.Join(dir, "link", "baz"),
					},
					{
						Path: filepath.Join(dir, "target", "foo"),
					},
				}

				expectedFiles := []string{}
				inputFiles := []string{}

				for _, f := range files {
					link := filepath.Join(dir, "link", f)
					inputFiles = append(inputFiles, link)

					if util.IsInProvidedWhitelist(link, wl) {
						t.Logf("skipping %s", link)
						continue
					}

					expectedFiles = append(expectedFiles, link)

					target := filepath.Join(dir, "target", f)

					if util.IsInProvidedWhitelist(target, wl) {
						t.Logf("skipping %s", target)
						continue
					}

					expectedFiles = append(expectedFiles, target)
				}

				link := filepath.Join(dir, "link", "zoom/")

				target := filepath.Join(dir, "target", "zaam/")
				if err := os.MkdirAll(target, 0777); err != nil {
					t.Fatal(err)
				}

				if err := ioutil.WriteFile(filepath.Join(target, "meow.txt"), []byte{}, 0777); err != nil {
					t.Fatal(err)
				}

				if err := os.Symlink(target, link); err != nil {
					t.Fatal(err)
				}

				file := filepath.Join(link, "meow.txt")
				inputFiles = append(inputFiles, file)

				expectedFiles = append(expectedFiles, link)

				targetFile := filepath.Join(target, "meow.txt")
				expectedFiles = append(expectedFiles, targetFile)

				expectedFiles = filesWithParentDirs(expectedFiles)

				files, err := ResolvePaths(inputFiles, wl)

				validateResults(t, files, expectedFiles, err)
			})
		})
	})

	t.Run("empty set of files", func(t *testing.T) {
		inputFiles := []string{}
		expectedFiles := []string{}

		wl := []util.WhitelistEntry{}

		files, err := ResolvePaths(inputFiles, wl)

		validateResults(t, files, expectedFiles, err)
	})
}
