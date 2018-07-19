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

import (
	"bytes"
	"github.com/GoogleContainerTools/kaniko/testutil"
	"github.com/docker/docker/builder/dockerfile/instructions"
	"github.com/docker/docker/builder/dockerfile/parser"
	"github.com/google/go-containerregistry/pkg/v1"
	"github.com/google/go-containerregistry/pkg/v1/empty"
	"testing"
)

var (
	dockerfile = `
	FROM gcr.io/distroless/base:latest as base
	COPY . .
	
	FROM scratch as second
	ENV foopath context/foo
	COPY --from=0 $foopath context/b* /foo/
	
	FROM base
	ARG file
	COPY --from=second /foo $file`
)

func Test_StandardImage(t *testing.T) {
	stages, err := parse(dockerfile)
	if err != nil {
		t.Error(err)
	}
	original := retrieveRemoteImage
	defer func() {
		retrieveRemoteImage = original
	}()
	mock := func(image string) (v1.Image, error) {
		return nil, nil
	}
	retrieveRemoteImage = mock
	actual, err := RetrieveSourceImage(0, nil, stages)
	testutil.CheckErrorAndDeepEqual(t, false, err, nil, actual)
}
func Test_ScratchImage(t *testing.T) {
	stages, err := parse(dockerfile)
	if err != nil {
		t.Error(err)
	}
	actual, err := RetrieveSourceImage(1, nil, stages)
	expected := empty.Image
	testutil.CheckErrorAndDeepEqual(t, false, err, expected, actual)
}

func Test_TarImage(t *testing.T) {
	stages, err := parse(dockerfile)
	if err != nil {
		t.Error(err)
	}
	original := retrieveTarImage
	defer func() {
		retrieveTarImage = original
	}()
	mock := func(index int) (v1.Image, error) {
		return nil, nil
	}
	retrieveTarImage = mock
	actual, err := RetrieveSourceImage(2, nil, stages)
	testutil.CheckErrorAndDeepEqual(t, false, err, nil, actual)
}

// parse parses the contents of a Dockerfile and returns a list of commands
func parse(s string) ([]instructions.Stage, error) {
	p, err := parser.Parse(bytes.NewReader([]byte(s)))
	if err != nil {
		return nil, err
	}
	stages, _, err := instructions.Parse(p.AST)
	if err != nil {
		return nil, err
	}
	return stages, err
}
