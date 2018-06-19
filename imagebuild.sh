#!/bin/bash
#==========================================================
#  Script name -- imagebuild.sh --  DESCRIPTION:
#
#
#  Author:  Lindsey Tulloch , ltulloch@redhat.com
#  CREATED:  2018-06-15 11:47:57 AM EDT

set -o errexit
set -o nounset
set -o pipefail

REGISTRY="quay.io"
REPO="onyiny_ang"
IMAGE="federation-v2"
TAG="proto"

root_dir="$(cd "$(dirname "$0")/.." ; pwd)"
base_dir="${root_dir}/federation-v2"
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

if [[ "${TRAVIS_TAG}" =~ ^v[0-9]+\.[0-9]+\.[0-9]+[a-z]*(-(r|R)(c|C)[0-9]+)*$ ]]; then
    echo "Pushing images with tags '${TRAVIS_TAG}' and 'latest'."
    VERSION="${TRAVIS_TAG}" MUTABLE_TAG="latest" docker build -t ${REGISTRY}${QUAY_USERNAME}/federation-v2:${MUTABLE_TAG}
  docker push ${REGISTRY}${QUAY_USERNAME}/federation-v2:${MUTABLE_TAG}
elif [[ "${TRAVIS_BRANCH}" == "master" ]]; then
    echo "Pushing images with default tags (git sha and 'canary')."
     docker build -t ${REGISTRY}${QUAY_USERNAME}/federation-v2:${TRAVIS_BRANCH}
  docker push ${REGISTRY}${QUAY_USERNAME}/federation-v2:${TRAVIS_BRANCH}

else
    echo "Nothing to deploy"
fi

