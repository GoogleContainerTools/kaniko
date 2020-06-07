#!/bin/bash
# Copyright 2020 Google LLC
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

# This script runs all integration tests except for
# TestRun and TestLayers
set -e

TESTS=$(./scripts/integration-test.sh -list=Test -mod=vendor)

TESTS=$(echo $TESTS | tr ' ' '\n' | grep 'Test'| grep -v 'TestRun' | grep -v 'TestLayers' | grep -v 'TestK8s' | grep -v 'TestSnapshotBenchmark')

RUN_ARG=''
count=0
for i in $TESTS; do
  if [ "$count" -gt "0" ]; then
    RUN_ARG="$RUN_ARG|$i"
  else
    RUN_ARG="$RUN_ARG$i"
  fi
  count=$((count+1))
done

echo $RUN_ARG
