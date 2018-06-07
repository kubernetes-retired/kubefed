#!/bin/bash

# Copyright 2017 The Kubernetes Authors.
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

set -euo pipefail

SCRIPT_ROOT="$( cd "$( dirname "${BASH_SOURCE[0]}" )" && pwd )"
REPO_ROOT=$(cd ${SCRIPT_ROOT}/..; pwd)
SCRIPT_BASE=$(cd ${SCRIPT_ROOT}/../..; pwd)
REPO_DIRNAME=$(basename $(dirname "${SCRIPT_ROOT}"))
TMP_GOPATH="$(mktemp -d /tmp/gopathXXXXXXXX)"
GEN_TMPDIR="$(mktemp -d /tmp/genXXXXXXXX)"

# Called on EXIT after the temporary directories are created.
function clean_up() {
  if [[ "${TMP_GOPATH}" == "/tmp/gopath"* ]]; then
    rm -rf "${TMP_GOPATH}"
  fi
  if [[ "${GEN_TMPDIR}" == "/tmp/gen"* ]]; then
    rm -rf "${GEN_TMPDIR}"
  fi
}
trap clean_up EXIT

# Generates code for the provided groupname ($1) and version ($2) using $3
# as the --output-base flag for all generation commands.
# To verify instead of generating, pass "--verify-only" as $4.
function generate_group() {
  local GROUP_NAME=$1
  local VERSION=$2
  local OUTPUT_BASE=$3
  local CLIENT_PKG=k8s.io/cluster-registry/pkg/client
  local CLIENTSET_PKG=${CLIENT_PKG}/clientset_generated
  local LISTERS_PKG=${CLIENT_PKG}/listers_generated
  local INFORMERS_PKG=${CLIENT_PKG}/informers_generated
  local OPENAPI_PKG=${CLIENT_PKG}/openapi_generated
  local APIS_PKG=k8s.io/cluster-registry/pkg/apis

  local INPUT_DIR="${APIS_PKG}/${GROUP_NAME}/${VERSION}"

  echo "generating clientset for group ${GROUP_NAME} and version ${VERSION} at ${SCRIPT_BASE}/${CLIENT_PKG}"
  bazel run //vendor/k8s.io/code-generator/cmd/client-gen -- \
    --go-header-file "${SCRIPT_ROOT}/boilerplate/boilerplate.go.txt" \
    --input-base ${APIS_PKG} \
    --input ${GROUP_NAME}/${VERSION} \
    --clientset-path ${CLIENTSET_PKG} \
    --output-base "${OUTPUT_BASE}" \
    --clientset-name "clientset" \
    "$4"

  echo "generating listers for group ${GROUP_NAME} and version ${VERSION} at ${SCRIPT_BASE}/${LISTERS_PKG}"
  bazel run //vendor/k8s.io/code-generator/cmd/lister-gen -- \
    --go-header-file "${SCRIPT_ROOT}/boilerplate/boilerplate.go.txt" \
    --input-dirs ${INPUT_DIR} \
    --output-package ${LISTERS_PKG} \
    --output-base "${OUTPUT_BASE}" \
    "$4"

  echo "generating informers for group ${GROUP_NAME} and version ${VERSION} at ${SCRIPT_BASE}/${INFORMERS_PKG}"
  bazel run //vendor/k8s.io/code-generator/cmd/informer-gen -- \
    --go-header-file "${SCRIPT_ROOT}/boilerplate/boilerplate.go.txt" \
    --input-dirs ${INPUT_DIR} \
    --versioned-clientset-package ${CLIENT_PKG}/clientset_generated/clientset \
    --listers-package ${LISTERS_PKG} \
    --output-package ${INFORMERS_PKG} \
    --output-base "${OUTPUT_BASE}" \
    "$4"

  echo "generating deep copies"
  bazel run //vendor/k8s.io/code-generator/cmd/deepcopy-gen -- \
    --go-header-file "${SCRIPT_ROOT}/boilerplate/boilerplate.go.txt" \
    --input-dirs ${INPUT_DIR} \
    --output-base "${OUTPUT_BASE}" \
    --output-file-base zz_generated.deepcopy \
    "$4"

  echo "generating defaults"
  bazel run //vendor/k8s.io/code-generator/cmd/defaulter-gen -- \
    --go-header-file "${SCRIPT_ROOT}/boilerplate/boilerplate.go.txt" \
    --input-dirs ${INPUT_DIR} \
    --output-base "${OUTPUT_BASE}" \
    --output-file-base zz_generated.defaults \
    "$4"

  echo "generating protocol buffers for group ${GROUP_NAME} and version ${VERSION} at ${SCRIPT_BASE}/${CLIENT_PKG}"

  # The generated go_binaries are not guaranteed to be in exactly the
  # corresponding package in bazel-bin; there may be another
  # architecture-specific subdirectory.
  # The call to xargs trims leading whitespace.
  PROTOC_GEN_GOGO_PATH="$(dirname "$(bazel build //vendor/k8s.io/code-generator/cmd/go-to-protobuf/protoc-gen-gogo:protoc-gen-gogo 2>&1 | grep bazel-bin | xargs)")"
  PROTOC_PATH="$(dirname "$(bazel build @com_google_protobuf//:protoc 2>&1 | grep bazel-bin | xargs)")"

  # The protocol buffer compiler expects that the tools that it runs are in the
  # PATH.
  PATH="$(bazel info workspace)/${PROTOC_PATH}:$(bazel info workspace)/${PROTOC_GEN_GOGO_PATH}:${PATH}" \
    bazel run //vendor/k8s.io/code-generator/cmd/go-to-protobuf -- \
      --go-header-file "${SCRIPT_ROOT}/boilerplate/boilerplate.go.txt" \
      --proto-import "$(bazel info output_base)/external/com_google_protobuf/src" \
      --proto-import "$(bazel info workspace)/vendor" \
      --packages ${INPUT_DIR}

  echo "generating openapi"
  mkdir -p "${OUTPUT_BASE}/${OPENAPI_PKG}"
  bazel run //vendor/k8s.io/code-generator/cmd/openapi-gen -- \
    --go-header-file "${SCRIPT_ROOT}/boilerplate/boilerplate.go.txt" \
    --input-dirs ${INPUT_DIR},k8s.io/apimachinery/pkg/apis/meta/v1,k8s.io/apimachinery/pkg/version,k8s.io/apimachinery/pkg/runtime \
    --output-base "${OUTPUT_BASE}" \
    --output-package "${APIS_PKG}/${GROUP_NAME}/${VERSION}" \
    --output-file-base zz_generated.openapi \
    "$4"
}

# Set up the temporary GOPATH with necessary dependencies.
mkdir -p "${TMP_GOPATH}/src/k8s.io/cluster-registry"
mkdir -p "${TMP_GOPATH}/src/k8s.io/apimachinery"
mkdir -p "${TMP_GOPATH}/src/k8s.io/kube-openapi"
mkdir -p "${TMP_GOPATH}/src/github.com/gogo"
cp -r "${SCRIPT_ROOT}/../"* "${TMP_GOPATH}/src/k8s.io/cluster-registry"
cp -r "${SCRIPT_ROOT}/../vendor/k8s.io/apimachinery/"* "${TMP_GOPATH}/src/k8s.io/apimachinery"
cp -r "${SCRIPT_ROOT}/../vendor/k8s.io/kube-openapi/"* "${TMP_GOPATH}/src/k8s.io/kube-openapi"
cp -r "${SCRIPT_ROOT}/../vendor/github.com/gogo/"* "${TMP_GOPATH}/src/github.com/gogo"

# In verify mode, generate into the temporary GOPATH.
OUTPUT_BASE="${GEN_TMPDIR}"
if [ -n "$@" ]; then
  OUTPUT_BASE="${TMP_GOPATH}/src"
fi

# Perform the code generation.
export GOPATH="${TMP_GOPATH}"
export GOROOT="$(bazel info output_base)/external/go_sdk"

generate_group clusterregistry v1alpha1 "${OUTPUT_BASE}" "${@-}"

# In generate mode, copy the generated files back into the tree.
if [ -n "$@" ]; then
  cp -r "${OUTPUT_BASE}/k8s.io/cluster-registry/"* "${SCRIPT_BASE}/${REPO_DIRNAME}"
fi
