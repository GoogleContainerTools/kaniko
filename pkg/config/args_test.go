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

package config

import "testing"

func TestMultiArg_Set_shouldAppendValue(t *testing.T) {
	var arg multiArg
	arg.Set("value1")
	if len(arg) != 1 || arg[0] != "value1" {
		t.Error("Fist value was not appended")
	}
	arg.Set("value2")
	if len(arg) != 2 || arg[1] != "value2" {
		t.Error("Second value was not appended")
	}
}

func Test_KeyValueArg_Set_shouldSplitArgument(t *testing.T) {
	arg := make(keyValueArg)
	arg.Set("key=value")
	if arg["key"] != "value" {
		t.Error("Invalid split. key=value should be split to key=>value")
	}
}

func Test_KeyValueArg_Set_shouldAcceptEqualAsValue(t *testing.T) {
	arg := make(keyValueArg)
	arg.Set("key=value=something")
	if arg["key"] != "value=something" {
		t.Error("Invalid split. key=value=something should be split to key=>value=something")
	}
}
