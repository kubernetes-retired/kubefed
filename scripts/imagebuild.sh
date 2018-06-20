# Copyright 2018 The Kubernetes Authors.
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

set -o errexit
set -o nounset
set -o pipefail

root_dir="$(cd "$(dirname "$0")/.." ; pwd)"
base_dir="${root_dir}"
dockerfile_dir="${base_dir}/images/federation-v2"
cd "$base_dir" || {
  echo "Cannot cd to '$base_dir'. Aborting." >&2
  exit 1

}
cd ${base_dir}

export REGISTRY=quay.io/

echo "Copy apiserver"

cp ${base_dir}/bin/apiserver apiserver
echo "Copy controller manager"

cp ${base_dir}/bin/controller-manager controller-manager
echo "Building Federation-v2 docker image"

docker login -u "${QUAY_USERNAME}" -p "{QUAY_PASSWORD}" quay.io

#For testing purposes
export TRAVIS_BRANCH="master"

if [[ "${TRAVIS_BRANCH}" == "master" ]]; then
    echo "Pushing images with default tags (git sha and 'canary')."
     docker build ${dockerfile_dir} -t ${REGISTRY}${QUAY_USERNAME}/federation-v2:canary
  docker push ${REGISTRY}${QUAY_USERNAME}/federation-v2:canary

else
    echo "Nothing to deploy"
fi

rm apiserver
rm controller-manager
