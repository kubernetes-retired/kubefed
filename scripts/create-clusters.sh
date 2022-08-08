#!/usr/bin/env bash

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

# This script handles the creation of multiple clusters using kind and the
# ability to create and configure an insecure container registry.

set -o errexit
set -o nounset
set -o pipefail

# shellcheck source=util.sh
source "${BASH_SOURCE%/*}/util.sh"
NUM_CLUSTERS="${NUM_CLUSTERS:-2}"
KIND_IMAGE="${KIND_IMAGE:-}"
KIND_TAG="${KIND_TAG:-v1.24.2@sha256:1f0cee2282f43150b52dc7933183ed96abdcfc8d293f30ec07082495874876f1}"
OS="$(uname)"

function create-clusters() {
  local num_clusters=${1}

  local image_arg=""
  if [[ "${KIND_IMAGE}" ]]; then
    image_arg="--image=${KIND_IMAGE}"
  elif [[ "${KIND_TAG}" ]]; then
    image_arg="--image=kindest/node:${KIND_TAG}"
  fi
  for i in $(seq "${num_clusters}"); do
    kind create cluster --name "cluster${i}" "${image_arg}"
    fixup-cluster "${i}"
    echo

  done

  echo "Waiting for clusters to be ready"
  check-clusters-ready "${num_clusters}"
}

function fixup-cluster() {
  local i=${1} # cluster num

  if [ "$OS" != "Darwin" ];then
    # Set container IP address as kube API endpoint in order for clusters to reach kube API servers in other clusters.
    local docker_ip
    docker_ip=$(docker inspect --format='{{range .NetworkSettings.Networks}}{{.IPAddress}}{{end}}' "cluster${i}-control-plane")
    kubectl config set-cluster "kind-cluster${i}" --server="https://${docker_ip}:6443"
  fi

  # Simplify context name
  kubectl config rename-context "kind-cluster${i}" "cluster${i}"
}

function check-clusters-ready() {
  for i in $(seq "${1}"); do
    util::wait-for-condition 'ok' "kubectl --context cluster${i} get --raw=/healthz &> /dev/null" 120
  done
}

echo "Creating ${NUM_CLUSTERS} clusters"
create-clusters "${NUM_CLUSTERS}"
kubectl config use-context cluster1

echo "Complete"
