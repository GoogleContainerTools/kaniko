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
set -e

# Ignore these paths in the following tests.
ignore="vendor\|out"
BOILERPLATEDIR=./hack/boilerplate
files=$(python3 ${BOILERPLATEDIR}/boilerplate.py --rootdir . --boilerplate-dir ${BOILERPLATEDIR})

# Grep returns a non-zero exit code if we don't match anything, which is good in this case.
set +e
relevant_files=$(echo "$files" | grep -v $ignore)
set -e

if [[ ! -z ${relevant_files} ]]; then
	echo "Boilerplate missing in:"
    echo "${relevant_files}"
	exit 1
fi
