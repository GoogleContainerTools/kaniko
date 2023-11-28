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

package executor

import (
	"os"
	"path"
	"path/filepath"
	"reflect"
	"testing"

	"github.com/GoogleContainerTools/kaniko/pkg/util"
)

func Test_NewCompositeCache(t *testing.T) {
	r := NewCompositeCache()
	if reflect.TypeOf(r).String() != "*executor.CompositeCache" {
		t.Errorf("expected return to be *executor.CompositeCache but was %v", reflect.TypeOf(r).String())
	}
}

func Test_CompositeCache_AddKey(t *testing.T) {
	keys := []string{
		"meow",
		"purr",
	}
	r := NewCompositeCache()
	r.AddKey(keys...)
	if len(r.keys) != 2 {
		t.Errorf("expected keys to have length 2 but was %v", len(r.keys))
	}
}

func Test_CompositeCache_Key(t *testing.T) {
	r := NewCompositeCache("meow", "purr")
	k := r.Key()
	if k != "meow-purr" {
		t.Errorf("expected result to equal meow-purr but was %v", k)
	}
}

func Test_CompositeCache_Hash(t *testing.T) {
	r := NewCompositeCache("meow", "purr")
	h, err := r.Hash()
	if err != nil {
		t.Errorf("expected error to be nil but was %v", err)
	}

	expectedHash := "b4fd5a11af812a11a79d794007c842794cc668c8e7ebaba6d1e6d021b8e06c71"
	if h != expectedHash {
		t.Errorf("expected result to equal %v but was %v", expectedHash, h)
	}
}

func Test_CompositeCache_AddPath_dir(t *testing.T) {
	tmpDir := t.TempDir()

	content := `meow meow meow`
	if err := os.WriteFile(filepath.Join(tmpDir, "foo.txt"), []byte(content), 0777); err != nil {
		t.Errorf("got error writing temp file %v", err)
	}

	fn := func() string {
		r := NewCompositeCache()
		if err := r.AddPath(tmpDir, util.FileContext{}); err != nil {
			t.Errorf("expected error to be nil but was %v", err)
		}

		if len(r.keys) != 1 {
			t.Errorf("expected len of keys to be 1 but was %v", len(r.keys))
		}
		hash, err := r.Hash()
		if err != nil {
			t.Errorf("couldnt generate hash from test cache")
		}
		return hash
	}

	hash1 := fn()
	hash2 := fn()
	if hash1 != hash2 {
		t.Errorf("expected hash %v to equal hash %v", hash1, hash2)
	}
}
func Test_CompositeCache_AddPath_file(t *testing.T) {
	tmpfile, err := os.CreateTemp("/tmp", "foo.txt")
	if err != nil {
		t.Errorf("got error setting up test %v", err)
	}
	defer os.Remove(tmpfile.Name()) // clean up

	content := `meow meow meow`
	if _, err := tmpfile.Write([]byte(content)); err != nil {
		t.Errorf("got error writing temp file %v", err)
	}
	if err := tmpfile.Close(); err != nil {
		t.Errorf("got error closing temp file %v", err)
	}

	p := tmpfile.Name()
	fn := func() string {
		r := NewCompositeCache()
		if err := r.AddPath(p, util.FileContext{}); err != nil {
			t.Errorf("expected error to be nil but was %v", err)
		}

		if len(r.keys) != 1 {
			t.Errorf("expected len of keys to be 1 but was %v", len(r.keys))
		}
		hash, err := r.Hash()
		if err != nil {
			t.Errorf("couldnt generate hash from test cache")
		}
		return hash
	}

	hash1 := fn()
	hash2 := fn()
	if hash1 != hash2 {
		t.Errorf("expected hash %v to equal hash %v", hash1, hash2)
	}
}

func createFilesystemStructure(root string, directories, files []string) error {
	for _, d := range directories {
		dirPath := path.Join(root, d)
		if err := os.MkdirAll(dirPath, 0755); err != nil {
			return err
		}
	}

	for _, fileName := range files {
		filePath := path.Join(root, fileName)
		err := os.WriteFile(filePath, []byte(fileName), 0644)
		if err != nil {
			return err
		}
	}

	return nil
}

func setIgnoreContext(t *testing.T, content string) (util.FileContext, error) {
	var fileContext util.FileContext
	dockerIgnoreDir := t.TempDir()
	err := os.WriteFile(dockerIgnoreDir+".dockerignore", []byte(content), 0644)
	if err != nil {
		return fileContext, err
	}
	return util.NewFileContextFromDockerfile(dockerIgnoreDir, "")
}

func hashDirectory(dirpath string, fileContext util.FileContext) (string, error) {
	cache1 := NewCompositeCache()
	err := cache1.AddPath(dirpath, fileContext)
	if err != nil {
		return "", err
	}

	hash, err := cache1.Hash()
	if err != nil {
		return "", err
	}
	return hash, nil
}

