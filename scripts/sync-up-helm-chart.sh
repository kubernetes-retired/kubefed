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

set -o errexit
set -o nounset
set -o pipefail

source "$(dirname "${BASH_SOURCE}")/util.sh"

ROOT_DIR="$(cd "$(dirname "$0")/.." ; pwd)"
WORKDIR=$(mktemp -d)
NS="${KUBEFED_NAMESPACE:-kube-federation-system}"
CHART_FEDERATED_CRD_DIR="${CHART_FEDERATED_CRD_DIR:-charts/kubefed/charts/controllermanager/templates}"
CHART_FEDERATED_PROPAGATION_DIR="${CHART_FEDERATED_PROPAGATION_DIR:-charts/kubefed/templates}"
TEMP_CRDS_YAML="/tmp/kubefed-crds.yaml"

OS=`uname`
SED=sed
if [ "${OS}" == "Darwin" ];then
  if ! which gsed > /dev/null ; then
    echo "gsed is required by this script. It can be installed via homebrew (https://brew.sh)"
    exit 1
  fi
  SED=gsed
fi

# Check for existence of kube-apiserver and etcd binaries in bin directory
if [[ ! -f ${ROOT_DIR}/bin/etcd || ! -f ${ROOT_DIR}/bin/kube-apiserver ]];
then
  echo "Missing 'etcd' and/or 'kube-apiserver' binaries in bin directory. Call './scripts/download-binaries.sh' to download them first"
  exit 1
fi

# Remove existing generated crds to ensure that stale content doesn't linger.
rm -f ./config/crds/*.yaml

# Generate CRD manifest files
go run vendor/sigs.k8s.io/controller-tools/cmd/controller-gen/main.go crd

# Merge all CRD manifest files into one file
echo "---" > ${TEMP_CRDS_YAML}
for filename in ./config/crds/*.yaml; do
  cat $filename >> ${TEMP_CRDS_YAML}
  echo "---" >> ${TEMP_CRDS_YAML}
done

# Add crd-install to make sure the CRDs can be installed first in a helm chart.
$SED -i 's/^metadata:/metadata:\n  annotations:\n    "helm.sh\/hook": crd-install/g' "${TEMP_CRDS_YAML}"

# "diff -U 4" will take 1 as return code which will cause the script failed to execute, here
# I was force returning true to get a return code as 0.
crd_diff=`(diff -U 4 ${TEMP_CRDS_YAML} ${CHART_FEDERATED_CRD_DIR}/crds.yaml; true;)`
if [ -n "${crd_diff}" ]; then
  cp -f ${TEMP_CRDS_YAML} $CHART_FEDERATED_CRD_DIR/crds.yaml
  $SED -i '1i{{ if (or (or (not .Values.global.scope) (eq .Values.global.scope "Cluster")) (not (.Capabilities.APIVersions.Has "core.kubefed.io\/v1beta1"))) }}' ${CHART_FEDERATED_CRD_DIR}/crds.yaml
  $SED -i '$a{{ end }}' ${CHART_FEDERATED_CRD_DIR}/crds.yaml
fi

# Generate kubeconfig to access kube-apiserver. It is cleaned when script is done.
cat <<EOF > ${WORKDIR}/kubeconfig
apiVersion: v1
clusters:
- cluster:
    server: 127.0.0.1:8080
  name: development
contexts:
- context:
    cluster: development
    user: ""
  name: kubefed
current-context: ""
kind: Config
preferences: {}
users: []
EOF

# Start kube-apiserver to generate CRDs
${ROOT_DIR}/bin/etcd --data-dir ${WORKDIR} &
util::wait-for-condition 'ok' "curl http://127.0.0.1:2379/version &> /dev/null" 30

${ROOT_DIR}/bin/kube-apiserver --etcd-servers=http://127.0.0.1:2379 --service-cluster-ip-range=10.0.0.0/16 --cert-dir ${WORKDIR} &
util::wait-for-condition 'ok' "kubectl --kubeconfig ${WORKDIR}/kubeconfig --context kubefed get --raw=/healthz &> /dev/null" 60

# Generate YAML templates to enable resource propagation for helm chart.
echo -n > ${CHART_FEDERATED_PROPAGATION_DIR}/federatedtypeconfig.yaml
echo -n > ${CHART_FEDERATED_PROPAGATION_DIR}/crds.yaml
for filename in ./config/enabletypedirectives/*.yaml; do
  full_name=${CHART_FEDERATED_PROPAGATION_DIR}/$(basename $filename)

  ./bin/kubefedctl --kubeconfig ${WORKDIR}/kubeconfig enable -f "${filename}" --kubefed-namespace="${NS}" --host-cluster-context kubefed -o yaml > ${full_name}
  $SED -n '/^---/,/^---/p' ${full_name} >> ${CHART_FEDERATED_PROPAGATION_DIR}/federatedtypeconfig.yaml
  $SED -i '$d' ${CHART_FEDERATED_PROPAGATION_DIR}/federatedtypeconfig.yaml

  echo "---" >> ${CHART_FEDERATED_PROPAGATION_DIR}/crds.yaml
  $SED -n '/^apiVersion: apiextensions.k8s.io\/v1beta1/,$p' ${full_name} >> ${CHART_FEDERATED_PROPAGATION_DIR}/crds.yaml

  rm ${full_name}
done
$SED -i 's/^metadata:/metadata:\n  annotations:\n    "helm.sh\/hook": crd-install/'  ${CHART_FEDERATED_PROPAGATION_DIR}/crds.yaml
$SED -i '1i{{ if (or (or (not .Values.global.scope) (eq .Values.global.scope "Cluster")) (not (.Capabilities.APIVersions.Has "types.kubefed.io\/v1beta1"))) }}' ${CHART_FEDERATED_PROPAGATION_DIR}/crds.yaml
$SED -i '$a{{ end }}' ${CHART_FEDERATED_PROPAGATION_DIR}/crds.yaml

# Clean kube-apiserver daemons and temporary files
kill %1 # etcd
kill %2 # kube-apiserver
rm -fr ${WORKDIR}
