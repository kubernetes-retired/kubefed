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

# This script automates the download of e2e binaries used in testing of
# federation.

set -o errexit
set -o nounset
set -o pipefail

# Use DEBUG=1 ./scripts/download-e2e-binaries.sh to get debug output
[[ -z "${DEBUG:-""}" ]] || {
  set -x
}

logEnd() {
  local msg='done.'
  [ "$1" -eq 0 ] || msg='Error downloading assets'
  echo "$msg"
}
trap 'logEnd $?' EXIT

echo "About to download some binaries. This might take a while..."

GOBIN="$(go env GOPATH)/bin"

# kind
# TODO(font): kind does not have versioning yet.
kind_version="4d7dded365359afeb1831292cd1a3a3e15fff0b2"
kind_bin="kind"
kind_url="sigs.k8s.io"
go get -d ${kind_url}/${kind_bin}
pushd ${GOPATH}/src/${kind_url}/${kind_bin}
git checkout ${kind_version}
popd
go install ${kind_url}/${kind_bin}

# Pull the busybox image (used in tests of workload types)
docker pull busybox

echo    "# bin destination:    ${GOBIN}"
echo    "# kind installation:  ${GOBIN}/kind"
