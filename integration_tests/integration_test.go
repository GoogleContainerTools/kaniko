// +build integration

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
package main

import (
	"encoding/json"
	"fmt"
	"os/exec"
	"reflect"
	"sort"
	"testing"
)

var imageTests = []struct {
	description string
	repo        string
	added       []string
	deleted     []string
	modified    []string
}{
	{
		description: "test extract filesystem",
		repo:        "extract-filesystem",
		added:       []string{"/workspace", "/workspace/executor", "/workspace/Dockerfile"},
		deleted:     []string{"/proc", "/sys", "/dev", "/etc/hosts", "/etc/resolv.conf"},
	},
}

func Test_images(t *testing.T) {
	daemonPrefix := "daemon://"
	testRepo := "gcr.io/kbuild-test/"
	dockerPrefix := "docker-"
	kbuildPrefix := "kbuild-"

	for _, test := range imageTests {
		dockerImage := daemonPrefix + testRepo + dockerPrefix + test.repo
		kbuildImage := daemonPrefix + testRepo + kbuildPrefix + test.repo

		cmdOut, err := exec.Command("container-diff-linux-amd64", "diff", dockerImage, kbuildImage, "--type=file", "-j").Output()

		if err != nil {
			t.Fatal(err)
		}

		fmt.Println(string(cmdOut))

		var f interface{}
		err = json.Unmarshal(cmdOut, &f)

		if err != nil {
			t.Fatal(err)
		}
		adds, dels, mods := parseDiffOutput(f)
		checkEqual(t, test.added, adds)
		checkEqual(t, test.deleted, dels)
		checkEqual(t, test.modified, mods)
	}
}

func parseDiffOutput(f interface{}) ([]string, []string, []string) {
	diff := (f.([]interface{})[0]).(map[string]interface{})["Diff"]
	diffs := diff.(map[string]interface{})
	var adds = getFilenames(diffs, "Adds")
	var dels = getFilenames(diffs, "Dels")
	var mods = getFilenames(diffs, "Mods")
	return adds, dels, mods
}

func getFilenames(diffs map[string]interface{}, key string) []string {
	array := diffs[key]
	if array == nil {
		return nil
	}
	arr := array.([]interface{})
	var filenames []string
	for _, a := range arr {
		filename := a.(map[string]interface{})["Name"]
		filenames = append(filenames, filename.(string))
	}
	return filenames
}

func checkEqual(t *testing.T, actual, expected []string) {
	sort.Strings(actual)
	sort.Strings(expected)
	if !reflect.DeepEqual(expected, actual) {
		t.Errorf("%T differ.\nExpected\n%+v\nActual\n%+v", expected, expected, actual)
		return
	}
}
