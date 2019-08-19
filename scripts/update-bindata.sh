#!/usr/bin/env bash

# Copyright 2019 The Kubernetes Authors.
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

if command -v go-bindata > /dev/null; then
  cd "${ROOT_DIR}"
  "go-bindata" \
    -nocompress \
    -nometadata \
    -pkg 'common' \
    -o test/common/bindata.go \
    test/common/fixtures \
    config/kubefedconfig.yaml \
    config/enabletypedirectives

  cat ./hack/boilerplate.go.txt ./test/common/bindata.go > tmp \
    && mv tmp ./test/common/bindata.go
else
  echo "go-bindata is not found. Use './scripts/download-binaries.sh' to download it."
  exit 1
fi

