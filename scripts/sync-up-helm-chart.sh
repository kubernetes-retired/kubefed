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
trap-add 'rm -rf "${WORKDIR}"' EXIT
NS="${KUBEFED_NAMESPACE:-kube-federation-system}"
CHART_FEDERATED_PROPAGATION_DIR="${CHART_FEDERATED_PROPAGATION_DIR:-charts/kubefed}"
TEMP_CRDS_YAML="/tmp/kubefed-crds.yaml"

export PATH=${ROOT_DIR}/bin:${PATH}

OS=`uname`
SED=sed
if [ "${OS}" == "Darwin" ];then
  if ! which gsed > /dev/null ; then
    echo "gsed is required by this script. It can be installed via homebrew (https://brew.sh)"
    exit 1
  fi
  SED=gsed
fi

# Remove existing generated crds to ensure that stale content doesn't linger.
rm -f ./config/crds/*.yaml

# Generate CRD manifest files
(cd ${ROOT_DIR}/tools && GOBIN=${ROOT_DIR}/bin go install sigs.k8s.io/controller-tools/cmd/controller-gen)
${ROOT_DIR}/bin/controller-gen crd:trivialVersions=true paths="./pkg/apis/..." output:crd:artifacts:config=config/crds

# Merge all CRD manifest files into one file
echo "" > ${TEMP_CRDS_YAML}
for filename in ./config/crds/*.yaml; do
  # Remove unwanted kubebuilder annotation
   ${SED} '/controller-gen.kubebuilder.io/d; /annotations:/d' $filename >> ${TEMP_CRDS_YAML}
done

mv ${TEMP_CRDS_YAML} ./charts/kubefed/charts/controllermanager/crds/crds.yaml

declare -rx KUBECONFIG="${WORKDIR}/kubeconfig"
kind create cluster --name=kubefed-dev
trap-add 'kind delete cluster --name=kubefed-dev' EXIT

# Generate YAML templates to enable resource propagation for helm chart.
echo -n > ${CHART_FEDERATED_PROPAGATION_DIR}/templates/federatedtypeconfig.yaml
echo -n > ${CHART_FEDERATED_PROPAGATION_DIR}/crds/crds.yaml
for filename in ./config/enabletypedirectives/*.yaml; do
  full_name=${CHART_FEDERATED_PROPAGATION_DIR}/templates/$(basename $filename)

  ./bin/kubefedctl --kubeconfig ${WORKDIR}/kubeconfig enable -f "${filename}" --kubefed-namespace="${NS}" -o yaml > ${full_name}
  $SED -n '/^---/,/^---/p' ${full_name} >> ${CHART_FEDERATED_PROPAGATION_DIR}/templates/federatedtypeconfig.yaml
  $SED -i '$d' ${CHART_FEDERATED_PROPAGATION_DIR}/templates/federatedtypeconfig.yaml

  echo "---" >> ${CHART_FEDERATED_PROPAGATION_DIR}/crds/crds.yaml
  $SED -n '/^apiVersion: apiextensions.k8s.io\/v1/,$p' ${full_name} >> ${CHART_FEDERATED_PROPAGATION_DIR}/crds/crds.yaml

  rm ${full_name}
done

# Clean kube-apiserver daemons and temporary files
echo "Helm chart synced successfully"
