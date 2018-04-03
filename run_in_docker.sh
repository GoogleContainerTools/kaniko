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

#!/bin/bash
set -e

if [ $# -ne 2 ];
    then echo "Usage: run_in_docker.sh <context directory> <image tag>"
fi


context=$1
tag=$2

if [[ ! -e $HOME/.config/gcloud/application_default_credentials.json ]]; then
    echo "Application Default Credentials do not exist. Run [gcloud auth application-default login] to configure them"
    exit 1
fi

docker run \
    -v $HOME/.config/gcloud:/root/.config/gcloud \
    -v ${context}:/workspace \
    gcr.io/kbuild-project/executor:latest \
    /kbuild/executor -d ${tag}
