# Copyright 2018 Google, Inc. All rights reserved.
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

FROM golang:1.24 AS builder
WORKDIR /src

# This arg is passed by docker buildx & contains the target CPU architecture (e.g., amd64, arm64, etc.)
ARG TARGETARCH
ARG TARGETOS

ENV GOARCH=$TARGETARCH
ENV GOOS=$TARGETOS

ENV CGO_ENABLED=0
ENV GOBIN=/usr/local/bin

# Add .docker config dir
RUN mkdir -p /kaniko/.docker

COPY . .

#  dependencies https://github.com/golang/go/issues/48332

# The versions of these tools and their transitive dependencies are controlled via go.mod.
# To update the version of any tool installed here, run
#
#   go get <tool>@<commit-sha-or-tag>
#   go mod vendor
#
# Then submit a PR with the changes to `go.mod`, `go.sum`, and `vendor`.

# Get GCR credential helper
RUN go install github.com/GoogleCloudPlatform/docker-credential-gcr/v2

# Get Amazon ECR credential helper
RUN go install github.com/awslabs/amazon-ecr-credential-helper/ecr-login/cli/docker-credential-ecr-login

# Get ACR docker env credential helper
RUN go install github.com/chrismellard/docker-credential-acr-env

RUN \
  --mount=type=cache,target=/root/.cache/go-build \
  --mount=type=cache,target=/go/pkg \
  make out/executor out/warmer

# Generate latest ca-certificates
FROM debian:bookworm-slim AS certs
RUN apt update && apt install -y ca-certificates

# use musl busybox since it's staticly compiled on all platforms
FROM busybox:musl AS busybox

FROM scratch AS kaniko-base-slim

# Create kaniko directory with world write permission to allow non root run
RUN --mount=from=busybox,dst=/usr/ ["busybox", "sh", "-c", "mkdir -p /kaniko && chmod 777 /kaniko"]

COPY --from=certs /etc/ssl/certs/ca-certificates.crt /kaniko/ssl/certs/
COPY files/nsswitch.conf /etc/nsswitch.conf
ENV HOME /root
ENV USER root
ENV PATH /usr/local/bin:/kaniko
ENV SSL_CERT_DIR=/kaniko/ssl/certs

FROM kaniko-base-slim AS kaniko-base

COPY --from=builder --chown=0:0 /usr/local/bin/docker-credential-gcr /kaniko/docker-credential-gcr
COPY --from=builder --chown=0:0 /usr/local/bin/docker-credential-ecr-login /kaniko/docker-credential-ecr-login
COPY --from=builder --chown=0:0 /usr/local/bin/docker-credential-acr-env /kaniko/docker-credential-acr-env

COPY --from=builder /kaniko/.docker /kaniko/.docker

ENV DOCKER_CONFIG /kaniko/.docker/
ENV DOCKER_CREDENTIAL_GCR_CONFIG /kaniko/.config/gcloud/docker_credential_gcr_config.json
WORKDIR /workspace

### FINAL STAGES ###

FROM kaniko-base AS kaniko-warmer

COPY --from=builder /src/out/warmer /kaniko/warmer

ENTRYPOINT ["/kaniko/warmer"]

FROM kaniko-base AS kaniko-executor

COPY --from=builder /src/out/executor /kaniko/executor

ENTRYPOINT ["/kaniko/executor"]

FROM kaniko-executor AS kaniko-debug

ENV PATH /usr/local/bin:/kaniko:/busybox

COPY --from=builder /src/out/warmer /kaniko/warmer

COPY --from=busybox /bin /busybox
# Declare /busybox as a volume to get it automatically in the path to ignore
VOLUME /busybox

RUN ["/busybox/mkdir", "-p", "/bin"]
RUN ["/busybox/ln", "-s", "/busybox/sh", "/bin/sh"]

FROM kaniko-base-slim AS kaniko-slim

COPY --from=builder /src/out/executor /kaniko/executor

ENTRYPOINT ["/kaniko/executor"]
