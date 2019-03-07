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

SUBCMD=
if [[ $# == 1 ]]; then
    SUBCMD="$1"
fi

REGISTRY=${REGISTRY:-quay.io}
REPO=${REPO:-kubernetes-multicluster}
IMAGE=${REGISTRY}/${REPO}/federation-v2

TRAVIS_TAG=${TRAVIS_TAG:-$(git tag -l --points-at HEAD)}
TRAVIS_BRANCH=${TRAVIS_BRANCH:-$(git rev-parse --abbrev-ref HEAD)}

[ -f "$base_dir/bin/hyperfed" ] || { echo "$base_dir/bin/hyperfed not found" ; exit 1 ;}
echo "travis tag: ${TRAVIS_TAG}"
echo "travis branch:${TRAVIS_BRANCH}"
if [[ "${TRAVIS_TAG}" =~ ^v([0-9]\.)+([0-9])[-a-zA-Z0-9]*([.0-9])* ]]; then
    echo "Using tag: '${TRAVIS_TAG}' and 'latest' ."
    TAG="${TRAVIS_TAG}"
    LATEST="latest"
elif [[ "${TRAVIS_BRANCH}" == "master" ]]; then
    echo "Using tag: 'canary'."
    TAG="canary"
    LATEST=""
else
    echo "Nothing to deploy. Image build skipped." >&2
    exit 0
fi

if [[ -z "${SUBCMD}" || "${SUBCMD}" == "build" ]]; then

    echo "Starting image build"
    echo "Copy hyperfed"
    cp ${base_dir}/bin/hyperfed ${dockerfile_dir}/hyperfed

    echo "Building Federation-v2 docker image"
    docker build ${dockerfile_dir} -t ${IMAGE}:${TAG}
    rm ${dockerfile_dir}/hyperfed
fi

if [[ -z "${SUBCMD}" || "${SUBCMD}" == "push" ]]; then

    echo "Logging into registry ${REGISTRY}"
    docker login -u "${QUAY_USERNAME}" -p "${QUAY_PASSWORD}" ${REGISTRY}

    echo "Pushing image with tag '${TAG}'."
    docker push ${IMAGE}:${TAG}

    if [ "$LATEST" == "latest" ]; then
        docker tag ${IMAGE}:${TAG} ${IMAGE}:${LATEST}
        echo "Pushing image with tag '${LATEST}'."
        docker push ${IMAGE}:${LATEST}
    fi
fi
