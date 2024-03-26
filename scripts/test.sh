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

DIR="$( cd "$( dirname "${BASH_SOURCE[0]}" )" && pwd )"

#set -e

RED='\033[0;31m'
GREEN='\033[0;32m'
RESET='\033[0m'

echo "Running go tests..."
go test -cover -coverprofile=out/coverage.out -v -timeout 120s `go list ./... | grep -v vendor | grep -v integration` | sed ''/PASS/s//$(printf "${GREEN}PASS${RESET}")/'' | sed ''/FAIL/s//$(printf "${RED}FAIL${RESET}")/''
GO_TEST_EXIT_CODE=${PIPESTATUS[0]}
if [[ $GO_TEST_EXIT_CODE -ne 0 ]]; then
    exit $GO_TEST_EXIT_CODE
fi

echo "Running validation scripts..."
scripts=(
    "$DIR/../hack/boilerplate.sh"
    "$DIR/../hack/gofmt.sh"
)
fail=0
for s in "${scripts[@]}"
do
    echo "RUN ${s}"
    if "${s}"; then
        echo -e "${GREEN}PASSED${RESET} ${s}"
    else
        echo -e "${RED}FAILED${RESET} ${s}"
        fail=1
    fi
done
exit $fail
