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

package testutil

import (
	"fmt"
	"os"
	"os/user"
	"path/filepath"
	"reflect"
	"testing"

	"github.com/google/go-cmp/cmp"
)

// SetupFiles creates files at path
func SetupFiles(path string, files map[string]string) error {
	for p, c := range files {
		path := filepath.Join(path, p)
		if err := os.MkdirAll(filepath.Dir(path), 0750); err != nil {
			return err
		}
		if err := os.WriteFile(path, []byte(c), 0644); err != nil {
			return err
		}
	}
	return nil
}

type CurrentUser struct {
	*user.User

	PrimaryGroup string
}

func GetCurrentUser(t *testing.T) CurrentUser {
	currentUser, err := user.Current()
	if err != nil {
		t.Fatalf("Cannot get current user: %s", err)
	}
	groups, err := currentUser.GroupIds()
	if err != nil || len(groups) == 0 {
		t.Fatalf("Cannot get groups for current user: %s", err)
	}
	primaryGroupObj, err := user.LookupGroupId(groups[0])
	if err != nil {
		t.Fatalf("Could not lookup name of group %s: %s", groups[0], err)
	}
	primaryGroup := primaryGroupObj.Name

	return CurrentUser{
		User:         currentUser,
		PrimaryGroup: primaryGroup,
	}
}

func CheckDeepEqual(t *testing.T, expected, actual interface{}) {
	t.Helper()
	if diff := cmp.Diff(actual, expected); diff != "" {
		t.Errorf("%T differ (-got, +want): %s", expected, diff)
		return
	}
}

func CheckErrorAndDeepEqual(t *testing.T, shouldErr bool, err error, expected, actual interface{}) {
	t.Helper()
	if err := checkErr(shouldErr, err); err != nil {
		t.Error(err)
		return
	}
	if !reflect.DeepEqual(expected, actual) {
		diff := cmp.Diff(actual, expected)
		t.Errorf("%T differ (-got, +want): %s", expected, diff)
		return
	}
}

func CheckError(t *testing.T, shouldErr bool, err error) {
	if err := checkErr(shouldErr, err); err != nil {
		t.Error(err)
	}
}

func CheckNoError(t *testing.T, err error) {
	if err != nil {
		t.Errorf("%+v", err)
	}
}

func checkErr(shouldErr bool, err error) error {
	if err == nil && shouldErr {
		return fmt.Errorf("Expected error, but returned none")
	}
	if err != nil && !shouldErr {
		return fmt.Errorf("Unexpected error: %w", err)
	}
	return nil
}
