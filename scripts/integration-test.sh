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

set -ex

GCS_BUCKET="${GCS_BUCKET:-gs://kaniko-test-bucket}"
IMAGE_REPO="${IMAGE_REPO:-gcr.io/kaniko-test}"

docker version

# Sets up a kokoro (Google internal integration testing tool) environment
echo "Running integration tests..."
make out/executor
make out/warmer
go test ./integration/... -v --bucket "${GCS_BUCKET}" --repo "${IMAGE_REPO}" --timeout 50m "$@"
