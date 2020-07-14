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

source "$(dirname "${BASH_SOURCE}")/util.sh"
CREATE_INSECURE_REGISTRY="${CREATE_INSECURE_REGISTRY:-}"
CONFIGURE_INSECURE_REGISTRY_HOST="${CONFIGURE_INSECURE_REGISTRY_HOST:-}"
CONFIGURE_INSECURE_REGISTRY_CLUSTER="${CONFIGURE_INSECURE_REGISTRY_CLUSTER-y}"
CONTAINER_REGISTRY_HOST="${CONTAINER_REGISTRY_HOST:-172.17.0.1:5000}"
NUM_CLUSTERS="${NUM_CLUSTERS:-2}"
KIND_IMAGE="${KIND_IMAGE:-}"
KIND_TAG="${KIND_TAG:-}"
docker_daemon_config="/etc/docker/daemon.json"
containerd_config="/etc/containerd/config.toml"
OS=`uname`

function create-insecure-registry() {
  # Run insecure registry as container
  docker run -d -p 5000:5000 --restart=always --name registry registry:2
}

function configure-insecure-registry() {
  local err=
  if sudo test -f "${docker_daemon_config}"; then
    if sudo grep -q "\"insecure-registries\": \[\"${CONTAINER_REGISTRY_HOST}\"\]" ${docker_daemon_config}; then
      return 0
    elif sudo grep -q "\"insecure-registries\": " ${docker_daemon_config}; then
      echo <<EOF "Error: ${docker_daemon_config} exists and \
is already configured with an 'insecure-registries' entry but not set to ${CONTAINER_REGISTRY_HOST}. \
Please make sure it is removed and try again."
EOF
      err=true
    fi
  elif pgrep -a dockerd | grep -q 'insecure-registry'; then
    echo <<EOF "Error: CONFIGURE_INSECURE_REGISTRY_HOST=${CONFIGURE_INSECURE_REGISTRY_HOST} \
and about to write ${docker_daemon_config}, but dockerd is already configured with \
an 'insecure-registry' command line option. Please make the necessary changes or disable \
the command line option and try again."
EOF
    err=true
  fi

  if [[ "${err}" ]]; then
    if [[ "${CREATE_INSECURE_REGISTRY}" ]]; then
      docker kill registry &> /dev/null
      docker rm registry &> /dev/null
    fi
    return 1
  fi

  configure-insecure-registry-and-reload "sudo bash -c" $(pgrep dockerd) ${docker_daemon_config}
}

function configure-insecure-registry-and-reload() {
  local cmd_context="${1}" # context to run command e.g. sudo, docker exec
  local docker_pid="${2}"
  local config_file="${3}"
  ${cmd_context} "$(insecure-registry-config-cmd ${config_file})"
  ${cmd_context} "$(reload-daemon-cmd "${docker_pid}")"
}

function insecure-registry-config-cmd() {
  local config_file="${1}"
  case $config_file in
    $docker_daemon_config)
      echo "cat <<EOF > ${docker_daemon_config}
{
    \"insecure-registries\": [\"${CONTAINER_REGISTRY_HOST}\"]
}
EOF
"
      ;;
    $containerd_config)
     echo 'echo -e "[plugins.\"io.containerd.grpc.v1.cri\".registry.mirrors.\"'${CONTAINER_REGISTRY_HOST}'\"]\n  endpoint = [\"http://'${CONTAINER_REGISTRY_HOST}'\"]\n" >> '${containerd_config}
     ;;
    *)
     echo "Sorry, config insecure registy is not supported for $config_file"
     ;;
  esac
}

function reload-daemon-cmd() {
  echo "kill -s SIGHUP ${1}"
}

function create-clusters() {
  local num_clusters=${1}

  local image_arg=""
  if [[ "${KIND_IMAGE}" ]]; then
    image_arg="--image=${KIND_IMAGE}"
  elif [[ "${KIND_TAG}" ]]; then
    image_arg="--image=kindest/node:${KIND_TAG}"
  fi
  for i in $(seq ${num_clusters}); do
    kind create cluster --name "cluster${i}" ${image_arg}
    # TODO(font): remove once all workarounds are addressed.
    fixup-cluster ${i}
    echo
  done

  echo "Waiting for clusters to be ready"
  check-clusters-ready ${num_clusters}

  if [[ "${CONFIGURE_INSECURE_REGISTRY_CLUSTER}" ]]; then
    # TODO(font): Configure insecure registry on kind host cluster. Remove once
    # https://github.com/kubernetes-sigs/kind/issues/110 is resolved.
    echo "Configuring insecure container registry on kind host cluster"
    configure-insecure-registry-on-cluster 1
  fi
}

function fixup-cluster() {
  local i=${1} # cluster num

  if [ "$OS" != "Darwin" ];then
    # Set container IP address as kube API endpoint in order for clusters to reach kube API servers in other clusters.
    OLD_KUBECONFIG=${KUBECONFIG:-}
    export KUBECONFIG=$(mktemp)
    kind get kubeconfig --name "cluster${i}" --internal > ${KUBECONFIG}
    SERVER=$(kubectl config view -o jsonpath='{.clusters[0].cluster.server}')
    rm ${KUBECONFIG}
    KUBECONFIG=${OLD_KUBECONFIG}
    kubectl config set-cluster kind-cluster${i} --server=${SERVER}
  fi
  # Simplify context name
  kubectl config rename-context "kind-cluster${i}" "cluster${i}"
}

function check-clusters-ready() {
  for i in $(seq ${1}); do
    util::wait-for-condition 'ok' "kubectl --context cluster${i} get --raw=/healthz &> /dev/null" 120
  done
}

function configure-insecure-registry-on-cluster() {
  cmd_context="docker exec cluster${1}-control-plane bash -c"
  containerd_id=`${cmd_context} "pgrep -x containerd"`

  configure-insecure-registry-and-reload "${cmd_context}" ${containerd_id} ${containerd_config}
}

if [[ "${CREATE_INSECURE_REGISTRY}" ]]; then
  echo "Creating container registry on host"
  create-insecure-registry
fi

if [[ "${CONFIGURE_INSECURE_REGISTRY_HOST}" ]]; then
  echo "Configuring container registry on host"
  configure-insecure-registry
fi

echo "Creating ${NUM_CLUSTERS} clusters"
create-clusters ${NUM_CLUSTERS}
kubectl config use-context cluster1

echo "Complete"
