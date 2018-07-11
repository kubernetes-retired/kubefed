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

base_dir="$(cd "$(dirname "$0")/.." ; pwd)"
dockerfile_dir="${base_dir}/images/federation-v2"

[ -f "$base_dir/bin/controller-manager" ] || { echo "$base_dir/bin/controller-manager not found" ; exit 1 ;}

echo "travis tag: ${TRAVIS_TAG}"
echo "travis branch:${TRAVIS_BRANCH}"
if [[ "${TRAVIS_TAG}" =~ ^v([0-9]\.)+([0-9])[-a-zA-Z0-9]*([.0-9])* ]]; then
   echo "Using tag: '${TRAVIS_TAG}' and 'latest' ."
    TAG="${TRAVIS_TAG}"
    LATEST="latest"
elif [[ "${TRAVIS_BRANCH}" == "master" ]]; then
   echo "Using tag: 'canary'."
    TAG="canary"
else
    echo "Nothing to deploy. Image build skipped." >&2
    exit 0
fi

echo "Starting image build"
export REGISTRY=quay.io/
export REPO=kubernetes-multicluster

echo "Copy controller manager"
cp ${base_dir}/bin/controller-manager ${dockerfile_dir}/controller-manager

echo "Logging into registry ${REGISTRY///}"
docker login -u "${QUAY_USERNAME}" -p "${QUAY_PASSWORD}" quay.io

echo "Building Federation-v2 docker image"
docker build ${dockerfile_dir} -t ${REGISTRY}${REPO}/federation-v2:${TAG}

echo "Pushing image with tag '${TAG}'."
docker push ${REGISTRY}${REPO}/federation-v2:${TAG}

if ! [ -z "${LATEST}" ]; then
   echo "Pushing image with tag '${LATEST}'."
   docker push ${REGISTRY}${REPO}/federation-v2:${LATEST}
fi

rm ${dockerfile_dir}/controller-manager
