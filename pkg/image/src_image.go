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

package image

import (
	img "github.com/GoogleCloudPlatform/container-diff/pkg/image"
	"github.com/containers/image/docker"
)

// SourceImage is the image that will be modified by the executor
var SourceImage img.MutableSource

// InitializeSourceImage initializes the source image with the base image
func InitializeSourceImage(srcImg string) error {
	ref, err := docker.ParseReference("//" + srcImg)
	if err != nil {
		return err
	}
	ms, err := img.NewMutableSource(ref)
	if err != nil {
		return err
	}
	SourceImage = *ms
	return nil
}
