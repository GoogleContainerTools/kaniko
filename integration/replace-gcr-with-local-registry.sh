#!/bin/bash
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


# This script is needed due to the following bug:
# https://github.com/GoogleContainerTools/kaniko/issues/966


if [ "$#" -ne 1 ]; then
    echo "Please specify path to dockerfiles as first argument."
    echo "Usage: `basename $0` integration/dockerfiles"
	exit 2
fi

dir_with_docker_files=$1

for dockerfile in $dir_with_docker_files/*; do
    cat $dockerfile | grep '^FROM' | grep "gcr" | while read -r line; do
        gcr_repo=$(echo "$line" | awk '{ print $2 }')
        local_repo=$(echo "$gcr_repo" | sed -e "s/^.*gcr.io\(\/.*\)$/localhost:5000\1/")
        remove_digest=$(echo "$local_repo" | cut -f1 -d"@")
        echo "Running docker pull $gcr_repo"
        docker pull "$gcr_repo"
        echo "Running docker tag $gcr_repo $remove_digest"
        docker tag "$gcr_repo" "$remove_digest"
        echo "Running docker push $remove_digest"
        docker push "$remove_digest"
        echo "Updating dockerfile $dockerfile to use local repo $local_repo"
        sed -i -e "s/^\(FROM \).*gcr.io\(.*\)$/\1localhost:5000\2/" $dockerfile
    done
done
