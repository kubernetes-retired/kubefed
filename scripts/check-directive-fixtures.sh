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

# This script ensures that enable type directives are not merged to
# the tree without corresponding fixture to ensure federation of the
# target type will be tested.

set -o errexit
set -o nounset
set -o pipefail

ROOT_DIR="$(cd "$(dirname "$0")/.." ; pwd)"

function check-directive-fixtures() {
  local directives=( "${ROOT_DIR}/config/enabletypedirectives/"*.yaml )
  for file in "${directives[@]}"; do
    local filename="$(basename "${file}")"
    local expected_file="${ROOT_DIR}/test/common/fixtures/${filename}"
    if [[ ! -f "${expected_file}" ]]; then
      echo "Fixture is missing for ${file}" >&2
      echo "Please add fixture with filename ${expected_file}" >&2
      exit 1
    fi
  done
}

check-directive-fixtures
