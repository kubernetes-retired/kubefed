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
CONFIGURE_INSECURE_REGISTRY="${CONFIGURE_INSECURE_REGISTRY:-}"
CONTAINER_REGISTRY_HOST="${CONTAINER_REGISTRY_HOST:-172.17.0.1:5000}"
NUM_CLUSTERS="${NUM_CLUSTERS:-2}"
OVERWRITE_KUBECONFIG="${OVERWRITE_KUBECONFIG:-}"
docker_daemon_config="/etc/docker/daemon.json"
kubeconfig="${HOME}/.kube/config"

function create-and-configure-insecure-registry() {
  # Run insecure registry as container
  docker run -d -p 5000:5000 --restart=always --name registry registry:2

  if [[ ! "${CONFIGURE_INSECURE_REGISTRY}" ]]; then
    return
  fi

  local err=
  if sudo test -f "${docker_daemon_config}"; then
    echo <<EOF "Error: ${docker_daemon_config} exists and \
CONFIGURE_INSECURE_REGISTRY=${CONFIGURE_INSECURE_REGISTRY}. This script needs \
to add an 'insecure-registries' entry with host '${CONTAINER_REGISTRY_HOST}' to \
${docker_daemon_config}. Please make the necessary changes or backup and try again."
EOF
    err=true
  elif pgrep -a dockerd | grep -q 'insecure-registry'; then
    echo <<EOF "Error: CONFIGURE_INSECURE_REGISTRY=${CONFIGURE_INSECURE_REGISTRY} \
and about to write ${docker_daemon_config}, but dockerd is already configured with \
an 'insecure-registry' command line option. Please make the necessary changes or disable \
the command line option and try again."
EOF
    err=true
  fi

  if [[ "${err}" ]]; then
    docker kill registry &> /dev/null
    docker rm registry &> /dev/null
    return 1
  fi

  echo "Configuring container registry on host"
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
    # kind will create cluster with name: kind-${i}
    kind create cluster --name ${i}
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

  local kubeconfig_path="$(kind get kubeconfig-path --name ${i})"
  export KUBECONFIG="${KUBECONFIG:-}:${kubeconfig_path}"

  # Simplify context name
  kubectl config rename-context kubernetes-admin@kind-${i} cluster${i}

  # TODO(font): Need to set container IP address in order for clusters to reach
  # kube API servers in other clusters until
  # https://github.com/kubernetes-sigs/kind/issues/111 is resolved.
  local container_ip_addr=$(docker inspect -f '{{range .NetworkSettings.Networks}}{{.IPAddress}}{{end}}' kind-${i}-control-plane)
  sed -i "s/localhost/${container_ip_addr}/" ${kubeconfig_path}

  # TODO(font): Need to rename auth user name to avoid conflicts when using
  # multiple cluster kubeconfigs. Remove once
  # https://github.com/kubernetes-sigs/kind/issues/112 is resolved.
  sed -i "s/kubernetes-admin/kubernetes-kind-${i}-admin/" ${kubeconfig_path}
}

function check-clusters-ready() {
  for i in $(seq ${1}); do
    util::wait-for-condition 'ok' "kubectl --context cluster${i} get --raw=/healthz &> /dev/null" 120
  done
}

function configure-insecure-registry-on-cluster() {
  configure-insecure-registry-and-reload "docker exec kind-${1}-control-plane bash -c" '$(pgrep dockerd)'
}

echo "Creating container registry on host"
create-and-configure-insecure-registry

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
