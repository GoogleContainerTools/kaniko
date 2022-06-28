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

# taken from https://github.com/kubernetes/minikube/blob/b45b29c5df6f88c6ac0afd60079a6190dc1e32c9/hack/jenkins/linux_integration_tests_none.sh#L38
if ! kubeadm &>/dev/null; then
  echo "WARNING: kubeadm is not installed. will try to install."
  curl -LO "https://storage.googleapis.com/kubernetes-release/release/$(curl -s https://storage.googleapis.com/kubernetes-release/release/stable.txt)/bin/linux/amd64/kubeadm"
  sudo install kubeadm /usr/local/bin/kubeadm
fi

# "none" driver specific cleanup from previous runs.
sudo kubeadm reset -f --cri-socket unix:///var/run/cri-dockerd.sock || true
# kubeadm reset may not stop pods immediately
docker rm -f $(docker ps -aq) >/dev/null 2>&1 || true

# always install minikube, because version inconsistency is possible and could lead to weird errors
curl -Lo minikube https://storage.googleapis.com/minikube/releases/latest/minikube-linux-amd64
chmod +x minikube
sudo mv minikube /usr/local/bin/

# Minikube needs cri-dockerd to run clusters 1.24+
CRI_DOCKERD_VERSION="${CRI_DOCKERD_VERSION:-0.2.3}"
CRI_DOCKERD_BINARY_URL="https://github.com/Mirantis/cri-dockerd/releases/download/v${CRI_DOCKERD_VERSION}/cri-dockerd-${CRI_DOCKERD_VERSION}.amd64.tgz"

curl -Lo cri-dockerd.tgz $CRI_DOCKERD_BINARY_URL
tar xfz cri-dockerd.tgz
chmod +x cri-dockerd/cri-dockerd
sudo mv cri-dockerd/cri-dockerd /usr/bin/cri-dockerd

git clone https://github.com/Mirantis/cri-dockerd.git /tmp/cri-dockerd
sudo cp /tmp/cri-dockerd/packaging/systemd/* /etc/systemd/system
sudo systemctl daemon-reload
sudo systemctl enable cri-docker.service
sudo systemctl enable --now cri-docker.socket

CRICTL_VERSION="v1.17.0"
curl -L https://github.com/kubernetes-sigs/cri-tools/releases/download/$CRICTL_VERSION/crictl-${CRICTL_VERSION}-linux-amd64.tar.gz --output crictl-${CRICTL_VERSION}-linux-amd64.tar.gz
sudo tar zxvf crictl-$CRICTL_VERSION-linux-amd64.tar.gz -C /usr/local/bin
rm -f crictl-$CRICTL_VERSION-linux-amd64.tar.gz

sudo apt-get update
sudo apt-get install -y liblz4-tool
cat /proc/cpuinfo

minikube start --vm-driver=none --force --addons="registry,default-storageclass,storage-provisioner" || minikube logs;
minikube status
kubectl cluster-info
