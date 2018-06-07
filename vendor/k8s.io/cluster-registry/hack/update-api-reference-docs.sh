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

# Generates updated api-reference docs from the latest OpenAPI spec for
# clusterregistry apiserver. The docs are generated at docs/api-reference
# Usage: ./update-api-reference-docs.sh <absolute output path>

set -euo pipefail

SCRIPT_ROOT="$( cd "$( dirname "${BASH_SOURCE[0]}" )" && pwd )"
REPO_DIR=$(cd ${SCRIPT_ROOT}/..; pwd)

DEFAULT_OUTPUT="${REPO_DIR}/docs/api-reference"
OUTPUT=${1:-${DEFAULT_OUTPUT}}

OPENAPI_SPEC_PATH="${REPO_DIR}/api"

# Entry in GROUP_VERSIONS must match entry in GV_DIRS
GROUP_VERSIONS=("clusterregistry/v1alpha1")
GV_DIRS=("${REPO_DIR}/pkg/apis/clusterregistry/v1alpha1")

# Generates API reference docs for the given API group versions.
# Required env vars:
#   GROUP_VERSIONS: Array of group versions to be included in the reference
#   docs.
#   GV_DIRS: Array of root directories for those group versions.
# Input vars:
#   $1: Root directory path for OpenAPI spec
#   $2: Root directory path where the reference docs should be generated.
gen_api_ref_docs() {
    : "${GROUP_VERSIONS?Must set GROUP_VERSIONS env var}"
    : "${GV_DIRS?Must set GV_DIRS env var}"

  echo "Generating API reference docs for group versions: ${GROUP_VERSIONS[@]}, at dirs: ${GV_DIRS[@]}"
  GROUP_VERSIONS=(${GROUP_VERSIONS[@]})
  GV_DIRS=(${GV_DIRS[@]})
  local swagger_spec_path=${1}
  local output_dir=${2}
  echo "Reading swagger spec from: ${swagger_spec_path}"
  echo "Generating the docs at: ${output_dir}"

  local tmp_path="_output"
  local tmp_subpath="${tmp_path}/generated_html"
  local output_tmp="${REPO_DIR}/${tmp_subpath}"

  echo "Generating api reference docs at ${output_tmp}"

  for ver in "${GROUP_VERSIONS[@]}"; do
    mkdir -p "${output_tmp}/${ver}"
  done

  user_flags="-u $(id -u)"
  if [[ $(uname) == "Darwin" ]]; then
    # mapping in a uid from OS X doesn't make any sense
    user_flags=""
  fi

  for i in "${!GROUP_VERSIONS[@]}"; do
    local ver=${GROUP_VERSIONS[i]}
    local dir=${GV_DIRS[i]}
    local tmp_in_host="${output_tmp}/${ver}"
    local register_file="${dir}/register.go"
    local swagger_json_name="swagger"

    docker run ${user_flags} \
      --rm -v "${tmp_in_host}":/output:z \
      -v "${swagger_spec_path}":/swagger-source:z \
      -v "${register_file}":/register.go:z \
      --net=host -e "https_proxy=${KUBERNETES_HTTPS_PROXY:-}" \
      k8s.gcr.io/gen-swagger-docs:v8 \
      "${swagger_json_name}"
  done

  # Check if we actually changed anything
  pushd "${output_tmp}" > /dev/null
  touch .generated_html
  find . -type f | cut -sd / -f 2- | LC_ALL=C sort > .generated_html
  popd > /dev/null

  if LANG=C sed --help 2>&1 | grep -q GNU; then
    SED="sed"
  elif which gsed &>/dev/null; then
    SED="gsed"
  else
    echo "Failed to find GNU sed as sed or gsed. If you are on Mac: brew install gnu-sed." >&2
    exit 1
  fi

  while read file; do
    if [[ -e "${output_dir}/${file}" && -e "${output_tmp}/${file}" ]]; then
      echo "comparing ${output_dir}/${file} with ${output_tmp}/${file}"

      # Remove the timestamp to reduce conflicts in PR(s)
      $SED -i 's/^Last updated.*$//' "${output_tmp}/${file}"

      # By now, the contents should be normalized and stripped of any
      # auto-managed content.
      if diff -NauprB "${output_dir}/${file}" "${output_tmp}/${file}" >/dev/null; then
        # actual contents same, overwrite generated with original.
        cp "${output_dir}/${file}" "${output_tmp}/${file}"
      fi
    fi
  done <"${output_tmp}/.generated_html"

  echo "Moving api reference docs from ${output_tmp} to ${output_dir}"

  # Create output_dir if doesn't exist. Prevents error on copy.
  mkdir -p "${output_dir}"

  cp -af "${output_tmp}"/* "${output_dir}"
  rm -r "${REPO_DIR}/${tmp_path}"
}

echo "Note: This assumes that OpenAPI spec has been updated. Please run hack/update-openapi-spec.sh to ensure that."

GROUP_VERSIONS="${GROUP_VERSIONS[@]}" GV_DIRS="${GV_DIRS[@]}" gen_api_ref_docs "${OPENAPI_SPEC_PATH}" "${OUTPUT}"
