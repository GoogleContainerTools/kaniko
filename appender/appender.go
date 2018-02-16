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

package appender

import (
	pkgimage "github.com/GoogleCloudPlatform/container-diff/pkg/image"
	"github.com/GoogleCloudPlatform/k8s-container-builder/pkg/constants"
	"github.com/GoogleCloudPlatform/k8s-container-builder/pkg/image"
	"github.com/containers/image/copy"
	"github.com/containers/image/signature"
	"github.com/containers/image/transports/alltransports"
	"github.com/sirupsen/logrus"

	"io/ioutil"
	"os"
	"sort"
	"strings"
)

// AppendLayersAndPushImage appends layers taken from snapshotter
// and then pushes the image to the specified destination
func AppendLayersAndPushImage(srcImg, dstImg string) error {
	if err := appendLayers(); err != nil {
		return err
	}
	return pushImage(dstImg)
}

func appendLayers() error {
	dir, err := os.Open(constants.WorkDir)
	if err != nil {
		return err
	}
	defer dir.Close()
	files, err := dir.Readdir(0)
	if err != nil {
		return err
	}
	var tars []string
	for _, file := range files {
		if strings.HasSuffix(file.Name(), ".tar") && strings.HasPrefix(file.Name(), "layer") {
			tars = append(tars, file.Name())
		}
	}
	sort.Strings(tars)
	for _, file := range tars {
		contents, err := ioutil.ReadFile(constants.WorkDir + file)
		if err != nil {
			return err
		}
		logrus.Debugf("Appending layer %s", file)
		image.SourceImage.AppendLayer(contents)
	}
	return nil
}

func pushImage(destImg string) error {
	logrus.Infof("Pushing image to %s", destImg)
	srcRef := &pkgimage.ProxyReference{
		ImageReference: nil,
		Src:            &image.SourceImage,
	}
	destRef, err := alltransports.ParseImageName("docker://" + destImg)
	if err != nil {
		return err
	}
	policyContext, err := getPolicyContext()
	if err != nil {
		return err
	}
	err = copy.Image(policyContext, destRef, srcRef, nil)
	return err
}

func getPolicyContext() (*signature.PolicyContext, error) {
	policy, err := signature.DefaultPolicy(nil)
	if err != nil {
		logrus.Debugf("Error retrieving policy: %s", err)
		return nil, err
	}
	policyContext, err := signature.NewPolicyContext(policy)
	if err != nil {
		logrus.Debugf("Error retrieving policy context: %s", err)
		return nil, err
	}
	return policyContext, nil
}
