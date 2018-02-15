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
	"flag"
	"github.com/GoogleCloudPlatform/k8s-container-builder/appender"
	"github.com/GoogleCloudPlatform/k8s-container-builder/commands"
	"github.com/GoogleCloudPlatform/k8s-container-builder/contexts/dest"
	"github.com/GoogleCloudPlatform/k8s-container-builder/pkg/constants"
	"github.com/GoogleCloudPlatform/k8s-container-builder/pkg/dockerfile"
	"github.com/GoogleCloudPlatform/k8s-container-builder/pkg/env"
	"github.com/GoogleCloudPlatform/k8s-container-builder/pkg/snapshot"
	"github.com/GoogleCloudPlatform/k8s-container-builder/pkg/util"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
	"io/ioutil"
)

var dockerfilePath = flag.String("dockerfile", "/dockerfile/Dockerfile", "Path to Dockerfile.")
var source = flag.String("source", "", "Source context location")
var destImg = flag.String("dest", "", "Destination of final image")
var v = flag.String("verbosity", "info", "Logging verbosity")

func main() {
	flag.Parse()
	if err := setLogLevel(); err != nil {
		logrus.Fatal(err)
	}
	// Read and parse dockerfile
	b, err := ioutil.ReadFile(*dockerfilePath)
	if err != nil {
		logrus.Fatal(err)
	}

	stages, err := dockerfile.Parse(b)
	if err != nil {
		logrus.Fatal(err)
	}
	from := stages[0].BaseName
	// Unpack file system to root
	logrus.Infof("Extracting filesystem for %s...", from)
	err = util.ExtractFileSystemFromImage(from)
	if err != nil {
		logrus.Fatal(err)
	}

	l := snapshot.NewLayeredMap(util.Hasher())
	snapshotter := snapshot.NewSnapshotter(l, constants.RootDir)

	// Take initial snapshot
	if err := snapshotter.Init(); err != nil {
		logrus.Fatal(err)
	}
	// Save environment variables
	env.SetEnvironmentVariables(from)

	// Get context information
	context := dest.GetContext(*source)

	// Execute commands and take snapshots
	for _, s := range stages {
		for _, cmd := range s.Commands {
			dockerCommand := commands.GetCommand(cmd, context)
			if err := dockerCommand.ExecuteCommand(); err != nil {
				logrus.Fatal(err)
			}
			if err := snapshotter.TakeSnapshot(); err != nil {
				logrus.Fatal(err)
			}
		}
	}

	// Append layers and push image
	if err := appender.AppendLayersAndPushImage(from, *destImg); err != nil {
		logrus.Fatal(err)
	}
}

func setLogLevel() error {
	lvl, err := logrus.ParseLevel(*v)
	if err != nil {
		return errors.Wrap(err, "parsing log level")
	}
	logrus.SetLevel(lvl)
	return nil
}
