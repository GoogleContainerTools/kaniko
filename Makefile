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
VERSION_MINOR ?= 23
VERSION_BUILD ?= 2

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
	GOARCH=$(GOARCH) GOOS=$(GOOS) CGO_ENABLED=0 go build -ldflags $(GO_LDFLAGS) -o $@ $(EXECUTOR_PACKAGE)

out/warmer: $(GO_FILES)
	GOARCH=$(GOARCH) GOOS=$(GOOS) CGO_ENABLED=0 go build -ldflags $(GO_LDFLAGS) -o $@ $(WARMER_PACKAGE)

.PHONY: install-container-diff
install-container-diff:
	@ curl -LO https://github.com/GoogleContainerTools/container-diff/releases/download/v0.17.0/container-diff-$(GOOS)-amd64 && \
		chmod +x container-diff-$(GOOS)-amd64 && sudo mv container-diff-$(GOOS)-amd64 /usr/local/bin/container-diff

.PHONY: k3s-setup
k3s-setup:
	@ ./scripts/k3s-setup.sh

.PHONY: test
test: out/executor
	@ ./scripts/test.sh

test-with-coverage: test
	go tool cover -html=out/coverage.out


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

.PHONY: k8s-executor-build-push
k8s-executor-build-push:
	DOCKER_BUILDKIT=1 docker build ${BUILD_ARG} --build-arg=GOARCH=$(GOARCH) --build-arg=TARGETOS=linux -t $(REGISTRY)/executor:latest -f deploy/Dockerfile --target kaniko-executor .
	docker push $(REGISTRY)/executor:latest


.PHONY: images
images: DOCKER_BUILDKIT=1
images:
	docker build ${BUILD_ARG} --build-arg=TARGETARCH=$(GOARCH) --build-arg=TARGETOS=linux -t $(REGISTRY)/executor:latest -f deploy/Dockerfile --target kaniko-executor .
	docker build ${BUILD_ARG} --build-arg=TARGETARCH=$(GOARCH) --build-arg=TARGETOS=linux -t $(REGISTRY)/executor:debug -f deploy/Dockerfile --target kaniko-debug .
	docker build ${BUILD_ARG} --build-arg=TARGETARCH=$(GOARCH) --build-arg=TARGETOS=linux -t $(REGISTRY)/executor:slim -f deploy/Dockerfile --target kaniko-slim .
	docker build ${BUILD_ARG} --build-arg=TARGETARCH=$(GOARCH) --build-arg=TARGETOS=linux -t $(REGISTRY)/warmer:latest -f deploy/Dockerfile --target kaniko-warmer .

.PHONY: push
push:
	docker push $(REGISTRY)/executor:latest
	docker push $(REGISTRY)/executor:debug
	docker push $(REGISTRY)/executor:slim
	docker push $(REGISTRY)/warmer:latest
