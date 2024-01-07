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

func Test_multiKeyMultiValueArg_Set_shouldSplitArgumentLikeKVA(t *testing.T) {
	arg := make(multiKeyMultiValueArg)
	arg.Set("key=value")
	if arg["key"][0] != "value" {
		t.Error("Invalid split. key=value should be split to key=>value")
	}
}

func Test_multiKeyMultiValueArg_Set_ShouldAppendIfRepeated(t *testing.T) {
	arg := make(multiKeyMultiValueArg)
	arg.Set("key=v1")
	arg.Set("key=v2")
	if arg["key"][0] != "v1" || arg["key"][1] != "v2" {
		t.Error("Invalid repeat behavior. Repeated keys should append values")
	}
}

func Test_multiKeyMultiValueArg_Set_Composed(t *testing.T) {
	arg := make(multiKeyMultiValueArg)
	arg.Set("key1=value1;key2=value2")
	if arg["key1"][0] != "value1" || arg["key2"][0] != "value2" {
		t.Error("Invalid composed value parsing. key=value;key2=value2 should generate 2 keys")
	}
}

func Test_multiKeyMultiValueArg_Set_WithEmptyValueShouldWork(t *testing.T) {
	arg := make(multiKeyMultiValueArg)
	err := arg.Set("")
	if len(arg) != 0 || err != nil {
		t.Error("multiKeyMultiValueArg must handle empty value")
	}
}
