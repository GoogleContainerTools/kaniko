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

# Builds the static Go image to execute in a Kubernetes job

FROM golang:1.15
ARG GOARCH=amd64
WORKDIR /go/src/github.com/GoogleContainerTools/kaniko

RUN echo $GOARCH > /goarch

#This arg is passed by docker buildx & contains the platform info in the form linux/amd64, linux/ppc64le etc.
ARG TARGETPLATFORM

#Capture ARCH has write to /goarch
RUN [ ! "x" = "x$TARGETPLATFORM" ] && `echo $TARGETPLATFORM |  awk '{split($0,a,"/"); print a[2]}' > /goarch` || echo "$GOARCH"

# Get GCR credential helper
RUN GOARCH=$(cat /goarch) && CGO_ENABLED=0 && \
  (mkdir -p /go/src/github.com/GoogleCloudPlatform || true)                  && \
  cd /go/src/github.com/GoogleCloudPlatform                                  && \
  git clone https://github.com/GoogleCloudPlatform/docker-credential-gcr.git && \
  cd /go/src/github.com/GoogleCloudPlatform/docker-credential-gcr            && \
  make deps OUT_DIR=/usr/local/bin                                           && \
  go build -ldflags "-linkmode external -extldflags -static" -i -o /usr/local/bin/docker-credential-gcr main.go

# Get Amazon ECR credential helper
RUN GOARCH=$(cat /goarch) && go get -u github.com/awslabs/amazon-ecr-credential-helper/ecr-login/cli/docker-credential-ecr-login && \
  make -C /go/src/github.com/awslabs/amazon-ecr-credential-helper

# ACR docker env credential helper
RUN GOARCH=$(cat /goarch) && (mkdir -p /go/src/github.com/chrismellard || true)   && \
  cd /go/src/github.com/chrismellard                                              && \
  git clone https://github.com/chrismellard/docker-credential-acr-env             && \
  cd  docker-credential-acr-env                                                   && \
  make build

# Add .docker config dir
RUN mkdir -p /kaniko/.docker

COPY . .
RUN make GOARCH=$(cat /goarch.txt)

# Generate latest ca-certificates

FROM debian:buster-slim AS certs

RUN \
  apt update && \
  apt install -y ca-certificates && \
  cat /etc/ssl/certs/* > /ca-certificates.crt

FROM scratch
COPY --from=0 /go/src/github.com/GoogleContainerTools/kaniko/out/executor /kaniko/executor
COPY --from=0 /usr/local/bin/docker-credential-gcr /kaniko/docker-credential-gcr
COPY --from=0 /go/src/github.com/awslabs/amazon-ecr-credential-helper/bin/local/docker-credential-ecr-login /kaniko/docker-credential-ecr-login
COPY --from=0 /go/src/github.com/chrismellard/docker-credential-acr-env/build/docker-credential-acr-env /kaniko/docker-credential-acr
COPY --from=certs /ca-certificates.crt /kaniko/ssl/certs/
COPY --from=0 /kaniko/.docker /kaniko/.docker
COPY files/nsswitch.conf /etc/nsswitch.conf
ENV HOME /root
ENV USER root
ENV PATH /usr/local/bin:/kaniko
ENV SSL_CERT_DIR=/kaniko/ssl/certs
ENV DOCKER_CONFIG /kaniko/.docker/
ENV DOCKER_CREDENTIAL_GCR_CONFIG /kaniko/.config/gcloud/docker_credential_gcr_config.json
WORKDIR /workspace
RUN ["docker-credential-gcr", "config", "--token-source=env"]

ENTRYPOINT ["/kaniko/executor"]