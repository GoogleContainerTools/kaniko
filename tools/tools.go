//go:build tools
// +build tools

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

package tools

// dependencies https://github.com/golang/go/issues/48332
// These are placeholder imports the make go mod include these tools in its dependency graph.
import (
	_ "github.com/GoogleCloudPlatform/docker-credential-gcr/v2"
	_ "github.com/awslabs/amazon-ecr-credential-helper/ecr-login/cli/docker-credential-ecr-login"
	_ "github.com/chrismellard/docker-credential-acr-env"
)
