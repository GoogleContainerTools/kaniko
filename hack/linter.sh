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

set -e -o pipefail

DIR=$( cd "$( dirname "${BASH_SOURCE[0]}" )" && pwd )
BIN=${DIR}/bin

if [ ! -x "${BIN}/golangci-lint" ]; then
	echo "Installing GolangCI-Lint"

	# https://golangci-lint.run/usage/install/#linux-and-windows
	curl -sSfL https://raw.githubusercontent.com/golangci/golangci-lint/master/install.sh | sh -s -- -b ${BIN} v1.45.2
	"${BIN}/golangci-lint" --version
fi

"${BIN}/golangci-lint" run
