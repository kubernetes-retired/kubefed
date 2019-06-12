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
# KubeFed.

set -o errexit
set -o nounset
set -o pipefail

logEnd() {
  local msg='done.'
  [ "$1" -eq 0 ] || msg='Error downloading assets'
  echo "$msg"
}
trap 'logEnd $?' EXIT

echo "About to download some binaries. This might take a while..."

root_dir="$(cd "$(dirname "$0")/.." ; pwd)"
dest_dir="${root_dir}/bin"
mkdir -p "${dest_dir}"

# kind
kind_version="v0.3.0"
kind_path="${dest_dir}/kind"
kind_url="https://github.com/kubernetes-sigs/kind/releases/download/${kind_version}/kind-linux-amd64"
curl -Lo "${kind_path}" "${kind_url}" && chmod +x "${kind_path}"

# Pull the busybox image (used in tests of workload types)
docker pull busybox

echo    "# kind installation:  ${kind_path}"
