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
E2E_TEST_CMD="go test -v ./test/e2e -args -kubeconfig=${HOME}/.kube/config -ginkgo.v -single-call-timeout=1m -ginkgo.trace"

function build-binaries() {
  ${MAKE_CMD} controller
  ${MAKE_CMD} kubefed2
}

function run-integration-tests() {
  # Ensure the test binaries are in the path.
  export TEST_ASSET_PATH="${base_dir}/bin"
  export TEST_ASSET_ETCD="${TEST_ASSET_PATH}/etcd"
  export TEST_ASSET_KUBE_APISERVER="${TEST_ASSET_PATH}/kube-apiserver"
  go test -v ./test/integration
  rc=$((rc || $?))
  return ${rc}
}

function launch-minikube-cluster() {
  # Move crictl to system path as expected by kubeadm.
  sudo mv ${TEST_ASSET_PATH}/crictl /usr/bin/
  sudo bash -c "export PATH=${PATH}; minikube start -p cluster1 --kubernetes-version v1.11.0 --vm-driver=none --v=4"

  # Change ownership of .kube and .minikube config to use without sudo.
  sudo chown -R $USER $HOME/.kube
  sudo chgrp -R $USER $HOME/.kube

  sudo chown -R $USER $HOME/.minikube
  sudo chgrp -R $USER $HOME/.minikube
}

function run-e2e-tests() {
  ${E2E_TEST_CMD}
  rc=$((rc || $?))
  return ${rc}
}

function run-namespaced-e2e-tests() {
  ${E2E_TEST_CMD} -federation-namespace=foo -registry-namespace=foo -limited-scope=true
  rc=$((rc || $?))
  return ${rc}
}

# Make sure, we run in the root of the repo and
# therefore run the tests on all packages
base_dir="$( cd "$(dirname "$0")/.." && pwd )"
cd "$base_dir" || {
  echo "Cannot cd to '$base_dir'. Aborting." >&2
  exit 1
}

rc=0

echo "Building federation binaries"
build-binaries

echo "Downloading test dependencies"
./scripts/download-binaries.sh

echo "Running go integration tests"
run-integration-tests

echo "Downloading e2e test dependencies"
./scripts/download-e2e-binaries.sh

export PATH=${TEST_ASSET_PATH}:${PATH}

echo "Launching minikube cluster"
launch-minikube-cluster

echo "Waiting for minikube cluster to be ready"
util::wait-for-condition 'ok' 'kubectl get --raw=/healthz' 120

echo "Deploying federation-v2"
DOCKER_PUSH=false ./scripts/deploy-federation.sh quay.io/kubernetes-multicluster/federation-v2:e2e

echo "Running go e2e tests"
run-e2e-tests

echo "Deleting federation-v2"
./scripts/delete-federation.sh

echo "Deploying namespaced federation-v2"
FEDERATION_NAMESPACE=foo NAMESPACED=y DOCKER_PUSH=false ./scripts/deploy-federation.sh quay.io/kubernetes-multicluster/federation-v2:e2e

echo "Running go e2e tests"
run-namespaced-e2e-tests

exit $rc
