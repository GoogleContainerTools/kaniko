#!/bin/bash
# Copyright 2020 Google LLC
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

set -ex

# conntrack is required for minikube 1.19 and higher for none driver
if ! conntrack --version &>/dev/null; then
  echo "WARNING: No contrack is not installed"
  sudo apt-get update -qq
  sudo apt-get -qq -y install conntrack
fi

if ! command -v minikube; then
  curl -Lo minikube https://storage.googleapis.com/minikube/releases/latest/minikube-linux-amd64
  chmod +x minikube
  sudo mv minikube /usr/local/bin/
fi


# Minikube needs cri-dockerd to run clusters 1.24+
CRI_DOCKERD_VERSION="${CRI_DOCKERD_VERSION:-0.2.3}"
OS_CODENAME=$(cat /etc/os-release | grep VERSION_CODENAME | cut -f2 -d"=")
CRI_DOCKERD_PACKAGE_URL="https://github.com/Mirantis/cri-dockerd/releases/download/v${CRI_DOCKERD_VERSION}/cri-dockerd_${CRI_DOCKERD_VERSION}.3-0.ubuntu-${OS_CODENAME}_amd64.deb"
curl -Lo cri-dockerd.deb $CRI_DOCKERD_PACKAGE_URL
sudo apt-get update
sudo apt-get install -y containerd runc ./cri-dockerd.deb

if ! command -v crictl; then
  CRICTL_VERSION="v1.24.1"
  curl -L https://github.com/kubernetes-sigs/cri-tools/releases/download/$CRICTL_VERSION/crictl-${CRICTL_VERSION}-linux-amd64.tar.gz --output crictl-${CRICTL_VERSION}-linux-amd64.tar.gz
  sudo tar zxvf crictl-$CRICTL_VERSION-linux-amd64.tar.gz -C /usr/local/bin
  rm -f crictl-$CRICTL_VERSION-linux-amd64.tar.gz
fi

sudo apt-get update
sudo apt-get install -y liblz4-tool
cat /proc/cpuinfo

sudo systemctl unmask docker
minikube start --vm-driver=none --force
minikube status
minikube addons enable registry
kubectl cluster-info

kubectl port-forward --namespace kube-system service/registry 5000:80 &
