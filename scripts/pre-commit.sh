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

# This script validates that binaries can be built and that all tests pass.

set -o errexit
set -o nounset
set -o pipefail

source "$(dirname "${BASH_SOURCE}")/util.sh"
ROOT_DIR="$(cd "$(dirname "$0")/.." ; pwd)"
MAKE_CMD="make -C ${ROOT_DIR}"
NUM_CLUSTERS="${NUM_CLUSTERS:-2}"
JOIN_CLUSTERS="${JOIN_CLUSTERS:-}"
CONFIGURE_INSECURE_REGISTRY="${CONFIGURE_INSECURE_REGISTRY:-}"
CONTAINER_REGISTRY_HOST="${CONTAINER_REGISTRY_HOST:-172.17.0.1:5000}"
MANAGED_E2E_TEST_CMD="go test -v ./test/e2e -args -ginkgo.v -single-call-timeout=1m -ginkgo.trace -ginkgo.randomizeAllSpecs"
# Specifying a kube config allows the tests to target deployed (unmanaged) fixture
UNMANAGED_E2E_TEST_CMD="${MANAGED_E2E_TEST_CMD} -kubeconfig=${HOME}/.kube/config"

function build-binaries() {
  ${MAKE_CMD} controller
  ${MAKE_CMD} kubefed2
}

function run-e2e-tests-with-managed-fixture() {
  # Ensure the test binaries are in the path.
  export TEST_ASSET_PATH="${base_dir}/bin"
  export TEST_ASSET_ETCD="${TEST_ASSET_PATH}/etcd"
  export TEST_ASSET_KUBE_APISERVER="${TEST_ASSET_PATH}/kube-apiserver"
  ${MANAGED_E2E_TEST_CMD}
}

docker_daemon_config="/etc/docker/daemon.json"

function create-and-configure-insecure-registry() {
  # Run insecure registry as container
  docker run -d -p 5000:5000 --restart=always --name registry registry:2

  if [[ -z "${CONFIGURE_INSECURE_REGISTRY}" ]]; then
    return
  fi

  if sudo test -f "${docker_daemon_config}"; then
    echo <<EOF "Error: ${docker_daemon_config} exists and \
CONFIGURE_INSECURE_REGISTRY=${CONFIGURE_INSECURE_REGISTRY}. This script needs \
to add an 'insecure-registries' entry with host '${CONTAINER_REGISTRY_HOST}' to \
${docker_daemon_config}. Please make the necessary changes or backup and try again."
EOF
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
    # kind will create cluster with name: kind-${i}
    kind create cluster --name ${i}
    # TODO(font): remove once all workarounds are addressed.
    fixup-cluster ${i}
  done

  # TODO(font): kind will create separate kubeconfig files for each cluster.
  # Remove once https://github.com/kubernetes-sigs/kind/issues/113 is resolved.
  kubectl config view --flatten > ~/.kube/config
  unset KUBECONFIG

  echo "Waiting for clusters to be ready"
  check-clusters-ready ${num_clusters}

  # TODO(font): Configure insecure registry on kind host cluster. Remove once
  # https://github.com/kubernetes-sigs/kind/issues/110 is resolved.
  configure-insecure-registry-on-cluster 1

  # Initialize list of clusters to join
  join-cluster-list > /dev/null
}

function fixup-cluster() {
  local i=${1} # cluster num

  local kubeconfig_path="$(kind get kubeconfig-path --name ${i})"
  export KUBECONFIG="${KUBECONFIG:-}:${kubeconfig_path}"

  # Simplify context name
  kubectl config rename-context kubernetes-admin@kind-${i} kind-${i}

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
    util::wait-for-condition 'ok' "kubectl --context kind-${i} get --raw=/healthz" 120
  done
}

function configure-insecure-registry-on-cluster() {
  configure-insecure-registry-and-reload "docker exec kind-${1}-control-plane bash -c" '$(pgrep dockerd)'
}

function join-cluster-list() {
  if [[ -z "${JOIN_CLUSTERS}" ]]; then
    for i in $(seq 2 ${NUM_CLUSTERS}); do
      JOIN_CLUSTERS+="kind-${i} "
    done
    export JOIN_CLUSTERS=$(echo ${JOIN_CLUSTERS} | sed 's/ $//')
  fi
  echo "${JOIN_CLUSTERS}"
}

function run-e2e-tests-with-unmanaged-fixture() {
  ${UNMANAGED_E2E_TEST_CMD}
}

function run-namespaced-e2e-tests-with-unmanaged-fixture() {
  local namespaced_e2e_test_cmd="${UNMANAGED_E2E_TEST_CMD} -federation-namespace=foo -registry-namespace=foo -limited-scope=true"
  # Run the placement test separately to avoid crud failures if
  # teardown doesn't remove namespace placement.
  ${namespaced_e2e_test_cmd} --ginkgo.skip=Placement
  ${namespaced_e2e_test_cmd} --ginkgo.focus=Placement
}

function check-kubebuilder-output() {
  ./bin/kubebuilder generate
  echo "Checking state of working tree after running 'kubebuilder generate'"
  check-git-state
}

function check-install-yaml() {
  PATH="${PATH}:${base_dir}/bin" FEDERATION_NAMESPACE=federation-system \
    INSTALL_YAML=./hack/install-latest.yaml \
    ./scripts/generate-install-yaml.sh \
    ${CONTAINER_REGISTRY_HOST}/federation-v2:latest
  echo "Checking state of working tree after generating install yaml"
  check-git-state
}

function check-git-state() {
  local output
  if output=$(git status --porcelain) && [ -z "${output}" ]; then
    return
  fi
  echo "ERROR: the working tree is dirty:"
  for line in "${output}"; do
    echo "${line}"
  done
  return 1
}

# Make sure, we run in the root of the repo and
# therefore run the tests on all packages
base_dir="$( cd "$(dirname "$0")/.." && pwd )"
cd "$base_dir" || {
  echo "Cannot cd to '$base_dir'. Aborting." >&2
  exit 1
}

echo "Downloading test dependencies"
./scripts/download-binaries.sh

echo "Checking initial state of working tree"
check-git-state

echo "Verifying Gofmt"
./hack/go-tools/verify-gofmt.sh

echo "Checking that 'kubebuilder generate' is up-to-date"
check-kubebuilder-output

echo "Checking that hack/install-latest.yaml is up-to-date"
check-install-yaml

echo "Building federation binaries"
build-binaries

echo "Running go e2e tests with managed fixture"
run-e2e-tests-with-managed-fixture

echo "Downloading e2e test dependencies"
./scripts/download-e2e-binaries.sh

export PATH=${TEST_ASSET_PATH}:${PATH}

echo "Creating container registry on host"
create-and-configure-insecure-registry

echo "Creating ${NUM_CLUSTERS} clusters"
create-clusters ${NUM_CLUSTERS}

echo "Deploying federation-v2"
./scripts/deploy-federation.sh ${CONTAINER_REGISTRY_HOST}/federation-v2:e2e $(join-cluster-list)

echo "Running go e2e tests with unmanaged fixture"
run-e2e-tests-with-unmanaged-fixture

echo "Deleting federation-v2"
./scripts/delete-federation.sh

echo "Deploying namespaced federation-v2"
FEDERATION_NAMESPACE=foo NAMESPACED=y ./scripts/deploy-federation.sh ${CONTAINER_REGISTRY_HOST}/federation-v2:e2e $(join-cluster-list)

echo "Running go e2e tests with unmanaged fixture"
run-namespaced-e2e-tests-with-unmanaged-fixture
