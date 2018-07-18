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
curl_args="-Ls"
[[ -z "${DEBUG:-""}" ]] || {
  set -x
  curl_args="-L"
}

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

# Minikube
mk_version="0.28.1"
mk_bin="minikube-linux-amd64"
mk_url="https://github.com/kubernetes/minikube/releases/download/v${mk_version}/${mk_bin}"
mk_dest="${dest_dir}/minikube"
curl "${curl_args}" "${mk_url}" --output "${mk_dest}"
chmod 755 "${mk_dest}"

# crictl
crictl_version="1.11.1"
crictl_tgz="crictl-v1.11.1-linux-amd64.tar.gz"
crictl_url="https://github.com/kubernetes-incubator/cri-tools/releases/download/v${crictl_version}/${crictl_tgz}"
curl "${curl_args}O" "${crictl_url}" \
  && tar xzfP "${crictl_tgz}" -C "${dest_dir}" \
  && rm "${crictl_tgz}"

echo    "# destination:"
echo    "#   ${dest_dir}"
echo    "# versions:"
echo -n "#   minikube:           "; ${dest_dir}/minikube version | awk '{print $3}'
echo -n "#     crictl:           "; ${dest_dir}/crictl --version | awk '{print $3}'
