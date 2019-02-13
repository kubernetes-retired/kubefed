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
NS="${FEDERATION_NAMESPACE:-federation-system}"
CHART_FEDERATED_CRD_DIR="${CHART_FEDERATED_CRD_DIR:-charts/federation-v2/charts/controllermanager/templates}"
CHART_FEDERATED_PROPAGATION_DIR="${CHART_FEDERATED_PROPAGATION_DIR:-charts/federation-v2/templates}"
INSTALL_CRDS_YAML="${INSTALL_CRDS_YAML:-hack/install-crds-latest.yaml}"

INSTALL_CRDS_YAML="${INSTALL_CRDS_YAML}" scripts/generate-install-crds-yaml.sh

BUILD_KUBEFED="${BUILD_KUBEFED:-true}"
if [[ "${BUILD_KUBEFED}" == true ]]; then
  make -C "${ROOT_DIR}" kubefed2
fi

# "diff -U 4" will take 1 as return code which will cause the script failed to execute, here
# I was force returning true to get a return code as 0.
crd_diff=`(diff -U 4 $INSTALL_CRDS_YAML $CHART_FEDERATED_CRD_DIR/crds.yaml; true;)`
if [ -n "${crd_diff}" ]; then
  cp -f $INSTALL_CRDS_YAML $CHART_FEDERATED_CRD_DIR/crds.yaml
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
  name: federation
current-context: ""
kind: Config
preferences: {}
users: []
EOF

# Start kube-apiserver to generate CRDs
./bin/etcd --data-dir ${WORKDIR} &
util::wait-for-condition 'ok' "curl http://127.0.0.1:2379/version &> /dev/null" 30
./bin/kube-apiserver --etcd-servers=http://127.0.0.1:2379 --service-cluster-ip-range=10.0.0.0/16 --cert-dir ${WORKDIR} &
util::wait-for-condition 'ok' "kubectl --kubeconfig ${WORKDIR}/kubeconfig --context federation get --raw=/healthz &> /dev/null" 60

# Generate YAML templates to enable resource propagation for helm chart.
for filename in ./config/enabletypedirectives/*.yaml; do
  ./bin/kubefed2 --kubeconfig ${WORKDIR}/kubeconfig enable -f "${filename}" --federation-namespace="${NS}" --host-cluster-context federation -o yaml > ${CHART_FEDERATED_PROPAGATION_DIR}/$(basename $filename)
done

# Clean kube-apiserver daemons and temporary files
kill %1 # etcd
kill %2 # kube-apiserver
rm -fr ${WORKDIR}
