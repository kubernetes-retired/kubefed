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
DOWNLOAD_BINARIES="${DOWNLOAD_BINARIES:-}"
CONTAINER_REGISTRY_HOST="${CONTAINER_REGISTRY_HOST:-172.17.0.1:5000}"
COMMON_TEST_CMD="go test -v"
COMMON_TEST_ARGS="./test/e2e -args -ginkgo.v -single-call-timeout=1m -ginkgo.trace -ginkgo.randomizeAllSpecs"
MANAGED_E2E_TEST_CMD="${COMMON_TEST_CMD} -race ${COMMON_TEST_ARGS}"
# Specifying a kube config allows the tests to target deployed (unmanaged) fixture
UNMANAGED_E2E_TEST_CMD="${COMMON_TEST_CMD} ${COMMON_TEST_ARGS} -kubeconfig=${HOME}/.kube/config"

function build-binaries() {
  ${MAKE_CMD} hyperfed
  ${MAKE_CMD} controller
  ${MAKE_CMD} kubefed2
}

function download-dependencies() {
  if [[ -z "${DOWNLOAD_BINARIES}" ]]; then
    return
  fi

  ./scripts/download-binaries.sh
}

function run-unit-tests() {
  ${MAKE_CMD} test
}

function run-e2e-tests-with-managed-fixture() {
  # Ensure the test binaries are in the path.
  export TEST_ASSET_PATH="${ROOT_DIR}/bin"
  export TEST_ASSET_ETCD="${TEST_ASSET_PATH}/etcd"
  export TEST_ASSET_KUBE_APISERVER="${TEST_ASSET_PATH}/kube-apiserver"
  ${MANAGED_E2E_TEST_CMD}
}

function join-cluster-list() {
  if [[ -z "${JOIN_CLUSTERS}" ]]; then
    for i in $(seq 2 ${NUM_CLUSTERS}); do
      JOIN_CLUSTERS+="cluster${i} "
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

function check-make-generate-output() {
  ${MAKE_CMD} generate
  echo "Checking state of working tree after running 'make generate'"
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
  git diff
  return 1
}

# Make sure, we run in the root of the repo and
# therefore run the tests on all packages
cd "$ROOT_DIR" || {
  echo "Cannot cd to '$ROOT_DIR'. Aborting." >&2
  exit 1
}

export PATH=${ROOT_DIR}/bin:${PATH}

echo "Downloading test dependencies"
download-dependencies

echo "Checking initial state of working tree"
check-git-state

echo "Verifying Gofmt"
./hack/go-tools/verify-gofmt.sh

echo "Checking that correct Error Package is used."
./hack/verify-errpkg.sh

echo "Checking that 'make generate' is up-to-date"
check-make-generate-output

echo "Checking that fixture is available for all federate directives"
./scripts/check-directive-fixtures.sh

echo "Building federation binaries"
build-binaries

echo "Running unit tests"
run-unit-tests

echo "Running go e2e tests with managed fixture"
run-e2e-tests-with-managed-fixture

echo "Downloading e2e test dependencies"
./scripts/download-e2e-binaries.sh

CREATE_INSECURE_REGISTRY=y CONFIGURE_INSECURE_REGISTRY=y OVERWRITE_KUBECONFIG=y \
    ./scripts/create-clusters.sh

# Initialize list of clusters to join
join-cluster-list > /dev/null

echo "Deploying cluster-scoped federation-v2"
./scripts/deploy-federation.sh ${CONTAINER_REGISTRY_HOST}/federation-v2:e2e $(join-cluster-list)

echo "Running go e2e tests with cluster-scoped unmanaged fixture"
run-e2e-tests-with-unmanaged-fixture

echo "Deleting cluster-scoped federation-v2"
./scripts/delete-federation.sh

echo "Deploying namespace-scoped federation-v2"
FEDERATION_NAMESPACE=foo NAMESPACED=y ./scripts/deploy-federation.sh ${CONTAINER_REGISTRY_HOST}/federation-v2:e2e $(join-cluster-list)

echo "Running go e2e tests with namespace-scoped unmanaged fixture"
run-namespaced-e2e-tests-with-unmanaged-fixture

echo "Deleting namespace-scoped federation-v2"
FEDERATION_NAMESPACE=foo NAMESPACED=y DELETE_CLUSTER_RESOURCE=y ./scripts/delete-federation.sh
