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

package commands

import (
	"compress/gzip"
	"os"
	"path/filepath"
	"sort"
	"testing"

	"github.com/GoogleContainerTools/kaniko/pkg/dockerfile"
	"github.com/GoogleContainerTools/kaniko/pkg/util"
	"github.com/GoogleContainerTools/kaniko/testutil"
	v1 "github.com/google/go-containerregistry/pkg/v1"
	"github.com/moby/buildkit/frontend/dockerfile/instructions"
)

type TarList struct {
	tarName    string
	directory  string
	compressed bool
}

func createFile(tempDir string) error {
	fileName := filepath.Join(tempDir, "text.txt")
	file, err := os.Create(fileName)
	if err != nil {
		return err
	}
	defer file.Close()

	err = os.WriteFile(fileName, []byte("This is a test!\n"), 0644)
	if err != nil {
		return err
	}
	return nil
}

func createTar(tempDir string, toCreate TarList) error {
	if toCreate.compressed {
		file, err := os.OpenFile(filepath.Join(tempDir, toCreate.tarName), os.O_RDWR|os.O_CREATE|os.O_TRUNC, 0644)
		if err != nil {
			return err
		}

		gzipWriter := gzip.NewWriter(file)
		defer gzipWriter.Close()

		err = util.CreateTarballOfDirectory(filepath.Join(tempDir, toCreate.directory), gzipWriter)
		if err != nil {
			return err
		}
		return nil
	}

	tarFile, err := os.Create(filepath.Join(tempDir, toCreate.tarName))
	if err != nil {
		return err
	}
	err = util.CreateTarballOfDirectory(filepath.Join(tempDir, toCreate.directory), tarFile)
	if err != nil {
		return err
	}

	return nil
}

func setupAddTest(t *testing.T) string {
	tempDir := t.TempDir()

	err := createFile(tempDir)
	if err != nil {
		t.Errorf("couldn't create the file %v", err)
	}

	var tarFiles = []TarList{
		{
			tarName:    "a.tar",
			directory:  "a",
			compressed: false,
		},
		{
			tarName:    "b.tar.gz",
			directory:  "b",
			compressed: true,
		},
	}

	// Create directories with files and then create tar
	for _, toCreate := range tarFiles {

		err = os.Mkdir(filepath.Join(tempDir, toCreate.directory), 0755)
		if err != nil {
			t.Errorf("couldn't create directory %v", err)
		}

		err = createFile(filepath.Join(tempDir, toCreate.directory))
		if err != nil {
			t.Errorf("couldn't create file inside directory %v", err)
		}
		err = createTar(tempDir, toCreate)

		if err != nil {
			t.Errorf("couldn't create the tar %v", err)
		}
	}

	return tempDir
}

func Test_AddCommand(t *testing.T) {
	tempDir := setupAddTest(t)

	fileContext := util.FileContext{Root: tempDir}
	cfg := &v1.Config{
		Cmd:        nil,
		Env:        []string{},
		WorkingDir: tempDir,
	}
	buildArgs := dockerfile.NewBuildArgs([]string{})

	var addTests = []struct {
		name           string
		sourcesAndDest []string
		expectedDest   []string
	}{
		{
			name:           "add files into tempAddExecuteTest/",
			sourcesAndDest: []string{"text.txt", "a.tar", "b.tar.gz", "tempAddExecuteTest/"},
			expectedDest: []string{
				"text.txt",
				filepath.Join(tempDir, "a/"),
				filepath.Join(tempDir, "a/text.txt"),
				filepath.Join(tempDir, "b/"),
				filepath.Join(tempDir, "b/text.txt"),
			},
		},
	}

	for _, testCase := range addTests {
		t.Run(testCase.name, func(t *testing.T) {
			c := AddCommand{
				cmd: &instructions.AddCommand{
					SourcesAndDest: instructions.SourcesAndDest{SourcePaths: testCase.sourcesAndDest[0 : len(testCase.sourcesAndDest)-1],
						DestPath: testCase.sourcesAndDest[len(testCase.sourcesAndDest)-1]},
				},
				fileContext: fileContext,
			}
			c.ExecuteCommand(cfg, buildArgs)

			expected := []string{}
			resultDir := filepath.Join(tempDir, "tempAddExecuteTest/")
			for _, file := range testCase.expectedDest {
				expected = append(expected, filepath.Join(resultDir, file))
			}
			sort.Strings(expected)
			sort.Strings(c.snapshotFiles)
			testutil.CheckDeepEqual(t, expected, c.snapshotFiles)
		})
	}
}
