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

DIR=$( cd "$( dirname "${BASH_SOURCE[0]}" )" && pwd )
EXAMPLES_DIR=${DIR}/../examples

# Prompt the user for the version
echo -n "Enter the version for this release - ex: v1.14.0: "
read VERSION

# Remove 'v' prefix from version
MAKEFILE_VERSION=$(echo $VERSION | sed 's/^[v]//')

# Extract major, minor, and build version numbers
VERSION_MAJOR=$(echo $MAKEFILE_VERSION | cut -d. -f1)
VERSION_MINOR=$(echo $MAKEFILE_VERSION | cut -d. -f2)
VERSION_BUILD=$(echo $MAKEFILE_VERSION | cut -d. -f3)

echo "Processing (takes some time)..."

# Get the current date
DATE=$(date +'%Y-%m-%d')

# you can pass your github token with --token here if you run out of requests
# Capture output and replace newline characters with a placeholder
PULL_REQS=$(go run ${DIR}/release_notes/listpullreqs.go | tr '\n' '|')
CONTRIBUTORS=$(git log "$(git describe  --abbrev=0)".. --format="%aN" --reverse | sort | uniq | awk '{printf "- %s\n", $0 }' | tr '\n' '|')

# Substitute placeholders with actual data in the template
TEMP_CHANGELOG=$(mktemp)
TEMP_CHANGELOG_FIXED=$(mktemp)
sed -e "s@{{PULL_REQUESTS}}@${PULL_REQS}@g" \
    -e "s@{{CONTRIBUTORS}}@${CONTRIBUTORS}@g" \
    -e "s@{{VERSION}}@${VERSION}@g" \
    -e "s@{{DATE}}@${DATE}@g" \
    ${DIR}/release_notes/changelog_template.txt > $TEMP_CHANGELOG

# Replace '|' with '\n' in temporary changelog
sed 's/|/\n/g' $TEMP_CHANGELOG > $TEMP_CHANGELOG_FIXED

# Prepend to CHANGELOG.md
cat $TEMP_CHANGELOG_FIXED CHANGELOG.md > TEMP && mv TEMP CHANGELOG.md

echo "Prepended the following release information to CHANGELOG.md"
echo ""
cat  $TEMP_CHANGELOG_FIXED

# Optionally, clean up the fixed temporary changlog file
rm $TEMP_CHANGELOG_FIXED

# Cleanup
rm $TEMP_CHANGELOG

echo "Updated Makefile for the new version: $VERSION"
# Update Makefile
sed -i.bak \
    -e "s|VERSION_MAJOR ?=.*|VERSION_MAJOR ?= $VERSION_MAJOR|" \
    -e "s|VERSION_MINOR ?=.*|VERSION_MINOR ?= $VERSION_MINOR|" \
    -e "s|VERSION_BUILD ?=.*|VERSION_BUILD ?= $VERSION_BUILD|" \
    ./Makefile

# Cleanup
rm ./Makefile.bak
