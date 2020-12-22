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
VERSION_MAJOR ?= 1
VERSION_MINOR ?= 3
VERSION_BUILD ?= 0

VERSION ?= v$(VERSION_MAJOR).$(VERSION_MINOR).$(VERSION_BUILD)
VERSION_PACKAGE = $(REPOPATH/pkg/version)

SHELL := /bin/bash
GOOS ?= $(shell go env GOOS)
GOARCH ?= $(shell go env GOARCH)
ORG := github.com/GoogleContainerTools
PROJECT := kaniko
REGISTRY?=gcr.io/kaniko-project

REPOPATH ?= $(ORG)/$(PROJECT)
VERSION_PACKAGE = $(REPOPATH)/pkg/version

GO_FILES := $(shell find . -type f -name '*.go' -not -path "./vendor/*")
GO_LDFLAGS := '-extldflags "-static"
GO_LDFLAGS += -X $(VERSION_PACKAGE).version=$(VERSION)
GO_LDFLAGS += -w -s # Drop debugging symbols.
GO_LDFLAGS += '

EXECUTOR_PACKAGE = $(REPOPATH)/cmd/executor
WARMER_PACKAGE = $(REPOPATH)/cmd/warmer
KANIKO_PROJECT = $(REPOPATH)/kaniko
BUILD_ARG ?=

# Force using Go Modules and always read the dependencies from
# the `vendor` folder.
export GO111MODULE = on
export GOFLAGS = -mod=vendor


out/executor: $(GO_FILES)
	GOARCH=$(GOARCH) GOOS=linux CGO_ENABLED=0 go build -ldflags $(GO_LDFLAGS) -o $@ $(EXECUTOR_PACKAGE)

out/warmer: $(GO_FILES)
	GOARCH=$(GOARCH) GOOS=linux CGO_ENABLED=0 go build -ldflags $(GO_LDFLAGS) -o $@ $(WARMER_PACKAGE)

.PHONY: travis-setup
travis-setup:
	@ ./scripts/travis-setup.sh

.PHONY: minikube-setup
minikube-setup:
	@ ./scripts/minikube-setup.sh

.PHONY: test
test: out/executor
	@ ./scripts/test.sh

.PHONY: integration-test
integration-test:
	@ ./scripts/integration-test.sh

.PHONY: integration-test-run
integration-test-run:
	@ ./scripts/integration-test.sh -run "TestRun"

.PHONY: integration-test-layers
integration-test-layers:
	@ ./scripts/integration-test.sh -run "TestLayers"

.PHONY: integration-test-k8s
integration-test-k8s:
	@ ./scripts/integration-test.sh -run "TestK8s"

.PHONY: integration-test-misc
integration-test-misc:
	$(eval RUN_ARG=$(shell ./scripts/misc-integration-test.sh))
	@ ./scripts/integration-test.sh -run "$(RUN_ARG)"

.PHONY: images
images:
	docker build ${BUILD_ARG} --build-arg=TARGETARCH=$(GOARCH) --build-arg=BUILDPLATFORM=linux/$(GOARCH) -t $(REGISTRY)/executor:latest -f deploy/Dockerfile .
	docker build ${BUILD_ARG} --build-arg=TARGETARCH=$(GOARCH) --build-arg=BUILDPLATFORM=linux/$(GOARCH) -t $(REGISTRY)/executor:debug -f deploy/Dockerfile_debug .
	docker build ${BUILD_ARG} --build-arg=TARGETARCH=$(GOARCH) --build-arg=BUILDPLATFORM=linux/$(GOARCH) -t $(REGISTRY)/warmer:latest -f deploy/Dockerfile_warmer .

.PHONY: push
push:
	docker push $(REGISTRY)/executor:latest
	docker push $(REGISTRY)/executor:debug
	docker push $(REGISTRY)/warmer:latest
