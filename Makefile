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
REGISTRY?=gcr.io/kbuild-project

GOARCH = amd64
ORG := github.com/GoogleCloudPlatform
PROJECT := k8s-container-builder

REPOPATH ?= $(ORG)/$(PROJECT)

GO_FILES := $(shell find . -type f -name '*.go' -not -path "./vendor/*")

# These build tags are from the containers/image library.
# 
# container_image_ostree_stub allows building the library without requiring the libostree development libraries
# container_image_openpgp forces a Golang-only OpenPGP implementation for signature verification instead of the default cgo/gpgme-based implementation
#
# These build tags are from the containers/storage library.
#
# exclude_graphdriver_devicemapper
# exclude_graphdriver_btrfs
GO_BUILD_TAGS := "containers_image_ostree_stub containers_image_openpgp exclude_graphdriver_devicemapper exclude_graphdriver_btrfs"
GO_LDFLAGS := '-extldflags "-static"'
EXECUTOR_PACKAGE = $(REPOPATH)/executor

out/executor: $(GO_FILES)
	GOOS=$* GOARCH=$(GOARCH) CGO_ENABLED=1 go build -ldflags $(GO_LDFLAGS) -tags $(GO_BUILD_TAGS) -o $@ $(EXECUTOR_PACKAGE)

.PHONY: executor-image
executor-image: out/executor
	docker build -t $(REGISTRY)/executor:latest -f Dockerfile .

.PHONY: push-executor-image
push-executor-image: executor-image
	docker push $(REGISTRY)/executor:latest

.PHONY: test
test: out/executor
	@ ./test.sh
