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

package util

// ResolvEnvironmentReplacement resolves replacing env variables in fp from envs
// Ex: fp = $foo/newdir, envs = [foo=/foodir], then this should return /foodir/newdir
func ResolveEnvironmentReplacement(fp string, envs []string) string {

}
