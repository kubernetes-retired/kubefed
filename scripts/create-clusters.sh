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
CONFIGURE_INSECURE_REGISTRY="${CONFIGURE_INSECURE_REGISTRY:-}"
CONTAINER_REGISTRY_HOST="${CONTAINER_REGISTRY_HOST:-172.17.0.1:5000}"
NUM_CLUSTERS="${NUM_CLUSTERS:-2}"
OVERWRITE_KUBECONFIG="${OVERWRITE_KUBECONFIG:-}"
docker_daemon_config="/etc/docker/daemon.json"
kubeconfig="${HOME}/.kube/config"

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
    echo <<EOF "Error: CONFIGURE_INSECURE_REGISTRY=${CONFIGURE_INSECURE_REGISTRY} \
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

  configure-insecure-registry-and-reload "sudo bash -c" $(pgrep dockerd)
}

function configure-insecure-registry-and-reload() {
  local cmd_context="${1}" # context to run command e.g. sudo, docker exec
  local docker_pid="${2}"
  ${cmd_context} "$(insecure-registry-config-cmd)"
  ${cmd_context} "$(reload-docker-daemon-cmd "${docker_pid}")"
}

function insecure-registry-config-cmd() {
  echo "cat <<EOF > ${docker_daemon_config}
{
    \"insecure-registries\": [\"${CONTAINER_REGISTRY_HOST}\"]
}
EOF
"
}

function reload-docker-daemon-cmd() {
  echo "kill -s SIGHUP ${1}"
}

function create-clusters() {
  local num_clusters=${1}

  for i in $(seq ${num_clusters}); do
    kind create cluster --name "cluster${i}"
    # TODO(font): remove once all workarounds are addressed.
    fixup-cluster ${i}
    echo
  done

  echo "Waiting for clusters to be ready"
  check-clusters-ready ${num_clusters}

  # TODO(font): kind will create separate kubeconfig files for each cluster.
  # Remove once https://github.com/kubernetes-sigs/kind/issues/113 is resolved.
  if [[ "${OVERWRITE_KUBECONFIG}" ]]; then
    kubectl config view --flatten > ${kubeconfig}
    unset KUBECONFIG
  fi

  # TODO(font): Configure insecure registry on kind host cluster. Remove once
  # https://github.com/kubernetes-sigs/kind/issues/110 is resolved.
  echo "Configuring insecure container registry on kind host cluster"
  configure-insecure-registry-on-cluster 1
}

function fixup-cluster() {
  local i=${1} # cluster num

  local kubeconfig_path="$(kind get kubeconfig-path --name cluster${i})"
  export KUBECONFIG="${KUBECONFIG:-}:${kubeconfig_path}"

  # Simplify context name
  kubectl config rename-context "kubernetes-admin@cluster${i}" "cluster${i}"

  # TODO(font): Need to set container IP address in order for clusters to reach
  # kube API servers in other clusters until
  # https://github.com/kubernetes-sigs/kind/issues/111 is resolved.
  local container_ip_addr=$(docker inspect -f '{{range .NetworkSettings.Networks}}{{.IPAddress}}{{end}}' cluster${i}-control-plane)
  # Using the container ip allows the use of port 6443 instead of the
  # random port intended to be exposed on localhost.
  sed -i "s/localhost.*$/${container_ip_addr}:6443/" ${kubeconfig_path}

  # TODO(font): Need to rename auth user name to avoid conflicts when using
  # multiple cluster kubeconfigs. Remove once
  # https://github.com/kubernetes-sigs/kind/issues/112 is resolved.
  sed -i "s/kubernetes-admin/kubernetes-cluster${i}-admin/" ${kubeconfig_path}
}

function check-clusters-ready() {
  for i in $(seq ${1}); do
    util::wait-for-condition 'ok' "kubectl --context cluster${i} get --raw=/healthz &> /dev/null" 120
  done
}

function configure-insecure-registry-on-cluster() {
  configure-insecure-registry-and-reload "docker exec cluster${1}-control-plane bash -c" '$(pgrep dockerd)'
}

if [[ "${CREATE_INSECURE_REGISTRY}" ]]; then
  echo "Creating container registry on host"
  create-insecure-registry
fi

if [[ "${CONFIGURE_INSECURE_REGISTRY}" ]]; then
  echo "Configuring container registry on host"
  configure-insecure-registry
fi

echo "Creating ${NUM_CLUSTERS} clusters"
create-clusters ${NUM_CLUSTERS}

echo "Complete"

if [[ ! "${OVERWRITE_KUBECONFIG}" ]]; then
    echo <<EOF "OVERWRITE_KUBECONFIG was not set so ${kubeconfig} was not modified. \
You can access your clusters by setting your KUBECONFIG environment variable using:

export KUBECONFIG=\"${KUBECONFIG}\"

Then you can overwrite ${kubeconfig} if you prefer using:

kubectl config view --flatten > ${kubeconfig}
unset KUBECONFIG
"
EOF
fi
