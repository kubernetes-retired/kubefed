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

# This script automates the download of binaries used by deployment
# and testing of federation.

set -o errexit
set -o nounset
set -o pipefail

# Use DEBUG=1 ./scripts/download-binaries.sh to get debug output
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

# Use BASE_URL=https://my/binaries/url ./scripts/download-binaries to download
# from a different bucket
: "${BASE_URL:="https://storage.googleapis.com/k8s-c10s-test-binaries"}"

os="$(uname -s)"
os_lowercase="$(echo "$os" | tr '[:upper:]' '[:lower:]' )"
arch="$(uname -m)"

echo "About to download some binaries. This might take a while..."

root_dir="$(cd "$(dirname "$0")/.." ; pwd)"
dest_dir="${root_dir}/bin"
mkdir -p "${dest_dir}"

kb_version="0.1.12"
kb_tgz="kubebuilder_${kb_version}_linux_amd64.tar.gz"
kb_url="https://github.com/kubernetes-sigs/kubebuilder/releases/download/v${kb_version}/${kb_tgz}"
curl "${curl_args}O" "${kb_url}" \
  && tar xzfP "${kb_tgz}" -C "${dest_dir}" --strip-components=2 \
  && rm "${kb_tgz}"

# Use a binary built from master; fedv2 requires >= 1.11 due to crd
# validation changes and the most recent stable release (beta2) is not
# compatible.
# TODO(marun) Remove when kubebuilder includes released 1.11 binaries
ci_version="$(curl "$curl_args" https://storage.googleapis.com/kubernetes-release-dev/ci/latest-green.txt)"
ks_tgz="kubernetes-server-linux-amd64.tar.gz"
ks_url="https://storage.googleapis.com/kubernetes-release-dev/ci/${ci_version}/${ks_tgz}"
curl "${curl_args}O" "${ks_url}" \
  && tar xzf "${ks_tgz}" kubernetes/server/bin/kube-apiserver -C "${dest_dir}" --strip-components=2 \
  && rm "${ks_tgz}"

echo    "# destination:"
echo    "#   ${dest_dir}"
echo    "# versions:"
echo -n "#   etcd:           "; "${dest_dir}/etcd" --version | head -n 1
echo -n "#   kube-apiserver: "; "${dest_dir}/kube-apiserver" --version
echo -n "#   kubectl:        "; "${dest_dir}/kubectl" version --client --short
echo -n "#   kubebuilder:    "; "${dest_dir}/kubebuilder" version
