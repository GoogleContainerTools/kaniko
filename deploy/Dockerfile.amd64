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

FROM golang:1.10
WORKDIR /go/src/github.com/GoogleContainerTools/kaniko
COPY . .
RUN make
# Get GCR credential helper
ADD https://github.com/GoogleCloudPlatform/docker-credential-gcr/releases/download/v1.4.3-static/docker-credential-gcr_linux_amd64-1.4.3.tar.gz /usr/local/bin/
RUN tar -C /usr/local/bin/ -xvzf /usr/local/bin/docker-credential-gcr_linux_amd64-1.4.3.tar.gz
# Get Amazon ECR credential helper
RUN go get -u github.com/awslabs/amazon-ecr-credential-helper/ecr-login/cli/docker-credential-ecr-login
RUN make -C /go/src/github.com/awslabs/amazon-ecr-credential-helper linux-amd64

FROM scratch
COPY --from=0 /go/src/github.com/GoogleContainerTools/kaniko/out/executor /kaniko/executor
COPY --from=0 /usr/local/bin/docker-credential-gcr /kaniko/docker-credential-gcr
COPY --from=0 /go/src/github.com/awslabs/amazon-ecr-credential-helper/bin/linux-amd64/docker-credential-ecr-login /kaniko/docker-credential-ecr-login
COPY files/ca-certificates.crt /kaniko/ssl/certs/
COPY files/config.json /kaniko/.docker/
ENV HOME /root
ENV USER /root
ENV PATH /usr/local/bin:/kaniko
ENV SSL_CERT_DIR=/kaniko/ssl/certs
ENV DOCKER_CONFIG /kaniko/.docker/
ENV DOCKER_CREDENTIAL_GCR_CONFIG /kaniko/.config/gcloud/docker_credential_gcr_config.json
WORKDIR /workspace
RUN ["docker-credential-gcr", "config", "--token-source=env"]
ENTRYPOINT ["/kaniko/executor"]
