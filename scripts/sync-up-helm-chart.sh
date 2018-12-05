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

NS="${FEDERATION_NAMESPACE:-federation-system}"
CHART_FEDERATED_CRD_DIR="${CHART_FEDERATED_CRD_DIR:-charts/federation-v2/charts/controllermanager/templates}"
CHART_FEDERATED_PROPAGATION_DIR="${CHART_FEDERATED_PROPAGATION_DIR:-charts/federation-v2/templates}"
INSTALL_CRDS_YAML="${INSTALL_CRDS_YAML:-hack/install-crds-latest.yaml}"

INSTALL_CRDS_YAML="${INSTALL_CRDS_YAML}" scripts/generate-install-crds-yaml.sh

# "diff -U 4" will take 1 as return code which will cause the script failed to execute, here
# I was force returning true to get a return code as 0.
crd_diff=`(diff -U 4 $INSTALL_CRDS_YAML $CHART_FEDERATED_CRD_DIR/crds.yaml; true;)`
if [ -n "${crd_diff}" ]; then
  cp -f $INSTALL_CRDS_YAML $CHART_FEDERATED_CRD_DIR/crds.yaml
fi

# Generate YAML templates to enable resource propagation for helm chart.
for filename in ./config/federatedirectives/*.yaml; do
  ./bin/kubefed2 federate enable -f "${filename}" --federation-namespace="${NS}" --dry-run -oyaml > ${CHART_FEDERATED_PROPAGATION_DIR}/$(basename $filename)
done
