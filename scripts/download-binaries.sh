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
etcd_dest="${dest_dir}/etcd"
kubectl_dest="${dest_dir}/kubectl"
kube_apiserver_dest="${dest_dir}/kube-apiserver"

curl "${curl_args}" "${BASE_URL}/etcd-${os}-${arch}" --output "$etcd_dest"
curl "${curl_args}" "${BASE_URL}/kube-apiserver-${os}-${arch}" --output "$kube_apiserver_dest"

kubectl_version="$(curl "$curl_args" https://storage.googleapis.com/kubernetes-release/release/stable.txt)"
kubectl_url="https://storage.googleapis.com/kubernetes-release/release/${kubectl_version}/bin/${os_lowercase}/amd64/kubectl"
curl "${curl_args}" "$kubectl_url" --output "$kubectl_dest"

crc_dest="${dest_dir}/crinit"
crc_tgz="clusterregistry-client.tar.gz"
crc_url="https://storage.googleapis.com/crreleases/v0.0.4/${crc_tgz}"
curl "${curl_args}O" "${crc_url}" \
  && tar -xzf "${crc_tgz}" -C "${dest_dir}" ./crinit \
  && rm "${crc_tgz}"

crs_dest="${dest_dir}/clusterregistry"
crs_tgz="clusterregistry-server.tar.gz"
crs_url="https://storage.googleapis.com/crreleases/v0.0.4/${crs_tgz}"
curl "${curl_args}O" "${crs_url}" \
  && tar -xzf "${crs_tgz}" -C "${dest_dir}" ./clusterregistry \
  && rm "${crs_tgz}"

sb_dest="${dest_dir}/apiserver-boot"
sb_tgz="apiserver-builder-v1.9-alpha.3-linux-amd64.tar.gz"
sb_url="https://github.com/kubernetes-incubator/apiserver-builder/releases/download/v1.9-alpha.3/${sb_tgz}"
curl "${curl_args}O" "${sb_url}" \
  && tar xzfP "${sb_tgz}" -C "${dest_dir}" --strip-components=1 \
  && rm "${sb_tgz}"

chmod +x "$etcd_dest" "$kubectl_dest" "$kube_apiserver_dest"

echo    "# destination:"
echo    "#   ${dest_dir}"
echo    "# versions:"
echo -n "#   etcd:            "; "$etcd_dest" --version | head -n 1
echo -n "#   kube-apiserver:  "; "$kube_apiserver_dest" --version
echo -n "#   kubectl:         "; "$kubectl_dest" version --client --short
echo -n "#   crinit:          "; "${crc_dest}" version --short
echo -n "#   clusterregistry: "; "${crs_dest}" version --short
echo -n "#   apiserver-boot:  "; "${sb_dest}" version