func Test_CompositeKey_AddPath_Works(t *testing.T) {
	tests := []struct {
		name        string
		directories []string
		files       []string
	}{
		{
			name:        "empty",
			directories: []string{},
			files:       []string{},
		},
		{
			name:        "dirs",
			directories: []string{"foo", "bar", "foobar", "f/o/o"},
			files:       []string{},
		},
		{
			name:        "files",
			directories: []string{},
			files:       []string{"foo", "bar", "foobar"},
		},
		{
			name:        "all",
			directories: []string{"foo", "bar"},
			files:       []string{"foo/bar", "bar/baz", "foobar"},
		},
	}

	fileContext := util.FileContext{}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			testDir1 := t.TempDir()
			err := createFilesystemStructure(testDir1, test.directories, test.files)
			if err != nil {
				t.Fatalf("Error creating filesytem structure: %s", err)
			}

			testDir2 := t.TempDir()
			err = createFilesystemStructure(testDir2, test.directories, test.files)
			if err != nil {
				t.Fatalf("Error creating filesytem structure: %s", err)
			}

			hash1, err := hashDirectory(testDir1, fileContext)
			if err != nil {
				t.Fatalf("Failed to calculate hash: %s", err)
			}
			hash2, err := hashDirectory(testDir2, fileContext)
			if err != nil {
				t.Fatalf("Failed to calculate hash: %s", err)
			}

			if hash1 != hash2 {
				t.Errorf("Expected equal hashes, got: %s and %s", hash1, hash2)
			}
		})
	}
}

func Test_CompositeKey_AddPath_WithExtraFile_Works(t *testing.T) {
	tests := []struct {
		name        string
		directories []string
		files       []string
		extraFile   string
	}{
		{
			name:        "empty",
			directories: []string{},
			files:       []string{},
			extraFile:   "file",
		},
		{
			name:        "dirs",
			directories: []string{"foo", "bar", "foobar", "f/o/o"},
			files:       []string{},
			extraFile:   "f/o/o/extra",
		},
		{
			name:        "files",
			directories: []string{},
			files:       []string{"foo", "bar", "foobar"},
			extraFile:   "foo.extra",
		},
		{
			name:        "all",
			directories: []string{"foo", "bar"},
			files:       []string{"foo/bar", "bar/baz", "foobar"},
			extraFile:   "bar/extra",
		},
	}

	fileContext := util.FileContext{}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			testDir1 := t.TempDir()
			err := createFilesystemStructure(testDir1, test.directories, test.files)
			if err != nil {
				t.Fatalf("Error creating filesytem structure: %s", err)
			}

			testDir2 := t.TempDir()
			err = createFilesystemStructure(testDir2, test.directories, test.files)
			if err != nil {
				t.Fatalf("Error creating filesytem structure: %s", err)
			}
			extraPath := path.Join(testDir2, test.extraFile)
			err = os.WriteFile(extraPath, []byte(test.extraFile), 0644)
			if err != nil {
				t.Fatalf("Error creating filesytem structure: %s", err)
			}

			hash1, err := hashDirectory(testDir1, fileContext)
			if err != nil {
				t.Fatalf("Failed to calculate hash: %s", err)
			}
			hash2, err := hashDirectory(testDir2, fileContext)
			if err != nil {
				t.Fatalf("Failed to calculate hash: %s", err)
			}

			if hash1 == hash2 {
				t.Errorf("Expected different hashes, got: %s and %s", hash1, hash2)
			}
		})
	}
}

func Test_CompositeKey_AddPath_WithExtraDir_Works(t *testing.T) {
	tests := []struct {
		name        string
		directories []string
		files       []string
		extraDir    string
	}{
		{
			name:        "empty",
			directories: []string{},
			files:       []string{},
			extraDir:    "extra",
		},
		{
			name:        "dirs",
			directories: []string{"foo", "bar", "foobar", "f/o/o"},
			files:       []string{},
			extraDir:    "f/o/o/extra",
		},
		{
			name:        "files",
			directories: []string{},
			files:       []string{"foo", "bar", "foobar"},
			extraDir:    "foo.extra",
		},
		{
			name:        "all",
			directories: []string{"foo", "bar"},
			files:       []string{"foo/bar", "bar/baz", "foobar"},
			extraDir:    "bar/extra",
		},
	}

	fileContext := util.FileContext{}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			testDir1 := t.TempDir()
			err := createFilesystemStructure(testDir1, test.directories, test.files)
			if err != nil {
				t.Fatalf("Error creating filesytem structure: %s", err)
			}

			testDir2 := t.TempDir()
			err = createFilesystemStructure(testDir2, test.directories, test.files)
			if err != nil {
				t.Fatalf("Error creating filesytem structure: %s", err)
			}
			extraPath := path.Join(testDir2, test.extraDir)
			err = os.MkdirAll(extraPath, 0644)
			if err != nil {
				t.Fatalf("Error creating filesytem structure: %s", err)
			}

			hash1, err := hashDirectory(testDir1, fileContext)
			if err != nil {
				t.Fatalf("Failed to calculate hash: %s", err)
			}
			hash2, err := hashDirectory(testDir2, fileContext)
			if err != nil {
				t.Fatalf("Failed to calculate hash: %s", err)
			}

			if hash1 == hash2 {
				t.Errorf("Expected different hashes, got: %s and %s", hash1, hash2)
			}
		})
	}
}

