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

# Bump these on release
VERSION_MAJOR ?= 0
VERSION_MINOR ?= 1
VERSION_BUILD ?= 0

VERSION ?= v$(VERSION_MAJOR).$(VERSION_MINOR).$(VERSION_BUILD)

GOOS ?= $(shell go env GOOS)
GOARCH = amd64
ORG := github.com/GoogleCloudPlatform
PROJECT := k8s-container-builder
REGISTRY?=gcr.io/kbuild-project

REPOPATH ?= $(ORG)/$(PROJECT)

GO_FILES := $(shell find . -type f -name '*.go' -not -path "./vendor/*")
GO_LDFLAGS := '-extldflags "-static"'
GO_BUILD_TAGS := "containers_image_ostree_stub containers_image_openpgp exclude_graphdriver_devicemapper exclude_graphdriver_btrfs exclude_graphdriver_overlay"

EXECUTOR_PACKAGE = $(REPOPATH)/executor
KBUILD_PACKAGE = $(REPOPATH)/kbuild

out/executor: $(GO_FILES)
	GOARCH=$(GOARCH) GOOS=linux CGO_ENABLED=0 go build -ldflags $(GO_LDFLAGS) -tags $(GO_BUILD_TAGS) -o $@ $(EXECUTOR_PACKAGE)

.PHONY: install
install: $(GO_FILES) $(BUILD_DIR)
	GOOS=linux GOARCH=amd64 CGO_ENABLED=0 go install -ldflags $(GO_LDFLAGS) -tags $(GO_BUILD_TAGS) $(EXECUTOR_PACKAGE)

out/kbuild: $(GO_FILES)
	GOOS=$* GOARCH=$(GOARCH) CGO_ENABLED=0 go build -ldflags $(GO_LDFLAGS) -tags $(GO_BUILD_TAGS) -o $@ $(KBUILD_PACKAGE)

.PHONY: test
test: out/executor out/kbuild
	@ ./test.sh

.PHONY: integration-test
integration-test: out/executor out/kbuild
	@ ./integration-test.sh

.PHONY: images
images: $(GO_FILES)
	docker build -t $(REGISTRY)/executor:latest -f deploy/Dockerfile .
