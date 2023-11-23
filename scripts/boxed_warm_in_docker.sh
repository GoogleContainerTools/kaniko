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

# Test the warmer in boxed memory conditions.
# Attempt to run the warmer inside a container limited to 16MB of RAM. Use gcr.io/kaniko-project/warmer:latest image."
# Example: ./boxed_warm_in_docker.sh --image debian:trixie-slim
# 
set -e

rc=0
docker run \
	--memory=16m --memory-swappiness=0 \
        gcr.io/kaniko-project/warmer:latest \
	"$@" || rc=$?
	
>&2 echo "RC=$rc"
exit $rc