func Test_CompositeKey_AddPath_WithExtraFilIgnored_Works(t *testing.T) {
	tests := []struct {
		name        string
		directories []string
		files       []string
		extraFile   string
	}{
		{
			name:        "empty",
			directories: []string{},
			files:       []string{},
			extraFile:   "extra",
		},
		{
			name:        "dirs",
			directories: []string{"foo", "bar", "foobar", "f/o/o"},
			files:       []string{},
			extraFile:   "f/o/o/extra",
		},
		{
			name:        "files",
			directories: []string{},
			files:       []string{"foo", "bar", "foobar"},
			extraFile:   "extra",
		},
		{
			name:        "all",
			directories: []string{"foo", "bar"},
			files:       []string{"foo/bar", "bar/baz", "foobar"},
			extraFile:   "bar/extra",
		},
	}

	fileContext, err := setIgnoreContext(t, "**/extra")
	if err != nil {
		t.Fatalf("Error setting exlusion context: %s", err)
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			testDir1 := t.TempDir()
			err = createFilesystemStructure(testDir1, test.directories, test.files)
			if err != nil {
				t.Fatalf("Error creating filesytem structure: %s", err)
			}

			testDir2 := t.TempDir()
			err = createFilesystemStructure(testDir2, test.directories, test.files)
			if err != nil {
				t.Fatalf("Error creating filesytem structure: %s", err)
			}
			extraPath := path.Join(testDir2, test.extraFile)
			err = os.WriteFile(extraPath, []byte(test.extraFile), 0644)
			if err != nil {
				t.Fatalf("Error creating filesytem structure: %s", err)
			}

			hash1, err := hashDirectory(testDir1, fileContext)
			if err != nil {
				t.Fatalf("Failed to calculate hash: %s", err)
			}
			hash2, err := hashDirectory(testDir2, fileContext)
			if err != nil {
				t.Fatalf("Failed to calculate hash: %s", err)
			}

			if hash1 != hash2 {
				t.Errorf("Expected equal hashes, got: %s and %s", hash1, hash2)
			}
		})
	}
}

func Test_CompositeKey_AddPath_WithExtraDirIgnored_Works(t *testing.T) {
	tests := []struct {
		name        string
		directories []string
		files       []string
		extraDir    string
	}{
		{
			name:        "empty",
			directories: []string{},
			files:       []string{},
			extraDir:    "extra",
		},
		{
			name:        "dirs",
			directories: []string{"foo", "bar", "foobar", "f/o/o"},
			files:       []string{},
			extraDir:    "f/o/o/extra",
		},
		{
			name:        "files",
			directories: []string{},
			files:       []string{"foo", "bar", "foobar"},
			extraDir:    "extra",
		},
		{
			name:        "all",
			directories: []string{"foo", "bar"},
			files:       []string{"foo/bar", "bar/baz", "foobar"},
			extraDir:    "bar/extra",
		},
	}

	fileContext, err := setIgnoreContext(t, "**/extra")
	if err != nil {
		t.Fatalf("Error setting exlusion context: %s", err)
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			testDir1 := t.TempDir()
			err := createFilesystemStructure(testDir1, test.directories, test.files)
			if err != nil {
				t.Fatalf("Error creating filesytem structure: %s", err)
			}

			testDir2 := t.TempDir()
			err = createFilesystemStructure(testDir2, test.directories, test.files)
			if err != nil {
				t.Fatalf("Error creating filesytem structure: %s", err)
			}
			extraPath := path.Join(testDir2, test.extraDir)
			err = os.MkdirAll(extraPath, 0644)
			if err != nil {
				t.Fatalf("Error creating filesytem structure: %s", err)
			}

			hash1, err := hashDirectory(testDir1, fileContext)
			if err != nil {
				t.Fatalf("Failed to calculate hash: %s", err)
			}
			hash2, err := hashDirectory(testDir2, fileContext)
			if err != nil {
				t.Fatalf("Failed to calculate hash: %s", err)
			}

			if hash1 != hash2 {
				t.Errorf("Expected equal hashes, got: %s and %s", hash1, hash2)
			}
		})
	}
}
