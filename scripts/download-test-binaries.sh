#!/usr/bin/env bash
set -eu

# Use DEBUG=1 ./bin/download-test-binaries.sh to get debug output
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

# Use BASE_URL=https://my/binaries/url ./bin/download-binaries to download
# from a different bucket
: "${BASE_URL:="https://storage.googleapis.com/k8s-c10s-test-binaries"}"

os="$(uname -s)"
os_lowercase="$(echo "$os" | tr '[:upper:]' '[:lower:]' )"
arch="$(uname -m)"

echo "About to download some binaries. This might take a while..."

root_dir="$(cd "$(dirname "$0")/.." ; pwd)"
dest_dir="${root_dir}/bin"
mkdir "${dest_dir}"
etcd_dest="${dest_dir}/etcd"
kubectl_dest="${dest_dir}/kubectl"
kube_apiserver_dest="${dest_dir}/kube-apiserver"

curl "${curl_args}" "${BASE_URL}/etcd-${os}-${arch}" --output "$etcd_dest"
curl "${curl_args}" "${BASE_URL}/kube-apiserver-${os}-${arch}" --output "$kube_apiserver_dest"

kubectl_version="$(curl "$curl_args" https://storage.googleapis.com/kubernetes-release/release/stable.txt)"
kubectl_url="https://storage.googleapis.com/kubernetes-release/release/${kubectl_version}/bin/${os_lowercase}/amd64/kubectl"
curl "${curl_args}" "$kubectl_url" --output "$kubectl_dest"

cr_dest="${dest_dir}/clusterregistry"
cr_tgz="clusterregistry-server.tar.gz"
cr_url="https://storage.googleapis.com/crreleases/v0.0.3/${cr_tgz}"
curl "${curl_args}O" "${cr_url}" \
  && tar -xzf "${cr_tgz}" -C "${dest_dir}" ./clusterregistry \
  && rm "${cr_tgz}"

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
echo -n "#   apiserver-boot:  "; "${sb_dest}" version
