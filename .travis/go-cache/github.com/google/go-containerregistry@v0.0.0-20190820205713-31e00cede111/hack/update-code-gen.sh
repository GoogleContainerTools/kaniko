#!/bin/bash

# Copyright 2018 Google LLC
#
# Licensed under the Apache License, Version 2.0 (the "License");
# you may not use this file except in compliance with the License.
# You may obtain a copy of the License at
#
#     http://www.apache.org/licenses/LICENSE-2.0
#
# Unless required by applicable law or agreed to in writing, software
# distributed under the License is distributed on an "AS IS" BASIS,
# WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
# See the License for the specific language governing permissions and
# limitations under the License.

set -o errexit
set -o nounset
set -o pipefail

PROJECT_ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
export BOILER_PLATE_FILE="${PROJECT_ROOT}/hack/boilerplate/boilerplate.go.txt"

pushd ${PROJECT_ROOT}
trap popd EXIT

export GOPATH=$(go env GOPATH)
export PATH="${GOPATH}/bin:${PATH}"


go install ./vendor/k8s.io/code-generator/cmd/deepcopy-gen
go install ./vendor/github.com/maxbrunsfeld/counterfeiter
go run $PROJECT_ROOT/cmd/crane/help/main.go --dir=$PROJECT_ROOT/cmd/crane/doc/
go generate ./...
