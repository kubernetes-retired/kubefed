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

# shellcheck source=util.sh
source "${BASH_SOURCE%/*}/util.sh"
ROOT_DIR="$(cd "${BASH_SOURCE%/*}/.." ; pwd)"
TEMP_DIR="$(mktemp -d)"
MAKE_CMD="make -C ${ROOT_DIR}"
OS="$(go env GOOS)"
ARCH="$(go env GOARCH)"
PLATFORM="${OS}-${ARCH}"
NUM_CLUSTERS="${NUM_CLUSTERS:-2}"
JOIN_CLUSTERS="${JOIN_CLUSTERS:-}"
DOWNLOAD_BINARIES="${DOWNLOAD_BINARIES:-}"
COMMON_TEST_ARGS="-kubeconfig=${HOME}/.kube/config -ginkgo.v -single-call-timeout=2m -ginkgo.trace -ginkgo.randomizeAllSpecs"
E2E_TEST_CMD="${TEMP_DIR}/e2e-${PLATFORM} ${COMMON_TEST_ARGS}"
# Disable limited scope in-memory controllers to ensure the controllers in the
# race detection test behave consistently with deployed controllers for a
# given control plane scope.
IN_MEMORY_E2E_TEST_CMD="go test -v -timeout 900s -race ./test/e2e -args ${COMMON_TEST_ARGS} -in-memory-controllers=true -limited-scope-in-memory-controllers=false"

KUBEFED_UPGRADE_TEST_NS="upgrade-test"

function build-binaries() {
  ${MAKE_CMD} hyperfed
  ${MAKE_CMD} controller
  ${MAKE_CMD} kubefedctl
  ${MAKE_CMD} e2e
  # Copying the test binary to ${TEMP_DIR} to ensure
  # there's no dependency on the static files in the path
  cp "${ROOT_DIR}/bin/e2e-${PLATFORM}" "${TEMP_DIR}"
}

function download-dependencies() {
  if [[ "${DOWNLOAD_BINARIES:-}" == "y"  ]]; then
    ./scripts/download-binaries.sh
  fi
}

function run-unit-tests() {
  KUBEBUILDER_ASSETS=${ROOT_DIR}/bin ${MAKE_CMD} test
}

function run-e2e-tests() {
  ${E2E_TEST_CMD}
}

function run-e2e-upgrade-test() {
  HOST_CLUSTER="$(kubectl config current-context)"

  echo "Adding a repo to install an older kubefed version"
  helm repo add kubefed-charts https://raw.githubusercontent.com/kubernetes-sigs/kubefed/master/charts
  helm repo  update

  # Get the previous version prior to our latest stable version
  KUBEFED_UPGRADE_TEST_VERSION=$(helm search repo kubefed-charts/kubefed  --versions | awk '{print $2}' | head -3 | tail -1)

  echo "Installing an older kubefed version v${KUBEFED_UPGRADE_TEST_VERSION}"
  helm install kubefed kubefed-charts/kubefed --namespace ${KUBEFED_UPGRADE_TEST_NS} --version=v${KUBEFED_UPGRADE_TEST_VERSION} --create-namespace --wait

  deployment-image-as-expected "${KUBEFED_UPGRADE_TEST_NS}" kubefed-admission-webhook admission-webhook "quay.io/kubernetes-multicluster/kubefed:v${KUBEFED_UPGRADE_TEST_VERSION}"
  deployment-image-as-expected "${KUBEFED_UPGRADE_TEST_NS}" kubefed-controller-manager controller-manager "quay.io/kubernetes-multicluster/kubefed:v${KUBEFED_UPGRADE_TEST_VERSION}"

  echo "Upgrading kubefed to current version"
  IMAGE_NAME="local/kubefed:e2e"
  local repository=${IMAGE_NAME%/*}
  local image_tag=${IMAGE_NAME##*/}
  local image=${image_tag%:*}
  local tag=${image_tag#*:}

  helm upgrade -i kubefed charts/kubefed --namespace ${KUBEFED_UPGRADE_TEST_NS} \
  --set controllermanager.controller.repository=${repository} \
  --set controllermanager.controller.image=${image} \
  --set controllermanager.controller.tag=${tag} \
  --set controllermanager.webhook.repository=${repository} \
  --set controllermanager.webhook.image=${image} \
  --set controllermanager.webhook.tag=${tag} \
  --set controllermanager.featureGates.RawResourceStatusCollection=Enabled \
  --wait

  deployment-image-as-expected "${KUBEFED_UPGRADE_TEST_NS}" kubefed-admission-webhook admission-webhook "local/kubefed:e2e"
  deployment-image-as-expected "${KUBEFED_UPGRADE_TEST_NS}" kubefed-controller-manager controller-manager "local/kubefed:e2e"
}

