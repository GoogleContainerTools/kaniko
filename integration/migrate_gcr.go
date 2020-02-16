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

package integration

import (
	"bufio"
	"fmt"
	"os"
	"os/exec"
	"path"
	"regexp"
	"strings"
)

// This function crawl through dockerfiles and replace all reference to "gcr.io/" in FROM instruction and replace it targetRepo
// The result is the array containing all modified file
// Each of this file is saved
func MigrateGCRRegistry(dockerfilesPath string, dockerfiles []string, targetRepo string) ([]string, error) {
	var savedFiles []string
	importedImages := map[string]interface{}{}

	for _, dockerfile := range dockerfiles {
		if referencedImages, savedFile, err := migrateFile(dockerfilesPath, dockerfile, targetRepo); err != nil {
			if savedFile {
				savedFiles = append(savedFiles, dockerfile)
			}
			return savedFiles, err
		} else if savedFile {
			savedFiles = append(savedFiles, dockerfile)
			for _, referencedImage := range referencedImages {
				importedImages[referencedImage] = nil
			}
		}
	}
	for image := range importedImages {
		if err := importImage(image, targetRepo); err != nil {
			return savedFiles, err
		}
	}
	return savedFiles, nil
}

// This function rollback all previously modified files
func RollbackMigratedFiles(dockerfilesPath string, dockerfiles []string) []error {
	var result []error
	for _, dockerfile := range dockerfiles {
		fmt.Printf("Rolling back %s\n", dockerfile)
		if err := recoverDockerfile(dockerfilesPath, dockerfile); err != nil {
			result = append(result, err)
		}
	}
	return result
}

// Import the gcr.io image such as gcr.io/my-image to targetRepo
func importImage(image string, targetRepo string) error {
	fmt.Printf("Importing %s to %s\n", image, targetRepo)
	targetImage := strings.ReplaceAll(image, "gcr.io/", targetRepo)
	pullCmd := exec.Command("docker", "pull", image)
	if out, err := RunCommandWithoutTest(pullCmd); err != nil {
		return fmt.Errorf("Failed to pull image %s with docker command \"%s\": %s %s", image, pullCmd.Args, err, string(out))
	}

	tagCmd := exec.Command("docker", "tag", image, targetImage)
	if out, err := RunCommandWithoutTest(tagCmd); err != nil {
		return fmt.Errorf("Failed to tag image %s to %s with docker command \"%s\": %s %s", image, targetImage, tagCmd.Args, err, string(out))
	}

	pushCmd := exec.Command("docker", "push", targetImage)
	if out, err := RunCommandWithoutTest(pushCmd); err != nil {
		return fmt.Errorf("Failed to push image %s with docker command \"%s\": %s %s", targetImage, pushCmd.Args, err, string(out))
	}
	return nil
}

// takes a dockerfile and replace each gcr.io/ occurrence in FROM instruction and replace it with imageRepo
// return true if the file was saved
// if so, the array is non nil and contains each gcr image name
func migrateFile(dockerfilesPath string, dockerfile string, imageRepo string) ([]string, bool, error) {
	var input *os.File
	var output *os.File
	var err error
	var referencedImages []string

	if input, err = os.Open(path.Join(dockerfilesPath, dockerfile)); err != nil {
		return nil, false, err
	}
	defer input.Close()

	var lines []string
	scanner := bufio.NewScanner(input)
	scanner.Split(bufio.ScanLines)
	for scanner.Scan() {
		line := scanner.Text()
		if isFromGcrBaseImageInstruction(line) {
			referencedImages = append(referencedImages, strings.Trim(strings.Split(line, " ")[1], " "))
			lines = append(lines, strings.ReplaceAll(line, gcrRepoPrefix, imageRepo))
		} else {
			lines = append(lines, line)
		}
	}
	rawOutput := []byte(strings.Join(append(lines, ""), "\n"))

	if len(referencedImages) == 0 {
		return nil, false, nil
	}

	if err = saveDockerfile(dockerfilesPath, dockerfile); err != nil {
		return nil, false, err
	}

	if output, err = os.Create(path.Join(dockerfilesPath, dockerfile)); err != nil {
		return nil, true, err
	}
	defer output.Close()

	if written, err := output.Write(rawOutput); err != nil {
		return nil, true, err
	} else if written != len(rawOutput) {
		return nil, true, fmt.Errorf("invalid number of byte written. Got %d, expected %d", written, len(rawOutput))
	}
	return referencedImages, true, nil

}

func isFromGcrBaseImageInstruction(line string) bool {
	result, _ := regexp.MatchString(fmt.Sprintf("FROM +%s", gcrRepoPrefix), line)
	return result
}

func saveDockerfile(dockerfilesPath string, dockerfile string) error {
	return os.Rename(path.Join(dockerfilesPath, dockerfile), path.Join(dockerfilesPath, saveName(dockerfile)))
}

func recoverDockerfile(dockerfilesPath string, dockerfile string) error {
	return os.Rename(path.Join(dockerfilesPath, saveName(dockerfile)), path.Join(dockerfilesPath, dockerfile))
}

func saveName(dockerfile string) string {
	return fmt.Sprintf("%s_save_%d", dockerfile, os.Getpid())
}
