#!/bin/bash

set -e
set -u

tag="${1}"
dir=$(mktemp -d)

git clone --depth 1 --branch "${tag}" git@github.com:kubernetes/kubernetes.git "${dir}"
cp -r "${dir}/pkg/credentialprovider/." .

find . \( -name "OWNERS" \
  -o -name "OWNERS_ALIASES" \
  -o -name "BUILD" \
  -o -name "BUILD.bazel" \) -exec rm -f {} +


oldpkg="k8s.io/kubernetes/pkg/credentialprovider"
newpkg="github.com/vdemeester/k8s-pkg-credentialprovider"

find ./ -type f ! -name "sync.sh" ! -name "README.md"  \
  -exec sed -i '' "s,${oldpkg},${newpkg},g" {} \;