function run-e2e-tests-with-in-memory-controllers() {
  ${IN_MEMORY_E2E_TEST_CMD}
}

function run-e2e-tests-with-not-ready-clusters() {
  # Run the tests without any verbosity. The unhealthy nodes generate
  # too much logs.
  go test -timeout 900s ./test/e2e \
    -args -kubeconfig=${HOME}/.kube/config \
    -single-call-timeout=2m \
    -ginkgo.randomizeAllSpecs \
    -limited-scope=true \
    -in-memory-controllers=true \
    -simulate-federation=true \
    -ginkgo.focus='\[NOT_READY\]'
}

function run-namespaced-e2e-tests() {
  local namespaced_e2e_test_cmd="${E2E_TEST_CMD} -kubefed-namespace=foo -limited-scope=true"
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
  for line in ${output}; do
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

echo "Checking boilerplate text"
./third-party/k8s.io/repo-infra/hack/verify_boilerplate.py --rootdir="${ROOT_DIR}"

echo "Linting"
golangci-lint run -c .golangci.yml --fix
check-git-state

echo "Checking that correct Error Package is used."
./hack/verify-errpkg.sh

echo "Checking that correct Logging Package is used."
./hack/verify-klog.sh

echo "Checking that 'make generate' is up-to-date"
check-make-generate-output

echo "Checking that fixture is available for all federate directives"
./scripts/check-directive-fixtures.sh

echo "Building KubeFed binaries"
build-binaries

echo "Running unit tests"
run-unit-tests

echo "Downloading e2e test dependencies"
./scripts/download-e2e-binaries.sh

KIND_TAG="v1.21.1@sha256:69860bda5563ac81e3c0057d654b5253219618a22ec3a346306239bba8cfa1a6" ./scripts/create-clusters.sh

declare -a join_cluster_list=()
if [[ -z "${JOIN_CLUSTERS}" ]]; then
  for i in $(seq 2 "${NUM_CLUSTERS}"); do
    join_cluster_list+=("cluster${i}")
  done
fi

echo "Deploying cluster-scoped kubefed"
KIND_CLUSTER_NAME=cluster1 KIND_LOAD_IMAGE=y ./scripts/deploy-kubefed.sh local/kubefed:e2e "${join_cluster_list[@]-}"

echo "Running e2e tests against cluster-scoped kubefed"
run-e2e-tests

echo "Validating KubeFed walkthrough"
./scripts/deploy-federated-nginx.sh
kubectl delete ns test-namespace

echo "Scaling down cluster-scoped controller manager"
kubectl scale deployments kubefed-controller-manager -n kube-federation-system --replicas=0

echo "Running e2e tests with race detector against cluster-scoped kubefed with in-memory controllers"
run-e2e-tests-with-in-memory-controllers

echo "Running e2e tests with not-ready clusters"
run-e2e-tests-with-not-ready-clusters

# FederatedTypeConfig controller is needed to remove finalizers from
# FederatedTypeConfigs in order to successfully delete the KubeFed
# control plane in the next step.
echo "Scaling back up cluster-scoped controller manager prior to deletion"
kubectl scale deployments kubefed-controller-manager -n kube-federation-system --replicas=1

echo "Deleting cluster-scoped kubefed"
./scripts/delete-kubefed.sh

echo "Deploying namespace-scoped kubefed"
KUBEFED_NAMESPACE=foo NAMESPACED=y ./scripts/deploy-kubefed.sh local/kubefed:e2e "${join_cluster_list[@]}"

echo "Running go e2e tests with namespace-scoped kubefed"
run-namespaced-e2e-tests

echo "Deleting namespace-scoped kubefed"
KUBEFED_NAMESPACE=foo NAMESPACED=y DELETE_CLUSTER_RESOURCE=y ./scripts/delete-kubefed.sh

echo "Running e2e upgrade test"
run-e2e-upgrade-test

echo "Deleting kubefed"
KUBEFED_NAMESPACE=${KUBEFED_UPGRADE_TEST_NS} DELETE_CLUSTER_RESOURCE=y ./scripts/delete-kubefed.sh
