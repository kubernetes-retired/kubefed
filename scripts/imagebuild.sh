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

[ -f "$base_dir/bin/apiserver" ] || { echo "$base_dir/bin/apiserver not found" ; exit 1 ;}
[ -f "$base_dir/bin/controller-manager" ] || { echo "$base_dir/bin/controller-manager not found" ; exit 1 ;}

if [[ "${TRAVIS_BRANCH}" != "master" ]]; then
  echo "Not on master branch. Image build skipped." >&2
  exit 0
fi

echo "Starting image build"
export REGISTRY=quay.io/
export REPO=kubernetes-multicluster
echo "Copy apiserver"
cp ${base_dir}/bin/apiserver ${dockerfile_dir}/apiserver

echo "Copy controller manager"
cp ${base_dir}/bin/controller-manager ${dockerfile_dir}/controller-manager

echo "Building Federation-v2 docker image"
docker login -u "${QUAY_USERNAME}" -p "${QUAY_PASSWORD}" quay.io

echo "Pushing images with default tags (git sha and 'canary')."
docker build ${dockerfile_dir} -t ${REGISTRY}${REPO}/federation-v2:canary
docker push ${REGISTRY}${REPO}/federation-v2:canary

rm ${dockerfile_dir}/apiserver
rm ${dockerfile_dir}/controller-manager
