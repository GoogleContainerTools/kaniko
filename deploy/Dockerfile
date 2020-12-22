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

# Get the BUILDPLATFORM ARG
FROM alpine
ARG BUILDPLATFORM

# Builds the static Go image to execute in a Kubernetes job
FROM --platform=$BUILDPLATFORM golang:1.14
ARG BUILDPLATFORM
ARG TARGETPLATFORM
ARG TARGETARCH

#Capture all the args
RUN echo "I am running on $BUILDPLATFORM building for $TARGETPLATFORM with $TARGETARCH"
WORKDIR /go/src/github.com/GoogleContainerTools/kaniko

# Get GCR credential helper
RUN GOARCH=${TARGETARCH} && CGO_ENABLED=0 && \
  (mkdir -p /go/src/github.com/GoogleCloudPlatform || true)                  && \
  cd /go/src/github.com/GoogleCloudPlatform                                  && \
  git clone https://github.com/GoogleCloudPlatform/docker-credential-gcr.git && \
  cd /go/src/github.com/GoogleCloudPlatform/docker-credential-gcr            && \
  make deps OUT_DIR=/usr/local/bin                                           && \
  go build -ldflags "-linkmode external -extldflags -static" -i -o /usr/local/bin/docker-credential-gcr main.go

# Get Amazon ECR credential helper
RUN GOARCH=${TARGETARCH} && go get -u github.com/awslabs/amazon-ecr-credential-helper/ecr-login/cli/docker-credential-ecr-login && \
  make -C /go/src/github.com/awslabs/amazon-ecr-credential-helper

# ACR docker credential helper
COPY ./acr.patch /
RUN GOARCH=${TARGETARCH} && (mkdir -p /go/src/github.com/Azure || true)  && \
  cd /go/src/github.com/Azure                                             && \
  git clone https://github.com/Azure/acr-docker-credential-helper         && \
  cd /go/src/github.com/Azure/acr-docker-credential-helper                && \
  git checkout a79b541f3ee761f6cc4511863ed41fb038c19464                   && \
  git apply < /acr.patch                                                  && \
  make && cp -f bin/linux/${GOARCH}/docker-credential-acr-linux /usr/local/bin/docker-credential-acr-linux

#ACR docker env credential helper
RUN GOARCH=${TARGETARCH} && (mkdir -p /go/src/github.com/chrismellard || true)   && \
  cd /go/src/github.com/chrismellard                                              && \
  git clone https://github.com/chrismellard/docker-credential-acr-env             && \
  cd  docker-credential-acr-env                                                   && \
  make build && cp -f ./build/docker-credential-acr-env /usr/local/bin

# Add .docker config dir
RUN mkdir -p /kaniko/.docker

COPY . .
RUN make GOARCH=${TARGETARCH}

FROM scratch
COPY --from=1 /go/src/github.com/GoogleContainerTools/kaniko/out/executor /kaniko/executor
COPY --from=1 /usr/local/bin/docker-credential-gcr /kaniko/docker-credential-gcr
COPY --from=1 /go/src/github.com/awslabs/amazon-ecr-credential-helper/bin/local/docker-credential-ecr-login /kaniko/docker-credential-ecr-login
COPY --from=1 /usr/local/bin/docker-credential-acr-linux /kaniko/docker-credential-acr
COPY --from=1 /usr/local/bin/docker-credential-acr-env /kaniko/docker-credential-acr-env
COPY files/ca-certificates.crt /kaniko/ssl/certs/
COPY --from=1 /kaniko/.docker /kaniko/.docker
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
