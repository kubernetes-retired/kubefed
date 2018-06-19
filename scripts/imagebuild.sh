#!/bin/bash

set -o errexit
set -o nounset
set -o pipefail

root_dir="$(cd "$(dirname "$0")/.." ; pwd)"
base_dir="${root_dir}/federation-v2"
dockerfile_dir="${base_dir}/images/federation-v2/Dockerfile"
cd "$base_dir" || {
  echo "Cannot cd to '$base_dir'. Aborting." >&2
  exit 1

}
cd ${base_dir}

export REGISTRY=quay.io/
temp_dir="build/temp"

mkdir -p ${temp_dir}
echo "Copy apiserver"

cp ${base_dir}/bin/apiserver ${temp_dir}/apiserver
echo "Copy controller manager"

cp ${base_dir}/bin/controller-manager ${temp_dir}/controller-manager
echo "Building Federation-v2 docker image"

docker login -i "${QUAY_USERNAME}" -p "{QUAY_PASSWORD}" quay.io

if [[ "${TRAVIS_BRANCH}" == "master" ]]; then
    echo "Pushing images with default tags (git sha and 'canary')."
     docker build -f ${dockerfile_dir} -t ${REGISTRY}${QUAY_USERNAME}/federation-v2:canary
  docker push ${REGISTRY}${QUAY_USERNAME}/federation-v2:canary

else
    echo "Nothing to deploy"
fi

